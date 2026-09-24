package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/store"
)

// procTree builds a synthetic snapshot: each proc is pid → parent, session,
// argv, age in seconds and whether stdin is a terminal. Pane session is 100.
type fakeProc struct {
	pid, ppid, sid int
	argv           []string
	ageSecs        uint64
	tty            bool
}

func fakeSnapshot(procs ...fakeProc) *procSnapshot {
	const uptime = 100000
	s := &procSnapshot{
		argv: map[int][]string{}, ppid: map[int]int{}, sid: map[int]int{},
		start: map[int]uint64{}, uptime: uptime, ttyIn: map[int]bool{},
	}
	for _, p := range procs {
		s.argv[p.pid] = p.argv
		s.ppid[p.pid] = p.ppid
		s.sid[p.pid] = p.sid
		s.start[p.pid] = uptime - p.ageSecs*clockTicks
		s.ttyIn[p.pid] = p.tty
	}
	return s
}

// The decision table ADR-0212 records: conditions → observed command.
func TestRunningCommandDecisionTable(t *testing.T) {
	wrapper := fakeProc{pid: 100, ppid: 1, sid: 100, argv: []string{"/bin/sh", "/data/bin/codex"}, ageSecs: 900, tty: true}
	cli := fakeProc{pid: 101, ppid: 100, sid: 100, argv: []string{"/opt/codex/bin/codex"}, ageSecs: 900, tty: true}
	mcp := fakeProc{pid: 110, ppid: 101, sid: 100, argv: []string{"picode", "mcp"}, ageSecs: 800}
	cases := []struct {
		name  string
		procs []fakeProc
		want  string
	}{
		{"idle CLI with helpers only", []fakeProc{wrapper, cli, mcp,
			{pid: 111, ppid: 101, sid: 100, argv: []string{"codex-code-mode"}, ageSecs: 700}}, ""},
		{"detached command (Codex !make)", []fakeProc{wrapper, cli, mcp,
			{pid: 120, ppid: 101, sid: 120, argv: []string{"make", "deploy"}, ageSecs: 30}}, "make"},
		{"detached bash -c names what it runs (Claude)", []fakeProc{wrapper, cli,
			{pid: 120, ppid: 101, sid: 120, argv: []string{"/bin/bash", "-c"}, ageSecs: 30},
			{pid: 121, ppid: 120, sid: 120, argv: []string{"sleep", "20"}, ageSecs: 30}}, "sleep"},
		{"pane-session sh -c on the terminal (Hermes)", []fakeProc{wrapper, cli,
			{pid: 120, ppid: 101, sid: 100, argv: []string{"/bin/sh", "-c"}, ageSecs: 30, tty: true},
			{pid: 121, ppid: 120, sid: 100, argv: []string{"go", "test"}, ageSecs: 30, tty: true}}, "go"},
		{"younger than the minimum age (hook reporter)", []fakeProc{wrapper, cli,
			{pid: 120, ppid: 101, sid: 120, argv: []string{"/bin/sh", "-c"}, ageSecs: 1}}, ""},
		{"unknown CLI name on the terminal is not a command", []fakeProc{wrapper,
			{pid: 101, ppid: 100, sid: 100, argv: []string{"/opt/codex/bin/codex-next"}, ageSecs: 900, tty: true}}, ""},
		{"helper's own detached child is not the CLI's command", []fakeProc{wrapper, cli,
			{pid: 110, ppid: 101, sid: 100, argv: []string{"gopls"}, ageSecs: 800},
			{pid: 111, ppid: 110, sid: 111, argv: []string{"gopls", "daemon"}, ageSecs: 800}}, ""},
		{"oldest of two commands wins", []fakeProc{wrapper, cli,
			{pid: 120, ppid: 101, sid: 120, argv: []string{"npm", "run"}, ageSecs: 600},
			{pid: 130, ppid: 101, sid: 130, argv: []string{"make"}, ageSecs: 10}}, "npm"},
		{"inhibitor launcher names its command (Grok)", []fakeProc{wrapper, cli,
			{pid: 120, ppid: 101, sid: 120, argv: []string{"systemd-inhibit", "--what=idle"}, ageSecs: 30},
			{pid: 121, ppid: 120, sid: 120, argv: []string{"sleep", "30"}, ageSecs: 30}}, "sleep"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := runningCommand(100, fakeSnapshot(tc.procs...))
			if tc.want == "" {
				if ok {
					t.Fatalf("got command %+v, want none", got)
				}
				return
			}
			if !ok || got.Name != tc.want {
				t.Fatalf("got %+v ok=%v, want %q", got, ok, tc.want)
			}
			if got.Since.IsZero() || time.Since(got.Since) < commandMinAge {
				t.Fatalf("since=%v is not the process start", got.Since)
			}
		})
	}
}

func TestObserveTermCommandAnnouncesOnlyChanges(t *testing.T) {
	f := &feed.Feed{}
	var got []map[string]any
	f.Listen(func(ev store.Event) {
		if ev.Type != "terminal.command" {
			t.Errorf("event %q", ev.Type)
		}
		var d map[string]any
		_ = json.Unmarshal(ev.Data, &d)
		got = append(got, d)
	})
	deps := Deps{TermRuntimes: NewTermRuntimes(), Feed: f}
	cmd := TermCommand{Name: "make", PID: 7, Start: 1, Since: time.Now()}

	observeTermCommand(deps, "t1", cmd, true)
	observeTermCommand(deps, "t1", cmd, true) // same process: no event
	observeTermCommand(deps, "t1", TermCommand{}, false)
	observeTermCommand(deps, "t1", TermCommand{}, false) // already clear: no event

	if len(got) != 2 || got[0]["command"] == nil || got[1]["command"] != nil {
		t.Fatalf("events %+v, want start then clear", got)
	}
	if _, ok := deps.TermRuntimes.Command("t1"); ok {
		t.Fatal("command survived its clear")
	}
}

func TestEndingRuntimeClearsCommand(t *testing.T) {
	deps := Deps{TermStates: NewTermStates(), TermRuntimes: NewTermRuntimes()}
	registerTermRuntime(deps, "t1", TermRuntime{CLI: "codex", Source: "wrapper", RunID: "run-1", PID: 9})
	observeTermCommand(deps, "t1", TermCommand{Name: "make", PID: 10, Start: 1}, true)
	finishTermRuntime(deps, "t1", "run-1")
	if _, ok := deps.TermRuntimes.Command("t1"); ok {
		t.Fatal("a command outlived the CLI run it belonged to")
	}
}
