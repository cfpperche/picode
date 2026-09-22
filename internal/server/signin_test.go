package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// ADR-0184: a sign-in terminal is internal and never outlives its purpose.
// One case per row of the plan's close table (the live-session rows need
// tmux and are covered by TestCredentialSignin*).
func TestSigninTerminalLifecycle(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	deps := Deps{Store: st}
	ctx := context.Background()
	signin := func(cli string) store.Terminal {
		t.Helper()
		tm, err := st.CreateSigninTerminal(cli+" sign-in", t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if err := st.SetTerminalLaunch(tm.ID, cli, clilaunch.Overrides{}); err != nil {
			t.Fatal(err)
		}
		return tm
	}
	gone := func(tm store.Terminal) bool {
		_, err := st.GetTerminal(tm.ID)
		return err != nil
	}
	shell, err := st.CreateTerminal("zsh", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// Off every list a person reads.
	a := signin("claude-code")
	list, err := computeTerminals(ctx, deps)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0]["id"] != shell.ID {
		t.Fatalf("list = %v, want only the shell", list)
	}
	if got := signinTerminals(deps, "claude-code"); len(got) != 1 || got[0].ID != a.ID || a.Kind != store.TerminalKindSignin {
		t.Fatalf("sign-in terminals = %v", got)
	}

	// Credential landed: that CLI's sign-in closes, another CLI's stays.
	b := signin("codex")
	closeSigninFor(ctx, deps, "claude-code")
	if !gone(a) || gone(b) {
		t.Fatalf("credential close: a gone=%v b gone=%v", gone(a), gone(b))
	}

	// Open past the limit: closed on the next tick; younger stays.
	now := time.Now()
	if n := reapSigninTerminals(ctx, deps, now, false); n != 0 || gone(b) {
		t.Fatalf("young sign-in reaped (%d)", n)
	}
	if n := reapSigninTerminals(ctx, deps, now.Add(signinOpenLimit+time.Minute), false); n != 1 || !gone(b) {
		t.Fatalf("old sign-in kept (%d)", n)
	}

	// Boot: every one closes. A shell never does.
	c := signin("grok")
	if n := reapSigninTerminals(ctx, deps, now, true); n != 1 || !gone(c) {
		t.Fatalf("boot reap = %d", n)
	}
	if gone(shell) {
		t.Fatal("a shell was reaped")
	}

	// Its process ended (no session): closed on the next tick.
	if tm := tmux.New(); tm.Available() {
		deps.Tmux = tm
		d := signin("hermes")
		if n := reapSigninTerminals(ctx, deps, now, false); n != 1 || !gone(d) {
			t.Fatalf("ended sign-in kept (%d)", n)
		}
	}
}

// A card that was left finds its sign-in again; none answers 204.
func TestCredentialSigninOpenNamesTheLiveOne(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New()}).Handler)
	t.Cleanup(ts.Close)
	res, err := http.Get(ts.URL + "/api/credentials/signin?cli=claude-code")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("no sign-in = %d", res.StatusCode)
	}
	res, err = http.Get(ts.URL + "/api/credentials/signin?cli=nope")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown cli = %d", res.StatusCode)
	}
}
