// Package workspace manages the registry of workspaces (project folders)
// where Pi agents run. File-backed JSON at ~/.picode/workspaces.json —
// a plain, inspectable format (philosophy: no proprietary lock-in).
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Workspace is one registered project folder.
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"createdAt"`
}

// Registry is a file-backed workspace store. Safe for concurrent use.
type Registry struct {
	mu   sync.Mutex
	file string
}

// DefaultPath returns the default registry location (~/.picode/workspaces.json).
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("workspace: home dir: %w", err)
	}
	return filepath.Join(home, ".picode", "workspaces.json"), nil
}

// Open loads (or creates) the registry at path.
func Open(path string) (*Registry, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}
	r := &Registry{file: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := r.save(nil); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, fmt.Errorf("workspace: stat %s: %w", path, err)
	}
	return r, nil
}

func (r *Registry) load() ([]Workspace, error) {
	data, err := os.ReadFile(r.file)
	if err != nil {
		return nil, fmt.Errorf("workspace: read %s: %w", r.file, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var ws []Workspace
	if err := unmarshalWorkspaces(data, &ws); err != nil {
		return nil, fmt.Errorf("workspace: parse %s: %w", r.file, err)
	}
	return ws, nil
}

func (r *Registry) save(ws []Workspace) error {
	data, err := marshalWorkspaces(ws)
	if err != nil {
		return fmt.Errorf("workspace: encode: %w", err)
	}
	tmp := r.file + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("workspace: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, r.file); err != nil {
		return fmt.Errorf("workspace: rename: %w", err)
	}
	return nil
}

// List returns workspaces ordered by name.
func (r *Registry) List() ([]Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ws, err := r.load()
	if err != nil {
		return nil, err
	}
	sort.Slice(ws, func(i, j int) bool { return ws[i].Name < ws[j].Name })
	return ws, nil
}

// Get returns a workspace by id.
func (r *Registry) Get(id string) (Workspace, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ws, err := r.load()
	if err != nil {
		return Workspace{}, false, err
	}
	for _, w := range ws {
		if w.ID == id {
			return w, true, nil
		}
	}
	return Workspace{}, false, nil
}

var idPattern = regexp.MustCompile(`[^a-z0-9]+`)

// Add registers a workspace. path must be an existing directory; it is
// resolved to an absolute path. The id is a readable slug with a short
// random suffix for uniqueness.
func (r *Registry) Add(name, path string) (Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return Workspace{}, errors.New("workspace: name is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Workspace{}, fmt.Errorf("workspace: path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Workspace{}, fmt.Errorf("workspace: path %s: %w", abs, err)
	}
	if !info.IsDir() {
		return Workspace{}, fmt.Errorf("workspace: path %s: not a directory", abs)
	}

	ws, err := r.load()
	if err != nil {
		return Workspace{}, err
	}
	// Same path twice? Return the existing entry (idempotent add).
	for _, w := range ws {
		if w.Path == abs {
			return w, nil
		}
	}

	slug := strings.Trim(idPattern.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" {
		slug = "workspace"
	}
	id := fmt.Sprintf("%s-%s", slug, randSuffix(4))

	w := Workspace{ID: id, Name: name, Path: abs, CreatedAt: time.Now().UTC()}
	ws = append(ws, w)
	if err := r.save(ws); err != nil {
		return Workspace{}, err
	}
	return w, nil
}

// Remove deletes a workspace from the registry (its files are untouched).
func (r *Registry) Remove(id string) (removed bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ws, err := r.load()
	if err != nil {
		return false, err
	}
	kept := ws[:0]
	for _, w := range ws {
		if w.ID != id {
			kept = append(kept, w)
		}
	}
	if len(kept) == len(ws) {
		return false, nil
	}
	return true, r.save(kept)
}
