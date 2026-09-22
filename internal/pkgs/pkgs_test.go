package pkgs

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/pipkg"
)

// The registry is the decision table: "" is Pi, the eight guests resolve by
// name, and anything else is unknown rather than a default driver.
func TestDriverForResolvesPiAndTheGuests(t *testing.T) {
	rows := []struct {
		cli   string
		id    string
		known bool
	}{
		{"", "pi", true},
		{"pi", "pi", true},
		{" omp ", "omp", true},
		{"claude-code", "claude-code", true},
		{"nope", "nope", false},
		{"pipi", "pipi", false},
	}
	for _, row := range rows {
		if got := DriverFor(row.cli).ID(); got != row.id {
			t.Fatalf("DriverFor(%q).ID() = %q, want %q", row.cli, got, row.id)
		}
		if got := Known(row.cli); got != row.known {
			t.Fatalf("Known(%q) = %v, want %v", row.cli, got, row.known)
		}
	}
}

// Who declares the agent scope, and the mechanism each one has for it: the
// layer is PiCode's own list on the agent row, never a vendor's, so a CLI whose
// launch cannot pass the entries on must not declare it — the radio would be a
// control that cannot work (ADR-0176 slice 4; the plan's decision table). The
// isolation switch travels with it for the same reason: it is the agent row's
// own flag, and it is offered where a launch honours it.
func TestWhoDeclaresTheAgentScope(t *testing.T) {
	mechanisms := map[string]string{
		"pi":  "pi -e per entry, plus pi's own isolation flags",
		"omp": "omp -e per entry, plus --no-extensions --no-skills when isolated",
	}
	for _, cli := range CLIs() {
		hasAgent := slices.ContainsFunc(DriverFor(cli).Scopes(), func(s ScopeRow) bool { return s.ID == Agent })
		_, hasMechanism := mechanisms[cli]
		if hasAgent != hasMechanism {
			t.Fatalf("%s: declares the agent scope = %v, has a launch mechanism = %v", cli, hasAgent, hasMechanism)
		}
		if got := DriverFor(cli).Caps().IsolatedSwitch; got != hasMechanism {
			t.Fatalf("%s: IsolatedSwitch = %v, want %v", cli, got, hasMechanism)
		}
	}
}

// The agent scope answers PiCode's own list for a guest exactly as it does for
// Pi: no vendor call is made (there is nothing in the CLI to call — the launch
// is what passes the entries on), the rows carry the store's entries with the
// agent's own vendor word, and the radio is offered only when the read named an
// agent (ADR-0176 slice 4).
func TestGuestAgentScopeAnswersPiCodeList(t *testing.T) {
	rep, err := DriverFor("omp").List(context.Background(), Query{
		Scope:        Agent,
		AgentName:    "Atlas",
		AgentSources: []string{" npm:pi-browser ", "", "git:github.com/x/y"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Rows) != 2 {
		t.Fatalf("rows = %+v", rep.Rows)
	}
	first := rep.Rows[0]
	if first.Name != "pi-browser" || first.Source != "npm:pi-browser" || first.Scope != Agent || first.Vendor != "agent" || !first.Enabled {
		t.Fatalf("row = %+v", first)
	}
	if !slices.ContainsFunc(rep.Scopes, func(s ScopeRow) bool { return s.ID == Agent }) {
		t.Fatalf("no agent row with an agent named: %+v", rep.Scopes)
	}
	if rep.AgentName != "Atlas" {
		t.Fatalf("agent name = %q, want Atlas", rep.AgentName)
	}
	// The pane also names the layer by the word its own row carries
	// (`vendor=agent`), which is what the unified reads send: one answer.
	byWord, err := DriverFor("omp").List(context.Background(), Query{
		Vendor:       "agent",
		AgentName:    "Atlas",
		AgentSources: []string{"npm:pi-browser"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(byWord.Rows) != 1 || byWord.Rows[0].Source != "npm:pi-browser" {
		t.Fatalf("the vendor-word spelling answered %+v", byWord.Rows)
	}
	// And the badge read answers empty for that layer rather than asking the
	// vendor for a catalog the layer does not have.
	updates, err := DriverFor("omp").CheckUpdates(context.Background(), Query{Vendor: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updates.Rows) != 0 {
		t.Fatalf("updates at the agent layer = %+v", updates.Rows)
	}
	// The same read with no agent named offers no agent radio: the layer is
	// one agent's list, so a read that named none cannot answer it.
	bare, err := DriverFor("omp").List(context.Background(), Query{Scope: Agent})
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(bare.Scopes, func(s ScopeRow) bool { return s.ID == Agent }) {
		t.Fatalf("agent radio offered with no agent: %+v", bare.Scopes)
	}
	if len(bare.Rows) != 0 {
		t.Fatalf("rows with no entries = %+v", bare.Rows)
	}
}

// Pi's settings become unified rows: the class is ours, the vendor word is
// Pi's, and the agent's own list arrives as data from the caller.
func TestPiDriverMapsSettingsToUnifiedRows(t *testing.T) {
	userDir := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return userDir }
	t.Cleanup(func() { pipkg.UserDir = old })

	mustWrite(t, filepath.Join(userDir, "settings.json"), `{"packages":["npm:pi-web-search"]}`)
	ws := t.TempDir()
	mustWrite(t, filepath.Join(ws, ".pi", "settings.json"), `{"packages":["/abs/pi-roles"]}`)

	rep, err := DriverFor("pi").List(context.Background(), Query{
		WorkspacePath: ws,
		AgentSources:  []string{"npm:pi-inbox"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.CLI != "pi" || rep.Catalog != CatalogGallery || rep.Gallery != pipkg.Gallery {
		t.Fatalf("report = %+v", rep)
	}
	if !rep.Capabilities["webSearch"] {
		t.Fatal("a configured search package must set webSearch")
	}
	if rep.Caps.Install != true || rep.Caps.Config != true {
		t.Fatalf("caps = %+v", rep.Caps)
	}
	want := []Row{
		{Name: "pi-web-search", Scope: Machine, Vendor: "user", Kind: "npm"},
		{Name: "pi-roles", Scope: Workspace, Vendor: "project", Kind: "path"},
		{Name: "pi-inbox", Scope: Agent, Vendor: "agent", Kind: "npm"},
	}
	if len(rep.Rows) != len(want) {
		t.Fatalf("rows = %+v", rep.Rows)
	}
	for i, w := range want {
		got := rep.Rows[i]
		if got.CLI != "pi" || got.Name != w.Name || got.Scope != w.Scope || got.Vendor != w.Vendor || got.Kind != w.Kind {
			t.Fatalf("row %d = %+v, want %+v", i, got, w)
		}
		if !got.Enabled {
			t.Fatalf("row %d is not enabled: %+v", i, got)
		}
	}
}

// A guest's word for a scope is kept beside the class, and its caps come from
// the declaration — never restated by hand.
func TestGuestDriverKeepsTheVendorScopeWords(t *testing.T) {
	omp := DriverFor("omp")
	if omp.Caps().Install != true || omp.Caps().Toggle != true {
		t.Fatalf("omp caps = %+v", omp.Caps())
	}
	scopes := omp.Scopes()
	if len(scopes) == 0 || scopes[0].ID != Machine || scopes[0].Vendor != "user" {
		t.Fatalf("omp scopes = %+v", scopes)
	}
	classes := map[Scope]string{}
	for _, s := range scopes {
		classes[s.ID] = s.Vendor
	}
	if classes[Workspace] != "project" {
		t.Fatalf("omp workspace vendor word = %q, want project", classes[Workspace])
	}
	// Claude Code's third layer is `local` (uncommitted project scope): the
	// class is the workspace, the word stays the CLI's.
	claude := map[string]Scope{}
	for _, s := range DriverFor("claude-code").Scopes() {
		claude[s.Vendor] = s.ID
	}
	if claude["local"] != Workspace || claude["user"] != Machine {
		t.Fatalf("claude scopes = %+v", claude)
	}
}

// The transport declaration is one fact per mutation verb, and OpenCode is the
// mixed CLI that pins it: its install is its own `plugin` command while its
// removal is a write of its own config file. The single bool this replaced
// (`Caps.Async`) could only have told the truth about one of the two.
func TestTheTransportDeclarationIsPerVerb(t *testing.T) {
	oc := DriverFor("opencode").Caps()
	if !oc.Install || !oc.Remove {
		t.Fatalf("OpenCode exposes both verbs: %+v", oc)
	}
	if !oc.Lane.Install || oc.Lane.Remove {
		t.Fatalf("OpenCode's install is its own command and its removal a write of its own file: %+v", oc.Lane)
	}
	// It has neither of the other two verbs, so neither can be on the lane.
	if oc.Lane.Update || oc.Lane.Marketplace {
		t.Fatalf("OpenCode has no update or marketplace verb: %+v", oc.Lane)
	}
	// Every mutation of a CLI that has vendor commands for them is the lane's.
	omp := DriverFor("omp").Caps()
	if !omp.Lane.Install || !omp.Lane.Remove || !omp.Lane.Update || !omp.Lane.Marketplace {
		t.Fatalf("omp's mutations run the vendor's own commands: %+v", omp.Lane)
	}
	// Pi keeps its own mutations: nothing of its is reserved as a job.
	if pi := DriverFor("pi").Caps(); pi.Lane != (Transport{}) {
		t.Fatalf("pi runs every mutation itself: %+v", pi.Lane)
	}
}

// A verb the declaration puts off the lane performs the mutation instead of
// handing back a command, and the empty command is how the route reads that:
// nothing for a lane to run, so the answer is the CLI's fresh list. Every byte
// of the config file outside the removed module survives.
func TestGuestRemoveOpenCodeWritesTheFileItReads(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", "")
	cfg := filepath.Join(dir, ".config", "opencode")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(cfg, "opencode.json")
	body := "{\n  \"plugin\": [\"opencode-wakatime\", \"keep-me\"],\n  \"model\": \"x\"\n}\n"
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd, err := DriverFor("opencode").Remove(context.Background(), Query{}, Target{
		Name: "opencode-wakatime", Source: "opencode-wakatime",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Exe != "" || cmd.Line != "" || len(cmd.Args) != 0 {
		t.Fatalf("a write has no command to hand a lane: %+v", cmd)
	}
	at := strings.Index(body, `"opencode-wakatime", `)
	if at < 0 {
		t.Fatal("the fixture does not name the removed module")
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body[:at]+body[at+len(`"opencode-wakatime", `):] {
		t.Fatalf("the file after the write = %q, want the original with only the module gone", got)
	}
}

func TestSourceNameReadsEveryKind(t *testing.T) {
	rows := map[string]string{
		"npm:pi-web-search":      "pi-web-search",
		"npm:@scope/pkg@1.2.3":   "@scope/pkg",
		"/home/u/picode/pkg":     "pkg",
		"git:github.com/x/y":     "y",
		"https://github.com/a/b": "b",
		"":                       "",
	}
	for source, want := range rows {
		if got := SourceName(source); got != want {
			t.Fatalf("SourceName(%q) = %q, want %q", source, got, want)
		}
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
