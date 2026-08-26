// Package backup takes point-in-time snapshots of the PiCode environment
// to a user-chosen directory (ADR-0014).
package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

const (
	FormatVersion = 1
	RootName      = "picode-backup"
	ManifestName  = "manifest.json"

	KeyDir        = "backup.dir"
	KeyEnabled    = "backup.enabled"
	KeyInterval   = "backup.interval_min"
	KeyKeep       = "backup.keep_days"
	KeySessions   = "backup.sessions"
	KeySecrets    = "backup.secrets"
	KeyLastOK     = "backup.last_ok"
	KeyLastErr    = "backup.last_error"
	KeyLastBytes  = "backup.last_bytes"
	DefaultIntMin = 60
	DefaultKeep   = 10
)

// Engine runs snapshots against one live store.
type Engine struct {
	Store   *store.Store
	DataDir string
	PiDir   string
	Version string
	Now     func() time.Time

	mu sync.Mutex
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now().UTC()
}

func (e *Engine) piDir() string {
	if e.PiDir != "" {
		return e.PiDir
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(h, ".pi", "agent")
}

// Settings is the user-facing backup config.
type Settings struct {
	Dir         string `json:"dir"`
	IntervalMin int    `json:"intervalMin"`
	KeepDays    int    `json:"keepDays"`
	Sessions    bool   `json:"sessions"`
	Secrets     bool   `json:"secrets"`
	Scheduled   bool   `json:"scheduled"`
	Enabled     bool   `json:"enabled"`
	LastOK      string `json:"lastOk"`
	LastError   string `json:"lastError"`
	LastBytes   int64  `json:"lastBytes"`
	SameFS      bool   `json:"sameFs"`
	DestOK      bool   `json:"destOk"`
	Hardlink    bool   `json:"hardlink"`
}

// FileEnt is one file in a snapshot.
type FileEnt struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	SHA  string `json:"sha256"`
	Link bool   `json:"hardlink,omitempty"`
}

// Manifest describes one snapshot directory.
type Manifest struct {
	Format     int       `json:"format"`
	ID         string    `json:"id"`
	Created    string    `json:"created"`
	Hostname   string    `json:"hostname,omitempty"`
	AppVersion string    `json:"appVersion"`
	Schema     int       `json:"schema"`
	Sessions   bool      `json:"sessions"`
	Secrets    bool      `json:"secrets"`
	SameFS     bool      `json:"sameFs"`
	Bytes      int64     `json:"bytes"`
	Files      []FileEnt `json:"files"`
}

// Snapshot is a listed backup.
type Snapshot struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Created    string `json:"created"`
	Bytes      int64  `json:"bytes"`
	Sessions   bool   `json:"sessions"`
	Secrets    bool   `json:"secrets"`
	Schema     int    `json:"schema"`
	Hostname   string `json:"hostname,omitempty"`
	AppVersion string `json:"appVersion,omitempty"`
}

func Root(dest string) string {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return ""
	}
	if strings.EqualFold(filepath.Base(dest), RootName) {
		return dest
	}
	return filepath.Join(dest, RootName)
}

func LoadSettings(st *store.Store, dataDir string) (Settings, error) {
	s := Settings{
		IntervalMin: DefaultIntMin,
		KeepDays:    DefaultKeep,
		Sessions:    true,
		Secrets:     true,
	}
	if st == nil {
		return s, nil
	}
	s.Dir, _, _ = st.GetSetting(KeyDir)
	if v, ok, _ := st.GetSetting(KeyInterval); ok {
		fmt.Sscanf(v, "%d", &s.IntervalMin)
	}
	if v, ok, _ := st.GetSetting(KeyKeep); ok {
		fmt.Sscanf(v, "%d", &s.KeepDays)
	}
	if v, ok, _ := st.GetSetting(KeySessions); ok {
		s.Sessions = v != "0"
	}
	if v, ok, _ := st.GetSetting(KeySecrets); ok {
		s.Secrets = v != "0"
	}
	s.LastOK, _, _ = st.GetSetting(KeyLastOK)
	s.LastError, _, _ = st.GetSetting(KeyLastErr)
	if v, ok, _ := st.GetSetting(KeyLastBytes); ok {
		fmt.Sscanf(v, "%d", &s.LastBytes)
	}
	if s.IntervalMin < 15 {
		s.IntervalMin = 15
	}
	if s.KeepDays < 1 {
		s.KeepDays = 1
	}
	s.Dir = strings.TrimSpace(s.Dir)
	if v, ok, _ := st.GetSetting(KeyEnabled); ok {
		s.Scheduled = v == "1" || v == "true"
	}
	s.Enabled = s.Dir != "" && s.Scheduled
	if s.Dir != "" {
		root := Root(s.Dir)
		if err := ValidateDest(s.Dir, dataDir, defaultPiHome()); err == nil {
			if st, err := os.Stat(root); err == nil && st.IsDir() {
				s.DestOK = true
			} else if err := os.MkdirAll(root, 0o755); err == nil {
				s.DestOK = true
			}
		}
		s.SameFS = SameFS(dataDir, s.Dir)
	}
	return s, nil
}

func SaveSettings(st *store.Store, in Settings) error {
	if st == nil {
		return fmt.Errorf("backup: no store")
	}
	if in.IntervalMin < 15 {
		in.IntervalMin = 15
	}
	if in.KeepDays < 1 {
		in.KeepDays = 1
	}
	bool01 := func(v bool) string {
		if v {
			return "1"
		}
		return "0"
	}
	dir := strings.TrimSpace(in.Dir)
	scheduled := in.Scheduled && dir != ""
	pairs := [][2]string{
		{KeyDir, dir},
		{KeyEnabled, bool01(scheduled)},
		{KeyInterval, fmt.Sprintf("%d", in.IntervalMin)},
		{KeyKeep, fmt.Sprintf("%d", in.KeepDays)},
		{KeySessions, bool01(in.Sessions)},
		{KeySecrets, bool01(in.Secrets)},
	}
	for _, p := range pairs {
		if err := st.SetSetting(p[0], p[1]); err != nil {
			return err
		}
	}
	return nil
}

func defaultPiHome() string { return PiHome() }

// PiHome is ~/.pi (the tree that must not hold the backup dest).
func PiHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(h, ".pi")
}

func (e *Engine) setLast(ok bool, bytes int64, err error) {
	if e.Store == nil {
		return
	}
	if ok {
		_ = e.Store.SetSetting(KeyLastOK, e.now().Format(time.RFC3339))
		_ = e.Store.SetSetting(KeyLastErr, "")
		_ = e.Store.SetSetting(KeyLastBytes, fmt.Sprintf("%d", bytes))
		return
	}
	msg := "backup failed"
	if err != nil {
		msg = err.Error()
	}
	_ = e.Store.SetSetting(KeyLastErr, msg)
}

func Due(s Settings, now time.Time) bool {
	if !s.Enabled {
		return false
	}
	if s.LastOK == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, s.LastOK)
	if err != nil {
		return true
	}
	return !now.Before(t.Add(time.Duration(s.IntervalMin) * time.Minute))
}

func readManifest(dir string) (Manifest, error) {
	var m Manifest
	b, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, err
	}
	return m, nil
}

func writeManifest(dir string, m Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, ManifestName+".tmp")
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, ManifestName))
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func latestComplete(root string) string {
	list, err := List(root)
	if err != nil || len(list) == 0 {
		return ""
	}
	return list[0].Path
}

// List snapshots newest first. Incomplete dirs (no manifest) are ignored.
func List(dest string) ([]Snapshot, error) {
	root := Root(dest)
	ents, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []Snapshot{}, nil
		}
		return nil, err
	}
	var out []Snapshot
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		m, err := readManifest(dir)
		if err != nil || m.ID == "" {
			continue
		}
		out = append(out, Snapshot{
			ID: m.ID, Path: dir, Created: m.Created, Bytes: m.Bytes,
			Sessions: m.Sessions, Secrets: m.Secrets, Schema: m.Schema,
			Hostname: m.Hostname, AppVersion: m.AppVersion,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	return out, nil
}
