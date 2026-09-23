package cliinstructions

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Read is what one agent's latest session actually loaded, from the record
// its own CLI wrote — not a prediction. Three CLIs record it today:
//
//	Claude Code  the transcript keeps "Contents of <path> (…instructions…)"
//	Codex        the rollout keeps "# AGENTS.md instructions for <folder>"
//	Grok         the session's prompt_context.json lists agents_md_files
//
// The other six write no such record, so PiCode says nothing for them rather
// than repeating its own prediction under another name.
type Read struct {
	TerminalID string   `json:"terminalId"`
	Name       string   `json:"name"`
	CLI        string   `json:"cli"`
	UpdatedAt  string   `json:"updatedAt,omitempty"`
	Files      []string `json:"files"`
}

// Records reports whether a CLI writes down the instruction files it loaded.
func Records(cli string) bool {
	return cli == "claude-code" || cli == "codex" || cli == "grok"
}

var (
	// Only the reminder's instruction entries: its memory entries are not
	// instruction files.
	claudeContents = regexp.MustCompile(`Contents of (/[^ \\"]+) \([^)]*instructions`)
	codexHeader    = regexp.MustCompile(`# AGENTS\.md instructions for (/[^\\"\n]+)`)
)

// ObservedFiles reads one session record. path is the session file PiCode
// pinned for the terminal (ADR-0084); for Grok it is the folder's
// prompt_history.jsonl and the session's own folder sits beside it.
func ObservedFiles(cli, sessionID, path string) ([]string, bool) {
	switch cli {
	case "claude-code":
		// What the session started with: everything before the first answer.
		// Later lines hold the conversation, which may quote any path.
		return scanOnce(path, claudeContents, "", `"type":"assistant"`)
	case "codex":
		return scanOnce(path, codexHeader, "AGENTS.md", "")
	case "grok":
		if sessionID == "" || path == "" {
			return nil, false
		}
		b, err := os.ReadFile(filepath.Join(filepath.Dir(path), sessionID, "prompt_context.json"))
		if err != nil {
			return nil, false
		}
		var doc struct {
			Files []struct {
				Path string `json:"file_path"`
			} `json:"agents_md_files"`
		}
		if json.Unmarshal(b, &doc) != nil {
			return nil, false
		}
		out := []string{}
		for _, f := range doc.Files {
			if f.Path != "" {
				out = append(out, f.Path)
			}
		}
		return out, true
	}
	return nil, false
}

// scanOnce collects each distinct first group of re in a JSON-lines file,
// in order; join names the file inside a folder the record points at, and a
// line containing stop ends the scan.
func scanOnce(path string, re *regexp.Regexp, join, stop string) ([]string, bool) {
	if path == "" {
		return nil, false
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer fh.Close()
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 0, 256*1024), 32*1024*1024)
	seen := map[string]bool{}
	out := []string{}
	for n := 0; sc.Scan() && n < 20000; n++ {
		if stop != "" && strings.Contains(sc.Text(), stop) {
			break
		}
		for _, m := range re.FindAllStringSubmatch(sc.Text(), -1) {
			p := strings.TrimSpace(m[1])
			if join != "" {
				p = filepath.Join(p, join)
			}
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out, true
}

// Display is how a path shows beside the report's rows: workspace-relative
// inside the workspace, ~/… under home, absolute elsewhere.
func Display(root, home, abs string) string {
	if within(root, abs) {
		if r, err := filepath.Rel(root, abs); err == nil && r != "." {
			return filepath.ToSlash(r)
		}
	}
	if home != "" && within(home, abs) {
		return "~/" + filepath.ToSlash(strings.TrimPrefix(abs, strings.TrimSuffix(home, "/")+"/"))
	}
	return abs
}
