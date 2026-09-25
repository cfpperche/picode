// Package skillcatalog is the Skills Marketplace (ADR-0196 slice 5): one
// catalog for every agent CLI, drawn from seed repositories the owner chose,
// the sources the person adds (GitHub repositories and sites that publish a
// .well-known skills index) and, only when switched on, skills.sh search.
//
// It follows internal/mcpcatalog: the cache on disk answers first, each
// source refreshes in the background once a day, and a source that fails
// keeps its last good list and says why. A card only describes a skill;
// installing goes through the Skills tab's own preview, scan and consent
// (internal/skills), and PiCode never vouches for anything listed here.
package skillcatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/skills"
)

// Seeds are the repositories the Marketplace lists before the person adds
// any: the three vendors' own collections (owner's call, 2026-09-24).
var Seeds = []string{"anthropics/skills", "openai/skills", "vercel-labs/agent-skills"}

// Item is one card.
type Item struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	License     string `json:"license,omitempty"`
	// Source is where it comes from as the card names it: owner/repo or a site.
	Source string `json:"source"`
	// Install is what the Add skill dialog reads: owner/repo/path or a site.
	Install string `json:"install"`
	URL     string `json:"url,omitempty"` // the skill's page at its source
	Origin  string `json:"origin"`        // seed | yours | skills.sh
	// Installs is skills.sh's own count; absent elsewhere.
	Installs int `json:"installs,omitempty"`
}

// SourceState is one source as the pane lists it.
type SourceState struct {
	Input       string `json:"input"`
	Builtin     bool   `json:"builtin"`
	Count       int    `json:"count"`
	RefreshedAt string `json:"refreshedAt,omitempty"`
	Error       string `json:"error,omitempty"`
	Reading     bool   `json:"reading,omitempty"` // no list yet, a read is running
}

const (
	refreshAfter  = 24 * time.Hour
	retryAfter    = 30 * time.Minute // a failed source is not retried on every search
	fetchTimeout  = 3 * time.Minute
	maxSkills     = 300      // per source
	maxSkillBytes = 64 << 10 // one SKILL.md header is read, never the rest
	maxHits       = 200
	workers       = 8
	cacheFileName = "skills-catalog.json"
	cacheVersion  = 1
)

// Defaults are the public endpoints; tests swap them.
const (
	DefaultAPI      = "https://api.github.com"
	DefaultRaw      = "https://raw.githubusercontent.com"
	DefaultSkillsSH = "https://skills.sh"
)

type cached struct {
	Items       []Item    `json:"items"`
	RefreshedAt time.Time `json:"refreshedAt"`
	Error       string    `json:"error,omitempty"`
	TriedAt     time.Time `json:"triedAt"`
}

// Store answers catalog searches. Use NewStore.
type Store struct {
	API, Raw, SkillsSH string
	HTTP               *http.Client
	Now                func() time.Time
	Home               string
	// OnRead is told when a source's read ends, so an open pane refreshes
	// from the feed instead of polling.
	OnRead func(input string)

	dataDir string

	mu      sync.Mutex
	sources map[string]*cached
	reading map[string]bool

	writeMu sync.Mutex
}

// NewStore loads the cache under dataDir ("" keeps it in memory). It never
// fetches on its own; Search starts the reads it needs.
func NewStore(dataDir string) *Store {
	home, _ := os.UserHomeDir()
	s := &Store{API: DefaultAPI, Raw: DefaultRaw, SkillsSH: DefaultSkillsSH, HTTP: skills.PublicClient(), Now: time.Now, Home: home,
		dataDir: dataDir, sources: map[string]*cached{}, reading: map[string]bool{}}
	s.load()
	return s
}

// ValidateSource answers the canonical input for a source the person adds:
// a GitHub repository (optionally a folder in it) or an https site. A local
// folder is refused: it is added with Add skill, not listed.
func ValidateSource(input, home string) (string, error) {
	src, err := skills.ParseSource(strings.TrimSpace(input), home)
	if err != nil {
		return "", err
	}
	switch src.Kind {
	case "github":
		out := src.Owner + "/" + src.Repo
		if src.Sub != "" {
			out += "/" + src.Sub
		}
		if src.Ref != "" {
			out += "#" + src.Ref
		}
		return out, nil
	case "well-known":
		return src.Origin, nil
	}
	return "", errors.New("A folder on this computer is added with Add skill; the Marketplace lists repositories and sites.")
}

// Search answers the cards for q from the sources given (seeds first, then
// the person's), and starts a background read for any source that has none
// yet or is a day old.
func (s *Store) Search(q string, userSources []string) ([]Item, []SourceState) {
	all := append(append([]string{}, Seeds...), userSources...)
	s.mu.Lock()
	var items []Item
	var states []SourceState
	var stale []string
	now := s.Now()
	for i, in := range all {
		c := s.sources[in]
		st := SourceState{Input: in, Builtin: i < len(Seeds)}
		if c != nil {
			st.Count = len(c.Items)
			st.Error = c.Error
			if !c.RefreshedAt.IsZero() {
				st.RefreshedAt = c.RefreshedAt.UTC().Format(time.RFC3339)
			}
			for _, it := range c.Items {
				it.Origin = "yours"
				if st.Builtin {
					it.Origin = "seed"
				}
				items = append(items, it)
			}
		}
		due := c == nil || (now.Sub(c.RefreshedAt) > refreshAfter && now.Sub(c.TriedAt) > retryAfter)
		if due && !s.reading[in] {
			stale = append(stale, in)
		}
		st.Reading = (c == nil || c.RefreshedAt.IsZero()) && (s.reading[in] || due)
		states = append(states, st)
	}
	s.mu.Unlock()
	for _, in := range stale {
		s.refresh(in)
	}
	return filter(items, q), states
}

func filter(items []Item, q string) []Item {
	needle := strings.ToLower(strings.TrimSpace(q))
	type scored struct {
		Item
		rank int
	}
	var hits []scored
	for _, it := range items {
		name, desc := strings.ToLower(it.Name), strings.ToLower(it.Description)
		rank := -1
		switch {
		case needle == "":
			rank = 3
		case name == needle:
			rank = 0
		case strings.Contains(name, needle):
			rank = 1
		case strings.Contains(desc, needle) || strings.Contains(strings.ToLower(it.Source), needle):
			rank = 2
		}
		if rank >= 0 {
			hits = append(hits, scored{it, rank})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].rank != hits[j].rank {
			return hits[i].rank < hits[j].rank
		}
		return hits[i].Name < hits[j].Name
	})
	out := make([]Item, 0, len(hits))
	for i, h := range hits {
		if i == maxHits {
			break
		}
		out = append(out, h.Item)
	}
	return out
}

// refresh reads one source in the background; one read per source at a time.
func (s *Store) refresh(input string) {
	s.mu.Lock()
	if s.reading[input] {
		s.mu.Unlock()
		return
	}
	s.reading[input] = true
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		s.Refresh(ctx, input)
	}()
}

// Refresh reads one source now. A failure keeps the last good list and
// records why.
func (s *Store) Refresh(ctx context.Context, input string) error {
	items, err := s.read(ctx, input)
	s.mu.Lock()
	c := s.sources[input]
	if c == nil {
		c = &cached{}
		s.sources[input] = c
	}
	c.TriedAt = s.Now()
	if err != nil {
		c.Error = err.Error()
		log.Printf("skillcatalog: %s: %v", input, err)
	} else {
		c.Items, c.Error, c.RefreshedAt = items, "", s.Now()
	}
	delete(s.reading, input)
	s.mu.Unlock()
	s.save()
	if s.OnRead != nil {
		s.OnRead(input)
	}
	return err
}

// Forget drops a source the person removed.
func (s *Store) Forget(input string) {
	s.mu.Lock()
	delete(s.sources, input)
	s.mu.Unlock()
	s.save()
}

func (s *Store) read(ctx context.Context, input string) ([]Item, error) {
	src, err := skills.ParseSource(input, s.Home)
	if err != nil {
		return nil, err
	}
	switch src.Kind {
	case "github":
		return s.readGitHub(ctx, src)
	case "well-known":
		return s.readWellKnown(ctx, src)
	}
	return nil, errors.New("not a repository or a site")
}

func (s *Store) getJSON(ctx context.Context, u string, limit int64, v any) error {
	body, err := s.get(ctx, u, limit)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (s *Store) get(ctx context.Context, u string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "picode-skills")
	res, err := s.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		switch res.StatusCode {
		case http.StatusNotFound:
			return nil, errors.New("not found")
		case http.StatusForbidden, http.StatusTooManyRequests:
			return nil, errors.New("the source refused the request (rate limit); PiCode tries again later")
		}
		return nil, fmt.Errorf("answered %s", res.Status)
	}
	return io.ReadAll(io.LimitReader(res.Body, limit))
}

// readGitHub lists every SKILL.md in the repository's tree (one API call) and
// reads each header from raw content (not rate-limited like the API).
func (s *Store) readGitHub(ctx context.Context, src skills.Source) ([]Item, error) {
	repo := src.Owner + "/" + src.Repo
	ref := src.Ref
	if ref == "" {
		var meta struct {
			DefaultBranch string `json:"default_branch"`
		}
		if err := s.getJSON(ctx, s.API+"/repos/"+repo, 1<<20, &meta); err != nil {
			return nil, fmt.Errorf("github.com/%s: %w", repo, err)
		}
		ref = meta.DefaultBranch
	}
	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
		Truncated bool `json:"truncated"`
	}
	if err := s.getJSON(ctx, s.API+"/repos/"+repo+"/git/trees/"+url.PathEscape(ref)+"?recursive=1", 32<<20, &tree); err != nil {
		return nil, fmt.Errorf("github.com/%s: %w", repo, err)
	}
	var dirs []string
	for _, e := range tree.Tree {
		if e.Type != "blob" || path.Base(e.Path) != "SKILL.md" {
			continue
		}
		dir := path.Dir(e.Path)
		if dir == "." {
			dir = ""
		}
		if src.Sub != "" && dir != src.Sub && !strings.HasPrefix(dir, src.Sub+"/") {
			continue
		}
		dirs = append(dirs, dir)
		if len(dirs) == maxSkills {
			break
		}
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("github.com/%s has no SKILL.md", repo)
	}
	items := make([]Item, len(dirs))
	errs := make([]error, len(dirs))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				items[i], errs[i] = s.gitHubItem(ctx, repo, ref, dirs[i])
			}
		}()
	}
	for i := range dirs {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	var out []Item
	failed := 0
	for i, it := range items {
		if errs[i] != nil {
			failed++
			continue
		}
		out = append(out, it)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("github.com/%s: no skill could be read (%v)", repo, errs[0])
	}
	return out, nil
}

func (s *Store) gitHubItem(ctx context.Context, repo, ref, dir string) (Item, error) {
	file := "SKILL.md"
	if dir != "" {
		file = dir + "/SKILL.md"
	}
	body, err := s.get(ctx, s.Raw+"/"+repo+"/"+url.PathEscape(ref)+"/"+escapePath(file), maxSkillBytes)
	if err != nil {
		return Item{}, err
	}
	fm, ok := skills.ParseFrontmatter(body)
	if !ok || fm.Description == "" {
		return Item{}, errors.New(file + " has no header")
	}
	name := fm.Name
	if name == "" {
		name = path.Base(dir)
	}
	install := repo
	if dir != "" {
		install += "/" + dir
	}
	page := "https://github.com/" + repo + "/tree/" + ref
	if dir != "" {
		page += "/" + escapePath(dir)
	}
	return Item{ID: "github:" + install, Name: name, Description: oneLine(fm.Description), License: fm.License,
		Source: repo, Install: install, URL: page}, nil
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// readWellKnown reads a site's discovery index (the agentskills.io RFC): its
// names and descriptions are the cards; the content is fetched and checked
// against its digest only when the person installs.
func (s *Store) readWellKnown(ctx context.Context, src skills.Source) ([]Item, error) {
	var idx struct {
		Skills []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"skills"`
	}
	if err := s.getJSON(ctx, src.Origin+"/.well-known/agent-skills/index.json", 4<<20, &idx); err != nil {
		return nil, fmt.Errorf("%s: %w", src.Origin, err)
	}
	host := strings.TrimPrefix(src.Origin, "https://")
	var out []Item
	for _, sk := range idx.Skills {
		if sk.Name == "" || len(out) == maxSkills {
			continue
		}
		out = append(out, Item{ID: "site:" + host + "/" + sk.Name, Name: sk.Name, Description: oneLine(sk.Description),
			Source: host, Install: src.Origin})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s lists no skills", src.Origin)
	}
	return out, nil
}

// SearchSkillsSH asks skills.sh (Vercel) — only when the person switched it
// on. Its answer carries no description; the card names skills.sh and its
// install count, and links to the page where skills.sh shows its audits.
func (s *Store) SearchSkillsSH(ctx context.Context, q string) ([]Item, error) {
	q = strings.TrimSpace(q)
	if len(q) < 2 {
		return nil, nil
	}
	var res struct {
		Skills []struct {
			ID       string `json:"id"`
			Source   string `json:"source"`
			SkillID  string `json:"skillId"`
			Name     string `json:"name"`
			Installs int    `json:"installs"`
		} `json:"skills"`
	}
	if err := s.getJSON(ctx, s.SkillsSH+"/api/search?q="+url.QueryEscape(q)+"&limit=30", 1<<20, &res); err != nil {
		return nil, fmt.Errorf("skills.sh did not answer: %w", err)
	}
	var out []Item
	for _, sk := range res.Skills {
		parts := strings.Split(sk.Source, "/")
		if len(parts) != 2 || sk.Name == "" {
			continue
		}
		out = append(out, Item{ID: "skills.sh:" + sk.ID, Name: sk.Name, Source: sk.Source, Install: sk.Source,
			URL: "https://skills.sh/" + sk.ID, Origin: "skills.sh", Installs: sk.Installs})
	}
	return out, nil
}

type cacheFile struct {
	Version int                `json:"version"`
	Sources map[string]*cached `json:"sources"`
}

func (s *Store) cachePath() string {
	if s.dataDir == "" {
		return ""
	}
	return filepath.Join(s.dataDir, cacheFileName)
}

func (s *Store) load() {
	p := s.cachePath()
	if p == "" {
		return
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var f cacheFile
	if json.Unmarshal(raw, &f) != nil || f.Version != cacheVersion || f.Sources == nil {
		return
	}
	s.sources = f.Sources
}

func (s *Store) save() {
	p := s.cachePath()
	if p == "" {
		return
	}
	s.mu.Lock()
	raw, err := json.Marshal(cacheFile{Version: cacheVersion, Sources: s.sources})
	s.mu.Unlock()
	if err != nil {
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tmp := p + ".tmp"
	if os.WriteFile(tmp, raw, 0o600) == nil {
		_ = os.Rename(tmp, p)
	}
}
