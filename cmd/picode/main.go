// Package main is the picode entrypoint: a single binary that serves the
// PiCode web UI and manages Pi agent processes (see docs/architecture.md).
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
	"os"

	"github.com/cfpperche/picode/internal/screenshot"
	"github.com/cfpperche/picode/internal/server"
	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/workspace"
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
  picode [flags]              start the server (see flags below)
  picode screenshot [flags]   capture a page to PNG (visual-review loop)
    --url string    page to capture (required)
    --out string    destination PNG (required)
    --width int     viewport width (default 1440)
    --height int    viewport height (default 900)
    --full          capture the full page height
    --wait-ms int   settle time after page ready (default 500)

Requires Chrome/Chromium installed (headless).`)
}

func serve() {
	addr := flag.String("addr", "127.0.0.1:7331", "listen address (localhost-only by default; see docs/architecture.md security model)")
	flag.Parse()

	// Security contract (ADR-bound): non-localhost binds are allowed only
	// as an explicit opt-in for now; token auth lands with the workspace
	// registry (M1). See docs/handoff.md "Known debts".
	if host, _, err := net.SplitHostPort(*addr); err == nil && host != "127.0.0.1" && host != "localhost" && host != "::1" {
		log.Printf("WARNING: listening on %q exposes PiCode beyond localhost. ", host)
		log.Printf("WARNING: agent processes run with your user permissions; do this only on trusted networks.")
	}

	regPath, err := workspace.DefaultPath()
	if err != nil {
		log.Fatalf("workspace registry: %v", err)
	}
	reg, err := workspace.Open(regPath)
	if err != nil {
		log.Fatalf("workspace registry: %v", err)
	}

	srv := server.New(*addr, server.Deps{
		Registry: reg,
		Tmux:     tmux.New(),
		AgentCmd: "pi", // ADR-0003: user-installed pi
	})
	log.Printf("PiCode listening on http://%s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
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
