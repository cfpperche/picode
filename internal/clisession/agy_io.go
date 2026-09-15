package clisession

// Antigravity brief handoff (Fatia 1 of docs/plans/launch-muse-agy.md):
// `--prompt-interactive` runs the initial prompt and continues the
// session. Verified against a real install (Antigravity 1.2.3,
// `agy --help` 2026-09-15). `--prompt` would answer once and exit, so it
// is not the handoff shape; a session id is never pre-assigned, so it is
// ignored, the way OpenCode's is.
func (AgySource) PromptArgs(prompt, _ string) []string {
	return []string{"--prompt-interactive", prompt}
}
