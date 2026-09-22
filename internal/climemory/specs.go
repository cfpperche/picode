package climemory

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The store declarations. Every path was verified against the nine CLIs
// installed on the owner's machine on 2026-09-20, except where the note says
// otherwise. A store PiCode cannot resolve is reported unresolved with a line
// saying why — never replaced by a plausible guess, because the guess would be
// another project's memory (ADR-0163).

var catalog = []spec{
	{
		id:     "claude-code",
		tier:   TierEditable,
		toggle: "autoMemoryEnabled",
		// Claude Code's index names each memory with a markdown link, its
		// memories cite each other with `[[name]]`, and it loads only the
		// first 200 lines or 25 KB of the index — whichever comes first —
		// dropping the rest silently (vendor docs, read 2026-09-20).
		indexLinks: true,
		indexLines: 200,
		indexBytes: 25 << 10,
		stores: []storeSpec{{
			scope: "workspace",
			label: "This workspace",
			dir:   claudeMemoryDir,
		}},
	},
	{
		id:   "codex",
		tier: TierReadOnly,
		// Chrome carries state and the next action; how the store is built
		// belongs in docs/architecture/cli-memory.md (.pi/skills/uiux-review).
		note: "Codex writes these files itself.",
		// `[features] memories` is the switch; `memories.use_memories` and
		// `memories.generate_memories` narrow it.
		toggle: "features.memories",
		// No clear command: the pane put `codex` on the clipboard as "the
		// vendor's own command", and `codex` on its own does nothing to a
		// memory (visual review, 2026-09-20). Codex clears these from inside
		// its own session, and inventing a flag here would be worse than the
		// note alone. Open question in docs/handoff/open/agent-clis-native.md.
		clear: "",
		stores: []storeSpec{{
			scope: "global",
			label: "Global",
			dir:   fixedDir(".codex", "memories"),
		}},
	},
	{
		id:     "grok",
		tier:   TierReadOnly,
		note:   "Grok writes these files itself.",
		clear:  "grok memory clear",
		toggle: "",
		stores: []storeSpec{
			{scope: "global", label: "Global", dir: fixedDir(".grok", "memory-v2", "global")},
			{scope: "workspace", label: "This workspace", dir: grokWorkspaceDir},
		},
	},
	{
		id:     "hermes",
		tier:   TierEditable,
		toggle: "memory.memory_enabled",
		stores: []storeSpec{{
			scope: "global",
			label: "Global",
			dir:   fixedDir(".hermes", "memories"),
		}},
	},
	{
		id:   "muse",
		tier: TierEditable,
		stores: []storeSpec{{
			scope: "workspace",
			label: "This workspace",
			dir:   workspaceDir(".agents", "memory"),
		}},
	},
	{
		id:     "omp",
		tier:   TierEditable,
		toggle: "memory.backend",
		stores: []storeSpec{{
			scope: "workspace",
			label: "This workspace",
			dir:   ompMemoryDir,
		}},
	},
	{
		id:   "pi",
		tier: TierNone,
		note: "Pi does not keep memory between sessions.",
	},
	{
		id:   "opencode",
		tier: TierNone,
		note: "OpenCode does not keep memory between sessions.",
	},
	{
		id:   "agy",
		tier: TierUnknown,
		note: "PiCode cannot confirm that Antigravity keeps memory between sessions.",
	},
}

func fixedDir(parts ...string) func(Paths) (string, string) {
	return func(p Paths) (string, string) {
		return filepath.Join(append([]string{p.home()}, parts...)...), ""
	}
}

func workspaceDir(parts ...string) func(Paths) (string, string) {
	return func(p Paths) (string, string) {
		if p.Cwd == "" {
			return "", "Open this CLI from a workspace to see its memory."
		}
		return filepath.Join(append([]string{p.Cwd}, parts...)...), ""
	}
}

// claudeMemoryDir reproduces Claude Code's own rule: one folder per project,
// named after the repository so every worktree of it shares one memory, under
// `~/.claude/projects/<project>/memory`. `autoMemoryDirectory` in the user's
// settings moves it, so the caller's settings lookup wins when it is set.
func claudeMemoryDir(p Paths) (string, string) {
	if p.Cwd == "" {
		return "", "Open this CLI from a workspace to see its memory."
	}
	if p.Setting != nil {
		if custom, ok := p.Setting("claude-code", "autoMemoryDirectory"); ok && strings.TrimSpace(custom) != "" {
			dir, why := boundedStore(expand(custom, p.home()), p.home())
			if why != "" {
				return "", why
			}
			return dir, ""
		}
	}
	base := filepath.Join(p.home(), ".claude", "projects")
	candidates := []string{repoRoot(p.Cwd), p.Cwd}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		dir := filepath.Join(base, slugify(candidate), "memory")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, ""
		}
	}
	// Nothing written yet: name where it will appear, so the empty state can
	// say where to look instead of saying nothing.
	root := repoRoot(p.Cwd)
	if root == "" {
		root = p.Cwd
	}
	return filepath.Join(base, slugify(root), "memory"), ""
}

// grokWorkspaceDir matches the folder Grok made for this workspace. Grok names
// it `<folder>-<hash>` and stores the hash nowhere PiCode can read, so this
// matches on the name and refuses when the match is not unique — the one case
// where showing something would mean showing another project's memory.
func grokWorkspaceDir(p Paths) (string, string) {
	if p.Cwd == "" {
		return "", "Open this CLI from a workspace to see its memory."
	}
	base := filepath.Join(p.home(), ".grok", "memory-v2", "workspaces")
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", "Grok has not written workspace memory on this machine yet."
	}
	want := regexp.MustCompile(`^` + regexp.QuoteMeta(filepath.Base(p.Cwd)) + `-[0-9a-f]{6,}$`)
	var hits []string
	for _, e := range entries {
		if e.IsDir() && want.MatchString(e.Name()) {
			hits = append(hits, filepath.Join(base, e.Name()))
		}
	}
	switch len(hits) {
	case 0:
		return "", "Grok has no memory for this workspace yet."
	case 1:
		return hits[0], ""
	default:
		return "", "Several Grok memory folders match this workspace name, so PiCode cannot tell which one is this project's."
	}
}

// ompMemoryDir finds the local backend's folder. Omp keeps memory off by
// default, and the local backend's path is not documented, so PiCode looks in
// the two places it uses and otherwise says the feature is off rather than
// naming a folder that may not be the right one.
func ompMemoryDir(p Paths) (string, string) {
	candidates := []string{}
	if p.Cwd != "" {
		candidates = append(candidates, filepath.Join(p.Cwd, ".omp", "memories"))
	}
	candidates = append(candidates, filepath.Join(p.home(), ".omp", "agent", "memories"))
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, ""
		}
	}
	return "", "Omp keeps memory off until it is turned on in Settings."
}

// repoRoot returns the main repository folder for cwd, following a worktree's
// pointer file back to the repository it belongs to. Empty when cwd is not in
// a repository.
func repoRoot(cwd string) string {
	dir := cwd
	for {
		marker := filepath.Join(dir, ".git")
		info, err := os.Lstat(marker)
		switch {
		case err == nil && info.IsDir():
			return dir
		case err == nil:
			// A worktree's `.git` is a file pointing into the main
			// repository's `.git/worktrees/<name>`.
			body, err := os.ReadFile(marker)
			if err != nil {
				return dir
			}
			pointer := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(body)), "gitdir:"))
			if idx := strings.Index(pointer, string(filepath.Separator)+".git"+string(filepath.Separator)); idx > 0 {
				return pointer[:idx]
			}
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

var nonSlug = regexp.MustCompile(`[^A-Za-z0-9]`)

// slugify is Claude Code's project-directory name: the absolute path with
// every separator and dot turned into a hyphen.
func slugify(path string) string {
	return nonSlug.ReplaceAllString(filepath.Clean(path), "-")
}

func expand(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// boundedStore keeps a store the user relocated inside their own home. The
// setting is honoured because it is the CLI's own, but the memory pane reads
// and writes whatever it names, so `/etc` or `/proc/self` would turn the pane
// into a file browser. Outside the home directory PiCode says where to look
// instead of looking there.
func boundedStore(dir, home string) (string, string) {
	if !filepath.IsAbs(dir) {
		return "", "This CLI's memory folder setting is not an absolute path, so PiCode cannot open it."
	}
	clean := filepath.Clean(dir)
	rel, err := filepath.Rel(home, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "This CLI's memory folder is outside your home directory, so PiCode does not open it here. Open it in Files."
	}
	return clean, ""
}
