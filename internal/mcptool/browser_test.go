package mcptool

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/browser"
)

// packages/pi-browser/test/logic.test.ts, ported.

func TestBrowserSnapshotRendersRolesAndNamesAndSkipsTheNoise(t *testing.T) {
	out := decode(t, `{"nodes":[
		{"role":{"value":"RootWebArea"},"name":{"value":"Example"}},
		{"role":{"value":"generic"},"name":{"value":""}},
		{"role":{"value":"heading"},"name":{"value":"  Hello   world "}},
		{"role":{"value":"InlineTextBox"},"name":{"value":"Hello world"}},
		{"role":{"value":"link"},"name":{"value":"More"}},
		{"role":{"value":"image"},"name":{"value":""}},
		{"role":{"value":"textbox"},"name":{"value":""}}]}`)
	lines, dropped := SummarizeAx(out, maxAxLines)
	want := []string{`RootWebArea "Example"`, `heading "Hello world"`, `link "More"`, "image", "textbox"}
	if strings.Join(lines, "|") != strings.Join(want, "|") || dropped != 0 {
		t.Fatalf("lines = %v dropped %d", lines, dropped)
	}
	lines, dropped = SummarizeAx(out, 2)
	if len(lines) != 2 || dropped != 3 {
		t.Fatalf("capped = %v dropped %d", lines, dropped)
	}
	if l, _ := SummarizeAx(nil, 10); len(l) != 0 {
		t.Fatal("nil output must be empty")
	}
	if l, _ := SummarizeAx(decode(t, `{"nodes":"no"}`), 10); len(l) != 0 {
		t.Fatal("non-array nodes must be empty")
	}
}

func TestBrowserEventsRenderTheTail(t *testing.T) {
	out := decode(t, `{"last":7,"events":[{"seq":6,"event":"Page.frameNavigated","params":{"url":"https://example.com"}},{"seq":7,"event":"Runtime.consoleAPICalled","params":{"text":"`+strings.Repeat("x", 400)+`"}}]}`)
	lines := SummarizeEvents(out, 40)
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "Page.frameNavigated {") || len([]rune(lines[1])) >= 200 {
		t.Fatalf("lines = %v", lines)
	}
	if got := SummarizeEvents(decode(t, `{"last":7,"events":[]}`), 40); len(got) != 1 || got[0] != "nothing since the cursor (last #7)" {
		t.Fatalf("quiet = %v", got)
	}
	if got := SummarizeEvents(map[string]any{}, 40); len(got) != 0 {
		t.Fatalf("empty = %v", got)
	}
}

func TestBrowserEvaluateAnswersWithTheValueNotATree(t *testing.T) {
	cases := map[string]string{
		`{"result":{"type":"number","value":2}}`:                 "2",
		`{"result":{"type":"string","value":"PI-CODE"}}`:         "PI-CODE",
		`{"result":{"type":"object","value":{"a":1}}}`:           `{"a":1}`,
		`{"result":{"type":"undefined"}}`:                        "undefined",
		`{"result":{"type":"function","description":"() => 1"}}`: "() => 1",
		`{}`: "evaluate returned nothing",
		`{"exceptionDetails":{"text":"Uncaught","exception":{"description":"Error: boom\n    at <anonymous>:1:1"}}}`: "the page threw: Error: boom",
	}
	for in, want := range cases {
		if got := SummarizeEvaluate(decode(t, in)); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestBrowserNavigateAndCdpAndHistory(t *testing.T) {
	if got := SummarizeNavigate(decode(t, `{"frameId":"F1"}`), "https://example.com/"); got != "navigated to https://example.com/" {
		t.Fatal(got)
	}
	if got := SummarizeNavigate(decode(t, `{"errorText":"net::ERR_NAME_NOT_RESOLVED"}`), "https://nope.invalid/"); got != "navigate failed: net::ERR_NAME_NOT_RESOLVED" {
		t.Fatal(got)
	}
	if got := SummarizeNavigate(map[string]any{}, ""); got != "navigated" {
		t.Fatal(got)
	}
	if got := SummarizeCdp(decode(t, `{"cookies":[]}`), 4000); got != `{"cookies":[]}` {
		t.Fatal(got)
	}
	big := SummarizeCdp(decode(t, `{"blob":"`+strings.Repeat("x", 5000)+`"}`), 100)
	if !strings.HasPrefix(big, `{"blob":"xxx`) || !regexp.MustCompile(`\(\d+ more chars\)$`).MatchString(big) {
		t.Fatal(big)
	}
	if got := SummarizeCdp("plain", 4000); got != `"plain"` {
		t.Fatal(got)
	}
	hist := SummarizeHistory(decode(t, `{"visits":[{"url":"https://a.example/","title":"A","visitedAt":"2026-09-18T10:00:00Z"},{"url":"https://b.example/","title":"https://b.example/","visitedAt":"2026-09-18T09:00:00Z"}]}`), 60)
	if hist != "2026-09-18 10:00  A — https://a.example/\n2026-09-18 09:00  https://b.example/\n(2 visits)" {
		t.Fatalf("history =\n%s", hist)
	}
	if got := SummarizeHistory(map[string]any{}, 60); got != "No visits match." {
		t.Fatal(got)
	}
}

func TestBrowserSchemaIsTheDaemonsVerbs(t *testing.T) {
	enum := browserSchema()["properties"].(map[string]any)["verb"].(map[string]any)["enum"].([]string)
	for _, v := range []string{"snapshot", "screenshot", "events", "evaluate", "navigate", "cdp", "history"} {
		if _, ok := browser.VerbFor(v); !ok {
			t.Errorf("daemon lost verb %s", v)
		}
		found := false
		for _, e := range enum {
			found = found || e == v
		}
		if !found {
			t.Errorf("schema misses %s", v)
		}
	}
	if len(enum) != len(browser.Verbs()) {
		t.Fatalf("enum = %v", enum)
	}
}

func TestBrowserRenderScreenshotAndEmptyTree(t *testing.T) {
	res := RenderBrowser("screenshot", decode(t, `{"data":"iVBORw0KGgo="}`), "", nil, time.Now())
	if res.IsError || len(res.Content) != 2 || res.Content[1] != (Content{Type: "image", Data: "iVBORw0KGgo=", MimeType: "image/png"}) {
		t.Fatalf("screenshot = %+v", res)
	}
	if res := RenderBrowser("screenshot", decode(t, `{"data":""}`), "", nil, time.Now()); !res.IsError {
		t.Fatal("an empty capture is an error")
	}
	if res := RenderBrowser("snapshot", decode(t, `{"nodes":[]}`), "", nil, time.Now()); res.Content[0].Text != "The page has no accessible content (it may still be loading)." {
		t.Fatalf("empty tree = %+v", res)
	}
	if res := RenderBrowser("events", decode(t, `{"events":[]}`), "", nil, time.Now()); res.Content[0].Text != "The tab recorded nothing." {
		t.Fatalf("no events = %+v", res)
	}
}
