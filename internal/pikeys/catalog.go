package pikeys

// Action is one Pi keybinding id (docs/keybindings.md).
type Action struct {
	ID       string   `json:"id"`
	Group    string   `json:"group"`
	Label    string   `json:"label"`
	Defaults []string `json:"defaults"`
	// Alt is the whole default on a platform that differs from the one above
	// — not an addition to it. Keyed by platform ("windows", "wsl"); an empty
	// list means pi binds nothing there. Each entry was read out of pi's own
	// docs/keybindings.md on 2026-09-21, which documents the alternate beside
	// the default ("`ctrl+-` (`alt+z` on WSL)"), so the pane stops printing a
	// key pi does not use on the machine it is running on.
	Alt map[string][]string `json:"alt,omitempty"`
}

// alt is a shorthand for the two-platform rows below.
func alt(windows, wsl []string) map[string][]string {
	m := map[string][]string{}
	if windows != nil {
		m["windows"] = windows
	}
	if wsl != nil {
		m["wsl"] = wsl
	}
	return m
}

// Catalog is the Pi default map. User JSON overrides per id.
var Catalog = []Action{
	{"tui.editor.cursorUp", "Editor", "Move cursor up", []string{"up"}, nil},
	{"tui.editor.cursorDown", "Editor", "Move cursor down", []string{"down"}, nil},
	{"tui.editor.historyPrevious", "Editor", "Previous prompt", nil, nil},
	{"tui.editor.historyNext", "Editor", "Next prompt", nil, nil},
	{"tui.editor.cursorLeft", "Editor", "Move left", []string{"left", "ctrl+b"}, nil},
	{"tui.editor.cursorRight", "Editor", "Move right", []string{"right", "ctrl+f"}, nil},
	{"tui.editor.cursorWordLeft", "Editor", "Word left", []string{"alt+left", "ctrl+left", "alt+b"}, nil},
	{"tui.editor.cursorWordRight", "Editor", "Word right", []string{"alt+right", "ctrl+right", "alt+f"}, nil},
	{"tui.editor.cursorLineStart", "Editor", "Line start", []string{"home", "ctrl+home", "ctrl+a"}, nil},
	{"tui.editor.cursorLineEnd", "Editor", "Line end", []string{"end", "ctrl+end", "ctrl+e"}, nil},
	{"tui.editor.jumpForward", "Editor", "Jump to character", []string{"ctrl+]"}, nil},
	{"tui.editor.jumpBackward", "Editor", "Jump back to character", []string{"ctrl+alt+]"}, nil},
	{"tui.editor.pageUp", "Editor", "Page up", []string{"pageUp", "ctrl+pageUp"}, nil},
	{"tui.editor.pageDown", "Editor", "Page down", []string{"pageDown", "ctrl+pageDown"}, nil},

	{"tui.editor.deleteCharBackward", "Delete", "Delete character", []string{"backspace"}, nil},
	{"tui.editor.deleteCharForward", "Delete", "Delete next character", []string{"delete", "ctrl+d"}, nil},
	{"tui.editor.deleteWordBackward", "Delete", "Delete word", []string{"ctrl+w", "alt+backspace"}, nil},
	{"tui.editor.deleteWordForward", "Delete", "Delete next word", []string{"alt+d", "alt+delete"}, nil},
	{"tui.editor.deleteToLineStart", "Delete", "Delete to line start", []string{"ctrl+u"}, nil},
	{"tui.editor.deleteToLineEnd", "Delete", "Delete to line end", []string{"ctrl+k"}, nil},

	{"tui.input.newLine", "Input", "New line", []string{"shift+enter", "ctrl+j"}, nil},
	{"tui.input.submit", "Input", "Send", []string{"enter"}, nil},
	{"tui.input.tab", "Input", "Tab / complete", []string{"tab"}, nil},

	{"tui.editor.yank", "Clipboard", "Yank", []string{"ctrl+y"}, nil},
	{"tui.editor.yankPop", "Clipboard", "Yank pop", []string{"alt+y"}, nil},
	{"tui.editor.undo", "Clipboard", "Undo", []string{"ctrl+-"}, alt([]string{"ctrl+z"}, []string{"alt+z"})},
	{"tui.input.copy", "Clipboard", "Copy selection", []string{"ctrl+c"}, nil},

	{"tui.select.up", "Lists", "List up", []string{"up"}, nil},
	{"tui.select.down", "Lists", "List down", []string{"down"}, nil},
	{"tui.select.pageUp", "Lists", "List page up", []string{"pageUp"}, nil},
	{"tui.select.pageDown", "Lists", "List page down", []string{"pageDown"}, nil},
	{"tui.select.confirm", "Lists", "Confirm", []string{"enter"}, nil},
	{"tui.select.cancel", "Lists", "Cancel list", []string{"escape", "ctrl+c"}, nil},

	{"tui.altScreen.pageUp", "Transcript", "Transcript page up", []string{"pageUp"}, nil},
	{"tui.altScreen.pageDown", "Transcript", "Transcript page down", []string{"pageDown"}, nil},
	{"tui.altScreen.halfPageUp", "Transcript", "Transcript half page up", nil, nil},
	{"tui.altScreen.halfPageDown", "Transcript", "Transcript half page down", nil, nil},
	{"tui.altScreen.lineUp", "Transcript", "Transcript line up", nil, nil},
	{"tui.altScreen.lineDown", "Transcript", "Transcript line down", nil, nil},
	{"tui.altScreen.previousPrompt", "Transcript", "Previous message", []string{"ctrl+shift+up", "ctrl+up"}, alt([]string{"ctrl+up"}, []string{"ctrl+up"})},
	{"tui.altScreen.nextPrompt", "Transcript", "Next message", []string{"ctrl+shift+down", "ctrl+down"}, alt([]string{"ctrl+down"}, []string{"ctrl+down"})},
	{"tui.altScreen.search", "Transcript", "Search transcript", []string{"ctrl+shift+f"}, alt([]string{"ctrl+f"}, []string{"ctrl+f"})},
	{"tui.altScreen.searchNext", "Transcript", "Next match", []string{"enter", "ctrl+g"}, nil},
	{"tui.altScreen.searchPrevious", "Transcript", "Previous match", []string{"shift+enter", "ctrl+shift+g"}, nil},
	{"tui.altScreen.searchClose", "Transcript", "Close search", []string{"escape"}, nil},
	{"tui.altScreen.top", "Transcript", "Jump to top", []string{"home"}, nil},
	{"tui.altScreen.bottom", "Transcript", "Jump to bottom", []string{"end"}, nil},

	{"app.interrupt", "App", "Cancel", []string{"escape"}, nil},
	{"app.clear", "App", "Clear / exit", []string{"ctrl+c"}, nil},
	{"app.exit", "App", "Exit when empty", []string{"ctrl+d"}, nil},
	// Windows terminals have no Unix job control, so pi binds nothing there;
	// WSL keeps the base binding (pi's own note, docs/keybindings.md:210).
	{"app.suspend", "App", "Suspend", []string{"ctrl+z"}, alt([]string{}, nil)},
	{"app.editor.external", "App", "Open in editor", []string{"ctrl+g"}, nil},
	{"app.clipboard.pasteImage", "App", "Paste", []string{"ctrl+v"}, alt([]string{"alt+v"}, []string{"alt+v"})},

	{"app.session.new", "Sessions", "New session", nil, nil},
	{"app.session.tree", "Sessions", "Session tree", nil, nil},
	{"app.session.fork", "Sessions", "Fork session", nil, nil},
	{"app.session.resume", "Sessions", "Resume session", nil, nil},
	{"app.session.togglePath", "Sessions", "Toggle path", []string{"ctrl+p"}, nil},
	{"app.session.toggleSort", "Sessions", "Toggle sort", []string{"ctrl+s"}, nil},
	{"app.session.toggleNamedFilter", "Sessions", "Named only", []string{"ctrl+n"}, nil},
	{"app.session.rename", "Sessions", "Rename session", []string{"ctrl+r"}, nil},
	{"app.session.delete", "Sessions", "Delete session", []string{"ctrl+d"}, nil},
	{"app.session.deleteNoninvasive", "Sessions", "Delete session (empty query)", []string{"ctrl+backspace"}, nil},

	{"app.model.select", "Models", "Model picker", []string{"ctrl+l"}, nil},
	{"app.model.cycleForward", "Models", "Next model", []string{"ctrl+p"}, nil},
	{"app.model.cycleBackward", "Models", "Previous model", []string{"shift+ctrl+p"}, alt([]string{"alt+p"}, []string{"alt+p"})},
	{"app.thinking.cycle", "Models", "Cycle thinking", []string{"shift+tab"}, nil},
	{"app.thinking.save", "Models", "Save thinking level", []string{"ctrl+s"}, nil},
	{"app.thinking.toggle", "Models", "Hide thinking", []string{"ctrl+t"}, nil},

	{"app.tools.expand", "Queue", "Expand tools", []string{"ctrl+o"}, nil},
	{"app.message.copy", "Queue", "Copy last reply", []string{"ctrl+x"}, nil},
	{"app.message.followUp", "Queue", "Queue follow-up", []string{"alt+enter"}, alt([]string{"ctrl+q"}, []string{"ctrl+q"})},
	{"app.message.dequeue", "Queue", "Restore queued", []string{"alt+up"}, alt([]string{"alt+q"}, []string{"alt+q"})},

	{"app.tree.foldOrUp", "Tree", "Fold / previous", []string{"ctrl+left", "alt+left"}, nil},
	{"app.tree.unfoldOrDown", "Tree", "Unfold / next", []string{"ctrl+right", "alt+right"}, nil},
	{"app.tree.editLabel", "Tree", "Edit label", []string{"shift+l"}, nil},
	{"app.tree.toggleLabelTimestamp", "Tree", "Label time", []string{"shift+t"}, nil},
	{"app.tree.filter.default", "Tree", "Filter default", []string{"ctrl+d"}, nil},
	{"app.tree.filter.noTools", "Tree", "Hide tools", []string{"ctrl+t"}, nil},
	{"app.tree.filter.userOnly", "Tree", "User only", []string{"ctrl+u"}, nil},
	{"app.tree.filter.labeledOnly", "Tree", "Labeled only", []string{"ctrl+l"}, nil},
	{"app.tree.filter.all", "Tree", "Show all", []string{"ctrl+a"}, nil},
	{"app.tree.filter.cycleForward", "Tree", "Next filter", []string{"ctrl+o"}, nil},
	{"app.tree.filter.cycleBackward", "Tree", "Previous filter", []string{"shift+ctrl+o"}, nil},

	{"app.models.save", "Model list", "Save models", []string{"ctrl+s"}, nil},
	{"app.models.enableAll", "Model list", "Enable all models", []string{"ctrl+a"}, nil},
	{"app.models.clearAll", "Model list", "Clear models", []string{"ctrl+x"}, nil},
	{"app.models.toggleProvider", "Model list", "Toggle provider", []string{"ctrl+p"}, nil},
	{"app.models.reorderUp", "Model list", "Move model up", []string{"alt+up"}, nil},
	{"app.models.reorderDown", "Model list", "Move model down", []string{"alt+down"}, nil},
}

func known(id string) bool {
	for _, a := range Catalog {
		if a.ID == id {
			return true
		}
	}
	return false
}
