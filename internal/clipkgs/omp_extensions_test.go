package clipkgs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Omp's own extension list is not its plugin roster: the CLI loads these from
// its settings (`<ws>/.omp/settings.json` for a project, its own config command
// for the user layer), so the pane that shows `omp plugin list --json` alone
// leaves every configured extension invisible. These tests pin the reader, the
// row it produces, the merge with the vendor's roster, and the removal the
// driver runs for each layer.

// writeOmpSettings writes one workspace settings file and answers its
// directory, which is the Paths.Cwd the read resolves the file from.
func writeOmpSettings(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	full := filepath.Join(dir, ".omp", "settings.json")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// stubOmpConfig puts an `omp` on PATH that answers `config get` by the key it
// was asked for — the reader asks once per key — and prints nothing else.
func stubOmpConfig(t *testing.T, extensionsOut, disabledOut string) {
	t.Helper()
	dir := t.TempDir()
	script := fmt.Sprintf(`#!/bin/sh
case "$*" in
  *"config get disabledExtensions"*) cat <<'OUT'
%s
OUT
    ;;
  *"config get extensions"*) cat <<'OUT'
%s
OUT
    ;;
esac
`, disabledOut, extensionsOut)
	if err := os.WriteFile(filepath.Join(dir, "omp"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	Invalidate("omp")
}

// TestOmpProjectExtensionsReadsTheWorkspaceFile pins the four states of the
// workspace file the reader has to tell apart: a declared list, no file at all,
// an empty list, and a file it cannot read — which is an error naming it, never
// an empty list (ADR-0150's rule for a malformed user file).
func TestOmpProjectExtensionsReadsTheWorkspaceFile(t *testing.T) {
	dir := writeOmpSettings(t, `{
  "extensions": ["packages/pi-browser/extensions/browser.ts", "pkgs/other/tool.ts"],
  "disabledExtensions": ["extension-module:browser"],
  "theme": {"keep": true}
}`)
	exts, disabled, err := ompProjectExtensions(Paths{Cwd: dir})
	if err != nil {
		t.Fatalf("ompProjectExtensions: %v", err)
	}
	if len(exts) != 2 || exts[0] != "packages/pi-browser/extensions/browser.ts" || exts[1] != "pkgs/other/tool.ts" {
		t.Errorf("extensions = %v", exts)
	}
	if len(disabled) != 1 || disabled[0] != "extension-module:browser" {
		t.Errorf("disabledExtensions = %v", disabled)
	}

	// No workspace: there is no project layer to read.
	if exts, disabled, err := ompProjectExtensions(Paths{}); err != nil || exts != nil || disabled != nil {
		t.Errorf("no workspace = %v / %v / %v, want nothing read", exts, disabled, err)
	}

	// No file: no extensions, and not an error.
	if exts, _, err := ompProjectExtensions(Paths{Cwd: t.TempDir()}); err != nil || len(exts) != 0 {
		t.Errorf("absent file = %v / %v, want none", exts, err)
	}

	// An empty list stays empty rather than an error.
	exts, disabled, err = ompProjectExtensions(Paths{Cwd: writeOmpSettings(t, `{"extensions": []}`)})
	if err != nil || len(exts) != 0 || len(disabled) != 0 {
		t.Errorf("empty list = %v / %v / %v", exts, disabled, err)
	}

	// A file that does not parse refuses by name.
	dir = writeOmpSettings(t, `{"extensions": [`)
	_, _, err = ompProjectExtensions(Paths{Cwd: dir})
	if err == nil || !strings.Contains(err.Error(), filepath.Join(dir, ".omp", "settings.json")) {
		t.Errorf("malformed file = %v, want the file named", err)
	}

	// A value that is not a list of strings is refused, not skipped: a row
	// PiCode cannot read is a row the CLI never loads.
	_, _, err = ompProjectExtensions(Paths{Cwd: writeOmpSettings(t, `{"extensions": "pkgs/other/tool.ts"}`)})
	if err == nil || !strings.Contains(err.Error(), "extensions is not a list of strings") {
		t.Errorf("non-list = %v", err)
	}
}

// TestOmpUserExtensionsFixture pins the CLI's own answer shape, taken verbatim
// from `omp config get <key> --json` on 18.2.8: the keys `key`, `value`, `type`
// and `description`, with the list in `value`.
func TestOmpUserExtensionsFixture(t *testing.T) {
	exts, err := parseOmpConfigArray(fixture(t, "omp.extensions.json"))
	if err != nil {
		t.Fatalf("parseOmpConfigArray: %v", err)
	}
	want := []string{"packages/pi-browser/extensions/browser.ts", "pkgs/other-ext/tool.ts"}
	if len(exts) != len(want) {
		t.Fatalf("extensions = %v, want %v", exts, want)
	}
	for i := range want {
		if exts[i] != want[i] {
			t.Errorf("extensions[%d] = %q, want %q", i, exts[i], want[i])
		}
	}
	disabled, err := parseOmpConfigArray(fixture(t, "omp.extensions.disabled.json"))
	if err != nil || len(disabled) != 1 || disabled[0] != "extension-module:browser" {
		t.Errorf("disabled = %v / %v", disabled, err)
	}

	// The vendor's own empty answer is an empty list, not a failure.
	if empty, err := parseOmpConfigArray("{\n  \"key\": \"extensions\",\n  \"value\": [],\n  \"type\": \"array\",\n  \"description\": \"\"\n}"); err != nil || len(empty) != 0 {
		t.Errorf("empty answer = %v / %v", empty, err)
	}

	// A shape PiCode does not recognize is ErrRosterShape, never "no
	// extensions": the pane would otherwise show a list that is not the CLI's.
	for _, out := range []string{"", "no extensions configured", "[{\"value\":1}]", "{\"value\":\"browser.ts\"}"} {
		if _, err := parseOmpConfigArray(out); err == nil || !errors.Is(err, ErrRosterShape) {
			t.Errorf("parseOmpConfigArray(%q) = %v, want ErrRosterShape", out, err)
		}
	}
}

// TestOmpExtensionRowsCarryTheirLayer pins the row one configured entry
// becomes: the entry as written is its source and id, the CLI's own name for it
// is the base without the extension, the scope is the layer it came from, the
// path is set only when something is there, and `disabledExtensions` decides
// whether the CLI loads it.
func TestOmpExtensionRowsCarryTheirLayer(t *testing.T) {
	dir := writeOmpSettings(t, `{}`)
	if err := os.MkdirAll(filepath.Join(dir, "packages", "pi-browser", "extensions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "packages", "pi-browser", "extensions", "browser.ts"), []byte("export default {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rows := ompExtensionRows([]string{
		"packages/pi-browser/extensions/browser.ts",
		"pkgs/other-ext/index.ts",
		"/nowhere/absent.ts",
	}, []string{"extension-module:browser"}, "project", dir)
	if len(rows) != 3 {
		t.Fatalf("rows = %+v, want one per entry", rows)
	}
	first := rows[0]
	if first.ID != "packages/pi-browser/extensions/browser.ts" || first.Source != first.ID {
		t.Errorf("row = %+v, want the entry as written as its identity", first)
	}
	if first.Name != "browser" || first.SourceKind != "extension" || first.Scope != "project" {
		t.Errorf("row = %+v, want the CLI's own name and the layer", first)
	}
	if !first.Installed || first.Enabled {
		t.Errorf("row = %+v, want installed and disabled", first)
	}
	if first.InstallPath != filepath.Join(dir, "packages", "pi-browser", "extensions", "browser.ts") {
		t.Errorf("install path = %q, want the resolved path", first.InstallPath)
	}
	if first.Note == "" || !strings.Contains(first.Note, ".omp/settings.json") {
		t.Errorf("note = %q, want the file the row comes from", first.Note)
	}
	// An `index.ts` is named for its directory, exactly as the CLI names it.
	if rows[1].Name != "other-ext" {
		t.Errorf("name = %q, want the directory of an index file", rows[1].Name)
	}
	// A path that is not there is not claimed.
	if rows[2].InstallPath != "" {
		t.Errorf("install path = %q, want empty for a path that does not exist", rows[2].InstallPath)
	}
	if rows[2].Enabled != true {
		t.Errorf("row = %+v, want enabled when nothing disables it", rows[2])
	}

	user := ompExtensionRows([]string{"/opt/exts/thing.ts"}, nil, "user", t.TempDir())
	if len(user) != 1 || user[0].Scope != "user" || !strings.Contains(user[0].Note, "user settings") {
		t.Errorf("user row = %+v, want the user layer", user)
	}
}

// TestOmpExtensionsMergesBothLayers is the reader's decision table: the
// workspace file decides the workspace rows, the CLI's own answer decides the
// user rows, and one entry named by both is one row.
func TestOmpExtensionsMergesBothLayers(t *testing.T) {
	// Measured: inside a project that declares `extensions`, the CLI's own
	// `config get` answers that project's array — the user's is replaced, not
	// merged — so the stub answers the same array the file declares.
	stubOmpConfig(t,
		`{"key":"extensions","value":["packages/pi-browser/extensions/browser.ts"],"type":"array","description":""}`,
		`{"key":"disabledExtensions","value":["extension-module:browser"],"type":"array","description":""}`)
	dir := writeOmpSettings(t, `{
  "extensions": ["packages/pi-browser/extensions/browser.ts"],
  "disabledExtensions": ["extension-module:browser"]
}`)

	rows, note, err := ompExtensions(context.Background(), Paths{Home: t.TempDir(), Cwd: dir})
	if err != nil {
		t.Fatalf("ompExtensions: %v", err)
	}
	if note != "" {
		t.Errorf("note = %q, want none", note)
	}
	// One entry named by both layers is one row, and it is the workspace's: the
	// layer PiCode reads and writes, and the layer the CLI's own answer for
	// this directory reports.
	if len(rows) != 1 {
		t.Fatalf("rows = %+v, want the workspace row alone", rows)
	}
	if rows[0].Scope != "project" || rows[0].Enabled {
		t.Errorf("row = %+v, want the workspace row, disabled", rows[0])
	}

	// No workspace: only the CLI's own answer is read, and it is the user's.
	stubOmpConfig(t, fixture(t, "omp.extensions.json"), fixture(t, "omp.extensions.disabled.json"))
	rows, _, err = ompExtensions(context.Background(), Paths{Home: t.TempDir()})
	if err != nil {
		t.Fatalf("ompExtensions without a workspace: %v", err)
	}
	if len(rows) != 2 || rows[0].Scope != "user" || rows[1].Scope != "user" {
		t.Fatalf("rows = %+v, want the user layer's two entries", rows)
	}
	if rows[0].Enabled {
		t.Errorf("row = %+v, want the entry `disabledExtensions` names turned off", rows[0])
	}
	if rows[1].InstallPath != "" {
		t.Errorf("row = %+v, want no install path for an entry that is not there", rows[1])
	}
}

// TestOmpExtensionsReportsAnUnreadableVendorLayer: the plugin roster is what
// the caller came for, so a user layer the CLI cannot answer is a note on the
// read, not a failed list — the entries the workspace file declares still show.
func TestOmpExtensionsReportsAnUnreadableVendorLayer(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "omp"), []byte("#!/bin/sh\necho 'the config store is locked' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	Invalidate("omp")

	ws := writeOmpSettings(t, `{"extensions": ["pkgs/other/tool.ts"]}`)
	rows, note, err := ompExtensions(context.Background(), Paths{Home: t.TempDir(), Cwd: ws})
	if err != nil {
		t.Fatalf("ompExtensions: %v", err)
	}
	if len(rows) != 1 || rows[0].Scope != "project" {
		t.Errorf("rows = %+v, want the workspace row the file declares", rows)
	}
	if !strings.Contains(note, "user extensions could not be read") || !strings.Contains(note, "the config store is locked") {
		t.Errorf("note = %q, want the vendor's own words", note)
	}

	// A malformed workspace file is not the same fact: it is this workspace's
	// store, and it refuses the whole read by name.
	bad := writeOmpSettings(t, `{"extensions": [`)
	if _, _, err := ompExtensions(context.Background(), Paths{Home: t.TempDir(), Cwd: bad}); err == nil || !errors.Is(err, ErrRosterShape) {
		t.Errorf("malformed workspace file = %v, want ErrRosterShape", err)
	}
}

// TestMergeOmpExtensionsSkipsRosterRows: the same package read through the
// plugin roster is one row, and the vendor's own row — the one that carries the
// plugin verbs — is the one that stays.
func TestMergeOmpExtensionsSkipsRosterRows(t *testing.T) {
	roster, _, err := parseVendorRows("omp", fixture(t, "omp.list.json"), false)
	if err != nil {
		t.Fatalf("parseVendorRows: %v", err)
	}
	listed := roster[0]
	overlap := Row{
		ID: listed.InstallPath, Name: ompExtensionName(listed.InstallPath), Source: listed.InstallPath,
		SourceKind: "extension", Scope: "user", Enabled: true, Installed: true, InstallPath: listed.InstallPath,
	}
	fresh := Row{ID: "pkgs/other/tool.ts", Name: "tool", Source: "pkgs/other/tool.ts", SourceKind: "extension", Scope: "user", Enabled: true, Installed: true}

	merged := mergeOmpExtensions(roster, []Row{overlap, fresh})
	if len(merged) != len(roster)+1 {
		t.Fatalf("merged = %d rows, want the roster plus the one entry it does not carry", len(merged))
	}
	for _, r := range merged {
		if r.ID == overlap.ID && r.SourceKind == "extension" {
			t.Errorf("the roster's own row was duplicated: %+v", r)
		}
	}
	// No extensions: the roster comes back untouched.
	if got := mergeOmpExtensions(roster, nil); len(got) != len(roster) {
		t.Errorf("merged = %d rows, want the roster alone", len(got))
	}
}

// TestOmpExtensionRemoveWritesTheWorkspaceFile: the workspace layer is a file
// the CLI has no verb for, so PiCode writes it itself — the entry and its
// disabled id leave, and everything else in the file survives.
func TestOmpExtensionRemoveWritesTheWorkspaceFile(t *testing.T) {
	// The user layer is empty on this machine: what the test is about is the
	// workspace file, and no vendor call may decide the outcome.
	stubOmpConfig(t,
		`{"key":"extensions","value":[],"type":"array","description":""}`,
		`{"key":"disabledExtensions","value":[],"type":"array","description":""}`)
	dir := writeOmpSettings(t, `{
  "extensions": ["packages/pi-browser/extensions/browser.ts", "pkgs/other/tool.ts"],
  "disabledExtensions": ["extension-module:browser"],
  "theme": {"dark": "titanium"}
}`)
	rm, ok, err := OmpExtensionRemove(context.Background(), Paths{Cwd: dir},
		Target{Name: "browser", Source: "packages/pi-browser/extensions/browser.ts"})
	if err != nil || !ok {
		t.Fatalf("OmpExtensionRemove = %+v / %v / %v", rm, ok, err)
	}
	if !rm.InProcess {
		t.Errorf("removal = %+v, want the engine's own write", rm)
	}
	// No argv: the CLI has no command for this layer.
	if rm.Args != nil || rm.Dir != "" || rm.Line != "" {
		t.Errorf("removal = %+v, want no vendor command", rm)
	}

	b, err := os.ReadFile(filepath.Join(dir, ".omp", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Extensions []string `json:"extensions"`
		Disabled   []string `json:"disabledExtensions"`
		Theme      map[string]string
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("the written file does not parse: %v\n%s", err, b)
	}
	if len(doc.Extensions) != 1 || doc.Extensions[0] != "pkgs/other/tool.ts" {
		t.Errorf("extensions = %v, want the other entry", doc.Extensions)
	}
	if len(doc.Disabled) != 0 {
		t.Errorf("disabledExtensions = %v, want the removed entry's id gone too", doc.Disabled)
	}
	if doc.Theme["dark"] != "titanium" {
		t.Errorf("theme = %v, want every other key untouched", doc.Theme)
	}
	// The edit is surgical: the untouched line is still the line that was there.
	if !strings.Contains(string(b), `"theme": {"dark": "titanium"}`) {
		t.Errorf("the write reformatted the file:\n%s", b)
	}

	// A target neither layer names is not an extension: the driver falls back
	// to the plugin verb.
	if _, ok, err := OmpExtensionRemove(context.Background(), Paths{Cwd: dir}, Target{Name: "probe@picode-probe-mp"}); ok || err != nil {
		t.Errorf("a plugin was read as an extension: %v / %v", ok, err)
	}
}

// stubOmpConfigRecords is stubOmpConfig plus a record of every invocation, so
// "the driver ran the command" is a fact about the vendor call and not just
// about the line PiCode rendered for it.
func stubOmpConfigRecords(t *testing.T, extensionsOut, disabledOut string) string {
	t.Helper()
	dir := t.TempDir()
	record := filepath.Join(dir, "calls")
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
case "$*" in
  *"config get disabledExtensions"*) cat <<'OUT'
%s
OUT
    ;;
  *"config get extensions"*) cat <<'OUT'
%s
OUT
    ;;
esac
`, record, disabledOut, extensionsOut)
	if err := os.WriteFile(filepath.Join(dir, "omp"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	Invalidate("omp")
	return record
}

// ompSettingsDoc reads one workspace settings file back as the keys a toggle is
// allowed to touch.
func ompSettingsDoc(t *testing.T, file string) (raw []byte, extensions, disabled []string, rest map[string]any) {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("the written file does not parse: %v\n%s", err, b)
	}
	for _, k := range []string{"extensions", "disabledExtensions"} {
		var list []string
		if raw, ok := doc[k]; ok {
			if err := json.Unmarshal(raw, &list); err != nil {
				t.Fatalf("%s = %s, want a list of strings", k, raw)
			}
		}
		if k == "extensions" {
			extensions = list
		} else {
			disabled = list
		}
		delete(doc, k)
	}
	rest = map[string]any{}
	for k, v := range doc {
		var any1 any
		if err := json.Unmarshal(v, &any1); err != nil {
			t.Fatalf("%s does not parse: %v", k, err)
		}
		rest[k] = any1
	}
	return b, extensions, disabled, rest
}

// TestOmpExtensionToggleWritesTheWorkspaceFile pins the workspace layer of a
// toggle: the CLI's own id joins `disabledExtensions` when the row goes off and
// leaves it when the row goes on, nothing else in the file moves — the entry
// list and every other key stay byte for byte — and a file that already says
// what the row asks for is not rewritten at all.
func TestOmpExtensionToggleWritesTheWorkspaceFile(t *testing.T) {
	// The user layer is empty on this machine: what the test is about is the
	// workspace file, and no vendor call may decide the outcome.
	stubOmpConfig(t,
		`{"key":"extensions","value":[],"type":"array","description":""}`,
		`{"key":"disabledExtensions","value":[],"type":"array","description":""}`)
	dir := writeOmpSettings(t, `{
  "extensions": ["pkgs/one/tool.ts", "pkgs/two/other.ts"],
  "disabledExtensions": ["extension-module:other"],
  "theme": {"dark": "titanium"}
}`)
	file := filepath.Join(dir, ".omp", "settings.json")
	p := Paths{Home: t.TempDir(), Cwd: dir}
	ctx := context.Background()

	// Off: the id joins the array — after the entry that is already in it —
	// and there is no vendor command for this layer.
	tog, ok, err := OmpExtensionToggle(ctx, p, Target{Name: "tool", Source: "pkgs/one/tool.ts", On: false})
	if err != nil || !ok {
		t.Fatalf("OmpExtensionToggle = %+v / %v / %v", tog, ok, err)
	}
	if !tog.InProcess || tog.Args != nil || tog.Dir != "" || tog.Line != "" {
		t.Errorf("toggle = %+v, want the engine's own write with no vendor command", tog)
	}
	raw, extensions, disabled, rest := ompSettingsDoc(t, file)
	if strings.Join(disabled, "|") != "extension-module:other|extension-module:tool" {
		t.Errorf("disabledExtensions = %v, want the id added and the other entry untouched", disabled)
	}
	if strings.Join(extensions, "|") != "pkgs/one/tool.ts|pkgs/two/other.ts" {
		t.Errorf("extensions = %v, want the entries untouched", extensions)
	}
	if rest["theme"] == nil {
		t.Errorf("the unrelated key left the file: %v", rest)
	}
	// The edit is surgical: the untouched lines are still the lines that were
	// there.
	if !strings.Contains(string(raw), `"theme": {"dark": "titanium"}`) {
		t.Errorf("the write reformatted the file:\n%s", raw)
	}
	off := raw

	// Off again: the file already says it, so nothing is written — a disable
	// must not name the id twice.
	tog, ok, err = OmpExtensionToggle(ctx, p, Target{Name: "tool", On: false})
	if err != nil || !ok || !tog.InProcess {
		t.Fatalf("a second disable = %+v / %v / %v", tog, ok, err)
	}
	if again, _, _, _ := ompSettingsDoc(t, file); !bytes.Equal(again, off) {
		t.Errorf("a second disable rewrote the file:\n%s", again)
	}

	// On: the id leaves the array, and the entry that is still off stays named.
	tog, ok, err = OmpExtensionToggle(ctx, p, Target{Name: "tool", On: true})
	if err != nil || !ok || !tog.InProcess {
		t.Fatalf("OmpExtensionToggle on = %+v / %v / %v", tog, ok, err)
	}
	on, _, disabled, _ := ompSettingsDoc(t, file)
	if len(disabled) != 1 || disabled[0] != "extension-module:other" {
		t.Errorf("disabledExtensions = %v, want only the other entry left", disabled)
	}
	if !strings.Contains(string(on), `"theme": {"dark": "titanium"}`) {
		t.Errorf("the write reformatted the file:\n%s", on)
	}

	// On again: nothing to cut, nothing written.
	if _, ok, err := OmpExtensionToggle(ctx, p, Target{Name: "tool", Source: "pkgs/one/tool.ts", On: true}); err != nil || !ok {
		t.Fatalf("a second enable = %v / %v", ok, err)
	}
	if again, _, _, _ := ompSettingsDoc(t, file); !bytes.Equal(again, on) {
		t.Errorf("a second enable rewrote the file:\n%s", again)
	}

	// A target neither layer names is not an extension: the driver falls back
	// to the plugin verb.
	if _, ok, err := OmpExtensionToggle(ctx, p, Target{Name: "probe@picode-probe-mp", On: false}); ok || err != nil {
		t.Errorf("a plugin was read as an extension: %v / %v", ok, err)
	}
}

// TestOmpExtensionToggleCreatesTheDisabledList: a workspace file whose first
// extension is being turned off has no `disabledExtensions` yet, so the key
// arrives with the id — beside the keys the file already has, without touching
// them.
func TestOmpExtensionToggleCreatesTheDisabledList(t *testing.T) {
	stubOmpConfig(t,
		`{"key":"extensions","value":[],"type":"array","description":""}`,
		`{"key":"disabledExtensions","value":[],"type":"array","description":""}`)
	dir := writeOmpSettings(t, `{
  "extensions": ["pkgs/ext/tool.ts"],
  "theme": {"keep": true}
}`)

	tog, ok, err := OmpExtensionToggle(context.Background(), Paths{Cwd: dir},
		Target{Name: "tool", Source: "pkgs/ext/tool.ts", On: false})
	if err != nil || !ok || !tog.InProcess {
		t.Fatalf("OmpExtensionToggle = %+v / %v / %v", tog, ok, err)
	}
	raw, extensions, disabled, rest := ompSettingsDoc(t, filepath.Join(dir, ".omp", "settings.json"))
	if len(disabled) != 1 || disabled[0] != "extension-module:tool" {
		t.Errorf("disabledExtensions = %v, want the created list", disabled)
	}
	if len(extensions) != 1 || extensions[0] != "pkgs/ext/tool.ts" {
		t.Errorf("extensions = %v, want the declared entry untouched", extensions)
	}
	if rest["theme"] == nil {
		t.Errorf("the unrelated key left the file: %v", rest)
	}
	if !strings.Contains(string(raw), `"theme": {"keep": true}`) {
		t.Errorf("the write reformatted the file:\n%s", raw)
	}
}

// TestOmpExtensionToggleRunsTheVendorCommandForTheUserLayer: the user layer is
// the CLI's own file, so the toggle is the CLI's own command — the same array
// with the id in or out, run where the CLI writes it. The CLI is a stub on PATH
// and the directory a temp one, so nothing is executed against a real HOME.
func TestOmpExtensionToggleRunsTheVendorCommandForTheUserLayer(t *testing.T) {
	record := stubOmpConfigRecords(t,
		`{"key":"extensions","value":["pkgs/one/tool.ts","pkgs/two/tool.ts"],"type":"array","description":""}`,
		`{"key":"disabledExtensions","value":["extension-module:two"],"type":"array","description":""}`)
	home := t.TempDir()
	ctx := context.Background()

	tog, ok, err := OmpExtensionToggle(ctx, Paths{Home: home}, Target{Name: "tool", Source: "pkgs/one/tool.ts", On: false})
	if err != nil || !ok {
		t.Fatalf("OmpExtensionToggle = %+v / %v / %v", tog, ok, err)
	}
	if tog.InProcess {
		t.Errorf("toggle = %+v, want the CLI's own command", tog)
	}
	wantArgs := []string{"config", "set", "disabledExtensions", `["extension-module:two","extension-module:tool"]`}
	if strings.Join(tog.Args, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Errorf("args = %v, want %v", tog.Args, wantArgs)
	}
	if tog.Dir != home {
		t.Errorf("dir = %q, want the user's own directory", tog.Dir)
	}
	wantLine := "cd " + home + " && omp config set disabledExtensions '[\"extension-module:two\",\"extension-module:tool\"]'"
	if tog.Line != wantLine {
		t.Errorf("line = %q, want %q", tog.Line, wantLine)
	}
	// It ran: the CLI got exactly that command, as its own argv.
	if got := string(mustRead(t, record)); !strings.Contains(got, "config set disabledExtensions "+wantArgs[3]) {
		t.Errorf("the CLI was not asked:\n%s", got)
	}

	// On: the same array without the id, and the entry that is still off stays.
	tog, ok, err = OmpExtensionToggle(ctx, Paths{Home: home}, Target{Name: "tool", On: true})
	if err != nil || !ok || tog.InProcess {
		t.Fatalf("OmpExtensionToggle on = %+v / %v / %v", tog, ok, err)
	}
	wantArgs = []string{"config", "set", "disabledExtensions", `["extension-module:two"]`}
	if strings.Join(tog.Args, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Errorf("args = %v, want %v", tog.Args, wantArgs)
	}
}

// mustRead reads a file a stub wrote, and is empty rather than fatal when the
// stub never ran — the caller's own assertion is the failure that names it.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

func TestOmpExtensionRemoveAnswersTheVendorCommandForTheUserLayer(t *testing.T) {
	stubOmpConfig(t, `{"key":"extensions","value":["pkgs/one/tool.ts","pkgs/two/tool.ts"],"type":"array","description":""}`, `{"key":"disabledExtensions","value":[],"type":"array","description":""}`)
	home := t.TempDir()

	rm, ok, err := OmpExtensionRemove(context.Background(), Paths{Home: home}, Target{Name: "tool", Source: "pkgs/one/tool.ts"})
	if err != nil || !ok {
		t.Fatalf("OmpExtensionRemove = %+v / %v / %v", rm, ok, err)
	}
	if rm.InProcess {
		t.Errorf("removal = %+v, want the CLI's own command", rm)
	}
	wantArgs := []string{"config", "set", "extensions", `["pkgs/two/tool.ts"]`}
	if strings.Join(rm.Args, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Errorf("args = %v, want %v", rm.Args, wantArgs)
	}
	if rm.Dir != home {
		t.Errorf("dir = %q, want the user's own directory", rm.Dir)
	}
	if want := "cd " + home + " && omp config set extensions '[\"pkgs/two/tool.ts\"]'"; rm.Line != want {
		t.Errorf("line = %q, want %q", rm.Line, want)
	}
}
