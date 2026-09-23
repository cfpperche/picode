package main

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/desktop"
)

// TestRestartFlowOrder is the decision table for the part that ends
// sessions:
//
//	update | update fails | shutdown fails | → calls                 | stopped | outcome
//	no     | —            | no             | shutdown, start         | yes     | done
//	yes    | no           | no             | update, shutdown, start | yes     | done
//	yes    | yes          | —              | update, start (no-op)   | no      | error names the update; nothing shut down
//	no     | —            | yes            | shutdown, start         | yes     | error names the shutdown; distro started anyway
func TestRestartFlowOrder(t *testing.T) {
	cases := []struct {
		name        string
		update      bool
		updErr      error
		errs        []error
		wantCalls   string
		wantStopped bool
		wantErrSub  string
	}{
		{"restart", false, nil, nil, "--shutdown | -d Ubuntu -u goat -- true", true, ""},
		{"update", true, nil, nil, "--update | --shutdown | -d Ubuntu -u goat -- true", true, ""},
		{"update declined", true, errors.New("exit status 1223"), nil, "--update | -d Ubuntu -u goat -- true", false, "wsl --update"},
		{"shutdown fails", false, nil, []error{errors.New("boom")}, "--shutdown | -d Ubuntu -u goat -- true", true, "wsl --shutdown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &diskStub{errs: tc.errs}
			var upd desktop.Runner
			if tc.update {
				upd = &recordingRunner{into: r, err: tc.updErr}
			}
			var out wslOutcome
			restartFlow(app{runner: r, distro: "Ubuntu", user: "goat"}, upd, func(string) {}, &out)
			var calls []string
			for _, c := range r.calls {
				if c[0] != desktop.WSLExe {
					t.Fatalf("unexpected program %v", c)
				}
				calls = append(calls, strings.Join(c[1:], " "))
			}
			if got := strings.Join(calls, " | "); got != tc.wantCalls {
				t.Errorf("calls = %q, want %q", got, tc.wantCalls)
			}
			if out.Stopped != tc.wantStopped {
				t.Errorf("stopped = %v, want %v", out.Stopped, tc.wantStopped)
			}
			if tc.wantErrSub == "" && (!out.Done || out.Error != "") {
				t.Errorf("want done, got %+v", out)
			}
			if tc.wantErrSub != "" && (out.Done || !strings.Contains(out.Error, tc.wantErrSub)) {
				t.Errorf("want error with %q, got %+v", tc.wantErrSub, out)
			}
		})
	}
}

// recordingRunner logs its calls into the shared stub (so the order across
// the two runners is one list) and answers with its own error.
type recordingRunner struct {
	into *diskStub
	err  error
}

func (r *recordingRunner) Output(name string, args ...string) ([]byte, error) {
	r.into.calls = append(r.into.calls, append([]string{name}, args...))
	return nil, r.err
}

func (r *recordingRunner) Run(name string, args ...string) error {
	_, err := r.Output(name, args...)
	return err
}

func TestCollectHostKeepsEachHalf(t *testing.T) {
	r := &diskStub{
		replies: [][]byte{
			[]byte("MemTotal: 32729120 kB\nMemAvailable: 18654340 kB\nSwapTotal: 8388608 kB\nSwapFree: 2220540 kB\n"),
			[]byte(`{"total":68438863872,"freeKiB":26151704,"vm":21808197632}`),
			wide("WSL version: 2.7.0.0\r\nKernel version: 6.6.114.1-1\r\n"),
		},
	}
	rep := collectHost(app{runner: r, distro: "Ubuntu", user: "goat"}, func() (string, error) { return "2.7.1", nil })
	if rep.Windows == nil || rep.VM == nil || rep.Versions == nil {
		t.Fatalf("halves missing: %+v", rep)
	}
	if !rep.Newer || rep.Latest != "2.7.1" {
		t.Errorf("2.7.1 over 2.7.0.0 must be newer: %+v", rep)
	}
	rep = collectHost(app{runner: &diskStub{}, distro: "Ubuntu"}, func() (string, error) { return "", errors.New("offline") })
	if rep.WindowsError == "" || rep.VMError == "" || rep.VersionError == "" || rep.LatestError != "offline" || rep.Newer {
		t.Errorf("every half must name its failure: %+v", rep)
	}
}

// TestTimedRunnerBounds: a call that outlives its deadline ends in an error
// naming the bound — the Management window never waits on it forever.
func TestTimedRunnerBounds(t *testing.T) {
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("no sleep binary")
	}
	start := time.Now()
	err = timedRunner{d: 300 * time.Millisecond}.Run(sleep, "10")
	if err == nil || !strings.Contains(err.Error(), "did not finish within") {
		t.Fatalf("err = %v", err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("took %s", d)
	}
}
