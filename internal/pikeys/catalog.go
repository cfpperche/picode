package pikeys

// Action is one Pi keybinding id (docs/keybindings.md).
type Action struct {
	ID       string   `json:"id"`
	Group    string   `json:"group"`
	Label    string   `json:"label"`
	Defaults []string `json:"defaults"`
}

// Catalog is the Pi default map. User JSON overrides per id.
var Catalog = []Action{
	{"tui.editor.cursorUp", "Editor", "Move cursor up", []string{"up"}},
	{"tui.editor.cursorDown", "Editor", "Move cursor down", []string{"down"}},
	{"tui.editor.historyPrevious", "Editor", "Previous prompt", nil},
	{"tui.editor.historyNext", "Editor", "Next prompt", nil},
	{"tui.editor.cursorLeft", "Editor", "Move left", []string{"left", "ctrl+b"}},
	{"tui.editor.cursorRight", "Editor", "Move right", []string{"right", "ctrl+f"}},
	{"tui.editor.cursorWordLeft", "Editor", "Word left", []string{"alt+left", "ctrl+left", "alt+b"}},
	{"tui.editor.cursorWordRight", "Editor", "Word right", []string{"alt+right", "ctrl+right", "alt+f"}},
	{"tui.editor.cursorLineStart", "Editor", "Line start", []string{"home", "ctrl+home", "ctrl+a"}},
	{"tui.editor.cursorLineEnd", "Editor", "Line end", []string{"end", "ctrl+end", "ctrl+e"}},
	{"tui.editor.jumpForward", "Editor", "Jump to character", []string{"ctrl+]"}},
	{"tui.editor.jumpBackward", "Editor", "Jump back to character", []string{"ctrl+alt+]"}},
	{"tui.editor.pageUp", "Editor", "Page up", []string{"pageUp", "ctrl+pageUp"}},
	{"tui.editor.pageDown", "Editor", "Page down", []string{"pageDown", "ctrl+pageDown"}},

	{"tui.editor.deleteCharBackward", "Delete", "Delete character", []string{"backspace"}},
	{"tui.editor.deleteCharForward", "Delete", "Delete next character", []string{"delete", "ctrl+d"}},
	{"tui.editor.deleteWordBackward", "Delete", "Delete word", []string{"ctrl+w", "alt+backspace"}},
	{"tui.editor.deleteWordForward", "Delete", "Delete next word", []string{"alt+d", "alt+delete"}},
	{"tui.editor.deleteToLineStart", "Delete", "Delete to line start", []string{"ctrl+u"}},
	{"tui.editor.deleteToLineEnd", "Delete", "Delete to line end", []string{"ctrl+k"}},

	{"tui.input.newLine", "Input", "New line", []string{"shift+enter", "ctrl+j"}},
	{"tui.input.submit", "Input", "Send", []string{"enter"}},
	{"tui.input.tab", "Input", "Tab / complete", []string{"tab"}},

	{"tui.editor.yank", "Clipboard", "Yank", []string{"ctrl+y"}},
	{"tui.editor.yankPop", "Clipboard", "Yank pop", []string{"alt+y"}},
	{"tui.editor.undo", "Clipboard", "Undo", []string{"ctrl+-"}},
	{"tui.input.copy", "Clipboard", "Copy selection", []string{"ctrl+c"}},

	{"tui.select.up", "Lists", "List up", []string{"up"}},
	{"tui.select.down", "Lists", "List down", []string{"down"}},
	{"tui.select.pageUp", "Lists", "List page up", []string{"pageUp"}},
	{"tui.select.pageDown", "Lists", "List page down", []string{"pageDown"}},
	{"tui.select.confirm", "Lists", "Confirm", []string{"enter"}},
	{"tui.select.cancel", "Lists", "Cancel list", []string{"escape", "ctrl+c"}},

	{"tui.altScreen.pageUp", "Transcript", "Transcript page up", []string{"pageUp"}},
	{"tui.altScreen.pageDown", "Transcript", "Transcript page down", []string{"pageDown"}},
	{"tui.altScreen.halfPageUp", "Transcript", "Transcript half page up", nil},
	{"tui.altScreen.halfPageDown", "Transcript", "Transcript half page down", nil},
	{"tui.altScreen.lineUp", "Transcript", "Transcript line up", nil},
	{"tui.altScreen.lineDown", "Transcript", "Transcript line down", nil},
	{"tui.altScreen.previousPrompt", "Transcript", "Previous message", []string{"ctrl+shift+up", "ctrl+up"}},
	{"tui.altScreen.nextPrompt", "Transcript", "Next message", []string{"ctrl+shift+down", "ctrl+down"}},
	{"tui.altScreen.search", "Transcript", "Search transcript", []string{"ctrl+shift+f"}},
	{"tui.altScreen.searchNext", "Transcript", "Next match", []string{"enter", "ctrl+g"}},
	{"tui.altScreen.searchPrevious", "Transcript", "Previous match", []string{"shift+enter", "ctrl+shift+g"}},
	{"tui.altScreen.searchClose", "Transcript", "Close search", []string{"escape"}},
	{"tui.altScreen.top", "Transcript", "Jump to top", []string{"home"}},
	{"tui.altScreen.bottom", "Transcript", "Jump to bottom", []string{"end"}},

	{"app.interrupt", "App", "Cancel", []string{"escape"}},
	{"app.clear", "App", "Clear / exit", []string{"ctrl+c"}},
	{"app.exit", "App", "Exit when empty", []string{"ctrl+d"}},
	{"app.suspend", "App", "Suspend", []string{"ctrl+z"}},
	{"app.editor.external", "App", "Open in editor", []string{"ctrl+g"}},
	{"app.clipboard.pasteImage", "App", "Paste", []string{"ctrl+v"}},

	{"app.session.new", "Sessions", "New session", nil},
	{"app.session.tree", "Sessions", "Session tree", nil},
	{"app.session.fork", "Sessions", "Fork session", nil},
	{"app.session.resume", "Sessions", "Resume session", nil},
	{"app.session.togglePath", "Sessions", "Toggle path", []string{"ctrl+p"}},
	{"app.session.toggleSort", "Sessions", "Toggle sort", []string{"ctrl+s"}},
	{"app.session.toggleNamedFilter", "Sessions", "Named only", []string{"ctrl+n"}},
	{"app.session.rename", "Sessions", "Rename session", []string{"ctrl+r"}},
	{"app.session.delete", "Sessions", "Delete session", []string{"ctrl+d"}},
	{"app.session.deleteNoninvasive", "Sessions", "Delete session (empty query)", []string{"ctrl+backspace"}},

	{"app.model.select", "Models", "Model picker", []string{"ctrl+l"}},
	{"app.model.cycleForward", "Models", "Next model", []string{"ctrl+p"}},
	{"app.model.cycleBackward", "Models", "Previous model", []string{"shift+ctrl+p"}},
	{"app.thinking.cycle", "Models", "Cycle thinking", []string{"shift+tab"}},
	{"app.thinking.toggle", "Models", "Hide thinking", []string{"ctrl+t"}},

	{"app.tools.expand", "Queue", "Expand tools", []string{"ctrl+o"}},
	{"app.message.copy", "Queue", "Copy last reply", []string{"ctrl+x"}},
	{"app.message.followUp", "Queue", "Queue follow-up", []string{"alt+enter"}},
	{"app.message.dequeue", "Queue", "Restore queued", []string{"alt+up"}},

	{"app.tree.foldOrUp", "Tree", "Fold / previous", []string{"ctrl+left", "alt+left"}},
	{"app.tree.unfoldOrDown", "Tree", "Unfold / next", []string{"ctrl+right", "alt+right"}},
	{"app.tree.editLabel", "Tree", "Edit label", []string{"shift+l"}},
	{"app.tree.toggleLabelTimestamp", "Tree", "Label time", []string{"shift+t"}},
	{"app.tree.filter.default", "Tree", "Filter default", []string{"ctrl+d"}},
	{"app.tree.filter.noTools", "Tree", "Hide tools", []string{"ctrl+t"}},
	{"app.tree.filter.userOnly", "Tree", "User only", []string{"ctrl+u"}},
	{"app.tree.filter.labeledOnly", "Tree", "Labeled only", []string{"ctrl+l"}},
	{"app.tree.filter.all", "Tree", "Show all", []string{"ctrl+a"}},
	{"app.tree.filter.cycleForward", "Tree", "Next filter", []string{"ctrl+o"}},
	{"app.tree.filter.cycleBackward", "Tree", "Previous filter", []string{"shift+ctrl+o"}},

	{"app.models.save", "Model list", "Save models", []string{"ctrl+s"}},
	{"app.models.enableAll", "Model list", "Enable all models", []string{"ctrl+a"}},
	{"app.models.clearAll", "Model list", "Clear models", []string{"ctrl+x"}},
	{"app.models.toggleProvider", "Model list", "Toggle provider", []string{"ctrl+p"}},
	{"app.models.reorderUp", "Model list", "Move model up", []string{"alt+up"}},
	{"app.models.reorderDown", "Model list", "Move model down", []string{"alt+down"}},
}

func known(id string) bool {
	for _, a := range Catalog {
		if a.ID == id {
			return true
		}
	}
	return false
}
