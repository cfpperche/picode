// Package mcpcatalog curates the Connectors marketplace catalog (ADR-0157):
// the pinned seed presets always answer first; the official MCP Registry is
// synced in the background, filtered by curation rules, and cached in the
// data dir so restarts answer warm. The registry can break or drift its
// schema — the catalog degrades to the seed, and the pane stays useful
// offline. No third-party proxy ever carries a connection: a card only
// describes a server, credentials stay with each CLI.
package mcpcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/mcp"
)

// Item is one card in the Connectors marketplace.
type Item struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Summary  string   `json:"summary"`
	Kind     string   `json:"kind"` // "url" | "stdio"
	URL      string   `json:"url,omitempty"`
	Command  string   `json:"command,omitempty"`
	Args     []string `json:"args,omitempty"`
	Auth     string   `json:"auth,omitempty"` // "oauth" when known
	Featured bool     `json:"featured"`
	Source   string   `json:"source"` // "picode" | "registry"
}

const (
	// DefaultBase is the official MCP Registry (ADR-0157).
	DefaultBase = "https://registry.modelcontextprotocol.io"
	// refreshAfter is the cache age past which Search kicks one
	// background refresh.
	refreshAfter = 24 * time.Hour
	// fetchTimeout bounds one background refresh.
	fetchTimeout = 60 * time.Second
	// maxSearchHits caps what Search returns to the pane.
	maxSearchHits = 200
	// maxRegistryPages bounds pagination (100/page → 12k servers, above
	// the registry's current size).
	maxRegistryPages = 120
	// statusMetaKey is the registry's official _meta namespace carrying
	// the publication status.
	statusMetaKey = "io.modelcontextprotocol.registry/official"
	// cacheFileName is the catalog cache inside the data dir.
	cacheFileName = "connectors-catalog.json"
)

// Store answers catalog searches. Zero-value is not usable; use NewStore.
type Store struct {
	// Base is the registry root URL; injectable for tests.
	Base string
	// HTTP fetches the registry; nil means a plain http.Client.
	HTTP *http.Client
	// Now returns the current time; injectable for tests.
	Now func() time.Time

	dataDir string // "" → memory only, cache is never persisted

	mu          sync.Mutex
	registry    []Item    // last synced registry items (sorted by id)
	refreshedAt time.Time // zero until a refresh or cache load succeeded
	refreshing  bool      // one background refresh at a time

	// writeMu serializes the cache write. Two refreshes at once (a manual
	// one and the background timer) shared one tmp path, so the slower
	// rename found the file already moved and failed with "no such file or
	// directory" — a CI flake that looked like a connectors bug (2026-09-19).
	writeMu sync.Mutex
}

// NewStore builds a store persisted under dataDir ("" → memory only).
// It never fetches: the first Search answers with the seed (plus any
// cache file on disk) and refreshes in the background.
func NewStore(dataDir string) *Store {
	s := &Store{Base: DefaultBase, dataDir: dataDir, Now: time.Now}
	s.loadCache()
	return s
}

// Seed maps the hand-picked presets (internal/mcp) into catalog cards.
// They are featured, pinned first, and always served — the registry can
// vanish without emptying the pane.
func Seed() []Item {
	presets := mcp.Presets()
	out := make([]Item, 0, len(presets))
	for _, p := range presets {
		it := Item{ID: p.ID, Name: p.Name, Summary: p.Summary, Featured: true, Source: "picode"}
		if p.Entry.URL != "" {
			it.Kind = "url"
			it.URL = p.Entry.URL
		} else {
			it.Kind = "stdio"
			it.Command = p.Entry.Command
			it.Args = p.Entry.Args
		}
		if p.Entry.Auth == "oauth" {
			it.Auth = "oauth"
		}
		out = append(out, it)
	}
	return out
}

// Search serves the cached catalog immediately: seed items always, then
// registry items when loaded. Empty query → featured first, registry by
// name; a query is a case-insensitive substring match on id, name and
// summary. When the cache is stale and no refresh is in flight, exactly
// one background refresh starts — this call still answers with what is
// cached right now.
func (s *Store) Search(ctx context.Context, q string) []Item {
	q = strings.ToLower(strings.TrimSpace(q))
	s.kickRefreshIfStale()

	seed := Seed()
	s.mu.Lock()
	registry := s.registry
	s.mu.Unlock()

	matches := func(it Item) bool {
		if q == "" {
			return true
		}
		return strings.Contains(strings.ToLower(it.ID), q) ||
			strings.Contains(strings.ToLower(it.Name), q) ||
			strings.Contains(strings.ToLower(it.Summary), q)
	}
	out := make([]Item, 0, len(seed)+len(registry))
	for _, it := range seed {
		if matches(it) {
			out = append(out, it)
		}
	}
	if q == "" {
		registry = append([]Item(nil), registry...)
		sort.Slice(registry, func(i, j int) bool { return registry[i].Name < registry[j].Name })
	}
	for _, it := range registry {
		if matches(it) {
			out = append(out, it)
		}
	}
	if len(out) > maxSearchHits {
		out = out[:maxSearchHits]
	}
	return out
}

// kickRefreshIfStale starts at most one background refresh when the cache
// is empty or older than the refresh interval. It never blocks the caller.
func (s *Store) kickRefreshIfStale() {
	s.mu.Lock()
	stale := s.refreshedAt.IsZero() || s.Now().Sub(s.refreshedAt) >= refreshAfter
	if stale && !s.refreshing {
		s.refreshing = true
	} else {
		stale = false
	}
	s.mu.Unlock()
	if !stale {
		return
	}
	go func() {
		defer func() {
			s.mu.Lock()
			s.refreshing = false
			s.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		_ = s.Refresh(ctx) // offline degrades to the seed; retried on a later search
	}()
}

// registryPage is one page of GET /v0/servers.
type registryPage struct {
	Servers  []registryEntry `json:"servers"`
	Metadata struct {
		NextCursor string `json:"nextCursor"`
	} `json:"metadata"`
}

type registryEntry struct {
	Server struct {
		Name        string `json:"name"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Remotes     []struct {
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"remotes"`
	} `json:"server"`
	Meta map[string]json.RawMessage `json:"_meta"`
}

// cacheFileV1 is the on-disk cache: the merged view at refresh time.
// Seed cards are re-derived from the running binary on load, so only
// registry rows survive a restart load.
type cacheFileV1 struct {
	RefreshedAt time.Time `json:"refreshedAt"`
	Items       []Item    `json:"items"`
}

// Refresh syncs the official registry: paginate /v0/servers, filter by
// the curation rules (active status, non-empty description, at least one
// HTTP remote), skip seed ids, sort by id, publish, and persist the cache.
// A failed or empty fetch returns an error and leaves the previous state.
func (s *Store) Refresh(ctx context.Context) error {
	var filtered []Item
	cursor := ""
	pages := 0
	for {
		pages++
		if pages > maxRegistryPages {
			return fmt.Errorf("registry pagination exceeded %d pages", maxRegistryPages)
		}
		u := strings.TrimSuffix(s.Base, "/") + "/v0/servers?version=latest&limit=100"
		if cursor != "" {
			u += "&cursor=" + url.QueryEscape(cursor)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		client := s.HTTP
		if client == nil {
			client = http.DefaultClient
		}
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		var page registryPage
		decErr := json.NewDecoder(res.Body).Decode(&page)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("registry status %d", res.StatusCode)
		}
		if decErr != nil {
			return fmt.Errorf("registry response is not JSON: %w", decErr)
		}
		for _, e := range page.Servers {
			if it, ok := itemFromEntry(e); ok {
				filtered = append(filtered, it)
			}
		}
		if page.Metadata.NextCursor == "" {
			break
		}
		cursor = page.Metadata.NextCursor
	}
	if len(filtered) == 0 {
		return fmt.Errorf("registry returned no servers")
	}

	seedIDs := map[string]bool{}
	for _, it := range Seed() {
		seedIDs[it.ID] = true
	}
	out := make([]Item, 0, len(filtered))
	for _, it := range filtered {
		if !seedIDs[it.ID] {
			out = append(out, it)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	s.mu.Lock()
	s.registry = out
	s.refreshedAt = s.Now()
	s.mu.Unlock()
	return s.writeCache()
}

// itemFromEntry applies the curation rules to one registry entry.
func itemFromEntry(e registryEntry) (Item, bool) {
	if entryStatus(e) != "active" || strings.TrimSpace(e.Server.Description) == "" {
		return Item{}, false
	}
	remoteURL := ""
	for _, r := range e.Server.Remotes {
		if strings.Contains(r.Type, "http") {
			remoteURL = r.URL
			break
		}
	}
	if remoteURL == "" {
		return Item{}, false
	}
	id := e.Server.Name
	if id == "" {
		return Item{}, false
	}
	name := e.Server.Title
	if name == "" {
		name = id
		if i := strings.LastIndex(id, "/"); i >= 0 {
			name = id[i+1:]
		}
	}
	return Item{
		ID:      id,
		Name:    name,
		Summary: e.Server.Description,
		Kind:    "url",
		URL:     remoteURL,
		Source:  "registry",
	}, true
}

func entryStatus(e registryEntry) string {
	raw, ok := e.Meta[statusMetaKey]
	if !ok {
		return ""
	}
	var m struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	return m.Status
}

func (s *Store) cachePath() string { return filepath.Join(s.dataDir, cacheFileName) }

// loadCache restores a previous refresh so restarts answer warm. Seed
// rows in the file are ignored: the seed is pinned by the running binary.
func (s *Store) loadCache() {
	if s.dataDir == "" {
		return
	}
	b, err := os.ReadFile(s.cachePath())
	if err != nil {
		return
	}
	var cf cacheFileV1
	if json.Unmarshal(b, &cf) != nil {
		return
	}
	registry := make([]Item, 0, len(cf.Items))
	for _, it := range cf.Items {
		if it.Source == "registry" {
			registry = append(registry, it)
		}
	}
	s.registry = registry
	s.refreshedAt = cf.RefreshedAt
}

// writeCache persists the merged view atomically: a unique tmp file of its
// own, renamed over the target, under a lock that keeps two writers from
// sharing (and stealing) that file.
func (s *Store) writeCache() error {
	if s.dataDir == "" {
		return nil
	}
	s.mu.Lock()
	cf := cacheFileV1{RefreshedAt: s.refreshedAt, Items: append(append([]Item(nil), Seed()...), s.registry...)}
	s.mu.Unlock()
	b, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tmp, err := os.CreateTemp(s.dataDir, cacheFileName+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, s.cachePath()); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
