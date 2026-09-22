package pkgs

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clipkgs"
)

// engineCommand is the command the engine itself builds for one verb: the argv
// the job lane spawned and the line the pane copied while the handlers called
// clipkgs directly. The driver's verbs must answer exactly this (slice 2b).
func engineCommand(t *testing.T, cli string, verb clipkgs.Verb, p clipkgs.Paths, target clipkgs.Target) Command {
	t.Helper()
	dir, args, err := clipkgs.Argv(cli, verb, p, target)
	if err != nil {
		t.Fatalf("engine argv: %v", err)
	}
	line, err := clipkgs.Command(cli, verb, p, target)
	if err != nil {
		t.Fatalf("engine command: %v", err)
	}
	return Command{Exe: clipkgs.Bin(cli), Args: args, Dir: dir, Line: line}
}

// sameCommand compares both halves of one command: the argv the lane spawns and
// the line the pane hands a person.
func sameCommand(t *testing.T, got, want Command) {
	t.Helper()
	if got.Exe != want.Exe || got.Dir != want.Dir || got.Line != want.Line || !slices.Equal(got.Args, want.Args) {
		t.Fatalf("command drifted:\n got %+v\nwant %+v", got, want)
	}
}

// sameRefusal compares two errors as the pane receives them: the same words,
// and the same fact underneath.
func sameRefusal(t *testing.T, got, want error) {
	t.Helper()
	if got == nil || want == nil {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got.Error() != want.Error() {
		t.Fatalf("refusal drifted:\n got %q\nwant %q", got, want)
	}
}

// engineRun is the refusal the engine itself answered for one verb, as the
// handler received it.
func engineRun(t *testing.T, cli string, verb clipkgs.Verb, p clipkgs.Paths, target clipkgs.Target) error {
	t.Helper()
	_, err := clipkgs.Run(context.Background(), cli, verb, p, target)
	if err == nil {
		t.Fatalf("%s %s did not refuse", cli, verb)
	}
	return err
}

// requireCLIBinary skips when the vendor binary a case needs is not
// installed.
//
// Almost every row here is pure argv construction and needs nothing on
// PATH. Omp's removal is the exception: it asks the engine whether the
// target is one of its `extensions`, and the engine answers by running
// `omp config get`. With omp installed — which it is on the machine this
// was written on — the row passes; on a runner it failed with
// `exec: "omp": executable file not found in $PATH`, which is a fact
// about the runner and not about the lane. Reproduce either state by
// putting omp on or off PATH.
func requireCLIBinary(t *testing.T, cli string) {
	t.Helper()
	if _, err := exec.LookPath(clipkgs.Bin(cli)); err != nil {
		t.Skipf("%s is not installed: %v", clipkgs.Bin(cli), err)
	}
}

// The lane's verbs build the command the engine built: the argv the job lane
// spawns, the directory it runs in and the line the pane copies. They never run
// it, which is why the same table covers every CLI (ADR-0176 slice 2b).
func TestGuestLaneCommandsMatchTheEngineByteForByte(t *testing.T) {
	ctx := context.Background()
	ws := t.TempDir()

	cases := []struct {
		name   string
		cli    string
		scope  string
		verb   clipkgs.Verb
		target Target
		call   func(Driver, Query, Target) (Command, error)
		// needsBin: this row reaches the vendor binary, so it is skipped
		// where that binary is not installed.
		needsBin bool
	}{
		{
			name: "claude code installs into its local layer", cli: "claude-code", scope: "local",
			verb: clipkgs.VerbInstall, target: Target{Source: "probe@mp"},
			call: func(d Driver, q Query, t Target) (Command, error) { return d.Install(ctx, q, t) },
		},
		{
			name: "omp installs a plugin", cli: "omp", scope: "user",
			verb: clipkgs.VerbInstall, target: Target{Source: "my-ext"},
			call: func(d Driver, q Query, t Target) (Command, error) { return d.Install(ctx, q, t) },
		},
		{
			// The one row that reaches the vendor binary; see requireCLIBinary.
			name: "omp removes a project plugin", cli: "omp", scope: "project", needsBin: true,
			verb: clipkgs.VerbRemove, target: Target{Name: "ext", Source: "ext"},
			call: func(d Driver, q Query, t Target) (Command, error) { return d.Remove(ctx, q, t) },
		},
		{
			name: "codex removes a marketplace plugin", cli: "codex", scope: "user",
			verb: clipkgs.VerbRemove, target: Target{Name: "pl", Source: "pl"},
			call: func(d Driver, q Query, t Target) (Command, error) { return d.Remove(ctx, q, t) },
		},
		{
			name: "grok updates a plugin", cli: "grok", scope: "user",
			verb: clipkgs.VerbUpdate, target: Target{Name: "probe"},
			call: func(d Driver, q Query, t Target) (Command, error) { return d.Update(ctx, q, t) },
		},
		{
			name: "agy installs a plugin", cli: "agy", scope: "user",
			verb: clipkgs.VerbInstall, target: Target{Source: "probe"},
			call: func(d Driver, q Query, t Target) (Command, error) { return d.Install(ctx, q, t) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.needsBin {
				requireCLIBinary(t, tc.cli)
			}
			q := Query{Vendor: tc.scope, WorkspacePath: ws}
			got, err := tc.call(DriverFor(tc.cli), q, tc.target)
			if err != nil {
				t.Fatal(err)
			}
			want := engineCommand(t, tc.cli, tc.verb, clipkgs.Paths{Cwd: ws}, clipkgs.Target{
				Name: tc.target.Name, Source: tc.target.Source, Scope: tc.scope, On: tc.target.On,
			})
			sameCommand(t, got, want)
		})
	}

	// A marketplace fetch is the lane's too, and it builds through the
	// marketplace table rather than the verb table.
	markets := []struct {
		name  string
		cli   string
		scope string
		req   MarketRequest
	}{
		{name: "claude code adds a source", cli: "claude-code", scope: "user", req: MarketRequest{Action: "add", Source: "owner/repo", Name: "mp", Ref: "project"}},
		{name: "omp updates a source", cli: "omp", scope: "user", req: MarketRequest{Action: "update", Name: "mp", Ref: "project"}},
		{name: "muse adds a named source", cli: "muse", scope: "user", req: MarketRequest{Action: "add", Source: "owner/repo", Name: "mp"}},
	}
	for _, tc := range markets {
		t.Run(tc.name, func(t *testing.T) {
			q := Query{Vendor: tc.scope, WorkspacePath: ws}
			got, err := DriverFor(tc.cli).Marketplace(ctx, q, tc.req)
			if err != nil {
				t.Fatal(err)
			}
			dir, args, err := clipkgs.MarketArgv(tc.cli, tc.req.Action, clipkgs.Paths{Cwd: ws}, clipkgs.MarketRequest{
				Action: tc.req.Action, Source: tc.req.Source, Name: tc.req.Name, Ref: tc.req.Ref,
			})
			if err != nil {
				t.Fatal(err)
			}
			line, err := clipkgs.MarketCommand(tc.cli, tc.req.Action, clipkgs.Paths{Cwd: ws}, clipkgs.MarketRequest{
				Action: tc.req.Action, Source: tc.req.Source, Name: tc.req.Name, Ref: tc.req.Ref,
			})
			if err != nil {
				t.Fatal(err)
			}
			sameCommand(t, got, Command{Exe: clipkgs.Bin(tc.cli), Args: args, Dir: dir, Line: line})
		})
	}
}

// A toggle runs, so its command and its refusal are compared where a vendor
// exists: the argv and the line stay the engine's, and the vendor's own words
// ride the error verbatim (the pane shows them).
func TestGuestToggleCommandAndRefusalMatchTheEngine(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	stubGuest(t, "grok", `case "$*" in
  *disable*) printf '%s\n' 'refusing to disable without confirmation'; exit 1 ;;
  *) exit 0 ;;
esac`)
	q := Query{Vendor: "user"}
	target := Target{Name: "probe", On: false}

	got, err := DriverFor("grok").Toggle(ctx, q, target)
	sameRefusal(t, err, engineRun(t, "grok", clipkgs.VerbDisable, clipkgs.Paths{}, clipkgs.Target{Name: "probe", Scope: "user"}))
	sameCommand(t, got, engineCommand(t, "grok", clipkgs.VerbDisable, clipkgs.Paths{}, clipkgs.Target{Name: "probe", Scope: "user"}))

	// Enabling the same plugin is the vendor's other verb, and its command is
	// the one the pane previews.
	stubGuest(t, "grok", `exit 0`)
	got, err = DriverFor("grok").Toggle(ctx, q, Target{Name: "probe", On: true})
	if err != nil {
		t.Fatal(err)
	}
	sameCommand(t, got, engineCommand(t, "grok", clipkgs.VerbEnable, clipkgs.Paths{}, clipkgs.Target{Name: "probe", Scope: "user", On: true}))

	// The one toggle a CLI only offers as a write of its own file: no command to
	// copy, and the write still happens where the engine does it.
	cfg := filepath.Join(dir, "opencode")
	t.Setenv("XDG_CONFIG_HOME", dir)
	mustWrite(t, filepath.Join(cfg, "opencode.json"), `{"plugin":["a","b"]}`)
	cmd, err := DriverFor("opencode").Toggle(ctx, Query{Vendor: "user"}, Target{Name: "a", On: false})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Exe != "" || cmd.Line != "" || len(cmd.Args) != 0 {
		t.Fatalf("a file write has no command to copy: %+v", cmd)
	}
	if body, err := os.ReadFile(filepath.Join(cfg, "opencode.json")); err != nil || strings.Contains(string(body), `"a"`) {
		t.Fatalf("the write did not happen: %s (%v)", body, err)
	}
}

// The toggle of an Omp extension is the CLI's own `disabledExtensions` write,
// asked of the engine before the plugin verbs — which do not know a configured
// extension. The workspace layer is a file PiCode splices and answers no
// command at all (the empty answer OpenCode's in-process toggle gives); the
// user layer is the CLI's own `config set`, run here because a toggle is
// synchronous, and the line the pane would copy is the one it ran.
func TestGuestToggleOmpExtensionWritesTheCLIsOwnList(t *testing.T) {
	ctx := context.Background()
	stubGuest(t, "omp", `case "$*" in
  *"config get disabledExtensions"*) printf '%s' '{"key":"disabledExtensions","value":[],"type":"array","description":""}' ;;
  *"config get extensions"*) printf '%s' '{"key":"extensions","value":["pkgs/one/tool.ts"],"type":"array","description":""}' ;;
  *) printf '%s' '{"npm":[],"marketplace":[]}' ;;
esac`)

	// The workspace layer: PiCode writes the settings file itself, so there is
	// no argv to hand the lane and nothing for a person to copy.
	ws := t.TempDir()
	settings := filepath.Join(ws, ".omp", "settings.json")
	mustWrite(t, settings, `{"extensions":["pkgs/ext/tool.ts"]}`)
	cmd, err := DriverFor("omp").Toggle(ctx, Query{Vendor: "project", WorkspacePath: ws},
		Target{Name: "tool", Source: "pkgs/ext/tool.ts", On: false})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Exe != "" || cmd.Line != "" || len(cmd.Args) != 0 {
		t.Fatalf("a workspace write has no command to copy: %+v", cmd)
	}
	body, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"disabledExtensions"`) || !strings.Contains(string(body), "extension-module:tool") {
		t.Fatalf("the toggle did not reach the CLI's own list: %s", body)
	}

	// The user layer: the CLI's own command, in the user's own directory, and
	// it is run — against the stub on PATH and a temp HOME, never the real one.
	home := t.TempDir()
	t.Setenv("HOME", home)
	cmd, err = DriverFor("omp").Toggle(ctx, Query{Vendor: "user"},
		Target{Name: "tool", Source: "pkgs/one/tool.ts", On: false})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"config", "set", "disabledExtensions", `["extension-module:tool"]`}
	if cmd.Exe != "omp" || cmd.Dir != home || !slices.Equal(cmd.Args, want) {
		t.Fatalf("command = %+v, want %s %v in %s", cmd, "omp", want, home)
	}
	if !strings.Contains(cmd.Line, "config set disabledExtensions") {
		t.Fatalf("line = %q, want the CLI's own command", cmd.Line)
	}

	// A plugin row is untouched by any of this: the vendor's own verb, as
	// before.
	cmd, err = DriverFor("omp").Toggle(ctx, Query{Vendor: "user"}, Target{Name: "probe@picode-probe-mp", On: false})
	if err != nil {
		t.Fatal(err)
	}
	sameCommand(t, cmd, engineCommand(t, "omp", clipkgs.VerbDisable, clipkgs.Paths{}, clipkgs.Target{
		Name: "probe@picode-probe-mp", Scope: "user",
	}))
}

// Inspect answers the vendor's own text and the command that produced it — the
// two things a refusal and a dialog need — and a CLI that declares no
// inspection refuses in its engine's own words.
func TestGuestInspectMatchesTheEngine(t *testing.T) {
	ctx := context.Background()
	stubGuest(t, "muse", `printf '%s' '{"capabilities":["read"]}'`)
	ws := t.TempDir()

	cmd, out, err := DriverFor("muse").Inspect(ctx, Query{Vendor: "user", WorkspacePath: ws}, Target{Name: "pl"})
	if err != nil {
		t.Fatal(err)
	}
	engineOut, err := clipkgs.Inspect(ctx, "muse", clipkgs.Paths{Cwd: ws}, clipkgs.Target{Name: "pl", Scope: "user"})
	if err != nil {
		t.Fatal(err)
	}
	if out != engineOut {
		t.Fatalf("inspect output = %q, want %q", out, engineOut)
	}
	sameCommand(t, cmd, engineCommand(t, "muse", clipkgs.VerbInspect, clipkgs.Paths{Cwd: ws}, clipkgs.Target{Name: "pl", Scope: "user"}))

	// A CLI that declares no inspection refuses where the engine refuses, in the
	// engine's own words — the sentence this route has always answered.
	_, _, err = DriverFor("opencode").Inspect(ctx, Query{}, Target{Name: "x"})
	_, engineErr := clipkgs.Inspect(ctx, "opencode", clipkgs.Paths{}, clipkgs.Target{Name: "x", Scope: "user"})
	sameRefusal(t, err, engineErr)
	if !errors.Is(err, clipkgs.ErrVerbAbsent) {
		t.Fatalf("err = %v, want the vendor's own refusal", err)
	}
}

// Every verb a CLI does not declare refuses with the engine's words: the pane
// reads the same sentence, and the route answers the same status, as when the
// handlers asked clipkgs themselves.
func TestGuestMutationRefusalsKeepTheEnginesWords(t *testing.T) {
	ctx := context.Background()

	// OpenCode removes by rewriting its own config file, so its removal is not
	// a command the job lane can carry and not a refusal either: the driver
	// performs the engine's own write and answers the empty command the route
	// reads as "there is nothing to run". This row used to pin the refusal
	// (*"this CLI does not expose that operation: opencode remove"*) that the
	// pane drew as an error under a live button.
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(home, "opencode", "opencode.json")
	if err := os.WriteFile(file, []byte(`{"plugin": ["a"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd, err := DriverFor("opencode").Remove(ctx, Query{Vendor: "user"}, Target{Name: "a", Source: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Exe != "" || cmd.Line != "" || len(cmd.Args) != 0 {
		t.Fatalf("a write has no command to hand a lane: %+v", cmd)
	}
	if got, err := os.ReadFile(file); err != nil || string(got) != `{"plugin": []}` {
		t.Fatalf("the engine did not perform the removal: %q (%v)", got, err)
	}
	if _, _, err := clipkgs.Argv("opencode", clipkgs.VerbRemove, clipkgs.Paths{}, clipkgs.Target{Name: "a", Scope: "user"}); !errors.Is(err, clipkgs.ErrVerbAbsent) {
		t.Fatalf("the CLI still exposes no removal command: %v", err)
	}

	// A CLI with no update verb, and one with no marketplace at all.
	_, err = DriverFor("agy").Update(ctx, Query{Vendor: "user"}, Target{Name: "a"})
	_, _, engineErr := clipkgs.Argv("agy", clipkgs.VerbUpdate, clipkgs.Paths{}, clipkgs.Target{Name: "a", Scope: "user"})
	sameRefusal(t, err, engineErr)

	_, err = DriverFor("hermes").Marketplace(ctx, Query{Vendor: "user"}, MarketRequest{Action: "add", Source: "owner/repo"})
	_, _, engineErr = clipkgs.MarketArgv("hermes", "add", clipkgs.Paths{}, clipkgs.MarketRequest{Action: "add", Source: "owner/repo"})
	sameRefusal(t, err, engineErr)

	// A CLI with no toggle: the verb's own refusal, where the run would be.
	_, err = DriverFor("codex").Toggle(ctx, Query{Vendor: "user"}, Target{Name: "pl", On: false})
	sameRefusal(t, err, engineRun(t, "codex", clipkgs.VerbDisable, clipkgs.Paths{}, clipkgs.Target{Name: "pl", Scope: "user", On: false}))
}

// The two answers a mutation adds are derived, not re-invented: the removal's
// remaining sources marshal to the bytes the one-key map answered, and the
// vendor's inspection text rides the two keys the pane parses.
func TestGuestMutationAnswersMatchTheEngineByteForByte(t *testing.T) {
	stubGuest(t, "grok", grokScript)
	ws := t.TempDir()
	ctx := context.Background()

	engine, err := clipkgs.Marketplaces(ctx, "grok", clipkgs.Paths{Cwd: ws})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := DriverFor("grok").Marketplaces(ctx, Query{Vendor: "user", WorkspacePath: ws})
	if err != nil {
		t.Fatal(err)
	}
	sameBytes(t, GuestMarketplaceSources(rows), map[string]any{"marketplaces": engine})

	// The toggle's answer is the CLI's list after the mutation, mapped the way
	// slice 2 maps every roster: the driver's report and the engine's report are
	// the same bytes (the read stamps are microseconds apart and are held
	// equal, as the read's own test holds them).
	stubGuest(t, "grok", grokScript)
	engineRep, err := clipkgs.List(ctx, "grok", clipkgs.Paths{Cwd: ws}, "user", true)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := DriverFor("grok").List(ctx, Query{Vendor: "user", WorkspacePath: ws, Fresh: true})
	if err != nil {
		t.Fatal(err)
	}
	engineRep.ReadAt = rep.ReadAt
	sameBytes(t, Guest("grok", rep), GuestViewOf("grok", engineRep))

	// Inspect is the vendor's own text under the pane's two keys.
	out, err := clipkgs.Inspect(ctx, "grok", clipkgs.Paths{Cwd: ws}, clipkgs.Target{Name: "probe", Scope: "user"})
	if err != nil {
		t.Fatal(err)
	}
	sameBytes(t, GuestInspect("grok", out), map[string]any{"cli": "grok", "output": out})
}

// The pane parses these two payloads, so they are pinned as bytes.
func TestGuestMutationViewsAreThePanesPayload(t *testing.T) {
	b, err := json.Marshal(GuestMarketplaceSources([]Row{
		{CLI: "grok", ID: "team", Name: "team", Source: "https://example.test/mp", Kind: "marketplace", Scope: Machine, Vendor: "user", Enabled: true, Installed: true, Status: "1.2.3"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"marketplaces":[{"id":"team","name":"team","scope":"user","enabled":true,"installed":true,` +
		`"source":"https://example.test/mp","sourceKind":"marketplace","status":"1.2.3"}]}`
	if string(b) != want {
		t.Fatalf("payload = %s\n      want %s", b, want)
	}

	// No sources left is an empty list, never null: the pane's empty state.
	b, err = json.Marshal(GuestMarketplaceSources(nil))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"marketplaces":[]}` {
		t.Fatalf("empty payload = %s", b)
	}

	b, err = json.Marshal(GuestInspect("muse", `{"capabilities":["read"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"cli":"muse","output":"{\"capabilities\":[\"read\"]}"}` {
		t.Fatalf("inspect payload = %s", b)
	}
}
