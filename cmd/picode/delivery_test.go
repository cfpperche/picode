package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcptool"
	"github.com/cfpperche/picode/internal/server"
	"github.com/cfpperche/picode/internal/store"
)

func TestDeliveryCLIThroughDaemon(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = repo
		b, e := c.CombinedOutput()
		if e != nil {
			t.Fatalf("git: %s: %v", b, e)
		}
		return strings.TrimSpace(string(b))
	}
	git("init", "-b", "main")
	git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "fixture")
	sha := git("rev-parse", "HEAD")
	st, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer st.Close()
	ws, e := st.AddWorkspace("delivery", repo)
	if e != nil {
		t.Fatal(e)
	}
	a, e := st.AddAgent(ws.ID, "agent", "")
	if e != nil {
		t.Fatal(e)
	}
	ts := httptest.NewServer(server.New("127.0.0.1:0", server.Deps{Store: st}).Handler)
	defer ts.Close()
	caller := &mcptool.Caller{Identity: mcptool.Identity{Agent: a.ID}, Daemon: mcptool.NewHTTPDaemon(ts.URL, func() string { return "" })}
	run := func(want int, args ...string) map[string]any {
		t.Helper()
		var out, errOut bytes.Buffer
		code := deliveryMain(context.Background(), args, caller, &out, &errOut)
		if code != want {
			t.Fatalf("code %d: %s %s", code, &out, &errOut)
		}
		var result map[string]any
		if json.Unmarshal(out.Bytes(), &result) != nil {
			t.Fatalf("not JSON: %s", &out)
		}
		return result
	}
	args := []string{"register", "--title", "Fix", "--branch", "main", "--revision", sha, "--target", "main", "--request-id", "create"}
	d := run(0, args...)["delivery"].(map[string]any)
	id := d["id"].(string)
	if run(0, args...)["replayed"] != true {
		t.Fatal("retry was not replayed")
	}
	review := run(0, "request-review", "--id", id, "--expected-version", "1", "--request-id", "review")
	if review["delivery"].(map[string]any)["review"] != "requested" {
		t.Fatal(review)
	}
	run(1, "withdraw-review", "--id", id, "--expected-version", "1", "--request-id", "stale")
	shown := run(0, "show", "--id", id)
	if shown["validation"] != "unknown" || shown["source"].(map[string]any)["status"] != "unchanged" {
		t.Fatal(shown)
	}
	run(1, "request-integration")
	listed := run(0, "list")
	if len(listed["deliveries"].([]any)) != 1 {
		t.Fatal(listed)
	}
	events, e := st.EventsOfType("delivery.changed", 100)
	if e != nil || len(events) != 2 {
		t.Fatalf("events %+v %v", events, e)
	}
}

func TestDeliveryCLIUsageAndIdentity(t *testing.T) {
	for _, row := range []struct {
		args     []string
		code     int
		contains string
	}{
		{[]string{"--help"}, 0, "Usage: picode delivery"},
		{[]string{"show", "--id", "x"}, 1, "no identity"},
		{[]string{"bad"}, 2, "unknown delivery action"},
		{[]string{"show", "--unknown"}, 2, "flag provided but not defined"},
		{[]string{"list", "extra"}, 2, "unexpected positional"},
	} {
		var out, errOut bytes.Buffer
		code := deliveryMain(context.Background(), row.args, &mcptool.Caller{}, &out, &errOut)
		if code != row.code || !strings.Contains(out.String()+errOut.String(), row.contains) {
			t.Fatalf("%v: %d %s %s", row.args, code, &out, &errOut)
		}
	}
}
