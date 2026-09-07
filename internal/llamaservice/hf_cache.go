package llamaservice

import (
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cfpperche/picode/internal/store"
)

var hfRepoPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$`)

func hfRepo(model string) string {
	repo, _, _ := strings.Cut(model, ":")
	if !hfRepoPattern.MatchString(repo) || strings.Contains(repo, "..") {
		return ""
	}
	return repo
}

type modelCandidate struct {
	path, snapshot string
	size           int64
}

// b10809/b10826 expose HF cache rows without a path. Correlate only complete
// observed filenames with their exact snapshot link and SHA-256 blob. Never
// infer ownership from a directory difference alone or from a model alias.
func (s *Service) hfDownloadFiles(j store.LlamaJob) []modelCandidate {
	if s.doc.Current == nil || (s.doc.Current.Version != "b10809" && s.doc.Current.Version != "b10826") {
		return nil
	}
	repo := hfRepo(j.Model)
	if repo == "" {
		return nil
	}
	base := filepath.Join(s.root, "models", "models--"+strings.ReplaceAll(repo, "/", "--"))
	var out []modelCandidate
	for _, progress := range j.Progress {
		if progress.Total <= 0 || progress.Done != progress.Total || !strings.HasSuffix(progress.File, ".gguf") || !filepath.IsLocal(progress.File) || strings.Contains(progress.File, "..") {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(base, "snapshots"))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			commit, err := hex.DecodeString(entry.Name())
			if err != nil || len(commit) != 20 || !entry.IsDir() {
				continue
			}
			link := filepath.Join(base, "snapshots", entry.Name(), progress.File)
			parent, err := filepath.EvalSymlinks(filepath.Dir(link))
			if err != nil || parent != filepath.Dir(link) {
				continue
			}
			target, err := os.Readlink(link)
			if err != nil {
				continue
			}
			blob := filepath.Clean(filepath.Join(filepath.Dir(link), target))
			digest, err := hex.DecodeString(filepath.Base(blob))
			if err != nil || len(digest) != 32 || filepath.Dir(blob) != filepath.Join(base, "blobs") {
				continue
			}
			rel, err := filepath.Rel(s.root, link)
			if err == nil {
				out = append(out, modelCandidate{blob, rel, progress.Total})
			}
		}
	}
	return out
}

// A blob can be shared by multiple snapshots or quantization aliases. Retain it
// if any link other than the recorded one references it, or that link changed.
func (s *Service) checkSnapshot(name string, model OwnedModel) error {
	if !filepath.IsLocal(model.Snapshot) || !strings.HasPrefix(model.Snapshot, "models"+string(filepath.Separator)) {
		return errors.New("Invalid cache snapshot record.")
	}
	blob := filepath.Join(s.root, name)
	link := filepath.Join(s.root, model.Snapshot)
	parent, err := filepath.EvalSymlinks(filepath.Dir(link))
	if err != nil || parent != filepath.Dir(link) {
		return errors.New("Cache snapshot changed; retained.")
	}
	target, err := os.Readlink(link)
	if err != nil || filepath.Clean(filepath.Join(filepath.Dir(link), target)) != blob {
		return errors.New("Cache snapshot changed; retained.")
	}
	return filepath.WalkDir(filepath.Join(s.root, "models"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink == 0 || path == link {
			return nil
		}
		target, err := filepath.EvalSymlinks(path)
		if err == nil && target == blob {
			return errors.New("Another cache snapshot references this file; retained.")
		}
		return nil
	})
}
