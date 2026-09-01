package automate

// Template is a suggested automation (ADR-0045 v2): local-repo work any
// agent can do with no external integration. Served by
// GET /api/automations/templates and pre-filled into the editor; the
// list is code, not data, like internal/slashres and internal/catalog.
type Template struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Category         string  `json:"category"` // quality | maintenance | reporting | security
	Description      string  `json:"description"`
	Prompt           string  `json:"prompt"`
	Cron             string  `json:"cron"`
	MaxCostUSD       float64 `json:"maxCostUsd"`
	MaxRuns          int     `json:"maxRuns"`
	MaxRunsWindowMin int     `json:"maxRunsWindowMin"`
}

// Template categories.
const (
	CategoryQuality     = "quality"
	CategoryMaintenance = "maintenance"
	CategoryReporting   = "reporting"
	CategorySecurity    = "security"
)

const noMutation = " Do not push, commit, delete or modify anything; report only."

// Templates returns the built-in suggestions in display order.
func Templates() []Template {
	return []Template{
		{
			ID: "morning-brief", Name: "Morning repo brief", Category: CategoryReporting,
			Description: "Every weekday morning: what changed since yesterday and what needs a look.",
			Prompt:      "Summarize this repository's activity since yesterday morning: commits (git log), branches that moved, open pull requests and failing checks if `gh` is available, and any TODO or FIXME added. Call out anything risky or unfinished. Keep it under 200 words, as bullets, newest first." + noMutation,
			Cron:        "0 9 * * 1-5", MaxCostUSD: 0.5, MaxRuns: 2, MaxRunsWindowMin: 1440,
		},
		{
			ID: "nightly-tests", Name: "Nightly test run", Category: CategoryQuality,
			Description: "Every night: run the test suite and explain each failure.",
			Prompt:      "Find this project's test command (Makefile, package.json, go.mod, pyproject or CI config) and run the full suite. Report the result: pass/fail counts, and for every failure the test name, the error, and the most likely cause with the file and line. If everything passes, say so in one line." + noMutation,
			Cron:        "0 2 * * *", MaxCostUSD: 2, MaxRuns: 2, MaxRunsWindowMin: 1440,
		},
		{
			ID: "docs-drift", Name: "Docs drift check", Category: CategoryQuality,
			Description: "Weekly: where README and docs no longer match the code.",
			Prompt:      "Compare the README and the docs folder against the code: commands, flags, routes, configuration keys, file paths and examples. List each place where the docs describe something the code no longer does, or the code has something the docs never mention, with the doc location and the code location. Rank by how misleading it is." + noMutation,
			Cron:        "0 10 * * 3", MaxCostUSD: 1, MaxRuns: 2, MaxRunsWindowMin: 1440,
		},
		{
			ID: "dependency-check", Name: "Outdated dependencies", Category: CategoryMaintenance,
			Description: "Weekly: which dependencies are behind, and which updates are risky.",
			Prompt:      "List this project's outdated dependencies using the right tool for each ecosystem present (npm outdated, go list -m -u all, pip list --outdated, cargo outdated). For each: current version, latest version, and whether the jump is a major. Group into safe to bump now, needs a look, and blocked, with one line of reasoning each." + noMutation,
			Cron:        "0 8 * * 1", MaxCostUSD: 1, MaxRuns: 2, MaxRunsWindowMin: 1440,
		},
		{
			ID: "stale-branches", Name: "Stale branches report", Category: CategoryMaintenance,
			Description: "Weekly: branches already merged or untouched for a month.",
			Prompt:      "List local and remote branches that are either fully merged into the default branch or have had no commits in 30 days. For each: name, last commit date and author, merged or not, and whether it looks safe to delete. End with the exact git commands a human would run to clean up." + noMutation,
			Cron:        "0 17 * * 5", MaxCostUSD: 0.5, MaxRuns: 2, MaxRunsWindowMin: 1440,
		},
		{
			ID: "changelog-draft", Name: "Changelog draft", Category: CategoryReporting,
			Description: "Weekly: draft release notes from the commits since the last tag.",
			Prompt:      "Read the commits since the last git tag (or the last 7 days if there is no tag) and draft changelog entries in Keep a Changelog style: Added, Changed, Fixed, Removed. Write for users, not developers: what they can now do or what stopped breaking. Skip internal refactors unless they change behaviour. Output the markdown only." + noMutation,
			Cron:        "0 16 * * 5", MaxCostUSD: 1, MaxRuns: 2, MaxRunsWindowMin: 1440,
		},
		{
			ID: "vuln-audit", Name: "Vulnerability audit", Category: CategorySecurity,
			Description: "Weekly: known vulnerabilities in dependencies, worst first.",
			Prompt:      "Run the vulnerability scanners that apply to this project (npm audit, govulncheck ./..., pip-audit, cargo audit) and summarize: each finding with severity, the affected package and version, whether the vulnerable code path is actually used here if you can tell, and the version that fixes it. Critical and high first. If there are none, say so in one line." + noMutation,
			Cron:        "0 7 * * 1", MaxCostUSD: 1, MaxRuns: 2, MaxRunsWindowMin: 1440,
		},
	}
}
