package llamaservice

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Creation struct {
	Stage string `json:"stage"`
	Token string `json:"token"`
}

const creationMarker = ".picode-creation"

func (s *Service) createProfile(c Config) (err error) {
	if s.doc.Creation != nil {
		if err = s.rollbackCreation(s.doc.Creation); err != nil {
			return err
		}
		s.doc.Creation = nil
		if err = s.save(); err != nil {
			return err
		}
	}
	if _, e := os.Lstat(s.root); !os.IsNotExist(e) {
		return errors.New("The service folder already exists or cannot be checked. Existing folders are never adopted.")
	}
	stage, err := os.MkdirTemp(filepath.Dir(s.root), ".llama-create-")
	if err != nil {
		return err
	}
	attempt := &Creation{Stage: filepath.Base(stage), Token: token()}
	committed := false
	defer func() {
		if !committed {
			cleanup := s.rollbackCreation(attempt)
			if cleanup != nil {
				err = errors.Join(err, cleanup)
			}
			if cleanup == nil && s.doc.Creation != nil {
				s.doc.Creation = nil
				if e := s.save(); e != nil {
					err = errors.Join(err, e)
				}
			}
		}
	}()
	if err = os.WriteFile(filepath.Join(stage, creationMarker), []byte(attempt.Token), 0600); err != nil {
		_ = os.Remove(stage) // Empty directory made by this call; never recursive.
		return err
	}
	// Persist intent before publishing a fixed path: restart can undo an empty
	// owned attempt after SIGKILL without ever adopting a preexisting folder.
	s.doc.Creation = attempt
	if err = s.save(); err != nil {
		return err
	}
	for _, child := range []string{"models", "cache"} {
		if err = s.ctx.Err(); err != nil {
			return err
		}
		if err = os.Mkdir(filepath.Join(stage, child), 0700); err != nil {
			return err
		}
	}
	if err = s.ctx.Err(); err != nil {
		return err
	}
	if err = publishCreation(stage, s.root); err != nil {
		return fmt.Errorf("Could not publish the service folder; existing folders are preserved: %w", err)
	}
	s.doc.Created = true
	s.doc.Config = c
	s.doc.Archives = map[string]string{}
	s.doc.Jobs = []Job{}
	s.doc.Creation = nil
	if err = s.save(); err != nil {
		return err
	}
	committed = true
	_ = os.Remove(filepath.Join(s.root, creationMarker))
	return nil
}

func (s *Service) rollbackCreation(attempt *Creation) error {
	if attempt == nil || filepath.Base(attempt.Stage) != attempt.Stage || !strings.HasPrefix(attempt.Stage, ".llama-create-") || len(attempt.Token) != 48 {
		return errors.New("Invalid setup recovery record; folders were preserved.")
	}
	for _, dir := range []string{filepath.Join(filepath.Dir(s.root), attempt.Stage), s.root} {
		info, err := os.Lstat(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		marker := filepath.Join(dir, creationMarker)
		resolved, err := filepath.EvalSymlinks(marker)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if resolved != marker {
			continue
		}
		proof, err := os.ReadFile(marker)
		if err != nil {
			return err
		}
		if string(proof) != attempt.Token {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		// Check the entire directory before touching anything. Files appearing
		// during setup are never removed, including a nonempty expected child.
		for _, entry := range entries {
			if entry.Name() == creationMarker {
				continue
			}
			if (entry.Name() != "models" && entry.Name() != "cache") || !entry.IsDir() {
				return errors.New("Incomplete setup contains unexpected files; its folder was preserved.")
			}
			children, err := os.ReadDir(filepath.Join(dir, entry.Name()))
			if err != nil || len(children) != 0 {
				return errors.New("Incomplete setup contains files; its folder was preserved.")
			}
		}
		for _, entry := range entries {
			if entry.Name() == creationMarker {
				continue
			}
			if err = os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
		if err = os.Remove(marker); err != nil {
			return err
		}
		if err = os.Remove(dir); err != nil {
			return err
		}
	}
	return nil
}
