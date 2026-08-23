// Package main is the picode entrypoint: a single binary that serves the
// PiCode web UI and manages Pi agent processes (see docs/architecture.md).
package main

import (
	"flag"
	"log"
	"net"

	"github.com/cfpperche/picode/internal/server"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:7331", "listen address (localhost-only by default; see docs/architecture.md security model)")
	flag.Parse()

	// Security contract (ADR-bound): non-localhost binds are allowed only
	// as an explicit opt-in for now; token auth lands with the workspace
	// registry (M1). See docs/handoff.md "Known debts".
	if host, _, err := net.SplitHostPort(*addr); err == nil && host != "127.0.0.1" && host != "localhost" && host != "::1" {
		log.Printf("WARNING: listening on %q exposes PiCode beyond localhost. ", host)
		log.Printf("WARNING: agent processes run with your user permissions; do this only on trusted networks.")
	}

	srv := server.New(*addr)
	log.Printf("PiCode listening on http://%s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
