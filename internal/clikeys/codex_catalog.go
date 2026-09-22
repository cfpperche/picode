package clikeys

// The Codex key map's catalog (docs/plans/keyboard-pane.md, P3): every key the
// CLI's own config struct accepts, in the context the file puts it in, with the
// description the vendor's generated schema carries and the chords it binds out
// of the box.
//
// Read on 2026-09-21 out of the vendor's artifacts at the tag the installed build
// pins — codex-cli 0.155.1, installed as @openai/codex -> @openai/codex-linux-x64
// 0.155.1, x86_64-unknown-linux-musl:
//
//   - the id list and the labels: codex-rs/core/config.schema.json, the vendor's
//     generated schema for TuiKeymap (149 keys in 12 contexts). The schema,
//     not the runtime inventory, is the source here because the two differ: the
//     inventory omits the three `global.*` fallback slots the struct accepts, and
//     an unknown action in this table makes codex refuse the whole file at start
//     (observed live: `unknown field nosuchaction, expected one of …`), so the
//     pane must expose every key the struct accepts;
//   - the defaults: codex-rs/tui/src/keymap.rs, built_in_defaults(), plus codex-rs/tui/src/keymap/vim_search.rs for
//     the four search actions.
//
// Group is the config context the key lives in — the same name the file's own
// table uses, so the pane's headings and the TOML a user reads agree; the order is the runtime
//
//	inventory's (what `/keymap` exposes), with any key the inventory omits
//	joining its context's group at the end. ID is
//
// `<context>.<action>`: the bare name is not unique (`move_left` exists in the
// editor and in vim_normal). Chords are rendered in the file's spelling
// (`ctrl-t`, `page-down`, `esc`), not in PiCode's: a chord here is a string the
// user writes, and the pane formats it for display from the CLI's own vocabulary
// (ADR-0174).
//
// Defaults nil means the key ships with no binding of its own. Three of those are
// not "unused": `global.submit`, `global.queue` and `global.toggle_shortcuts` are
// the global fallbacks the resolver reads for the composer's slots when those are
// unset (built_in_defaults() zeroes them whenever the composer's own key is set),
// and codex's own keymap editor writes the composer's binding into them. The pane
// shows them unset until a user sets one, which is what the file says. One
// built-in chord is missing on purpose: `vim_normal.enter_insert` also binds
// Insert, and the vendor's key vocabulary has no word for that key, so the row
// lists what a user can write.
var CodexCatalog = []Action{
	// global
	{ID: "global.open_agents", Group: "global", Label: "Open the shared agent-session overview.", Defaults: nil},
	{ID: "global.open_transcript", Group: "global", Label: "Open the transcript overlay.", Defaults: []string{"ctrl-t"}},
	{ID: "global.open_external_editor", Group: "global", Label: "Open the external editor for the current draft.", Defaults: []string{"ctrl-g"}},
	{ID: "global.copy", Group: "global", Label: "Copy the last agent response to the clipboard.", Defaults: []string{"ctrl-o"}},
	{ID: "global.clear_terminal", Group: "global", Label: "Clear the terminal UI.", Defaults: []string{"ctrl-l"}},
	{ID: "global.toggle_vim_mode", Group: "global", Label: "Toggle Vim mode for the composer input.", Defaults: nil},
	{ID: "global.toggle_fast_mode", Group: "global", Label: "Toggle Fast mode.", Defaults: nil},
	{ID: "global.toggle_raw_output", Group: "global", Label: "Toggle raw scrollback mode for copy-friendly transcript selection.", Defaults: []string{"alt-r"}},
	{ID: "global.toggle_side_conversation", Group: "global", Label: "Switch between a side conversation and its parent without closing either.", Defaults: []string{"ctrl-/"}},
	{ID: "global.queue", Group: "global", Label: "Queue the current composer draft while a task is running.", Defaults: nil},
	{ID: "global.submit", Group: "global", Label: "Submit the current composer draft.", Defaults: nil},
	{ID: "global.toggle_shortcuts", Group: "global", Label: "Toggle the composer shortcut overlay.", Defaults: nil},
	// chat
	{ID: "chat.interrupt_turn", Group: "chat", Label: "Interrupt the active turn.", Defaults: []string{"esc"}},
	{ID: "chat.decrease_reasoning_effort", Group: "chat", Label: "Decrease the active reasoning effort.", Defaults: []string{"alt-,", "shift-down"}},
	{ID: "chat.increase_reasoning_effort", Group: "chat", Label: "Increase the active reasoning effort.", Defaults: []string{"alt-.", "shift-up"}},
	{ID: "chat.previous_permission_mode", Group: "chat", Label: "Switch to the previous available permission mode.", Defaults: nil},
	{ID: "chat.next_permission_mode", Group: "chat", Label: "Switch to the next available permission mode.", Defaults: nil},
	{ID: "chat.edit_queued_message", Group: "chat", Label: "Move up through pending async questions, then edit the most recently queued message.", Defaults: []string{"alt-up", "shift-left"}},
	{ID: "chat.prompt_stack_back", Group: "chat", Label: "Move back through pending async questions toward the composer.", Defaults: []string{"alt-down", "shift-right"}},
	{ID: "chat.skip_question", Group: "chat", Label: "Skip the focused question.", Defaults: []string{"ctrl-]"}},
	{ID: "chat.toggle_voice_mute", Group: "chat", Label: "Toggle the microphone in an active voice conversation.", Defaults: []string{"ctrl-x"}},
	// composer
	{ID: "composer.submit", Group: "composer", Label: "Submit the current composer draft.", Defaults: []string{"enter"}},
	{ID: "composer.queue", Group: "composer", Label: "Queue the current composer draft while a task is running.", Defaults: []string{"tab"}},
	{ID: "composer.toggle_shortcuts", Group: "composer", Label: "Toggle the composer shortcut overlay.", Defaults: []string{"?", "shift-?"}},
	{ID: "composer.history_search_previous", Group: "composer", Label: "Open reverse history search or move to the previous match.", Defaults: []string{"ctrl-r"}},
	{ID: "composer.history_search_next", Group: "composer", Label: "Move to the next match in reverse history search.", Defaults: []string{"ctrl-s"}},
	// editor
	{ID: "editor.insert_newline", Group: "editor", Label: "Insert a newline in the editor.", Defaults: []string{"ctrl-j", "ctrl-m", "enter", "shift-enter", "alt-enter"}},
	{ID: "editor.move_left", Group: "editor", Label: "Move cursor left by one grapheme.", Defaults: []string{"left", "ctrl-b"}},
	{ID: "editor.move_right", Group: "editor", Label: "Move cursor right by one grapheme.", Defaults: []string{"right", "ctrl-f"}},
	{ID: "editor.move_up", Group: "editor", Label: "Move cursor up one visual line.", Defaults: []string{"up", "ctrl-p"}},
	{ID: "editor.move_down", Group: "editor", Label: "Move cursor down one visual line.", Defaults: []string{"down", "ctrl-n"}},
	{ID: "editor.move_word_left", Group: "editor", Label: "Move cursor to beginning of previous word.", Defaults: []string{"alt-b", "alt-left", "ctrl-left"}},
	{ID: "editor.move_word_right", Group: "editor", Label: "Move cursor to end of next word.", Defaults: []string{"alt-f", "alt-right", "ctrl-right"}},
	{ID: "editor.move_line_start", Group: "editor", Label: "Move cursor to beginning of line.", Defaults: []string{"home", "ctrl-a"}},
	{ID: "editor.move_line_end", Group: "editor", Label: "Move cursor to end of line.", Defaults: []string{"end", "ctrl-e"}},
	{ID: "editor.delete_backward", Group: "editor", Label: "Delete one grapheme to the left.", Defaults: []string{"backspace", "shift-backspace", "ctrl-h"}},
	{ID: "editor.delete_forward", Group: "editor", Label: "Delete one grapheme to the right.", Defaults: []string{"delete", "shift-delete", "ctrl-d"}},
	{ID: "editor.delete_backward_word", Group: "editor", Label: "Delete the previous word.", Defaults: []string{"alt-backspace", "ctrl-backspace", "ctrl-backspace", "ctrl-w", "ctrl-h"}},
	{ID: "editor.delete_forward_word", Group: "editor", Label: "Delete the next word.", Defaults: []string{"alt-delete", "ctrl-delete", "ctrl-delete", "alt-d"}},
	{ID: "editor.kill_line_start", Group: "editor", Label: "Kill text from cursor to line start.", Defaults: []string{"ctrl-u"}},
	{ID: "editor.kill_whole_line", Group: "editor", Label: "Kill the current line.", Defaults: nil},
	{ID: "editor.kill_line_end", Group: "editor", Label: "Kill text from cursor to line end.", Defaults: []string{"ctrl-k"}},
	{ID: "editor.yank", Group: "editor", Label: "Yank the kill buffer.", Defaults: []string{"ctrl-y"}},
	// vim_normal
	{ID: "vim_normal.enter_insert", Group: "vim_normal", Label: "Enter insert mode at cursor (`i`).", Defaults: []string{"i"}},
	{ID: "vim_normal.append_after_cursor", Group: "vim_normal", Label: "Enter insert mode after cursor (`a`).", Defaults: []string{"a"}},
	{ID: "vim_normal.append_line_end", Group: "vim_normal", Label: "Enter insert mode at end of line (`A`).", Defaults: []string{"shift-a", "A"}},
	{ID: "vim_normal.insert_line_start", Group: "vim_normal", Label: "Enter insert mode at first non-blank of line (`I`).", Defaults: []string{"shift-i", "I"}},
	{ID: "vim_normal.open_line_below", Group: "vim_normal", Label: "Open a new line below and enter insert mode (`o`).", Defaults: []string{"o"}},
	{ID: "vim_normal.open_line_above", Group: "vim_normal", Label: "Open a new line above and enter insert mode (`O`).", Defaults: []string{"shift-o", "O"}},
	{ID: "vim_normal.enter_replace_mode", Group: "vim_normal", Label: "Enter replace mode and overwrite characters under the cursor (`R`).", Defaults: []string{"shift-r", "R"}},
	{ID: "vim_normal.move_left", Group: "vim_normal", Label: "Move cursor left (`h`).", Defaults: []string{"h", "left"}},
	{ID: "vim_normal.move_right", Group: "vim_normal", Label: "Move cursor right (`l`).", Defaults: []string{"l", "right"}},
	{ID: "vim_normal.move_up", Group: "vim_normal", Label: "Move cursor up (`k`), or recall older composer history at history boundaries.", Defaults: []string{"k", "up"}},
	{ID: "vim_normal.move_down", Group: "vim_normal", Label: "Move cursor down (`j`), or recall newer composer history at history boundaries.", Defaults: []string{"j", "down"}},
	{ID: "vim_normal.move_word_forward", Group: "vim_normal", Label: "Move cursor to start of next word (`w`).", Defaults: []string{"w"}},
	{ID: "vim_normal.move_word_backward", Group: "vim_normal", Label: "Move cursor to start of previous word (`b`).", Defaults: []string{"b"}},
	{ID: "vim_normal.move_word_end", Group: "vim_normal", Label: "Move cursor to end of current/next word (`e`).", Defaults: []string{"e"}},
	{ID: "vim_normal.move_line_start", Group: "vim_normal", Label: "Move cursor to start of line (`0`).", Defaults: []string{"0"}},
	{ID: "vim_normal.move_line_end", Group: "vim_normal", Label: "Move cursor to end of line (`$`).", Defaults: []string{"$", "shift-$"}},
	{ID: "vim_normal.find_forward", Group: "vim_normal", Label: "Find the next character on the current line (`f`).", Defaults: []string{"f"}},
	{ID: "vim_normal.find_backward", Group: "vim_normal", Label: "Find the previous character on the current line (`F`).", Defaults: []string{"shift-f"}},
	{ID: "vim_normal.till_forward", Group: "vim_normal", Label: "Stop before the next character on the current line (`t`).", Defaults: []string{"t"}},
	{ID: "vim_normal.till_backward", Group: "vim_normal", Label: "Stop after the previous character on the current line (`T`).", Defaults: []string{"shift-t"}},
	{ID: "vim_normal.jump_top", Group: "vim_normal", Label: "Begin a jump to the first buffer line (`gg`).", Defaults: nil},
	{ID: "vim_normal.jump_bottom", Group: "vim_normal", Label: "Jump to the last buffer line (`G`).", Defaults: []string{"shift-g", "G"}},
	{ID: "vim_normal.delete_char", Group: "vim_normal", Label: "Delete character under cursor (`x`).", Defaults: []string{"x"}},
	{ID: "vim_normal.replace_char", Group: "vim_normal", Label: "Replace the character under the cursor (`r`).", Defaults: []string{"r"}},
	{ID: "vim_normal.repeat_last_change", Group: "vim_normal", Label: "Repeat the last complete edit (`.`).", Defaults: []string{"."}},
	{ID: "vim_normal.substitute_char", Group: "vim_normal", Label: "Delete character under cursor and enter insert mode (`s`).", Defaults: []string{"s"}},
	{ID: "vim_normal.delete_to_line_end", Group: "vim_normal", Label: "Delete from cursor to end of line (`D`).", Defaults: []string{"shift-d", "D"}},
	{ID: "vim_normal.change_to_line_end", Group: "vim_normal", Label: "Change from cursor to end of line and enter insert mode (`C`).", Defaults: []string{"shift-c", "C"}},
	{ID: "vim_normal.yank_line", Group: "vim_normal", Label: "Yank the entire line (`Y`).", Defaults: []string{"shift-y", "Y"}},
	{ID: "vim_normal.paste_after", Group: "vim_normal", Label: "Paste after cursor (`p`).", Defaults: []string{"p"}},
	{ID: "vim_normal.start_delete_operator", Group: "vim_normal", Label: "Begin delete operator; next key selects motion (`d`).", Defaults: []string{"d"}},
	{ID: "vim_normal.start_yank_operator", Group: "vim_normal", Label: "Begin yank operator; next key selects motion (`y`).", Defaults: []string{"y"}},
	{ID: "vim_normal.start_change_operator", Group: "vim_normal", Label: "Begin change operator; next keys select a text object.", Defaults: []string{"c"}},
	{ID: "vim_normal.undo", Group: "vim_normal", Label: "Undo the last complete edit (`u`).", Defaults: []string{"u"}},
	{ID: "vim_normal.redo", Group: "vim_normal", Label: "Redo the last undone edit (`ctrl-r`).", Defaults: []string{"ctrl-r"}},
	{ID: "vim_normal.cancel_operator", Group: "vim_normal", Label: "Cancel a pending operator and return to normal mode.", Defaults: []string{"esc"}},
	// vim_search
	{ID: "vim_search.forward", Group: "vim_search", Label: "Search forward in the active buffer (`/`).", Defaults: []string{"/"}},
	{ID: "vim_search.backward", Group: "vim_search", Label: "Search backward in the active buffer (`?`).", Defaults: []string{"?", "shift-?"}},
	{ID: "vim_search.next", Group: "vim_search", Label: "Repeat the accepted search (`n`).", Defaults: []string{"n"}},
	{ID: "vim_search.previous", Group: "vim_search", Label: "Repeat in the opposite direction (`N`).", Defaults: []string{"N", "shift-n"}},
	// vim_operator
	{ID: "vim_operator.delete_line", Group: "vim_operator", Label: "Repeat delete operator to delete the whole line (`dd`).", Defaults: []string{"d"}},
	{ID: "vim_operator.yank_line", Group: "vim_operator", Label: "Repeat yank operator to yank the whole line (`yy`).", Defaults: []string{"y"}},
	{ID: "vim_operator.motion_left", Group: "vim_operator", Label: "Motion: left (`h`).", Defaults: []string{"h"}},
	{ID: "vim_operator.motion_right", Group: "vim_operator", Label: "Motion: right (`l`).", Defaults: []string{"l"}},
	{ID: "vim_operator.motion_up", Group: "vim_operator", Label: "Motion: up one line (`k`).", Defaults: []string{"k"}},
	{ID: "vim_operator.motion_down", Group: "vim_operator", Label: "Motion: down one line (`j`).", Defaults: []string{"j"}},
	{ID: "vim_operator.motion_word_forward", Group: "vim_operator", Label: "Motion: to start of next word (`w`).", Defaults: []string{"w"}},
	{ID: "vim_operator.motion_word_backward", Group: "vim_operator", Label: "Motion: to start of previous word (`b`).", Defaults: []string{"b"}},
	{ID: "vim_operator.motion_word_end", Group: "vim_operator", Label: "Motion: to end of current/next word (`e`).", Defaults: []string{"e"}},
	{ID: "vim_operator.motion_line_start", Group: "vim_operator", Label: "Motion: to start of line (`0`).", Defaults: []string{"0"}},
	{ID: "vim_operator.motion_line_end", Group: "vim_operator", Label: "Motion: to end of line (`$`).", Defaults: []string{"$", "shift-$"}},
	{ID: "vim_operator.motion_find_forward", Group: "vim_operator", Label: "Motion: find the next character on the current line (`f`).", Defaults: []string{"f"}},
	{ID: "vim_operator.motion_find_backward", Group: "vim_operator", Label: "Motion: find the previous character on the current line (`F`).", Defaults: []string{"shift-f"}},
	{ID: "vim_operator.motion_till_forward", Group: "vim_operator", Label: "Motion: stop before the next character on the current line (`t`).", Defaults: []string{"t"}},
	{ID: "vim_operator.motion_till_backward", Group: "vim_operator", Label: "Motion: stop after the previous character on the current line (`T`).", Defaults: []string{"shift-t"}},
	{ID: "vim_operator.motion_jump_top", Group: "vim_operator", Label: "Motion: begin a jump to the first buffer line (`gg`).", Defaults: nil},
	{ID: "vim_operator.motion_jump_bottom", Group: "vim_operator", Label: "Motion: jump to the last buffer line (`G`).", Defaults: []string{"shift-g", "G"}},
	{ID: "vim_operator.select_inner_text_object", Group: "vim_operator", Label: "Select an inner text object after an operator.", Defaults: []string{"i"}},
	{ID: "vim_operator.select_around_text_object", Group: "vim_operator", Label: "Select an around text object after an operator.", Defaults: []string{"a"}},
	{ID: "vim_operator.cancel", Group: "vim_operator", Label: "Cancel the pending operator and return to normal mode.", Defaults: []string{"esc"}},
	// vim_text_object
	{ID: "vim_text_object.word", Group: "vim_text_object", Label: "Text object: word.", Defaults: []string{"w"}},
	{ID: "vim_text_object.big_word", Group: "vim_text_object", Label: "Text object: whitespace-delimited WORD.", Defaults: []string{"shift-w", "W"}},
	{ID: "vim_text_object.parentheses", Group: "vim_text_object", Label: "Text object: parentheses.", Defaults: []string{"(", "shift-(", ")", "shift-)", "b"}},
	{ID: "vim_text_object.brackets", Group: "vim_text_object", Label: "Text object: brackets.", Defaults: []string{"[", "]"}},
	{ID: "vim_text_object.braces", Group: "vim_text_object", Label: "Text object: braces.", Defaults: []string{"{", "shift-{", "}", "shift-}", "shift-b", "B"}},
	{ID: "vim_text_object.double_quote", Group: "vim_text_object", Label: "Text object: double quotes.", Defaults: []string{"\"", "shift-\""}},
	{ID: "vim_text_object.single_quote", Group: "vim_text_object", Label: "Text object: single quotes.", Defaults: []string{"'"}},
	{ID: "vim_text_object.backtick", Group: "vim_text_object", Label: "Text object: backticks.", Defaults: []string{"`"}},
	{ID: "vim_text_object.cancel", Group: "vim_text_object", Label: "Cancel the pending text-object command.", Defaults: []string{"esc"}},
	// pager
	{ID: "pager.scroll_up", Group: "pager", Label: "Scroll up by one row.", Defaults: []string{"up", "k"}},
	{ID: "pager.scroll_down", Group: "pager", Label: "Scroll down by one row.", Defaults: []string{"down", "j"}},
	{ID: "pager.page_up", Group: "pager", Label: "Scroll up by one page.", Defaults: []string{"page-up", "shift-space", "ctrl-b"}},
	{ID: "pager.page_down", Group: "pager", Label: "Scroll down by one page.", Defaults: []string{"page-down", "space", "ctrl-f"}},
	{ID: "pager.half_page_up", Group: "pager", Label: "Scroll up by half a page.", Defaults: []string{"ctrl-u"}},
	{ID: "pager.half_page_down", Group: "pager", Label: "Scroll down by half a page.", Defaults: []string{"ctrl-d"}},
	{ID: "pager.jump_top", Group: "pager", Label: "Jump to the beginning.", Defaults: []string{"home"}},
	{ID: "pager.jump_bottom", Group: "pager", Label: "Jump to the end.", Defaults: []string{"end"}},
	{ID: "pager.close", Group: "pager", Label: "Close the pager overlay.", Defaults: []string{"q", "ctrl-c"}},
	{ID: "pager.close_transcript", Group: "pager", Label: "Close the transcript overlay via its dedicated toggle key.", Defaults: []string{"ctrl-t"}},
	// list
	{ID: "list.move_up", Group: "list", Label: "Move list selection up.", Defaults: []string{"up", "ctrl-p", "ctrl-k", "k"}},
	{ID: "list.move_down", Group: "list", Label: "Move list selection down.", Defaults: []string{"down", "ctrl-n", "ctrl-j", "j"}},
	{ID: "list.move_left", Group: "list", Label: "Move horizontally left in list pickers that support horizontal actions.", Defaults: []string{"left", "ctrl-h"}},
	{ID: "list.move_right", Group: "list", Label: "Move horizontally right in list pickers that support horizontal actions.", Defaults: []string{"right", "ctrl-l"}},
	{ID: "list.page_up", Group: "list", Label: "Move list selection up by one page.", Defaults: []string{"page-up", "ctrl-b"}},
	{ID: "list.page_down", Group: "list", Label: "Move list selection down by one page.", Defaults: []string{"page-down", "ctrl-f"}},
	{ID: "list.jump_top", Group: "list", Label: "Jump to the first list item.", Defaults: []string{"home"}},
	{ID: "list.jump_bottom", Group: "list", Label: "Jump to the last list item.", Defaults: []string{"end"}},
	{ID: "list.accept", Group: "list", Label: "Accept current selection.", Defaults: []string{"enter"}},
	{ID: "list.cancel", Group: "list", Label: "Cancel and close selection view.", Defaults: []string{"esc"}},
	// agents
	{ID: "agents.resume", Group: "agents", Label: "Open the session resume picker.", Defaults: []string{"ctrl-o"}},
	{ID: "agents.search", Group: "agents", Label: "Search the available agent tasks.", Defaults: []string{"ctrl-f"}},
	{ID: "agents.new_task", Group: "agents", Label: "Start composing a new agent task.", Defaults: []string{"ctrl-n"}},
	{ID: "agents.rename", Group: "agents", Label: "Rename the selected task.", Defaults: []string{"ctrl-r"}},
	{ID: "agents.stop", Group: "agents", Label: "Stop the selected running task.", Defaults: []string{"ctrl-x"}},
	{ID: "agents.archive", Group: "agents", Label: "Archive the selected task and its child agents after confirmation.", Defaults: []string{"ctrl-e"}},
	{ID: "agents.delete", Group: "agents", Label: "Permanently delete the selected task and its child agents after confirmation.", Defaults: []string{"delete"}},
	{ID: "agents.hide", Group: "agents", Label: "Hide the selected task until explicitly resumed or the TUI restarts.", Defaults: []string{"ctrl-w"}},
	{ID: "agents.toggle_grouping", Group: "agents", Label: "Toggle grouping tasks by status or project.", Defaults: []string{"ctrl-s"}},
	// approval
	{ID: "approval.open_fullscreen", Group: "approval", Label: "Open the full-screen approval details view.", Defaults: []string{"ctrl-a", "ctrl-a"}},
	{ID: "approval.open_thread", Group: "approval", Label: "Open the thread that requested approval when shown from another thread.", Defaults: []string{"o"}},
	{ID: "approval.approve", Group: "approval", Label: "Approve the primary option.", Defaults: []string{"y"}},
	{ID: "approval.approve_for_session", Group: "approval", Label: "Approve for session when that option exists.", Defaults: []string{"a"}},
	{ID: "approval.approve_for_prefix", Group: "approval", Label: "Approve with exec-policy prefix when that option exists.", Defaults: []string{"p"}},
	{ID: "approval.deny", Group: "approval", Label: "Deny without providing follow-up guidance.", Defaults: []string{"d"}},
	{ID: "approval.decline", Group: "approval", Label: "Decline and provide corrective guidance.", Defaults: []string{"esc", "n"}},
	{ID: "approval.cancel", Group: "approval", Label: "Cancel an elicitation request.", Defaults: []string{"c"}},
}
