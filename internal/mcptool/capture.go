package mcptool

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// captureKeep is how many captures one principal keeps on disk: enough to
// scroll back through a task, small enough to never be a disk question.
const captureKeep = 50

// SaveCapture writes a base64 image under dir/<principal>/ and returns its
// path. The MCP result carries the image as a block already; the file is
// for clients that render files but not blocks. Errors are reported, not
// fatal: the block is the answer, the file is a courtesy.
func SaveCapture(dir, principal, name, mime, b64 string, now time.Time) (string, error) {
	if dir == "" || b64 == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	ext := ".png"
	if strings.Contains(mime, "jpeg") {
		ext = ".jpg"
	}
	sub := filepath.Join(dir, safeName(principal))
	if err := os.MkdirAll(sub, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(sub, fmt.Sprintf("%s-%s%s", now.UTC().Format("20060102-150405.000"), safeName(name), ext))
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return "", err
	}
	prune(sub)
	return path, nil
}

func safeName(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		}
		return '_'
	}, s)
	if s == "" {
		return "anonymous"
	}
	return s
}

// prune drops the oldest files past captureKeep. Names sort by time.
func prune(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for len(names) > captureKeep {
		_ = os.Remove(filepath.Join(dir, names[0]))
		names = names[1:]
	}
}
