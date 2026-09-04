package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

// Drain is how long HTTP Shutdown may wait for in-flight requests.
// Hard is the wall clock after which we stop waiting even if Shutdown
// is stuck (Go waits on listenerGroup without honouring the context).
// Both stay well below systemd TimeoutStopSec=30.
const (
	httpDrain = 5 * time.Second
	httpHard  = 8 * time.Second
)

type httpStopper interface {
	Shutdown(context.Context) error
	Close() error
}

func gracefulShutdown(srv *http.Server) {
	drainHTTP(srv, httpDrain, httpHard)
}

func drainHTTP(srv httpStopper, drain, hard time.Duration) {
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), drain)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := srv.Shutdown(ctx); err != nil && err != context.DeadlineExceeded && err != http.ErrServerClosed {
			log.Printf("server: shutdown: %v", err)
		}
	}()
	timer := time.NewTimer(hard)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		log.Printf("server: shutdown blocked after %s — closing listeners", hard)
		_ = srv.Close()
	}
}
