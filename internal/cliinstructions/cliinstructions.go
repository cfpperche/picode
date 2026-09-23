// Package cliinstructions reports which instruction files each agent CLI
// reads for a session started in a folder — AGENTS.md, CLAUDE.md, GEMINI.md
// and their kin — and why it leaves the others out.
//
// One rule per CLI, declared and never inferred at runtime: each was
// measured at the version its Source names, with the probes recorded in
// docs/benchmarks/2026-09-23-agents-md.md (loaders read in source, `grok
// inspect`, the Hermes manifest, model probes on sentinel fixtures). A rule
// the study could only read in a vendor's docs says so in its Source.
//
// Read-only and on demand, like the Memory pane (ADR-0163): the report is
// computed from the files on disk each time it is asked for and stored
// nowhere. File contents never leave this package; the report carries
// paths, sizes and verdicts.
package cliinstructions

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Status is what one CLI does with one file.
type Status string

const (
	StatusReads     Status = "reads"     // in the agent's instructions from the start of a session
	StatusOnDemand  Status = "on-demand" // read once the agent works in that folder
	StatusShadowed  Status = "shadowed"  // another file wins; Cell.By names it
	StatusNotRead   Status = "not-read"  // a name or a place this CLI never reads
	StatusUntrusted Status = "untrusted" // the CLI ignores the folder until it is trusted
	StatusUnknown   Status = "unknown"   // PiCode cannot tell from here
)

// Cell is one CLI's verdict on one file.
type Cell struct {
	Status Status `json:"status"`
	Why    string `json:"why"`
	By     string `json:"by,omitempty"`  // the winning file, for StatusShadowed
	Cut    string `json:"cut,omitempty"` // a limit the file crosses, in words
}

// File is one instruction file found on disk.
type File struct {
	Path    string          `json:"path"`          // as shown: workspace-relative, or ~/… outside it
	Rel     string          `json:"rel,omitempty"` // workspace-relative, only for files inside the workspace
	Scope   string          `json:"scope"`         // "project", "above" (a folder holding the workspace) or "personal"
	Folder  string          `json:"folder"`        // workspace-relative folder ("" = the root) for project files
	Bytes   int64           `json:"bytes"`
	Lines   int             `json:"lines"`
	Imports int             `json:"imports,omitempty"` // lines that start with an import (`@path` or `@[label](path)`)
	Ignored bool            `json:"ignored,omitempty"` // excluded by .gitignore
	Cells   map[string]Cell `json:"cells"`

	abs    string
	target string // where a symbolic link points ("" for a plain file)
	dir    string // the folder the file speaks for
	name   string // its name there: "AGENTS.md", ".claude/CLAUDE.md"
	chars  int
	empty  bool
	sum    [32]byte
	text   string // kept only for the findings that read it; never serialised
}

// CLIInfo is one column: the CLI and where its rule was read.
type CLIInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Installed bool     `json:"installed"`
	Source    string   `json:"source"`
	Notes     []string `json:"notes,omitempty"`
}

// Action is the one thing a finding offers.
type Action struct {
	Kind  string `json:"kind"` // "open" (a workspace file) or "link" (a URL)
	Label string `json:"label"`
	Path  string `json:"path,omitempty"`
	URL   string `json:"url,omitempty"`
}

// Finding is one line about something a person may want to change.
type Finding struct {
	ID     string   `json:"id"`
	Text   string   `json:"text"`
	File   string   `json:"file,omitempty"`
	CLIs   []string `json:"clis,omitempty"`
	Action *Action  `json:"action,omitempty"`
}

// Report is the whole answer for one start folder.
type Report struct {
	Root     string    `json:"root"`
	Start    string    `json:"start"`              // workspace-relative ("" = the root)
	Repo     string    `json:"repo,omitempty"`     // the repository root, when Start is in one
	Worktree bool      `json:"worktree,omitempty"` // Start is in a worktree nested inside its main checkout
	Folders  []string  `json:"folders"`            // workspace-relative folders holding instruction files, for the start picker
	CLIs     []CLIInfo `json:"clis"`
	Files    []*File   `json:"files"`
	Findings []Finding `json:"findings"`
}

// Env is what a report reads. Zero values mean the real machine.
type Env struct {
	Home      string
	Root      string // the workspace folder
	Start     string // the folder a session starts in; "" = Root
	Installed func(cli string) bool
}

// projectNames is every instruction file name any of the nine CLIs reads
// in a folder of the project, checked in every folder of the walk.
var projectNames = []string{
	"AGENTS.override.md", "AGENTS.md", "AGENTS.MD", "Agents.md", "agents.md", "AGENT.md",
	"CLAUDE.md", "CLAUDE.MD", "Claude.md", "claude.md", "CLAUDE.local.md",
	"GEMINI.md", ".hermes.md", "HERMES.md", "CONTEXT.md", ".cursorrules",
	".claude/CLAUDE.md", ".claude/AGENTS.md", ".omp/AGENTS.md", ".agents/AGENTS.md", ".agent/AGENTS.md",
	".gemini/GEMINI.md", ".github/copilot-instructions.md",
}

// plainNames are projectNames without the config-folder entries.
var plainNames = func() []string {
	var out []string
	for _, n := range projectNames {
		if !strings.Contains(n, "/") {
			out = append(out, n)
		}
	}
	return out
}()

// nestedNames are the names worth a row when they sit in a folder below
// the start: the ones some CLI reads once the agent works there. CONTEXT.md
// and .cursorrules are common file names that no CLI reads from a subfolder.
var nestedNames = []string{
	"AGENTS.override.md", "AGENTS.md", "Agents.md", "agents.md", "AGENT.md",
	"CLAUDE.md", "Claude.md", "claude.md", "CLAUDE.local.md",
	"GEMINI.md", ".hermes.md", "HERMES.md", ".claude/CLAUDE.md", ".claude/AGENTS.md",
}

// configDirs are the folders whose instruction file speaks for their parent.
var configDirs = map[string]bool{".claude": true, ".omp": true, ".agents": true, ".agent": true, ".gemini": true, ".github": true}

// personalNames are the per-user files, relative to the home folder.
var personalNames = []string{
	".claude/CLAUDE.md",
	".codex/AGENTS.override.md", ".codex/AGENTS.md",
	".pi/agent/AGENTS.override.md", ".pi/agent/AGENTS.md", ".pi/agent/AGENTS.MD", ".pi/agent/CLAUDE.md", ".pi/agent/CLAUDE.MD",
	".omp/agent/AGENTS.md",
	".config/opencode/AGENTS.md",
	".agents/AGENTS.md", ".agent/AGENTS.md",
	".gemini/GEMINI.md",
	".copilot/copilot-instructions.md",
	".grok/AGENTS.md", ".grok/CLAUDE.md",
}

// scan is the file system as the rules see it.
type scan struct {
	home, root, start string
	top               string   // repository root of start ("" outside one)
	main              string   // the main checkout when top is a worktree nested inside it
	chain             []string // start, its parent, … up to "/"
	files             map[string]*File
	order             []*File
	nested            []*File
	personal          map[string]*File // by personalNames entry
}

// Resolve builds the report for one workspace and start folder.
func Resolve(env Env) (*Report, error) {
	s, err := newScan(env)
	if err != nil {
		return nil, err
	}
	rep := &Report{Root: s.root, Start: s.rel(s.start), Repo: s.top, Worktree: s.main != ""}
	for _, r := range rules {
		installed := true
		if env.Installed != nil {
			installed = env.Installed(r.id)
		}
		r.resolve(s)
		info := CLIInfo{ID: r.id, Name: r.name, Installed: installed, Source: r.source}
		if r.notes != nil {
			info.Notes = r.notes(s)
		}
		rep.CLIs = append(rep.CLIs, info)
	}
	all := append(append([]*File{}, s.order...), s.nested...)
	for _, f := range s.personalList() {
		all = append(all, f)
	}
	for _, f := range all {
		for _, r := range rules {
			if _, ok := f.Cells[r.id]; !ok {
				f.Cells[r.id] = r.fallback(s, f)
			}
		}
	}
	rep.Files = all
	rep.Folders = s.folders()
	rep.Findings = findings(s, rep)
	return rep, nil
}

func newScan(env Env) (*scan, error) {
	home := env.Home
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		home = h
	}
	root, err := canon(env.Root)
	if err != nil {
		return nil, err
	}
	start := root
	if env.Start != "" {
		st, err := canon(env.Start)
		if err != nil {
			return nil, err
		}
		if !within(root, st) {
			return nil, os.ErrPermission
		}
		start = st
	}
	s := &scan{home: filepath.Clean(home), root: root, start: start, files: map[string]*File{}, personal: map[string]*File{}}
	s.top, s.main = gitFacts(start)
	for d := start; ; d = filepath.Dir(d) {
		s.chain = append(s.chain, d)
		if filepath.Dir(d) == d {
			break
		}
	}
	// Top-down, so the rows read the way a person reads the tree.
	for i := len(s.chain) - 1; i >= 0; i-- {
		d := s.chain[i]
		names := projectNames
		if d == s.home {
			// ~/.claude/CLAUDE.md and its kin are personal files, listed below.
			names = plainNames
		}
		for _, name := range present(d, names) {
			scope := "above"
			if within(root, d) {
				scope = "project"
			}
			s.add(filepath.Join(d, name), scope)
		}
	}
	for _, abs := range s.nestedPaths() {
		if f := s.load(abs, "project"); f != nil {
			s.nested = append(s.nested, f)
		}
	}
	for _, name := range personalNames {
		// An empty personal file changes nothing for any CLI; it is not a row.
		if f := s.load(filepath.Join(s.home, name), "personal"); f != nil && !f.empty {
			s.personal[name] = f
		}
	}
	s.markIgnored()
	return s, nil
}

func (s *scan) add(abs, scope string) {
	if f := s.load(abs, scope); f != nil {
		s.order = append(s.order, f)
	}
}

func (s *scan) load(abs, scope string) *File {
	if f, ok := s.files[abs]; ok {
		return f
	}
	st, err := os.Stat(abs)
	if err != nil || !st.Mode().IsRegular() {
		return nil
	}
	body, err := readCapped(abs, 4<<20)
	if err != nil {
		return nil
	}
	f := &File{abs: abs, Scope: scope, Bytes: st.Size(), Cells: map[string]Cell{}}
	if t, err := filepath.EvalSymlinks(abs); err == nil && t != abs {
		f.target = t
	}
	f.text = string(body)
	f.chars = utf8.RuneCount(body)
	f.empty = len(bytes.TrimSpace(body)) == 0
	f.sum = sha256.Sum256(body)
	if len(body) > 0 {
		f.Lines = bytes.Count(body, []byte("\n"))
		if body[len(body)-1] != '\n' {
			f.Lines++
		}
	}
	f.Imports = len(importLine.FindAllIndex(body, -1))
	// .claude/CLAUDE.md and its kin belong to the folder holding the config folder.
	f.dir, f.name = filepath.Dir(abs), filepath.Base(abs)
	if configDirs[filepath.Base(f.dir)] {
		f.name = filepath.Base(f.dir) + "/" + f.name
		f.dir = filepath.Dir(f.dir)
	}
	switch scope {
	case "project":
		f.Rel = s.rel(abs)
		f.Path = f.Rel
		f.Folder = s.rel(f.dir)
	default:
		f.Path = s.tilde(abs)
	}
	s.files[abs] = f
	return f
}

var importLine = regexp.MustCompile(`(?m)^[ \t]*@(\[[^\]]*\]\([^)]+\)|[^\s@][^\s]*)`)

func readCapped(path string, max int64) ([]byte, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return io.ReadAll(io.LimitReader(fh, max))
}

// present lists which of names exist in dir, by the directory's own entries,
// so a case-insensitive file system never reports one file under two names.
func present(dir string, names []string) []string {
	entries := map[string]map[string]bool{}
	list := func(d string) map[string]bool {
		if m, ok := entries[d]; ok {
			return m
		}
		m := map[string]bool{}
		if es, err := os.ReadDir(d); err == nil {
			for _, e := range es {
				m[e.Name()] = true
			}
		}
		entries[d] = m
		return m
	}
	var out []string
	for _, n := range names {
		parent, base := filepath.Split(n)
		d := dir
		if parent != "" {
			sub := strings.TrimSuffix(parent, "/")
			if !list(dir)[sub] {
				continue
			}
			d = filepath.Join(dir, sub)
		}
		if list(d)[base] {
			out = append(out, n)
		}
	}
	return out
}

// nestedPaths finds instruction files in folders below start: git's own list
// when start is in a repository (tracked and untracked-not-ignored, fast on
// any size), a bounded walk otherwise.
func (s *scan) nestedPaths() []string {
	want := map[string]bool{}
	for _, n := range nestedNames {
		want[n] = true
	}
	match := func(rel string) bool {
		rel = filepath.ToSlash(rel)
		dir, base := filepath.Split(rel)
		if dir == "" {
			return false // start's own folder is already on the chain
		}
		if want[base] {
			return true
		}
		parent := filepath.Base(strings.TrimSuffix(dir, "/"))
		return strings.HasPrefix(parent, ".") && want[parent+"/"+base] && strings.Count(rel, "/") >= 2
	}
	var out []string
	if s.top != "" {
		cmd := exec.Command("git", "-C", s.start, "ls-files", "-co", "--exclude-standard", "-z")
		if b, err := cmd.Output(); err == nil {
			for _, rel := range strings.Split(string(b), "\x00") {
				if rel != "" && match(rel) {
					out = append(out, filepath.Join(s.start, rel))
				}
			}
			sort.Strings(out)
			return out
		}
	}
	skip := map[string]bool{"node_modules": true, ".git": true, ".worktrees": true, "vendor": true, "dist": true, "build": true}
	seen := 0
	filepath.WalkDir(s.start, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		seen++
		if seen > 20000 {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if p != s.start && (skip[d.Name()] || strings.Count(strings.TrimPrefix(p, s.start), string(filepath.Separator)) > 6) {
				return filepath.SkipDir
			}
			return nil
		}
		if rel, err := filepath.Rel(s.start, p); err == nil && match(rel) {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// markIgnored asks git which of the files it would ignore (Grok skips them).
func (s *scan) markIgnored() {
	if s.top == "" {
		return
	}
	var paths []string
	for _, f := range append(append([]*File{}, s.order...), s.nested...) {
		if within(s.top, f.abs) {
			paths = append(paths, f.abs)
		}
	}
	if len(paths) == 0 {
		return
	}
	cmd := exec.Command("git", "-C", s.top, "check-ignore", "--stdin", "-z")
	cmd.Stdin = strings.NewReader(strings.Join(paths, "\x00") + "\x00")
	b, _ := cmd.Output() // exit 1 means "none ignored"
	for _, p := range strings.Split(string(b), "\x00") {
		if f, ok := s.files[p]; ok && p != "" {
			f.Ignored = true
		}
	}
}

func (s *scan) personalList() []*File {
	var out []*File
	for _, n := range personalNames {
		if f := s.personal[n]; f != nil {
			out = append(out, f)
		}
	}
	return out
}

func (s *scan) folders() []string {
	seen := map[string]bool{"": true}
	out := []string{""}
	for _, f := range append(append([]*File{}, s.order...), s.nested...) {
		if f.Scope == "project" && !seen[f.Folder] {
			seen[f.Folder] = true
			out = append(out, f.Folder)
		}
	}
	sort.Strings(out)
	return out
}

// at is the file named name in dir, or nil.
func (s *scan) at(dir, name string) *File {
	return s.files[filepath.Join(dir, name)]
}

// dirs is the chain from start up to and including stop (start alone when
// stop is not an ancestor). bottomUp=false returns it top-down.
func (s *scan) dirs(stop string, bottomUp bool) []string {
	var out []string
	for _, d := range s.chain {
		out = append(out, d)
		if d == stop {
			break
		}
	}
	if stop == "" || out[len(out)-1] != stop {
		out = []string{s.start}
	}
	if !bottomUp {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out
}

func (s *scan) rel(abs string) string {
	r, err := filepath.Rel(s.root, abs)
	if err != nil || r == "." {
		return ""
	}
	return filepath.ToSlash(r)
}

func (s *scan) tilde(abs string) string {
	if abs == s.home {
		return "~"
	}
	if within(s.home, abs) {
		return "~/" + filepath.ToSlash(strings.TrimPrefix(abs, s.home+string(filepath.Separator)))
	}
	return abs
}

func canon(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		abs = r
	}
	return filepath.Clean(abs), nil
}

// within reports whether p is dir or below it.
func within(dir, p string) bool {
	if dir == "" {
		return false
	}
	return p == dir || strings.HasPrefix(p, strings.TrimSuffix(dir, string(filepath.Separator))+string(filepath.Separator))
}

// gitFacts is the repository root of dir and, when that root is a linked
// worktree nested inside its own main checkout (the layout PiCode's worktree
// actions create, `<repo>/.worktrees/<name>`), the main checkout's root.
func gitFacts(dir string) (top, main string) {
	out := func(args ...string) string {
		b, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	top = out("rev-parse", "--show-toplevel")
	if top == "" {
		return "", ""
	}
	if t, err := canon(top); err == nil {
		top = t
	}
	common := out("rev-parse", "--path-format=absolute", "--git-common-dir")
	if common == "" || filepath.Base(common) != ".git" {
		return top, ""
	}
	mainRoot, err := canon(filepath.Dir(common))
	if err != nil || mainRoot == top || !within(mainRoot, top) {
		return top, ""
	}
	return top, mainRoot
}
