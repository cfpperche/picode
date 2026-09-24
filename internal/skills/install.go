package skills

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Conflict is a refusal the person can answer: the code says which question
// the pane asks (ADR-0196's decision table, docs/plans/skills.md).
type Conflict struct {
	Code    string `json:"code"` // exists | update | critical | modified | unlocked | invalid | stale | gone | lock-version
	Message string `json:"message"`
}

func (c *Conflict) Error() string { return c.Message }

func conflict(code, format string, a ...any) error {
	return &Conflict{Code: code, Message: fmt.Sprintf(format, a...)}
}

// ErrNotInstalled is a remove or update of a skill that is not there.
var ErrNotInstalled = errors.New("that skill is not installed there")

// Candidate is one skill a staged source carries.
type Candidate struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Path        string    `json:"path"`      // folder inside the source, slash-separated ("" = the source root)
	SkillPath   string    `json:"skillPath"` // path/SKILL.md, as the lock records it
	Digest      string    `json:"digest"`
	Files       []string  `json:"files"`
	Size        int64     `json:"size"`
	Problems    []string  `json:"problems,omitempty"`
	Findings    []Finding `json:"findings,omitempty"`
}

// Preview is a staged source waiting for consent.
type Preview struct {
	ID         string      `json:"id"`
	Source     Source      `json:"source"`
	Candidates []Candidate `json:"candidates"`
	Notes      []string    `json:"notes,omitempty"`
}

type staged struct {
	Preview
	dir     string
	created time.Time
}

// Manager installs, updates and removes skills. The folders and the locks
// are the truth; the manager keeps only previews in flight.
type Manager struct {
	StageRoot string
	// CacheRoot holds agent skills by digest: <CacheRoot>/<digest>/<name>
	// (slice 4). An agent's list names the folder; the folder never changes
	// once written, so two agents with the same content share one copy.
	CacheRoot string
	Fetcher   *Fetcher
	Home      string // defaults to the user's home
	Now       func() time.Time

	mu     sync.Mutex // one write at a time: the locks are shared files
	stages map[string]*staged
}

func NewManager(stageRoot string) *Manager {
	return &Manager{StageRoot: stageRoot, Fetcher: NewFetcher(), Now: time.Now, stages: map[string]*staged{}}
}

const stageTTL = 30 * time.Minute

func (m *Manager) home() string {
	if m.Home != "" {
		return m.Home
	}
	h, _ := os.UserHomeDir()
	return h
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (m *Manager) sweep() {
	for id, s := range m.stages {
		if m.Now().Sub(s.created) > stageTTL {
			_ = os.RemoveAll(s.dir)
			delete(m.stages, id)
		}
	}
}

// Preview fetches a source into the stage and lists the skills it carries,
// each validated, hashed and scanned. Nothing is installed.
func (m *Manager) Preview(ctx context.Context, input string) (Preview, error) {
	src, err := ParseSource(input, m.home())
	if err != nil {
		return Preview{}, err
	}
	if err := os.MkdirAll(m.StageRoot, 0o700); err != nil {
		return Preview{}, err
	}
	id := randomID()
	dir := filepath.Join(m.StageRoot, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Preview{}, err
	}
	notes, err := m.Fetcher.Fetch(ctx, src, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return Preview{}, err
	}
	cands := discover(dir, src.Sub)
	if len(cands) == 0 {
		_ = os.RemoveAll(dir)
		where := src.Input
		return Preview{}, fmt.Errorf("%s has no SKILL.md", where)
	}
	p := Preview{ID: id, Source: src, Candidates: cands, Notes: notes}
	m.mu.Lock()
	m.sweep()
	m.stages[id] = &staged{Preview: p, dir: dir, created: m.Now()}
	m.mu.Unlock()
	return p, nil
}

// discover lists every folder with a SKILL.md under root (under sub, when
// the source named one), validated, hashed and scanned.
func discover(root, sub string) []Candidate {
	var out []Candidate
	start := root
	if sub != "" {
		start = filepath.Join(root, filepath.FromSlash(sub))
	}
	_ = filepath.WalkDir(start, func(p string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if d.Name() == ".git" || d.Name() == "node_modules" {
			return filepath.SkipDir
		}
		if _, err := os.Stat(filepath.Join(p, "SKILL.md")); err != nil {
			return nil
		}
		if len(out) >= 200 {
			return filepath.SkipAll
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}
		fm, hasHeader := readFrontmatter(filepath.Join(p, "SKILL.md"))
		folder := filepath.Base(p)
		c := Candidate{Name: fm.Name, Description: fm.Description, Path: rel, SkillPath: path.Join(rel, "SKILL.md")}
		if c.Name == "" {
			c.Name = folder
		}
		if rel == "" {
			// A source whose root is the skill: the folder name is the
			// staging id, so the header's name is the only name there is.
			folder = c.Name
		}
		c.Problems = problems(fm, hasHeader, folder)
		c.Digest, _ = Digest(p)
		if c.Digest == "" {
			c.Problems = append(c.Problems, "larger than 1,000 files or 25 MiB")
		}
		_ = filepath.WalkDir(p, func(f string, e os.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return nil
			}
			if info, err := e.Info(); err == nil {
				c.Size += info.Size()
			}
			if len(c.Files) < 200 {
				r, _ := filepath.Rel(p, f)
				c.Files = append(c.Files, filepath.ToSlash(r))
			}
			return nil
		})
		sort.Strings(c.Files)
		c.Findings = Scan(p)
		out = append(out, c)
		return filepath.SkipDir
	})
	return out
}

// Target is where an install lands.
type Target struct {
	Scope     Scope
	Workspace string
}

func (m *Manager) base(t Target) (string, error) {
	switch t.Scope {
	case Workspace:
		if t.Workspace == "" {
			return "", errors.New("a workspace install needs a workspace")
		}
		return t.Workspace, nil
	case Machine:
		h := m.home()
		if h == "" {
			return "", errors.New("no home folder")
		}
		return h, nil
	}
	return "", fmt.Errorf("unknown scope %q", t.Scope)
}

// canonicalDir is the one folder every CLI that reads .agents/skills sees.
func (m *Manager) canonicalDir(t Target, name string) (string, error) {
	b, err := m.base(t)
	if err != nil {
		return "", err
	}
	return filepath.Join(b, ".agents", "skills", name), nil
}

// linkDirs are the folders of CLIs that do not read .agents/skills at this
// scope, derived from the declarations: Claude Code's .claude/skills in a
// workspace; on the machine, Claude Code's, Hermes' and Antigravity's user
// folders — only for a CLI whose home folder exists, so an install never
// creates a folder for a CLI that is not there.
func (m *Manager) linkDirs(t Target) []string {
	b, _ := m.base(t)
	seen := map[string]bool{}
	var out []string
	for _, s := range specs {
		readsCanonical := false
		first := ""
		for _, r := range s.Roots {
			if r.Scope != t.Scope {
				continue
			}
			p := r.Path
			if t.Scope == Machine {
				if !strings.HasPrefix(p, "~/") {
					continue
				}
				p = strings.TrimPrefix(p, "~/")
			}
			if p == ".agents/skills" {
				readsCanonical = true
			}
			if first == "" {
				first = p
			}
		}
		if readsCanonical || first == "" {
			continue
		}
		if t.Scope == Machine {
			top := strings.SplitN(first, "/", 2)[0]
			if st, err := os.Stat(filepath.Join(b, top)); err != nil || !st.IsDir() {
				continue
			}
		}
		d := filepath.Join(b, filepath.FromSlash(first))
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

// --- locks -----------------------------------------------------------------

// lockFile is either installer's lock, read so unknown fields survive.
type lockFile struct {
	path    string
	version int
	top     map[string]json.RawMessage
	skills  map[string]map[string]any
	rev     string // sha256 of the bytes read; "" when the file was absent
}

func (m *Manager) lockPath(t Target) string {
	b, _ := m.base(t)
	if t.Scope == Workspace {
		return filepath.Join(b, "skills-lock.json")
	}
	return filepath.Join(b, ".agents", ".skill-lock.json")
}

func lockVersion(t Target) int {
	if t.Scope == Workspace {
		return 1
	}
	return 3
}

func readLock(p string, want int) (*lockFile, error) {
	lf := &lockFile{path: p, version: want, top: map[string]json.RawMessage{}, skills: map[string]map[string]any{}}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return lf, nil
	}
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(b)
	lf.rev = hex.EncodeToString(sum[:])
	if err := json.Unmarshal(b, &lf.top); err != nil {
		return nil, conflict("lock-version", "%s is not valid JSON; PiCode will not rewrite it", filepath.Base(p))
	}
	if raw, ok := lf.top["version"]; ok {
		_ = json.Unmarshal(raw, &lf.version)
	}
	if lf.version != want {
		return nil, conflict("lock-version", "%s is version %d; PiCode writes version %d only and never migrates a lock", filepath.Base(p), lf.version, want)
	}
	if raw, ok := lf.top["skills"]; ok {
		if err := json.Unmarshal(raw, &lf.skills); err != nil {
			return nil, conflict("lock-version", "%s has a skills map PiCode cannot read", filepath.Base(p))
		}
	}
	return lf, nil
}

func (lf *lockFile) save() error {
	// Re-read before write: another tool may have written it since.
	if b, err := os.ReadFile(lf.path); err == nil {
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != lf.rev {
			return conflict("stale", "%s changed while PiCode was writing it; read it again", filepath.Base(lf.path))
		}
	} else if !os.IsNotExist(err) || lf.rev != "" {
		return conflict("stale", "%s changed while PiCode was writing it; read it again", filepath.Base(lf.path))
	}
	v, _ := json.Marshal(lf.version)
	lf.top["version"] = v
	s, err := json.Marshal(lf.skills) // map keys are sorted: merge-friendly
	if err != nil {
		return err
	}
	lf.top["skills"] = s
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(lf.top); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(lf.path), 0o755); err != nil {
		return err
	}
	tmp := lf.path + ".picode-tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, lf.path)
}

func str(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return s
}

// lockHash is the content hash an entry records: computedHash (the project
// lock's field, and the one PiCode adds to the global entries it writes).
func lockHash(e map[string]any) string { return str(e, "computedHash") }

func (m *Manager) entry(t Target, src Source, c Candidate, prev map[string]any) map[string]any {
	source, sourceType, sourceURL := src.lockSource()
	e := map[string]any{}
	for k, v := range prev {
		e[k] = v
	}
	e["source"], e["sourceType"] = source, sourceType
	if sourceURL != "" {
		e["sourceUrl"] = sourceURL
	}
	if src.Ref != "" {
		e["ref"] = src.Ref
	}
	if src.Kind != "local" {
		e["skillPath"] = c.SkillPath
	}
	e["computedHash"] = c.Digest
	if t.Scope == Machine {
		now := m.Now().UTC().Format(time.RFC3339Nano)
		if _, ok := e["installedAt"]; !ok {
			e["installedAt"] = now
		}
		e["updatedAt"] = now
		// The skills CLI compares a GitHub tree SHA here; PiCode does not fetch
		// one, and an empty value only makes that CLI offer an update.
		if _, ok := e["skillFolderHash"]; !ok {
			e["skillFolderHash"] = ""
		}
	}
	return e
}

// --- placing folders --------------------------------------------------------

// place puts src's copy at dst atomically: a sibling temp folder renamed over
// the target (the old folder moved aside first, then deleted).
func place(srcDir, dst string) error {
	parent := filepath.Dir(dst)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(parent, ".picode-"+randomID())
	if _, err := copyTree(srcDir, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		old := filepath.Join(parent, ".picode-old-"+randomID())
		if err := os.Rename(dst, old); err != nil {
			_ = os.RemoveAll(tmp)
			return err
		}
		if err := os.Rename(tmp, dst); err != nil {
			_ = os.Rename(old, dst)
			_ = os.RemoveAll(tmp)
			return err
		}
		return os.RemoveAll(old)
	}
	return os.Rename(tmp, dst)
}

// link points dir/name at canonical with a relative link, or copies it when
// links fail. An unrelated folder already there is left alone and named.
func link(canonical, dir, name string) (string, string) {
	at := filepath.Join(dir, name)
	if fi, err := os.Lstat(at); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			if target, err := filepath.EvalSymlinks(at); err == nil && target == mustEval(canonical) {
				return at, ""
			}
			_ = os.Remove(at)
		} else {
			d1, _ := Digest(at)
			d2, _ := Digest(canonical)
			if d1 != "" && d1 == d2 {
				return at, ""
			}
			return "", "a different " + name + " already exists in " + dir + "; left as it is"
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "could not create " + dir + ": " + err.Error()
	}
	rel, err := filepath.Rel(dir, canonical)
	if err == nil && os.Symlink(rel, at) == nil {
		return at, ""
	}
	if err := place(canonical, at); err != nil {
		return "", "could not link or copy into " + dir + ": " + err.Error()
	}
	return at, "copied into " + dir + " (links are not available here)"
}

func mustEval(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// unlink removes dir/name when it is PiCode's link to canonical or a copy with
// the same content.
func unlink(canonical, canonicalDigest, dir, name string) string {
	at := filepath.Join(dir, name)
	fi, err := os.Lstat(at)
	if err != nil {
		return ""
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		if t, err := os.Readlink(at); err == nil {
			if !filepath.IsAbs(t) {
				t = filepath.Join(dir, t)
			}
			if filepath.Clean(t) == filepath.Clean(canonical) {
				_ = os.Remove(at)
				return ""
			}
		}
		return "left " + at + ": it points elsewhere"
	}
	if d, _ := Digest(at); d != "" && d == canonicalDigest {
		_ = os.RemoveAll(at)
		return ""
	}
	return "left " + at + ": its content differs"
}

// --- verbs ------------------------------------------------------------------

// Result says what a verb did.
type Result struct {
	Status string   `json:"status"` // installed | already | adopted | replaced | updated | current | removed
	Name   string   `json:"name"`
	Dir    string   `json:"dir,omitempty"`
	Links  []string `json:"links,omitempty"`
	Notes  []string `json:"notes,omitempty"`
	// Digest and Source: an agent install, for the agent's list.
	Digest string `json:"digest,omitempty"`
	Source string `json:"source,omitempty"`
}

// InstallReq installs one candidate of a preview.
type InstallReq struct {
	Preview        string `json:"preview"`
	Path           string `json:"path"`
	Scope          Scope  `json:"scope"`
	Workspace      string `json:"-"`
	Replace        bool   `json:"replace"`
	Adopt          bool   `json:"adopt"`
	AcceptCritical bool   `json:"acceptCritical"`
}

func (m *Manager) Install(req InstallReq) (Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.stages[req.Preview]
	if !ok || m.Now().Sub(st.created) > stageTTL {
		return Result{}, conflict("gone", "this preview expired; look at the source again")
	}
	var cand *Candidate
	for i := range st.Candidates {
		if st.Candidates[i].Path == req.Path {
			cand = &st.Candidates[i]
		}
	}
	if cand == nil {
		return Result{}, conflict("invalid", "the preview has no skill at %q", req.Path)
	}
	if len(cand.Problems) > 0 {
		return Result{}, conflict("invalid", "%s breaks the skill format: %s", cand.Name, strings.Join(cand.Problems, "; "))
	}
	if hasCritical(cand.Findings) && !req.AcceptCritical {
		return Result{}, conflict("critical", "%s has critical findings; confirm that you reviewed its files", cand.Name)
	}
	if req.Scope == Agent {
		return m.cache(st, cand)
	}
	t := Target{Scope: req.Scope, Workspace: req.Workspace}
	dst, err := m.canonicalDir(t, cand.Name)
	if err != nil {
		return Result{}, err
	}
	lf, err := readLock(m.lockPath(t), lockVersion(t))
	if err != nil {
		return Result{}, err
	}
	prev, inLock := lf.skills[cand.Name]
	res := Result{Name: cand.Name, Dir: dst, Status: "installed"}
	if _, err := os.Stat(dst); err == nil {
		cur, _ := Digest(dst)
		switch {
		case cur == cand.Digest:
			res.Status = "already"
		case req.Adopt && !inLock:
			res.Status = "adopted"
		case req.Replace:
			res.Status = "replaced"
		case inLock:
			return Result{}, conflict("update", "%s is already installed from %s with other content; update it instead", cand.Name, str(prev, "source"))
		default:
			return Result{}, conflict("exists", "A %s folder that no installer recorded is already in %s. Keep it and record where this one comes from, or replace it.", cand.Name, filepath.Dir(dst))
		}
	}
	srcDir := filepath.Join(st.dir, filepath.FromSlash(cand.Path))
	if res.Status == "installed" || res.Status == "replaced" {
		if err := place(srcDir, dst); err != nil {
			return Result{}, err
		}
	}
	if res.Status != "adopted" {
		for _, d := range m.linkDirs(t) {
			at, note := link(dst, d, cand.Name)
			if at != "" {
				res.Links = append(res.Links, at)
			}
			if note != "" {
				res.Notes = append(res.Notes, note)
			}
		}
	}
	c := *cand
	if res.Status == "adopted" {
		c.Digest, _ = Digest(dst)
	}
	lf.skills[cand.Name] = m.entry(t, st.Source, c, prev)
	if err := lf.save(); err != nil {
		return res, err
	}
	return res, nil
}

// cache writes a staged skill into the digest-addressed cache for an
// agent's list. No lock, no link: the agent's CLI receives the folder at
// launch, and nothing else reads it.
func (m *Manager) cache(st *staged, cand *Candidate) (Result, error) {
	dst, err := CacheDir(m.CacheRoot, cand.Digest, cand.Name)
	if err != nil {
		return Result{}, err
	}
	res := Result{Name: cand.Name, Dir: dst, Status: "installed", Digest: cand.Digest, Source: st.Source.Input}
	if cur, err := Digest(dst); err == nil && cur == cand.Digest {
		return res, nil
	}
	if err := place(filepath.Join(st.dir, filepath.FromSlash(cand.Path)), dst); err != nil {
		return Result{}, err
	}
	return res, nil
}

var hexDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)

// CacheDir is where an agent skill of this content and name lives.
func CacheDir(root, digest, name string) (string, error) {
	if root == "" {
		return "", errors.New("no skill cache configured")
	}
	if !hexDigest.MatchString(digest) || !validName(name) {
		return "", conflict("invalid", "%q is not a cacheable skill", name)
	}
	return filepath.Join(root, digest, name), nil
}

// RemoveReq removes one installed skill.
type RemoveReq struct {
	Name      string `json:"name"`
	Scope     Scope  `json:"scope"`
	Workspace string `json:"-"`
	Confirm   bool   `json:"confirm"`
}

func (m *Manager) Remove(req RemoveReq) (Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !validName(req.Name) {
		return Result{}, conflict("invalid", "%q is not a skill name", req.Name)
	}
	t := Target{Scope: req.Scope, Workspace: req.Workspace}
	dst, err := m.canonicalDir(t, req.Name)
	if err != nil {
		return Result{}, err
	}
	if _, err := os.Stat(dst); err != nil {
		return Result{}, ErrNotInstalled
	}
	lf, err := readLock(m.lockPath(t), lockVersion(t))
	if err != nil {
		return Result{}, err
	}
	e, inLock := lf.skills[req.Name]
	cur, _ := Digest(dst)
	if !req.Confirm {
		if !inLock {
			return Result{}, conflict("unlocked", "neither PiCode nor the skills CLI installed %s; remove it anyway?", req.Name)
		}
		if h := lockHash(e); h != "" && h != cur {
			return Result{}, conflict("modified", "%s changed since it was installed; remove it anyway?", req.Name)
		}
	}
	res := Result{Name: req.Name, Dir: dst, Status: "removed"}
	for _, d := range m.linkDirs(t) {
		if note := unlink(dst, cur, d, req.Name); note != "" {
			res.Notes = append(res.Notes, note)
		}
	}
	if err := os.RemoveAll(dst); err != nil {
		return res, err
	}
	if inLock {
		delete(lf.skills, req.Name)
		if err := lf.save(); err != nil {
			return res, err
		}
	}
	return res, nil
}

// sourceOf rebuilds the Source a lock entry names.
func sourceOf(e map[string]any) (Source, string, error) {
	sp := str(e, "skillPath")
	sub := strings.TrimSuffix(strings.TrimSuffix(sp, "SKILL.md"), "/")
	switch str(e, "sourceType") {
	case "github":
		parts := strings.SplitN(str(e, "source"), "/", 2)
		if len(parts) != 2 {
			return Source{}, "", fmt.Errorf("the lock's source %q is not owner/repo", str(e, "source"))
		}
		return Source{Kind: "github", Owner: parts[0], Repo: parts[1], Ref: str(e, "ref"), Input: str(e, "source")}, sub, nil
	case "well-known":
		return Source{Kind: "well-known", Origin: "https://" + str(e, "source"), Input: str(e, "source")}, sub, nil
	case "local":
		return Source{Kind: "local", Dir: str(e, "source"), Input: str(e, "source")}, "", nil
	}
	return Source{}, "", fmt.Errorf("PiCode cannot fetch a %q source", str(e, "sourceType"))
}

// fetched is one source downloaded into the stage.
type fetched struct {
	root  string
	cands []Candidate
	err   error
}

func (m *Manager) fetchEntry(ctx context.Context, e map[string]any) (*fetched, func()) {
	src, _, err := sourceOf(e)
	if err != nil {
		return &fetched{err: err}, func() {}
	}
	if err := os.MkdirAll(m.StageRoot, 0o700); err != nil {
		return &fetched{err: err}, func() {}
	}
	dir := filepath.Join(m.StageRoot, "u-"+randomID())
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return &fetched{err: err}, cleanup
	}
	if _, err := m.Fetcher.Fetch(ctx, src, dir); err != nil {
		return &fetched{err: err}, cleanup
	}
	return &fetched{root: dir, cands: discover(dir, "")}, cleanup
}

// pick finds the entry's skill in a fetched source: by its recorded path,
// else by name.
func (f *fetched) pick(name string, e map[string]any) (Candidate, string, error) {
	if f.err != nil {
		return Candidate{}, "", f.err
	}
	_, sub, _ := sourceOf(e)
	for _, c := range f.cands {
		if (sub != "" && c.Path == sub) || (sub == "" && c.Name == name) {
			return c, filepath.Join(f.root, filepath.FromSlash(c.Path)), nil
		}
	}
	return Candidate{}, "", fmt.Errorf("%s no longer carries %s", str(e, "source"), name)
}

// latest fetches what the entry's source carries now.
func (m *Manager) latest(ctx context.Context, name string, e map[string]any) (Candidate, string, func(), error) {
	f, cleanup := m.fetchEntry(ctx, e)
	c, dir, err := f.pick(name, e)
	return c, dir, cleanup, err
}

// UpdateReq refreshes one skill from the source its lock names.
type UpdateReq struct {
	Name      string `json:"name"`
	Scope     Scope  `json:"scope"`
	Workspace string `json:"-"`
	Force     bool   `json:"force"`
}

func (m *Manager) Update(ctx context.Context, req UpdateReq) (Result, error) {
	t := Target{Scope: req.Scope, Workspace: req.Workspace}
	dst, err := m.canonicalDir(t, req.Name)
	if err != nil {
		return Result{}, err
	}
	lf, err := readLock(m.lockPath(t), lockVersion(t))
	if err != nil {
		return Result{}, err
	}
	e, ok := lf.skills[req.Name]
	if !ok {
		return Result{}, conflict("unlocked", "no installer recorded where %s came from, so there is nothing to update from", req.Name)
	}
	if _, err := os.Stat(dst); err != nil {
		return Result{}, ErrNotInstalled
	}
	cand, dir, cleanup, err := m.latest(ctx, req.Name, e)
	defer cleanup()
	if err != nil {
		return Result{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, _ := Digest(dst)
	if cand.Digest == cur {
		return Result{Name: req.Name, Dir: dst, Status: "current"}, nil
	}
	if h := lockHash(e); h != "" && h != cur && !req.Force {
		return Result{}, conflict("modified", "%s changed since it was installed; updating would discard those edits", req.Name)
	}
	if len(cand.Problems) > 0 {
		return Result{}, conflict("invalid", "the new %s breaks the skill format: %s", req.Name, strings.Join(cand.Problems, "; "))
	}
	if err := place(dir, dst); err != nil {
		return Result{}, err
	}
	// The lock file may have moved while the source downloaded.
	lf, err = readLock(m.lockPath(t), lockVersion(t))
	if err != nil {
		return Result{}, err
	}
	src, _, _ := sourceOf(e)
	lf.skills[req.Name] = m.entry(t, src, cand, lf.skills[req.Name])
	res := Result{Name: req.Name, Dir: dst, Status: "updated"}
	if len(cand.Findings) > 0 {
		res.Notes = append(res.Notes, fmt.Sprintf("the new version has %d scan findings; open the skill to read them", len(cand.Findings)))
	}
	return res, lf.save()
}

// UpdateRow is one lock entry compared with its source.
type UpdateRow struct {
	Name   string `json:"name"`
	Scope  Scope  `json:"scope"`
	Source string `json:"source"`
	Status string `json:"status"` // current | behind | modified | unreachable | missing
	Reason string `json:"reason,omitempty"`
}

// Check compares every lock entry installed at t with its source.
func (m *Manager) Check(ctx context.Context, t Target) ([]UpdateRow, error) {
	lf, err := readLock(m.lockPath(t), lockVersion(t))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(lf.skills))
	for n := range lf.skills {
		names = append(names, n)
	}
	sort.Strings(names)
	out := []UpdateRow{}
	cache := map[string]*fetched{}
	for _, n := range names {
		e := lf.skills[n]
		row := UpdateRow{Name: n, Scope: t.Scope, Source: str(e, "source")}
		dst, err := m.canonicalDir(t, n)
		if err != nil {
			return nil, err
		}
		cur, derr := Digest(dst)
		switch {
		case derr != nil:
			row.Status, row.Reason = "missing", "the folder is gone"
		default:
			key := str(e, "sourceType") + "|" + str(e, "source") + "|" + str(e, "ref")
			f, ok := cache[key]
			if !ok {
				var cleanup func()
				f, cleanup = m.fetchEntry(ctx, e)
				defer cleanup()
				cache[key] = f
			}
			cand, _, err := f.pick(n, e)
			switch {
			case err != nil:
				row.Status, row.Reason = "unreachable", err.Error()
			case cand.Digest == cur:
				row.Status = "current"
			case lockHash(e) != "" && lockHash(e) != cur:
				row.Status, row.Reason = "modified", "changed since it was installed"
			default:
				row.Status = "behind"
			}
		}
		out = append(out, row)
	}
	return out, nil
}
