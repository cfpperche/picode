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
func (s *Service) ObserveDownload(j store.LlamaJob) {
	if j.Operation != "download" || j.State != "succeeded" {
		return
	}
	s.mu.Lock()
	if s.closed || s.process == nil || !s.isAppliedEndpoint(j.Endpoint) {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.process == nil || !s.isAppliedEndpoint(j.Endpoint) {
		return
	}
	for _, m := range models {
		if m.ID != j.Model || m.Path == "" {
			continue
		}
		rel, err := filepath.Rel(s.root, m.Path)
		if err != nil || !strings.HasPrefix(rel, "models"+string(filepath.Separator)) || strings.Contains(rel, "..") {
			continue
		}
		existing := false
		for _, old := range baseline {
			if old == rel {
				existing = true
			}
		}
		if existing {
			continue
		}
		h, size, err := hashRegular(m.Path)
		if err != nil {
			continue
		}
		if s.doc.Models == nil {
			s.doc.Models = map[string]OwnedModel{}
		}
		info, err := os.Stat(m.Path)
		if err != nil {
			continue
		}
		s.doc.Models[rel] = OwnedModel{Model: m.ID, SHA256: h, Job: j.ID, Size: size, Modified: info.ModTime().UnixNano()}
	}
	delete(s.doc.Downloads, j.ID)
	_ = s.save()
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
			if (a.Provider == nil || *a.Provider == "llama.cpp") && (a.Model == nil || *a.Model == model.Model) {
				references = append(references, a.Name)
			}
		}
		if len(references) > 0 {
			return "", "", errors.New("Referenced by " + strings.Join(references, ", ") + "; retained.")
		}
		return filepath.Join(s.root, name), model.SHA256, nil
	}
	pin, ok := s.doc.Archives[name]
	if !ok || filepath.Base(name) != name {
		return "", "", errors.New("The selected file has no ownership record.")
	}
	return filepath.Join(s.root, "cache", name), pin, nil
}
