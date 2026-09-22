package clipkgs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestArgvTable pins every vendor command PiCode builds. The vendor's own
// subcommand names are the contract here: a flag that moves breaks this table
// before it breaks a user's CLI.
func TestArgvTable(t *testing.T) {
	cases := []struct {
		name  string
		cli   string
		verb  Verb
		scope string
		p     Paths
		tgt   Target
		dir   string
		args  []string
	}{
		{"claude install user", "claude-code", VerbInstall, "user", Paths{}, Target{Source: "foo@mp"}, "", []string{"plugin", "install", "foo@mp", "-s", "user", "--json"}},
		{"claude install project runs in the workspace", "claude-code", VerbInstall, "project", Paths{Cwd: "/ws"}, Target{Source: "foo@mp"}, "/ws", []string{"plugin", "install", "foo@mp", "-s", "project", "--json"}},
		{"claude local is its own scope", "claude-code", VerbDisable, "local", Paths{Cwd: "/ws"}, Target{Name: "foo"}, "/ws", []string{"plugin", "disable", "foo", "-s", "local", "--json"}},
		{"claude update takes no scope", "claude-code", VerbUpdate, "user", Paths{}, Target{Name: "foo"}, "", []string{"plugin", "update", "foo", "--json"}},
		{"codex add", "codex", VerbInstall, "user", Paths{}, Target{Source: "gmail@openai-curated-remote"}, "", []string{"plugin", "add", "gmail@openai-curated-remote", "--json"}},
		{"codex remove", "codex", VerbRemove, "user", Paths{}, Target{Name: "gmail@openai-curated-remote"}, "", []string{"plugin", "remove", "gmail@openai-curated-remote", "--json"}},
		{"grok install from a repo", "grok", VerbInstall, "user", Paths{}, Target{Source: "user/repo#subdir"}, "", []string{"plugin", "install", "user/repo#subdir"}},
		{"grok update all", "grok", VerbUpdate, "user", Paths{}, Target{}, "", []string{"plugin", "update"}},
		{"hermes installs disabled", "hermes", VerbInstall, "user", Paths{}, Target{Source: "owner/repo"}, "", []string{"plugins", "install", "owner/repo", "--no-enable"}},
		{"hermes enable", "hermes", VerbEnable, "user", Paths{}, Target{Name: "picode-native"}, "", []string{"plugins", "enable", "picode-native"}},
		{"opencode global add", "opencode", VerbInstall, "user", Paths{}, Target{Source: "opencode-wakatime"}, "", []string{"plugin", "opencode-wakatime", "-g"}},
		{"opencode project add", "opencode", VerbInstall, "project", Paths{Cwd: "/ws"}, Target{Source: "opencode-wakatime"}, "/ws", []string{"plugin", "opencode-wakatime"}},
		{"muse local bundle carries a scope", "muse", VerbInstall, "project", Paths{Cwd: "/ws"}, Target{Source: "./bundle"}, "/ws", []string{"plugins", "install", "./bundle", "--scope", "project", "--json"}},
		{"muse marketplace ref carries none", "muse", VerbInstall, "user", Paths{}, Target{Source: "name@marketplace"}, "", []string{"plugins", "install", "name@marketplace", "--json"}},
		{"muse remove never deletes data", "muse", VerbRemove, "user", Paths{}, Target{Name: "pl"}, "", []string{"plugins", "remove", "pl", "--json"}},
		{"agy install is positional", "agy", VerbInstall, "user", Paths{}, Target{Source: "name@marketplace"}, "", []string{"plugin", "install", "name@marketplace"}},
		{"agy uninstall is positional", "agy", VerbRemove, "user", Paths{}, Target{Name: "pl"}, "", []string{"plugin", "uninstall", "pl"}},
		{"omp install project", "omp", VerbInstall, "project", Paths{Cwd: "/ws"}, Target{Source: "my-ext"}, "/ws", []string{"plugin", "install", "my-ext", "--json", "--scope=project"}},
		{"omp instal user", "omp", VerbInstall, "user", Paths{}, Target{Source: "my-ext"}, "", []string{"plugin", "install", "my-ext", "--json"}},
		{"omp update runs upgrade", "omp", VerbUpdate, "user", Paths{}, Target{Name: "ext"}, "", []string{"plugin", "upgrade", "ext", "--json"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, args, err := Argv(tc.cli, tc.verb, tc.p, Target{
				Name: tc.tgt.Name, Source: tc.tgt.Source, Scope: tc.scope, On: true,
			})
			if err != nil {
				t.Fatalf("Argv: %v", err)
			}
			if dir != tc.dir {
				t.Fatalf("dir = %q, want %q", dir, tc.dir)
			}
			if strings.Join(args, " ") != strings.Join(tc.args, " ") {
				t.Fatalf("args = %v, want %v", args, tc.args)
			}
		})
	}
}

// TestNoAutoConsentFlag is ADR-0167's hard rule: PiCode never answers a
// vendor's consent prompt for the user. The flag list is the vendors' own
// escape hatches, read from their --help.
func TestNoAutoConsentFlag(t *testing.T) {
	banned := map[string]string{
		"-y":                       "claude plugin install/uninstall confirmation",
		"--trust":                  "grok plugin install trust prompt",
		"--confirm":                "grok plugin uninstall multi-plugin confirmation",
		"--allow-tool-override":    "hermes plugins enable tool replacement grant",
		"--accept-command":         "claude marketplace-declared command hash",
		"--accept-hooks":           "hermes launch hook consent",
		"--delete-data":            "muse plugins remove data deletion",
		"--no-allow-tool-override": "hermes tool override denial (still the user's answer)",
	}
	for _, s := range registry() {
		for verb, build := range s.argv {
			for _, scope := range []string{"user", "project", "local"} {
				if _, err := s.scope(scope); err != nil {
					continue
				}
				_, args, err := build(Paths{Cwd: "/ws"}, Target{Name: "x", Source: "x@mp", Scope: scope, On: true})
				if err != nil {
					continue
				}
				for _, a := range args {
					if why, hit := banned[a]; hit {
						t.Fatalf("%s %s passes %s (%s)", s.cli, verb, a, why)
					}
				}
			}
		}
		for action, build := range s.market {
			_, args, err := build(Paths{Cwd: "/ws"}, MarketRequest{Action: action, Source: "s", Name: "n", Ref: "user"})
			if err != nil {
				continue
			}
			for _, a := range args {
				if why, hit := banned[a]; hit {
					t.Fatalf("%s marketplace %s passes %s (%s)", s.cli, action, a, why)
				}
			}
		}
	}
}

// TestCapabilitiesAreDerived pins what each CLI really exposes, so a later
// edit to a declaration cannot quietly claim a verb the vendor lacks.
func TestCapabilitiesAreDerived(t *testing.T) {
	want := map[string]Caps{
		"claude-code": {Install: true, Remove: true, Toggle: true, Update: true, Marketplace: true, Available: true, CatalogInstall: true},
		"codex":       {Install: true, Remove: true, Marketplace: true, Available: true, CatalogInstall: true},
		"grok":        {Install: true, Remove: true, Toggle: true, Update: true, Inspect: true, Marketplace: true, Available: true, CatalogInstall: true},
		"hermes":      {Install: true, Remove: true, Toggle: true, Update: true, Available: true, CatalogInstall: true},
		"opencode":    {Install: true, Remove: true, Toggle: true},
		"muse":        {Install: true, Remove: true, Toggle: true, Update: true, Inspect: true, Marketplace: true, Available: true, CatalogInstall: true},
		"agy":         {Install: true, Remove: true, Toggle: true},
		// Omp's catalog is information only: `omp plugin discover` prints a name
		// and a version and never the marketplace an install needs.
		"omp": {Install: true, Remove: true, Toggle: true, Update: true, Inspect: true, Marketplace: true, Available: true},
	}
	for cli, expect := range want {
		if got := Capabilities(cli); got != expect {
			t.Fatalf("%s caps = %+v, want %+v", cli, got, expect)
		}
	}
	// Pi keeps its own pane: this package answers nothing for it.
	if caps := Capabilities("pi"); caps != (Caps{}) {
		t.Fatalf("pi must have no driver here, got %+v", caps)
	}
	if For("pi") != nil {
		t.Fatal("pi must not be in the package catalog (ADR-0102)")
	}
	if Bin("pi") != "" {
		t.Fatal("pi has no vendor binary for this driver")
	}
}

// TestNotesDescribeAbsentVerbs keeps the copy honest: a note exists only where
// the control cannot, and never beside a control that does.
func TestNotesDescribeAbsentVerbs(t *testing.T) {
	// Named concepts the pane looks up by name, not verbs.
	conceptNotes := map[string]bool{"capabilities": true, "consent": true, "marketplace": true, "local": true, "approve": true}
	for _, s := range registry() {
		notes := Notes(s.cli)
		known := map[string]bool{}
		for _, verb := range []Verb{VerbInstall, VerbRemove, VerbEnable, VerbDisable, VerbUpdate, VerbInspect, VerbMarketAdd, VerbMarketList, VerbMarketUpdate, VerbMarketRemove} {
			known[string(verb)] = true
			marketKey := ""
			if strings.HasPrefix(string(verb), "marketplace-") {
				marketKey = string(verb)[len("marketplace-"):]
			}
			if s.argv[verb] == nil && (marketKey == "" || s.market[marketKey] == nil) {
				continue
			}
			if _, ok := notes[string(verb)]; ok {
				t.Fatalf("%s declares %s and also carries a note for it", s.cli, verb)
			}
		}
		for key := range notes {
			if known[key] || conceptNotes[key] {
				continue
			}
			t.Fatalf("%s has a note under an unknown key %q", s.cli, key)
		}
	}
	// The absent-verb notes that exist are the ones the pane renders.
	for cli, keys := range map[string][]string{
		"codex":       {"enable", "disable", "update"},
		"opencode":    {"update", "marketplace", "local"},
		"agy":         {"update", "marketplace-add"},
		"hermes":      {"marketplace-add"},
		"claude-code": {"consent"},
		"muse":        {"approve"},
	} {
		for _, key := range keys {
			if strings.TrimSpace(Notes(cli)[key]) == "" {
				t.Fatalf("%s is missing its %q note", cli, key)
			}
		}
	}
}

// TestScopeRules pins the scope gate: no guest CLI has a per-agent layer, a
// machine-only CLI refuses a project scope by name, and an unknown CLI is not
// silently treated as Pi.
func TestScopeRules(t *testing.T) {
	for _, cli := range CLIs() {
		if err := ValidateScope(cli, "agent"); !errors.Is(err, ErrAgentScope) {
			t.Fatalf("%s accepted the agent scope: %v", cli, err)
		}
	}
	for _, cli := range []string{"codex", "grok", "hermes", "agy"} {
		if err := ValidateScope(cli, "project"); !errors.Is(err, ErrScope) {
			t.Fatalf("%s accepted a project scope: %v", cli, err)
		}
	}
	for _, cli := range []string{"claude-code", "opencode", "muse", "omp"} {
		if err := ValidateScope(cli, "project"); err != nil {
			t.Fatalf("%s must accept a project scope: %v", cli, err)
		}
	}
	if err := ValidateScope("pi", "user"); !errors.Is(err, ErrNoDriver) {
		t.Fatalf("pi must not be resolved by this driver: %v", err)
	}
	if err := ValidateScope("codex", "local"); !errors.Is(err, ErrScope) {
		t.Fatalf("local is Claude's third scope only: %v", err)
	}
	// A project action without a workspace folder is refused rather than run
	// in the daemon's own directory.
	if _, _, err := Argv("omp", VerbInstall, Paths{}, Target{Source: "x", Scope: "project"}); !errors.Is(err, ErrNoWorkspace) {
		t.Fatalf("project scope without a workspace: %v", err)
	}
}

// TestMarkUpdates is the comparison itself: pair by id then by name, and never
// claim a version pair the comparator cannot read.
func TestMarkUpdates(t *testing.T) {
	installed := []Row{
		{ID: "picode-spare@picode-probe-live", Name: "picode-spare", Version: "0.2.0"},
		{ID: "grokly", Name: "grokly", Version: "1.0.0"},
		{ID: "odd", Name: "odd", Version: "2024-05-01"},
		{ID: "alone", Name: "alone", Version: "1.0.0"},
	}
	catalog := []Row{
		{ID: "picode-spare@picode-probe-live", Name: "picode-spare", Version: "0.3.0"},
		{ID: "grokly", Name: "grokly", Version: "1.0.0"},
		{ID: "odd", Name: "odd", Version: "main"},
		{ID: "other", Name: "other", Version: "9.9.9"},
	}
	if behind := markUpdates(installed, catalog); behind != 1 {
		t.Fatalf("behind = %d, want exactly the one newer semver pair", behind)
	}
	if !installed[0].UpdateAvailable || installed[0].Latest != "0.3.0" {
		t.Errorf("row = %+v, want the catalog's newer version named", installed[0])
	}
	if installed[1].UpdateAvailable || installed[1].Latest != "" {
		t.Errorf("row = %+v, want equal versions to carry no badge and no latest", installed[1])
	}
	if installed[2].UpdateAvailable || installed[2].Latest != "" {
		t.Errorf("row = %+v, want no claim about a version the comparator cannot read", installed[2])
	}
	if installed[3].UpdateAvailable {
		t.Errorf("a plugin the catalog does not carry must not be marked: %+v", installed[3])
	}
}

// TestCheckUpdatesAgainstAStubVendor covers the wiring: the roster and the
// catalog are the CLI's own answers, and a catalog that cannot be read is a
// note with no badges — never a silent "up to date".
func TestCheckUpdatesAgainstAStubVendor(t *testing.T) {
	writeStub := func(body string) {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "muse"), []byte("#!/bin/sh\n"+body), 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
		Invalidate("muse")
	}
	const roster = `printf '%s' '{"plugins":[{"active":true,"plugin":{"id":"picode-probe","version":"0.1.0"},"record":{"id":"picode-probe","version":"0.1.0","enabled":true,"cache_path":"/tmp/c"}}]}'`
	const catalog = `printf '%s' '{"available":[{"name":"picode-probe","version":"0.2.0","marketplace":"mp","status":"available","install":{"source":"/tmp/x"}}],"skipped":[],"warnings":[]}'`
	writeStub(`case "$*" in *--available*) ` + catalog + ` ;; *) ` + roster + ` ;; esac`)

	rep, err := CheckUpdates(context.Background(), "muse", Paths{Home: t.TempDir()}, "user", true)
	if err != nil {
		t.Fatalf("CheckUpdates: %v", err)
	}
	if len(rep.Rows) != 1 || !rep.Rows[0].UpdateAvailable || rep.Rows[0].Latest != "0.2.0" {
		t.Fatalf("rows = %+v, want the installed row marked with the catalog's version", rep.Rows)
	}
	if rep.CheckedAt == "" {
		t.Error("a check must say when it ran")
	}

	writeStub(`case "$*" in *--available*) printf '%s\n' 'marketplace read failed' >&2; exit 1 ;; *) ` + roster + ` ;; esac`)
	rep, err = CheckUpdates(context.Background(), "muse", Paths{Home: t.TempDir()}, "user", true)
	if err != nil {
		t.Fatalf("a failed catalog must not fail the roster: %v", err)
	}
	if len(rep.Rows) != 1 || rep.Rows[0].UpdateAvailable {
		t.Fatalf("rows = %+v, want no badge when nothing was compared", rep.Rows)
	}
	if !strings.Contains(rep.Note, "Could not read") {
		t.Errorf("note = %q, want the reason no comparison happened", rep.Note)
	}

	// A CLI without an update verb refuses the check rather than inventing one.
	if _, err := CheckUpdates(context.Background(), "codex", Paths{}, "user", true); err == nil {
		t.Fatal("codex has no plugin update verb; the check must refuse")
	}
}

// TestTheDefaultScopeReadsTheMachineScope covers a defect that shipped. The web
// client omits `scope` for the machine — it sets the query only for another
// scope — and Claude Code's rows name theirs ("user"), so a read that passed the
// empty scope through to the parser dropped every row: the pane showed an empty
// list for a CLI that has plugins, and the file's own rows were unreachable.
func TestTheDefaultScopeReadsTheMachineScope(t *testing.T) {
	const installed = `{"installed":[` +
		`{"id":"picode-native@local","version":"1.0.0","scope":"user","enabled":true,"installPath":"/tmp/p"},` +
		`{"id":"memory-kit@synced","version":"2.1.0","scope":"user","enabled":false}],"available":[]}`
	const catalog = `{"installed":[],"available":[{"pluginId":"picode-native@local","name":"picode-native","version":"1.1.0","marketplaceName":"local"}]}`
	dir := t.TempDir()
	stub := "#!/bin/sh\ncase \" $* \" in *\" --available \"*) printf '%s' '" + catalog + "' ;; *) printf '%s' '" + installed + "' ;; esac\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	Invalidate("claude-code")

	rep, err := List(context.Background(), "claude-code", Paths{}, "", true)
	if err != nil {
		t.Fatalf("List with the default scope: %v", err)
	}
	if len(rep.Rows) != 2 {
		t.Fatalf("rows = %d (%+v), want the machine's two: the empty scope is the machine scope", len(rep.Rows), rep.Rows)
	}

	// The check reads both halves through the same scope, and its cache write
	// must land under the key the roster read later uses.
	checked, err := CheckUpdates(context.Background(), "claude-code", Paths{}, "", true)
	if err != nil {
		t.Fatalf("CheckUpdates with the default scope: %v", err)
	}
	if len(checked.Rows) != 2 {
		t.Fatalf("checked rows = %d, want the same two", len(checked.Rows))
	}
	var marked bool
	for _, row := range checked.Rows {
		if row.Name == "picode-native" && row.UpdateAvailable && row.Latest == "1.1.0" {
			marked = true
		}
	}
	if !marked {
		t.Fatalf("rows = %+v, want the catalog's newer version marked on picode-native", checked.Rows)
	}
}
