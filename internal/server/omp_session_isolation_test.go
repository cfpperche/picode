package server

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
)

func TestOmpAgentSessionDirIsDurableAndScoped(t *testing.T) {
	dataDir := t.TempDir()
	got := ompAgentSessionDir(dataDir, "agent-123")
	want := filepath.Join(dataDir, "omp-sessions", "agent-123")
	if got != want {
		t.Fatalf("ompAgentSessionDir = %q, want %q", got, want)
	}
	if other := ompAgentSessionDir(dataDir, "agent-456"); other == got {
		t.Fatal("different agents must not share the Omp session directory")
	}
}

// ompLaunchFixture is one Omp agent bound to a terminal whose CLI resolves to a
// stub binary, so a launch can be prepared without the vendor being installed.
// The agent carries the packages and the isolation switch the case asks for.
func ompLaunchFixture(t *testing.T, cfg clilaunch.Config, packages []string, isolated bool) (Deps, string, *store.TerminalLaunch) {
	t.Helper()
	data := t.TempDir()
	st, err := store.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	w, err := st.AddWorkspace("Fixture", data)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(w.ID, "Fixture", data)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(data, "omp")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if cfg.Executable == "" {
		cfg.Executable = binary
	}
	if err := st.SetCLIConfig("omp", cfg); err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(term.ID, "omp", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	a, err := st.AddAgentWithCLI(w.ID, "omp", "Atlas", data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpdateAgent(a.ID, store.AgentPatch{TerminalID: &term.ID}); err != nil {
		t.Fatal(err)
	}
	if len(packages) > 0 {
		if _, err := st.SetAgentPackages(a.ID, packages); err != nil {
			t.Fatal(err)
		}
	}
	if isolated {
		on := true
		if _, err := st.UpdateAgent(a.ID, store.AgentPatch{PackagesIsolated: &on}); err != nil {
			t.Fatal(err)
		}
	}
	launch, err := st.TerminalLaunch(term.ID)
	if err != nil {
		t.Fatal(err)
	}
	return Deps{Store: st, DataDir: data}, data, launch
}

// The agent's own scope reaches the next Omp launch (ADR-0176 slice 4): every
// stored entry as `-e`, and "only this agent's packages" as the two flags this
// CLI has. Pi's half of the same contract is the store's own CLIFlags; this is
// the guest half, and it is what makes the pane's agent scope real for Omp
// rather than a list the CLI never sees.
func TestOmpLaunchCarriesTheAgentsScope(t *testing.T) {
	rows := []struct {
		name     string
		packages []string
		isolated bool
	}{
		{name: "no scope declared"},
		{name: "one entry", packages: []string{"npm:pi-browser"}},
		{name: "isolated with an entry", packages: []string{"npm:pi-browser"}, isolated: true},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			deps, cwd, launch := ompLaunchFixture(t, clilaunch.Config{}, row.packages, row.isolated)
			prepared, err := prepareCLITerminal(deps, cwd, launch)
			if err != nil {
				t.Fatal(err)
			}
			defer prepared.discard()
			body, err := os.ReadFile(prepared.script)
			if err != nil {
				t.Fatal(err)
			}
			// The script shell-quotes every argument, so the words are asserted
			// rather than one rendered line.
			hasEntry := strings.Contains(string(body), "npm:pi-browser")
			hasIsolation := strings.Contains(string(body), "--no-extensions")
			if hasEntry != (len(row.packages) > 0) {
				t.Fatalf("launch script entry = %v, want %v:\n%s", hasEntry, len(row.packages) > 0, body)
			}
			if hasIsolation != row.isolated {
				t.Fatalf("launch script isolation = %v, want %v:\n%s", hasIsolation, row.isolated, body)
			}
		})
	}
}

// A --trusted-extension flag and PiCode's own entries cannot ride one Omp
// launch: the CLI refuses the run outright ("--trusted-extension cannot be
// combined with --extension, -e, or --hook"). The integration guard covers its
// activity extension; the agent's own list is injected after that guard runs,
// so it carries the same refusal (ADR-0176 slice 4).
func TestOmpLaunchRefusesTrustedExtensionWithAgentEntries(t *testing.T) {
	trusted := clilaunch.Config{Args: []string{"--trusted-extension"}}
	deps, cwd, launch := ompLaunchFixture(t, trusted, []string{"npm:pi-browser"}, false)
	if _, err := prepareCLITerminal(deps, cwd, launch); err == nil {
		t.Fatal("a launch the CLI would refuse was prepared")
	} else if !strings.Contains(err.Error(), "--trusted-extension") {
		t.Fatalf("refusal does not name the flag: %v", err)
	}
	// Without entries there is nothing to conflict with: the same flags launch.
	deps, cwd, launch = ompLaunchFixture(t, trusted, nil, false)
	if _, err := prepareCLITerminal(deps, cwd, launch); err != nil {
		t.Fatalf("a --trusted-extension terminal with no agent entries must still launch: %v", err)
	}
}

// The agent's own scope reaches Omp's launch as `-e` per entry, and isolation
// as the two flags this CLI has — never one it would refuse (ADR-0176 slice 4).
// The order is the flags first, then the entries, exactly as Pi's list is
// appended to pi's launch.
func TestAgentOmpScopeFlagsFollowTheStoredList(t *testing.T) {
	rows := []struct {
		name  string
		agent store.Agent
		want  []string
	}{
		{"no list, no switch", store.Agent{}, nil},
		{"one entry, trimmed", store.Agent{Packages: []string{" npm:pi-browser "}}, []string{"-e", "npm:pi-browser"}},
		{"blank entries are not arguments", store.Agent{Packages: []string{"", "   "}}, nil},
		{"isolation alone", store.Agent{PackagesIsolated: true}, []string{"--no-extensions", "--no-skills"}},
		{"every entry rides", store.Agent{Packages: []string{"/tmp/a.ts", "/tmp/b.ts"}}, []string{"-e", "/tmp/a.ts", "-e", "/tmp/b.ts"}},
		{"isolated and listed", store.Agent{PackagesIsolated: true, Packages: []string{"x"}}, []string{"--no-extensions", "--no-skills", "-e", "x"}},
	}
	for _, row := range rows {
		if got := agentOmpScopeFlags(row.agent); !reflect.DeepEqual(got, row.want) {
			t.Fatalf("%s: %v, want %v", row.name, got, row.want)
		}
	}
}
