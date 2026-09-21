package clipkgs

// Fixture tests: each roster parser is pinned against output a real vendor
// binary printed on this machine (2026-09-20), captured verbatim into
// testdata/. They are what makes a parser a measured shape instead of a guess:
// a vendor release that moves the shape fails here and in the live suite
// (live_test.go), and never quietly reports an empty roster.
//
// Two fixtures are excerpts by necessity, both noted where they are used:
// testdata/hermes.available.json is the first five of the 222 catalog rows the
// vendor returned (the full answer is 214 KB), and testdata/claude-code.list.json
// is read-only output of the real HOME because that is the only place Claude
// Code's `synced` rows exist.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

func rowByID(t *testing.T, rows []Row, id string) Row {
	t.Helper()
	for _, r := range rows {
		if r.ID == id {
			return r
		}
	}
	got := make([]string, 0, len(rows))
	for _, r := range rows {
		got = append(got, r.ID)
	}
	t.Fatalf("no row %q; rows were %v", id, got)
	return Row{}
}

// TestClaudeRealRoster pins the array `claude plugin list --json` prints
// (testdata/claude-code.list.json, the machine's own install: two scoped
// rows of one plugin, plus two `synced` plugins pushed from claude.ai).
func TestClaudeRealRoster(t *testing.T) {
	out := fixture(t, "claude-code.list.json")

	user, note, err := parseClaude(out, "user", false)
	if err != nil {
		t.Fatalf("parseClaude(user): %v", err)
	}
	if note != "" {
		t.Errorf("note = %q, want empty", note)
	}
	if len(user) != 3 {
		t.Fatalf("user scope rows = %d, want 3 (user + synced): %v", len(user), user)
	}
	for _, r := range user {
		if r.Scope == "local" {
			t.Errorf("the local row leaked into the user scope view: %+v", r)
		}
		if !r.Installed {
			t.Errorf("row %s is not marked installed", r.ID)
		}
	}
	mine := rowByID(t, user, "rust-analyzer-lsp@claude-plugins-official")
	if mine.Scope != "user" || mine.Enabled || mine.Name != "rust-analyzer-lsp" {
		t.Errorf("user row = %+v, want scope user, disabled, name rust-analyzer-lsp", mine)
	}
	if mine.Marketplace != "claude-plugins-official" || mine.SourceKind != "marketplace" {
		t.Errorf("user row origin = %q/%q, want the marketplace in the id", mine.Marketplace, mine.SourceKind)
	}
	synced := rowByID(t, user, "cowork-plugin-management@synced")
	if synced.Scope != "synced" || !synced.Enabled || synced.Version != "0.2.2" {
		t.Errorf("synced row = %+v, want scope synced, enabled, version 0.2.2", synced)
	}
	if !strings.Contains(synced.Note, "claude.ai") {
		t.Errorf("synced row note = %q, want the claude.ai caveat", synced.Note)
	}
	if !strings.Contains(synced.InstallPath, filepath.Join("plugins", "synced")) {
		t.Errorf("synced install path = %q", synced.InstallPath)
	}

	local, _, err := parseClaude(out, "local", false)
	if err != nil {
		t.Fatalf("parseClaude(local): %v", err)
	}
	if len(local) != 1 || local[0].Scope != "local" {
		t.Fatalf("local scope rows = %+v, want exactly the local row", local)
	}
	if local[0].Name != "rust-analyzer-lsp" {
		t.Errorf("local row name = %q", local[0].Name)
	}

	project, _, err := parseClaude(out, "project", false)
	if err != nil {
		t.Fatalf("parseClaude(project): %v", err)
	}
	if len(project) != 0 {
		t.Errorf("project scope rows = %+v, want none", project)
	}
}

// TestClaudeMarketplaceEnvelope pins the envelope `claude plugin list --json
// --available` prints (measured, sandboxed): the installed half is the array
// shape above, the catalog half has its own fields and is not installed.
func TestClaudeMarketplaceEnvelope(t *testing.T) {
	out := fixture(t, "claude-code.available.json")

	rows, note, err := parseClaude(out, "user", true)
	if err != nil {
		t.Fatalf("parseClaude(available): %v", err)
	}
	if note != "" {
		t.Errorf("note = %q, want empty", note)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want the installed plugin and one catalog row: %+v", len(rows), rows)
	}
	installed := rowByID(t, rows, "probe@picode-probe-mp")
	if !installed.Installed || !installed.Enabled || installed.Version != "0.1.0" {
		t.Errorf("installed half = %+v, want installed and enabled", installed)
	}
	catalog := rowByID(t, rows, "probe2@picode-probe-mp")
	if catalog.Installed || catalog.Enabled {
		t.Errorf("catalog row = %+v, want neither installed nor enabled", catalog)
	}
	if catalog.Name != "probe2" || catalog.Version != "0.9.0" {
		t.Errorf("catalog row = %+v, want name probe2 at 0.9.0", catalog)
	}
	if catalog.Marketplace != "picode-probe-mp" || catalog.Source != "probe2@picode-probe-mp" {
		t.Errorf("catalog row origin = %q / %q, want the marketplace and an installable id", catalog.Marketplace, catalog.Source)
	}
	if catalog.Description != "second claude probe plugin" {
		t.Errorf("catalog description = %q", catalog.Description)
	}

	// The same output read as an installed roster must not grow a catalog.
	roster, _, err := parseClaude(out, "user", false)
	if err != nil {
		t.Fatalf("parseClaude(installed): %v", err)
	}
	if len(roster) != 1 || roster[0].ID != "probe@picode-probe-mp" {
		t.Errorf("installed read = %+v, want only the installed row", roster)
	}
}

// TestCodexRealRoster pins `codex plugin list --json` (sandboxed: one plugin
// installed from a local marketplace, whose `source` is an object).
func TestCodexRealRoster(t *testing.T) {
	rows, note, err := parseCodex(fixture(t, "codex.list.json"), false)
	if err != nil {
		t.Fatalf("parseCodex: %v", err)
	}
	// The envelope carries an empty `available` key even here: the marketplace
	// note belongs to the --available read, not to every list read.
	if note != "" {
		t.Errorf("note = %q, want empty for an installed-roster read", note)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1: %+v", len(rows), rows)
	}
	row := rows[0]
	if row.ID != "probe@picode-probe-mp" || row.Name != "probe" || row.Version != "0.3.0" {
		t.Errorf("row = %+v", row)
	}
	if !row.Installed || !row.Enabled || row.Marketplace != "picode-probe-mp" {
		t.Errorf("row = %+v, want installed and enabled from its marketplace", row)
	}
	if row.Source != "/tmp/clipkgs-bundles/codex-mp2/plugins/probe" || row.SourceKind != "local" {
		t.Errorf("source = %q/%q, want the path with the vendor's kind", row.Source, row.SourceKind)
	}
	if !strings.Contains(row.Status, "AVAILABLE") || !strings.Contains(row.Status, "ON_INSTALL") {
		t.Errorf("status = %q, want the install and auth policies", row.Status)
	}
}

func TestCodexMarketplaceEnvelope(t *testing.T) {
	rows, note, err := parseCodex(fixture(t, "codex.available.json"), true)
	if err != nil {
		t.Fatalf("parseCodex(available): %v", err)
	}
	if note == "" {
		t.Error("the marketplace read carries no note about its network dependency")
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want the installed plugin and one catalog row: %+v", len(rows), rows)
	}
	if installed := rowByID(t, rows, "probe@picode-probe-mp"); !installed.Installed {
		t.Errorf("installed row = %+v, want installed", installed)
	}
	catalog := rowByID(t, rows, "probe2@picode-probe-mp")
	if catalog.Installed || catalog.Enabled || catalog.Version != "0.9.0" {
		t.Errorf("catalog row = %+v, want not installed at 0.9.0", catalog)
	}
	if catalog.Source != "/tmp/clipkgs-bundles/codex-mp2/plugins/probe2" {
		t.Errorf("catalog source = %q", catalog.Source)
	}
}

// TestHermesRealRoster pins `hermes plugins list --json` (sandboxed: the 59
// plugins Hermes ships, each one present and disabled). PiCode's own
// `picode-native` row is not among them: internal/server installs it on a real
// machine, and this sandbox never ran that.
func TestHermesRealRoster(t *testing.T) {
	rows, note, err := parseHermes(fixture(t, "hermes.list.json"), false)
	if err != nil {
		t.Fatalf("parseHermes: %v", err)
	}
	if note != "" {
		t.Errorf("note = %q, want empty for the installed roster", note)
	}
	if len(rows) != 59 {
		t.Fatalf("rows = %d, want the 59 bundled plugins", len(rows))
	}
	for _, r := range rows {
		if r.Installed != true || r.Enabled {
			t.Errorf("row %s = installed %v / enabled %v, want installed and disabled", r.ID, r.Installed, r.Enabled)
		}
		if r.Status != "not enabled" || r.SourceKind != "bundled" {
			t.Errorf("row %s = status %q / source %q", r.ID, r.Status, r.SourceKind)
		}
		if !strings.Contains(r.Note, "Ships with Hermes") {
			t.Errorf("row %s note = %q, want the bundled caveat", r.ID, r.Note)
		}
		if r.ID == "picode-native" {
			t.Error("picode-native is listed, but nothing installed it in this sandbox")
		}
	}
	first := rowByID(t, rows, "browser-browser-use")
	if first.Version != "1.0.0" || !strings.Contains(first.Description, "Browser Use") {
		t.Errorf("first row = %+v", first)
	}
}

// TestHermesCatalogFixture pins the curated-catalog shape (an excerpt of the
// 222 rows `hermes plugins search --json` returned): the row's source is the
// repository `hermes plugins install` takes, not its own name.
func TestHermesCatalogFixture(t *testing.T) {
	rows, note, err := parseHermes(fixture(t, "hermes.available.json"), true)
	if err != nil {
		t.Fatalf("parseHermes(catalog): %v", err)
	}
	if note == "" {
		t.Error("the catalog read carries no note")
	}
	if len(rows) != 5 {
		t.Fatalf("rows = %d, want the 5 pinned catalog rows", len(rows))
	}
	row := rowByID(t, rows, "adspirer")
	if row.Source != "https://github.com/Adspirer/adspirer-hermes-plugin" || row.SourceKind != "git" {
		t.Errorf("catalog source = %q/%q, want the repository", row.Source, row.SourceKind)
	}
	if row.Installed || row.Enabled || row.Version != "1.0.0" {
		t.Errorf("catalog row = %+v, want not installed at 1.0.0", row)
	}
	for _, r := range rows {
		if !strings.HasPrefix(r.Source, "https://github.com/") {
			t.Errorf("catalog row %s source = %q, want its repository", r.ID, r.Source)
		}
	}
}

// TestGrokRealRoster pins `grok plugin list --json` (sandboxed: one plugin
// installed from a local path, one from a local marketplace).
func TestGrokRealRoster(t *testing.T) {
	rows, note, err := parseVendorRows("grok", fixture(t, "grok.list.json"), false)
	if err != nil {
		t.Fatalf("parseVendorRows: %v", err)
	}
	if note != "" {
		t.Errorf("note = %q, want empty", note)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2: %+v", len(rows), rows)
	}
	fromPath := rowByID(t, rows, "picode-probe")
	if fromPath.Version != "0.1.0" || fromPath.Marketplace != "" || !fromPath.Enabled || !fromPath.Installed {
		t.Errorf("path row = %+v", fromPath)
	}
	if fromPath.Source != "/tmp/clipkgs-bundles/grok2" || fromPath.SourceKind != "path" {
		t.Errorf("path row source = %q/%q", fromPath.Source, fromPath.SourceKind)
	}
	if !strings.HasSuffix(fromPath.InstallPath, "installed-plugins/grok2-288ee1fa") {
		t.Errorf("path row install path = %q", fromPath.InstallPath)
	}
	fromMarketplace := rowByID(t, rows, "mp-probe")
	if fromMarketplace.Marketplace != "grok-mp2" || fromMarketplace.Version != "0.2.0" {
		t.Errorf("marketplace row = %+v", fromMarketplace)
	}
	if fromMarketplace.Source != "/tmp/clipkgs-bundles/grok-mp2/plugins/mp-probe" {
		t.Errorf("marketplace row source = %q", fromMarketplace.Source)
	}
}

// TestGrokAvailableFixture pins the mixed answer `grok plugin list --json
// --available` gives: the installed rows plus the marketplace's uninstalled
// ones, told apart by a status word instead of a boolean.
func TestGrokAvailableFixture(t *testing.T) {
	rows, _, err := parseVendorRows("grok", fixture(t, "grok.available.json"), true)
	if err != nil {
		t.Fatalf("parseVendorRows(available): %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3: %+v", len(rows), rows)
	}
	for _, id := range []string{"picode-probe", "mp-probe"} {
		if r := rowByID(t, rows, id); !r.Installed || !r.Enabled {
			t.Errorf("installed row %s = %+v, want installed and enabled", id, r)
		}
	}
	available := rowByID(t, rows, "mp-probe2")
	if available.Installed || available.Enabled {
		t.Errorf("available row = %+v, want neither installed nor enabled", available)
	}
	if available.Marketplace != "grok-mp2" || available.Version != "0.3.1" || available.Description != "second marketplace probe" {
		t.Errorf("available row = %+v", available)
	}
}

// TestOmpRealRoster pins `omp plugin list --json` (sandboxed: one npm package
// and one plugin installed from a local marketplace, whose version and install
// path live in a nested `entries` array).
func TestOmpRealRoster(t *testing.T) {
	rows, _, err := parseVendorRows("omp", fixture(t, "omp.list.json"), false)
	if err != nil {
		t.Fatalf("parseVendorRows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want the npm and marketplace halves: %+v", len(rows), rows)
	}
	npm := rowByID(t, rows, "picode-probe")
	if npm.Version != "0.1.0" || !npm.Enabled || !npm.Installed || npm.Marketplace != "" {
		t.Errorf("npm row = %+v", npm)
	}
	if !strings.HasSuffix(npm.InstallPath, "plugins/node_modules/picode-probe") {
		t.Errorf("npm row install path = %q", npm.InstallPath)
	}
	mp := rowByID(t, rows, "probe@picode-probe-mp")
	if mp.Name != "probe" || mp.Marketplace != "picode-probe-mp" {
		t.Errorf("marketplace row = %+v, want the id split into name and marketplace", mp)
	}
	if mp.Version != "0.4.0" || !strings.HasSuffix(mp.InstallPath, "picode-probe-mp___probe___0.4.0") {
		t.Errorf("marketplace row = %+v, want the version and path from its entry", mp)
	}
	if !mp.Installed || !mp.Enabled {
		t.Errorf("marketplace row = %+v, want installed and enabled", mp)
	}
}

// TestOmpMarketplaceProse pins `omp plugin marketplace list --json`, which
// ignores its own flag and prints a block of lines (measured: both empty and
// non-empty), so the shared reader has to accept it.
func TestOmpMarketplaceProse(t *testing.T) {
	rows, err := parseMarketplaces(fixture(t, "omp.marketplaces.txt"))
	if err != nil {
		t.Fatalf("parseMarketplaces: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %+v, want the one configured marketplace", rows)
	}
	if rows[0].Name != "picode-probe-mp" || rows[0].Source != "/tmp/clipkgs-bundles/omp-mp" {
		t.Errorf("row = %+v", rows[0])
	}

	// The same command with nothing configured: prose again, and no rows
	// rather than an unreadable-list error.
	empty := "No marketplaces configured\n\nAdd one with: omp plugin marketplace add <source>\n"
	rows, err = parseMarketplaces(empty)
	if err != nil {
		t.Fatalf("parseMarketplaces(empty): %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %+v, want none", rows)
	}
}

// TestMarketplaceFixtures pins the three JSON marketplace lists, whose shapes
// differ per vendor while the rows they produce must not.
func TestMarketplaceFixtures(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		name    string
		source  string
	}{
		{"claude-code.marketplaces.json", "picode-probe-mp", "/tmp/clipkgs-bundles/claude-mp"},
		// The real HOME's own marketplace list: here `source` is the kind word
		// "github" and the location is in `repo`.
		{"claude-code.marketplaces.github.json", "claude-plugins-official", "anthropics/claude-plugins-official"},
		{"codex.marketplaces.json", "picode-probe-mp", "/tmp/clipkgs-bundles/codex-mp2"},
	} {
		rows, err := parseMarketplaces(fixture(t, tc.fixture))
		if err != nil {
			t.Fatalf("%s: %v", tc.fixture, err)
		}
		if len(rows) != 1 {
			t.Fatalf("%s: rows = %+v, want 1", tc.fixture, rows)
		}
		if rows[0].Name != tc.name || rows[0].Source != tc.source {
			t.Errorf("%s: row = %+v, want %s at %s", tc.fixture, rows[0], tc.name, tc.source)
		}
	}

	rows, err := parseMarketplaces(fixture(t, "grok.marketplaces.json"))
	if err != nil {
		t.Fatalf("grok.marketplaces.json: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("grok rows = %+v, want 3 sources", rows)
	}
	first := rowByID(t, rows, "grok-mp-empty")
	if first.Source != "/tmp/clipkgs-bundles/grok-mp-empty" {
		t.Errorf("grok row = %+v, want the local path", first)
	}
}

// TestAgyRealRoster pins `agy plugin list` with plugins imported. The CLI has
// no --json flag, but its non-empty answer is a JSON envelope; the empty answer
// is one sentence, and both are the same command.
func TestAgyRealRoster(t *testing.T) {
	rows, note, err := parseAgy(fixture(t, "agy.list.txt"))
	if err != nil {
		t.Fatalf("parseAgy: %v", err)
	}
	if note == "" {
		t.Error("note is empty; the pane has to say that agy prints no JSON")
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %+v, want the two imported plugins", rows)
	}
	fromClaude := rowByID(t, rows, "probe")
	if fromClaude.Source != "claude-code" || fromClaude.SourceKind != "import" {
		t.Errorf("row = %+v, want the claude-code origin", fromClaude)
	}
	if !fromClaude.Installed || !fromClaude.Enabled || fromClaude.Scope != "user" {
		t.Errorf("row = %+v, want an installed user-scope plugin", fromClaude)
	}
	if other := rowByID(t, rows, "agy-probe"); other.Source != "antigravity" {
		t.Errorf("row = %+v, want the antigravity origin", other)
	}

	empty, note, err := parseAgy(fixture(t, "agy.empty.txt"))
	if err != nil {
		t.Fatalf("parseAgy(empty): %v", err)
	}
	if len(empty) != 0 || note != "" {
		t.Errorf("empty roster = %+v / %q, want no rows and no note", empty, note)
	}
}

// --- the rules the live measurements forced ----------------------------------
//
// These two stand in a stub binary for the vendors the live suite exercises for
// real, so the contracts hold in CI too: a vendor's own refusal words reach the
// caller (Hermes writes them on stdout and exits non-zero), and a failed read is
// an error rather than a roster.

// stubVendor puts a fake vendor on PATH that prints out and exits with code.
func stubVendor(t *testing.T, bin, out string, code int) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\ncat <<'STUBOUT'\n" + out + "\nSTUBOUT\nexit " + strconv.Itoa(code) + "\n"
	if err := os.WriteFile(filepath.Join(dir, bin), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestVendorRefusalOnStdoutReachesTheCaller(t *testing.T) {
	stubVendor(t, "hermes", "Error: Could not download the plugin from https://example.invalid/x.git.", 1)
	out, err := Run(context.Background(), "hermes", VerbInstall, Paths{Home: t.TempDir()},
		Target{Name: "probe", Source: "https://example.invalid/x.git"})
	if err == nil {
		t.Fatalf("the vendor refused but clipkgs reported success: %q", out)
	}
	if !strings.Contains(err.Error(), "Could not download the plugin") {
		t.Errorf("the vendor's own words did not reach the caller: %v", err)
	}
	if strings.Contains(err.Error(), "exit status") {
		t.Errorf("the refusal was reduced to a bare exit status: %v", err)
	}
}

func TestRosterReadFailureIsNeverAnEmptyReport(t *testing.T) {
	stubVendor(t, "grok", "error: the marketplace is unreachable", 1)
	rep, err := List(context.Background(), "grok", Paths{Home: t.TempDir()}, "user", true)
	if err == nil {
		t.Fatalf("a failed vendor read came back as a report: %+v", rep)
	}
	if errors.Is(err, ErrRosterShape) {
		t.Errorf("a vendor failure is reported as a shape error: %v", err)
	}

	// A vendor that answers something unreadable is ErrRosterShape — loud, and
	// never an empty roster that reads as "nothing installed".
	stubVendor(t, "grok", "error: something went wrong on the vendor side", 0)
	rep, err = List(context.Background(), "grok", Paths{Home: t.TempDir()}, "user", true)
	if !errors.Is(err, ErrRosterShape) {
		t.Fatalf("unreadable vendor output gave %+v / %v, want ErrRosterShape", rep, err)
	}
}

// --- OpenCode, the one roster that is a file ---------------------------------

// TestOpencodeFixtureRoster pins OpenCode's roster, the one that is not a
// vendor command: testdata/opencode.json is the file `opencode plugin is-odd
// -g` itself wrote in the sandbox, placed where the CLI reads its user config.
func TestOpencodeFixtureRoster(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	dir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(filepath.Join(dir, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte(fixture(t, "opencode.json")), 0o644); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(dir, "plugins", "probe.js")
	if err := os.WriteFile(local, []byte("export const Probe = async () => ({})\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rows, _, err := opencodeRoster(context.Background(), Paths{Home: home}, "user", false)
	if err != nil {
		t.Fatalf("opencodeRoster: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %+v, want the npm module and the local file", rows)
	}
	npm := rowByID(t, rows, "is-odd")
	if npm.SourceKind != "npm" || !npm.Enabled || !npm.Installed {
		t.Errorf("npm row = %+v", npm)
	}
	if npm.InstallPath != filepath.Join(dir, "opencode.jsonc") {
		t.Errorf("npm row install path = %q, want the config that names it", npm.InstallPath)
	}
	file := rowByID(t, rows, local)
	if file.SourceKind != "local-file" || file.Name != "probe.js" {
		t.Errorf("local row = %+v", file)
	}
	if file.Scope != "user" {
		t.Errorf("local row scope = %q", file.Scope)
	}
}

// TestMuseRealRoster pins the shape `muse plugins list --json` really prints
// (measured 2026-09-21 on Muse Code 1.3.0 in a sandbox whose vendor feature
// gate was on, one local bundle installed). The id is not on the row — it
// lives in `record` — which is exactly what the tolerant reader could not
// read, so a machine with plugins got a refusal instead of a roster.
func TestMuseRealRoster(t *testing.T) {
	rows, note, err := parseMuse(fixture(t, "muse.list.json"), false)
	if err != nil {
		t.Fatalf("parseMuse: %v", err)
	}
	if note != "" {
		t.Errorf("note = %q, want empty for a clean installed roster", note)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want the one installed plugin: %+v", len(rows), rows)
	}
	row := rows[0]
	if row.ID != "picode-probe" || row.Name != "picode-probe" || row.Version != "0.1.0" {
		t.Errorf("row = %+v", row)
	}
	if !row.Installed || !row.Enabled || row.Scope != "user" {
		t.Errorf("row = %+v, want an installed, enabled, user-scope plugin", row)
	}
	if row.Description != "probe" || !strings.Contains(row.InstallPath, "plugins/cache/local/picode-probe") {
		t.Errorf("row description/path = %q / %q", row.Description, row.InstallPath)
	}
	if row.SourceKind != "native-local" || row.Status != "user-local" {
		t.Errorf("row source kind/status = %q / %q, want the vendor's own provenance and trust words", row.SourceKind, row.Status)
	}
	if !strings.Contains(row.Source, "bundle") {
		t.Errorf("row source = %q, want the path the bundle was installed from", row.Source)
	}
}

// TestMuseEmptyAndCatalogFixtures: an empty install and the catalog envelope
// the same CLI prints with nothing configured (`available`/`skipped`/
// `warnings`, none of which is a plugin row).
func TestMuseEmptyAndCatalogFixtures(t *testing.T) {
	rows, _, err := parseMuse(fixture(t, "muse.empty.json"), false)
	if err != nil {
		t.Fatalf("parseMuse(empty): %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %+v, want none", rows)
	}
	cat, note, err := parseMuse(fixture(t, "muse.available.json"), true)
	if err != nil {
		t.Fatalf("parseMuse(catalog): %v", err)
	}
	if len(cat) != 0 || note != "" {
		t.Fatalf("catalog rows/note = %+v / %q, want an empty catalog with no warning", cat, note)
	}
	mps, err := parseMarketplaces(fixture(t, "muse.marketplaces.json"))
	if err != nil {
		t.Fatalf("parseMarketplaces: %v", err)
	}
	if len(mps) != 0 {
		t.Fatalf("marketplaces = %+v, want none (the envelope's other keys are not sources)", mps)
	}
}

// TestMuseInactivePluginSaysSo: `active` is the vendor's own word for whether
// the plugin loaded in this session; a disabled one must not read as active.
func TestMuseInactivePluginSaysSo(t *testing.T) {
	body := `{"plugins":[{"active":false,"active_scope":"disabled","plugin":{"id":"x","version":"1.0.0"},"record":{"id":"x","version":"1.0.0","enabled":false,"cache_path":"/c"}}]}`
	rows, _, err := parseMuse(body, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Enabled || !strings.Contains(rows[0].Note, "not active") {
		t.Fatalf("row = %+v", rows)
	}
}

// TestMuseRefusesAnUnknownEnvelope: a shape mismatch is an error, never a
// silent empty roster.
func TestMuseRefusesAnUnknownEnvelope(t *testing.T) {
	if _, _, err := parseMuse(`[{"id":"x"}]`, false); err == nil {
		t.Fatal("a shape PiCode does not recognize must be refused")
	}
}
