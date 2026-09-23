package llamaservice

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func (s *Service) rememberRelease(r *Release) {
	if r == nil {
		return
	}
	if s.doc.Releases == nil {
		s.doc.Releases = map[string]Release{}
	}
	s.doc.Releases[filepath.Base(r.Dir)] = *r
}

// sweepInterrupted removes what an install killed mid-way leaves behind, at
// start, when no install can be running (2026-09-23 review): `cache/.install-*`
// are always PiCode's own download temps; a `release-*` folder with no record
// is removed only when it is also incomplete (no llama-server — an extraction
// cut short). A complete unrecorded one stays: unknown ownership is preserved
// (ADR-0090), and the service lists it as "Unknown installation".
func (s *Service) sweepInterrupted() {
	if temps, err := filepath.Glob(filepath.Join(s.root, "cache", ".install-*")); err == nil {
		for _, t := range temps {
			if info, err := os.Lstat(t); err == nil && info.Mode().IsRegular() {
				_ = os.Remove(t)
			}
		}
	}
	records := s.releaseRecords()
	dirs, err := filepath.Glob(filepath.Join(s.root, "release-*"))
	if err != nil {
		return
	}
	for _, d := range dirs {
		name := filepath.Base(d)
		if _, recorded := records[name]; recorded {
			continue
		}
		info, err := os.Lstat(d)
		if err != nil || !info.IsDir() {
			continue
		}
		if _, err := os.Lstat(filepath.Join(d, "llama-server")); err == nil {
			continue // complete: an unknown installation, kept
		}
		_ = os.RemoveAll(d)
	}
}

func (s *Service) releaseRecords() map[string]Release {
	out := map[string]Release{}
	for name, r := range s.doc.Releases {
		out[name] = r
	}
	for _, r := range []*Release{s.doc.Current, s.doc.Previous} {
		if r != nil {
			out[filepath.Base(r.Dir)] = *r
		}
	}
	return out
}

func (s *Service) releaseTarget(name string) (Release, int64, error) {
	r, ok := s.releaseRecords()[name]
	if !ok || filepath.Base(name) != name || !strings.HasPrefix(name, "release-") || r.Dir != filepath.Join(s.root, name) {
		return r, 0, errors.New("Installation ownership is unverified; retained.")
	}
	for _, active := range []*Release{s.doc.Current, s.doc.Previous} {
		if active != nil && filepath.Clean(active.Dir) == r.Dir {
			return r, 0, errors.New("Current or rollback installation; retained.")
		}
	}
	if err := verifyRelease(&r); err != nil {
		return r, 0, err
	}
	var size int64
	for file := range r.Files {
		info, err := os.Stat(filepath.Join(r.Dir, file))
		if err != nil {
			return r, 0, err
		}
		size += info.Size()
	}
	return r, size, nil
}

// Both previews and workers call this. Installed directories are compared to
// their complete manifest, including refusal of extra files and symlinks.
func (s *Service) cleanupFingerprint(name string) (string, int64, error) {
	if strings.HasPrefix(name, "release-") {
		r, size, err := s.releaseTarget(name)
		return fingerprint(r), size, err
	}
	path, pin, err := s.cacheTarget(name)
	if err != nil {
		return "", 0, err
	}
	h, size, err := hashRegular(path)
	if err != nil || h != pin {
		return "", 0, errors.New("A selected cache file changed. Refresh and review again.")
	}
	return h, size, nil
}

func (s *Service) removeRelease(name string) error {
	r, _, err := s.releaseTarget(name)
	if err != nil {
		return err
	}
	// No recursive deletion: a file appearing after verification survives and
	// prevents removal of the directory. Partial IO failures are reported.
	for file := range r.Files {
		if err = os.Remove(filepath.Join(r.Dir, file)); err != nil {
			return err
		}
	}
	if err = os.Remove(r.Dir); err != nil {
		return err
	}
	delete(s.doc.Releases, name)
	return nil
}

func (s *Service) installationFiles() []CacheFile {
	var out []CacheFile
	entries, _ := os.ReadDir(s.root)
	records := s.releaseRecords()
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "release-") {
			continue
		}
		name := entry.Name()
		r, known := records[name]
		label := "Unknown installation"
		if known {
			label = "llama.cpp " + r.Version
		}
		_, size, err := s.releaseTarget(name)
		eligible := err == nil && s.process == nil && !s.busy
		reason := "Verified older installation; current and rollback versions are retained."
		if err != nil {
			reason = err.Error()
		} else if !eligible {
			reason = "Stop the service and refresh before removing this installation."
		}
		out = append(out, CacheFile{Name: name, Bytes: size, Eligible: eligible, Reason: reason, Label: label})
	}
	return out
}
