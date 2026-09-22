package clikeys

// The Antigravity CLI's key map catalog (docs/plans/keyboard-pane.md, P3): the
// 36 actions its own keybindings file carries, in the ten namespaces the ids
// use, with the chords the installed build ships.
//
// Read on 2026-09-21 out of `~/.gemini/antigravity-cli/keybindings.json` on this
// machine — which the vendor's docs describe as an *override* layer merged
// per-action over defaults built into the binary, so removing a row returns that
// action to the built-in default rather than leaving a hole — and cross-checked
// against the two vendor doc pages that match the installed build
// (antigravity.google/docs/cli/using and /docs/cli/vim-editor-mode; the
// /docs/cli/reference table has drifted to renamed ids and is not a source).
// Twenty rows carry the vendor's own description from those pages; the other
// sixteen are labelled from their id, which those pages leave unnamed.
//
// Group is PiCode's own, from the id namespace — the CLI has no group field.
// Chords keep the file's spelling (`ctrl+l`, `pgdown`, `esc`, `ctrl+_`): a chord
// here is a string the user writes, and the pane formats it for display from the
// CLI's own vocabulary (ADR-0174).
var AgyCatalog = []Action{
	// CLI
	{ID: "cli.clear_screen", Group: "CLI", Label: "Clear terminal output", Defaults: []string{"ctrl+l"}},
	{ID: "cli.cycle_mode", Group: "CLI", Label: "Cycle mode", Defaults: []string{"shift+tab"}},
	{ID: "cli.enter", Group: "CLI", Label: "Enter", Defaults: []string{"enter"}},
	{ID: "cli.escape", Group: "CLI", Label: "Stop stream, close menus, or clear prompt", Defaults: []string{"ctrl+c", "esc"}},
	{ID: "cli.exit", Group: "CLI", Label: "Terminate CLI TUI session", Defaults: []string{"ctrl+d"}},
	{ID: "cli.suspend", Group: "CLI", Label: "Push CLI session to terminal background", Defaults: []string{"ctrl+z"}},
	// Confirm
	{ID: "confirm.edit_command", Group: "Confirm", Label: "Open editor to edit proposed terminal command", Defaults: []string{"e"}},
	{ID: "confirm.no", Group: "Confirm", Label: "Decline terminal command execution", Defaults: []string{"n"}},
	{ID: "confirm.yes", Group: "Confirm", Label: "Approve terminal command execution", Defaults: []string{"y"}},
	// Editing
	{ID: "edit.open_editor", Group: "Editing", Label: "Edit prompt inside your default shell editor", Defaults: []string{"ctrl+g"}},
	{ID: "edit.paste", Group: "Editing", Label: "Paste text from your clipboard", Defaults: []string{"ctrl+v"}},
	{ID: "edit.redo", Group: "Editing", Label: "Redo last undone text change", Defaults: []string{"ctrl+shift+z"}},
	{ID: "edit.undo", Group: "Editing", Label: "Undo last text change", Defaults: []string{"ctrl+_", "ctrl+shift+-"}},
	{ID: "edit.yank", Group: "Editing", Label: "Yank/copy selected text", Defaults: []string{"ctrl+y"}},
	// Items
	{ID: "item.delete", Group: "Items", Label: "Delete", Defaults: []string{"f4"}},
	{ID: "item.rename", Group: "Items", Label: "Rename", Defaults: []string{"f2"}},
	// Navigation
	{ID: "navigation.down", Group: "Navigation", Label: "Scroll down in menu lists", Defaults: []string{"down"}},
	{ID: "navigation.go_to_bottom", Group: "Navigation", Label: "Jump TUI view directly to the bottom", Defaults: []string{"ctrl+end"}},
	{ID: "navigation.go_to_top", Group: "Navigation", Label: "Jump TUI view directly to the top", Defaults: []string{"ctrl+home"}},
	{ID: "navigation.half_page_down", Group: "Navigation", Label: "Half page down", Defaults: []string{"shift+down"}},
	{ID: "navigation.half_page_up", Group: "Navigation", Label: "Half page up", Defaults: []string{"shift+up"}},
	{ID: "navigation.left", Group: "Navigation", Label: "Move prompt cursor left", Defaults: []string{"left"}},
	{ID: "navigation.page_down", Group: "Navigation", Label: "Page down", Defaults: []string{"pgdown"}},
	{ID: "navigation.page_up", Group: "Navigation", Label: "Page up", Defaults: []string{"pgup"}},
	{ID: "navigation.right", Group: "Navigation", Label: "Move prompt cursor right", Defaults: []string{"right"}},
	{ID: "navigation.tab", Group: "Navigation", Label: "Auto-complete choices or switch component focus", Defaults: []string{"tab"}},
	{ID: "navigation.up", Group: "Navigation", Label: "Scroll up in menu lists", Defaults: []string{"up"}},
	// Prompt
	{ID: "prompt.insert_newline", Group: "Prompt", Label: "Add newline to prompt without submitting", Defaults: []string{"alt+enter", "ctrl+j", "shift+enter"}},
	// Subagents
	{ID: "subagent.approve_fast", Group: "Subagents", Label: "Approve fast", Defaults: []string{"ctrl+k"}},
	{ID: "subagent.jump_to_waiting", Group: "Subagents", Label: "Jump to waiting", Defaults: []string{"alt+j"}},
	// View
	{ID: "view.review_artifact", Group: "View", Label: "Review artifact", Defaults: []string{"ctrl+r"}},
	{ID: "view.toggle_trajectory", Group: "View", Label: "Toggle trajectory", Defaults: []string{"ctrl+o"}},
	// Vim
	{ID: "vim.insert.insert_newline", Group: "Vim", Label: "Insert.insert newline", Defaults: []string{"alt+enter", "ctrl+j", "enter", "shift+enter"}},
	{ID: "vim.insert.submit", Group: "Vim", Label: "Insert.submit", Defaults: []string{"ctrl+enter", "ctrl+s"}},
	{ID: "vim.normal.submit", Group: "Vim", Label: "Normal.submit", Defaults: []string{"ctrl+enter", "ctrl+s"}},
	// Voice
	{ID: "voice.start_dictation", Group: "Voice", Label: "Start dictation", Defaults: []string{"f5"}},
}
