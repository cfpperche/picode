package cliinstructions

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Fixes (ADR-0204): a finding may carry a small, exact edit the server
// computes from the files on disk. The tab shows the diff and writes it only
// when the person confirms; a write is refused when any file changed since
// the diff was computed (compared by content hash), never overwrites a
// personal file, stays inside the workspace folder, and never runs git.
//
// Fix ids:
//
//	bridge:<rel>      add an @AGENTS.md import to the CLAUDE.md at rel
//	personal:<rel>    make sure the personal file at rel exists and that the
//	                  .gitignore in its folder names it
//
// The first set is these two, and only these two.

// Change is one file a fix touches.
type Change struct {
	Path   string `json:"path"`   // workspace-relative
	Exists bool   `json:"exists"` // false: the fix creates the file
	Before string `json:"before"` // the file's text now ("" when new)
	After  string `json:"after"`  // the file's text after the fix
	Hash   string `json:"hash"`   // sha256 of Before; "" when new
}

// Fix is a proposed edit, exactly as it would be written.
type Fix struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Changes []Change `json:"changes"`
}

// ErrNoFix: the fix does not apply to the tree as it is now.
var ErrNoFix = errors.New("this fix no longer applies")

// ErrDrift: a file changed since the diff was shown.
var ErrDrift = errors.New("the files changed since this change was shown")

// personalFixNames are the personal files a fix may create.
var personalFixNames = map[string]bool{"CLAUDE.local.md": true, "AGENTS.override.md": true}

// ProposeFix computes one fix against the workspace root.
func ProposeFix(root, id string) (*Fix, error) {
	root, err := canon(root)
	if err != nil {
		return nil, err
	}
	kind, rel, ok := strings.Cut(id, ":")
	if !ok || rel == "" {
		return nil, ErrNoFix
	}
	abs, err := confined(root, rel)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "bridge":
		return proposeBridge(root, rel, abs)
	case "personal":
		return proposePersonal(root, rel, abs)
	}
	return nil, ErrNoFix
}

// ApplyFix recomputes the fix and writes it, unless a file's hash differs
// from the one the person saw.
func ApplyFix(root, id string, seen map[string]string) (*Fix, error) {
	fix, err := ProposeFix(root, id)
	if err != nil {
		return nil, err
	}
	for _, c := range fix.Changes {
		if seen[c.Path] != c.Hash {
			return fix, ErrDrift
		}
	}
	root, _ = canon(root)
	for _, c := range fix.Changes {
		abs, err := confined(root, c.Path)
		if err != nil {
			return nil, err
		}
		mode := os.FileMode(0o644)
		if st, err := os.Lstat(abs); err == nil {
			mode = st.Mode().Perm()
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return nil, err
		}
		if err := writeAtomic(abs, []byte(c.After), mode); err != nil {
			return nil, err
		}
	}
	return fix, nil
}

func proposeBridge(root, rel, abs string) (*Fix, error) {
	name := filepath.ToSlash(rel)
	base := filepath.Base(abs)
	inClaudeDir := filepath.Base(filepath.Dir(abs)) == ".claude"
	if base != "CLAUDE.md" {
		return nil, ErrNoFix
	}
	text, hash, ok := readPlain(abs)
	if !ok {
		return nil, ErrNoFix
	}
	folder := filepath.Dir(abs)
	importLine := "@AGENTS.md"
	if inClaudeDir {
		folder = filepath.Dir(folder)
		importLine = "@../AGENTS.md"
	}
	if _, _, ok := readPlain(filepath.Join(folder, "AGENTS.md")); !ok {
		return nil, ErrNoFix
	}
	f := &File{abs: abs, text: text}
	for _, target := range imports(f) {
		if target == filepath.Join(folder, "AGENTS.md") {
			return nil, ErrNoFix // already bridged
		}
	}
	if !strings.Contains(text, "AGENTS.md") {
		return nil, ErrNoFix
	}
	var after string
	if onlyPointer(text) {
		after = importLine + "\n"
	} else {
		after = importLine + "\n\n" + text
	}
	return &Fix{
		ID:      "bridge:" + name,
		Title:   "Make Claude Code load AGENTS.md from " + name,
		Changes: []Change{{Path: name, Exists: true, Before: text, After: after, Hash: hash}},
	}, nil
}

// onlyPointer: the file says nothing but a sentence sending the reader to
// AGENTS.md (headings aside), so the import can take its place.
func onlyPointer(text string) bool {
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		lines = append(lines, l)
	}
	return len(lines) == 1 && strings.Contains(lines[0], "AGENTS.md") && len(lines[0]) <= 200
}

func proposePersonal(root, rel, abs string) (*Fix, error) {
	name := filepath.Base(abs)
	if !personalFixNames[name] {
		return nil, ErrNoFix
	}
	folder := filepath.Dir(abs)
	var changes []Change
	if st, err := os.Lstat(abs); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		changes = append(changes, Change{Path: filepath.ToSlash(rel), After: ""})
	} else if !st.Mode().IsRegular() {
		return nil, ErrNoFix
	}
	// The .gitignore beside the file, inside the workspace: it names the file
	// wherever the repository root is.
	ignoreAbs := filepath.Join(folder, ".gitignore")
	ignoreRel := filepath.ToSlash(filepath.Join(filepath.Dir(rel), ".gitignore"))
	text, hash, ok := readPlain(ignoreAbs)
	if !ok {
		if _, err := os.Lstat(ignoreAbs); err == nil {
			return nil, ErrNoFix // present but not a plain text file
		}
		changes = append(changes, Change{Path: ignoreRel, After: name + "\n"})
	} else if !names(text, name) {
		after := text
		if after != "" && !strings.HasSuffix(after, "\n") {
			after += "\n"
		}
		changes = append(changes, Change{Path: ignoreRel, Exists: true, Before: text, After: after + name + "\n", Hash: hash})
	}
	if len(changes) == 0 {
		return nil, ErrNoFix
	}
	return &Fix{ID: "personal:" + filepath.ToSlash(rel), Title: "Keep " + filepath.ToSlash(rel) + " personal", Changes: changes}, nil
}

// names reports whether a .gitignore has a line for exactly this file name.
func names(gitignore, name string) bool {
	for _, l := range strings.Split(gitignore, "\n") {
		l = strings.TrimSpace(l)
		if l == name || l == "/"+name || l == "**/"+name {
			return true
		}
	}
	return false
}

// confined resolves rel inside root and refuses anything that leaves it,
// including through a symbolic link.
func confined(root, rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", os.ErrPermission
	}
	abs := filepath.Join(root, clean)
	dir := filepath.Dir(abs)
	if real, err := filepath.EvalSymlinks(dir); err == nil && !within(root, real) {
		return "", os.ErrPermission
	}
	if st, err := os.Lstat(abs); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return "", os.ErrPermission
	}
	return abs, nil
}

// readPlain reads a regular file (not a link) with its content hash.
func readPlain(abs string) (string, string, bool) {
	st, err := os.Lstat(abs)
	if err != nil || !st.Mode().IsRegular() || st.Size() > 1<<20 {
		return "", "", false
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", "", false
	}
	sum := sha256.Sum256(b)
	return string(b), hex.EncodeToString(sum[:]), true
}

func writeAtomic(abs string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".picode-fix-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Chmod(name, mode); err != nil {
		os.Remove(name)
		return err
	}
	return os.Rename(name, abs)
}
