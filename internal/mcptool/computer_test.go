package mcptool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/computer"
)

// These are packages/pi-computer/test/logic.test.ts, line for line: the MCP
// wire must print what the pi wire prints (ADR-0154).

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("decode %q: %v", s, err)
	}
	return m
}

func TestComputerCaptureBecomesAnImageBlockPlusItsPixelSpace(t *testing.T) {
	output := decode(t, `{"ok":true,"image":"iVBORw0KGgo=","mime":"image/png","meta":{"display":1,"window":null,"width":1280,"height":800,"scale":2,"seq":3,"cursor":[236,220],"preview":"data:image/jpeg;base64,/9j/"}}`)
	img, ok := ImageBlock(output)
	if !ok || img != (Content{Type: "image", Data: "iVBORw0KGgo=", MimeType: "image/png"}) {
		t.Fatalf("image block = %+v, %v", img, ok)
	}
	if _, ok := ImageBlock(decode(t, `{"ok":true}`)); ok {
		t.Fatal("no image must be no block")
	}
	want := "Screenshot of display 1 (1280×800 image, scale 2) — coordinates you send are pixels of this image; cursor at [236, 220]."
	if got := CaptureLine(output); got != want {
		t.Fatalf("captureLine =\n%s\nwant\n%s", got, want)
	}
	win := decode(t, `{"ok":true,"image":"x","meta":{"display":1,"window":{"exe":"notepad.exe","title":"Untitled - Notepad"},"width":1280,"height":800,"scale":2,"cursor":null}}`)
	if got := CaptureLine(win); !strings.HasPrefix(got, `Screenshot of window notepad.exe "Untitled - Notepad"`) {
		t.Fatalf("window captureLine = %s", got)
	}
}

func TestComputerSnapshotsAndWindowListsRenderAsLines(t *testing.T) {
	snap := decode(t, `{"window":{"exe":"notepad.exe","title":"Untitled - Notepad"},"lines":["e1 Window \"Untitled - Notepad\" @center(640,400) depth=0","e3 Edit \"Text Editor\" @center(640,420) depth=2 value=\"hello\""],"dropped":2}`)
	text := SummarizeSnapshot(snap, maxSnapshotLines)
	if !strings.HasPrefix(text, `Accessibility tree of notepad.exe "Untitled - Notepad"`) || !strings.Contains(text, `e3 Edit "Text Editor"`) || !strings.HasSuffix(text, "… 2 more node(s) not shown") {
		t.Fatalf("snapshot =\n%s", text)
	}
	if got := SummarizeSnapshot(decode(t, `{"lines":[]}`), maxSnapshotLines); !strings.Contains(got, "nothing is exposed") {
		t.Fatalf("empty snapshot = %s", got)
	}
	wins := SummarizeWindows(decode(t, `{"displays":[{"index":1,"primary":true,"left":0,"top":0,"right":2560,"bottom":1600}],"windows":[{"id":132458,"exe":"notepad.exe","title":"Untitled - Notepad","display":1,"foreground":true,"minimized":false}]}`))
	want := "display 1 (primary): 2560×1600 at 0,0\nwindow 132458 notepad.exe \"Untitled - Notepad\" display=1 [front]"
	if wins != want {
		t.Fatalf("windows =\n%s\nwant\n%s", wins, want)
	}
	if got := SummarizeWindows(map[string]any{}); got != "No windows are open." {
		t.Fatalf("no windows = %s", got)
	}
}

func TestComputerActionsWithoutAnImageAnswerInOneLine(t *testing.T) {
	cases := map[string][2]string{
		"cursor_position":  {`{"coordinate":[4,5]}`, "Cursor at [4, 5] in the last image."},
		"clipboard_read":   {`{"text":"hi","dropped":0}`, "Clipboard text:\nhi"},
		"clipboard_read_2": {`{"text":""}`, "The clipboard has no text."},
		"clipboard_write":  {`{"chars":3}`, "Clipboard set (3 characters)."},
		"focus":            {`{"window":{"exe":"notepad.exe"}}`, "Focused notepad.exe."},
		"open":             {`{"target":"notepad.exe"}`, "Asked Windows to open notepad.exe."},
		"mouse_move":       {`{"ok":true}`, "Moved the pointer."},
		"left_mouse_up":    {`{}`, "Left button released."},
		"something_else":   {`{"ok":true}`, "something_else: ok"},
	}
	for action, c := range cases {
		name := strings.TrimSuffix(action, "_2")
		if got := SummarizeAct(name, decode(t, c[0])); got != c[1] {
			t.Errorf("%s: got %q want %q", action, got, c[1])
		}
	}
	if got := SummarizeAct("cursor_position", decode(t, `{"coordinate":null,"screen":[9,9]}`)); !strings.Contains(got, "outside the last image") {
		t.Fatalf("outside = %s", got)
	}
}

func TestComputerSchemaIsTheDaemonsCatalog(t *testing.T) {
	schema := computerSchema()
	props := schema["properties"].(map[string]any)
	enum := props["action"].(map[string]any)["enum"].([]string)
	if len(enum) != 23 || len(computer.Actions()) != 23 {
		t.Fatalf("actions = %d", len(enum))
	}
	for _, a := range []string{"screenshot", "left_click", "open", "clipboard_write"} {
		found := false
		for _, e := range enum {
			found = found || e == a
		}
		if !found {
			t.Errorf("%s missing from the schema", a)
		}
	}
	// Every parameter the pi package declares, no more, no less.
	want := []string{"action", "coordinate", "start_coordinate", "text", "scroll_direction", "scroll_amount", "duration", "repeat", "region", "display", "window", "depth", "target", "mode"}
	if len(props) != len(want) {
		t.Fatalf("properties = %d, want %d", len(props), len(want))
	}
	for _, k := range want {
		if _, ok := props[k]; !ok {
			t.Errorf("property %s missing", k)
		}
	}
	src, err := os.ReadFile(filepath.Join("..", "..", "packages", "pi-computer", "extensions", "computer.ts"))
	if err != nil {
		t.Skip("pi-computer source not beside this package")
	}
	for _, k := range want[1:] {
		if !strings.Contains(string(src), k+": Type.Optional(") {
			t.Errorf("pi-computer no longer declares %s — the MCP schema must follow", k)
		}
	}
	if !strings.Contains(string(src), strings.Split(computerDescription, ": ")[0]) {
		t.Error("description drifted from the pi package")
	}
}

func TestComputerRenderSavesTheCaptureBesideTheBlock(t *testing.T) {
	dir := t.TempDir()
	c := &Caller{Captures: dir, Identity: Identity{Term: "t1"}}
	out := decode(t, `{"ok":true,"image":"iVBORw0KGgo=","mime":"image/png","meta":{"display":1,"width":1,"height":1}}`)
	res := RenderComputer("screenshot", out, c, time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC))
	if res.IsError || len(res.Content) != 2 || res.Content[1].Type != "image" {
		t.Fatalf("result = %+v", res)
	}
	if !strings.Contains(res.Content[0].Text, "Also saved as "+filepath.Join(dir, "term_t1")) {
		t.Fatalf("text = %s", res.Content[0].Text)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "term_t1"))
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), "-screenshot.png") {
		t.Fatalf("entries = %v", entries)
	}
	if res := RenderComputer("left_click", decode(t, `{"ok":true}`), c, time.Now()); !res.IsError || !strings.Contains(res.Content[0].Text, "no image came back") {
		t.Fatalf("no image = %+v", res)
	}
}
