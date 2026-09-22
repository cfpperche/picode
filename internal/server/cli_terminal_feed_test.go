package server

import (
	"bufio"
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// A terminal created while a page is open arrives through the feed, and the
// store's terminal.created carries the bare row: id, name, cwd, workspace
// and createdAt. The identity a sidebar row, a tab and a face draw — the CLI
// this terminal launches — lives outside that row, so creation has to
// announce the same live view every other terminal response uses. Without it
// a Muse Code or Antigravity terminal (no adapter, so nothing else ever
// enriches the record) renders as a plain shell until a reload.
func TestCLITerminalCreationAnnouncesItsIdentityOnTheFeed(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	ts, home, stream := cliFeedServer(t)
	defer stream.Close()
	// The fixture answers --version and then stays attached, so the launch
	// succeeds without a vendor CLI on PATH.
	binary := filepath.Join(home, "fake-muse")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nif [ \"$1\" = --version ]; then printf 'fixture-muse 1.0\\n'; exit 0; fi\nexec cat\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	created := launchFixture(t, ts, "muse", map[string]any{
		"name":      "Feed fixture",
		"cwd":       home,
		"overrides": map[string]any{"executable": binary},
	}, 201)
	id, _ := created["id"].(string)
	if id == "" || created["launchError"] != nil {
		t.Fatalf("creation: %v", created)
	}
	killFixture(t, id)
	waitFixtureSession(t, id)

	announced := waitForTerminalFrame(t, stream, id)
	if !strings.Contains(announced, `"launchCli":"muse"`) {
		t.Fatalf("feed frame lost the CLI identity: %s", announced)
	}
	if !strings.Contains(announced, `"running":true`) {
		t.Fatalf("feed frame lost the live state: %s", announced)
	}
}

// A launch that cannot resolve yet is refused before any terminal row
// exists: there is nothing to announce and nothing left to clean up. This is
// the other side of the same decision table as the test above — the store
// row is only created once the launch can actually be prepared.
func TestCLITerminalRefusedLaunchCreatesNoRow(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	ts, home, stream := cliFeedServer(t)
	defer stream.Close()
	launchFixture(t, ts, "muse", map[string]any{
		"name":      "Never created",
		"cwd":       home,
		"overrides": map[string]any{"executable": filepath.Join(home, "missing-cli")},
	}, 400)
	launchFixture(t, ts, "muse", map[string]any{
		"name":      "Never created",
		"cwd":       filepath.Join(home, "gone"),
		"overrides": map[string]any{"executable": filepath.Join(home, "missing-cli")},
	}, 400)
	list := cliRequest(t, ts, "GET", "/api/terminals", nil, 200)
	rows, _ := list["terminals"].([]any)
	if len(rows) != 0 {
		t.Fatalf("refused launch left %d terminal(s): %v", len(rows), rows)
	}
}

// cliFeedServer is cleanupServer with a feed, so a test can read what a page
// would receive. It returns the server, the fake home and the open stream.
func cliFeedServer(t *testing.T) (*httptest.Server, string, *sseStream) {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	dataDir := filepath.Join(root, "data")
	for _, dir := range []string{home, dataDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SHELL", "/bin/bash")
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	f := &feed.Feed{Store: st}
	st.OnEvent = f.Publish
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store:    st,
		Tmux:     tmux.New(),
		Runtime:  rpc.NewRuntime("cat", st, nil),
		AgentCmd: "cat",
		DataDir:  dataDir,
		Feed:     f,
	}).Handler)
	t.Cleanup(ts.Close)
	body, closeStream := openStream(t, ts, "")
	readFrames(t, body, 1, 3*time.Second) // hello
	return ts, home, &sseStream{body: body, close: closeStream}
}

type sseStream struct {
	body  *bufio.Reader
	close func()
}

func (s *sseStream) Close() { s.close() }

// waitForTerminalFrame reads until the terminal.changed frame for id shows
// up, so the assertion is about presence rather than frame order.
func waitForTerminalFrame(t *testing.T, stream *sseStream, id string) string {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		for _, fr := range readFrames(t, stream.body, 1, 2*time.Second) {
			if fr.Event != "change" || !strings.Contains(fr.Data, `"type":"terminal.changed"`) {
				continue
			}
			if strings.Contains(fr.Data, `"id":"`+id+`"`) {
				return fr.Data
			}
		}
	}
	t.Fatalf("no terminal.changed frame for %s", id)
	return ""
}

// waitFixtureSession polls until the launched pane exists. Creation answers
// before tmux has the session, so a cleanup that runs first finds nothing and
// the session appears afterwards — in the shared tmux server, where the
// owner's sidebar would list it.
func waitFixtureSession(t *testing.T, id string) {
	t.Helper()
	name := tmux.ShellSessionName(id)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for i := 0; i < 100; i++ {
		if has, err := tmux.New().HasSession(ctx, name); err == nil && has {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("tmux session %s never appeared", name)
}

// killFixture removes the terminal's tmux session when the test ends. The
// context is its own: t.Context() is canceled before cleanup functions run,
// and a canceled context made every tmux call fail — which is how these
// fixtures used to leak into the shared tmux server.
func killFixture(t *testing.T, id string) {
	t.Helper()
	name := tmux.ShellSessionName(id)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		for i := 0; i < 100; i++ {
			_ = tmux.New().KillSession(ctx, name)
			has, err := tmux.New().HasSession(ctx, name)
			if err != nil || !has {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Errorf("tmux session %s survived the cleanup", name)
	})
}
