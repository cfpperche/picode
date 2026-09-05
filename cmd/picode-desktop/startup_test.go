package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/desktop"
)

func startupReady() desktop.TaskStatus {
	return desktop.TaskStatus{Schema: 1, Exists: true, Enabled: true, State: 3,
		UserID: "owner", CurrentUserID: "owner", Interactive: true, Limited: true, Logon: true,
		Executable: `C:\PiCode\picode-desktop.exe`, Arguments: "--tray", ExecutableExists: true,
		ExecutionTimeLimit: "PT0S", MultipleInstances: 2, RestartCount: 3, RestartInterval: "PT1M"}
}

func TestStartupReportDistinguishesConfigurationAndRuntime(t *testing.T) {
	tests := []struct {
		name   string
		change func(*desktop.TaskStatus)
		err    error
		ready  bool
		text   string
	}{
		{"stopped after quit", func(*desktop.TaskStatus) {}, nil, true, "Tray task: stopped"},
		{"running", func(s *desktop.TaskStatus) { s.State = 4; s.LastResult = 0x41301 }, nil, true, "Tray task: running"},
		{"disabled", func(s *desktop.TaskStatus) { s.Enabled = false; s.State = 1 }, nil, false, "choice was preserved"},
		{"missing", func(s *desktop.TaskStatus) { s.Exists = false }, nil, false, "Run picode-desktop install"},
		{"legacy policy", func(s *desktop.TaskStatus) { s.ExecutionTimeLimit = "PT72H" }, nil, false, "startup-repair"},
		{"other owner", func(s *desktop.TaskStatus) { s.UserID = "other" }, nil, false, "Inspect the task registration"},
		{"query failed", func(*desktop.TaskStatus) {}, errors.New("access denied"), false, "cannot inspect task"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := startupReady()
			tt.change(&s)
			var out bytes.Buffer
			if ready := printStartupStatus(&out, s, tt.err); ready != tt.ready || !strings.Contains(out.String(), tt.text) {
				t.Fatalf("ready = %v, output = %q", ready, out.String())
			}
			if tt.err != nil && strings.Contains(out.String(), "not registered") {
				t.Fatal("inspection error was presented as a missing task")
			}
		})
	}
}

type startupRunner struct {
	reports [][]byte
	calls   int
}

func (r *startupRunner) Output(string, ...string) ([]byte, error) {
	r.calls++
	return r.reports[r.calls-1], nil
}
func (r *startupRunner) Run(string, ...string) error {
	panic("startup repair must not run unrelated commands")
}

func TestStartupRepairElevation(t *testing.T) {
	legacy := startupReady()
	legacy.ExecutionTimeLimit = "PT72H"
	before, _ := json.Marshal(legacy)
	for _, tt := range []struct {
		name, failure string
		relaunched    bool
		elevationErr  error
		wantCalls     int
		wantErr       bool
	}{
		{"handoff", `{"schema":1,"error":"Access denied","code":-2147024891}`, true, nil, 1, false},
		{"declined", `{"schema":1,"error":"Access denied","code":-2147024891}`, false, errors.New("declined"), 1, true},
		{"already admin", `{"schema":1,"error":"Access denied","code":-2147024891}`, false, nil, 1, true},
		{"not a permission error", `{"schema":1,"error":"RPC unavailable","code":-2147023174}`, false, nil, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &startupRunner{reports: [][]byte{before, []byte(tt.failure)}}
			var out bytes.Buffer
			calls := 0
			err := repairStartup(r, func() (bool, error) { calls++; return tt.relaunched, tt.elevationErr }, &out)
			if (err != nil) != tt.wantErr || calls != tt.wantCalls {
				t.Fatalf("err = %v, elevation calls = %d", err, calls)
			}
			if strings.Contains(out.String(), "policy repaired") {
				t.Fatal("elevation handoff/error claimed repair success")
			}
		})
	}
}

func TestTrayResultDecisionTable(t *testing.T) {
	setupErr := errors.New("distro not found")
	for _, err := range []error{nil, setupErr} {
		for _, quit := range []bool{false, true} {
			got := trayResult(err, quit)
			if quit && got != nil || !quit && got != err {
				t.Fatalf("setup=%v quit=%v returned %v", err, quit, got)
			}
		}
	}
}

// Regression: exit(nil) previously returned to the unknown-command branch,
// making a normal tray Quit incorrectly report exit status 2.
func TestCommandExitDoesNotFallThrough(t *testing.T) {
	if mode := os.Getenv("PICODE_TEST_COMMAND_EXIT"); mode != "" {
		if mode == "failure" {
			exit(errors.New("expected failure"))
		} else {
			exit(nil)
		}
		fmt.Println("fell through")
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"success", "failure"} {
		cmd := exec.Command(executable, "-test.run=^TestCommandExitDoesNotFallThrough$")
		cmd.Env = append(os.Environ(), "PICODE_TEST_COMMAND_EXIT="+mode)
		out, err := cmd.CombinedOutput()
		if strings.Contains(string(out), "fell through") {
			t.Fatalf("%s exit continued execution: %s", mode, out)
		}
		if mode == "success" && err != nil {
			t.Fatalf("successful exit: %v: %s", err, out)
		}
		if mode == "failure" {
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("failure exit = %v, expected status 1", err)
			}
		}
	}
}
