package server

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// signinOpenLimit is how long a sign-in terminal may sit idle (ADR-0184):
// a vendor login takes a minute or two; one left behind is closed, never
// kept running where nobody sees it.
const signinOpenLimit = 15 * time.Minute

// createSigninTerminal opens the CLI's own sign-in in a terminal the server
// owns (ADR-0184): kind signin, free, no agent and no grant, off the
// sidebar. It is shown only on the credential card that opened it. On a
// launch failure the row is kept with launchError, so the card can say so.
func createSigninTerminal(deps Deps, r *http.Request, cli clilaunch.CLI, name string, args []string) (store.Terminal, map[string]any, int, error) {
	ov := clilaunch.Overrides{Args: &args}
	if _, status, err := checkCLILaunch(deps, cli, ov); err != nil {
		return store.Terminal{}, nil, status, err
	}
	unlockWorkspace := terminalLock(deps, "workspace:"+store.FreeWorkspaceID)
	defer unlockWorkspace()
	t, err := deps.Store.CreateSigninTerminal(name, "")
	if err != nil {
		return store.Terminal{}, nil, http.StatusBadRequest, err
	}
	unlock := terminalLock(deps, t.ID)
	defer unlock()
	if err := deps.Store.SetTerminalLaunch(t.ID, cli.ID, ov); err != nil {
		_ = deps.Store.DeleteTerminal(t.ID)
		return store.Terminal{}, nil, http.StatusInternalServerError, err
	}
	session := tmux.ShellSessionName(t.ID)
	created, err := ensureShell(deps, r, session, t.ID, t.Cwd)
	if err != nil {
		return t, map[string]any{"id": t.ID, "kind": t.Kind, "launchError": err.Error()}, http.StatusCreated, nil
	}
	return t, termViewForCreation(deps, r, t, session, created), http.StatusCreated, nil
}

// signinTerminals lists the sign-in terminals, optionally for one CLI.
func signinTerminals(deps Deps, cli string) []store.Terminal {
	rows, err := deps.Store.ListTerminals()
	if err != nil {
		return nil
	}
	out := []store.Terminal{}
	for _, t := range rows {
		if t.Kind != store.TerminalKindSignin {
			continue
		}
		if cli != "" {
			if l, e := deps.Store.TerminalLaunch(t.ID); e != nil || l == nil || l.CLI != cli {
				continue
			}
		}
		out = append(out, t)
	}
	return out
}

// signinGrace is how long a sign-in terminal may have no session: the row
// exists a moment before its tmux session does, and a reaper tick or a
// second click in that moment must not take it for an ended one.
var signinGrace = 30 * time.Second

// signinStamps holds, per sign-in terminal, the stamp of the account the
// CLI's store held when the sign-in started, so a card that comes back to
// it can still tell a new login from the old one. In memory: boot closes
// every sign-in anyway.
var signinStamps sync.Map

// signinStamp is the stamp recorded when the sign-in started, or "".
func signinStamp(termID string) string {
	v, _ := signinStamps.Load(termID)
	stamp, _ := v.(string)
	return stamp
}

// closeSigninTerminal ends one sign-in terminal: its exact tmux session,
// its runtime state, its launch files and its row. The store's
// terminal.deleted event tells the card.
func closeSigninTerminal(ctx context.Context, deps Deps, t store.Terminal) error {
	unlock := terminalLock(deps, t.ID)
	defer unlock()
	return closeSigninLocked(ctx, deps, t)
}

func closeSigninLocked(ctx context.Context, deps Deps, t store.Terminal) error {
	if deps.Tmux != nil && deps.Tmux.Available() {
		if err := deps.Tmux.KillSession(ctx, tmux.ShellSessionName(t.ID)); err != nil {
			return err
		}
	}
	if deps.TermStates != nil {
		deps.TermStates.Drop(t.ID)
	}
	if deps.TermRuntimes != nil {
		deps.TermRuntimes.Drop(t.ID)
	}
	if deps.DataDir != "" {
		_ = cleanCLILaunches(deps.DataDir, t.ID, "")
	}
	signinStamps.Delete(t.ID)
	if err := deps.Store.DeleteTerminal(t.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	invalidateTerminals(deps)
	return nil
}

// closeSigninFor closes a CLI's sign-in terminals once its credential has
// landed in the vault — the login is done.
func closeSigninFor(ctx context.Context, deps Deps, cli string) {
	for _, t := range signinTerminals(deps, cli) {
		_ = closeSigninTerminal(ctx, deps, t)
	}
}

// signinEnded reports whether a sign-in is over: its session is gone (the
// CLI exited — the launch script ends with it) and it is past the grace, or
// nobody has typed or seen output in it for signinOpenLimit.
func signinEnded(ctx context.Context, deps Deps, t store.Terminal, now time.Time) bool {
	created, err := time.Parse(time.RFC3339, t.CreatedAt)
	if err != nil {
		created = now
	}
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return now.Sub(created) > signinOpenLimit
	}
	name := tmux.ShellSessionName(t.ID)
	alive, err := deps.Tmux.HasSession(ctx, name)
	if err != nil {
		return false
	}
	if !alive {
		return now.Sub(created) > signinGrace
	}
	last, err := deps.Tmux.SessionActivity(ctx, name)
	if err != nil || last.Before(created) {
		last = created
	}
	return now.Sub(last) > signinOpenLimit
}

// reapSigninTerminals closes every sign-in that has ended (signinEnded);
// all closes every one (boot: nothing from a previous run keeps running
// unseen). The verdict is taken under the terminal's lock, so a sign-in
// being created or reused in the same moment is judged on its real state.
// Returns how many it closed.
func reapSigninTerminals(ctx context.Context, deps Deps, now time.Time, all bool) int {
	n := 0
	for _, t := range signinTerminals(deps, "") {
		func() {
			unlock := terminalLock(deps, t.ID)
			defer unlock()
			cur, err := deps.Store.GetTerminal(t.ID)
			if err != nil {
				return
			}
			if (all || signinEnded(ctx, deps, cur, now)) && closeSigninLocked(ctx, deps, cur) == nil {
				n++
			}
		}()
	}
	return n
}

// StartSigninReaper closes leftover sign-in terminals at boot, then checks
// every tick (ADR-0184).
func StartSigninReaper(ctx context.Context, deps Deps, every time.Duration) {
	if deps.Store == nil {
		return
	}
	reapSigninTerminals(ctx, deps, time.Now(), true)
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			reapSigninTerminals(ctx, deps, now, false)
		}
	}
}
