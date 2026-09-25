package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/desktop"
)

// TestRelocateFlowOrder is the decision table for the part that stops the
// distro:
//
//	kind   | copy fails | → calls                                          | outcome
//
// Every row is wrapped in the keepalive task off → … → on, run: the task
// restarts on failure and would boot the distro mid-copy.
//
//	move   | no         | terminate, move, start                           | done, "Moved to"
//	backup | no         | terminate, create folder, export, start          | done, "Backed up to"
//	move   | yes        | terminate, move, start                           | error names the move; distro started anyway
//	backup | yes        | terminate, create folder, export, start          | error names the backup; distro started anyway
func TestRelocateFlowOrder(t *testing.T) {
	cases := []struct {
		name      string
		backup    bool
		copyErr   error
		wantCalls string
		wantErr   string
	}{
		{"move", false, nil, "task /disable | wsl --terminate Ubuntu | wsl --manage Ubuntu --move E:\\WSL\\Ubuntu | wsl -d Ubuntu -u goat -- true | task /enable | task PiCodeDistro", ""},
		{"backup", true, nil, "task /disable | wsl --terminate Ubuntu | ps New-Item | wsl --export Ubuntu E:\\WSL\\Ubuntu\\u.vhdx --format vhd | wsl -d Ubuntu -u goat -- true | task /enable | task PiCodeDistro", ""},
		{"move fails", false, errors.New("disk full"), "task /disable | wsl --terminate Ubuntu | wsl --manage Ubuntu --move E:\\WSL\\Ubuntu | wsl -d Ubuntu -u goat -- true | task /enable | task PiCodeDistro", "move to"},
		{"backup fails", true, errors.New("disk full"), "task /disable | wsl --terminate Ubuntu | ps New-Item | wsl --export Ubuntu E:\\WSL\\Ubuntu\\u.vhdx --format vhd | wsl -d Ubuntu -u goat -- true | task /enable | task PiCodeDistro", "back up to"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &diskStub{}
			long := &recordingRunner{into: r, err: tc.copyErr}
			var out wslOutcome
			relocateFlow(app{runner: r, distro: "Ubuntu", user: "goat"}, long, tc.backup, `E:\WSL\Ubuntu`, `E:\WSL\Ubuntu\u.vhdx`, func(string) {}, &out)
			var calls []string
			for _, c := range r.calls {
				switch c[0] {
				case desktop.WSLExe:
					calls = append(calls, "wsl "+strings.Join(c[1:], " "))
				case desktop.PowerShellExe:
					calls = append(calls, "ps New-Item")
				case desktop.SchtasksExe:
					calls = append(calls, "task "+c[len(c)-1])
				default:
					t.Fatalf("unexpected program %v", c)
				}
			}
			if got := strings.Join(calls, " | "); got != tc.wantCalls {
				t.Errorf("calls =\n %s\nwant\n %s", got, tc.wantCalls)
			}
			if !out.Stopped {
				t.Error("stopped must be set")
			}
			if tc.wantErr == "" && (!out.Done || out.Note == "" || !out.Copied) {
				t.Errorf("want done: %+v", out)
			}
			if tc.wantErr != "" && (out.Done || !strings.Contains(out.Error, tc.wantErr)) {
				t.Errorf("want error %q: %+v", tc.wantErr, out)
			}
		})
	}
}

// TestRelocateFlowPartialFailures: the rows where the copy did or did not
// happen must never be confused.
//
//	terminate fails → no copy, start still runs, copied=false
//	New-Item fails (backup) → no export, copied=false
//	copy ok, start fails → copied=true with its note, error names the start
func TestRelocateFlowPartialFailures(t *testing.T) {
	run := func(backup bool, errAt int) (wslOutcome, []string) {
		r := &failAt{n: errAt}
		var out wslOutcome
		relocateFlow(app{runner: r, distro: "Ubuntu", user: "goat"}, r, backup, `E:\WSL\B`, `E:\WSL\B\u.vhdx`, func(string) {}, &out)
		return out, r.calls
	}
	// call order (backup): 0 disable, 1 terminate, 2 New-Item, 3 export, 4 start, 5 enable, 6 run
	out, calls := run(true, 1)
	if out.Copied || strings.Contains(strings.Join(calls, "|"), "--export") || !strings.Contains(out.Error, "stop") {
		t.Errorf("terminate fails: %+v %v", out, calls)
	}
	out, calls = run(true, 2)
	if out.Copied || strings.Contains(strings.Join(calls, "|"), "--export") || !strings.Contains(out.Error, "create") {
		t.Errorf("New-Item fails: %+v %v", out, calls)
	}
	// move order: 0 disable, 1 terminate, 2 move, 3 start
	out, _ = run(false, 3)
	if !out.Copied || out.Done || !strings.Contains(out.Error, "start") || !strings.Contains(out.Note, "Moved to") {
		t.Errorf("start fails after the move: %+v", out)
	}
}

// failAt fails its n-th call (0-based) and records every call.
type failAt struct {
	n     int
	i     int
	calls []string
}

func (f *failAt) Output(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	i := f.i
	f.i++
	if i == f.n {
		return nil, errors.New("boom")
	}
	return nil, nil
}

func (f *failAt) Run(name string, args ...string) error {
	_, err := f.Output(name, args...)
	return err
}
