package clisettings

import "path/filepath"

// The declarations. Every key below is the vendor's own name, verified on
// 2026-09-20 against the config files of the nine CLIs installed on the
// owner's machine, or against the vendor documentation fetched the same day
// where the key was absent from the local file. A key PiCode could not verify
// is not declared: a control that writes a name the CLI ignores is worse than
// no control (the 2026-09-17 launch-presets rule, applied to files).
//
// Pi is absent on purpose. It keeps its own editor, API, layers and trust
// rules (ADR-0101); ADR-0163 amends only the registry that said Pi was the
// single CLI with any editor at all.

// Every boolean declares what its CLI does when nobody sets it, because the
// switch has to draw that value. Where it came from, read 2026-09-20:
//
//	claude autoMemoryEnabled      on   — gated only by an explicit `=== false`
//	codex  features.memories      off  — an opt-in `[features]` flag
//	codex  memories.generate/use  on   — on while the feature above is on
//	grok   cli.auto_update        on   — its own config template says so
//	grok   ui.yolo, ui.compact    off  — opt-in behaviours; no source line
//	hermes memory.*_enabled       on   — hermes_cli/config_defaults.py
//	hermes display.show_reasoning on   — same file; this pane had it wrong
//	hermes display.compact        off  — same file
const (
	groupModel     = "Model"
	groupRoles     = "Model roles"
	groupFallbacks = "Fallbacks"
	groupApproval  = "Approvals"
	groupKeyboard  = "Keyboard"
	groupMemory    = "Memory"
	groupSurface   = "Interface"
	groupUpdates   = "Updates"
)

func opts(pairs ...[2]string) []Option {
	out := make([]Option, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, Option{Value: p[0], Label: p[1]})
	}
	return out
}

func plain(values ...string) []Option {
	out := make([]Option, 0, len(values))
	for _, v := range values {
		out = append(out, Option{Value: v, Label: v})
	}
	return out
}

func userFile(parts ...string) func(Paths) string {
	return func(p Paths) string { return filepath.Join(append([]string{p.home()}, parts...)...) }
}

func projectFile(parts ...string) func(Paths) string {
	return func(p Paths) string {
		if p.Cwd == "" {
			return ""
		}
		return filepath.Join(append([]string{p.Cwd}, parts...)...)
	}
}

var catalog = []spec{
	{
		id: "claude-code",
		layers: []layerSpec{
			{scope: "user", label: "Global", format: FormatJSON, file: userFile(".claude", "settings.json")},
			{scope: "project", label: "This workspace", format: FormatJSON, file: projectFile(".claude", "settings.json")},
		},
		fields: []Field{
			{Key: "model", Label: "Model", Kind: KindText, Group: groupModel, Fallback: "Claude Code default", Help: "Alias or full model id."},
			{Key: "permissions.defaultMode", Label: "Approvals", Kind: KindSelect, Group: groupApproval, Fallback: "Auto", Options: opts(
				[2]string{"auto", "Auto"},
				[2]string{"default", "Ask before edits"},
				[2]string{"acceptEdits", "Edit automatically"},
				[2]string{"plan", "Plan only"},
				[2]string{"dontAsk", "Deny unless allowed"},
				[2]string{"bypassPermissions", "Skip all prompts"},
			), Danger: "bypassPermissions", DangerNote: "Runs without permission prompts. For containers or disposable VMs."},
			{Key: "autoMemoryEnabled", Label: "Auto memory", Kind: KindBool, Group: groupMemory, Fallback: "On", DefaultOn: true, Help: "Claude writes its own notes between sessions."},
			{Key: "autoMemoryDirectory", Label: "Memory folder", Kind: KindText, Group: groupMemory, Fallback: "~/.claude/projects/<project>/memory", Help: "Where those notes are kept."},
			{Key: "cleanupPeriodDays", Label: "Keep transcripts for", Kind: KindNumber, Group: groupUpdates, Fallback: "30 days", Help: "Days before old session transcripts are removed. Memory files are never swept."},
		},
	},
	{
		id: "codex",
		// Codex reads one file. There is no per-workspace config, so the pane
		// shows no layer switcher rather than an empty second tab.
		layers: []layerSpec{
			{scope: "user", label: "Global", format: FormatTOML, file: userFile(".codex", "config.toml")},
		},
		fields: []Field{
			{Key: "model", Label: "Model", Kind: KindText, Group: groupModel, Fallback: "Codex default"},
			{Key: "model_reasoning_effort", Label: "Reasoning effort", Kind: KindSelect, Group: groupModel, Fallback: "Codex default", Options: plain("minimal", "low", "medium", "high", "xhigh")},
			{Key: "approval_policy", Label: "Approvals", Kind: KindSelect, Group: groupApproval, Fallback: "Codex default", Options: opts(
				[2]string{"untrusted", "Ask before untrusted commands"},
				[2]string{"on-request", "Ask when the agent requests"},
				[2]string{"on-failure", "Ask after a command fails"},
				[2]string{"never", "Never ask"},
			), Danger: "never", DangerNote: "The agent runs commands without asking."},
			{Key: "sandbox_mode", Label: "Sandbox", Kind: KindSelect, Group: groupApproval, Fallback: "Codex default", Options: opts(
				[2]string{"read-only", "Read only"},
				[2]string{"workspace-write", "Write inside the workspace"},
				[2]string{"danger-full-access", "Full access"},
			), Danger: "danger-full-access", DangerNote: "No sandbox: full file and network access."},
			{Key: "features.memories", Label: "Memories", Kind: KindBool, Group: groupMemory, Fallback: "Off", Help: "Codex turns finished chats into local notes."},
			{Key: "memories.generate_memories", Label: "Write new memories", Kind: KindBool, Group: groupMemory, Fallback: "On while memories are on", DefaultOn: true},
			{Key: "memories.use_memories", Label: "Read memories back", Kind: KindBool, Group: groupMemory, Fallback: "On while memories are on", DefaultOn: true},
		},
	},
	{
		id: "grok",
		layers: []layerSpec{
			{scope: "user", label: "Global", format: FormatTOML, file: userFile(".grok", "config.toml")},
			{scope: "project", label: "This workspace", format: FormatTOML, file: projectFile(".grok", "config.toml")},
		},
		fields: []Field{
			{Key: "models.default", Label: "Model", Kind: KindText, Group: groupModel, Fallback: "Grok default"},
			{Key: "models.default_reasoning_effort", Label: "Reasoning effort", Kind: KindText, Group: groupModel, Fallback: "Grok default", Help: "Effort level; the range depends on the model."},
			{Key: "ui.permission_mode", Label: "Approvals", Kind: KindSelect, Group: groupApproval, Fallback: "Grok default", Options: opts(
				[2]string{"default", "Ask before edits"},
				[2]string{"auto", "Edit automatically"},
				[2]string{"plan", "Plan only"},
				[2]string{"always-approve", "Approve everything"},
				[2]string{"bypassPermissions", "Skip all prompts"},
			), Danger: "always-approve", DangerNote: "Edits and commands go through without a prompt."},
			{Key: "ui.yolo", Label: "Auto-approve tools", Kind: KindBool, Group: groupApproval, Fallback: "Off", Danger: "true", DangerNote: "Every tool call is approved without asking."},
			{Key: "ui.compact_mode", Label: "Compact display", Kind: KindBool, Group: groupSurface, Fallback: "Off"},
			{Key: "cli.auto_update", Label: "Update automatically", Kind: KindBool, Group: groupUpdates, Fallback: "On", DefaultOn: true},
		},
	},
	{
		id: "hermes",
		layers: []layerSpec{
			{scope: "user", label: "Global", format: FormatYAML, file: userFile(".hermes", "config.yaml")},
		},
		fields: []Field{
			{Key: "model.default", Label: "Model", Kind: KindText, Group: groupModel, Fallback: "Hermes default"},
			{Key: "model.provider", Label: "Provider", Kind: KindText, Group: groupModel, Fallback: "Hermes default"},
			{Key: "agent.max_turns", Label: "Turn limit", Kind: KindNumber, Group: groupApproval, Fallback: "20", Help: "How many turns one run may take before it stops."},
			{Key: "display.compact", Label: "Compact display", Kind: KindBool, Group: groupSurface, Fallback: "Off"},
			{Key: "display.show_reasoning", Label: "Show reasoning", Kind: KindBool, Group: groupSurface, Fallback: "On", DefaultOn: true},
			// The three keys Hermes lets you rebind (docs/plans/keyboard-pane.md):
			// the rest are built into cli.py, so this group is the whole of the
			// CLI's keyboard surface. Defaults and value ranges read from the
			// CLI's own hermes_cli/config_defaults.py on 2026-09-21 —
			// display.busy_input_mode "interrupt | queue | steer", copy_shortcut
			// "auto | ctrl_c | ctrl_shift_c | disabled", record_key "ctrl+b".
			{Key: "voice.record_key", Label: "Push-to-talk key", Kind: KindText, Group: groupKeyboard, Fallback: "ctrl+b", Help: "Hold it down to record; the transcription goes into the prompt."},
			{Key: "copy_shortcut", Label: "Copy shortcut", Kind: KindSelect, Group: groupKeyboard, Fallback: "Auto", Options: opts(
				[2]string{"auto", "Automatic"},
				[2]string{"ctrl_c", "Ctrl+C"},
				[2]string{"ctrl_shift_c", "Ctrl+Shift+C"},
				[2]string{"disabled", "Disabled"},
			)},
			{Key: "display.busy_input_mode", Label: "Typing while it works", Kind: KindSelect, Group: groupKeyboard, Fallback: "Interrupt the turn", Options: opts(
				[2]string{"interrupt", "Interrupt the turn"},
				[2]string{"queue", "Queue it for after"},
				[2]string{"steer", "Steer the running turn"},
			)},
			{Key: "memory.memory_enabled", Label: "Agent notes", Kind: KindBool, Group: groupMemory, Fallback: "On", DefaultOn: true, Help: "What Hermes learned about this environment."},
			{Key: "memory.user_profile_enabled", Label: "Profile of you", Kind: KindBool, Group: groupMemory, Fallback: "On", DefaultOn: true},
			{Key: "memory.memory_char_limit", Label: "Notes size limit", Kind: KindNumber, Group: groupMemory, Fallback: "2200 characters"},
			{Key: "memory.user_char_limit", Label: "Profile size limit", Kind: KindNumber, Group: groupMemory, Fallback: "1375 characters"},
		},
	},
	{
		id: "opencode",
		layers: []layerSpec{
			{scope: "user", label: "Global", format: FormatJSONC, file: userFile(".config", "opencode", "opencode.json")},
			{scope: "project", label: "This workspace", format: FormatJSONC, file: projectFile("opencode.json")},
		},
		fields: []Field{
			{Key: "model", Label: "Model", Kind: KindText, Group: groupModel, Fallback: "OpenCode default", Help: "In provider/model form."},
			{Key: "theme", Label: "Theme", Kind: KindText, Group: groupSurface, Fallback: "OpenCode default"},
			// `autoupdate` is `true | false | "notify"` in OpenCode's own
			// schema. A switch cannot hold the third value and its first
			// toggle would destroy it, so PiCode does not offer the row
			// (adversarial review, 2026-09-20).
		},
	},
	{
		id: "muse",
		layers: []layerSpec{
			{scope: "user", label: "Global", format: FormatJSON, file: userFile(".config", "muse", "settings.json")},
		},
		fields: []Field{
			{Key: "model", Label: "Model", Kind: KindText, Group: groupModel, Fallback: "Muse Code default"},
			{Key: "provider", Label: "Provider", Kind: KindText, Group: groupModel, Fallback: "Muse Code default"},
			{Key: "reasoning_effort", Label: "Reasoning effort", Kind: KindText, Group: groupModel, Fallback: "Muse Code default"},
			{Key: "permissions.default_profile", Label: "Permission profile", Kind: KindText, Group: groupApproval, Fallback: "Muse Code default"},
		},
	},
	{
		id: "agy",
		// Antigravity keeps one settings file whose other keys are structures
		// PiCode must not touch: `title` is the reporter PiCode itself
		// installs, `trustedWorkspaces` is a list the CLI maintains. One
		// declared scalar is thin, and it is what is true.
		layers: []layerSpec{
			{scope: "user", label: "Global", format: FormatJSON, file: userFile(".gemini", "antigravity-cli", "settings.json")},
		},
		fields: []Field{
			{Key: "model", Label: "Model", Kind: KindText, Group: groupModel, Fallback: "Antigravity default"},
		},
	},
	{
		id: "omp",
		layers: []layerSpec{
			{scope: "user", label: "Global", format: FormatYAML, file: userFile(".omp", "agent", "config.yml")},
			{scope: "project", label: "This workspace", format: FormatYAML, file: projectFile(".omp", "config.yml")},
		},
		// The role matrix owns `modelRoles.*` (ADR-0181): the rows come from
		// Omp's own catalog plus whatever the files add, so the single
		// `modelRoles.default` row this pane used to declare is now one row of
		// fifteen and is not declared here.
		roles: ompRoles,
		fields: []Field{
			{Key: "modelRoleStorage", Label: "Omp's own picker saves to", Kind: KindSelect, Group: groupRoles, Fallback: "Global", Options: opts(
				[2]string{"global", "Global"},
				[2]string{"project", "This workspace"},
			), Help: "Where a role you pick inside Omp is saved. This pane always writes the layer you are editing."},
			{Key: "defaultThinkingLevel", Label: "Thinking level", Kind: KindSelect, Group: groupModel, Fallback: "High", Options: opts(
				[2]string{"off", "Off"},
				[2]string{"minimal", "Minimal"},
				[2]string{"low", "Low"},
				[2]string{"medium", "Medium"},
				[2]string{"high", "High"},
				[2]string{"xhigh", "Extra high"},
				[2]string{"max", "Max"},
				[2]string{"auto", "Auto"},
			), Help: "Reasoning depth for models that support it. Auto lets Omp pick per turn."},
			{Key: "retry.enabled", Label: "Retry after a provider error", Kind: KindBool, Group: groupFallbacks, Fallback: "On", DefaultOn: true},
			{Key: "retry.maxRetries", Label: "Retry attempts", Kind: KindNumber, Group: groupFallbacks, Fallback: "10", Help: "How many times one request is retried before the chain below is used."},
			{Key: "retry.modelFallback", Label: "Fall back to another model", Kind: KindBool, Group: groupFallbacks, Fallback: "On", DefaultOn: true, Help: "Off leaves the chains below unused."},
			{Key: "retry.fallbackRevertPolicy", Label: "Return to the first model", Kind: KindSelect, Group: groupFallbacks, Fallback: "After its wait time", Options: opts(
				[2]string{"cooldown-expiry", "After its wait time"},
				[2]string{"never", "Never, until the session ends"},
			)},
			{Key: "retry.usageAwareFallback", Label: "Switch before the plan limit", Kind: KindBool, Group: groupFallbacks, Fallback: "Off", Help: "Reads the plan's remaining allowance and switches early."},
			{Key: "retry.usageReservePct", Label: "Reserve margin (%)", Kind: KindNumber, Group: groupFallbacks, Fallback: "10", Help: "Below this much allowance left, the model counts as near its limit."},
			{Key: "retry.usageReservePolicy", Label: "Inside the margin", Kind: KindSelect, Group: groupFallbacks, Fallback: "Ask", Options: opts(
				[2]string{"confirm", "Ask"},
				[2]string{"auto", "Switch without asking"},
				[2]string{"fail-closed", "Stop the turn"},
			)},
			{Key: "retry.waitForUsageReset", Label: "Wait for the allowance to reset", Kind: KindBool, Group: groupFallbacks, Fallback: "Off", Help: "Sleeps until the window resets instead of failing."},
			// The Models pane's two keys (ADR-0181 list kind). Both are arrays,
			// so a project layer that sets one *replaces* the global list
			// rather than extending it (omp's own docs/settings.md: "the most
			// common surprise") — the pane writes a layer's complete list for
			// that reason. `enabledModels` is a hard filter, not a favourite:
			// once it holds anything, omp uses only the models it matches, and
			// none matching means no usable model (`resolveAllowedModels`,
			// src/config/model-resolver.ts, read 2026-09-22).
			{Key: "enabledModels", Label: "Allowed models", Kind: KindList, Pane: "models", Fallback: "Every model Omp can reach"},
			{Key: "disabledProviders", Label: "Hidden providers", Kind: KindList, Pane: "models", Fallback: "None"},
			{Key: "memory.backend", Label: "Memory", Kind: KindSelect, Group: groupMemory, Fallback: "Off", Options: opts(
				[2]string{"off", "Off"},
				[2]string{"local", "On this machine"},
				[2]string{"hindsight", "Hindsight server"},
				[2]string{"mnemopi", "Mnemopi"},
				[2]string{"sharpshooter", "Sharpshooter"},
			), Help: "Where Omp keeps what it learns between sessions."},
			{Key: "symbolPreset", Label: "Symbols", Kind: KindText, Group: groupSurface, Fallback: "Omp default"},
			{Key: "theme.dark", Label: "Dark theme", Kind: KindText, Group: groupSurface, Fallback: "Omp default"},
		},
	},
}
