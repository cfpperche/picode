package clikeys

// The OpenCode TUI key map's catalog (docs/plans/keyboard-pane.md, P4): every
// action the installed build's own Definitions table declares, with the
// description that table carries and the chords it binds out of the box.
//
// Read on 2026-09-21 out of packages/tui/src/config/keybind.ts at the tag the
// installed build pins — opencode 1.18.32, installed as opencode-ai +
// opencode-linux-x64 1.18.32 — the table the loader itself parses
// (KeybindOverrides). `leader` is deliberately absent: it is the leader-key
// *configuration*, a single keystroke, not a bindable action. So is
// `input_paste`: the vendor declares it in an object form that carries a
// behaviour flag (preventDefault) PiCode's row model does not, and rewriting it
// as a plain chord would drop that flag.
//
// Group is PiCode's own, from the id namespace; the CLI has no group field.
// Defaults nil means the action ships disabled — the vendor's own table says so
// with the literal "none". Chords keep the file's spelling, including two-stroke
// `<leader>` sequences, and the pane formats them from this vocabulary
// (ADR-0174).
//
// Disabling an action is written the vendor's way: the literal string "none"
// (or false, or an empty array — all in the published schema), never null, which
// the loader treats as invalid and drops the file's whole contribution for.
var OpenCodeCatalog = []Action{
	// App
	{ID: "app_exit", Group: "App", Label: "Exit the application", Defaults: []string{"ctrl+c", "ctrl+d", "<leader>q"}},
	{ID: "app_debug", Group: "App", Label: "Toggle debug panel", Defaults: nil},
	{ID: "app_console", Group: "App", Label: "Toggle console", Defaults: nil},
	{ID: "app_heap_snapshot", Group: "App", Label: "Write heap snapshot", Defaults: nil},
	{ID: "app_toggle_animations", Group: "App", Label: "Toggle animations", Defaults: nil},
	{ID: "app_toggle_file_context", Group: "App", Label: "Toggle file context", Defaults: nil},
	{ID: "app_toggle_diffwrap", Group: "App", Label: "Toggle diff wrapping", Defaults: nil},
	{ID: "app_toggle_paste_summary", Group: "App", Label: "Toggle paste summary", Defaults: nil},
	{ID: "app_toggle_session_directory_filter", Group: "App", Label: "Toggle session directory filtering", Defaults: nil},
	// Commands
	{ID: "command_list", Group: "Commands", Label: "List available commands", Defaults: []string{"ctrl+p"}},
	// Help
	{ID: "help_show", Group: "Help", Label: "Open help dialog", Defaults: nil},
	// Docs
	{ID: "docs_open", Group: "Docs", Label: "Open documentation", Defaults: nil},
	// Diff
	{ID: "diff_open", Group: "Diff", Label: "Open diff viewer", Defaults: nil},
	{ID: "diff_close", Group: "Diff", Label: "Close diff viewer", Defaults: []string{"escape", "q"}},
	{ID: "diff_toggle", Group: "Diff", Label: "Toggle diff viewer item", Defaults: []string{"enter", "space"}},
	{ID: "diff_expand", Group: "Diff", Label: "Expand diff viewer item", Defaults: []string{"right"}},
	{ID: "diff_expand_all", Group: "Diff", Label: "Expand all diff viewer folders", Defaults: []string{"E"}},
	{ID: "diff_collapse", Group: "Diff", Label: "Collapse diff viewer item", Defaults: []string{"left"}},
	{ID: "diff_switch_focus", Group: "Diff", Label: "Switch diff viewer focus", Defaults: []string{"tab"}},
	{ID: "diff_next_hunk", Group: "Diff", Label: "Jump to next diff hunk", Defaults: []string{"]"}},
	{ID: "diff_previous_hunk", Group: "Diff", Label: "Jump to previous diff hunk", Defaults: []string{"["}},
	{ID: "diff_next_file", Group: "Diff", Label: "Jump to next diff file", Defaults: []string{"n"}},
	{ID: "diff_previous_file", Group: "Diff", Label: "Jump to previous diff file", Defaults: []string{"p"}},
	{ID: "diff_toggle_file_tree", Group: "Diff", Label: "Toggle diff viewer file tree", Defaults: []string{"b"}},
	{ID: "diff_single_patch", Group: "Diff", Label: "Toggle single patch view", Defaults: []string{"s"}},
	{ID: "diff_switch_source", Group: "Diff", Label: "Switch diff viewer source", Defaults: []string{"d"}},
	{ID: "diff_toggle_view", Group: "Diff", Label: "Toggle diff viewer split or unified view", Defaults: []string{"v"}},
	{ID: "diff_help", Group: "Diff", Label: "Show more diff viewer shortcuts", Defaults: []string{"?"}},
	// Editor
	{ID: "editor_open", Group: "Editor", Label: "Open external editor", Defaults: []string{"<leader>e"}},
	// Theme
	{ID: "theme_list", Group: "Theme", Label: "List available themes", Defaults: []string{"<leader>t"}},
	{ID: "theme_switch_mode", Group: "Theme", Label: "Switch between light and dark theme mode", Defaults: nil},
	{ID: "theme_mode_lock", Group: "Theme", Label: "Lock or unlock theme mode", Defaults: nil},
	// Sidebar
	{ID: "sidebar_toggle", Group: "Sidebar", Label: "Toggle sidebar", Defaults: []string{"<leader>b"}},
	// Scrollbar
	{ID: "scrollbar_toggle", Group: "Scrollbar", Label: "Toggle session scrollbar", Defaults: nil},
	// Status
	{ID: "status_view", Group: "Status", Label: "View status", Defaults: []string{"<leader>s"}},
	// Debug
	{ID: "debug_view", Group: "Debug", Label: "View debug info", Defaults: nil},
	// Sessions
	{ID: "session_export", Group: "Sessions", Label: "Export session to editor", Defaults: []string{"<leader>x"}},
	{ID: "session_copy", Group: "Sessions", Label: "Copy session transcript", Defaults: nil},
	{ID: "session_move", Group: "Sessions", Label: "Move session", Defaults: nil},
	{ID: "session_new", Group: "Sessions", Label: "Create a new session", Defaults: []string{"<leader>n"}},
	{ID: "session_list", Group: "Sessions", Label: "List all sessions", Defaults: []string{"<leader>l"}},
	{ID: "session_timeline", Group: "Sessions", Label: "Show session timeline", Defaults: []string{"<leader>g"}},
	{ID: "session_fork", Group: "Sessions", Label: "Fork session from message", Defaults: nil},
	{ID: "session_rename", Group: "Sessions", Label: "Rename session", Defaults: []string{"ctrl+r"}},
	{ID: "session_delete", Group: "Sessions", Label: "Delete session", Defaults: []string{"ctrl+d"}},
	{ID: "session_share", Group: "Sessions", Label: "Share current session", Defaults: nil},
	{ID: "session_unshare", Group: "Sessions", Label: "Unshare current session", Defaults: nil},
	{ID: "session_interrupt", Group: "Sessions", Label: "Interrupt current session", Defaults: []string{"escape"}},
	{ID: "session_background", Group: "Sessions", Label: "Background synchronous subagents", Defaults: []string{"ctrl+b"}},
	{ID: "session_compact", Group: "Sessions", Label: "Compact the session", Defaults: []string{"<leader>c"}},
	{ID: "session_toggle_timestamps", Group: "Sessions", Label: "Toggle message timestamps", Defaults: nil},
	{ID: "session_toggle_generic_tool_output", Group: "Sessions", Label: "Toggle generic tool output", Defaults: nil},
	{ID: "session_queued_prompts", Group: "Sessions", Label: "Manage queued prompts", Defaults: []string{"<leader>q"}},
	{ID: "session_child_first", Group: "Sessions", Label: "Go to first child session", Defaults: []string{"<leader>down"}},
	{ID: "session_child_cycle", Group: "Sessions", Label: "Go to next child session", Defaults: []string{"right"}},
	{ID: "session_child_cycle_reverse", Group: "Sessions", Label: "Go to previous child session", Defaults: []string{"left"}},
	{ID: "session_parent", Group: "Sessions", Label: "Go to parent session", Defaults: []string{"up"}},
	{ID: "session_pin_toggle", Group: "Sessions", Label: "Pin or unpin session in the session list", Defaults: []string{"ctrl+f"}},
	{ID: "session_quick_switch_1", Group: "Sessions", Label: "Switch to session in quick slot 1", Defaults: []string{"<leader>1"}},
	{ID: "session_quick_switch_2", Group: "Sessions", Label: "Switch to session in quick slot 2", Defaults: []string{"<leader>2"}},
	{ID: "session_quick_switch_3", Group: "Sessions", Label: "Switch to session in quick slot 3", Defaults: []string{"<leader>3"}},
	{ID: "session_quick_switch_4", Group: "Sessions", Label: "Switch to session in quick slot 4", Defaults: []string{"<leader>4"}},
	{ID: "session_quick_switch_5", Group: "Sessions", Label: "Switch to session in quick slot 5", Defaults: []string{"<leader>5"}},
	{ID: "session_quick_switch_6", Group: "Sessions", Label: "Switch to session in quick slot 6", Defaults: []string{"<leader>6"}},
	{ID: "session_quick_switch_7", Group: "Sessions", Label: "Switch to session in quick slot 7", Defaults: []string{"<leader>7"}},
	{ID: "session_quick_switch_8", Group: "Sessions", Label: "Switch to session in quick slot 8", Defaults: []string{"<leader>8"}},
	{ID: "session_quick_switch_9", Group: "Sessions", Label: "Switch to session in quick slot 9", Defaults: []string{"<leader>9"}},
	// Stash
	{ID: "stash_delete", Group: "Stash", Label: "Delete stash entry", Defaults: []string{"ctrl+d"}},
	// Model
	{ID: "model_provider_list", Group: "Model", Label: "Open provider list from model dialog", Defaults: []string{"ctrl+a"}},
	{ID: "model_favorite_toggle", Group: "Model", Label: "Toggle model favorite status", Defaults: []string{"ctrl+f"}},
	{ID: "model_list", Group: "Model", Label: "List available models", Defaults: []string{"<leader>m"}},
	{ID: "model_cycle_recent", Group: "Model", Label: "Next recently used model", Defaults: []string{"f2"}},
	{ID: "model_cycle_recent_reverse", Group: "Model", Label: "Previous recently used model", Defaults: []string{"shift+f2"}},
	{ID: "model_cycle_favorite", Group: "Model", Label: "Next favorite model", Defaults: nil},
	{ID: "model_cycle_favorite_reverse", Group: "Model", Label: "Previous favorite model", Defaults: nil},
	// Mcp
	{ID: "mcp_list", Group: "Mcp", Label: "List MCP servers", Defaults: nil},
	// Provider
	{ID: "provider_connect", Group: "Provider", Label: "Connect provider", Defaults: nil},
	// Console
	{ID: "console_org_switch", Group: "Console", Label: "Switch console organization", Defaults: nil},
	// Agents
	{ID: "agent_list", Group: "Agents", Label: "List agents", Defaults: []string{"<leader>a"}},
	{ID: "agent_cycle", Group: "Agents", Label: "Next agent", Defaults: []string{"tab"}},
	{ID: "agent_cycle_reverse", Group: "Agents", Label: "Previous agent", Defaults: []string{"shift+tab"}},
	// Variant
	{ID: "variant_cycle", Group: "Variant", Label: "Cycle model variants", Defaults: []string{"ctrl+t"}},
	{ID: "variant_list", Group: "Variant", Label: "List model variants", Defaults: nil},
	// Messages
	{ID: "messages_page_up", Group: "Messages", Label: "Scroll messages up by one page", Defaults: []string{"pageup", "ctrl+alt+b"}},
	{ID: "messages_page_down", Group: "Messages", Label: "Scroll messages down by one page", Defaults: []string{"pagedown", "ctrl+alt+f"}},
	{ID: "messages_line_up", Group: "Messages", Label: "Scroll messages up by one line", Defaults: []string{"ctrl+alt+y"}},
	{ID: "messages_line_down", Group: "Messages", Label: "Scroll messages down by one line", Defaults: []string{"ctrl+alt+e"}},
	{ID: "messages_half_page_up", Group: "Messages", Label: "Scroll messages up by half page", Defaults: []string{"ctrl+alt+u"}},
	{ID: "messages_half_page_down", Group: "Messages", Label: "Scroll messages down by half page", Defaults: []string{"ctrl+alt+d"}},
	{ID: "messages_first", Group: "Messages", Label: "Navigate to first message", Defaults: []string{"ctrl+g", "home"}},
	{ID: "messages_last", Group: "Messages", Label: "Navigate to last message", Defaults: []string{"ctrl+alt+g", "end"}},
	{ID: "messages_next", Group: "Messages", Label: "Navigate to next message", Defaults: nil},
	{ID: "messages_previous", Group: "Messages", Label: "Navigate to previous message", Defaults: nil},
	{ID: "messages_last_user", Group: "Messages", Label: "Navigate to last user message", Defaults: nil},
	{ID: "messages_copy", Group: "Messages", Label: "Copy message", Defaults: []string{"<leader>y"}},
	{ID: "messages_undo", Group: "Messages", Label: "Undo message", Defaults: []string{"<leader>u"}},
	{ID: "messages_redo", Group: "Messages", Label: "Redo message", Defaults: []string{"<leader>r"}},
	{ID: "messages_toggle_conceal", Group: "Messages", Label: "Toggle code block concealment in messages", Defaults: []string{"<leader>h"}},
	// Tool
	{ID: "tool_details", Group: "Tool", Label: "Toggle tool details visibility", Defaults: nil},
	// Display
	{ID: "display_thinking", Group: "Display", Label: "Toggle thinking blocks visibility", Defaults: nil},
	// Prompt
	{ID: "prompt_submit", Group: "Prompt", Label: "Submit prompt", Defaults: nil},
	{ID: "prompt_editor_context_clear", Group: "Prompt", Label: "Clear editor context", Defaults: nil},
	{ID: "prompt_skills", Group: "Prompt", Label: "Open skill selector", Defaults: nil},
	{ID: "prompt_stash", Group: "Prompt", Label: "Stash prompt", Defaults: nil},
	{ID: "prompt_stash_pop", Group: "Prompt", Label: "Pop stashed prompt", Defaults: nil},
	{ID: "prompt_stash_list", Group: "Prompt", Label: "List stashed prompts", Defaults: nil},
	// Workspace
	{ID: "workspace_set", Group: "Workspace", Label: "Set workspace", Defaults: nil},
	// Input
	{ID: "input_clear", Group: "Input", Label: "Clear input field", Defaults: []string{"ctrl+c"}},
	{ID: "input_submit", Group: "Input", Label: "Submit input", Defaults: []string{"return"}},
	{ID: "input_newline", Group: "Input", Label: "Insert newline in input", Defaults: []string{"shift+return", "ctrl+return", "alt+return", "ctrl+j"}},
	{ID: "input_move_left", Group: "Input", Label: "Move cursor left in input", Defaults: []string{"left", "ctrl+b"}},
	{ID: "input_move_right", Group: "Input", Label: "Move cursor right in input", Defaults: []string{"right", "ctrl+f"}},
	{ID: "input_move_up", Group: "Input", Label: "Move cursor up in input", Defaults: []string{"up"}},
	{ID: "input_move_down", Group: "Input", Label: "Move cursor down in input", Defaults: []string{"down"}},
	{ID: "input_select_left", Group: "Input", Label: "Select left in input", Defaults: []string{"shift+left"}},
	{ID: "input_select_right", Group: "Input", Label: "Select right in input", Defaults: []string{"shift+right"}},
	{ID: "input_select_up", Group: "Input", Label: "Select up in input", Defaults: []string{"shift+up"}},
	{ID: "input_select_down", Group: "Input", Label: "Select down in input", Defaults: []string{"shift+down"}},
	{ID: "input_line_home", Group: "Input", Label: "Move to start of line in input", Defaults: []string{"ctrl+a"}},
	{ID: "input_line_end", Group: "Input", Label: "Move to end of line in input", Defaults: []string{"ctrl+e"}},
	{ID: "input_select_line_home", Group: "Input", Label: "Select to start of line in input", Defaults: []string{"ctrl+shift+a"}},
	{ID: "input_select_line_end", Group: "Input", Label: "Select to end of line in input", Defaults: []string{"ctrl+shift+e"}},
	{ID: "input_visual_line_home", Group: "Input", Label: "Move to start of visual line in input", Defaults: []string{"alt+a"}},
	{ID: "input_visual_line_end", Group: "Input", Label: "Move to end of visual line in input", Defaults: []string{"alt+e"}},
	{ID: "input_select_visual_line_home", Group: "Input", Label: "Select to start of visual line in input", Defaults: []string{"alt+shift+a"}},
	{ID: "input_select_visual_line_end", Group: "Input", Label: "Select to end of visual line in input", Defaults: []string{"alt+shift+e"}},
	{ID: "input_buffer_home", Group: "Input", Label: "Move to start of buffer in input", Defaults: []string{"home"}},
	{ID: "input_buffer_end", Group: "Input", Label: "Move to end of buffer in input", Defaults: []string{"end"}},
	{ID: "input_select_buffer_home", Group: "Input", Label: "Select to start of buffer in input", Defaults: []string{"shift+home"}},
	{ID: "input_select_buffer_end", Group: "Input", Label: "Select to end of buffer in input", Defaults: []string{"shift+end"}},
	{ID: "input_delete_line", Group: "Input", Label: "Delete line in input", Defaults: []string{"ctrl+shift+d"}},
	{ID: "input_delete_to_line_end", Group: "Input", Label: "Delete to end of line in input", Defaults: []string{"ctrl+k"}},
	{ID: "input_delete_to_line_start", Group: "Input", Label: "Delete to start of line in input", Defaults: []string{"ctrl+u"}},
	{ID: "input_backspace", Group: "Input", Label: "Backspace in input", Defaults: []string{"backspace", "shift+backspace"}},
	{ID: "input_delete", Group: "Input", Label: "Delete character in input", Defaults: []string{"ctrl+d", "delete", "shift+delete"}},
	{ID: "input_undo", Group: "Input", Label: "Undo in input", Defaults: []string{"ctrl+-", "super+z"}},
	{ID: "input_redo", Group: "Input", Label: "Redo in input", Defaults: []string{"ctrl+.", "super+shift+z"}},
	{ID: "input_word_forward", Group: "Input", Label: "Move word forward in input", Defaults: []string{"alt+f", "alt+right", "ctrl+right"}},
	{ID: "input_word_backward", Group: "Input", Label: "Move word backward in input", Defaults: []string{"alt+b", "alt+left", "ctrl+left"}},
	{ID: "input_select_word_forward", Group: "Input", Label: "Select word forward in input", Defaults: []string{"alt+shift+f", "alt+shift+right"}},
	{ID: "input_select_word_backward", Group: "Input", Label: "Select word backward in input", Defaults: []string{"alt+shift+b", "alt+shift+left"}},
	{ID: "input_delete_word_forward", Group: "Input", Label: "Delete word forward in input", Defaults: []string{"alt+d", "alt+delete", "ctrl+delete"}},
	{ID: "input_delete_word_backward", Group: "Input", Label: "Delete word backward in input", Defaults: []string{"ctrl+w", "ctrl+backspace", "alt+backspace"}},
	{ID: "input_select_all", Group: "Input", Label: "Select all in input", Defaults: []string{"super+a"}},
	// History
	{ID: "history_previous", Group: "History", Label: "Previous history item", Defaults: []string{"up"}},
	{ID: "history_next", Group: "History", Label: "Next history item", Defaults: []string{"down"}},
	// Terminal
	{ID: "terminal_suspend", Group: "Terminal", Label: "Suspend terminal", Defaults: []string{"ctrl+z"}},
	{ID: "terminal_title_toggle", Group: "Terminal", Label: "Toggle terminal title", Defaults: nil},
	// Tips
	{ID: "tips_toggle", Group: "Tips", Label: "Toggle tips on home screen", Defaults: []string{"<leader>h"}},
	// Plugin
	{ID: "plugin_manager", Group: "Plugin", Label: "Open plugin manager dialog", Defaults: nil},
	{ID: "plugin_install", Group: "Plugin", Label: "Install plugin", Defaults: nil},
	// Which
	{ID: "which_key_toggle", Group: "Which", Label: "Toggle which-key panel", Defaults: []string{"ctrl+alt+k"}},
	{ID: "which_key_layout_toggle", Group: "Which", Label: "Switch which-key layout", Defaults: []string{"ctrl+alt+shift+k"}},
	{ID: "which_key_pending_toggle", Group: "Which", Label: "Toggle which-key pending preview", Defaults: []string{"ctrl+alt+shift+p"}},
	{ID: "which_key_group_previous", Group: "Which", Label: "Previous which-key group", Defaults: []string{"ctrl+alt+left", "ctrl+alt+["}},
	{ID: "which_key_group_next", Group: "Which", Label: "Next which-key group", Defaults: []string{"ctrl+alt+right", "ctrl+alt+]"}},
	{ID: "which_key_scroll_up", Group: "Which", Label: "Scroll which-key up", Defaults: []string{"ctrl+alt+up", "ctrl+alt+p"}},
	{ID: "which_key_scroll_down", Group: "Which", Label: "Scroll which-key down", Defaults: []string{"ctrl+alt+down", "ctrl+alt+n"}},
	{ID: "which_key_page_up", Group: "Which", Label: "Page which-key up", Defaults: []string{"ctrl+alt+pageup"}},
	{ID: "which_key_page_down", Group: "Which", Label: "Page which-key down", Defaults: []string{"ctrl+alt+pagedown"}},
	{ID: "which_key_home", Group: "Which", Label: "Jump to first which-key binding", Defaults: []string{"ctrl+alt+home"}},
	{ID: "which_key_end", Group: "Which", Label: "Jump to last which-key binding", Defaults: []string{"ctrl+alt+end"}},
}
