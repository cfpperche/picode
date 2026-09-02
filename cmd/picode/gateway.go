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
	"strings"
	"syscall"
	"time"

	"github.com/cfpperche/picode/internal/gateway"
	"github.com/cfpperche/picode/internal/install"
	"github.com/cfpperche/picode/internal/share"
	"github.com/cfpperche/picode/internal/tlsutil"
)

// GatewayBin is where `picode gateway install` puts the binary so every
// member unit and the gateway run the same build.
const GatewayBin = "/usr/local/bin/picode"

// runGateway: `picode gateway [install|status] [--config PATH]`.
func runGateway(args []string) {
	sub := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("gateway", flag.ExitOnError)
	cfgPath := fs.String("config", gateway.DefaultConfigPath, "gateway config")
	plain := fs.String("plain", "", "also serve plain HTTP on this loopback address, behind a TLS proxy (Caddy, Cloudflare Tunnel) — or alone for tests")
	fs.StringVar(plain, "insecure-listen", "", "alias of --plain")
	purge := fs.Bool("purge", false, "uninstall: also delete the config directory (config, certificate) and the shared binary")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("gateway: %v", err)
	}
	switch sub {
	case "":
		serveGateway(*cfgPath, *plain)
	case "oidc":
		gatewayOIDC(*cfgPath, fs.Args())
	case "install":
		installGateway(*cfgPath)
	case "uninstall":
		uninstallGateway(*cfgPath, *purge)
	case "status":
		gatewayStatus(*cfgPath)
	default:
		fmt.Fprintf(os.Stderr, "picode gateway: unknown subcommand %q\n", sub)
		os.Exit(2)
	}
}

func serveGateway(cfgPath, plain string) {
	cfg, err := gateway.Load(cfgPath)
	if err != nil {
		log.Fatalf("gateway: %v (run `picode gateway install`)", err)
	}
	if plain == "" {
		plain = cfg.PlainListen
	}
	sec, err := gateway.LoadSecrets(gateway.SecretPath(cfgPath))
	if err != nil {
		log.Fatalf("gateway: %v", err)
	}
	for name, pc := range cfg.OIDC {
		if pc.Overridden() && plain == "" {
			log.Fatalf("gateway: provider %s has hand-set endpoints (a test seam); refused on the tailnet listener", name)
		}
	}
	s, err := gateway.New(cfg, sec)
	if err != nil {
		log.Fatalf("gateway: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ln net.Listener
	srv := &http.Server{Handler: s, ReadHeaderTimeout: 10 * time.Second}
	// The tailnet door: skipped when only a plain listener is asked for
	// (a scratch run, or a box that is public-only behind a proxy).
	tailnetDoor := plain == "" || cfg.PlainListen != ""
	if plain != "" {
		host, _, _ := net.SplitHostPort(plain)
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			log.Fatalf("gateway: --plain must be a loopback address (a TLS proxy sits in front), got %q", plain)
		}
		if !tailnetDoor {
			s.Secure = false // scratch: the browser talks plain http to us
		}
		// Test seam: PICODE_GATEWAY_FAKE_IDENTITY=ip=login,ip=login — only
		// with a loopback plain-HTTP listener, never on the tailnet.
		if fake := os.Getenv("PICODE_GATEWAY_FAKE_IDENTITY"); fake != "" {
			s.Fake = map[string]string{}
			for _, kv := range strings.Split(fake, ",") {
				if ip, login, ok := strings.Cut(strings.TrimSpace(kv), "="); ok {
					s.Fake[ip] = login
				}
			}
			log.Printf("gateway: FAKE identities in use: %v", s.Fake)
		}
		// PICODE_GATEWAY_FAKE_HOMES=linux=/dir,... — members' homes for a
		// scratch run where one Linux user stands in for several.
		if homes := os.Getenv("PICODE_GATEWAY_FAKE_HOMES"); homes != "" {
			m := map[string]string{}
			for _, kv := range strings.Split(homes, ",") {
				if u, dir, ok := strings.Cut(strings.TrimSpace(kv), "="); ok {
					m[u] = dir
				}
			}
			s.Resolve = func(linux string) (gateway.Backend, error) {
				if dir, ok := m[linux]; ok {
					return gateway.ResolveHome(linux, dir)
				}
				return gateway.Resolve(linux)
			}
			log.Printf("gateway: FAKE homes in use: %v", m)
		}
		pln, err := net.Listen("tcp", plain)
		if err != nil {
			log.Fatalf("gateway: %v", err)
		}
		log.Printf("gateway: plain HTTP on %s (behind a proxy; trusted: %v)", plain, cfg.TrustedProxies)
		go func() {
			if err := srv.Serve(pln); err != nil && err != http.ErrServerClosed {
				log.Fatalf("gateway: %v", err)
			}
		}()
	}
	if tailnetDoor {
		if _, err := tlsutil.Ensure(cfg.DataDir); err != nil { // self-signed until the Tailscale leaf lands
			log.Fatalf("gateway: tls: %v", err)
		}
		go tlsutil.KeepTailscaleCert(ctx, cfg.DataDir, func() string { return cfg.Hostname }, 24*time.Hour)
		srv.TLSConfig = tlsutil.LiveConfig(cfg.DataDir)
		if ln, err = net.Listen("tcp", cfg.Listen); err != nil {
			log.Fatalf("gateway: %v", err)
		}
		log.Printf("gateway: https on %s for %s (%d member(s))", cfg.Listen, cfg.Hostname, len(cfg.Users))
		go func() {
			if err := srv.ServeTLS(ln, "", ""); err != nil && err != http.ErrServerClosed {
				log.Fatalf("gateway: %v", err)
			}
		}()
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	log.Printf("gateway: %v — shutting down", sig)
	sctx, scancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer scancel()
	_ = srv.Shutdown(sctx)
}

func mustRoot(what string) {
	if os.Geteuid() != 0 {
		log.Fatalf("%s needs root (sudo)", what)
	}
}

func installGateway(cfgPath string) {
	shippable("gateway install")
	mustRoot("picode gateway install")
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("gateway install: %v", err)
	}
	if err := install.CopyExe(exe, GatewayBin); err != nil {
		log.Fatalf("gateway install: copy to %s: %v", GatewayBin, err)
	}
	if _, err := gateway.Load(cfgPath); err != nil {
		name := share.MagicDNSName()
		if name == "" {
			log.Fatalf("gateway install: Tailscale is not running or has no MagicDNS name — this box needs one")
		}
		if err := gateway.Save(cfgPath, gateway.Default(name)); err != nil {
			log.Fatalf("gateway install: %v", err)
		}
		fmt.Println("Wrote " + cfgPath + " for " + name)
	}
	if err := os.WriteFile(install.GatewayUnitPath, []byte(install.GatewayUnitFile(GatewayBin, cfgPath)), 0o644); err != nil {
		log.Fatalf("gateway install: %v", err)
	}
	for _, args := range [][]string{{"daemon-reload"}, {"enable", "--now", install.GatewayUnitName}} {
		if err := install.Run("systemctl", args...); err != nil {
			log.Fatalf("gateway install: systemctl %s: %v", strings.Join(args, " "), err)
		}
	}
	fmt.Println("Gateway installed and running (" + install.GatewayUnitName + ").")
	fmt.Println("  add members:   picode users add <tailscale login> <linux user>")
	fmt.Println("  create one:    picode provision --user <linux user> --shared")
}

// uninstallGateway stops and removes the front door. Members' daemons
// and data are untouched: they are ordinary user units. --purge also
// removes the config directory (map + certificate) and the shared
// binary — only sensible once no member unit points at it.
func uninstallGateway(cfgPath string, purge bool) {
	mustRoot("picode gateway uninstall")
	_ = install.Run("systemctl", "disable", "--now", install.GatewayUnitName)
	if err := os.Remove(install.GatewayUnitPath); err != nil && !os.IsNotExist(err) {
		log.Fatalf("gateway uninstall: %v", err)
	}
	_ = install.Run("systemctl", "daemon-reload")
	fmt.Println("Gateway stopped and its unit removed.")
	if !purge {
		fmt.Println("Kept: " + cfgPath + " (members map, certificate) and " + GatewayBin + ". Add --purge to delete them.")
		fmt.Println("Members' daemons keep running as their own users; stop one with: runuser -u <user> -- picode uninstall [--purge]")
		return
	}
	dir := filepath.Dir(cfgPath)
	if err := os.RemoveAll(dir); err != nil {
		log.Fatalf("gateway uninstall: %v", err)
	}
	fmt.Println("Removed " + dir + ".")
	if members := memberUnitsUsing(GatewayBin); len(members) > 0 {
		fmt.Println("Kept " + GatewayBin + ": these members' units run it — " + strings.Join(members, ", "))
	} else if err := os.Remove(GatewayBin); err == nil {
		fmt.Println("Removed " + GatewayBin + ".")
	}
}

// memberUnitsUsing lists Linux users whose picode.service runs bin.
func memberUnitsUsing(bin string) []string {
	var out []string
	homes, _ := filepath.Glob("/home/*/.config/systemd/user/" + install.UnitName)
	for _, unit := range homes {
		b, err := os.ReadFile(unit)
		if err == nil && strings.Contains(string(b), "ExecStart="+bin) {
			out = append(out, filepath.Base(unit[:len(unit)-len("/.config/systemd/user/"+install.UnitName)]))
		}
	}
	return out
}

func gatewayStatus(cfgPath string) {
	cfg, err := gateway.Load(cfgPath)
	if err != nil {
		log.Fatalf("gateway status: %v", err)
	}
	fmt.Printf("config    %s\nhostname  %s\nlisten    %s\n", cfgPath, cfg.Hostname, cfg.Listen)
	st := tlsutil.TailscaleLeaf(cfg.DataDir, cfg.Hostname)
	switch {
	case st.Present && st.Covers:
		fmt.Printf("cert      Tailscale leaf for %s, valid until %s\n", cfg.Hostname, st.NotAfter.Format("2006-01-02"))
	default:
		fmt.Printf("cert      none yet — the gateway issues it on start (needs HTTPS certificates enabled for the tailnet)\n")
	}
	if ip := share.TailscaleIPv4(); ip != "" {
		id := gateway.NewIdentity(0)
		if login, node, err := id.Whois(context.Background(), ip); err == nil {
			fmt.Printf("whois     %s → %s (%s)\n", ip, login, node)
		} else {
			fmt.Printf("whois     %v\n", err)
		}
	}
	if cfg.PublicURL != "" || len(cfg.OIDC) > 0 {
		fmt.Printf("public    %s (plain listener %s, trusted proxies %v)\n", cfg.PublicURL, cfg.PlainListen, cfg.TrustedProxies)
		sec, _ := gateway.LoadSecrets(gateway.SecretPath(cfgPath))
		for name := range cfg.OIDC {
			state := "no client id/secret — picode gateway oidc set " + name + " <id> <secret>"
			if ps, ok := sec.Providers[name]; ok && ps.ClientID != "" && ps.ClientSecret != "" {
				state = "callback " + strings.TrimRight(cfg.PublicURL, "/") + "/-/auth/callback/" + name
			}
			fmt.Printf("  %-8s %s\n", name, state)
		}
	}
	fmt.Printf("members   %d\n", len(cfg.Users))
	for _, l := range cfg.Logins() {
		u := cfg.Users[l]
		state := "running"
		if _, err := gateway.Resolve(u); err != nil {
			state = err.Error()
		}
		fmt.Printf("  %-32s → %-12s %s\n", l, u, state)
	}
}

// runUsers: `picode users add <login> <linux> | remove <login> | list`.
func runUsers(args []string) {
	fs := flag.NewFlagSet("users", flag.ExitOnError)
	cfgPath := fs.String("config", gateway.DefaultConfigPath, "gateway config")
	// flags may come after the verb; parse whatever is flag-shaped.
	var words []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			_ = fs.Parse([]string{a})
			continue
		}
		words = append(words, a)
	}
	cfg, err := gateway.Load(*cfgPath)
	if err != nil {
		log.Fatalf("users: %v (run `picode gateway install` first)", err)
	}
	if len(words) == 0 {
		words = []string{"list"}
	}
	switch words[0] {
	case "list":
		if len(cfg.Users) == 0 {
			fmt.Println("No members yet. picode users add <tailscale login> <linux user>")
			return
		}
		for _, l := range cfg.Logins() {
			fmt.Printf("%-32s → %s\n", l, cfg.Users[l])
		}
	case "add":
		if len(words) != 3 {
			log.Fatalf("usage: picode users add <tailscale login> <linux user>")
		}
		mustRoot("picode users add")
		if err := cfg.AddUser(words[1], words[2]); err != nil {
			log.Fatalf("users: %v", err)
		}
		if err := gateway.Save(*cfgPath, cfg); err != nil {
			log.Fatalf("users: %v", err)
		}
		fmt.Printf("%s → %s. The gateway reads the file on every request; nothing to restart.\n", words[1], words[2])
	case "remove":
		if len(words) != 2 {
			log.Fatalf("usage: picode users remove <tailscale login>")
		}
		mustRoot("picode users remove")
		cfg.RemoveUser(words[1])
		if err := gateway.Save(*cfgPath, cfg); err != nil {
			log.Fatalf("users: %v", err)
		}
		fmt.Printf("%s removed. Their daemon keeps running; stop it with `runuser -u <user> -- picode uninstall`.\n", words[1])
	default:
		log.Fatalf("users: unknown verb %q", words[0])
	}
}

// gatewayOIDC: `picode gateway oidc set <google|github> <client id> <client secret> [--public-url URL]`
// and `... oidc unset <provider>`. Secrets go to gateway.secret.json
// (0600); the provider's presence and the public URL to gateway.json.
func gatewayOIDC(cfgPath string, args []string) {
	mustRoot("picode gateway oidc")
	var words []string
	pub := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--public-url" && i+1 < len(args) {
			pub = args[i+1]
			i++
			continue
		}
		words = append(words, args[i])
	}
	cfg, err := gateway.Load(cfgPath)
	if err != nil {
		log.Fatalf("gateway oidc: %v (run `picode gateway install` first)", err)
	}
	sec, err := gateway.LoadSecrets(gateway.SecretPath(cfgPath))
	if err != nil {
		log.Fatalf("gateway oidc: %v", err)
	}
	if cfg.OIDC == nil {
		cfg.OIDC = map[string]gateway.ProviderConfig{}
	}
	switch {
	case len(words) == 4 && words[0] == "set":
		name := words[1]
		if name != "google" && name != "github" {
			log.Fatalf("gateway oidc: provider must be google or github")
		}
		if pub != "" {
			cfg.PublicURL = pub
		}
		if cfg.PublicURL == "" {
			log.Fatalf("gateway oidc: set the public origin too: --public-url https://picode.example.com")
		}
		if cfg.PlainListen == "" {
			cfg.PlainListen = "127.0.0.1:8480"
		}
		if len(cfg.TrustedProxies) == 0 {
			cfg.TrustedProxies = []string{"127.0.0.1/32"}
		}
		cfg.OIDC[name] = gateway.ProviderConfig{}
		sec.Providers[name] = gateway.ProviderSecret{ClientID: words[2], ClientSecret: words[3]}
		if err := gateway.SaveSecrets(gateway.SecretPath(cfgPath), sec); err != nil {
			log.Fatalf("gateway oidc: %v", err)
		}
		if err := gateway.Save(cfgPath, cfg); err != nil {
			log.Fatalf("gateway oidc: %v", err)
		}
		fmt.Printf("%s configured. Register this callback with the provider:\n  %s/-/auth/callback/%s\n", name, strings.TrimRight(cfg.PublicURL, "/"), name)
		fmt.Printf("Plain listener %s for your TLS proxy; trusted proxies %v. Restart: systemctl restart %s\n", cfg.PlainListen, cfg.TrustedProxies, install.GatewayUnitName)
	case len(words) == 2 && words[0] == "unset":
		delete(cfg.OIDC, words[1])
		delete(sec.Providers, words[1])
		_ = gateway.SaveSecrets(gateway.SecretPath(cfgPath), sec)
		if err := gateway.Save(cfgPath, cfg); err != nil {
			log.Fatalf("gateway oidc: %v", err)
		}
		fmt.Printf("%s removed. Restart: systemctl restart %s\n", words[1], install.GatewayUnitName)
	default:
		log.Fatalf("usage: picode gateway oidc set <google|github> <client id> <client secret> [--public-url https://…]\n       picode gateway oidc unset <provider>")
	}
}
