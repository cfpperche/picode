package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Digest limits: a skill folder larger than this is reported without a
// digest instead of being read whole on every report.
const (
	maxDigestFiles = 1000
	maxDigestBytes = 25 << 20
)

var errTooLarge = errors.New("skill folder is too large to hash")

// Digest is the folder hash Vercel's skills CLI records as computedHash in
// skills-lock.json (src/local-lock.ts, computeSkillFolderHash): sha256 over
// every regular file's slash-separated relative path followed by its bytes,
// files ordered by JavaScript's localeCompare, .git and node_modules skipped.
func Digest(dir string) (string, error) {
	// A skill folder that is itself a link (the Claude Code link to a
	// workspace install) is hashed at its target: WalkDir does not follow a
	// linked root, and the link alone read as "edited" (found 2026-09-24).
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	type file struct{ rel, abs string }
	var files []file
	var total int64
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != dir && (d.Name() == ".git" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // the CLI's readdir skips links and devices too
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		if len(files) >= maxDigestFiles || total > maxDigestBytes {
			return errTooLarge
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		files = append(files, file{rel: filepath.ToSlash(rel), abs: p})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.SliceStable(files, func(i, j int) bool { return localeLess(files[i].rel, files[j].rel) })
	h := sha256.New()
	for _, f := range files {
		h.Write([]byte(f.rel))
		fh, err := os.Open(f.abs)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(h, fh)
		fh.Close()
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// asciiOrder is printable ASCII in the order V8's localeCompare (ICU root
// collation, punctuation not ignorable) sorts single characters, measured
// with Node 24 on 2026-09-23. Letters appear lower before upper: case is a
// tertiary difference, decided only when the case-folded strings tie.
const asciiOrder = " _-,;:!?.'\"()[]{}@*/\\&#%`^+<=>|~$0123456789aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"

var primaryRank = func() map[rune]int {
	m := map[rune]int{}
	rank := 0
	for _, r := range asciiOrder {
		if r >= 'A' && r <= 'Z' {
			m[r] = m[r+('a'-'A')]
			continue
		}
		rank++
		m[r] = rank
	}
	return m
}()

func primary(r rune) int {
	if w, ok := primaryRank[r]; ok {
		return w
	}
	// Outside printable ASCII: after every ASCII weight, by code point. Skill
	// files are ASCII in practice; a non-ASCII path may order differently
	// from the JS CLI, which the report states rather than hides.
	return 1000 + int(r)
}

// localeLess orders two strings the way localeCompare does for ASCII:
// primary weights (case folded) first, then lowercase before uppercase,
// then code points.
func localeLess(a, b string) bool {
	ra, rb := []rune(a), []rune(b)
	for i := 0; i < len(ra) && i < len(rb); i++ {
		if pa, pb := primary(ra[i]), primary(rb[i]); pa != pb {
			return pa < pb
		}
	}
	if len(ra) != len(rb) {
		return len(ra) < len(rb)
	}
	for i := range ra {
		ua, ub := isUpper(ra[i]), isUpper(rb[i])
		if ua != ub {
			return !ua
		}
	}
	return a < b
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
