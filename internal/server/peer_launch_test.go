package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/store"
)

func TestPeerLaunchResumeBoundary(t *testing.T) {
	data := t.TempDir()
	before := clisession.CodexTestRoot
	clisession.CodexTestRoot = filepath.Join(data, "codex-sessions")
	t.Cleanup(func() { clisession.CodexTestRoot = before })
	rollout := filepath.Join(clisession.CodexTestRoot, "rollout-2026-09-24T00-00-00-fixture-session.jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rollout, []byte(`{"type":"session_meta","payload":{"id":"fixture-session","cwd":"`+data+`"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
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
		name string
		args []string
	}{
		{"exact resume", resume}, {"fresh", nil}, {"other conversation", []string{"resume", "another"}}, {"fork", []string{"fork", "fixture-session"}}, {"continue", []string{"resume", "--last"}},
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
			if strings.Contains(string(body), token) {
				t.Fatal("Codex must discover credentials per native tool call")
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

func TestPeerResumeExactDecisionTable(t *testing.T) {
	codex := &store.TerminalLastSession{CLI: "codex", SessionID: "fixture-session", ResumeArgs: []string{"resume", "fixture-session"}}
	claude := &store.TerminalLastSession{CLI: "claude-code", SessionID: "cc-1", ResumeArgs: []string{"--resume", "cc-1"}}
	for _, tc := range []struct {
		name string
		ls   *store.TerminalLastSession
		args []string
		want bool
	}{
		{"codex exact", codex, []string{"resume", "fixture-session"}, true},
		{"codex nil session", nil, []string{"resume", "fixture-session"}, false},
		{"codex empty argv", codex, nil, false},
		{"codex leading sandbox", codex, []string{"--sandbox", "workspace-write", "resume", "fixture-session"}, false},
		{"codex leading config", codex, []string{"-c", "model=x", "resume", "fixture-session"}, false},
		{"codex leading dashdash", codex, []string{"--", "resume", "fixture-session"}, false},
		{"codex resume dashdash id", codex, []string{"resume", "--", "fixture-session"}, false},
		{"codex trailing dashdash", codex, []string{"resume", "fixture-session", "--"}, false},
		{"codex extra operand", codex, []string{"resume", "fixture-session", "--dangerously-bypass-approvals-and-sandbox"}, false},
		{"codex continue last", codex, []string{"resume", "--last"}, false},
		{"codex other id", codex, []string{"resume", "another"}, false},
		{"codex fork", codex, []string{"fork", "fixture-session"}, false},
		{"claude exact", claude, []string{"--resume", "cc-1"}, true},
		{"claude leading global", claude, []string{"--dangerously-skip-permissions", "--resume", "cc-1"}, false},
		{"claude dashdash", claude, []string{"--resume", "--", "cc-1"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := peerResumeExact(tc.args, tc.ls); got != tc.want {
				t.Fatalf("peerResumeExact(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

func TestPeerLaunchCustomResumeDoesNotAttach(t *testing.T) {
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
	binary := filepath.Join(data, "claude")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := st.SetCLIConfig("claude-code", clilaunch.Config{Executable: binary}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(term.ID, "claude-code", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	resume := []string{"--resume", "fixture-session"}
	if err := st.SetTerminalLastSession(term.ID, store.TerminalLastSession{CLI: "claude-code", SessionID: "fixture-session", ResumeArgs: resume}); err != nil {
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
		attach bool
	}{
		{"exact resume", resume, true},
		{"leading skip-permissions", []string{"--dangerously-skip-permissions", "--resume", "fixture-session"}, false},
		{"resume dashdash id", []string{"--resume", "--", "fixture-session"}, false},
		{"fresh", nil, false},
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
			script := string(body)
			got := strings.Contains(script, "--mcp-config") && strings.Contains(script, p.ID)
			if got != tc.attach {
				t.Fatalf("credential attached=%v, want %v\n%s", got, tc.attach, script)
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

func TestPeerOpenCodeInlineConfiguration(t *testing.T) {
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
	binary := filepath.Join(data, "opencode")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(term.ID, "opencode", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	args := []string{"--session", "fixture-session"}
	if err := st.SetTerminalLastSession(term.ID, store.TerminalLastSession{CLI: "opencode", SessionID: "fixture-session", ResumeArgs: args}); err != nil {
		t.Fatal(err)
	}
	p, token, err := st.EnablePeer("terminal", term.ID, "fixture-session")
	if err != nil {
		t.Fatal(err)
	}
	if err := communication.SaveLaunch(data, communication.LaunchConfig{Connection: p, Token: token, URL: "http://localhost" + communication.Path}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENCODE_CONFIG_CONTENT", `{"model":"inherited-model"}`)
	for _, tc := range []struct {
		name string
		env  map[string]string
		want string
		fail bool
	}{
		{"inherited", nil, "inherited-model", false},
		{"resolved override", map[string]string{"OPENCODE_CONFIG_CONTENT": `{"model":"override-model"}`}, "override-model", false},
		{"explicit empty", map[string]string{"OPENCODE_CONFIG_CONTENT": ""}, "", false},
		{"invalid override", map[string]string{"OPENCODE_CONFIG_CONTENT": "{"}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := st.SetCLIConfig("opencode", clilaunch.Config{Executable: binary, Env: tc.env}); err != nil {
				t.Fatal(err)
			}
			launch, err := st.TerminalLaunch(term.ID)
			if err != nil {
				t.Fatal(err)
			}
			launch.Overrides.Args = &args
			prepared, err := prepareCLITerminal(Deps{Store: st, DataDir: data}, data, launch)
			if tc.fail {
				if err == nil {
					prepared.discard()
					t.Fatal("invalid inline config accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer prepared.discard()
			raw, err := os.ReadFile(prepared.script)
			if err != nil {
				t.Fatal(err)
			}
			body := string(raw)
			if !strings.Contains(body, token) || (tc.want != "" && !strings.Contains(body, tc.want)) {
				t.Fatal("settings missing")
			}
			if tc.name != "inherited" && strings.Contains(body, "inherited-model") {
				t.Fatal("inherited content overrode explicit config")
			}
		})
	}
	if err := os.WriteFile(filepath.Join(data, "communication", p.ID, "connection.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	launch, err := st.TerminalLaunch(term.ID)
	if err != nil {
		t.Fatal(err)
	}
	launch.Overrides.Args = &args
	if prepared, err := prepareCLITerminal(Deps{Store: st, DataDir: data}, data, launch); err == nil {
		prepared.discard()
		t.Fatal("corrupt setup did not stop launch")
	}
}
