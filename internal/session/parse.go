package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// FileEntry is one message's contribution to a window, before any window is
// applied.
//
// It carries no text. A caller reads a session file for counts, costs and
// identities; the message bodies stay in the file, which is what lets the
// dashboard be an aggregate surface rather than a second transcript view.
type FileEntry struct {
	At time.Time
	// Cwd and Name are captured per entry, not per file. A session file may
	// re-declare its folder part-way through (a fork, or a session that
	// moved), and the messages before that line belong to the folder that
	// was current when they were written — the running-truth rule the v1
	// scan already applied and a regression test pins.
	Cwd        string
	Name       string
	Role       string
	Provider   string
	Model      string
	StopReason string
	Cost       float64
	Split      CostSplit
	Usage      TokenTotals
	Tools      []string
}

// FileCompaction is one compaction marker and the folder current when it
// was written.
type FileCompaction struct {
	At  time.Time
	Cwd string
}

// FileStats is one pi session file parsed into window-independent facts.
//
// This exists so a caller can parse once and cut several windows — and
// several *polls* — from the same result. The dashboard reads a 437 MB tree
// every minute; re-parsing it to answer "has anything changed" was the
// single largest cost in the aggregate (measured: 2.3 s of a 2.4 s refresh,
// with every other CLI already cached).
type FileStats struct {
	Entries     []FileEntry
	Compactions []FileCompaction
}

// ParseFile reads one pi session file. A file it cannot open yields the
// zero value rather than an error: one unreadable session must not fail an
// aggregate over hundreds.
//
// Mtime is the fallback timestamp for lines that carry none, so the caller
// passes the value it already stat-ed rather than making this function stat
// again.
func ParseFile(path string, mtime time.Time) FileStats {
	f, err := os.Open(path)
	if err != nil {
		return FileStats{}
	}
	defer f.Close()

	var out FileStats
	cwd := filepath.Base(filepath.Dir(path)) // fallback: pi's encoded folder name
	name := ""
	provider, model := "", ""

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if json.Unmarshal(line, &raw) != nil {
			continue
		}
		switch raw["type"] {
		case "session":
			if c, _ := raw["cwd"].(string); c != "" {
				cwd = c
			}
		case "session_info":
			if n, _ := raw["name"].(string); n != "" {
				name = n
			}
		case "model_change":
			if p, _ := raw["provider"].(string); p != "" {
				provider = p
			}
			if m, _ := raw["modelId"].(string); m != "" {
				model = m
			}
		case "compaction", "compaction_summary":
			// The marker carries its own RFC3339 timestamp — verified on
			// 2026-09-07 across every compaction line on this machine's
			// tree (62/62). mtime stays the fallback for files written
			// before pi added it. Bucketing by mtime alone charged a long
			// session's entire compaction history to the day it was last
			// touched: the fabricated-spike failure ADR-0041 refused for
			// cost, reached through a different door.
			t := mtime
			if s, _ := raw["timestamp"].(string); s != "" {
				if p, err := time.Parse(time.RFC3339, s); err == nil {
					t = p
				}
			}
			out.Compactions = append(out.Compactions, FileCompaction{At: t, Cwd: cwd})
		case "message":
			t := mtime
			if ts := entryTS(raw); ts > 0 {
				// pi writes message.timestamp in epoch milliseconds
				// (JS Date.now() convention), not seconds.
				t = time.UnixMilli(ts)
			}
			facts := factsFrom(raw["message"])
			// An assistant message names its own provider/model and
			// becomes the running truth for the user/tool lines that
			// follow it; a model_change line is only the fallback for
			// messages that carry neither (older pi versions).
			if facts.provider != "" {
				provider = facts.provider
			} else {
				facts.provider = provider
			}
			if facts.model != "" {
				model = facts.model
			} else {
				facts.model = model
			}
			out.Entries = append(out.Entries, FileEntry{
				At:         t,
				Cwd:        cwd,
				Name:       name,
				Role:       facts.role,
				Provider:   facts.provider,
				Model:      facts.model,
				StopReason: facts.stopReason,
				Cost:       costFrom(raw["message"]),
				Split:      costSplitFrom(raw["message"]),
				Usage:      facts.usage,
				Tools:      facts.tools,
			})
		}
	}
	return out
}
