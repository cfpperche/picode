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
	// The bounds prove the call returns and which deadline released it, not
	// timer precision: a Windows runner took 153 ms to honour the 80 ms hard
	// deadline (2026-09-09), so each row gets the same generous slack.
	drain := 40 * time.Millisecond
	hard := 80 * time.Millisecond
	slack := 400 * time.Millisecond
	rows := []struct {
		name      string
		hang      bool
		honour    bool
		wantClose bool
		max       time.Duration
	}{
		{name: "returns immediately", max: slack},
		{name: "honours drain timeout", honour: true, max: drain + slack},
		{name: "hangs past hard deadline", hang: true, wantClose: true, max: hard + slack},
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
