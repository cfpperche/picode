package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/clicreds"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/clisettings"
)

// The agent CLIs' own files (ADR-0014 amendment, ADR-0179). Pi has its own
// "pi" part; every other CLI gets clis/<cli>/ in the snapshot:
//
//	clis/<cli>/home/<path under $HOME>  single files: settings always,
//	                                    credentials only with secrets
//	clis/<cli>/trees/<path under $HOME> session directories, with sessions
//	clis/<cli>/trees.json               which trees/ entries are directories
//
// Paths come from each CLI's own declarations (clisettings user layers,
// clicreds native files, clisession file roots), so a CLI PiCode learns about
// is backed up by declaring it once. Everything is written 0600: a settings
// file can hold a token. Session stores kept in SQLite are not copied.

var (
	cliSettingsFiles = clisettings.UserFiles
	cliCredFiles     = clicreds.CredentialFiles
	cliSessionRoots  = clisession.FileSessionRoots
)

func cliHome() string {
	h, _ := os.UserHomeDir()
	return h
}

// underHome returns p relative to home, or false when p is outside it.
func underHome(home, p string) (string, bool) {
	if home == "" || p == "" {
		return "", false
	}
	rel, err := filepath.Rel(home, p)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", false
	}
	return rel, true
}

func (e *Engine) snapshotCLIs(dir, prev string, sessions, secrets bool, files *[]FileEnt) error {
	home := cliHome()
	settings := cliSettingsFiles(home)
	creds := map[string][]string{}
	if secrets {
		creds = cliCredFiles()
	}
	roots := map[string]string{}
	if sessions {
		roots = cliSessionRoots()
	}
	ids := map[string]bool{}
	for id := range settings {
		ids[id] = true
	}
	for id := range creds {
		ids[id] = true
	}
	for id := range roots {
		ids[id] = true
	}
	order := make([]string, 0, len(ids))
	for id := range ids {
		if id != "pi" {
			order = append(order, id)
		}
	}
	sort.Strings(order)
	for _, id := range order {
		base := filepath.Join(dir, "clis", id)
		prevBase := ""
		if prev != "" {
			prevBase = filepath.Join(prev, "clis", id)
		}
		single := append(append([]string{}, settings[id]...), creds[id]...)
		for _, f := range single {
			rel, ok := underHome(home, f)
			if !ok {
				continue
			}
			if st, err := os.Stat(f); err != nil || !st.Mode().IsRegular() {
				continue
			}
			dst := filepath.Join(base, "home", rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
				return err
			}
			if err := putFile(f, dst, join(prevBase, filepath.Join("home", rel)), 0o600, files); err != nil {
				return fmt.Errorf("backup: %s %s: %w", id, rel, err)
			}
		}
		if root := roots[id]; root != "" {
			rel, ok := underHome(home, root)
			if st, err := os.Stat(root); ok && err == nil && st.IsDir() {
				if err := walkCopy(root, filepath.Join(base, "trees", rel), join(prevBase, filepath.Join("trees", rel)), files); err != nil {
					return fmt.Errorf("backup: %s sessions: %w", id, err)
				}
				b, _ := json.Marshal([]string{filepath.ToSlash(rel)})
				if err := os.MkdirAll(base, 0o700); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(base, "trees.json"), b, 0o600); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// restoreCLIs puts every clis/<cli>/ part of a snapshot back under $HOME:
// single files one by one (swapRegular, nothing left behind in a directory
// the CLI owns), session trees whole (swapTree). A part the snapshot does not
// carry is left alone.
func restoreCLIs(snapPath string) error {
	home := cliHome()
	root := filepath.Join(snapPath, "clis")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil // an older snapshot, or no CLI files: nothing to put back
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		base := filepath.Join(root, ent.Name())
		homePart := filepath.Join(base, "home")
		if err := filepath.WalkDir(homePart, func(p string, d os.DirEntry, werr error) error {
			if werr != nil || d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(homePart, p)
			if err != nil {
				return nil
			}
			dst := filepath.Join(home, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
				return err
			}
			return swapRegular(p, dst, 0o600, discardReplaced)
		}); err != nil {
			return fmt.Errorf("backup: restore %s: %w", ent.Name(), err)
		}
		b, err := os.ReadFile(filepath.Join(base, "trees.json"))
		if err != nil {
			continue
		}
		var trees []string
		if json.Unmarshal(b, &trees) != nil {
			continue
		}
		for _, rel := range trees {
			src := filepath.Join(base, "trees", filepath.FromSlash(rel))
			if st, err := os.Stat(src); err != nil || !st.IsDir() {
				continue
			}
			dst := filepath.Join(home, filepath.FromSlash(rel))
			if _, ok := underHome(home, dst); !ok {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
				return err
			}
			if err := swapTree(src, dst); err != nil {
				return fmt.Errorf("backup: restore %s sessions: %w", ent.Name(), err)
			}
		}
	}
	return nil
}
