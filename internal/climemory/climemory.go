// Package climemory reports what each agent CLI has remembered between
// sessions, and edits it only where the vendor expects a human to (ADR-0163).
//
// Four tiers, declared per CLI and never inferred at runtime:
//
//	editable  plain markdown the vendor treats as user-owned — list, open,
//	          edit, delete (Claude Code, Hermes, Omp, Muse Code)
//	readonly  a generated index backed by a sidecar database or a git
//	          baseline — read here, cleared by the vendor's own command
//	          (Grok, Codex)
//	none      the CLI has no native memory (Pi, OpenCode)
//	unknown   PiCode cannot confirm one, and says so (Antigravity)
//
// Memory is user data. It is read on demand, never copied into SQLite, never
// published on the change feed, and secret-shaped runs are masked on read.
package climemory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// Paths locates the stores a request touches. Home empty → os.UserHomeDir.
// Cwd is the workspace folder; empty drops every workspace-scoped store.
type Paths struct {
	Home string
	Cwd  string
	// Setting reads one of the CLI's own settings, so a store the user moved
	// is found where they moved it (Claude Code's `autoMemoryDirectory`). The
	// server wires it to internal/clisettings; nil means "use the defaults".
	Setting func(cli, key string) (string, bool)
}

func (p Paths) home() string {
	if p.Home != "" {
		return p.Home
	}
	h, _ := os.UserHomeDir()
	return h
}

// Tier is how much PiCode may do with one CLI's memory.
type Tier string

const (
	TierEditable Tier = "editable"
	TierReadOnly Tier = "readonly"
	TierNone     Tier = "none"
	TierUnknown  Tier = "unknown"
)

// Store is one folder a CLI keeps memory in. Resolved false means PiCode could
// not map this workspace to a store and is saying so rather than showing
// another project's memory.
type Store struct {
	Scope    string `json:"scope"`
	Label    string `json:"label"`
	Path     string `json:"path,omitempty"`
	Exists   bool   `json:"exists"`
	Resolved bool   `json:"resolved"`
	Note     string `json:"note,omitempty"`
	Items    int    `json:"items"`
}

// Item is one memory file. Beyond what the file says about itself, an item
// carries what only the whole folder can answer: whether the index the CLI
// loads at session start still points at it, how many other memories cite it,
// and which of its own citations lead nowhere (2026-09-20).
type Item struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Kind     string `json:"kind,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Bytes    int64  `json:"bytes"`
	Modified string `json:"modified"`
	// ModifiedFrom says which clock this is: "memory" when the CLI wrote the
	// time into the file itself, "file" when it is only the filesystem's.
	ModifiedFrom string   `json:"modifiedFrom,omitempty"`
	Session      string   `json:"session,omitempty"`
	Index        bool     `json:"index,omitempty"`
	Generated    bool     `json:"generated,omitempty"`
	Indexed      *bool    `json:"indexed,omitempty"`
	CitedBy      *int     `json:"citedBy,omitempty"`
	Broken       []string `json:"broken,omitempty"`
}

// IndexHealth measures the file the CLI loads at the start of every session
// against the limits that CLI actually applies. Claude Code reads the first
// 200 lines or 25 KB of MEMORY.md, whichever comes first, and silently drops
// the rest — a limit a reader cannot see from the CLI itself.
type IndexHealth struct {
	Exists    bool     `json:"exists"`
	Lines     int      `json:"lines"`
	Bytes     int      `json:"bytes"`
	LineLimit int      `json:"lineLimit,omitempty"`
	ByteLimit int      `json:"byteLimit,omitempty"`
	Dangling  []string `json:"dangling,omitempty"`
}

// Report is what the Memory pane renders.
type Report struct {
	CLI    string  `json:"cli"`
	Tier   Tier    `json:"tier"`
	Note   string  `json:"note,omitempty"`
	Toggle string  `json:"toggle,omitempty"`
	Clear  string  `json:"clear,omitempty"`
	Stores []Store `json:"stores"`
}

// Limits keep one folder from becoming an unbounded read.
const (
	maxItems     = 500
	maxFileSize  = 512 << 10
	summaryRunes = 180
)

type storeSpec struct {
	scope string
	label string
	// dir resolves the folder, "" when this store does not apply here, and
	// note explains an unresolved one in a single line.
	dir func(Paths) (string, string)
}

type spec struct {
	id     string
	tier   Tier
	note   string
	toggle string
	clear  string
	stores []storeSpec
	// indexLinks marks a CLI whose index names its memories with links, which
	// is what makes "in the index" and "cited by" answerable. Where a CLI has
	// no such convention the columns are absent, not zero.
	indexLinks bool
	// The limits that CLI applies to the index it loads at session start.
	indexLines int
	indexBytes int
}

// For returns the declaration for cli, or nil when PiCode has none.
func For(cli string) *spec {
	for i := range catalog {
		if catalog[i].id == cli {
			return &catalog[i]
		}
	}
	return nil
}

// Supported lists every CLI with a declaration, in catalog order.
func Supported() []string {
	out := make([]string, 0, len(catalog))
	for _, s := range catalog {
		out = append(out, s.id)
	}
	return out
}

// Describe reports the tier and the stores, with a count per store. It never
// reads a memory's text.
func Describe(cli string, p Paths) (Report, error) {
	s := For(cli)
	if s == nil {
		return Report{}, fmt.Errorf("PiCode has no memory driver for %q", cli)
	}
	rep := Report{CLI: cli, Tier: s.tier, Note: s.note, Toggle: s.toggle, Clear: s.clear}
	for _, ss := range s.stores {
		dir, note := ss.dir(p)
		store := Store{Scope: ss.scope, Label: ss.label, Path: dir, Note: note, Resolved: dir != ""}
		if dir != "" {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				store.Exists = true
				if n, err := count(dir); err == nil {
					// The real number of memories, not the capped list length:
					// a 20 000-file store reported exactly 500 as a fact.
					store.Items = n
				}
			}
		}
		rep.Stores = append(rep.Stores, store)
	}
	return rep, nil
}

// List returns the items of one store, newest first. The index file, when the
// store has one, sorts to the top: it is what the CLI loads every session.
func List(cli string, p Paths, scope string) ([]Item, error) {
	items, _, err := Survey(cli, p, scope)
	return items, err
}

// Survey lists a store and, where the CLI declares an index convention, the
// citation graph and the index's own health. A memory nothing points at, a
// citation that leads nowhere and an index past the size its CLI will read are
// all invisible from the file alone.
// Audit is what the folder says that no single file can: the index's health
// where the CLI declares one, and the store's own revision history where the
// CLI keeps one. A CLI that answers neither returns a zero Audit rather than
// zeroed numbers, so the pane can drop the column instead of printing a zero
// that reads as a finding.
type Audit struct {
	Index   IndexHealth `json:"index"`
	History *History    `json:"history,omitempty"`
}

func Survey(cli string, p Paths, scope string) ([]Item, Audit, error) {
	s := For(cli)
	if s == nil {
		return nil, Audit{}, fmt.Errorf("PiCode has no memory driver for %q", cli)
	}
	dir, err := storeDir(cli, p, scope)
	if err != nil {
		return nil, Audit{}, err
	}
	items, err := list(dir)
	if err != nil {
		return nil, Audit{}, err
	}
	out := Audit{History: history(dir)}
	if s.indexLinks {
		// Without a declared index convention, counting citations would report
		// zero for every row, which reads as a finding rather than an absence.
		items, out.Index, err = survey(dir, items, s)
		if err != nil {
			return nil, Audit{}, err
		}
	}
	return out.History.stamp(items), out, nil
}

// Read returns one memory's text. id is a path relative to the store and is
// resolved inside it: a traversal, a symlink out, or a file the store does not
// contain is refused.
func Read(cli string, p Paths, scope, id string) (string, error) {
	dir, err := storeDir(cli, p, scope)
	if err != nil {
		return "", err
	}
	path, err := within(dir, id)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		// A raw stat error prints the whole absolute path back at the reader.
		// The store's path is already named in the pane; the missing item is
		// named by the id the caller used.
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s is not in this memory folder", filepath.Base(id))
		}
		return "", fmt.Errorf("%s could not be read", filepath.Base(id))
	}
	// A pseudo-file and a device node both report size 0, which walked
	// straight past the size guard below (adversarial review, 2026-09-20).
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a memory file", filepath.Base(id))
	}
	if info.Size() > maxFileSize {
		return "", fmt.Errorf("this memory is %d KB, larger than PiCode shows here; open it in Files", info.Size()>>10)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s could not be read", filepath.Base(id))
	}
	return mask(string(body)), nil
}

// Write replaces one memory. Only the editable tier accepts it: a generated
// index belongs to the CLI that writes it.
func Write(cli string, p Paths, scope, id, body string) error {
	s := For(cli)
	if s == nil {
		return fmt.Errorf("PiCode has no memory driver for %q", cli)
	}
	if s.tier != TierEditable {
		return fmt.Errorf("%s keeps this memory itself; PiCode reads it and does not write it", cli)
	}
	dir, err := storeDir(cli, p, scope)
	if err != nil {
		return err
	}
	path, err := within(dir, id)
	if err != nil {
		return err
	}
	// The guard was inverted: it read the replacement, so a file whose own
	// first lines say the CLI generates it was overwritten happily
	// (adversarial review, 2026-09-20). Check what is on disk.
	if head, ok := peek(path); ok && generatedBanner(head) {
		return fmt.Errorf("%s says it is generated by %s; PiCode will not write it", filepath.Base(id), cli)
	}
	if len(body) > maxFileSize {
		return fmt.Errorf("a memory PiCode writes is at most %d KB", maxFileSize>>10)
	}
	return writeAtomic(path, []byte(body))
}

// Delete removes one memory. Editable tier only, and never the index: the CLI
// rebuilds an index, and deleting it loses the map to everything else.
func Delete(cli string, p Paths, scope, id string) error {
	s := For(cli)
	if s == nil {
		return fmt.Errorf("PiCode has no memory driver for %q", cli)
	}
	if s.tier != TierEditable {
		if s.clear == "" {
			return fmt.Errorf("%s keeps this memory itself", cli)
		}
		return fmt.Errorf("%s keeps this memory itself; clear it with: %s", cli, s.clear)
	}
	dir, err := storeDir(cli, p, scope)
	if err != nil {
		return err
	}
	path, err := within(dir, id)
	if err != nil {
		return err
	}
	if filepath.Clean(path) == filepath.Clean(dir) {
		return fmt.Errorf("that is the memory folder itself, not a memory in it")
	}
	if isIndex(filepath.Base(path)) {
		return fmt.Errorf("%s is the index of this folder; empty it instead of deleting it", filepath.Base(path))
	}
	if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a memory file", filepath.Base(id))
	}
	return os.Remove(path)
}

func storeDir(cli string, p Paths, scope string) (string, error) {
	rep, err := Describe(cli, p)
	if err != nil {
		return "", err
	}
	for _, store := range rep.Stores {
		if store.Scope != scope {
			continue
		}
		if !store.Resolved {
			return "", fmt.Errorf("%s", firstLine(store.Note, "PiCode cannot tell which memory folder belongs to this workspace"))
		}
		if !store.Exists {
			return "", fmt.Errorf("%s has not written any memory here yet", cli)
		}
		return store.Path, nil
	}
	return "", fmt.Errorf("%s has no %q memory", cli, scope)
}

func firstLine(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// within resolves id inside dir and refuses anything that escapes it.
func within(dir, id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("no memory named")
	}
	clean := filepath.Clean("/" + filepath.FromSlash(id))
	path := filepath.Join(dir, clean)
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is outside this memory folder", id)
	}
	// Resolve the deepest existing ancestor, not the file: a file being
	// created does not exist yet, so resolving the file itself skipped the
	// check entirely and a write through a symlinked subdirectory landed
	// outside the store (adversarial review, 2026-09-20).
	base, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", fmt.Errorf("this memory folder cannot be resolved")
	}
	probe := path
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			if rel, err := filepath.Rel(base, resolved); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return "", fmt.Errorf("%s points outside this memory folder", id)
			}
			break
		}
		parent := filepath.Dir(probe)
		if parent == probe || len(parent) < len(dir) {
			// Walked above the store without resolving anything: the store
			// itself resolved, so this cannot escape.
			break
		}
		probe = parent
	}
	return path, nil
}

func list(dir string) ([]Item, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || len(items) >= maxItems {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		// Only ordinary files. A symlink named `notes.md` pointing outside was
		// opened and its first line published as a summary, and a FIFO made
		// the whole pane hang on os.Open (adversarial review, 2026-09-20).
		if !info.Mode().IsRegular() {
			continue
		}
		item := Item{
			ID:           name,
			Title:        titleOf(name),
			Bytes:        info.Size(),
			Modified:     info.ModTime().UTC().Format(time.RFC3339),
			ModifiedFrom: "file",
			Index:        isIndex(name),
		}
		if head, ok := peek(filepath.Join(dir, name)); ok {
			var modified, session string
			item.Kind, item.Summary, modified, session, item.Generated = describeHead(head)
			item.Session = session
			// The CLI's own timestamp beats the filesystem's: a file copied or
			// touched keeps the time the memory was actually written.
			if modified != "" {
				item.Modified, item.ModifiedFrom = modified, "memory"
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Index != items[j].Index {
			return items[i].Index
		}
		if items[i].Modified != items[j].Modified {
			return items[i].Modified > items[j].Modified
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

// peek reads the first bytes of a memory file: enough for frontmatter and a
// first line, never the whole corpus.
func peek(path string) (string, bool) {
	if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", false
	}
	defer f.Close()
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	if n <= 0 {
		return "", false
	}
	return string(buf[:n]), true
}

// describeHead pulls the kind and a one-line summary out of a memory file.
// Claude Code writes YAML frontmatter with `description` and a `type` under
// `metadata`; the other CLIs write plain markdown, where the first heading and
// the first paragraph carry the same information.
func describeHead(head string) (kind, summary, modified, session string, generated bool) {
	generated = generatedBanner(head)
	body := head
	if strings.HasPrefix(head, "---") {
		if end := strings.Index(head[3:], "\n---"); end >= 0 {
			front := head[3 : 3+end]
			body = head[3+end+4:]
			for _, line := range strings.Split(front, "\n") {
				trimmed := strings.TrimSpace(line)
				name, value, found := strings.Cut(trimmed, ":")
				if !found {
					continue
				}
				value = strings.TrimSpace(value)
				switch strings.TrimSpace(name) {
				case "type":
					kind = clip(mask(value))
				case "description":
					if summary == "" {
						summary = value
					}
				case "modified":
					modified = strings.Trim(value, `"'`)
				case "originSessionId":
					session = strings.Trim(value, `"'`)
				}
			}
		}
	}
	if summary == "" {
		for _, line := range strings.Split(body, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ">") {
				continue
			}
			summary = trimmed
			break
		}
	}
	return kind, clip(mask(summary)), modified, session, generated
}

// generatedBanner reports the "do not edit" marker Grok and Codex put at the
// top of an index they rebuild.
func generatedBanner(head string) bool {
	lower := strings.ToLower(head)
	if len(lower) > 600 {
		lower = lower[:600]
	}
	return strings.Contains(lower, "do not edit this file") || strings.Contains(lower, "generated by grok")
}

// indexName is the file every store with an index convention uses.
const indexName = "MEMORY.md"

func isIndex(name string) bool { return strings.EqualFold(name, indexName) }

func titleOf(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if isIndex(name) {
		return "Index"
	}
	base = strings.ReplaceAll(strings.ReplaceAll(base, "-", " "), "_", " ")
	if base == "" {
		return name
	}
	return strings.ToUpper(base[:1]) + base[1:]
}

func clip(s string) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= summaryRunes {
		return string(runes)
	}
	return strings.TrimSpace(string(runes[:summaryRunes])) + "…"
}

func writeAtomic(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".picode-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, mode); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// count reports how many memories a store holds, without reading any of them.
func count(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		if info, err := e.Info(); err == nil && info.Mode().IsRegular() {
			n++
		}
	}
	return n, nil
}
