package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Provenance is what an installer's lock says about one skill. PiCode reads
// these files; it does not own them (ADR-0196).
type Provenance struct {
	// Installer names the tool whose lock carries the entry: "skills"
	// (Vercel's skills CLI) or "hermes" (the Hermes hub).
	Installer string `json:"installer"`
	// Lock is the file, as the pane may show it.
	Lock       string `json:"lock"`
	Source     string `json:"source,omitempty"`
	SourceType string `json:"sourceType,omitempty"`
	SourceURL  string `json:"sourceUrl,omitempty"`
	Trust      string `json:"trust,omitempty"`
	// Hash is the lock's own record: computedHash for a project lock (the
	// same algorithm as Digest), a GitHub tree SHA for the global one, the
	// Hermes hub's content hash.
	Hash string `json:"hash,omitempty"`
	// Modified: the folder no longer matches the project lock's
	// computedHash (only that lock hashes what is on disk).
	Modified bool `json:"modified,omitempty"`
}

type projectLock struct {
	Version int `json:"version"`
	Skills  map[string]struct {
		Source       string `json:"source"`
		SourceURL    string `json:"sourceUrl"`
		SourceType   string `json:"sourceType"`
		ComputedHash string `json:"computedHash"`
	} `json:"skills"`
}

type globalLock struct {
	Version int `json:"version"`
	Skills  map[string]struct {
		Source          string `json:"source"`
		SourceType      string `json:"sourceType"`
		SourceURL       string `json:"sourceUrl"`
		SkillFolderHash string `json:"skillFolderHash"`
	} `json:"skills"`
}

type hermesLock struct {
	Installed map[string]struct {
		Source      string `json:"source"`
		Identifier  string `json:"identifier"`
		TrustLevel  string `json:"trust_level"`
		ContentHash string `json:"content_hash"`
		InstallPath string `json:"install_path"`
	} `json:"installed"`
}

// locks is every lock a report reads, loaded once per report.
type locks struct {
	project map[string]Provenance // by skill name, workspace installs
	global  map[string]Provenance // by skill name, machine installs
	hermes  map[string]Provenance // by path relative to ~/.hermes/skills
	notes   []string
}

func readJSON(path string, v any) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, json.Unmarshal(b, v)
}

func loadLocks(homeDir, workspace string) locks {
	l := locks{project: map[string]Provenance{}, global: map[string]Provenance{}, hermes: map[string]Provenance{}}
	if workspace != "" {
		var pl projectLock
		ok, err := readJSON(filepath.Join(workspace, "skills-lock.json"), &pl)
		switch {
		case err != nil:
			l.notes = append(l.notes, "skills-lock.json could not be read: "+err.Error())
		case ok:
			for name, e := range pl.Skills {
				l.project[name] = Provenance{Installer: "skills", Lock: "skills-lock.json", Source: e.Source, SourceType: e.SourceType, SourceURL: e.SourceURL, Hash: e.ComputedHash}
			}
		}
	}
	if homeDir != "" {
		var gl globalLock
		ok, err := readJSON(filepath.Join(homeDir, ".agents", ".skill-lock.json"), &gl)
		switch {
		case err != nil:
			l.notes = append(l.notes, "~/.agents/.skill-lock.json could not be read: "+err.Error())
		case ok:
			for name, e := range gl.Skills {
				l.global[name] = Provenance{Installer: "skills", Lock: "~/.agents/.skill-lock.json", Source: e.Source, SourceType: e.SourceType, SourceURL: e.SourceURL, Hash: e.SkillFolderHash}
			}
		}
		var hl hermesLock
		ok, err = readJSON(filepath.Join(homeDir, ".hermes", "skills", ".hub", "lock.json"), &hl)
		switch {
		case err != nil:
			l.notes = append(l.notes, "the Hermes hub lock could not be read: "+err.Error())
		case ok:
			for name, e := range hl.Installed {
				key := strings.Trim(filepath.ToSlash(e.InstallPath), "/")
				if key == "" {
					key = name
				}
				src := e.Identifier
				if src == "" {
					src = e.Source
				}
				l.hermes[key] = Provenance{Installer: "hermes", Lock: "~/.hermes/skills/.hub/lock.json", Source: src, SourceType: e.Source, Trust: e.TrustLevel, Hash: e.ContentHash}
			}
		}
	}
	return l
}
