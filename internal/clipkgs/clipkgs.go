// Package clipkgs manages agent CLIs' native plugins (ADR-0167).
//
// One declaration per CLI (specs.go) says what that CLI really exposes: the
// argv of each verb, where its roster comes from, the scopes it installs into,
// and the sentence the pane shows where a verb does not exist. The vendor
// binary stays the authority — PiCode keeps no plugin database of its own,
// never passes a vendor's auto-consent flag (`-y`, `--trust`, `--confirm`,
// `--allow-tool-override`, an accepted command hash), and never reports an
// empty roster it did not read.
//
// Pi is not here. It keeps its own pane, API and package semantics
// (ADR-0102/0099): machine, workspace and agent scope, PiCode's gallery, and
// the config-descriptor engine. This package covers the other eight CLIs,
// whose plugin systems are one shared shape — a vendor command and a roster —
// with very uneven depth, which is what the capability set records.
package clipkgs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// listTimeout bounds one synchronous vendor call. A roster read must not hang
// a pane: Codex resolves remote marketplaces over the network, Claude forks.
// Mutations run in the durable job lane (internal/clijob) with its own budget.
const listTimeout = 45 * time.Second

// listTTL is how long a roster answers from cache. A pane reload must not
// re-run a vendor command that forks or fetches; mutations invalidate.
const listTTL = 60 * time.Second

// Paths locates what a request touches. Home empty → os.UserHomeDir. Cwd is
// the workspace folder: it is both the scope gate for CLIs whose project layer
// lives in the repository and the working directory a project-scope install
// must run in.
type Paths struct {
	Home string
	Cwd  string
}

func (p Paths) home() string {
	if p.Home != "" {
		return p.Home
	}
	h, _ := os.UserHomeDir()
	return h
}

// Scope is one layer a CLI installs plugins into. Label is what the pane's
// radio says; Note carries a caveat the CLI itself declares (Claude's `local`
// is uncommitted, Omp's project scope applies to marketplace installs).
type Scope struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Note  string `json:"note,omitempty"`
}

// Verb is one operation a CLI may expose.
type Verb string

const (
	VerbInstall      Verb = "install"
	VerbRemove       Verb = "remove"
	VerbEnable       Verb = "enable"
	VerbDisable      Verb = "disable"
	VerbUpdate       Verb = "update"
	VerbInspect      Verb = "inspect"
	VerbMarketAdd    Verb = "marketplace-add"
	VerbMarketList   Verb = "marketplace-list"
	VerbMarketUpdate Verb = "marketplace-update"
	VerbMarketRemove Verb = "marketplace-remove"
)

// Row is one plugin as the pane shows it. Status is the vendor's own word when
// it prints one; Note is the one line the row must carry (a PiCode-managed
// integration, a text-parsed roster, a vendor caveat).
type Row struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Enabled     bool   `json:"enabled"`
	// Installed separates "this CLI has it" from "it is turned on": Claude and
	// Codex disable without uninstalling, and a marketplace catalog row is
	// neither.
	Installed       bool   `json:"installed"`
	Source          string `json:"source,omitempty"`
	SourceKind      string `json:"sourceKind,omitempty"`
	Marketplace     string `json:"marketplace,omitempty"`
	InstallPath     string `json:"installPath,omitempty"`
	Status          string `json:"status,omitempty"`
	Note            string `json:"note,omitempty"`
	ManagedByPiCode bool   `json:"managedByPiCode,omitempty"`
}

// Report is one roster read. Note is about the read itself (a marketplace that
// needed the network, a text roster), never about a plugin.
type Report struct {
	CLI    string `json:"cli"`
	Rows   []Row  `json:"rows"`
	Note   string `json:"note,omitempty"`
	ReadAt string `json:"readAt,omitempty"`
}

// Caps is what the pane may offer. Every field is derived from the declaration
// (the argv builder that exists), never restated by hand.
type Caps struct {
	Install     bool `json:"install"`
	Remove      bool `json:"remove"`
	Toggle      bool `json:"toggle"`
	Update      bool `json:"update"`
	Inspect     bool `json:"inspect"`
	Marketplace bool `json:"marketplace"`
	Available   bool `json:"available"`
}

// Target is one plugin the caller names.
type Target struct {
	Name   string `json:"name"`   // the id the vendor's own list prints
	Source string `json:"source"` // what install takes: npm spec, git URL, path, name@marketplace
	Scope  string `json:"scope"`
	On     bool   `json:"on"`
}

// MarketRequest is one marketplace-source action.
type MarketRequest struct {
	Action string `json:"action"` // add | remove | update
	Source string `json:"source"`
	Name   string `json:"name"`
	Ref    string `json:"ref"`
}

var (
	// ErrNoDriver is a CLI PiCode manages no plugins for (including Pi).
	ErrNoDriver = errors.New("PiCode manages no plugins for this CLI")
	// ErrAgentScope names the rule: no guest CLI has a per-agent plugin layer.
	ErrAgentScope = errors.New("CLI plugins have no per-agent scope")
	// ErrScope is a scope this CLI does not declare.
	ErrScope = errors.New("this CLI has no such plugin scope")
	// ErrNoWorkspace is a project-scope action without a workspace folder.
	ErrNoWorkspace = errors.New("this scope needs a workspace folder")
	// ErrVerbAbsent is a verb the vendor does not expose.
	ErrVerbAbsent = errors.New("this CLI does not expose that operation")
	// ErrBadTarget is a missing or malformed target.
	ErrBadTarget = errors.New("name the plugin to act on")
	// ErrStale is a config file that changed since the report it was built on.
	ErrStale = errors.New("this file changed on disk since it was read")
	// ErrRosterShape is vendor output PiCode does not recognize. Never an
	// empty roster: an unrecognized shape reported as "none installed" is the
	// failure this error exists to prevent.
	ErrRosterShape = errors.New("the CLI's plugin list is not a shape PiCode recognizes")
)

// spec is one CLI's declaration.
type spec struct {
	cli    string
	name   string
	bin    string
	scopes []Scope
	// roster reads the CLI's own list. available=true asks for its marketplace
	// catalog where the CLI has one.
	roster func(ctx context.Context, p Paths, scope string, available bool) ([]Row, string, error)
	// argv builds one verb's command. A nil builder is how "the CLI does not
	// do this" is expressed — one place, so Capabilities() cannot drift.
	argv   map[Verb]func(p Paths, t Target) (dir string, args []string, err error)
	market map[string]func(p Paths, r MarketRequest) (dir string, args []string, err error)
	// writer is the in-process path for a CLI whose only way to change its
	// plugin list is its own config file (OpenCode's `plugin` array). Every
	// other CLI declares argv builders instead, and a CLI with both uses the
	// vendor command wherever one exists.
	writer func(ctx context.Context, p Paths, t Target) error
	// notes is the one-line sentence the pane shows where a verb or concept
	// has no control. The key is the verb, or a named concept (`capabilities`,
	// `marketplace`) the pane looks up by name.
	notes map[string]string
	// available is the CLI's own "include uninstalled marketplace plugins"
	// flag on its list command. It is a separate fact from `roster` because
	// the flag, not the command, is what the Marketplace tab needs.
	available bool
}

func (s *spec) scope(scope string) (Scope, error) {
	if scope == "" {
		scope = "user"
	}
	if scope == "agent" {
		return Scope{}, ErrAgentScope
	}
	for _, sc := range s.scopes {
		if sc.ID == scope {
			return sc, nil
		}
	}
	return Scope{}, fmt.Errorf("%w: %s", ErrScope, scope)
}

func (s *spec) caps() Caps {
	return Caps{
		Install:     s.argv[VerbInstall] != nil,
		Remove:      s.argv[VerbRemove] != nil || s.writer != nil,
		Toggle:      s.argv[VerbEnable] != nil || s.argv[VerbDisable] != nil || s.writer != nil,
		Update:      s.argv[VerbUpdate] != nil,
		Inspect:     s.argv[VerbInspect] != nil,
		Marketplace: s.market["add"] != nil || s.market["remove"] != nil,
		Available:   s.roster != nil && s.hasAvailable(),
	}
}

// hasAvailable is true when the CLI's own list command takes an
// include-available flag (declared per CLI in specs.go).
func (s *spec) hasAvailable() bool { return s.available }

var catalog []*spec

func registry() []*spec {
	if catalog == nil {
		catalog = []*spec{claude, codex, grok, hermes, opencode, muse, agy, omp}
	}
	return catalog
}

// CLIs lists the CLIs with a package declaration, in catalog order.
func CLIs() []string {
	out := make([]string, 0, len(registry()))
	for _, s := range registry() {
		out = append(out, s.cli)
	}
	return out
}

// For returns one CLI's declaration, or nil when PiCode manages no plugins for
// it. Nil covers Pi itself: its pane is its own (ADR-0102).
func For(cli string) *spec { return lookup(cli) }

func lookup(cli string) *spec {
	id := strings.TrimSpace(cli)
	for _, s := range registry() {
		if s.cli == id {
			return s
		}
	}
	return nil
}

// Scopes returns the layers a CLI installs into, in the order it declares
// them. An unknown CLI answers with an empty list rather than an error: the
// pane asks before it knows whether the CLI is supported.
func Scopes(cli string) []Scope {
	s := lookup(cli)
	if s == nil {
		return []Scope{}
	}
	return append([]Scope{}, s.scopes...)
}

// Capabilities returns what a CLI exposes. An unknown CLI exposes nothing.
func Capabilities(cli string) Caps {
	s := lookup(cli)
	if s == nil {
		return Caps{}
	}
	return s.caps()
}

// Notes returns the one-line sentence shown where a verb does not exist. A CLI
// with every verb has no notes.
func Notes(cli string) map[string]string {
	s := lookup(cli)
	if s == nil || len(s.notes) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(s.notes))
	for key, note := range s.notes {
		out[key] = note
	}
	return out
}

// ValidateScope refuses a scope that is not this CLI's, by name and before any
// path resolution: a link carrying `?scope=agent` must never edit a machine
// file instead (ADR-0163's rule, applied here).
func ValidateScope(cli, scope string) error {
	s := lookup(cli)
	if s == nil {
		return ErrNoDriver
	}
	_, err := s.scope(scope)
	return err
}

// Argv builds one verb's command without running it: the pure half, so the
// argv table is testable and the pane could preview it (ADR-0070).
func Argv(cli string, verb Verb, p Paths, t Target) (dir string, args []string, err error) {
	s := lookup(cli)
	if s == nil {
		return "", nil, ErrNoDriver
	}
	build := s.argv[verb]
	if build == nil {
		return "", nil, fmt.Errorf("%w: %s %s", ErrVerbAbsent, cli, verb)
	}
	if _, err := s.scope(t.Scope); err != nil {
		return "", nil, err
	}
	return build(p, t)
}

// Bin returns the vendor binary a CLI's declaration names, for the job lane
// that has to run it without asking for argv a second time.
func Bin(cli string) string {
	s := lookup(cli)
	if s == nil {
		return ""
	}
	return s.bin
}

// Installed reports whether the vendor binary is on PATH.
func Installed(cli string) bool {
	s := lookup(cli)
	if s == nil {
		return false
	}
	_, err := exec.LookPath(s.bin)
	return err == nil
}

// List reads one CLI's roster for one scope. A CLI with no roster command of
// its own is read from its own files (OpenCode). Output PiCode cannot parse is
// ErrRosterShape, never an empty report.
func List(ctx context.Context, cli string, p Paths, scope string, fresh bool) (Report, error) {
	s := lookup(cli)
	if s == nil {
		return Report{}, ErrNoDriver
	}
	if _, err := s.scope(scope); err != nil {
		return Report{}, err
	}
	if s.roster == nil {
		return Report{}, ErrVerbAbsent
	}
	if !installedBin(s.bin) {
		return Report{}, fmt.Errorf("%w: %s is not installed", ErrNoDriver, s.bin)
	}
	key := listKey(cli, p, scope, false)
	if !fresh {
		if rows, note, ok := cachedList(key); ok {
			return report(s, rows, note), nil
		}
	}
	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()
	rows, note, err := s.roster(ctx, p, scope, false)
	if err != nil {
		return Report{}, err
	}
	putList(key, rows, note)
	return report(s, rows, note), nil
}

// Available reads the CLI's own marketplace list — the add surface. It is the
// vendor's catalog, marked installed or not, never a PiCode-curated one.
func Available(ctx context.Context, cli string, p Paths, scope string) (Report, error) {
	s := lookup(cli)
	if s == nil {
		return Report{}, ErrNoDriver
	}
	if s.roster == nil || !s.available {
		return Report{}, fmt.Errorf("%w: %s has no marketplace PiCode can list", ErrVerbAbsent, cli)
	}
	if _, err := s.scope(scope); err != nil {
		return Report{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()
	key := listKey(cli, p, scope, true)
	if rows, note, ok := cachedList(key); ok {
		return report(s, rows, note), nil
	}
	rows, note, err := s.roster(ctx, p, scope, true)
	if err != nil {
		return Report{}, err
	}
	putList(key, rows, note)
	return report(s, rows, note), nil
}

// Run executes one verb through the vendor binary, on the user's behalf and
// with no auto-consent flag. It returns the vendor's stdout; a refusal rides
// the error verbatim so the pane shows the vendor's own words.
func Run(ctx context.Context, cli string, verb Verb, p Paths, t Target) (string, error) {
	s := lookup(cli)
	if s == nil {
		return "", ErrNoDriver
	}
	if s.argv[verb] == nil && s.writer != nil && (verb == VerbRemove || verb == VerbDisable) {
		if _, err := s.scope(t.Scope); err != nil {
			return "", err
		}
		if err := s.writer(ctx, p, t); err != nil {
			return "", err
		}
		Invalidate(cli)
		return "", nil
	}
	dir, args, err := Argv(cli, verb, p, t)
	if err != nil {
		return "", err
	}
	if !installedBin(s.bin) {
		return "", fmt.Errorf("%w: %s is not installed", ErrNoDriver, s.bin)
	}
	out, err := runVendor(ctx, s.bin, dir, args...)
	Invalidate(cli)
	return out, err
}

// MarketArgv is Market's pure half: the argv of one marketplace action,
// testable without a vendor binary.
func MarketArgv(cli, action string, p Paths, r MarketRequest) (dir string, args []string, err error) {
	s := lookup(cli)
	if s == nil {
		return "", nil, ErrNoDriver
	}
	build := s.market[action]
	if build == nil {
		return "", nil, fmt.Errorf("%w: %s marketplace %s", ErrVerbAbsent, cli, action)
	}
	return build(p, r)
}

// Inspect runs the vendor's own inspection verb where the CLI declares one and
// returns its output verbatim: the pane shows the vendor's words in a dialog
// rather than a PiCode summary of them.
func Inspect(ctx context.Context, cli string, p Paths, t Target) (string, error) {
	s := lookup(cli)
	if s == nil {
		return "", ErrNoDriver
	}
	if s.argv[VerbInspect] == nil {
		return "", fmt.Errorf("%w: %s inspects nothing", ErrVerbAbsent, cli)
	}
	dir, args, err := s.argv[VerbInspect](p, t)
	if err != nil {
		return "", err
	}
	return runVendor(ctx, s.bin, dir, args...)
}

// Market runs one marketplace-source action.
func Market(ctx context.Context, cli, action string, p Paths, r MarketRequest) (string, error) {
	s := lookup(cli)
	if s == nil {
		return "", ErrNoDriver
	}
	dir, args, err := MarketArgv(cli, action, p, r)
	if err != nil {
		return "", err
	}
	out, err := runVendor(ctx, s.bin, dir, args...)
	Invalidate(cli)
	return out, err
}

// Marketplaces lists the CLI's configured marketplace sources as rows, for the
// pane's Marketplace tab header. A CLI without the verb answers ErrVerbAbsent.
func Marketplaces(ctx context.Context, cli string, p Paths) ([]Row, error) {
	s := lookup(cli)
	if s == nil {
		return nil, ErrNoDriver
	}
	build := s.market["list"]
	if build == nil {
		return nil, fmt.Errorf("%w: %s lists no marketplaces", ErrVerbAbsent, cli)
	}
	dir, args, err := build(p, MarketRequest{})
	if err != nil {
		return nil, err
	}
	out, err := runVendor(ctx, s.bin, dir, args...)
	if err != nil {
		return nil, err
	}
	rows, err := parseMarketplaces(out)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func report(s *spec, rows []Row, note string) Report {
	// The cached slice is shared between requests: sort a copy, never the
	// cache itself.
	rows = append([]Row{}, rows...)
	if rows == nil {
		rows = []Row{}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Enabled != rows[j].Enabled {
			return rows[i].Enabled
		}
		return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
	})
	return Report{CLI: s.cli, Rows: rows, Note: note, ReadAt: time.Now().UTC().Format(time.RFC3339)}
}

// installedBin is installed() with the name it reads best.
func installedBin(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// runVendor shells out to the CLI itself. Stderr rides the error so a refusal
// reaches the pane verbatim; the subcommand is named (never the full argv,
// which on a marketplace add can carry a token-bearing URL's query).
func runVendor(ctx context.Context, bin, dir string, args ...string) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, listTimeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			// Measured 2026-09-20: Hermes writes its refusals on *stdout* and
			// exits non-zero. Reading only stderr reduced the vendor's own
			// words to "exit status 1", so the pane had nothing to show while
			// the vendor had already explained itself.
			msg = bounded(strings.TrimSpace(stdout.String()), 400)
		}
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), fmt.Errorf("%s %s failed: %s", bin, subcommandOf(args), msg)
	}
	return stdout.String(), nil
}

func subcommandOf(args []string) string {
	switch {
	case len(args) >= 2:
		return args[0] + " " + args[1]
	case len(args) == 1:
		return args[0]
	}
	return ""
}

// --- roster cache -----------------------------------------------------------
//
// A roster read forks a vendor CLI, and Codex's reaches the network. The cache
// is keyed by CLI, scope and the two directories the read depends on; every
// mutation invalidates that CLI.

type listEntry struct {
	at   time.Time
	rows []Row
	note string
}

var listCache = struct {
	mu      sync.Mutex
	entries map[string]listEntry
}{entries: map[string]listEntry{}}

func listKey(cli string, p Paths, scope string, available bool) string {
	return strings.Join([]string{cli, scope, p.home(), p.Cwd, fmt.Sprint(available)}, "\x00")
}

func cachedList(key string) ([]Row, string, bool) {
	listCache.mu.Lock()
	defer listCache.mu.Unlock()
	e, ok := listCache.entries[key]
	if !ok || time.Since(e.at) >= listTTL {
		return nil, "", false
	}
	return e.rows, e.note, true
}

func putList(key string, rows []Row, note string) {
	listCache.mu.Lock()
	defer listCache.mu.Unlock()
	listCache.entries[key] = listEntry{at: time.Now(), rows: rows, note: note}
}

// Invalidate drops a CLI's cached rosters. Every mutation calls it, so a
// succeeded install never leaves the pane reading the previous list.
func Invalidate(cli string) {
	listCache.mu.Lock()
	defer listCache.mu.Unlock()
	for key := range listCache.entries {
		if strings.HasPrefix(key, cli+"\x00") {
			delete(listCache.entries, key)
		}
	}
}

// bounded keeps a vendor's refusal readable as an error: enough of its text to
// carry the reason — a vendor may explain itself over several lines — and never
// a whole listing. It is only used when stderr was silent.
func bounded(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// firstLine is the vendor's own first line, for an error that names what it
// actually printed instead of a JSON byte offset.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 160 {
		s = s[:160] + "…"
	}
	return strings.TrimSpace(s)
}
