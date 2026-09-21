package clipkgs

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
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
		"claude-code": {Install: true, Remove: true, Toggle: true, Update: true, Marketplace: true, Available: true},
		"codex":       {Install: true, Remove: true, Marketplace: true, Available: true},
		"grok":        {Install: true, Remove: true, Toggle: true, Update: true, Inspect: true, Marketplace: true, Available: true},
		"hermes":      {Install: true, Remove: true, Toggle: true, Update: true, Available: true},
		"opencode":    {Install: true, Remove: true, Toggle: true},
		"muse":        {Install: true, Remove: true, Toggle: true, Update: true, Inspect: true, Marketplace: true, Available: true},
		"agy":         {Install: true, Remove: true, Toggle: true},
		"omp":         {Install: true, Remove: true, Toggle: true, Update: true, Inspect: true, Marketplace: true},
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

// TestCLIListMatchesJS is the seam between the Go catalog and the list the
// pane uses to decide whether to ask for a guest view. Two files, one truth
// (the shape internal/clisettings already established).
func TestCLIListMatchesJS(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "web", "shared", "domain", "cliPackages.js"))
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`GUEST_PACKAGES = \[([^\]]*)\]`).FindSubmatch(body)
	if match == nil {
		t.Fatal("GUEST_PACKAGES not found in web/shared/domain/cliPackages.js")
	}
	var js []string
	for _, part := range strings.Split(string(match[1]), ",") {
		if id := strings.Trim(strings.TrimSpace(part), `"`); id != "" {
			js = append(js, id)
		}
	}
	got := CLIs()
	if strings.Join(got, ",") != strings.Join(js, ",") {
		t.Fatalf("the UI list and the Go catalog disagree:\n  go: %v\n  js: %v", got, js)
	}
	for _, id := range got {
		if id == "pi" {
			t.Fatal("Pi keeps its own pane (ADR-0102) and must not be in this catalog")
		}
	}
}
