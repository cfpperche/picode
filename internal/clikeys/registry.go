// Package clikeys declares the Keyboard pane's per-CLI facts: what shape of key
// map a CLI keeps, where it lives, when the CLI picks an edit up, and how far
// PiCode's own support has got. P1 (docs/plans/keyboard-pane.md) is this
// registry plus one endpoint; the readers and writers arrive per CLI in P2.
//
// Every vendor fact below carries the source it was read from, because
// declaring another product's behaviour is a claim about software PiCode does
// not own — the rule internal/clisettings already follows for its defaults.
// "unknown" is a first-class pickup: the pane must not promise a reload the
// vendor never documented.
package clikeys

// Keymap is the shape of a CLI's key map on disk — what a reader and writer
// would have to understand.
type Keymap string

const (
	// Flat is one object of action -> chord list (pi, omp, agy).
	Flat Keymap = "flat"
	// Nested is action -> chord list inside a table per context (codex).
	Nested Keymap = "nested"
	// Inverted keys *chords* to actions, grouped by context (claude-code), so a
	// row's "add key" is a different edit.
	Inverted Keymap = "inverted"
	// Partial is a handful of scalar chord settings, not a map (hermes): there
	// is nothing for a key-map writer to do, so its support stays Planned while
	// the settings driver edits the keys it has.
	Partial Keymap = "partial"
	// None is a CLI that will not read a user key map at all (grok, muse).
	None Keymap = "none"
)

// Pickup is when the CLI reads an edit it did not make itself.
type Pickup string

const (
	// PickupLive applies without a restart (claude-code hot-reloads its file).
	PickupLive Pickup = "live"
	// PickupReload applies on the CLI's own reload command (pi's `/reload`).
	PickupReload Pickup = "reload"
	// PickupRestart needs a new process (codex reads its keymap once, at start).
	PickupRestart Pickup = "restart"
	// PickupUnknown is honest: nothing vendor-published says.
	PickupUnknown Pickup = "unknown"
)

// State is how far PiCode's own support has got.
type State string

const (
	// Shipped: PiCode reads and writes this CLI's map today.
	Shipped State = "shipped"
	// Planned: the map exists, the adapter has not shipped, and the pane says
	// so rather than pretending the file is out of reach.
	Planned State = "planned"
	// Refused: the vendor does not allow remapping at all.
	Refused State = "refused"
)

// CLI is one row of the registry.
type CLI struct {
	ID    string
	Label string
	// Keymap, Pickup and Contexts are the vendor's facts; Source is where they
	// were read, dated — the answer to "how do you know?".
	Keymap   Keymap
	Pickup   Pickup
	Contexts []string
	State    State
	// Vocab names the chord spelling family, by id. The pane turns it into
	// display; it is not a wire format.
	Vocab string
	// OwnStore is true for the one CLI whose map PiCode keeps through its own
	// store, API and trust rules (pi, ADR-0101): the pane's envelope wraps that
	// report instead of reading a file. Every other shipped row is read and
	// written by the flat engine over the settings package's document
	// primitives (ADR-0174).
	OwnStore bool
	Source   string
}

// Registry is every CLI the pane can be opened for, in catalog order (the pane
// lists them the way the sidebar does).
var Registry = []CLI{
	{
		ID: "pi", Label: "Pi",
		Keymap: Flat, Pickup: PickupReload, State: Shipped, Vocab: "pi", OwnStore: true,
		Source: "pi 0.86.1 docs/keybindings.md (installed bundle, read 2026-09-21): " +
			"~/<cli>/agent/keybindings.json, applied on /reload or the next run",
	},
	{
		ID: "claude-code", Label: "Claude Code",
		Keymap: Inverted, Pickup: PickupLive, State: Planned, Vocab: "claude",
		Contexts: []string{"Global", "Chat", "Autocomplete", "Settings", "Confirmation", "Tabs", "Help", "Transcript", "HistorySearch", "Task", "ThemePicker", "Attachments", "Footer", "MessageSelector", "DiffDialog", "DiffPanel", "ModelPicker", "EffortSlider", "Select", "Plugin", "Agents", "Scroll"},
		Source: "code.claude.com/docs/en/keybindings (2026-09-21): ~/.claude/keybindings.json, " +
			"bindings[] of {context, bindings}, hot-reloaded, v2.1.18+",
	},
	{
		ID: "codex", Label: "Codex",
		Keymap: Nested, Pickup: PickupRestart, State: Planned, Vocab: "codex",
		Contexts: []string{"global", "composer", "chat", "editor", "vim_normal", "vim_operator", "vim_text_object", "pager", "list", "approval", "agents"},
		Source: "developers.openai.com/codex config reference (2026-09-21): " +
			"[tui.keymap.<context>.<action>] in ~/.codex/config.toml, [] unbinds, one key on two " +
			"actions in a surface is a config error; the runtime keymap is built once at startup",
	},
	{
		ID: "grok", Label: "Grok",
		Keymap: None, Pickup: PickupUnknown, State: Refused,
		Source: "docs.x.ai/build/keyboard-shortcuts (2026-09-21): bindings are built in and cannot " +
			"currently be remapped; ~/.grok/config.toml carries [ui] vim_mode/simple_mode only",
	},
	{
		ID: "hermes", Label: "Hermes Agent",
		Keymap: Partial, Pickup: PickupUnknown, State: Planned, Vocab: "pi",
		Source: "hermes voice guide + NousResearch/hermes-agent#4256 (2026-09-21): three keys are " +
			"configurable in ~/.hermes/config.yaml (voice.record_key, copy_shortcut, " +
			"display.busy_input_mode); the rest are hardcoded in cli.py",
	},
	{
		ID: "opencode", Label: "OpenCode",
		Keymap: Flat, Pickup: PickupUnknown, State: Planned, Vocab: "opencode",
		Source: "opencode.ai/docs/keybinds (2026-09-21): keybinds map in tui.json " +
			"(~/.config/opencode/tui.json or a project one), <leader> sequences, " +
			"\"none\"/false disables; pickup unconfirmed",
	},
	{
		ID: "muse", Label: "Muse Code",
		Keymap: None, Pickup: PickupUnknown, State: Refused,
		Source: "ai.developer.meta.com/docs/muse-code (2026-09-21): /keymap lists the map, it does " +
			"not edit it; settings.json declares no keymap key",
	},
	{
		ID: "agy", Label: "Antigravity",
		Keymap: Flat, Pickup: PickupUnknown, State: Planned, Vocab: "agy",
		Source: "~/.gemini/antigravity-cli/keybindings.json on this machine (read 2026-09-21): " +
			"action -> array of chords, own namespaces (cli./edit./navigation./vim./view./subagent.), " +
			"pgdown-style key names; pickup unmeasured",
	},
	{
		ID: "omp", Label: "Omp",
		Keymap: Flat, Pickup: PickupRestart, State: Shipped, Vocab: "pi",
		Source: "the bundle installed on this machine, @oh-my-pi/pi-coding-agent " +
			"18.2.8 (which pins @oh-my-pi/pi-tui 18.2.8), read 2026-09-21: ONE file " +
			"in its agent dir — keybindings.yml, else .yaml, else the legacy .json, " +
			"which the CLI migrates to YAML on its next own write — flat " +
			"action -> chord | [chord], [] unbinds, no contexts; 70 action ids (32 " +
			"tui.* + 38 app.*), each with a description, and the id namespaces are " +
			"the only grouping the CLI has (the pane's headings are PiCode's). The " +
			"agent dir is the CLI's own resolution: $PI_CONFIG_DIR or ~/.omp, then " +
			"profiles/$OMP_PROFILE or $PI_PROFILE, then agent. Pickup: KeybindingsManager " +
			"has a reload() nothing user-facing is documented to call, so the pane " +
			"reads the file when the manager is " +
			"created, at startup. ",
	},
}

// For returns the row for a CLI id, or nil when the pane cannot be opened for
// it at all.
func For(id string) *CLI {
	for i := range Registry {
		if Registry[i].ID == id {
			return &Registry[i]
		}
	}
	return nil
}

// Supported lists every id the registry knows, in catalog order — the seam a
// test holds against web/shared/domain/cliKeys.js.
func Supported() []string {
	out := make([]string, 0, len(Registry))
	for _, cli := range Registry {
		out = append(out, cli.ID)
	}
	return out
}
