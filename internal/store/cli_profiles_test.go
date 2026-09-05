package store

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

func TestCLIProfilesCopyPersistAndRollback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	p := CLIProfile{ID: "review", CLI: "pi", Name: "Review", Config: clilaunch.Config{Args: []string{}, Env: map[string]string{"SECRET": "private-profile"}}}
	if err := s.SetCLIProfile(p); err != nil {
		t.Fatal(err)
	}
	base := clilaunch.Config{Args: []string{"--base"}, Env: map[string]string{"DROP": "base"}, Integration: true}
	tm, _ := s.CreateTerminalIn("", "Copied", dir)
	if err := s.SetTerminalLaunch(tm.ID, p.CLI, clilaunch.CopyOverrides(p.Config, base)); err != nil {
		t.Fatal(err)
	}
	p.Name = "Renamed"
	p.Config.Args = []string{"--later"}
	if err := s.SetCLIProfile(p); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteCLIProfile(p.ID); err != nil {
		t.Fatal(err)
	}
	v, _ := s.TerminalLaunch(tm.ID)
	c := clilaunch.Resolve(base, v.Overrides)
	if len(c.Args) != 0 || c.Integration || c.Env["DROP"] != "" || c.Env["SECRET"] != "private-profile" {
		t.Fatal("profile removal changed copied settings", c)
	}
	if err := s.SetCLICheck("pi", clilaunch.Diagnostic{Version: "1.2", CheckedAt: "now"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalLaunchAttempt(tm.ID, clilaunch.Attempt{At: "now", Error: "failed"}); err != nil {
		t.Fatal(err)
	}
	events, _ := s.ListEventsSince(0, 100)
	raw, _ := json.Marshal(events)
	if strings.Contains(string(raw), "private-profile") {
		t.Fatal("profile secret in feed")
	}
	_ = s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d, _ := s.CLICheck("pi")
	v, _ = s.TerminalLaunch(tm.ID)
	if d.Version != "1.2" || v.Attempt.Error != "failed" {
		t.Fatal("diagnostics lost on reopen")
	}
	if err := s.SetCLIProfile(p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_cli_profile BEFORE INSERT ON events WHEN NEW.type='cli.profile' BEGIN SELECT RAISE(ABORT,'test refusal'); END`); err != nil {
		t.Fatal(err)
	}
	p.Name = "Not committed"
	if err := s.SetCLIProfile(p); err == nil {
		t.Fatal("profile committed without event")
	}
	if err := s.DeleteCLIProfile(p.ID); err == nil {
		t.Fatal("profile removed without event")
	}
	rows, _ := s.CLIProfiles()
	if len(rows) != 1 || rows[0].Name != "Renamed" {
		t.Fatal(rows)
	}
	if err := s.SetCLIProfile(CLIProfile{ID: "bad", Name: " ", CLI: "pi"}); err == nil {
		t.Fatal("blank profile name accepted")
	}
}
