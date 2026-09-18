package computer

import (
	"sort"
	"strings"
	"time"
)

// Actions is the closed catalog (ADR-0148): Anthropic's seventeen computer
// members plus the six house actions. A name outside it is refused by the
// daemon before anything reaches the shell, and the shell keeps its own copy
// of the list.
var actions = map[string]struct{}{
	"screenshot": {}, "zoom": {}, "snapshot": {}, "cursor_position": {}, "wait": {},
	"windows": {}, "focus": {},
	"left_click": {}, "right_click": {}, "middle_click": {}, "double_click": {}, "triple_click": {},
	"left_click_drag": {}, "mouse_move": {}, "left_mouse_down": {}, "left_mouse_up": {}, "scroll": {},
	"type": {}, "key": {}, "hold_key": {},
	"clipboard_read": {}, "clipboard_write": {}, "open": {},
}

// ActionFor normalises a name and says whether the catalog has it.
func ActionFor(name string) (string, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	_, ok := actions[n]
	return n, ok
}

// Actions lists the catalog, sorted, for refusals and docs.
func Actions() []string {
	out := make([]string, 0, len(actions))
	for a := range actions {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// Timeout bounds how long the daemon waits for the shell on one action. A
// wait or a held key may take up to five seconds by the shell's own cap;
// everything else answers in well under a second.
func Timeout(action string) time.Duration {
	switch action {
	case "wait", "hold_key", "snapshot", "left_click_drag":
		return 30 * time.Second
	}
	return 20 * time.Second
}
