package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type fakeHTTP struct {
	shutdown func(context.Context) error
	closed   atomic.Bool
}

func (f *fakeHTTP) Shutdown(ctx context.Context) error {
	if f.shutdown != nil {
		return f.shutdown(ctx)
	}
	return nil
}

func (f *fakeHTTP) Close() error {
	f.closed.Store(true)
	return nil
}

func TestDrainHTTPDecisionTable(t *testing.T) {
	// Shutdown that returns, Shutdown that honours drain, Shutdown that
	// hangs past the hard deadline (the systemd stop-sigterm case).
	rows := []struct {
		name      string
		hang      bool
		honour    bool
		wantClose bool
		max       time.Duration
	}{
		{name: "returns immediately", max: 80 * time.Millisecond},
		{name: "honours drain timeout", honour: true, max: 150 * time.Millisecond},
		{name: "hangs past hard deadline", hang: true, wantClose: true, max: 150 * time.Millisecond},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			f := &fakeHTTP{}
			switch {
			case row.hang:
				block := make(chan struct{})
				t.Cleanup(func() { close(block) })
				f.shutdown = func(context.Context) error {
					<-block
					return nil
				}
			case row.honour:
				f.shutdown = func(ctx context.Context) error {
					<-ctx.Done()
					return ctx.Err()
				}
			}
			drain := 40 * time.Millisecond
			hard := 80 * time.Millisecond
			start := time.Now()
			drainHTTP(f, drain, hard)
			elapsed := time.Since(start)
			if elapsed > row.max {
				t.Fatalf("took %s, want ≤ %s", elapsed, row.max)
			}
			if f.closed.Load() != row.wantClose {
				t.Fatalf("Close=%v, want %v", f.closed.Load(), row.wantClose)
			}
		})
	}
}
