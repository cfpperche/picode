package mcptool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/computer"
)

// The computer family (ADR-0148 over MCP): one tool, the daemon's 23
// actions. Description, guidelines and parameters mirror
// packages/pi-computer/extensions/computer.ts; the answers mirror its
// src/logic.ts. A drift between the two is a test failure here.

const computerDescription = "Use the Windows desktop through PiCode's desktop app: see a monitor or a window (screenshot, zoom, snapshot of its accessibility tree), " +
	"list and focus windows, click, drag, scroll, type, press keys, read and write the clipboard, open a program or a file. " +
	"Needs the human's grant for this agent (Settings ▸ Computer); with it you act with their own permissions on their desktop."

var computerGuidelines = []string{
	"A screenshot is one whole monitor (display 1 by default) or one window when you pass `window`; every coordinate you send is a pixel of the LAST image you received — not a screen coordinate. Take a screenshot before the first click and after a window changes.",
	"Clicks, typing, scrolling and keys return a fresh image: read it before the next step. `windows` lists what is open; `focus` brings a window forward before you type into it.",
	"`snapshot` reads a window's accessibility tree with each element's centre in the last image — use it to find small controls and to read text back instead of guessing from pixels.",
	"You act with the human's permissions on their desktop. Before paying, sending a message, deleting or overwriting files, or typing a password, stop and confirm with the human.",
	"A refusal names what is missing (the grant, the desktop app, a stale window id, a window that is no longer in front: `foreground_changed` means the human moved — look again or `focus` the window, then act); say so and do not retry around it.",
}

// actionsWithImage answer with a fresh capture (the pi package's set).
var actionsWithImage = map[string]bool{
	"screenshot": true, "zoom": true, "wait": true, "left_click": true, "right_click": true, "middle_click": true,
	"double_click": true, "triple_click": true, "left_click_drag": true, "scroll": true, "type": true, "key": true, "hold_key": true,
}

func computerSchema() map[string]any {
	pair := func(desc string) map[string]any {
		return map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "minItems": 2, "maxItems": 2, "description": desc}
	}
	return map[string]any{
		"type":     "object",
		"required": []string{"action"},
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string", "enum": computer.Actions(),
				"description": "screenshot | zoom | snapshot | cursor_position | wait | windows | focus | left_click | right_click | middle_click | double_click | triple_click | left_click_drag | mouse_move | left_mouse_down | left_mouse_up | scroll | type | key | hold_key | clipboard_read | clipboard_write | open",
			},
			"coordinate":       pair("[x, y] in pixels of the last image (clicks, mouse_move, scroll, drag end)"),
			"start_coordinate": pair("left_click_drag: where the drag starts"),
			"text":             map[string]any{"type": "string", "description": "type: the text; key/hold_key: the chord (ctrl+s, Return, alt+F4); clicks: modifiers to hold (shift, ctrl+shift); clipboard_write: the text"},
			"scroll_direction": map[string]any{"type": "string", "enum": []string{"up", "down", "left", "right"}},
			"scroll_amount":    map[string]any{"type": "integer", "minimum": 1, "maximum": 50, "description": "scroll: wheel notches (default 3)"},
			"duration":         map[string]any{"type": "number", "minimum": 0, "maximum": 5, "description": "wait/hold_key: seconds, at most 5"},
			"repeat":           map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "description": "key: how many times"},
			"region":           map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "minItems": 4, "maxItems": 4, "description": "zoom: [x0, y0, x1, y1] in the last image"},
			"display":          map[string]any{"type": "integer", "minimum": 1, "description": "screenshot: which monitor (1 = primary)"},
			"window":           map[string]any{"type": "integer", "description": "screenshot/snapshot/focus: a window id from `windows`"},
			"depth":            map[string]any{"type": "integer", "minimum": 0, "maximum": 24, "description": "snapshot: how deep to walk (default 24)"},
			"target":           map[string]any{"type": "string", "description": "open: a program name (notepad.exe), a file path or a URL"},
		},
	}
}

var computerFamily = Family{
	Name:         "computer",
	Instructions: "computer tool:\n- " + strings.Join(computerGuidelines, "\n- "),
	Tools: func(c *Caller) []Tool {
		return []Tool{{
			Name:        "computer",
			Description: computerDescription,
			InputSchema: computerSchema(),
			Call:        func(ctx context.Context, args json.RawMessage) Result { return computerCall(ctx, c, args, time.Now()) },
		}}
	},
}

func computerCall(ctx context.Context, c *Caller, args json.RawMessage, now time.Time) Result {
	var params map[string]any
	if err := json.Unmarshal(args, &params); err != nil || params == nil {
		return Fail("computer: arguments must be an object")
	}
	action, _ := params["action"].(string)
	if _, ok := computer.ActionFor(action); !ok {
		return Fail("computer: unknown action " + fmt.Sprintf("%q", action) + " — use one of " + strings.Join(computer.Actions(), ", "))
	}
	rest := make(map[string]any, len(params))
	for k, v := range params {
		if k != "action" {
			rest[k] = v
		}
	}
	answer, err := c.post(ctx, "/api/computer/tool", map[string]any{"action": action, "params": rest}, "action")
	if err != nil {
		return Fail("computer: " + err.Error())
	}
	return RenderComputer(action, answer.Output, c, now)
}

// RenderComputer turns the daemon's output into the pi package's answer.
func RenderComputer(action string, output map[string]any, c *Caller, now time.Time) Result {
	if actionsWithImage[action] {
		image, ok := ImageBlock(output)
		if !ok {
			return Fail("computer: " + action + " ran but no image came back")
		}
		text := CaptureLine(output)
		if c != nil && c.Captures != "" {
			if path, err := SaveCapture(c.Captures, c.Identity.Principal(), action, image.MimeType, image.Data, now); err == nil && path != "" {
				text += " Also saved as " + path + "."
			}
		}
		return Result{Content: []Content{{Type: "text", Text: text}, image}}
	}
	switch action {
	case "snapshot":
		return Text(SummarizeSnapshot(output, maxSnapshotLines))
	case "windows":
		return Text(SummarizeWindows(output))
	}
	return Text(SummarizeAct(action, output))
}

// ImageBlock turns a capture answer ({image, mime}) into the block.
func ImageBlock(output map[string]any) (Content, bool) {
	data := str(output, "image")
	if data == "" {
		return Content{}, false
	}
	mime := str(output, "mime")
	if mime == "" {
		mime = "image/png"
	}
	return Content{Type: "image", Data: data, MimeType: mime}, true
}

// CaptureLine names what the image shows and the space its pixels live in.
func CaptureLine(output map[string]any) string {
	m := obj(output, "meta")
	w := obj(m, "window")
	var what string
	if exe := str(w, "exe"); w != nil && exe != "" {
		what = "window " + exe
		if t := str(w, "title"); t != "" {
			what += ` "` + t + `"`
		}
	} else {
		what = "display " + displayOf(m)
	}
	size := ""
	if wd, ok := numOK(m, "width"); ok && wd != 0 {
		if ht, ok := numOK(m, "height"); ok && ht != 0 {
			scale := "1"
			if s, ok := numOK(m, "scale"); ok {
				scale = num(s)
			}
			size = " (" + num(wd) + "×" + num(ht) + " image, scale " + scale + ")"
		}
	}
	cursor := ""
	if cur := arr(m, "cursor"); len(cur) >= 2 {
		cursor = "; cursor at [" + num(cur[0]) + ", " + num(cur[1]) + "]"
	}
	return "Screenshot of " + what + size + " — coordinates you send are pixels of this image" + cursor + "."
}

func displayOf(m map[string]any) string {
	if d, ok := numOK(m, "display"); ok {
		return num(d)
	}
	return "1"
}

const maxSnapshotLines = 400

// SummarizeSnapshot renders the shell's accessibility lines, capped.
func SummarizeSnapshot(output map[string]any, max int) string {
	var lines []string
	for _, l := range arr(output, "lines") {
		if s, ok := l.(string); ok {
			lines = append(lines, s)
		}
	}
	w := obj(output, "window")
	head := "Accessibility tree"
	if w != nil && (str(w, "exe") != "" || str(w, "title") != "") {
		exe := str(w, "exe")
		if exe == "" {
			exe = "window"
		}
		head = "Accessibility tree of " + exe
		if t := str(w, "title"); t != "" {
			head += ` "` + t + `"`
		}
	}
	if len(lines) == 0 {
		return head + ": nothing is exposed (the window may have no accessibility support)."
	}
	shown := lines
	if len(shown) > max {
		shown = shown[:max]
	}
	dropped := len(lines) - len(shown)
	if d, ok := numOK(output, "dropped"); ok {
		dropped += int(d)
	}
	out := append([]string{head + " (refs, roles, names, centres in the last image):"}, shown...)
	if dropped > 0 {
		out = append(out, "… "+itoa(dropped)+" more node(s) not shown")
	}
	return strings.Join(out, "\n")
}

// SummarizeWindows lists the windows the way the model should name them.
func SummarizeWindows(output map[string]any) string {
	var lines []string
	for _, item := range arr(output, "displays") {
		d, _ := item.(map[string]any)
		primary := ""
		if truthy(d["primary"]) {
			primary = " (primary)"
		}
		lines = append(lines, "display "+num(d["index"])+primary+": "+num(fnum(d["right"])-fnum(d["left"]))+"×"+num(fnum(d["bottom"])-fnum(d["top"]))+" at "+num(d["left"])+","+num(d["top"]))
	}
	for _, item := range arr(output, "windows") {
		w, _ := item.(map[string]any)
		var flags []string
		if truthy(w["foreground"]) {
			flags = append(flags, "front")
		}
		if truthy(w["minimized"]) {
			flags = append(flags, "minimized")
		}
		title := []rune(stringOf(w["title"]))
		if len(title) > 80 {
			title = title[:80]
		}
		line := "window " + num(w["id"]) + " " + stringOf(w["exe"]) + ` "` + string(title) + `" display=` + num(w["display"])
		if len(flags) > 0 {
			line += " [" + strings.Join(flags, ", ") + "]"
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "No windows are open."
	}
	return strings.Join(lines, "\n")
}

func fnum(v any) float64 {
	n, _ := v.(float64)
	return n
}

// SummarizeAct is the one-line answer for actions that return no image.
func SummarizeAct(action string, output map[string]any) string {
	o := output
	if o == nil {
		o = map[string]any{}
	}
	switch action {
	case "cursor_position":
		if cur := arr(o, "coordinate"); len(cur) >= 2 {
			return "Cursor at [" + num(cur[0]) + ", " + num(cur[1]) + "] in the last image."
		}
		screen := "null"
		if s, ok := o["screen"]; ok && s != nil {
			screen = jsonString(s)
		}
		return "Cursor is outside the last image (screen " + screen + ")."
	case "clipboard_read":
		text := str(o, "text")
		dropped := ""
		if d, ok := numOK(o, "dropped"); ok && d > 0 {
			dropped = "\n… " + num(d) + " more character(s) not shown"
		}
		if text == "" {
			return "The clipboard has no text."
		}
		return "Clipboard text:\n" + text + dropped
	case "clipboard_write":
		chars := "0"
		if c, ok := o["chars"]; ok && c != nil {
			chars = num(c)
		}
		return "Clipboard set (" + chars + " characters)."
	case "focus":
		w := obj(o, "window")
		if w == nil {
			return "Focused."
		}
		exe := str(w, "exe")
		if exe == "" {
			exe = "window"
		}
		if t := str(w, "title"); t != "" {
			return "Focused " + exe + ` "` + t + `".`
		}
		return "Focused " + exe + "."
	case "open":
		return "Asked Windows to open " + stringOf(o["target"]) + "."
	case "mouse_move":
		return "Moved the pointer."
	case "left_mouse_down":
		return "Left button held down."
	case "left_mouse_up":
		return "Left button released."
	}
	if ok, _ := o["ok"].(bool); ok {
		return action + ": ok"
	}
	return action + ": done"
}
