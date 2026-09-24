package cliinstructions

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sort"
)

// Revisions is which instruction files an agent read, as they stood when it
// was removed (ADR-0194, amended 2026-09-24): a path and a short hash per
// file, never the text. Two exits with the same hashes ran on the same
// instructions; that is what lets outcomes be compared across edits.
type Revisions struct {
	// Source says where the list came from: "observed" is the CLI's own
	// session record (Claude Code, Codex, Grok); "declared" is this
	// package's rules for the CLI and the folder it started in.
	Source string     `json:"source"`
	Files  []Revision `json:"files"`
}

// Revision is one file. SHA is the first 12 hex digits of the SHA-256 of
// its content at removal; "" when the file was gone by then.
type Revision struct {
	Path  string `json:"path"` // workspace-relative, ~/… under home, else absolute
	SHA   string `json:"sha"`
	Bytes int64  `json:"bytes"`
}

// AgentRevisions answers for one agent: root is its workspace folder, start
// the folder its CLI ran in, and sessionID/sessionPath the session PiCode
// pinned (empty when none). The CLI's own record wins when it keeps one;
// otherwise the rules decide. nil when neither can answer.
func AgentRevisions(root, start, cli, sessionID, sessionPath string) *Revisions {
	home, _ := os.UserHomeDir()
	if Records(cli) && sessionPath != "" {
		if files, ok := ObservedFiles(cli, sessionID, sessionPath); ok {
			out := &Revisions{Source: "observed", Files: []Revision{}}
			for _, abs := range files {
				out.Files = append(out.Files, revisionOf(root, home, abs))
			}
			sortRevisions(out.Files)
			return out
		}
	}
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return nil // the workspace folder is gone: nothing to read rules from
	}
	env := Env{Root: root}
	if start != "" && start != root {
		env.Start = start
	}
	rep, err := Resolve(env)
	if err != nil {
		return nil
	}
	out := &Revisions{Source: "declared", Files: []Revision{}}
	for _, f := range rep.Files {
		if c, ok := f.Cells[cli]; !ok || c.Status != StatusReads {
			continue
		}
		out.Files = append(out.Files, Revision{Path: f.Path, SHA: hex.EncodeToString(f.sum[:])[:12], Bytes: f.Bytes})
	}
	sortRevisions(out.Files)
	return out
}

func revisionOf(root, home, abs string) Revision {
	r := Revision{Path: Display(root, home, abs)}
	st, err := os.Stat(abs)
	if err != nil || !st.Mode().IsRegular() {
		return r
	}
	body, err := readCapped(abs, 4<<20)
	if err != nil {
		return r
	}
	sum := sha256.Sum256(body)
	r.SHA, r.Bytes = hex.EncodeToString(sum[:])[:12], st.Size()
	return r
}

func sortRevisions(rs []Revision) {
	sort.Slice(rs, func(i, j int) bool { return rs[i].Path < rs[j].Path })
}
