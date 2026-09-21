package clikeys

// The Omp key map's catalog (docs/plans/keyboard-pane.md, P2b): every action id
// the CLI's own keybinding registry declares, with the description it carries
// and the chords it binds out of the box.
//
// Read on 2026-09-21 out of the bundle installed on this machine —
// @oh-my-pi/pi-coding-agent 18.2.8, which pins @oh-my-pi/pi-tui 18.2.8, whose
// published sources carry the same two tables — by taking every
// `"<id>":{defaultKeys…,description…}` entry of the registry: 70 rows in all,
// 32 under `tui.` and 38 under `app.`, and the five that ship unbound. A row is
// an action because it *has* a defaultKeys field; entries with a description and
// no bindings are that CLI's settings keys, not actions, and are not here.
//
// Group is PiCode's own: unlike pi's hotkeys, Omp's registry has no group field,
// so the headings below come from the id namespaces, ordered to read like the
// CLI's manual. An edit to this file that changes a label is a claim about the
// CLI — re-read the bundle first.
var OmpCatalog = []Action{
	// Editing
	{ID: "tui.editor.cursorUp", Group: "Editing", Label: "Move cursor up", Defaults: []string{"up"}},
	{ID: "tui.editor.cursorDown", Group: "Editing", Label: "Move cursor down", Defaults: []string{"down"}},
	{ID: "tui.editor.cursorLeft", Group: "Editing", Label: "Move cursor left", Defaults: []string{"left", "ctrl+b"}},
	{ID: "tui.editor.cursorRight", Group: "Editing", Label: "Move cursor right", Defaults: []string{"right", "ctrl+f"}},
	{ID: "tui.editor.cursorWordLeft", Group: "Editing", Label: "Move cursor word left", Defaults: []string{"alt+left", "ctrl+left", "alt+b"}},
	{ID: "tui.editor.cursorWordRight", Group: "Editing", Label: "Move cursor word right", Defaults: []string{"alt+right", "ctrl+right", "alt+f"}},
	{ID: "tui.editor.cursorLineStart", Group: "Editing", Label: "Move to line start", Defaults: []string{"home", "ctrl+a"}},
	{ID: "tui.editor.cursorLineEnd", Group: "Editing", Label: "Move to line end", Defaults: []string{"end", "ctrl+e"}},
	{ID: "tui.editor.jumpForward", Group: "Editing", Label: "Jump forward to character", Defaults: []string{"ctrl+]"}},
	{ID: "tui.editor.jumpBackward", Group: "Editing", Label: "Jump backward to character", Defaults: []string{"ctrl+alt+]"}},
	{ID: "tui.editor.pageUp", Group: "Editing", Label: "Page up", Defaults: []string{"pageUp"}},
	{ID: "tui.editor.pageDown", Group: "Editing", Label: "Page down", Defaults: []string{"pageDown"}},
	{ID: "tui.editor.deleteCharBackward", Group: "Editing", Label: "Delete character backward", Defaults: []string{"backspace"}},
	{ID: "tui.editor.deleteCharForward", Group: "Editing", Label: "Delete character forward", Defaults: []string{"delete", "ctrl+d"}},
	{ID: "tui.editor.deleteWordBackward", Group: "Editing", Label: "Delete word backward", Defaults: []string{"ctrl+w", "alt+backspace", "ctrl+backspace", "super+alt+backspace"}},
	{ID: "tui.editor.deleteWordForward", Group: "Editing", Label: "Delete word forward", Defaults: []string{"alt+delete", "alt+d", "super+alt+delete", "super+alt+d"}},
	{ID: "tui.editor.deleteToLineStart", Group: "Editing", Label: "Delete to line start", Defaults: []string{"ctrl+u"}},
	{ID: "tui.editor.deleteToLineEnd", Group: "Editing", Label: "Delete to line end", Defaults: []string{"ctrl+k"}},
	{ID: "tui.editor.yank", Group: "Editing", Label: "Yank", Defaults: []string{"ctrl+y"}},
	{ID: "tui.editor.yankPop", Group: "Editing", Label: "Yank pop", Defaults: []string{"alt+y"}},
	{ID: "tui.editor.undo", Group: "Editing", Label: "Undo", Defaults: []string{"ctrl+-", "ctrl+_"}},
	{ID: "tui.editor.spellingSuggestions", Group: "Editing", Label: "Show spelling replacements", Defaults: []string{"ctrl+."}},
	{ID: "app.editor.external", Group: "Editing", Label: "Open external editor", Defaults: []string{"ctrl+g"}},
	// Input
	{ID: "tui.input.newLine", Group: "Input", Label: "Insert newline", Defaults: []string{"shift+enter", "ctrl+j"}},
	{ID: "tui.input.submit", Group: "Input", Label: "Submit input", Defaults: []string{"enter"}},
	{ID: "tui.input.tab", Group: "Input", Label: "Tab / autocomplete", Defaults: []string{"tab"}},
	{ID: "tui.input.copy", Group: "Input", Label: "Copy selection", Defaults: []string{"ctrl+c"}},
	// Lists
	{ID: "tui.select.up", Group: "Lists", Label: "Move selection up", Defaults: []string{"up"}},
	{ID: "tui.select.down", Group: "Lists", Label: "Move selection down", Defaults: []string{"down"}},
	{ID: "tui.select.pageUp", Group: "Lists", Label: "Selection page up", Defaults: []string{"pageUp"}},
	{ID: "tui.select.pageDown", Group: "Lists", Label: "Selection page down", Defaults: []string{"pageDown"}},
	{ID: "tui.select.confirm", Group: "Lists", Label: "Confirm selection", Defaults: []string{"enter"}},
	{ID: "tui.select.cancel", Group: "Lists", Label: "Cancel selection", Defaults: []string{"escape", "ctrl+c"}},
	// App
	{ID: "app.interrupt", Group: "App", Label: "Interrupt current operation", Defaults: []string{"escape"}},
	{ID: "app.clear", Group: "App", Label: "Clear screen or cancel", Defaults: []string{"ctrl+c"}},
	{ID: "app.exit", Group: "App", Label: "Exit application", Defaults: []string{"ctrl+d"}},
	{ID: "app.suspend", Group: "App", Label: "Suspend application", Defaults: []string{"ctrl+z"}},
	// Display
	{ID: "app.display.reset", Group: "Display", Label: "Reset terminal display", Defaults: []string{"alt+l"}},
	// Model
	{ID: "app.model.cycleForward", Group: "Model", Label: "Cycle to next model", Defaults: []string{"ctrl+p"}},
	{ID: "app.model.cycleBackward", Group: "Model", Label: "Cycle to previous model", Defaults: []string{"shift+ctrl+p"}},
	{ID: "app.model.select", Group: "Model", Label: "Select model", Defaults: []string{"alt+m"}},
	{ID: "app.model.selectTemporary", Group: "Model", Label: "Select temporary model for current session", Defaults: []string{"alt+p"}},
	{ID: "app.thinking.cycle", Group: "Model", Label: "Cycle thinking level", Defaults: []string{"shift+tab"}},
	{ID: "app.thinking.toggle", Group: "Model", Label: "Toggle thinking mode", Defaults: []string{"ctrl+t"}},
	// Tools
	{ID: "app.tools.expand", Group: "Tools", Label: "Expand tools", Defaults: []string{"ctrl+o"}},
	{ID: "app.tools.toggleVisibility", Group: "Tools", Label: "Show or hide tool activity", Defaults: []string{"ctrl+shift+o"}},
	// Messages
	{ID: "app.message.followUp", Group: "Messages", Label: "Send follow-up message", Defaults: []string{"ctrl+q", "ctrl+enter"}},
	{ID: "app.message.dequeue", Group: "Messages", Label: "Dequeue message", Defaults: []string{"alt+up", "shift+up"}},
	{ID: "app.retry", Group: "Messages", Label: "Retry last failed assistant turn", Defaults: []string{"f5", "alt+r"}},
	// Clipboard
	{ID: "app.clipboard.pasteImage", Group: "Clipboard", Label: "Paste image or text from clipboard", Defaults: []string{"ctrl+v"}, Alt: map[string][]string{
		"win32":  {"ctrl+v", "alt+v"},
		"darwin": {"ctrl+v", "super+v"},
	}},
	{ID: "app.clipboard.pasteTextRaw", Group: "Clipboard", Label: "Paste text from clipboard as raw text (no collapse)", Defaults: []string{"ctrl+shift+v", "alt+shift+v"}},
	{ID: "app.clipboard.copyLine", Group: "Clipboard", Label: "Copy current line", Defaults: []string{"alt+shift+l"}},
	{ID: "app.clipboard.copyPrompt", Group: "Clipboard", Label: "Copy prompt", Defaults: []string{"alt+shift+c"}},
	// Sessions
	{ID: "app.session.new", Group: "Sessions", Label: "Create new session", Defaults: nil},
	{ID: "app.session.tree", Group: "Sessions", Label: "Show session tree", Defaults: nil},
	{ID: "app.session.fork", Group: "Sessions", Label: "Fork session", Defaults: nil},
	{ID: "app.session.resume", Group: "Sessions", Label: "Resume session", Defaults: nil},
	{ID: "app.session.observe", Group: "Sessions", Label: "Open the agent hub", Defaults: []string{"ctrl+s"}},
	{ID: "app.session.togglePath", Group: "Sessions", Label: "Toggle session path display", Defaults: []string{"ctrl+p"}},
	{ID: "app.session.toggleSort", Group: "Sessions", Label: "Toggle session sort order", Defaults: []string{"ctrl+s"}},
	{ID: "app.session.rename", Group: "Sessions", Label: "Rename session", Defaults: []string{"ctrl+r"}},
	{ID: "app.session.delete", Group: "Sessions", Label: "Delete session", Defaults: []string{"ctrl+d"}},
	{ID: "app.session.deleteNoninvasive", Group: "Sessions", Label: "Delete session (non-invasive)", Defaults: []string{"ctrl+backspace"}},
	{ID: "app.agents.hub", Group: "Sessions", Label: "Open the agent hub", Defaults: []string{"alt+a"}},
	// History
	{ID: "app.history.search", Group: "History", Label: "Search history", Defaults: []string{"ctrl+r"}},
	// Files
	{ID: "app.tree.foldOrUp", Group: "Files", Label: "Fold or move up", Defaults: []string{"ctrl+left", "alt+left"}},
	{ID: "app.tree.unfoldOrDown", Group: "Files", Label: "Unfold or move down", Defaults: []string{"ctrl+right", "alt+right"}},
	// Plan
	{ID: "app.plan.toggle", Group: "Plan", Label: "Toggle plan mode", Defaults: []string{"alt+shift+p"}},
	// Voice
	{ID: "app.stt.toggle", Group: "Voice", Label: "Toggle speech-to-text (default gesture: hold Space)", Defaults: nil},
	{ID: "app.live.toggle", Group: "Voice", Label: "Start or stop live voice mode (/live)", Defaults: []string{"ctrl+l"}},
}
