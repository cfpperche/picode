package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/store"
)

func TestPeerLaunchResumeBoundary(t *testing.T) {
	data := t.TempDir()
	st, err := store.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	w, err := st.AddWorkspace("Fixture", data)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(w.ID, "Fixture", data)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(data, "codex")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := st.SetCLIConfig("codex", clilaunch.Config{Executable: binary}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(term.ID, "codex", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	resume := []string{"resume", "fixture-session"}
	if err := st.SetTerminalLastSession(term.ID, store.TerminalLastSession{CLI: "codex", SessionID: "fixture-session", ResumeArgs: resume}); err != nil {
		t.Fatal(err)
	}
	p, token, err := st.EnablePeer("terminal", term.ID, "fixture-session")
	if err != nil {
		t.Fatal(err)
	}
	if err := communication.SaveLaunch(data, communication.LaunchConfig{Connection: p, Token: token, URL: "http://localhost" + communication.Path}); err != nil {
		t.Fatal(err)
	}
	deps := Deps{Store: st, DataDir: data}
	for _, tc := range []struct {
		name   string
		args   []string
		inject bool
	}{
		{"exact resume", resume, true}, {"fresh", nil, false}, {"other conversation", []string{"resume", "another"}, false}, {"fork", []string{"fork", "fixture-session"}, false}, {"continue", []string{"resume", "--last"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			launch, err := st.TerminalLaunch(term.ID)
			if err != nil {
				t.Fatal(err)
			}
			launch.Overrides.Args = &tc.args
			prepared, err := prepareCLITerminal(deps, data, launch)
			if err != nil {
				t.Fatal(err)
			}
			defer prepared.discard()
			body, err := os.ReadFile(prepared.script)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(body), token) != tc.inject {
				t.Fatal("incorrect credential injection")
			}
			if strings.Contains(string(body), "export PICODE_PEER_TOKEN") {
				t.Fatal("credential inherited by return shell")
			}
			snapshot, _ := json.Marshal(prepared.snapshot)
			if strings.Contains(string(snapshot), token) {
				t.Fatal("credential in diagnostic snapshot")
			}
			for _, env := range prepared.environment {
				if strings.Contains(env, token) {
					t.Fatal("credential in tmux environment")
				}
			}
		})
	}
}

func TestPiTerminalResumeUsesRecordedPath(t *testing.T) {
	ls := &store.TerminalLastSession{CLI: "pi", SessionID: "native", Path: "/private/conversation.jsonl"}
	got := terminalResumeArgs(ls)
	if len(got) != 2 || got[0] != "--session" || got[1] != ls.Path {
		t.Fatalf("resume = %v", got)
	}
}
