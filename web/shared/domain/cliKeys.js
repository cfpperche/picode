// The Keyboard pane's per-CLI table (docs/plans/keyboard-pane.md).
//
// The server's registry (internal/clikeys) is the authority on what a CLI's key
// map *is*: its shape, where it lives, when it is picked up, how far PiCode's
// support has got. This table owns what the pane *says* — the one pickup
// sentence per state, and, for a CLI whose editor has not shipped, one line
// plus one action (chrome carries state and the next action, never an essay).
//
// TestJSListMatchesTheKeyboardRegistry keeps the ids, states and pickups of the
// two in step; the label and the vendor link live here because they are copy.

export const KEYBOARD_CLIS = [
  { id: "pi", label: "Pi", state: "shipped", pickup: "reload", vocab: "pi" },
  { id: "claude-code", label: "Claude Code", state: "planned", pickup: "live", vocab: "claude", docs: "https://code.claude.com/docs/en/keybindings" },
  { id: "codex", label: "Codex", state: "planned", pickup: "restart", vocab: "codex", docs: "https://developers.openai.com/codex/config-basic" },
  { id: "grok", label: "Grok", state: "refused", pickup: "unknown", vocab: "", docs: "https://docs.x.ai/build/keyboard-shortcuts", listHint: "Ctrl+. inside a Grok session lists its keys" },
  { id: "hermes", label: "Hermes Agent", state: "planned", pickup: "unknown", vocab: "pi", docs: "https://github.com/NousResearch/hermes-agent/issues/4256" },
  { id: "opencode", label: "OpenCode", state: "planned", pickup: "unknown", vocab: "opencode", docs: "https://opencode.ai/docs/keybinds" },
  { id: "muse", label: "Muse Code", state: "refused", pickup: "unknown", vocab: "", docs: "https://ai.developer.meta.com/docs/muse-code/interactive", listHint: "/keymap inside Muse Code lists its keys" },
  { id: "agy", label: "Antigravity", state: "planned", pickup: "unknown", vocab: "agy", docs: "https://antigravity.google/docs/cli/settings" },
  { id: "omp", label: "Omp", state: "planned", pickup: "unknown", vocab: "pi", docs: "https://github.com/can1357/oh-my-pi/blob/main/docs/keybindings.md" },
];

// keyboardRow is the table row for a CLI, or undefined when the pane cannot be
// opened for it at all.
export const keyboardRow = (id) => KEYBOARD_CLIS.find((cli) => cli.id === id);

export const supportsKeyboard = (id) => !!keyboardRow(id);

export const keyboardState = (id) => keyboardRow(id)?.state || "";

// pickupLine is the sentence under the pane's title: when a change lands. A
// vendor that documents nothing says so — the pane never promises a reload.
export function pickupLine(id) {
  const cli = keyboardRow(id);
  if (!cli) return "";
  switch (cli.pickup) {
    case "reload": return cli.label + " applies a change on /reload or the next run.";
    case "live": return cli.label + " picks a change up as you save it.";
    case "restart": return cli.label + " reads this file when it starts — restart it to apply a change.";
    default: return "When " + cli.label + " picks a change up is not documented.";
  }
}

// blockNote is what a pane with no editor shows: one line of state and one
// action. `refused` is a fact about the vendor; `planned` is a fact about us,
// and saying which is the difference between honesty and a promise.
export function blockNote(id) {
  const cli = keyboardRow(id);
  if (!cli) return null;
  if (cli.state === "refused") {
    return {
      line: cli.label + " keeps its keys built in — it does not allow remapping." +
        (cli.listHint ? " " + cli.listHint.charAt(0).toUpperCase() + cli.listHint.slice(1) + "." : ""),
      action: "Open the key list",
      href: cli.docs,
    };
  }
  if (cli.state === "planned") {
    return {
      line: cli.label + " keeps its own key map; PiCode's editor for it has not shipped yet.",
      action: "Open the documentation",
      href: cli.docs,
    };
  }
  return null;
}
