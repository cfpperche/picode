package rpc

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestBeginMCPAuthShort(t *testing.T) {
	t.Setenv("PICODE_FAKE_RPC", "1")
	r := NewRuntime(os.Args[0], nil, nil)
	t.Cleanup(r.CloseMCPAuth)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, url, err := r.BeginMCPAuth(ctx, "", t.TempDir(), "docs")
	if err != nil {
		t.Fatal(err)
	}
	if id != "ui-auth" {
		t.Fatalf("id = %q", id)
	}
	if url != "https://example.test/oauth" {
		t.Fatalf("url = %q", url)
	}
	done, _, found := r.MCPAuthStatus(id)
	if !found || done {
		t.Fatalf("status pending got done=%v found=%v", done, found)
	}
	if err := r.ReplyMCPAuth(id, "http://127.0.0.1/cb", false); err != nil {
		t.Fatal(err)
	}
}

func TestBeginMCPAuthAlreadyDone(t *testing.T) {
	t.Setenv("PICODE_FAKE_RPC", "1")
	r := NewRuntime(os.Args[0], nil, nil)
	t.Cleanup(r.CloseMCPAuth)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, url, err := r.BeginMCPAuth(ctx, "", t.TempDir(), "already")
	if err != nil {
		t.Fatal(err)
	}
	if id != "" || url != "" {
		t.Fatalf("id=%q url=%q (want already-signed-in)", id, url)
	}
}
