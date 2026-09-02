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
	"github.com/cfpperche/picode/internal/automate"
	"github.com/cfpperche/picode/internal/backup"
	"github.com/cfpperche/picode/internal/binwatch"
	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/config"
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
		runInstall()
	case cmd == "provision":
		runProvision(args)
	case cmd == "update":
		runUpdate()
	case cmd == "deploy":
		runDeploy()
	case cmd == "uninstall":
		runUninstall(args)
	case cmd == "extension-install":
		runExtensionInstall()
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
  picode install              copy to ~/.local/bin and start on Linux login (systemd --user)
  picode provision [flags]    converge this machine on what PiCode needs (ADR-0020)
    --dry-run       report what would change, touch nothing
    --json          emit results as JSON
    --user string   provision for this account (default: the current user)
  picode update               check GitHub for a newer release (and install it if there is one)
  picode deploy               replace the installed binary with this one and restart (repo)
  picode uninstall [--purge]  stop that; --purge also deletes ~/.picode
  picode extension-install    register the Chrome native host (ADR-0043)
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

func runInstall() {
	shippable("install")
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("install: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("install: %v", err)
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
	fmt.Println("Downloading…")
	if err := install.Download(rel.AssetURL, tmp.Name()); err != nil {
		log.Fatalf("update: %v", err)
	}
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
	return server.PortSnapshot{
		Current:    port,
		Configured: cfg.Port.String(),
		URL:        fmt.Sprintf("%s://%s:%d", scheme, advertiseHost(cfg.Host), port),
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

	if stamp, err := binwatch.Capture(); err != nil {
		log.Printf("picode: cannot watch binary: %v", err)
	} else {
		log.Printf("picode: binary %s", stamp.Path)
		binwatch.Watch(stamp, func() {
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
	runtime.OnWaiting = func(agentID, agentName, title, message string) {
		changes.Ephemeral("agent.waiting", map[string]string{"agentId": agentID, "agentName": agentName, "title": title, "message": message})
	}

	// Web Push (ADR-0047): one VAPID key per install; the notifier
	// consumes the feed and stays quiet while a browser on this machine
	// is alive.
	devices := presence.New(share.ReachableIPv4())
	devices.OnChange = func(d presence.Device) { changes.Ephemeral("device.online", d) }
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

	deps := server.Deps{
		Store:    st,
		Tmux:     tmux.New(),
		Runtime:  runtime,
		AgentCmd: "pi", // ADR-0003: user-installed pi
		DataDir:  dataDir,
		Backup:   bak,
		Presence: devices,
		Push:     notifier,
		Feed:     changes,
		Rebind: func() {
			select {
			case rebindCh <- struct{}{}:
			default:
			}
		},
		PortSnapshot: state.snapshot,
		// Apps host (ADR-0036). PICODE_DEMO_APP=1 adds the hidden QA app;
		// the env read lives here, never inside internal/apps.
		Apps: apps.NewRegistry(apps.BuiltIns(os.Getenv("PICODE_DEMO_APP") == "1")...),
	}

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

	srv, port, err := bindAndServe(cfg, deps)
	if err != nil {
		log.Fatalf("bind: %v (configured %s)", err, cfg.Port)
	}
	state.port.Store(int64(port))
	writeServerJSON(dataDir, cfg, port)
	logStartup(cfg, dataDir, port)
	share.EnsureTrustHTTP()

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
			newSrv, newPort, berr := bindAndServe(newCfg, deps)
			if berr != nil {
				log.Printf("server: rebind failed (%v) — reverting to %s, keeping port %d",
					berr, cfg.Port, port)
				_ = st.SetSetting(config.PortSettingKey, cfg.Port.String())
				continue
			}
			gracefulShutdown(srv)
			srv, port, cfg = newSrv, newPort, newCfg
			deps.BindHost = cfg.Host
			deps.Insecure = cfg.Insecure
			state.cfg.Store(cfg)
			state.port.Store(int64(newPort))
			writeServerJSON(dataDir, cfg, newPort)
			logStartup(cfg, dataDir, newPort)

		case sig := <-sigs:
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
	log.Printf("  data dir             %s", dataDir)
	log.Printf("")
}

func gracefulShutdown(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = srv.Shutdown(ctx)
	cancel()
}

// bindAndServe tries the configured port range in order and serves the app
// on the first free port.
func bindAndServe(cfg config.Config, deps server.Deps) (*http.Server, int, error) {
	handler := server.New("127.0.0.1:0", deps).Handler // addr unused; we serve explicitly

	var ln net.Listener
	var port int
	var lastErr error
	for p := cfg.Port.Min; p <= cfg.Port.Max; p++ {
		l, err := net.Listen("tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(p)))
		if err != nil {
			lastErr = err
			continue
		}
		ln = l
		port = p
		break
	}
	if ln == nil {
		return nil, 0, fmt.Errorf("no free port in %s: %w", cfg.Port, lastErr)
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		var err error
		if cfg.Insecure {
			err = srv.Serve(ln)
		} else {
			if _, cerr := tlsutil.Ensure(cfg.DataDir); cerr != nil {
				log.Fatalf("tls: %v", cerr)
			}
			tlsutil.WarnIfExpiring(cfg.DataDir, 30*24*time.Hour)
			srv.TLSConfig = tlsutil.LiveConfig(cfg.DataDir)
			err = srv.ServeTLS(ln, "", "")
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()
	return srv, port, nil
}

// writeServerJSON drops the discovery file for scripts/CLI.
func writeServerJSON(dataDir string, cfg config.Config, port int) {
	host := advertiseHost(cfg.Host)
	scheme := "https"
	if cfg.Insecure {
		scheme = "http"
	}
	body := fmt.Sprintf(`{"url":%q,"scheme":%q,"host":%q,"port":%d,"pid":%d,"time":%q}`,
		fmt.Sprintf("%s://%s:%d", scheme, host, port), scheme, host, port, os.Getpid(),
		time.Now().UTC().Format(time.RFC3339))
	_ = os.MkdirAll(dataDir, 0o755)
	_ = os.WriteFile(filepath.Join(dataDir, "server.json"), []byte(body+"\n"), 0o644)
}

func advertiseHost(host string) string {
	if host == "0.0.0.0" || host == "::" {
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
