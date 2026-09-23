package cliinstructions

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ompWorktreeIssue is where the nested-worktree walk is reported upstream.
const ompWorktreeIssue = "https://github.com/can1357/oh-my-pi/issues/13010"

// findings are the lines a person may want to act on. Each speaks only of
// installed CLIs, and each measured case in the study has one: a CLAUDE.md
// that hides AGENTS.md from Claude Code, a file over a CLI's limit, imports
// most CLIs read as text, Omp in a nested worktree, an untrusted Grok folder.
func findings(s *scan, rep *Report) []Finding {
	installed := map[string]bool{}
	name := map[string]string{}
	for _, c := range rep.CLIs {
		installed[c.ID] = c.Installed
		name[c.ID] = c.Name
	}
	byPath := map[string]*File{}
	for _, f := range rep.Files {
		byPath[f.Path] = f
	}
	readers := func(f *File, skip ...string) []string {
		var out []string
		for _, c := range rep.CLIs {
			if installed[c.ID] && !in(skip, c.ID) {
				if st := f.Cells[c.ID].Status; st == StatusReads || st == StatusOnDemand {
					out = append(out, c.Name)
				}
			}
		}
		return out
	}
	open := func(f *File) *Action {
		if f == nil || f.Rel == "" {
			return nil
		}
		return &Action{Kind: "open", Label: "Open " + filepath.Base(f.abs), Path: f.Rel}
	}
	settings := func(cli string) *Action {
		return &Action{Kind: "settings", Label: "Open " + name[cli] + " settings", URL: "#/clis/" + cli + "/settings"}
	}
	var out []Finding
	add := func(id, text string, f *File, clis []string, a *Action) {
		fn := Finding{ID: id, Text: text, CLIs: clis, Action: a}
		if f != nil {
			fn.File = f.Path
		}
		out = append(out, fn)
	}

	// A CLAUDE.md or CLAUDE.local.md that keeps AGENTS.md from Claude Code.
	if installed["claude-code"] {
		done := map[*File]bool{}
		for _, a := range rep.Files {
			c := a.Cells["claude-code"]
			if a.Scope != "project" || a.name != "AGENTS.md" || c.Status != StatusShadowed {
				continue
			}
			win := byPath[c.By]
			if win == nil || done[win] {
				continue
			}
			before := len(out)
			switch {
			case win.name == "CLAUDE.local.md":
				add("claude-local", "CLAUDE.local.md turns AGENTS.md off for Claude Code. Setting Claude Code's Project instructions to “CLAUDE.md and AGENTS.md” keeps both.", win, []string{"claude-code"}, settings("claude-code"))
			case strings.Contains(win.text, "AGENTS.md") && win.Imports == 0:
				add("claude-pointer", win.Path+" sends Claude Code to AGENTS.md in words, so Claude Code does not load AGENTS.md. A line @AGENTS.md in "+filepath.Base(win.abs)+" makes it load.", win, []string{"claude-code"}, open(win))
			case win.sum == a.sum:
				// The same text under both names: every CLI gets the same instructions.
			default:
				if others := readers(a, "claude-code"); len(others) > 0 {
					text := "Claude Code reads " + win.Path + " here, while " + list(others) + " read " + a.Path + "."
					if a.Folder != "" {
						text = "In " + a.Folder + "/, Claude Code reads " + win.name + ", while " + list(others) + " read " + a.name + "."
					}
					add("claude-split", text, win, []string{"claude-code"}, open(win))
				}
			}
			done[win] = len(out) > before
		}
	}

	// Omp in a worktree nested in its main checkout.
	if installed["omp"] && s.main != "" {
		for _, f := range rep.Files {
			if f.dir == s.main && f.Cells["omp"].Status == StatusReads {
				add("omp-worktree", "Omp also reads the main checkout's "+f.name+" in this worktree, so it sees both versions.", f, []string{"omp"},
					&Action{Kind: "link", Label: "Omp issue #13010", URL: ompWorktreeIssue})
				break
			}
		}
	}

	// Grok in an untrusted folder.
	if installed["grok"] {
		for _, f := range rep.Files {
			if f.Cells["grok"].Status == StatusUntrusted {
				add("grok-untrusted", "Grok ignores this folder's instructions until you trust the folder in Grok.", nil, []string{"grok"}, nil)
				break
			}
		}
	}

	// .hermes.md hides every other kind of file from Hermes.
	if installed["hermes"] {
		for _, f := range rep.Files {
			c := f.Cells["hermes"]
			if f.Scope == "project" && (f.name == "AGENTS.md" || f.name == "CLAUDE.md") && c.Status == StatusShadowed {
				if win := byPath[c.By]; win != nil && (win.name == ".hermes.md" || win.name == "HERMES.md") {
					add("hermes-kind", "Hermes reads "+win.Path+" here and skips "+f.Path+".", win, []string{"hermes"}, open(win))
					break
				}
			}
		}
	}

	// AGENTS.override.md, which only three CLIs read.
	for _, f := range rep.Files {
		if f.Scope != "project" || f.name != "AGENTS.override.md" {
			continue
		}
		var ignore []string
		for _, id := range []string{"claude-code", "grok", "opencode", "muse", "agy", "omp"} {
			if installed[id] {
				ignore = append(ignore, name[id])
			}
		}
		if len(ignore) > 0 {
			add("override-partial", f.Path+" is read by Codex, Pi and Hermes only; "+list(ignore)+" read AGENTS.md instead.", f, nil, open(f))
		}
	}

	// Limits: a CLI that cuts the file, and Claude Code's 200-line guidance.
	for _, f := range rep.Files {
		var parts []string
		var clis []string
		for _, c := range rep.CLIs {
			cell := f.Cells[c.ID]
			if !installed[c.ID] || cell.Status != StatusReads {
				continue
			}
			if cell.Cut != "" {
				parts = append(parts, cell.Cut)
				clis = append(clis, c.ID)
			}
			if c.ID == "claude-code" && f.Lines > 200 {
				parts = append(parts, fmt.Sprintf("Claude Code's guidance is under 200 lines (it has %s)", count(f.Lines)))
				clis = append(clis, c.ID)
			}
		}
		if len(parts) > 0 {
			// Hermes and Codex take their limit from a setting; the others only
			// from a shorter file.
			action := open(f)
			for _, id := range clis {
				if id == "hermes" || id == "codex" {
					action = settings(id)
					break
				}
			}
			add("limit", f.Path+" is "+count(int(f.Bytes))+" bytes: "+strings.Join(parts, "; ")+".", f, clis, action)
		}
	}

	// Import lines most CLIs read as plain text.
	for _, f := range rep.Files {
		if f.Imports == 0 {
			continue
		}
		plain := readers(f, "claude-code", "omp", "agy")
		if len(plain) == 0 {
			continue
		}
		lines := "import line"
		if f.Imports > 1 {
			lines = "import lines"
		}
		add("imports", fmt.Sprintf("%s has %d %s. %s read them as plain text; Claude Code and Omp expand @path.", f.Path, f.Imports, lines, list(plain)), f, nil, open(f))
	}

	// Antigravity's shared rules budget.
	if installed["agy"] {
		total := int64(0)
		for _, f := range rep.Files {
			if f.Cells["agy"].Status == StatusReads {
				total += f.Bytes
			}
		}
		if total/4 > 20000 {
			add("agy-budget", "The files Antigravity reads here pass its 20,000-token rules budget, so it keeps the rest as file pointers.", nil, []string{"agy"}, nil)
		}
	}
	if out == nil {
		out = []Finding{}
	}
	return out
}

// list is "A", "A and B", "A, B and C", or "A, B, C and N more".
func list(names []string) string {
	switch n := len(names); {
	case n == 0:
		return ""
	case n == 1:
		return names[0]
	case n <= 3:
		return strings.Join(names[:n-1], ", ") + " and " + names[n-1]
	default:
		return strings.Join(names[:3], ", ") + fmt.Sprintf(" and %d more", n-3)
	}
}
