package climemory

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// The audit a single file cannot answer. Three questions, all of them about
// the folder rather than the memory:
//
//   - does the index the CLI loads at session start still point at this file?
//     a memory it does not name is on disk and out of reach
//   - how many other memories cite this one? the load-bearing notes rise and
//     the ones nothing refers to become candidates to merge or drop
//   - does this memory cite something that is gone? a `[[link]]` with no file
//     behind it is a dead end the reader only finds by following it
//
// Measured on the owner's own corpus the day this shipped: 54 memories, one
// broken link, 23 cited by nothing, the index at 41 % of the bytes Claude Code
// will actually read.

// wikiLink is the citation syntax Claude Code's own memory instructions use.
var wikiLink = regexp.MustCompile(`\[\[([A-Za-z0-9][A-Za-z0-9._-]*)\]\]`)

// indexLink is how an index row points at a file: a markdown link to it.
var indexLink = regexp.MustCompile(`\(([A-Za-z0-9][A-Za-z0-9._-]*\.md)\)`)

// linkScanCap bounds one file's contribution to the citation pass. A memory is
// a note, not a corpus; past this the tail cannot change who cites whom
// enough to be worth the read.
const linkScanCap = 32 << 10

func survey(dir string, items []Item, s *spec) ([]Item, IndexHealth, error) {
	names := map[string]bool{}
	for _, item := range items {
		if !item.Index {
			names[strings.TrimSuffix(item.ID, filepath.Ext(item.ID))] = true
		}
	}

	health := IndexHealth{LineLimit: s.indexLines, ByteLimit: s.indexBytes}
	indexed := map[string]bool{}
	if raw, err := os.ReadFile(filepath.Join(dir, indexName)); err == nil {
		health.Exists = true
		health.Bytes = len(raw)
		health.Lines = strings.Count(string(raw), "\n")
		if len(raw) > 0 && !strings.HasSuffix(string(raw), "\n") {
			health.Lines++
		}
		for _, m := range indexLink.FindAllStringSubmatch(string(raw), -1) {
			name := strings.TrimSuffix(m[1], ".md")
			indexed[name] = true
			if !names[name] {
				health.Dangling = append(health.Dangling, m[1])
			}
		}
		sort.Strings(health.Dangling)
	}

	cited := map[string]int{}
	broken := map[string][]string{}
	for _, item := range items {
		if item.Index {
			continue
		}
		body, ok := scan(filepath.Join(dir, item.ID))
		if !ok {
			continue
		}
		from := strings.TrimSuffix(item.ID, filepath.Ext(item.ID))
		seen := map[string]bool{}
		for _, m := range wikiLink.FindAllStringSubmatch(body, -1) {
			target := m[1]
			if target == from || seen[target] {
				continue
			}
			seen[target] = true
			if names[target] {
				cited[target]++
				continue
			}
			broken[from] = append(broken[from], target)
		}
	}

	out := make([]Item, 0, len(items))
	for _, item := range items {
		if item.Index {
			out = append(out, item)
			continue
		}
		name := strings.TrimSuffix(item.ID, filepath.Ext(item.ID))
		// Pointers, so a CLI without an index convention reports absence
		// rather than a zero that reads like a finding.
		inIndex := indexed[name]
		count := cited[name]
		item.Indexed = &inIndex
		item.CitedBy = &count
		if b := broken[name]; len(b) > 0 {
			sort.Strings(b)
			item.Broken = b
		}
		out = append(out, item)
	}
	return out, health, nil
}

// scan reads the head of a memory for its citations, bounded.
func scan(path string) (string, bool) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	size := info.Size()
	if size > linkScanCap {
		size = linkScanCap
	}
	buf := make([]byte, size)
	n, _ := f.Read(buf)
	if n <= 0 {
		return "", false
	}
	return string(buf[:n]), true
}
