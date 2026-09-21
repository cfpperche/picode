package clipkgs

// Live vendor parity (ADR-0167): every roster parser is measured against the
// real vendor CLI in a sandbox HOME, and the vendor's own answers are recorded
// verbatim. This is a measurement, not a CI gate: it shells out to the actual
// binaries, touches nothing outside the sandbox, and skips unless it is
// explicitly pointed at one.
//
// Run it like this (the sandbox HOME is what the vendor CLIs themselves
// resolve, so their stores land there and the real user's files are never read
// or written — with two read-only exceptions on the machine's own home: one
// `claude plugin list` used to capture testdata/claude-code.list.json, and
// Muse's feature config, copied into the sandbox so its plugin surface is
// measurable at all instead of answering "not available in this build"):
//
//	SB=$(mktemp -d)
//	env HOME=$SB XDG_CONFIG_HOME=$SB/.config \
//	    PICODE_LIVE_SANDBOX=$SB PICODE_PKGS_LIVE=1 \
//	    go test ./internal/clipkgs -run TestLiveVendorParity -v -count=1
//
// What counts as a failure, and what does not:
//
//   - A vendor that refuses (no sign-in, no network, a feature its own config
//     turns off — Muse's plugin gate) is a *measurement*: its exit status and the first 200
//     characters of stdout and stderr are logged, and the test goes on. The
//     one thing such a refusal may never do is come back from clipkgs as an
//     empty roster with no error — that is the failure the whole pane rests on
//     (ADR-0167: never an empty list that reads as "nothing installed").
//   - Vendor output that parses as nothing (ErrRosterShape) after a read that
//     *succeeded* IS a failure: the shape moved, and a fixture test and a
//     parser fix belong next to it.
//
// The second half hands each CLI the smallest real bundle its own verbs accept
// offline and re-reads the roster, so the non-empty shapes are measured too —
// those are exactly the shapes PiCode could not observe when the parsers were
// written. Vendors that need a human consent answer (Grok's `--trust`) are
// measured for their refusal: PiCode never passes an auto-consent flag, so the
// refusal is the expected answer and the copyable command is what the pane
// shows.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	liveEnv     = "PICODE_PKGS_LIVE"
	liveSandbox = "PICODE_LIVE_SANDBOX"
	// liveTimeout bounds one vendor call in the harness. It is longer than
	// clipkgs' own list timeout because an install may fetch.
	liveTimeout = 180 * time.Second
)

// liveSkip guards the whole harness: it must be opted in, the process HOME
// must be the declared sandbox, and the sandbox must exist — otherwise a stray
// run would point the vendor CLIs at the real user's stores.
func liveSkip(t *testing.T) string {
	t.Helper()
	if os.Getenv(liveEnv) != "1" {
		t.Skip("set PICODE_PKGS_LIVE=1 with HOME pointed at a sandbox to measure the real vendor CLIs")
	}
	sb := os.Getenv(liveSandbox)
	if sb == "" || sb != os.Getenv("HOME") {
		t.Skip("PICODE_LIVE_SANDBOX must equal the process HOME (the vendor CLIs write there)")
	}
	if st, err := os.Stat(sb); err != nil || !st.IsDir() {
		t.Fatalf("PICODE_LIVE_SANDBOX %q is not an existing directory", sb)
	}
	return sb
}

func liveInstalled(t *testing.T, cli string) {
	t.Helper()
	bin := Bin(cli)
	if bin == "" || !Installed(cli) {
		t.Skipf("%s is not installed on this machine", bin)
	}
}

// liveProbe is the vendor's own roster command, one entry per CLI. It mirrors
// the roster closure in specs.go, and is repeated here on purpose: a clipkgs
// error carries the vendor's stderr, so without this the harness could not
// record what a refusing vendor printed on stdout.
var liveProbe = map[string][]string{
	"claude-code": {"plugin", "list", "--json"},
	"codex":       {"plugin", "list", "--json"},
	"grok":        {"plugin", "list", "--json"},
	"hermes":      {"plugins", "list", "--json"},
	"muse":        {"plugins", "list", "--json"},
	"agy":         {"plugin", "list"},
	"omp":         {"plugin", "list", "--json"},
}

// liveAvailableProbe mirrors each CLI's marketplace read (the available/search
// closure in specs.go). OpenCode has none, and Omp's list command rejects
// --available, which is why Capabilities("omp").Available is false.
var liveAvailableProbe = map[string][]string{
	"claude-code": {"plugin", "list", "--json", "--available"},
	"codex":       {"plugin", "list", "--json", "--available"},
	"grok":        {"plugin", "list", "--json", "--available"},
	"hermes":      {"plugins", "search", "--json"},
	"muse":        {"plugins", "list", "--json", "--available"},
}

type liveResult struct {
	stdout string
	stderr string
	err    error
}

// liveRun shells out to a real vendor CLI with a hard timeout; a hung CLI
// fails the test instead of wedging the run.
func liveRun(t *testing.T, bin, dir string, args ...string) liveResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), liveTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("%s %s timed out after %s", bin, strings.Join(args, " "), liveTimeout)
	}
	return liveResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

// liveHead is the first 200 characters of a vendor answer, verbatim.
func liveHead(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}

// liveAnswer records one vendor call: its status and the first 200 characters
// of both streams. This is the measurement a person reads in -v output and in
// the parity report.
func liveAnswer(t *testing.T, cli string, args []string, r liveResult) {
	t.Helper()
	status := "exit ok"
	if r.err != nil {
		status = r.err.Error()
	}
	t.Logf("MEASURED %s %s → %s", cli, strings.Join(args, " "), status)
	t.Logf("  stdout %q", liveHead(r.stdout))
	t.Logf("  stderr %q", liveHead(r.stderr))
}

func liveLogRows(t *testing.T, what string, rows []Row) {
	t.Helper()
	t.Logf("MEASURED %s: %d row(s)", what, len(rows))
	for i, r := range rows {
		if i >= 4 {
			t.Logf("  … %d more", len(rows)-i)
			break
		}
		t.Logf("  %s | v=%s | scope=%s | enabled=%v | installed=%v | %s:%s | mp=%s",
			r.ID, r.Version, r.Scope, r.Enabled, r.Installed, r.SourceKind, r.Source, r.Marketplace)
	}
}

// liveRowsOnly is the rule the pane depends on: a read that answered is a
// non-nil row list with readable rows, and a read that failed is an error.
// Anything else is the "empty roster that reads as nothing installed" failure
// ADR-0167 names.
func liveRowsOnly(t *testing.T, what string, rep Report) {
	t.Helper()
	if rep.Rows == nil {
		t.Errorf("%s: nil rows and no error", what)
	}
	for _, r := range rep.Rows {
		if r.ID == "" || r.Name == "" {
			t.Errorf("%s: unreadable row %+v", what, r)
		}
	}
}

func liveFind(rows []Row, id string) *Row {
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}

// liveRealHome is the machine's own home. It cannot come from the environment:
// the recipe in this file's header swaps HOME for the whole test process, so a
// vendor whose feature surface is gated by its own cached config (Muse — every
// plugin verb answers "not available in this build" without it) would be
// measured with the gate off, which is the wrong conclusion. os/user reads the
// account record instead. This is the one read outside the sandbox, and it is
// one directory of the vendor's own cache.
var liveRealHome = func() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.HomeDir
}()

func TestLiveVendorParity(t *testing.T) {
	sb := liveSkip(t)
	if liveSeedMuseGate(t, sb) {
		// The sandbox now answers the same plugin surface a configured machine
		// does, so every phase below measures the vendor and not the gate.
		defer t.Log("MEASURED muse plugin surface enabled in the sandbox")
	}
	t.Setenv("HOME", sb)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(sb, ".config"))
	if err := os.MkdirAll(filepath.Join(sb, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	paths := Paths{Home: sb, Cwd: t.TempDir()}

	for _, cli := range CLIs() {
		cli := cli
		t.Run(cli, func(t *testing.T) {
			liveInstalled(t, cli)
			t.Run("roster", func(t *testing.T) { liveMeasureRoster(t, cli, paths) })
			t.Run("available", func(t *testing.T) { liveMeasureAvailable(t, cli, paths) })
			t.Run("sources", func(t *testing.T) { liveMeasureSources(t, cli, paths) })
		})
	}

	t.Run("nonempty", func(t *testing.T) {
		// The vendor CLIs remember what an earlier run registered — Claude keeps
		// a source directory's original marketplace name, Omp refuses a package
		// name it has already installed — so the install half measures once per
		// sandbox. A sandbox that already ran keeps the read-only measurements
		// above and skips this half; the recipe's `mktemp -d` is what makes a
		// fresh one.
		marker := filepath.Join(sb, "picode-live-fixtures", "seeded")
		if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(marker); err == nil {
			t.Skipf("this sandbox was seeded by an earlier run; run the install half against a fresh one (SB=$(mktemp -d))")
		}
		if err := os.WriteFile(marker, []byte("seeded\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, cli := range CLIs() {
			cli := cli
			t.Run(cli, func(t *testing.T) {
				liveInstalled(t, cli)
				liveSeedAndRead(t, cli, paths)
			})
		}
	})
}

func liveMeasureRoster(t *testing.T, cli string, p Paths) {
	t.Helper()
	var raw liveResult
	if args, ok := liveProbe[cli]; ok {
		raw = liveRun(t, Bin(cli), "", args...)
		liveAnswer(t, cli, args, raw)
	}
	rep, err := List(context.Background(), cli, p, "user", true)
	if err != nil {
		if errors.Is(err, ErrRosterShape) && raw.err == nil {
			t.Fatalf("%s answered a shape the parser does not recognize: %v", cli, err)
		}
		if raw.err != nil {
			t.Logf("MEASURED %s refused or failed inside clipkgs: %v", cli, err)
			return
		}
		// The vendor answered; clipkgs must not turn that into a failure.
		t.Fatalf("%s: vendor read succeeded but clipkgs failed: %v", cli, err)
	}
	liveRowsOnly(t, cli+" roster", rep)
	if raw.err != nil {
		t.Errorf("%s exited %v but clipkgs returned a roster with no error — a refusal must never read as an empty list", cli, raw.err)
	}
	if len(rep.Rows) == 0 {
		t.Logf("MEASURED %s: empty roster in this sandbox (the vendor read itself succeeded)", cli)
	}
	if rep.Note != "" {
		t.Logf("MEASURED %s note: %q", cli, rep.Note)
	}
	liveLogRows(t, cli+" roster", rep.Rows)
}

func liveMeasureAvailable(t *testing.T, cli string, p Paths) {
	t.Helper()
	if !Capabilities(cli).Available {
		t.Skipf("%s declares no marketplace list PiCode reads", cli)
	}
	var raw liveResult
	if args, ok := liveAvailableProbe[cli]; ok {
		raw = liveRun(t, Bin(cli), "", args...)
		liveAnswer(t, cli, args, raw)
	}
	rep, err := Available(context.Background(), cli, p, "user")
	if err != nil {
		if errors.Is(err, ErrRosterShape) && raw.err == nil {
			t.Fatalf("%s's marketplace answered a shape the parser does not recognize: %v", cli, err)
		}
		// A refusal (sign-in, network, a build without the feature) is a
		// measurement; it must simply never come back as empty rows.
		t.Logf("MEASURED %s marketplace unavailable: %v", cli, err)
		return
	}
	liveRowsOnly(t, cli+" marketplace", rep)
	if raw.err != nil {
		t.Errorf("%s exited %v but clipkgs returned marketplace rows with no error", cli, raw.err)
	}
	if rep.Note != "" {
		t.Logf("MEASURED %s marketplace note: %q", cli, rep.Note)
	}
	liveLogRows(t, cli+" marketplace", rep.Rows)
}

func liveMeasureSources(t *testing.T, cli string, p Paths) {
	t.Helper()
	rows, err := Marketplaces(context.Background(), cli, p)
	if errors.Is(err, ErrVerbAbsent) {
		t.Skipf("%s lists no marketplace sources", cli)
	}
	if err != nil {
		t.Logf("MEASURED %s marketplace sources failed: %v", cli, err)
		return
	}
	liveLogRows(t, cli+" marketplace sources", rows)
}

// --- non-empty shapes -------------------------------------------------------
//
// Each seed below is the smallest bundle the vendor's own verb accepts, written
// in a temp dir (never in the sandbox HOME, which only the vendors write).

func liveWrite(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// liveFixtureDir is a stable directory inside the sandbox rather than a
// t.TempDir(): Omp registers a marketplace by the directory it was added from,
// so a second run against the same sandbox must still find the bundle there.
func liveFixtureDir(t *testing.T, home, cli string) string {
	t.Helper()
	dir := filepath.Join(home, "picode-live-fixtures", cli)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// liveFixtureName is the marketplace source this harness registers, and the
// prefix of every plugin id it asserts.
const liveFixtureName = "picode-probe-live"

// liveProbeMarketplace writes a two-plugin marketplace: `picode-probe` is the
// one installed, `picode-spare` stays uninstalled so the catalog half of the
// marketplace read has a row too.
func liveProbeMarketplace(t *testing.T, dir, manifest, pluginManifest string) string {
	t.Helper()
	liveWrite(t, filepath.Join(dir, manifest), `{
  "name": "`+liveFixtureName+`",
  "owner": {"name": "PiCode live parity"},
  "plugins": [
    {"name": "picode-probe", "source": "./plugins/picode-probe", "description": "PiCode live probe", "version": "0.1.0"},
    {"name": "picode-spare", "source": "./plugins/picode-spare", "description": "PiCode live spare", "version": "0.2.0"}
  ]
}`)
	for name, version := range map[string]string{"picode-probe": "0.1.0", "picode-spare": "0.2.0"} {
		liveWrite(t, filepath.Join(dir, "plugins", name, pluginManifest),
			`{"name": "`+name+`", "version": "`+version+`", "description": "PiCode live probe plugin"}`)
	}
	return dir
}

// liveProbeBundle writes the minimal single-plugin directory: a manifest the
// vendor's own `validate`/`install` accepts for a local path install.
func liveProbeBundle(t *testing.T, dir, manifest string) string {
	t.Helper()
	liveWrite(t, filepath.Join(dir, manifest), `{"name": "picode-probe", "version": "0.1.0", "description": "PiCode live probe plugin"}`)
	return dir
}

// liveSeedMuseGate copies Muse's own feature config from the real home into
// the sandbox. Without it the vendor answers every plugin verb with "plugins
// are not available in this build", which is a property of the machine's
// configuration, not of the build — the wrong conclusion an earlier run of
// this harness drew. Answers false when the real home has no such config, so
// the caller can skip with the reason instead of reporting a refusal shape.
func liveSeedMuseGate(t *testing.T, sandboxHome string) bool {
	t.Helper()
	if liveRealHome == "" {
		return false
	}
	src := filepath.Join(liveRealHome, ".local", "share", "muse", "feature-config")
	entries, err := os.ReadDir(src)
	if err != nil || len(entries) == 0 {
		return false
	}
	dir := filepath.Join(sandboxHome, ".local", "share", "muse", "feature-config")
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatalf("read Muse feature config: %v", err)
		}
		liveWrite(t, filepath.Join(dir, e.Name()), string(b))
	}
	t.Logf("MEASURED muse feature config seeded from %s (%d file(s))", src, len(entries))
	return true
}

// liveMuseBundle writes the smallest bundle `muse plugins validate` accepts
// (measured 2026-09-21): a manifest declaring schemaVersion 1, the manifest
// directory it lives in, and one capability — a plugin whose manifest declares
// no capability is refused at install with "no supported behavior
// capabilities".
func liveMuseBundle(t *testing.T, dir, name, version string) string {
	t.Helper()
	liveWrite(t, filepath.Join(dir, ".muse-plugin", "plugin.json"), `{
  "schemaVersion": 1,
  "name": "`+name+`",
  "version": "`+version+`",
  "description": "PiCode live probe",
  "compat": {"manifestDir": ".muse-plugin"},
  "capabilities": {"skills": [{"id": "probe", "path": "skills/probe/SKILL.md"}]}
}`)
	liveWrite(t, filepath.Join(dir, "skills", "probe", "SKILL.md"), "# PiCode live probe\n")
	return dir
}

// liveMuseMarketplace writes the marketplace Muse accepts: a catalog in
// Claude's format (its plugin-store reader probes `.claude-plugin/marketplace.json`,
// measured 2026-09-21) whose entries point at Muse-format bundles.
func liveMuseMarketplace(t *testing.T, dir string) string {
	t.Helper()
	liveWrite(t, filepath.Join(dir, ".claude-plugin", "marketplace.json"), `{
  "name": "`+liveFixtureName+`",
  "owner": {"name": "PiCode live parity"},
  "plugins": [
    {"name": "picode-probe", "source": "./plugins/picode-probe", "description": "PiCode live probe", "version": "0.1.0"},
    {"name": "picode-spare", "source": "./plugins/picode-spare", "description": "PiCode live spare", "version": "0.2.0"}
  ]
}`)
	liveMuseBundle(t, filepath.Join(dir, "plugins", "picode-probe"), "picode-probe", "0.1.0")
	liveMuseBundle(t, filepath.Join(dir, "plugins", "picode-spare"), "picode-spare", "0.2.0")
	return dir
}

// liveAddMarketplace adds a marketplace source, tolerating one that is already
// configured: a person re-runs this harness against the same sandbox (Omp
// answers "Marketplace … already exists" and fails, measured 2026-09-20).
func liveAddMarketplace(t *testing.T, cli, source, ref string, p Paths) {
	t.Helper()
	out, err := Market(context.Background(), cli, "add", p, MarketRequest{Action: "add", Source: source, Ref: ref})
	if err == nil {
		t.Logf("MEASURED %s marketplace add: %s", cli, liveHead(out))
		return
	}
	if strings.Contains(strings.ToLower(err.Error()+" "+out), "already") {
		t.Logf("MEASURED %s marketplace source was already configured: %v", cli, err)
		return
	}
	t.Fatalf("marketplace add: %v (%s)", err, liveHead(out))
}

func liveSeedAndRead(t *testing.T, cli string, p Paths) {
	t.Helper()
	ctx := context.Background()
	switch cli {
	case "claude-code":
		mp := liveProbeMarketplace(t, liveFixtureDir(t, p.Home, cli), ".claude-plugin/marketplace.json", ".claude-plugin/plugin.json")
		liveAddMarketplace(t, cli, mp, "user", p)
		if out, err := Run(ctx, cli, VerbInstall, p, Target{Name: "picode-probe", Source: "picode-probe@" + liveFixtureName, Scope: "user"}); err != nil {
			t.Fatalf("install: %v (%s)", err, liveHead(out))
		}
		rep, err := List(ctx, cli, p, "user", true)
		if err != nil {
			t.Fatalf("roster after install: %v", err)
		}
		liveLogRows(t, cli+" after install", rep.Rows)
		row := liveFind(rep.Rows, "picode-probe@"+liveFixtureName)
		if row == nil {
			t.Fatalf("the plugin the vendor just installed is not in the roster: %+v", rep.Rows)
		}
		if !row.Installed || !row.Enabled || row.Marketplace != liveFixtureName || row.SourceKind != "marketplace" {
			t.Errorf("installed row = %+v", *row)
		}
		cat, err := Available(ctx, cli, p, "user")
		if err != nil {
			t.Fatalf("marketplace after install: %v", err)
		}
		liveLogRows(t, cli+" after install (marketplace)", cat.Rows)
		if spare := liveFind(cat.Rows, "picode-spare@"+liveFixtureName); spare == nil {
			t.Errorf("the uninstalled marketplace plugin is missing: %+v", cat.Rows)
		} else if spare.Installed || spare.Enabled || spare.Marketplace != liveFixtureName {
			t.Errorf("catalog row = %+v, want neither installed nor enabled", *spare)
		}

	case "codex":
		mp := liveProbeMarketplace(t, liveFixtureDir(t, p.Home, cli), ".agents/plugins/marketplace.json", ".codex-plugin/plugin.json")
		liveAddMarketplace(t, cli, mp, "", p)
		if out, err := Run(ctx, cli, VerbInstall, p, Target{Name: "picode-probe", Source: "picode-probe@" + liveFixtureName}); err != nil {
			t.Fatalf("install: %v (%s)", err, liveHead(out))
		}
		rep, err := List(ctx, cli, p, "user", true)
		if err != nil {
			t.Fatalf("roster after install: %v", err)
		}
		liveLogRows(t, cli+" after install", rep.Rows)
		row := liveFind(rep.Rows, "picode-probe@"+liveFixtureName)
		if row == nil {
			t.Fatalf("the plugin the vendor just installed is not in the roster: %+v", rep.Rows)
		}
		want := filepath.Join(mp, "plugins", "picode-probe")
		if row.Source != want || row.SourceKind != "local" {
			t.Errorf("installed row source = %q/%q, want the plugin path %q with kind local", row.Source, row.SourceKind, want)
		}
		if !row.Installed || !row.Enabled {
			t.Errorf("installed row = %+v", *row)
		}
		cat, err := Available(ctx, cli, p, "user")
		if err != nil {
			t.Fatalf("marketplace after install: %v", err)
		}
		liveLogRows(t, cli+" after install (marketplace)", cat.Rows)
		if spare := liveFind(cat.Rows, "picode-spare@"+liveFixtureName); spare == nil {
			t.Errorf("the uninstalled marketplace plugin is missing: %+v", cat.Rows)
		} else if spare.Installed {
			t.Errorf("catalog row = %+v, want not installed", *spare)
		}

	case "omp":
		mp := liveProbeMarketplace(t, liveFixtureDir(t, p.Home, cli), ".omp-plugin/marketplace.json", ".claude-plugin/plugin.json")
		liveAddMarketplace(t, cli, mp, "", p)
		if out, err := Run(ctx, cli, VerbInstall, p, Target{Name: "picode-probe", Source: "picode-probe@" + liveFixtureName}); err != nil {
			t.Fatalf("install from marketplace: %v (%s)", err, liveHead(out))
		}
		bundle := liveProbeBundle(t, liveFixtureDir(t, p.Home, cli), "package.json")
		if out, err := Run(ctx, cli, VerbInstall, p, Target{Name: "picode-probe-npm", Source: bundle}); err != nil {
			t.Logf("MEASURED %s local-path install failed: %v (%s)", cli, err, liveHead(out))
		}
		rep, err := List(ctx, cli, p, "user", true)
		if err != nil {
			t.Fatalf("roster after install: %v", err)
		}
		liveLogRows(t, cli+" after install", rep.Rows)
		row := liveFind(rep.Rows, "picode-probe@"+liveFixtureName)
		if row == nil {
			t.Fatalf("the plugin the vendor just installed is not in the roster: %+v", rep.Rows)
		}
		if row.Version == "" || row.InstallPath == "" {
			t.Errorf("marketplace row = %+v, want the version and path its entry carries", *row)
		}
		if !row.Installed || !row.Enabled || row.Name != "picode-probe" || row.Marketplace != liveFixtureName {
			t.Errorf("marketplace row = %+v, want the id split into name and marketplace", *row)
		}

	case "agy":
		bundle := liveProbeBundle(t, liveFixtureDir(t, p.Home, cli), "plugin.json")
		out, err := Run(ctx, cli, VerbInstall, p, Target{Name: "picode-probe", Source: bundle})
		if err != nil {
			t.Fatalf("install: %v (%s)", err, liveHead(out))
		}
		rep, err := List(ctx, cli, p, "user", true)
		if err != nil {
			t.Fatalf("roster after install: %v", err)
		}
		liveLogRows(t, cli+" after install", rep.Rows)
		if rep.Note == "" {
			t.Error("the roster note is empty; the pane has to say agy prints no JSON")
		}
		if len(rep.Rows) == 0 {
			t.Fatalf("agy installed a plugin and printed no row: %+v", rep)
		}
		row := liveFind(rep.Rows, "picode-probe")
		if row == nil {
			t.Fatalf("the plugin the vendor just installed is not in the roster: %+v", rep.Rows)
		}
		// Measured 2026-09-20: a plain directory install is attributed to
		// "antigravity"; an import from Claude carries "claude-code".
		if row.Source != "antigravity" || row.SourceKind != "import" {
			t.Errorf("installed row = %+v, want the antigravity origin", *row)
		}
		if !row.Installed || !row.Enabled || row.Scope != "user" {
			t.Errorf("installed row = %+v, want an installed user-scope plugin", *row)
		}

	case "grok":
		// Grok's install requires --trust, which PiCode never passes: the
		// vendor's own refusal is the answer, and the roster must stay exactly
		// as readable as before.
		bundle := liveProbeBundle(t, liveFixtureDir(t, p.Home, cli), "plugin.json")
		out, err := Run(ctx, cli, VerbInstall, p, Target{Name: "picode-probe", Source: bundle})
		if err != nil {
			t.Logf("MEASURED %s refused the local install (no auto-consent flag is ever passed): %v (%s)", cli, err, liveHead(out))
		} else {
			t.Errorf("%s installed a local bundle with no consent flag: %s", cli, liveHead(out))
		}
		rep, err := List(ctx, cli, p, "user", true)
		if err != nil {
			t.Fatalf("roster after a refused install: %v", err)
		}
		liveLogRows(t, cli+" after a refused install", rep.Rows)
		if row := liveFind(rep.Rows, "picode-probe"); row != nil {
			t.Errorf("a refused install left a row behind: %+v", *row)
		}

	case "hermes":
		// Hermes takes a catalog entry, a Git URL or owner/repo — never a local
		// path, which it reads as a GitHub shorthand and fails to clone. Its
		// refusal arrives on *stdout* with a non-zero exit, which is why
		// runVendor falls back to stdout: the pane has to show these words.
		bundle := liveProbeBundle(t, liveFixtureDir(t, p.Home, cli), "plugin.json")
		out, err := Run(ctx, cli, VerbInstall, p, Target{Name: "picode-probe", Source: bundle})
		if err == nil {
			t.Logf("MEASURED %s accepted a local path after all: %s", cli, liveHead(out))
		} else {
			t.Logf("MEASURED %s refused the local install: %v", cli, err)
			if strings.Contains(err.Error(), "exit status") {
				t.Errorf("the refusal reached the pane as a bare exit status instead of the vendor's own words: %v", err)
			}
		}

	case "muse":
		// Muse gates its whole plugin surface per machine through the vendor's
		// own feature config, so a fresh HOME answers "plugins are not
		// available in this build" for every verb — measured 2026-09-21, and
		// the reason an earlier run of this harness read Muse as unmeasurable.
		// The harness copies that config from the real home into the sandbox
		// (the CLI's own cache, not a fixture PiCode ships) and then installs
		// the smallest bundle Muse's own validator accepts.
		if _, err := os.Stat(filepath.Join(p.Home, ".local", "share", "muse", "feature-config")); err != nil {
			t.Skip("this machine's Muse carries no feature config, so its plugin surface is off in any sandbox: nothing to measure")
		}
		mp := liveMuseMarketplace(t, liveFixtureDir(t, p.Home, cli))
		if out, err := Market(ctx, cli, "add", p, MarketRequest{Action: "add", Name: liveFixtureName, Source: mp}); err != nil {
			t.Fatalf("marketplace add: %v (%s)", err, liveHead(out))
		}
		// The plugin the catalog offers, installed through the vendor's own
		// command, using the spec the pane builds from the catalog row.
		if out, err := Run(ctx, cli, VerbInstall, p, Target{Name: "picode-spare", Source: "picode-spare@" + liveFixtureName}); err != nil {
			t.Fatalf("install from marketplace: %v (%s)", err, liveHead(out))
		}
		rep, err := List(ctx, cli, p, "user", true)
		if err != nil {
			t.Fatalf("roster after install: %v", err)
		}
		liveLogRows(t, cli+" after install", rep.Rows)
		row := liveFind(rep.Rows, "picode-spare")
		if row == nil {
			t.Fatalf("the plugin the vendor just installed is not in the roster: %+v", rep.Rows)
		}
		if !row.Installed || !row.Enabled || row.Version != "0.2.0" {
			t.Errorf("installed row = %+v", *row)
		}
		// Measured 2026-09-21: the provenance word depends on how the plugin
		// arrived — `native-local` for a direct path install, and
		// `marketplace-user-added` for one installed from a marketplace — so
		// the assertion is on the fact (it came from a marketplace) and the
		// vendor's own word is logged and pinned by the fixture next to it.
		if !strings.Contains(row.SourceKind, "marketplace") || row.InstallPath == "" || row.Source == "" {
			t.Errorf("installed row = %+v, want a marketplace provenance, its path and origin", *row)
		}
		cat, err := Available(ctx, cli, p, "user")
		if err != nil {
			t.Fatalf("catalog: %v", err)
		}
		liveLogRows(t, cli+" catalog", cat.Rows)
		spare := liveFind(cat.Rows, "picode-spare")
		if spare == nil {
			t.Fatalf("the marketplace plugin is missing from the catalog: %+v", cat.Rows)
		}
		// Measured: Muse's catalog keeps saying "available" after an install, so
		// Installed here is the join with the CLI's own roster.
		if spare.Source != "picode-spare@"+liveFixtureName {
			t.Errorf("catalog install spec = %q, want name@marketplace", spare.Source)
		}
		if !spare.Installed || spare.Marketplace != liveFixtureName {
			t.Errorf("catalog row = %+v, want the installed join and the marketplace name", *spare)
		}
		sources, err := Marketplaces(ctx, cli, p)
		if err != nil {
			t.Fatalf("marketplace sources: %v", err)
		}
		if len(sources) != 1 || sources[0].Name != liveFixtureName {
			t.Errorf("marketplace sources = %+v", sources)
		}

	case "opencode":
		// OpenCode has no roster command at all: its store is its config and
		// its plugins directory, and testdata/opencode.json is the file its own
		// `opencode plugin is-odd -g` wrote. The harness writes that same file
		// where the CLI reads it, plus one local plugin file.
		dir := filepath.Join(p.Home, ".config", "opencode")
		liveWrite(t, filepath.Join(dir, "opencode.jsonc"), fixture(t, "opencode.json"))
		liveWrite(t, filepath.Join(dir, "plugins", "picode-probe.js"), "export const PicodeProbe = async () => ({})\n")
		rep, err := List(ctx, cli, p, "user", true)
		if err != nil {
			t.Fatalf("roster: %v", err)
		}
		liveLogRows(t, cli+" store", rep.Rows)
		if liveFind(rep.Rows, "is-odd") == nil || liveFind(rep.Rows, filepath.Join(dir, "plugins", "picode-probe.js")) == nil {
			t.Fatalf("the config module and the local file are both part of the roster: %+v", rep.Rows)
		}

		// Measured with `opencode debug config`: the vendor reads the project
		// file `opencode plugin <module>` (no -g) writes, <cwd>/.opencode/
		// opencode.json, as well as <cwd>/opencode.json. PiCode's project-scope
		// read (connectors.OpenCodeConfigFiles) names only the latter two, so a
		// project-scope install through the vendor's own command lands in a file
		// the roster read does not open. Logged, not asserted: the path rule has
		// one owner and it is internal/connectors.
		liveWrite(t, filepath.Join(p.Cwd, ".opencode", "opencode.json"), `{"plugin": ["picode-project-probe"]}`)
		project, err := List(ctx, cli, p, "project", true)
		if err != nil {
			t.Fatalf("project roster: %v", err)
		}
		if liveFind(project.Rows, "picode-project-probe") == nil {
			t.Logf("DIVERGENCE: %s was written by the vendor's own project-scope command and the project-scope roster does not read it (rows: %+v)",
				filepath.Join(p.Cwd, ".opencode", "opencode.json"), project.Rows)
		}

	default:
		t.Skipf("no live seed is known for %s", cli)
	}
}
