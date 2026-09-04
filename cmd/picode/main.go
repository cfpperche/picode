// Package main is the picode entrypoint: a single binary that serves the
// PiCode web UI and manages Pi agent processes (see docs/architecture.md).
//
// Serve loop (ADR-0007): bind (port range from config.Resolve) → serve →
// wait (rebind signal | shutdown signal). Rebind binds the NEW listener
// before shutting the old one down; on bind failure the setting reverts and
// the old server keeps serving. HTTPS by default (tlsutil); discovery file
// at <data>/server.json.
//
// Subcommands (run without args to start the server):
//
//	picode screenshot --url <url> --out <file.png>
//	picode extension-install   (Chrome native host, ADR-0043)
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cfpperche/picode/internal/apps"
	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/automate"
	"github.com/cfpperche/picode/internal/backup"
	"github.com/cfpperche/picode/internal/binwatch"
	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/config"
	"github.com/cfpperche/picode/internal/docker"
	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/install"
	"github.com/cfpperche/picode/internal/presence"
	"github.com/cfpperche/picode/internal/proclock"
	"github.com/cfpperche/picode/internal/push"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/screenshot"
	"github.com/cfpperche/picode/internal/server"
	"github.com/cfpperche/picode/internal/share"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tlsutil"
	"github.com/cfpperche/picode/internal/tmux"
	providerusage "github.com/cfpperche/picode/internal/usage"
	"github.com/cfpperche/picode/internal/version"
	"github.com/cfpperche/picode/internal/web"
)

func main() {
	if len(os.Args) > 1 {
		if dispatch(os.Args[1], os.Args[2:]) {
			return
		}
		// An unrecognised subcommand used to fall through to serve(). That is
		// how an older picode, asked to `provision` by a newer caller, quietly
		// came up as a *second* server — as root, in /root/.picode, on
		// whatever port was free. A typo did the same. Flags still belong to
		// the server; a bare word has to be a command we know.
		if !strings.HasPrefix(os.Args[1], "-") {
			fmt.Fprintf(os.Stderr, "picode: unknown command %q\n\n", os.Args[1])
			usage()
			os.Exit(2)
		}
	}
	serve()
}

// dispatch runs a subcommand and reports whether it handled the argument.
// Keeping the list here — rather than repeating it in the guard above — is
// what stops the two from drifting apart.
func dispatch(cmd string, args []string) bool {
	switch {
	case browserhost.IsHostArg(cmd):
		runBrowserHost()
	case cmd == "screenshot":
		runScreenshot(args)
	case cmd == "install":
		runInstall(args)
	case cmd == "provision":
		runProvision(args)
	case cmd == "update":
		runUpdate()
	case cmd == "deploy":
		runDeploy()
	case cmd == "uninstall":
		runUninstall(args)
	case cmd == "pair":
		runPair()
	case cmd == "token":
		runToken(args)
	case cmd == "gateway":
		runGateway(args)
	case cmd == "users":
		runUsers(args)
	case cmd == "extension-install":
		runExtensionInstall(args)
	case cmd == "extension-uninstall":
		runExtensionUninstall()
	case cmd == "help" || cmd == "-h" || cmd == "--help":
		usage()
	default:
		return false
	}
	return true
}

func usage() {
	fmt.Println(`picode — a browser ADE for Pi agents (server mode is the default)

Usage:
  picode [flags]              start the server
  picode pair                 print a one-time link to pair another device
  picode token [rotate]       print the install token path, or rotate it
  picode install [--env K=V]  copy to ~/.local/bin and start on Linux login (systemd --user)
    --env KEY=VALUE   service environment (repeatable), e.g. --env PICODE_DATA=/srv/picode;
                      written to ~/.config/systemd/user/picode.service.d/env.conf
  picode provision [flags]    converge this machine on what PiCode needs (ADR-0020)
    --dry-run       report what would change, touch nothing
    --json          emit results as JSON
    --user string   provision for this account (default: the current user)
  picode update               check GitHub for a newer release; verifies SHA256SUMS, then installs
  picode gateway              serve the shared box's front door (ADR-0051; needs /etc/picode/gateway.json)
  picode gateway install      root: binary to /usr/local/bin, config, system unit, start
  picode gateway status       config, certificate, whois self-test, members
  picode gateway uninstall    root: stop and remove the front door; --purge also deletes /etc/picode
  picode gateway oidc set P ID SECRET [--public-url URL]   root: Google/GitHub login for people off the tailnet (ADR-0052)
  picode gateway --plain 127.0.0.1:8480   also answer plain HTTP behind a TLS proxy (Caddy, Cloudflare Tunnel)
  picode provision --user U --shared --container [--remove]   root: the member's daemon in a systemd-nspawn container
  picode users add L U        root: map a Tailscale login to a Linux user; remove L; list
  picode provision --user U --shared   root: create/prepare a member's account and daemon
  picode deploy               replace the installed binary with this one and restart (repo)
  picode uninstall [--purge]  stop that; --purge also deletes ~/.picode
  picode extension-install    register the Chrome native host (ADR-0043)
    --server URL    a PiCode on another machine (writes <data>/remote.json)
    --token T       that server's install token (prompted if omitted)
    --ca FILE       PEM to trust for it (its mkcert rootCA.pem), optional
  picode extension-uninstall  remove that host registration
  picode screenshot [flags]   capture a page to PNG (visual-review loop)
    --url string    page to capture (required)
    --out string    destination PNG (required)
    --width int     viewport width (default 1440)
    --height int    viewport height (default 900)
    --full          capture the full page height
    --wait-ms int   settle time after page ready (default 500)

Environment:
  PICODE_HOST       bind host (default 0.0.0.0)
  PICODE_PORT       port or range, e.g. 8446 or 8445-8460
                    (default 8445-8455; the Settings UI overrides)
  PICODE_DATA       data dir (default ~/.picode)
  PICODE_INSECURE   =1 disables TLS (dev / behind proxy)

HTTPS is served with data-dir certs (mkcert via scripts/setup-cert.sh)
or a generated self-signed certificate. Discovery: <data>/server.json.`)
}

// shippable refuses to hand the service a binary with no UI inside. A disk
// build (ADR-0023) reads the UI from `internal/web/public` relative to its
// working directory, so installing one leaves the browser on "the UI has not
// been built yet" — or, worse, appears to work for as long as the process
// happens to run from the repository, and breaks the next time that directory
// is rebuilt. This is not checked inside install.Deploy: `picode update`
// deploys a *downloaded* release, and asking whether *this* binary embeds the
// UI would be the wrong question there.
func shippable(cmd string) {
	if web.Embedded() {
		return
	}
	log.Fatalf("%s: this binary has no UI embedded.\n"+
		"Build one with `make build` (which passes -tags embedui); a plain "+
		"`go build` produces a binary that reads the UI from disk.", cmd)
}

// envFlags collects repeatable --env KEY=VALUE.
type envFlags map[string]string

func (e envFlags) String() string { return "" }
func (e envFlags) Set(s string) error {
	k, v, err := install.ParseEnvFlag(s)
	if err != nil {
		return err
	}
	e[k] = v
	return nil
}

func runInstall(args []string) {
	shippable("install")
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	env := envFlags{}
	fs.Var(env, "env", "KEY=VALUE for the service environment (repeatable; ADR-0050)")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("install: %v", err)
	}
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("install: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("install: %v", err)
	}
	if len(env) > 0 {
		path, err := install.WriteEnvDropIn(home, env)
		if err != nil {
			log.Fatalf("install: %v", err)
		}
		fmt.Println("Service environment written to " + path)
	}
	fmt.Println("Installing PiCode (systemd --user)…")
	if err := install.Install(exe, home, os.Getenv("PATH")); err != nil {
		log.Fatalf("install: %v", err)
	}
	fmt.Println("Starts when this Linux user session starts.")
	fmt.Println("  https://localhost:8445")
	fmt.Println("  systemctl --user status picode")
}

func runDeploy() {
	shippable("deploy")
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("deploy: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("deploy: %v", err)
	}
	fmt.Println("Deploying this binary…")
	if err := install.Deploy(exe, home, os.Getenv("PATH")); err != nil {
		log.Fatalf("deploy: %v", err)
	}
	fmt.Println("Restarted.")
	fmt.Println("  https://localhost:8445")
}

func runUpdate() {
	fmt.Printf("This build is %s.\n", version.Build())
	rel, err := install.LatestRelease()
	if err != nil {
		if err.Error() == "no published release" {
			fmt.Println("No published release yet.")
			return
		}
		log.Fatalf("update: %v", err)
	}
	if !install.Newer(version.Version, rel.Tag) {
		fmt.Printf("Already up to date (%s).\n", version.Version)
		return
	}
	fmt.Printf("Version %s is out (you have %s).\n  %s\n", rel.Tag, version.Version, rel.URL)
	if rel.AssetURL == "" {
		fmt.Println("No binary for this OS in that release yet.")
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("update: %v", err)
	}
	tmp, err := os.CreateTemp("", "picode-")
	if err != nil {
		log.Fatalf("update: %v", err)
	}
	_ = tmp.Close()
	defer os.Remove(tmp.Name())
	if rel.SumsURL == "" {
		log.Fatalf("update: release %s has no %s — refusing to install an unverified binary", rel.Tag, install.SumsAsset)
	}
	fmt.Println("Downloading…")
	if err := install.Download(rel.AssetURL, tmp.Name()); err != nil {
		log.Fatalf("update: %v", err)
	}
	sums, err := install.Fetch(rel.SumsURL)
	if err != nil {
		log.Fatalf("update: %s: %v", install.SumsAsset, err)
	}
	if err := install.VerifySHA256(tmp.Name(), sums, rel.Asset); err != nil {
		log.Fatalf("update: %v", err)
	}
	fmt.Println("Checksum verified.")
	if err := install.Deploy(tmp.Name(), home, os.Getenv("PATH")); err != nil {
		log.Fatalf("update: %v", err)
	}
	fmt.Println("Updated to " + rel.Tag + ".")
	fmt.Println("  https://localhost:8445")
}

func runUninstall(args []string) {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	purge := fs.Bool("purge", false, "also delete ~/.picode")
	_ = fs.Parse(args)
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("uninstall: %v", err)
	}
	fmt.Println("Removing PiCode from systemd…")
	if err := install.Uninstall(home, *purge); err != nil {
		log.Fatalf("uninstall: %v", err)
	}
	if *purge {
		fmt.Println("Removed the service, binary, and ~/.picode.")
		return
	}
	fmt.Println("Removed the service. ~/.picode is still there.")
}

// serveState shares live port info with the HTTP handlers.
type serveState struct {
	cfg  atomic.Value // config.Config
	port atomic.Int64
}

func (s *serveState) snapshot() server.PortSnapshot {
	cfg := s.cfg.Load().(config.Config)
	port := int(s.port.Load())
	scheme := "https"
	if cfg.Insecure {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%d", scheme, advertiseHost(cfg.Host), port)
	if cfg.PublicURL != "" {
		url = cfg.PublicURL
	}
	return server.PortSnapshot{
		Current:    port,
		Configured: cfg.Port.String(),
		URL:        url,
		Host:       cfg.Host,
		PublicURL:  cfg.PublicURL,
	}
}

func serve() {
	dataDir := os.Getenv("PICODE_DATA")
	if dataDir == "" {
		dataDir = defaultDataDir()
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("data dir: %v", err)
	}
	unlock, err := proclock.Acquire(filepath.Join(dataDir, "picode.lock"))
	if err != nil {
		log.Fatalf("%v", err)
	}
	var unlockOnce sync.Once
	safeUnlock := func() { unlockOnce.Do(unlock) }
	defer safeUnlock()

	watchCtx, stopWatch := context.WithCancel(context.Background())
	defer stopWatch()
	if stamp, err := binwatch.Capture(); err != nil {
		log.Printf("picode: cannot watch binary: %v", err)
	} else {
		log.Printf("picode: binary %s", stamp.Path)
		binwatch.Watch(watchCtx, stamp, func() {
			if watchCtx.Err() != nil {
				return
			}
			safeUnlock()
			binwatch.Reexec(stamp.Path)
		})
	}

	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	runtime := rpc.NewRuntime("pi", st, nil)
	runtime.DataDir = dataDir
	defer runtime.StopAll()

	backupCtx, backupCancel := context.WithCancel(context.Background())
	defer backupCancel()
	bak := &backup.Engine{Store: st, DataDir: dataDir, Version: version.Version}
	go bak.Loop(backupCtx)

	rebindCh := make(chan struct{}, 1)
	state := &serveState{}

	// Change feed (ADR-0048): every committed store event fans out to
	// SSE subscribers and in-process listeners; presence and waiting
	// agents ride it as ephemeral notices.
	changes := &feed.Feed{Store: st}
	st.OnEvent = changes.Publish
	runtime.OnState = func(agentID string, streaming, waiting bool, dialog *rpc.UIDialog) {
		changes.Ephemeral("agent.state", map[string]any{"agentId": agentID, "streaming": streaming, "waiting": waiting, "dialog": dialog})
	}
	runtime.OnUsage = func(agentID string, u rpc.Usage) {
		changes.Ephemeral("agent.usage", map[string]any{"agentId": agentID, "input": u.Input, "output": u.Output,
			"cacheRead": u.CacheRead, "cacheWrite": u.CacheWrite, "totalTokens": u.TotalTokens, "cost": u.Cost})
	}
	runtime.OnWaiting = func(agentID, agentName, title, message string) {
		changes.Ephemeral("agent.waiting", map[string]string{"agentId": agentID, "agentName": agentName, "title": title, "message": message})
	}

	// Web Push (ADR-0047): one VAPID key per install; the notifier
	// consumes the feed and stays quiet while a browser on this machine
	// is alive.
	devices := presence.New(share.ReachableIPv4())
	devices.OnChange = func(d presence.Device) {
		if d.Online {
			changes.Ephemeral("device.online", d)
		} else {
			changes.Ephemeral("device.offline", d)
		}
	}
	go devices.Watch(backupCtx, 5*time.Second) // process-scoped, like the backup loop

	// Session housekeeping (ADR-0049): PruneSessions existed but had no
	// caller, so revoked and expired rows piled up forever —Devices showed
	// every browser session ever minted. Daily sweep, 7-day retention
	// (ListSessions already hides dead rows, so the lag is invisible).
	go func() {
		prune := func() {
			if _, err := st.PruneSessions(time.Now().Add(-7 * 24 * time.Hour)); err != nil {
				log.Printf("auth: prune sessions: %v", err)
			}
		}
		prune()
		t := time.NewTicker(24 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-backupCtx.Done():
				return
			case <-t.C:
				prune()
			}
		}
	}()
	var notifier *push.Notifier
	if keys, err := push.LoadOrCreate(dataDir); err != nil {
		log.Printf("push: disabled: %v", err)
	} else {
		notifier = &push.Notifier{
			Store:    st,
			Sender:   &push.Sender{Keys: keys, Subject: "https://github.com/cfpperche/picode"},
			Presence: devices,
			Log:      log.Default(),
		}
		changes.Listen(notifier.OnEvent)
	}

	// Request gate (ADR-0049): install token at <data>/token, loopback
	// auto-pairs in the default mode, everything else pairs a device.
	hostname, _ := os.Hostname()
	gate, err := auth.New(auth.Config{
		Store: st, DataDir: dataDir, Insecure: os.Getenv("PICODE_INSECURE") == "1", Hostname: hostname,
		PublicURL:   func() string { v, _, _ := st.GetSetting("server.public_url"); return v },
		SessionLive: devices.SessionLive, // ADR-0049 amendment: reuse must not rotate an active session's cookie
	})
	if err != nil {
		log.Fatalf("auth: %v", err)
	}

	// The TUI keeps the writer, so startup only reconciles truth: pending
	// replies are settled against their session JSONL and unanswered items
	// reopen for retry (ADR-0060).
	server.ReconcilePendingReplies(st, dataDir)

	dockerService, err := docker.NewService(backupCtx, st, nil, func() {
		changes.Ephemeral("docker.changed", map[string]any{})
	})
	if err != nil {
		log.Fatalf("docker operations: %v", err)
	}
	defer dockerService.Close()

	deps := server.Deps{
		Store:    st,
		Auth:     gate,
		Tmux:     tmux.New(),
		Runtime:  runtime,
		AgentCmd: "pi", // ADR-0003: user-installed pi
		DataDir:  dataDir,
		Backup:   bak,
		Presence: devices,
		Push:     notifier,
		Feed:     changes,
		Docker:   dockerService,
		Rebind: func() {
			select {
			case rebindCh <- struct{}{}:
			default:
			}
		},
		PortSnapshot: state.snapshot,
		// Guest terminal state (ADR-0056 tier 1): live signal registry, also
		// read by the sweep watcher below.
		TermStates:   server.NewTermStates(),
		TermRuntimes: server.NewTermRuntimes(),
		// Apps host (ADR-0036). PICODE_DEMO_APP=1 adds the hidden QA app;
		// the env read lives here, never inside internal/apps.
		Apps: apps.NewRegistry(apps.BuiltIns(os.Getenv("PICODE_DEMO_APP") == "1")...),
	}

	// tmux watcher (ADR-0048 follow-up): one scrape per tick for the whole
	// fleet, published as agent.tui, instead of one per browser per 3 s.
	go server.StartTuiWatch(backupCtx, deps, 3*time.Second)

	// MCP live watcher (ADR-0048 follow-up): one snapshot read per tick for
	// the whole fleet, published as mcp.updated, instead of one poll per
	// open panel per 2.5 s.
	go server.StartMcpWatch(backupCtx, deps, 3*time.Second)

	// Git watcher (ADR-0048 follow-up): one Inspect per directory per tick
	// for the whole fleet, published as git.updated, so the sidebar's
	// branch pills and the file tree follow commits instead of waiting
	// for a fleet refetch or a manual Refresh.
	go server.StartGitWatch(backupCtx, deps, 3*time.Second)

	// Guest terminal state sweep (ADR-0056 tier 1): expires stale
	// "working" reports so a silenced sensor cannot spin forever.
	go server.StartTermStateSweep(backupCtx, deps, time.Minute)

	// Terminal CLI presence watcher (ADR-0062): validates wrapper leases
	// against the live pane process and reconciles sessions created before
	// runId reporting existed. It never infers activity from pane pixels.
	go server.StartTermRuntimeWatch(backupCtx, deps, 3*time.Second)

	// Package updates watcher (ADR-0048 follow-up): one scan set per tick
	// for the whole fleet, published as packages.updates only when a
	// scope's result changes, instead of one 30 min poll per browser.
	go server.StartPackageUpdatesWatch(backupCtx, deps, 30*time.Minute)

	// Provider plan windows (ADR-0058): keep the active slot of each
	// meterable provider warm so #/providers can show a true number
	// without eight vendor calls on every page load.
	go server.StartUsageRefresh(backupCtx, deps, providerusage.DefaultRefresh)

	// Automations scheduler (ADR-0045): lives with the process, not the
	// HTTP server, so a rebind never drops a schedule.
	autoCtx, autoCancel := context.WithCancel(context.Background())
	defer autoCancel()
	go (&automate.Engine{Store: st, Runner: server.AutomationRunner(deps)}).Loop(autoCtx)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Initial bind + banner.
	cfg, err := config.Resolve(st.GetSetting)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	deps.BindHost = cfg.Host
	deps.Insecure = cfg.Insecure
	state.cfg.Store(cfg)
	if !cfg.Insecure {
		// Tailscale leaf for the MagicDNS name (ADR-0050 B.2): issued when
		// missing or near expiry, checked daily; silent without Tailscale.
		go tlsutil.KeepTailscaleCert(backupCtx, dataDir, share.MagicDNSName, 24*time.Hour)
	}

	srv, port, err := bindAndServe(cfg, deps)
	if err != nil {
		log.Fatalf("bind: %v (configured %s)", err, cfg.Port)
	}
	state.port.Store(int64(port))
	writeServerJSON(dataDir, cfg, port)
	logStartup(cfg, dataDir, port)
	if !cfg.Insecure {
		share.EnsureTrustHTTP() // hands out the mkcert CA; pointless behind a gateway
	}

	// Serving loop: rebind on demand — bind-new-then-drop-old, revert the
	// setting on failure so the current server never disappears.
	for {
		select {
		case <-rebindCh:
			log.Printf("server: port change requested — rebinding")
			newCfg, rerr := config.Resolve(st.GetSetting)
			if rerr != nil {
				log.Printf("server: bad port setting (%v) — keeping current", rerr)
				continue
			}
			if newCfg.Host == cfg.Host && newCfg.Port == cfg.Port {
				// Only advisory state changed (public URL): refresh what
				// clients read, keep the listener.
				cfg = newCfg
				deps.BindHost = cfg.Host
				state.cfg.Store(cfg)
				writeServerJSON(dataDir, cfg, port)
				logStartup(cfg, dataDir, port)
				continue
			}
			newSrv, newPort, berr := bindAndServe(newCfg, deps)
			if berr != nil && newCfg.Host != cfg.Host {
				// A host change on the same port cannot bind-new-first:
				// 0.0.0.0 and a specific address overlap. Drop the old
				// listener, bind the new one, and put the old one back if
				// that fails — the server never stays down.
				log.Printf("server: %v — moving the listener to %s", berr, newCfg.Host)
				gracefulShutdown(srv)
				newSrv, newPort, berr = bindAndServe(newCfg, deps)
				if berr != nil {
					log.Printf("server: bind on %s failed (%v) — restoring %s", newCfg.Host, berr, cfg.Host)
					_ = st.SetSetting(config.HostSettingKey, cfg.Host)
					_ = st.SetSetting(config.PortSettingKey, cfg.Port.String())
					if srv, port, berr = bindAndServe(cfg, deps); berr != nil {
						log.Fatalf("server: cannot restore the listener on %s: %v", cfg.Host, berr)
					}
					continue
				}
				srv, port, cfg = newSrv, newPort, newCfg
			} else if berr != nil {
				log.Printf("server: rebind failed (%v) — reverting to %s on %s, keeping port %d",
					berr, cfg.Port, cfg.Host, port)
				_ = st.SetSetting(config.PortSettingKey, cfg.Port.String())
				_ = st.SetSetting(config.HostSettingKey, cfg.Host)
				continue
			} else {
				gracefulShutdown(srv)
				srv, port, cfg = newSrv, newPort, newCfg
			}
			deps.BindHost = cfg.Host
			deps.Insecure = cfg.Insecure
			state.cfg.Store(cfg)
			state.port.Store(int64(newPort))
			writeServerJSON(dataDir, cfg, newPort)
			logStartup(cfg, dataDir, newPort)

		case sig := <-sigs:
			stopWatch()
			log.Printf("server: %v — shutting down", sig)
			gracefulShutdown(srv)
			return
		}
	}
}

func logStartup(cfg config.Config, dataDir string, port int) {
	scheme := "https"
	if cfg.Insecure {
		scheme = "http"
	}
	log.Printf("")
	log.Printf("  PiCode listening on  %s://%s:%d", scheme, advertiseHost(cfg.Host), port)
	if cfg.PublicURL != "" {
		log.Printf("  public URL           %s", cfg.PublicURL)
	}
	log.Printf("  data dir             %s", dataDir)
	log.Printf("")
}

// keepsLoopback: a bind to one specific outside address (the tailnet, a
// LAN card) still listens on 127.0.0.1, so this machine — its browser,
// its scripts, `picode pair` — never loses the server (ADR-0050). Only
// an explicit loopback bind, or an unspecified one that already covers
// it, needs nothing extra.
func keepsLoopback(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && !ip.IsUnspecified() && !ip.IsLoopback()
}

// bindAndServe tries the configured port range in order and serves the app
// on the first free port — on the configured host, plus loopback when the
// host is a specific outside address.
func bindAndServe(cfg config.Config, deps server.Deps) (*http.Server, int, error) {
	handler := server.New("127.0.0.1:0", deps).Handler // addr unused; we serve explicitly

	var lns []net.Listener
	var port int
	var lastErr error
	for p := cfg.Port.Min; p <= cfg.Port.Max; p++ {
		l, err := net.Listen("tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(p)))
		if err != nil {
			lastErr = err
			continue
		}
		lns = append(lns, l)
		port = p
		break
	}
	if len(lns) == 0 {
		return nil, 0, fmt.Errorf("no free port in %s: %w", cfg.Port, lastErr)
	}
	if keepsLoopback(cfg.Host) {
		if l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port))); err == nil {
			lns = append(lns, l)
		} else {
			log.Printf("server: loopback listener on %d unavailable (%v) — only %s answers", port, err, cfg.Host)
		}
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if !cfg.Insecure {
		if _, cerr := tlsutil.Ensure(cfg.DataDir); cerr != nil {
			log.Fatalf("tls: %v", cerr)
		}
		tlsutil.WarnIfExpiring(cfg.DataDir, 30*24*time.Hour)
		srv.TLSConfig = tlsutil.LiveConfig(cfg.DataDir)
	}
	for _, ln := range lns {
		go func(ln net.Listener) {
			var err error
			if cfg.Insecure {
				err = srv.Serve(ln)
			} else {
				err = srv.ServeTLS(ln, "", "")
			}
			if err != nil && err != http.ErrServerClosed {
				log.Fatalf("serve: %v", err)
			}
		}(ln)
	}
	return srv, port, nil
}

// writeServerJSON drops the discovery file for scripts/CLI. url is the
// address a client on this machine uses (localhost unless the bind is a
// specific address); publicUrl is the configured origin for everyone
// else (ADR-0050), "" when none.
func writeServerJSON(dataDir string, cfg config.Config, port int) {
	host := advertiseHost(cfg.Host)
	scheme := "https"
	if cfg.Insecure {
		scheme = "http"
	}
	body := fmt.Sprintf(`{"url":%q,"scheme":%q,"host":%q,"bind":%q,"port":%d,"publicUrl":%q,"pid":%d,"time":%q}`,
		fmt.Sprintf("%s://%s:%d", scheme, host, port), scheme, host, cfg.Host, port, cfg.PublicURL, os.Getpid(),
		time.Now().UTC().Format(time.RFC3339))
	_ = os.MkdirAll(dataDir, 0o755)
	_ = os.WriteFile(filepath.Join(dataDir, "server.json"), []byte(body+"\n"), 0o644)
}

// advertiseHost is the name a client on this machine dials: localhost
// when the bind covers loopback, else the specific address bound.
func advertiseHost(host string) string {
	if host == "" || host == "0.0.0.0" || host == "::" || host == "127.0.0.1" || host == "::1" {
		return "localhost"
	}
	return host
}

// defaultDataDir returns ~/.picode.
func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".picode")
	}
	return ".picode"
}

func runScreenshot(args []string) {
	fs := flag.NewFlagSet("screenshot", flag.ExitOnError)
	url := fs.String("url", "", "page to capture (required)")
	out := fs.String("out", "", "destination PNG (required)")
	width := fs.Int("width", 1440, "viewport width")
	height := fs.Int("height", 900, "viewport height")
	full := fs.Bool("full", false, "capture full page height")
	wait := fs.Int("wait-ms", 500, "settle time after page ready")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("screenshot: %v", err)
	}

	if err := screenshot.Capture(context.Background(), screenshot.Options{
		URL: *url, Out: *out, Width: *width, Height: *height, Full: *full, WaitMS: *wait,
	}); err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Println(*out)
}

// runPair prints a one-time pairing link for another device (ADR-0049):
// the recovery path when no browser is paired yet. It talks to the
// running daemon with the install token.
func runPair() {
	base, err := browserhost.ReadServerURL()
	if err != nil {
		fmt.Fprintln(os.Stderr, "PiCode is not running (no server.json)")
		os.Exit(1)
	}
	tok, err := os.ReadFile(filepath.Join(browserhost.DataDir(), "token"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "no install token yet — start PiCode once")
		os.Exit(1)
	}
	req, _ := http.NewRequest(http.MethodPost, base+"/api/auth/pairings", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(tok)))
	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec
	res, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "PiCode is not reachable:", err)
		os.Exit(1)
	}
	defer res.Body.Close()
	var out struct {
		URL       string `json:"url"`
		ExpiresAt string `json:"expiresAt"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil || out.URL == "" {
		fmt.Fprintf(os.Stderr, "pairing failed: HTTP %d\n", res.StatusCode)
		os.Exit(1)
	}
	fmt.Println("Open this link on the device to pair (valid 10 minutes):")
	fmt.Println("  " + out.URL)
}

// runToken prints the install token path (`picode token`) or rotates it
// (`picode token rotate`) — rotation restarts nothing; the daemon reads
// the file again on its next request.
func runToken(args []string) {
	path := filepath.Join(browserhost.DataDir(), "token")
	if len(args) > 0 && args[0] == "rotate" {
		base, err := browserhost.ReadServerURL()
		if err != nil {
			fmt.Fprintln(os.Stderr, "PiCode is not running (no server.json)")
			os.Exit(1)
		}
		tok, _ := os.ReadFile(path)
		req, _ := http.NewRequest(http.MethodPost, base+"/api/auth/token/rotate", nil)
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(tok)))
		client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec
		res, err := client.Do(req)
		if err != nil || res.StatusCode != http.StatusOK {
			fmt.Fprintln(os.Stderr, "rotation failed")
			os.Exit(1)
		}
		res.Body.Close()
		fmt.Println("Rotated. Scripts read the new token from " + path)
		return
	}
	fmt.Println(path)
}
