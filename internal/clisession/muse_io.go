package clisession

// Muse Code brief handoff (Fatia 1 of docs/plans/launch-muse-agy.md): a
// positional prompt starts the session with it. Verified against a real
// install (Muse Code 1.3.0, `muse --help` 2026-09-15): "pass a prompt to
// start a session". A session id is never pre-assigned — `resume` only
// reopens an existing uuid — so it is ignored, the way Codex's is.
func (MuseSource) PromptArgs(prompt, _ string) []string {
	return []string{prompt}
}
