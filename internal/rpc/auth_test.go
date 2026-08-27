package rpc

import (
	"context"
	"testing"
	"time"
)

func TestBeginMCPAuthShort(t *testing.T) {
	AuthTestInstant = true
	t.Cleanup(func() { AuthTestInstant = false })
	r := NewRuntime("pi", nil, nil)
	r.DataDir = t.TempDir()
	t.Cleanup(r.CloseMCPAuth)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := r.BeginMCPAuth(ctx, "", t.TempDir(), "docs", "https://example.test/mcp", "")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("empty id")
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		done, e, _, found := r.MCPAuthStatus(id)
		if found && done {
			if e != nil {
				t.Fatal(e)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("status done=%v found=%v err=%v", done, found, e)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
