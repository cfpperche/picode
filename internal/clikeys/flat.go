package clikeys

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cfpperche/picode/internal/clisettings"
)

// The flat key map: one file, one object of action id -> chord list, which is
// the shape pi, Omp and Antigravity use (ADR-0174).
//
// A declaration says where the file lives, what its catalog is and which
// platform vocabulary it speaks; the engine below reads it and writes one row at
// a time through the settings package's document primitives, so the byte-span
// splice, the atomic write, the revision check and the refusals are the ones
// that package already guarantees — not a second implementation of them.

// FlatMap is one CLI's flat key map, declared.
type FlatMap struct {
	CLI     string
	Catalog []Action
	// File resolves the file this machine reads and the syntax it is written
	// in. It is called per request: the answer depends on environment the CLI
	// itself reads (its config root, an active profile).
	File func() (string, clisettings.Format, error)
	// Platform is the CLI's own platform vocabulary, for the rows that bind
	// differently elsewhere.
	Platform string
}

// Map is one flat key map as the pane renders it.
type Map struct {
	File     string
	Exists   bool
	Revision string
	// Values holds only the actions this file binds: an action it does not
	// mention is running on the CLI's own default.
	Values map[string][]string
	// Unreadable names the actions the file sets in a shape PiCode does not
	// rewrite — a table where a chord list belongs. The read still answers, so
	// one hand-written oddity does not take the whole map away from the reader;
	// the pane drops those rows and says how many, and a write to one is
	// refused with ErrShape.
	Unreadable []string
}

// FlatFor returns the declaration for a CLI, or false when PiCode has no flat
// engine for it.
func FlatFor(id string) (FlatMap, bool) {
	f, ok := flatDeclarations[id]
	return f, ok
}

var flatDeclarations = map[string]FlatMap{"omp": OmpFlat}

// ReadFlat reads a declaration's file. A file that is not there is a map with
// no values, not an error: the CLI is running on its own defaults and the first
// write creates it.
func ReadFlat(f FlatMap) (Map, error) {
	path, format, err := f.File()
	if err != nil {
		return Map{}, err
	}
	doc, err := clisettings.OpenDoc(path, format)
	if err != nil {
		return Map{}, err
	}
	out := Map{File: path, Exists: doc.Exists(), Revision: doc.Revision(), Values: map[string][]string{}}
	for _, a := range f.Catalog {
		values, found, err := doc.Strings(a.ID)
		if err != nil {
			out.Unreadable = append(out.Unreadable, a.ID)
			continue
		}
		if found {
			out.Values[a.ID] = values
		}
	}
	return out, nil
}

// WriteFlat sets one action's chords, or hands the action back to the CLI's own
// default. keys empty with reset false writes an empty list, which is how these
// CLIs unbind; reset true removes the key, and the revision the editor was built
// from is checked before anything is written.
func WriteFlat(f FlatMap, action string, keys []string, reset bool, revision string) error {
	if !f.declares(action) {
		return fmt.Errorf("%s is not an action %s declares", action, f.CLI)
	}
	path, format, err := f.File()
	if err != nil {
		return err
	}
	doc, err := clisettings.OpenDoc(path, format)
	if err != nil {
		return err
	}
	if reset {
		err = doc.RemoveKey(action)
	} else {
		err = doc.SetStrings(action, keys)
	}
	if err != nil {
		return err
	}
	return doc.Save(revision)
}

// ResetFlat hands every action this file binds back to the CLI's default. A key
// the catalog does not know is left alone: it may be another version's action,
// or something the CLI itself put there.
func ResetFlat(f FlatMap, revision string) error {
	path, format, err := f.File()
	if err != nil {
		return err
	}
	doc, err := clisettings.OpenDoc(path, format)
	if err != nil {
		return err
	}
	changed := false
	for _, a := range f.Catalog {
		if _, found, err := doc.Strings(a.ID); err != nil || !found {
			// A row PiCode does not understand is not PiCode's to remove.
			continue
		}
		if err := doc.RemoveKey(a.ID); err != nil {
			return err
		}
		changed = true
	}
	if !changed {
		return nil
	}
	return doc.Save(revision)
}

// declares reports whether the catalog knows an action id.
func (f FlatMap) declares(action string) bool {
	for _, a := range f.Catalog {
		if a.ID == action {
			return true
		}
	}
	return false
}

// OmpFlat declares Omp's key map: one machine-level file the CLI reads, with a
// catalog of 70 actions (omp_catalog.go).
var OmpFlat = FlatMap{
	CLI:      "omp",
	Catalog:  OmpCatalog,
	File:     ompKeyMapFile,
	Platform: ompPlatform(),
}

// ompKeyMapFile resolves the file Omp itself reads and the syntax it is in:
// `keybindings.yml` if it is there, else `.yaml`, else the legacy `.json` —
// which the CLI migrates to YAML the next time it writes, so PiCode edits the
// file that exists rather than shadowing it with a second one. A machine with
// none of the three gets `.yml`, the name the CLI itself would create.
//
// The agent directory is the CLI's own resolution, read out of its bundle
// (`@oh-my-pi/pi-coding-agent` 18.2.8) on 2026-09-21: `$PI_CONFIG_DIR` or
// `~/.omp`, then `profiles/$OMP_PROFILE` (or `$PI_PROFILE`) when a profile is
// active, then `agent`.
func ompKeyMapFile() (string, clisettings.Format, error) {
	dir, err := ompAgentDir()
	if err != nil {
		return "", "", err
	}
	candidates := []struct {
		name   string
		format clisettings.Format
	}{
		{"keybindings.yml", clisettings.FormatYAML},
		{"keybindings.yaml", clisettings.FormatYAML},
		{"keybindings.json", clisettings.FormatJSONC},
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(dir, c.name)); err == nil {
			return filepath.Join(dir, c.name), c.format, nil
		}
	}
	return filepath.Join(dir, "keybindings.yml"), clisettings.FormatYAML, nil
}

func ompAgentDir() (string, error) {
	root := os.Getenv("PI_CONFIG_DIR")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".omp")
	}
	profile := os.Getenv("OMP_PROFILE")
	if profile == "" {
		profile = os.Getenv("PI_PROFILE")
	}
	if profile != "" {
		root = filepath.Join(root, "profiles", profile)
	}
	return filepath.Join(root, "agent"), nil
}

// ompPlatform is the platform name Omp resolves its own defaults with — Node's
// `process.platform`, which is what its bundle tests against. A WSL machine
// reports `linux` to the CLI, whatever PiCode calls the host.
func ompPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "win32"
	case "darwin":
		return "darwin"
	}
	return "linux"
}
