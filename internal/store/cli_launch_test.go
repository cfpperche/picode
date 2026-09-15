package store

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

func TestCLIConfigImportAndAtomicEvents(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.ImportCLIConfigs(map[string]bool{"pi": true, "codex": true}); err != nil {
		t.Fatal(err)
	}
	c, _, _ := s.CLIConfig("codex")
	if !c.Integration {
		t.Fatal("legacy preference lost")
	}
	c.Integration = false
	c.Env = map[string]string{"TEST_SECRET": "not-in-feed"}
	if err := s.SetCLIConfig("codex", c); err != nil {
		t.Fatal(err)
	}
	before, _ := s.LatestEventID()
	if err := s.ImportCLIConfigs(map[string]bool{"codex": true}); err != nil {
		t.Fatal(err)
	}
	after, _ := s.LatestEventID()
	if before != after {
		t.Fatal("import repeated events")
	}
	c, _, _ = s.CLIConfig("codex")
	if c.Integration {
		t.Fatal("legacy flags overwrote saved preference")
	}
	events, _ := s.ListEventsSince(0, 100)
	raw, _ := json.Marshal(events)
	if strings.Contains(string(raw), "not-in-feed") {
		t.Fatal("secret in change feed")
	}
	_, err = s.db.Exec(`CREATE TRIGGER reject_cli_event BEFORE INSERT ON events WHEN NEW.type='cli.updated' BEGIN SELECT RAISE(ABORT,'test refusal'); END`)
	if err != nil {
		t.Fatal(err)
	}
	c.Integration = true
	if err := s.SetCLIConfig("codex", c); err == nil {
		t.Fatal("event refusal accepted")
	}
	c, _, _ = s.CLIConfig("codex")
	if c.Integration {
		t.Fatal("config committed without event")
	}
}

func TestImportCLIConfigsDefaultsIntegrationOn(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.ImportCLIConfigs(map[string]bool{"opencode": false}); err != nil {
		t.Fatal(err)
	}
	off, _, _ := s.CLIConfig("opencode")
	if off.Integration {
		t.Fatal("explicit false was not stored")
	}
	on, _, _ := s.CLIConfig("pi")
	if !on.Integration {
		t.Fatal("missing enabled.json key must default Activity on")
	}
	s2, err := Open(filepath.Join(t.TempDir(), "store2.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if err := s2.ImportCLIConfigs(nil); err != nil {
		t.Fatal(err)
	}
	for _, cli := range clilaunch.Catalog() {
		c, found, _ := s2.CLIConfig(cli.ID)
		if !cli.Integrable() {
			if !found || c.Integration {
				t.Fatalf("%s has no adapter: found=%v integration=%v", cli.ID, found, c.Integration)
			}
			continue
		}
		if !found || !c.Integration {
			t.Fatalf("%s: found=%v integration=%v", cli.ID, found, c.Integration)
		}
	}
}

func TestSeedCatalogIntegrationDefaultsDecisionTable(t *testing.T) {
	type row struct {
		name   string
		before clilaunch.Config
		seeded bool
		wantOn bool
	}
	cases := []row{
		{name: "empty false", before: clilaunch.Config{}, wantOn: true},
		{name: "already on", before: clilaunch.Config{Integration: true}, wantOn: true},
		{name: "customized off", before: clilaunch.Config{Executable: "/opt/opencode", Integration: false}, wantOn: false},
		{name: "already seeded", before: clilaunch.Config{}, seeded: true, wantOn: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := Open(filepath.Join(t.TempDir(), "store.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if err := s.SetCLIConfig("opencode", tc.before); err != nil {
				t.Fatal(err)
			}
			if tc.seeded {
				if err := s.SetSetting(catalogIntegrationSeedKey, "1"); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.SeedCatalogIntegrationDefaults(); err != nil {
				t.Fatal(err)
			}
			c, _, _ := s.CLIConfig("opencode")
			if c.Integration != tc.wantOn {
				t.Fatalf("integration=%v want %v", c.Integration, tc.wantOn)
			}
			if err := s.SeedCatalogIntegrationDefaults(); err != nil {
				t.Fatal(err)
			}
			c2, _, _ := s.CLIConfig("opencode")
			if c2.Integration != c.Integration {
				t.Fatal("second seed changed the row")
			}
		})
	}
}

func TestTerminalLaunchPersistenceAndCascade(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "store.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	w, _ := s.AddWorkspace("project", dir)
	tm, _ := s.CreateTerminalIn(w.ID, "Codex", "")
	value := "private-value"
	if err := s.SetTerminalLaunch(tm.ID, "codex", clilaunch.Overrides{Env: map[string]*string{"PRIVATE": &value}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalLaunchApplied(tm.ID, clilaunch.Snapshot{Executable: "/bin/codex", Fingerprint: "one"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalLaunch(tm.ID, "pi", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	v, err := s.TerminalLaunch(tm.ID)
	if err != nil || v.CLI != "pi" || v.Applied.Fingerprint != "one" {
		t.Fatalf("saved=%+v err=%v", v, err)
	}
	if _, err := s.RemoveWorkspace(w.ID); err != nil {
		t.Fatal(err)
	}
	if v, err := s.TerminalLaunch(tm.ID); err != nil || v != nil {
		t.Fatalf("orphan launch=%+v err=%v", v, err)
	}
	if err := s.SetTerminalLaunch("missing", "pi", clilaunch.Overrides{}); err != ErrNotFound {
		t.Fatalf("missing terminal: %v", err)
	}
}
