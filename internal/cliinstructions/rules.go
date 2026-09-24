package cliinstructions

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

// rule is one CLI's reading of the tree. resolve sets a cell for every file
// the CLI has an opinion on; every other file gets fallback's "not read".
type rule struct {
	id, name string
	source   string          // where the rule was read, at which version
	names    map[string]bool // every name this CLI reads somewhere
	resolve  func(s *scan)
	notes    func(s *scan) []string
}

func set(names ...string) map[string]bool {
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}
	return m
}

func (r *rule) fallback(s *scan, f *File) Cell {
	switch {
	case f.Scope == "personal":
		return Cell{Status: StatusNotRead, Why: r.name + " does not read this personal file"}
	case !r.names[f.name]:
		return Cell{Status: StatusNotRead, Why: r.name + " does not read " + f.name}
	case f.Scope == "above":
		return Cell{Status: StatusNotRead, Why: r.name + " does not look in folders above the repository"}
	}
	return Cell{Status: StatusNotRead, Why: r.name + " does not look in this folder"}
}

// The order is the Agent CLIs catalog's (internal/clilaunch).
var rules = []*rule{rulePi, ruleClaude, ruleCodex, ruleGrok, ruleHermes, ruleOpenCode, ruleMuse, ruleAgy, ruleOmp}

func (s *scan) put(f *File, cli string, c Cell) {
	if f == nil {
		return
	}
	if _, ok := f.Cells[cli]; !ok {
		f.Cells[cli] = c
	}
}

func (s *scan) reads(f *File, cli, why string) {
	s.put(f, cli, Cell{Status: StatusReads, Why: why})
}

func (s *scan) shadow(f *File, cli string, by *File, why string) {
	c := Cell{Status: StatusShadowed, Why: why}
	if by != nil {
		c.By = by.Path
	}
	s.put(f, cli, c)
}

func (s *scan) notRead(f *File, cli, why string) {
	s.put(f, cli, Cell{Status: StatusNotRead, Why: why})
}

// cut records a limit on a file the CLI reads.
func (s *scan) cut(f *File, cli, cut string) {
	if f == nil || cut == "" {
		return
	}
	if c, ok := f.Cells[cli]; ok && c.Status == StatusReads {
		c.Cut = cut
		f.Cells[cli] = c
	}
}

// nestedFor applies one verdict to every file below the start with a name
// the CLI reads.
func (s *scan) nestedFor(cli string, names map[string]bool, c Cell) {
	for _, f := range s.nested {
		if names[f.name] {
			s.put(f, cli, c)
		}
	}
}

// aboveFor marks the CLI's names in folders above stop as not read.
func (s *scan) aboveFor(cli string, names map[string]bool, stop, why string) {
	past := false
	for _, d := range s.chain {
		if past {
			for _, f := range s.order {
				if f.dir == d && names[f.name] {
					s.notRead(f, cli, why)
				}
			}
		}
		if d == stop {
			past = true
		}
	}
}

// ── Pi 0.87.1: dist/core/resource-loader.js, read and run on fixtures ──

var piNames = []string{"AGENTS.override.md", "AGENTS.md", "AGENTS.MD", "CLAUDE.md", "CLAUDE.MD"}

var rulePi = &rule{
	id: "pi", name: "Pi", source: "Pi 0.87.1, measured (its loader, run on fixtures)",
	names: set(piNames...),
	resolve: func(s *scan) {
		const why = "Pi reads one file per folder, from the start folder up to /: the first of AGENTS.override.md, AGENTS.md and CLAUDE.md"
		agentDir := filepath.Join(s.home, ".pi", "agent")
		var mine *File
		for _, n := range piNames {
			if f := s.at(agentDir, n); f != nil {
				if mine == nil {
					mine = f
					s.reads(f, "pi", "Pi reads the first of AGENTS.override.md, AGENTS.md and CLAUDE.md in ~/.pi/agent")
				} else {
					s.shadow(f, "pi", mine, "Pi reads one personal file")
				}
			}
		}
		// A worktree nested in its main checkout hides the main checkout's copy
		// of the worktree's own file (findShadowedContextFile).
		var own *File
		hidden := ""
		if s.main != "" {
			for _, n := range piNames {
				if own = s.at(s.top, n); own != nil {
					hidden = filepath.Join(s.main, filepath.Base(own.abs))
					break
				}
			}
		}
		for _, d := range s.chain {
			var first *File
			for _, n := range piNames {
				f := s.at(d, n)
				if f == nil {
					continue
				}
				switch {
				case first == nil && f.abs == hidden:
					first = f
					s.shadow(f, "pi", own, "Pi skips the main checkout's copy inside a worktree nested in it")
				case first == nil:
					first = f
					s.reads(f, "pi", why)
				default:
					s.shadow(f, "pi", first, why)
				}
			}
		}
		s.nestedFor("pi", set(piNames...), Cell{Status: StatusNotRead, Why: "Pi reads the start folder and the folders above it, not the ones below"})
	},
}

// ── Claude Code 2.1.280: the memory docs, the binary's agents-md mod, two probes ──

var (
	claudeFamily = []string{"CLAUDE.md", ".claude/CLAUDE.md", "CLAUDE.local.md"}
	claudeAgents = []string{"AGENTS.md", ".claude/AGENTS.md"}
)

var ruleClaude = &rule{
	id: "claude-code", name: "Claude Code", source: "Claude Code 2.1.280, measured (docs, the binary, two probes)",
	names: set(append(append([]string{}, claudeFamily...), claudeAgents...)...),
	resolve: func(s *scan) {
		mode := claudeMode(s.home)
		if mode == "managed-only" {
			for _, f := range append(append([]*File{}, s.order...), s.nested...) {
				s.notRead(f, "claude-code", "Claude Code's Project instructions setting is “managed only”")
			}
			if f := s.personal[".claude/CLAUDE.md"]; f != nil {
				s.notRead(f, "claude-code", "Claude Code's Project instructions setting is “managed only”")
			}
			return
		}
		if f := s.personal[".claude/CLAUDE.md"]; f != nil {
			s.reads(f, "claude-code", "Claude Code reads ~/.claude/CLAUDE.md in every session")
		}
		// Measured: inside a worktree nested in its main checkout, the main
		// checkout's own files stay out (both probes, AGENTS.md and CLAUDE.md).
		var walk []string
		for _, d := range s.chain {
			if d == s.main {
				for _, f := range s.order {
					if f.dir == d && (in(claudeFamily, f.name) || in(claudeAgents, f.name)) {
						s.notRead(f, "claude-code", "Claude Code skips the main checkout's files inside a worktree nested in it")
					}
				}
				continue
			}
			walk = append(walk, d)
		}
		var nearest *File
		for _, d := range walk {
			for _, n := range claudeFamily {
				if f := s.at(d, n); f != nil {
					if nearest == nil {
						nearest = f
					}
					s.reads(f, "claude-code", "Claude Code reads CLAUDE.md, .claude/CLAUDE.md and CLAUDE.local.md from the start folder and every folder above it")
				}
			}
		}
		for _, d := range walk {
			for _, n := range claudeAgents {
				f := s.at(d, n)
				if f == nil {
					continue
				}
				if by := s.claudeIncludes(f); by != nil {
					s.reads(f, "claude-code", by.name+" includes it ("+importOrLink(by, f)+")")
					continue
				}
				switch {
				case mode == "claude-md":
					s.notRead(f, "claude-code", "Claude Code's Project instructions setting is “CLAUDE.md only”")
				case mode == "claude-md-and-agents-md":
					s.reads(f, "claude-code", "Claude Code reads AGENTS.md beside CLAUDE.md (Project instructions: CLAUDE.md and AGENTS.md)")
				case nearest != nil:
					s.shadow(f, "claude-code", nearest, "Claude Code reads AGENTS.md only where no CLAUDE.md or CLAUDE.local.md exists in the start folder or above it")
				default:
					s.reads(f, "claude-code", "Claude Code reads AGENTS.md when there is no CLAUDE.md here or above")
				}
			}
		}
		for _, f := range s.nested {
			switch {
			case in(claudeFamily, f.name):
				s.put(f, "claude-code", Cell{Status: StatusOnDemand, Why: "Claude Code reads a subfolder's CLAUDE.md when it opens a file there"})
			case in(claudeAgents, f.name):
				own := s.firstOf(f.dir, claudeFamily, false)
				switch {
				case mode == "claude-md":
					s.notRead(f, "claude-code", "Claude Code's Project instructions setting is “CLAUDE.md only”")
				case mode == "claude-md-and-agents-md":
					s.put(f, "claude-code", Cell{Status: StatusOnDemand, Why: "Claude Code reads a subfolder's AGENTS.md when it opens a file there"})
				case own != nil:
					s.shadow(f, "claude-code", own, "that folder has a CLAUDE.md of its own")
				case nearest != nil:
					s.shadow(f, "claude-code", nearest, "Claude Code reads AGENTS.md only where no CLAUDE.md or CLAUDE.local.md exists in the start folder or above it")
				default:
					s.put(f, "claude-code", Cell{Status: StatusOnDemand, Why: "Claude Code reads a subfolder's AGENTS.md when it opens a file there"})
				}
			}
		}
	},
	notes: func(s *scan) []string {
		switch claudeMode(s.home) {
		case "claude-md-and-agents-md":
			return []string{"Project instructions: CLAUDE.md and AGENTS.md"}
		case "claude-md":
			return []string{"Project instructions: CLAUDE.md only"}
		case "managed-only":
			return []string{"Project instructions: managed only"}
		}
		return nil
	},
}

// claudeIncludes is the CLAUDE-family file in f's folder that brings f in
// through an import line or a symbolic link, or nil.
func (s *scan) claudeIncludes(f *File) *File {
	for _, n := range claudeFamily {
		c := s.at(f.dir, n)
		if c == nil {
			continue
		}
		if c.target == f.abs {
			return c
		}
		for _, target := range imports(c) {
			if target == f.abs {
				return c
			}
		}
	}
	return nil
}

func importOrLink(by, f *File) string {
	if by.target == f.abs {
		return "a link to it"
	}
	return "an @" + filepath.Base(f.abs) + " import"
}

// ── Codex 0.156.1: the AGENTS.md docs; a rollout on this machine ──

var ruleCodex = &rule{
	id: "codex", name: "Codex", source: "Codex 0.156.1, from its docs (and a rollout on this machine)",
	names: set("AGENTS.override.md", "AGENTS.md"),
	resolve: func(s *scan) {
		home := filepath.Join(s.home, ".codex")
		cfg := readCodexConfig(filepath.Join(home, "config.toml"))
		var mine *File
		for _, n := range []string{"AGENTS.override.md", "AGENTS.md"} {
			f := s.at(home, n)
			switch {
			case f == nil:
			case f.empty:
				s.notRead(f, "codex", "Codex skips an empty file")
			case mine == nil:
				mine = f
				s.reads(f, "codex", "Codex reads the first non-empty of AGENTS.override.md and AGENTS.md in ~/.codex")
			default:
				s.shadow(f, "codex", mine, "Codex reads one personal file")
			}
		}
		names := append([]string{"AGENTS.override.md", "AGENTS.md"}, cfg.fallbacks...)
		stop := s.top
		if stop == "" {
			stop = s.start
		}
		used := 0
		for _, d := range s.dirs(stop, false) {
			var first *File
			for _, n := range names {
				f := s.at(d, n)
				switch {
				case f == nil:
				case f.empty:
					s.notRead(f, "codex", "Codex skips an empty file")
				case first != nil:
					s.shadow(f, "codex", first, "Codex reads one file per folder: AGENTS.override.md, then AGENTS.md, then the fallback names")
				case used >= cfg.maxBytes:
					first = f
					s.notRead(f, "codex", "past Codex's "+size(cfg.maxBytes)+" budget for the whole chain (project_doc_max_bytes)")
				default:
					first = f
					s.reads(f, "codex", "Codex reads one file per folder, from the repository root down to the start folder")
					if used+int(f.Bytes) > cfg.maxBytes {
						s.cut(f, "codex", "Codex keeps the first "+size(cfg.maxBytes)+" of the whole chain")
					}
					used += int(f.Bytes)
				}
			}
			if !in(names, "CLAUDE.md") {
				s.notRead(s.at(d, "CLAUDE.md"), "codex", "Codex reads CLAUDE.md only when project_doc_fallback_filenames lists it")
			}
		}
		s.aboveFor("codex", set(names...), stop, "Codex starts at the repository root")
		s.nestedFor("codex", set(names...), Cell{Status: StatusNotRead, Why: "Codex reads the start folder and the folders above it, up to the repository root"})
	},
}

// ── Grok 1.0.41: `grok inspect --json` on fixtures, session files ──

var grokNames = []string{"AGENTS.md", "Agents.md", "AGENT.md", "CLAUDE.md", "Claude.md", "CLAUDE.local.md"}

var ruleGrok = &rule{
	id: "grok", name: "Grok", source: "Grok 1.0.41, measured (grok inspect on fixtures)",
	names: set(grokNames...),
	resolve: func(s *scan) {
		for _, n := range []string{".grok/AGENTS.md", ".grok/CLAUDE.md"} {
			s.reads(s.personal[n], "grok", "Grok reads its personal files in ~/.grok, trusted folder or not")
		}
		project := s.top
		if project == "" {
			project = s.start
		}
		trusted := grokTrusted(s.home, project)
		for _, d := range s.dirs(project, false) {
			for _, n := range grokNames {
				f := s.at(d, n)
				switch {
				case f == nil:
				case f.Ignored:
					s.notRead(f, "grok", "Grok skips files that .gitignore excludes")
				case !trusted:
					s.put(f, "grok", Cell{Status: StatusUntrusted, Why: "Grok reads a project's instructions only once you trust the folder in Grok"})
				default:
					s.reads(f, "grok", "Grok reads every AGENTS.md, AGENT.md, CLAUDE.md and CLAUDE.local.md from the repository root down to the start folder")
				}
			}
			// Not measured: Grok's own docs name .claude/ rules, not .claude/CLAUDE.md.
			s.put(s.at(d, ".claude/CLAUDE.md"), "grok", Cell{Status: StatusUnknown, Why: "not measured for Grok"})
		}
		s.aboveFor("grok", set(grokNames...), project, "Grok starts at the repository root")
		s.nestedFor("grok", set(grokNames...), Cell{Status: StatusNotRead, Why: "Grok reads a subfolder's file only for a session started there"})
	},
}

// ── Hermes (git 1769024c): agent/prompt_builder.py, the manifest run on fixtures ──

var ruleHermes = &rule{
	id: "hermes", name: "Hermes", source: "Hermes 1769024c, measured (its source and /context manifest)",
	names: set(".hermes.md", "HERMES.md", "AGENTS.override.md", "AGENTS.md", "agents.md", "CLAUDE.md", "claude.md", ".cursorrules"),
	resolve: func(s *scan) {
		stop := s.top // outside a repository Hermes reads the start folder alone
		explicit := hermesCap(s.home)
		cut := func(f *File) { s.cut(f, "hermes", hermesCut(f.chars, explicit)) }
		// Kind 1: the nearest .hermes.md or HERMES.md up to the repository root.
		var kind *File
		for _, d := range s.dirs(stop, true) {
			for _, n := range []string{".hermes.md", "HERMES.md"} {
				f := s.at(d, n)
				switch {
				case f == nil:
				case f.empty:
					s.notRead(f, "hermes", "Hermes skips an empty file")
				case kind == nil:
					kind = f
					s.reads(f, "hermes", "Hermes loads .hermes.md first and then no other kind of file")
					cut(f)
				default:
					s.shadow(f, "hermes", kind, "Hermes reads the nearest .hermes.md only")
				}
			}
		}
		// Kind 2: the AGENTS.md chain, repository root first.
		var chain []*File
		seen := map[[32]byte]*File{}
		for _, d := range s.dirs(stop, false) {
			var first *File
			for _, n := range []string{"AGENTS.override.md", "AGENTS.md", "agents.md"} {
				f := s.at(d, n)
				switch {
				case f == nil:
				case f.empty:
					s.notRead(f, "hermes", "Hermes skips an empty file")
				case first != nil:
					s.shadow(f, "hermes", first, "Hermes reads one file per folder: AGENTS.override.md, then AGENTS.md")
				case kind != nil:
					first = f
					s.shadow(f, "hermes", kind, "Hermes loads one kind of file per session, and .hermes.md comes first")
				case seen[f.sum] != nil:
					first = f
					s.shadow(f, "hermes", seen[f.sum], "the same text as a file Hermes already read")
				default:
					first = f
					seen[f.sum] = f
					chain = append(chain, f)
					s.reads(f, "hermes", "Hermes reads the AGENTS.md chain from the repository root down to the start folder")
					cut(f)
				}
			}
		}
		if kind == nil && len(chain) > 0 {
			kind = chain[0]
		}
		// Kind 3 and 4: CLAUDE.md, then .cursorrules, in the start folder only.
		for _, n := range []string{"CLAUDE.md", "claude.md", ".cursorrules"} {
			f := s.at(s.start, n)
			switch {
			case f == nil:
			case f.empty:
				s.notRead(f, "hermes", "Hermes skips an empty file")
			case kind != nil:
				s.shadow(f, "hermes", kind, "Hermes loads one kind of file per session: .hermes.md, then AGENTS.md, then CLAUDE.md, then .cursorrules")
			default:
				kind = f
				s.reads(f, "hermes", "Hermes reads CLAUDE.md from the start folder when there is no AGENTS.md")
				cut(f)
			}
		}
		for _, f := range s.order {
			if f.dir != s.start && (f.name == "CLAUDE.md" || f.name == "claude.md" || f.name == ".cursorrules") {
				s.notRead(f, "hermes", "Hermes reads CLAUDE.md only in the start folder")
			}
		}
		s.aboveFor("hermes", rules4Hermes, stop, "Hermes stops at the repository root")
		s.nestedFor("hermes", set(".hermes.md", "HERMES.md", "AGENTS.override.md", "AGENTS.md", "agents.md", "CLAUDE.md", "claude.md"),
			Cell{Status: StatusOnDemand, Why: "Hermes reads a subfolder's file as the agent works there"})
	},
	notes: func(s *scan) []string {
		return []string{"Hermes also refuses a file its prompt-injection scan flags; PiCode cannot predict that scan."}
	},
}

var rules4Hermes = set(".hermes.md", "HERMES.md", "AGENTS.override.md", "AGENTS.md", "agents.md")

// hermesCut is Hermes's context-file limit in words. Without an explicit
// context_file_max_chars the cap is 6% of the model's window in characters
// (4 per token), never under 20,000 or over 500,000, so a file is whole once
// 0.24 × window reaches its length.
func hermesCut(chars, explicit int) string {
	if explicit > 0 {
		if chars > explicit {
			return "Hermes keeps " + count(explicit) + " characters (context_file_max_chars)"
		}
		return ""
	}
	switch {
	case chars <= 20000:
		return ""
	case chars > 500000:
		return "Hermes cuts it on every model (over its 500,000-character ceiling)"
	}
	k := int(math.Ceil(float64(chars) / 0.24 / 1000))
	return fmt.Sprintf("Hermes cuts it on models with a window under %dk tokens", k)
}

// ── OpenCode 1.18.32: the rules docs and the bundle's Instruction service ──

var ruleOpenCode = &rule{
	id: "opencode", name: "OpenCode", source: "OpenCode 1.18.32, from its docs and bundle",
	names: set("AGENTS.md", "CLAUDE.md", "CONTEXT.md"),
	resolve: func(s *scan) {
		var mine *File
		for _, n := range []string{".config/opencode/AGENTS.md", ".claude/CLAUDE.md"} {
			f := s.personal[n]
			switch {
			case f == nil:
			case mine == nil:
				mine = f
				s.reads(f, "opencode", "OpenCode reads ~/.config/opencode/AGENTS.md, or ~/.claude/CLAUDE.md when that is missing")
			default:
				s.shadow(f, "opencode", mine, "OpenCode reads one personal file")
			}
		}
		stop := s.top
		if stop == "" {
			stop = s.chain[len(s.chain)-1]
		}
		walk := s.dirs(stop, true)
		var chosen []*File
		for _, n := range []string{"AGENTS.md", "CLAUDE.md", "CONTEXT.md"} {
			for _, d := range walk {
				if f := s.at(d, n); f != nil {
					chosen = append(chosen, f)
				}
			}
			if len(chosen) > 0 {
				for _, f := range chosen {
					s.reads(f, "opencode", "OpenCode reads every "+n+" from the start folder up to the repository root")
				}
				break
			}
		}
		if len(chosen) > 0 {
			for _, d := range walk {
				for _, n := range []string{"AGENTS.md", "CLAUDE.md", "CONTEXT.md"} {
					if n != chosen[0].name {
						s.shadow(s.at(d, n), "opencode", chosen[0], "OpenCode uses AGENTS.md when there is one anywhere up the walk, CLAUDE.md only when there is none")
					}
				}
			}
		}
		s.aboveFor("opencode", set("AGENTS.md", "CLAUDE.md", "CONTEXT.md"), stop, "OpenCode stops at the repository root")
		s.nestedFor("opencode", set("AGENTS.md", "CLAUDE.md"), Cell{Status: StatusNotRead, Why: "OpenCode reads the start folder and the folders above it"})
	},
}

// ── Muse Code 1.3.0: measured 2026-09-24 (six echo-provider runs; the
// session record names every rules file it loaded) ──

var museNames = []string{"AGENTS.md", "CLAUDE.md", ".agents/AGENTS.md", ".claude/CLAUDE.md"}

var ruleMuse = &rule{
	id: "muse", name: "Muse Code", source: "Muse Code 1.3.0, measured (six runs, its session record)",
	names: set(museNames...),
	resolve: func(s *scan) {
		stop := s.top
		if stop == "" {
			stop = s.start
		}
		for _, d := range s.dirs(stop, true) {
			var first *File
			for _, n := range museNames {
				f := s.at(d, n)
				switch {
				case f == nil:
				case first == nil:
					first = f
					s.reads(f, "muse", "Muse Code reads the first of AGENTS.md, CLAUDE.md, .agents/AGENTS.md and .claude/CLAUDE.md in each folder up to the repository root")
				default:
					s.shadow(f, "muse", first, "Muse Code reads one file per folder")
				}
			}
		}
		s.aboveFor("muse", set(museNames...), stop, "Muse Code stops at the repository root")
		s.nestedFor("muse", set(museNames...), Cell{Status: StatusUnknown, Why: "Muse Code's docs do not say whether it reads a subfolder's file"})
	},
	notes: func(s *scan) []string {
		return []string{"Muse Code reads a project's files only after you trust the workspace, and PiCode cannot see Muse Code's trust list."}
	},
}

// ── Antigravity CLI 1.2.10: its embedded docs, measured 2026-09-24 in
// interactive sessions on fixtures. In a trusted start folder it reads
// GEMINI.md and AGENTS.md from there up to the repository root at a
// session's start, keeping the first 24,000 bytes of a file. Trust is per
// exact folder (~/.gemini/antigravity-cli/settings.json trustedWorkspaces):
// neither a trusted parent nor a trusted parent's child counts. A session
// in an untrusted folder, and every headless `agy -p` run, loads none. ──

var ruleAgy = &rule{
	id: "agy", name: "Antigravity", source: "Antigravity CLI 1.2.10, measured (interactive runs on fixtures)",
	names: set("GEMINI.md", "AGENTS.md"),
	resolve: func(s *scan) {
		stop := s.top
		if stop == "" {
			stop = s.start
		}
		trusted := agyTrusted(s.home, s.start)
		seen := map[[32]byte]*File{}
		for _, d := range s.dirs(stop, false) {
			for _, n := range []string{"GEMINI.md", "AGENTS.md"} {
				f := s.at(d, n)
				switch {
				case f == nil:
				case !trusted:
					s.put(f, "agy", Cell{Status: StatusUntrusted, Why: "Antigravity reads a project's rules only in a folder you trusted in Antigravity, and only that exact folder counts"})
				case seen[f.sum] != nil:
					s.shadow(f, "agy", seen[f.sum], "the same text as a file Antigravity already read")
				default:
					seen[f.sum] = f
					s.reads(f, "agy", "Antigravity reads both GEMINI.md and AGENTS.md from the start folder up to the repository root")
					if f.Bytes > 24000 {
						s.cut(f, "agy", "Antigravity keeps the first 24,000 bytes")
					}
				}
			}
		}
		if f := s.personal[".gemini/GEMINI.md"]; f != nil {
			s.put(f, "agy", Cell{Status: StatusUnknown, Why: "Antigravity's CLI keeps personal rules in ~/.gemini/config/; its docs do not say whether it reads this file"})
		}
		s.aboveFor("agy", set("GEMINI.md", "AGENTS.md"), stop, "Antigravity stops at the repository root")
		s.nestedFor("agy", set("GEMINI.md", "AGENTS.md"), Cell{Status: StatusOnDemand, Why: "Antigravity reads a subfolder's file when the agent opens a file there"})
	},
	notes: func(s *scan) []string {
		return []string{"A headless run (agy -p) loads no project rules, even in a trusted folder."}
	},
}

// ── Omp 18.2.11: the context-file capability and its providers, two probes ──

// ompPriority is the source priority Omp's context-files capability keeps
// one file per folder depth by (the higher wins; a tie goes to the first
// registered, which the probe showed to be AGENTS.md over CLAUDE.md).
var ompPriority = map[string]int{
	".omp/AGENTS.md": 100, ".claude/CLAUDE.md": 80, ".agents/AGENTS.md": 70, ".agent/AGENTS.md": 70,
	".gemini/GEMINI.md": 60, ".github/copilot-instructions.md": 30, "AGENTS.md": 10, "CLAUDE.md": 9,
}

var ompPersonal = []string{
	".omp/agent/AGENTS.md", ".claude/CLAUDE.md", ".agents/AGENTS.md", ".agent/AGENTS.md", ".codex/AGENTS.md",
	".gemini/GEMINI.md", ".config/opencode/AGENTS.md", ".copilot/copilot-instructions.md",
}

var ruleOmp = &rule{
	id: "omp", name: "Omp", source: "Omp 18.2.11, measured (its bundle and source; every provider run 2026-09-24)",
	names: set(".omp/AGENTS.md", ".claude/CLAUDE.md", ".agents/AGENTS.md", ".agent/AGENTS.md", ".gemini/GEMINI.md", ".github/copilot-instructions.md", "AGENTS.md", "CLAUDE.md"),
	resolve: func(s *scan) {
		var mine *File
		for _, n := range ompPersonal {
			f := s.personal[n]
			switch {
			case f == nil:
			case f.empty:
				s.notRead(f, "omp", "Omp skips an empty file")
			case mine == nil:
				mine = f
				s.reads(f, "omp", "Omp reads one personal file, the highest-priority one that exists")
			default:
				s.shadow(f, "omp", mine, "Omp reads one personal file, the highest-priority one that exists")
			}
		}
		repo := s.top
		if repo == "" {
			repo = s.start
		}
		// candidates by folder: which file each provider offers there.
		cand := map[string][]*File{}
		offer := func(d, n string) {
			if f := s.at(d, n); f != nil && !f.empty {
				cand[d] = append(cand[d], f)
			}
		}
		// .claude/CLAUDE.md, .gemini/GEMINI.md and Copilot's file: the start folder only.
		for _, n := range []string{".claude/CLAUDE.md", ".gemini/GEMINI.md", ".github/copilot-instructions.md"} {
			offer(s.start, n)
		}
		// .omp/AGENTS.md: the nearest one up to the repository root.
		for _, d := range s.dirs(repo, true) {
			if f := s.at(d, ".omp/AGENTS.md"); f != nil {
				offer(d, ".omp/AGENTS.md")
				break
			}
		}
		// .agents/AGENTS.md: every folder up to the repository root.
		for _, d := range s.dirs(repo, true) {
			// A tie at priority 70 goes to .agent/ (measured 2026-09-24).
			offer(d, ".agent/AGENTS.md")
			offer(d, ".agents/AGENTS.md")
		}
		// Plain AGENTS.md and CLAUDE.md: the walk that goes past the repository
		// root when the repository sits under home, stopping before home itself.
		for _, d := range s.ompWalk() {
			if strings.HasPrefix(filepath.Base(d), ".") {
				continue // Omp skips a plain file whose folder name starts with a dot
			}
			offer(d, "AGENTS.md")
			offer(d, "CLAUDE.md")
		}
		for d, fs := range cand {
			win := fs[0]
			for _, f := range fs[1:] {
				if ompPriority[f.name] > ompPriority[win.name] {
					win = f
				}
			}
			for _, f := range fs {
				if f == win {
					why := "Omp keeps one file per folder, by source priority"
					if d == s.main {
						why = "Omp's walk reaches the main checkout from a worktree nested in it"
					}
					s.reads(f, "omp", why)
				} else {
					s.shadow(f, "omp", win, "Omp keeps one file per folder: .omp/ first, then .claude/, .agents/, .gemini/, and AGENTS.md before CLAUDE.md")
				}
			}
		}
		for _, f := range s.order {
			if f.dir != s.start && (f.name == ".claude/CLAUDE.md" || f.name == ".gemini/GEMINI.md" || f.name == ".github/copilot-instructions.md") {
				s.notRead(f, "omp", "Omp reads "+f.name+" only in the start folder")
			}
		}
		s.nestedFor("omp", set("AGENTS.md", "CLAUDE.md", ".claude/CLAUDE.md"), Cell{Status: StatusNotRead, Why: "Omp reads the start folder and the folders above it"})
	},
}

// ompWalk is loadStandaloneContextFiles's walk (src/discovery/helpers.ts).
func (s *scan) ompWalk() []string {
	home, cwd, repo := s.home, s.start, s.top
	cwdUnderHome := within(home, cwd)
	repoIsHome := repo != "" && repo == home
	repoUnderHome := repo != "" && within(home, repo) && !repoIsHome
	scanToHome := repo != "" && cwdUnderHome && repoUnderHome
	boundary := s.chain[len(s.chain)-1]
	switch {
	case scanToHome:
		boundary = home
	case repo != "":
		boundary = repo
	case cwdUnderHome:
		boundary = home
	}
	includeBoundary := cwdUnderHome
	if repo != "" {
		includeBoundary = boundary != home || repoIsHome
	}
	var out []string
	for _, d := range s.chain {
		atBoundary := d == boundary
		atHome := scanToHome && d == home
		if !(atHome || (atBoundary && !includeBoundary)) {
			out = append(out, d)
		}
		if atBoundary {
			break
		}
	}
	return out
}

// firstOf is the first of names present in dir.
func (s *scan) firstOf(dir string, names []string, nonEmpty bool) *File {
	for _, n := range names {
		if f := s.at(dir, n); f != nil && (!nonEmpty || !f.empty) {
			return f
		}
	}
	return nil
}

func in(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func size(n int) string {
	if n%1024 == 0 {
		return fmt.Sprintf("%d KiB", n/1024)
	}
	return count(n) + " bytes"
}

func count(n int) string {
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}
