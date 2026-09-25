package llamaservice

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/llama"
	"github.com/cfpperche/picode/internal/store"
)

type OwnedModel struct {
	Model    string `json:"model"`
	SHA256   string `json:"sha256"`
	Job      string `json:"job"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
	Snapshot string `json:"snapshot,omitempty"`
}

func (s *Service) modelFiles() ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(filepath.Join(s.root, "models"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, e := filepath.Rel(s.root, path)
			if e != nil {
				return e
			}
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

// ObserveDownload records only the exact model path returned by our own router,
// after a tracked download created a new file. Existing/unknown files stay unowned.
// A download that ended any other way only drops the file list it started with.
func (s *Service) ObserveDownload(j store.LlamaJob) {
	if j.Operation != "download" {
		return
	}
	if j.State != "succeeded" {
		s.forgetDownload(j.ID)
		return
	}
	s.mu.Lock()
	if s.closed || s.process == nil || !s.isAppliedEndpoint(j.Endpoint) {
		// A success that can no longer be observed (the service stopped or
		// moved) owns nothing; its starting list goes with it.
		if _, ok := s.doc.Downloads[j.ID]; ok && !s.closed {
			delete(s.doc.Downloads, j.ID)
			_ = s.save()
		}
		s.mu.Unlock()
		return
	}
	baseline, tracked := s.doc.Downloads[j.ID]
	if !tracked {
		s.mu.Unlock()
		return
	}
	s.wg.Add(1)
	s.mu.Unlock()
	defer s.wg.Done()
	c, err := llama.New(j.Endpoint, "")
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	models, err := c.ListContext(ctx)
	if err != nil {
		return
	}
	// Pick the new files under the lock, hash them without it (a GGUF is
	// gigabytes: hashing under s.mu held every model operation, even for a
	// remote server, for as long as it took — 2026-09-23 review), then record
	// them under the lock again, only if nothing moved meanwhile.
	type found struct {
		rel, path, model, snapshot string
		size                       int64
	}
	var picked []found
	s.mu.Lock()
	if s.closed || s.process == nil || !s.isAppliedEndpoint(j.Endpoint) {
		s.mu.Unlock()
		return
	}
	for _, m := range models {
		if m.ID != j.Model {
			continue
		}
		candidates := []modelCandidate{{path: m.Path}}
		if m.Path == "" {
			candidates = s.hfDownloadFiles(j)
		}
		for _, candidate := range candidates {
			rel, err := filepath.Rel(s.root, candidate.path)
			if err != nil || !strings.HasPrefix(rel, "models"+string(filepath.Separator)) || strings.Contains(rel, "..") {
				continue
			}
			existing := false
			for _, old := range baseline {
				if old == rel || (candidate.snapshot != "" && old == candidate.snapshot) {
					existing = true
				}
			}
			if !existing {
				picked = append(picked, found{rel: rel, path: candidate.path, model: m.ID, snapshot: candidate.snapshot, size: candidate.size})
			}
		}
	}
	s.mu.Unlock()

	type hashed struct {
		found
		sha      string
		size     int64
		modified int64
	}
	var owned []hashed
	for _, f := range picked {
		h, size, err := hashRegular(f.path)
		if err != nil {
			continue
		}
		if f.snapshot != "" && (h != filepath.Base(f.path) || size != f.size) {
			continue
		}
		info, err := os.Stat(f.path)
		if err != nil {
			continue
		}
		owned = append(owned, hashed{found: f, sha: h, size: size, modified: info.ModTime().UnixNano()})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.process == nil || !s.isAppliedEndpoint(j.Endpoint) {
		return
	}
	if _, still := s.doc.Downloads[j.ID]; !still {
		return
	}
	for _, o := range owned {
		// The file must be the one hashed: same size and time as then.
		info, err := os.Stat(o.path)
		if err != nil || info.Size() != o.size || info.ModTime().UnixNano() != o.modified {
			continue
		}
		if s.doc.Models == nil {
			s.doc.Models = map[string]OwnedModel{}
		}
		s.doc.Models[o.rel] = OwnedModel{Model: o.model, SHA256: o.sha, Job: j.ID, Size: o.size, Modified: o.modified, Snapshot: o.snapshot}
	}
	delete(s.doc.Downloads, j.ID)
	_ = s.save()
}

// forgetDownload drops a download's starting file list once the download can
// no longer succeed; nothing is owned from it.
func (s *Service) forgetDownload(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	if _, ok := s.doc.Downloads[id]; ok {
		delete(s.doc.Downloads, id)
		_ = s.save()
	}
}

func (s *Service) cacheTarget(name string) (string, string, error) {
	if model, ok := s.doc.Models[name]; ok {
		if !strings.HasPrefix(name, "models"+string(filepath.Separator)) || strings.Contains(name, "..") {
			return "", "", errors.New("Invalid model path.")
		}
		agents, err := s.st.ListAllAgents()
		if err != nil {
			return "", "", err
		}
		var references []string
		for _, a := range agents {
			if (a.Provider == nil || *a.Provider == "llama.cpp") && (a.Model == nil || *a.Model == model.Model || (model.Snapshot != "" && hfRepo(*a.Model) == hfRepo(model.Model))) {
				references = append(references, a.Name)
			}
		}
		if len(references) > 0 {
			return "", "", errors.New("Referenced by " + strings.Join(references, ", ") + "; retained.")
		}
		if model.Snapshot != "" {
			if err := s.checkSnapshot(name, model); err != nil {
				return "", "", err
			}
		}
		return filepath.Join(s.root, name), model.SHA256, nil
	}
	pin, ok := s.doc.Archives[name]
	if !ok || filepath.Base(name) != name {
		return "", "", errors.New("The selected file has no ownership record.")
	}
	return filepath.Join(s.root, "cache", name), pin, nil
}
