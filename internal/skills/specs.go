package skills

// Root is one folder a CLI reads skills from. Path is relative to the
// workspace for a workspace root and "~/"-rooted for a machine root.
type Root struct {
	Path string
	// WalkUp: the CLI reads this folder in the workspace and in every parent
	// up to the repository root, nearest first.
	WalkUp bool
	// Recursive: SKILL.md is found at any depth (Hermes' categories, Grok's
	// synced folders); otherwise only <root>/<name>/SKILL.md.
	Recursive bool
}

// Spec is one CLI's declaration: where it reads skills, in its own
// precedence order (the first root that has a name wins), and the vendor
// facts a pane states instead of drawing a control that cannot work.
// Measured 2026-09-23 on the installed versions; the study
// (docs/benchmarks/2026-09-23-skills-marketplace.md §3) carries the receipts.
type Spec struct {
	CLI string
	// Roots in precedence order: every entry names its scope.
	Roots []ScopedRoot
	// ProjectTrust: the CLI reads workspace skills only in a folder the
	// user trusted in that CLI. TrustCommand is the vendor's own line.
	ProjectTrust bool
	TrustCommand string
	TrustNote    string
	// Toggle is how the CLI switches one skill off, in the vendor's words;
	// empty when it has no per-skill switch.
	Toggle string
	// Launch is the CLI's own per-run mechanism for an agent's skill set;
	// empty when there is none.
	Launch string
	// AgentScope: PiCode passes an agent's own skills at launch through
	// Launch (slice 4, measured 2026-09-24), so the pane offers the agent as
	// a scope. AgentOrder says which copy wins when an agent skill and a
	// folder skill share a name.
	AgentScope bool
	AgentOrder string // AgentFoldersWin | AgentWins | AgentNamespaced
}

// How a CLI resolves an agent skill against a folder skill of the same name
// (measured with each CLI's own command list, 2026-09-24).
const (
	AgentFoldersWin = "folders"    // Pi: the first skill discovered keeps the name
	AgentWins       = "agent"      // Omp: skills.customDirectories outrank the folders
	AgentNamespaced = "namespaced" // Claude Code: plugin skills are plugin:name
)

// ScopedRoot pairs a root with the scope it belongs to.
type ScopedRoot struct {
	Scope Scope
	Root
}

func ws(path string, walk, rec bool) ScopedRoot {
	return ScopedRoot{Scope: Workspace, Root: Root{Path: path, WalkUp: walk, Recursive: rec}}
}

func home(path string, rec bool) ScopedRoot {
	return ScopedRoot{Scope: Machine, Root: Root{Path: path, Recursive: rec}}
}

var specs = []Spec{
	{
		CLI: "pi",
		// pi keeps the first skill it discovers: user folders, then the
		// project's (docs/skills.md; internal/slashres reads the same four).
		Roots: []ScopedRoot{
			home("~/.pi/agent/skills", true),
			home("~/.agents/skills", true),
			ws(".pi/skills", false, true),
			ws(".agents/skills", true, true),
		},
		ProjectTrust: true,
		TrustCommand: "/trust",
		TrustNote:    "Pi loads this workspace's skills once the folder is trusted in Pi.",
		Launch:       "pi --skill <folder> for each of the agent's skills; an isolated agent adds --no-skills, and --skill still loads",
		AgentScope:   true,
		AgentOrder:   AgentFoldersWin,
	},
	{
		CLI: "omp",
		// "Load skills from .agent/skills and .agents/skills (project walk-up +
		// user home)" — the provider string in omp's own bundle, 18.2.11.
		Roots: []ScopedRoot{
			ws(".omp/skills", true, true),
			ws(".agents/skills", true, true),
			ws(".agent/skills", true, true),
			ws(".claude/skills", false, false),
			ws(".github/skills", false, false),
			home("~/.omp/agent/skills", true),
			home("~/.agents/skills", true),
			home("~/.agent/skills", true),
		},
		Toggle:     "skills.ignoredSkills in Omp's settings",
		Launch:     "omp --config <overlay> naming the agent's skills in skills.customDirectories; an isolated agent's overlay turns the folders off",
		AgentScope: true,
		AgentOrder: AgentWins,
	},
	{
		CLI: "claude-code",
		// Personal skills outrank project skills (Claude Code's docs).
		Roots: []ScopedRoot{
			home("~/.claude/skills", false),
			ws(".claude/skills", false, false),
		},
		Toggle:     "skillOverrides in Claude Code's settings",
		Launch:     "claude --plugin-dir <folder>: the agent's skills as the picode-agent plugin, invoked as /picode-agent:<name>",
		AgentScope: true,
		AgentOrder: AgentNamespaced,
	},
	{
		CLI: "codex",
		Roots: []ScopedRoot{
			ws(".agents/skills", true, false),
			home("~/.agents/skills", false),
			{Scope: Machine, Root: Root{Path: "/etc/codex/skills"}},
		},
		Toggle: "[[skills.config]] enabled = false in ~/.codex/config.toml",
	},
	{
		CLI: "grok",
		// grok inspect --json, 1.0.41: ~/.agents wins over ~/.claude; an
		// untrusted project contributes no skills.
		Roots: []ScopedRoot{
			ws(".grok/skills", true, true),
			ws(".agents/skills", true, true),
			ws(".claude/skills", true, true),
			home("~/.grok/skills", true),
			home("~/.agents/skills", true),
			home("~/.claude/skills", true),
		},
		ProjectTrust: true,
		TrustCommand: "/hooks-trust",
		TrustNote:    "Grok loads this workspace's skills only in a folder it trusts: run /hooks-trust in Grok there.",
		Toggle:       "[skills] disabled in ~/.grok/config.toml",
	},
	{
		CLI: "hermes",
		Roots: []ScopedRoot{
			ws(".hermes/skills", false, true),
			ws(".agents/skills", false, true),
			home("~/.hermes/skills", true),
		},
		ProjectTrust: true,
		TrustCommand: "hermes skills trust",
		TrustNote:    "Hermes loads this workspace's skills after hermes skills trust.",
		Toggle:       "skills.disabled in ~/.hermes/config.yaml",
		Launch:       "hermes --skills=<name> preloads a skill",
	},
	{
		CLI: "opencode",
		Roots: []ScopedRoot{
			ws(".opencode/skills", true, false),
			ws(".claude/skills", true, false),
			ws(".agents/skills", true, false),
			home("~/.config/opencode/skills", false),
			home("~/.claude/skills", false),
			home("~/.agents/skills", false),
		},
		Toggle: "permission.skill in opencode.json",
		Launch: "permission.skill through OPENCODE_CONFIG_CONTENT",
	},
	{
		CLI: "muse",
		// muse skills list --json, 1.3.0: project .agents > project .claude >
		// user .agents > user .claude.
		Roots: []ScopedRoot{
			ws(".agents/skills", false, false),
			ws(".claude/skills", false, false),
			home("~/.agents/skills", false),
			home("~/.claude/skills", false),
		},
		ProjectTrust: true,
		TrustNote:    "Muse loads this workspace's skills once you trust the folder; it asks the first time you start Muse there.",
		Toggle:       "muse skills disable",
	},
	{
		CLI: "agy",
		Roots: []ScopedRoot{
			ws(".agents/skills", false, false),
			ws(".agent/skills", false, false),
			home("~/.gemini/antigravity-cli/skills", false),
		},
	},
}

// For returns the declaration of one CLI.
func For(cli string) (Spec, bool) {
	for _, s := range specs {
		if s.CLI == cli {
			return s, true
		}
	}
	return Spec{}, false
}

// CLIs lists the CLIs with a declaration, in declaration order.
func CLIs() []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.CLI)
	}
	return out
}
