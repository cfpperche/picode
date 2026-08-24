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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cfpperche/picode/internal/config"
	"github.com/cfpperche/picode/internal/proclock"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/screenshot"
	"github.com/cfpperche/picode/internal/server"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tlsutil"
	"github.com/cfpperche/picode/internal/tmux"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "screenshot":
			runScreenshot(os.Args[2:])
			return
		case "help", "-h", "--help":
			usage()
			return
		}
	}
	serve()
}

func usage() {
	fmt.Println(`picode — a browser ADE for Pi agents (server mode is the default)

Usage:
  picode [flags]              start the server
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
	defer unlock()

	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	runtime := rpc.NewRuntime("pi", st, nil)
	defer runtime.StopAll()

	rebindCh := make(chan struct{}, 1)
	state := &serveState{}

	deps := server.Deps{
		Store:    st,
		Tmux:     tmux.New(),
		Runtime:  runtime,
		AgentCmd: "pi", // ADR-0003: user-installed pi
		DataDir:  dataDir,
		Rebind: func() {
			select {
			case rebindCh <- struct{}{}:
			default:
			}
		},
		PortSnapshot: state.snapshot,
	}

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
