package mcptool

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/browser"
)

// The browser family (ADR-0132/0134 over MCP): one tool, the daemon's
// verbs. Text mirrors packages/pi-browser/extensions/browser.ts and the
// answers mirror its src/logic.ts.

const browserDescription = "Read the web page the human has open in PiCode's work browser (the desktop app). " +
	"Verbs: snapshot (the page as an accessibility tree — roles and names), screenshot (the page as an image you see), " +
	"events (what the tab recorded: navigation, console, network), cdp (one Chrome DevTools Protocol method by name; needs Developer mode), history (where the human has been; off until they allow it in Settings). " +
	"Read-only by default: it cannot click, type or navigate."

var browserGuidelines = []string{
	"Use snapshot to read the page's structure, screenshot when the visual matters, events for what happened since the last poll.",
	"This reads the tab the human has on screen; if they are looking elsewhere, say which page you read.",
	"It cannot act on the page: clicking and navigation need a per-agent grant (Settings ▸ Browser).",
	"cdp names one protocol method and needs two things the human controls: Developer mode on, and the Full tier for this agent. A refusal says which is missing — do not retry around it.",
}

// browserVerbs is the daemon's catalog, sorted, for the schema enum.
func browserVerbs() []string {
	verbs := browser.Verbs()
	out := make([]string, 0, len(verbs))
	for name := range verbs {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func browserSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"verb"},
		"properties": map[string]any{
			"verb":       map[string]any{"type": "string", "enum": browserVerbs(), "description": "snapshot | screenshot | events | evaluate | navigate | cdp (evaluate/navigate need a grant; cdp needs Developer mode and the Full tier)"},
			"since":      map[string]any{"type": "number", "description": "events only: the last sequence number you saw"},
			"expression": map[string]any{"type": "string", "description": "evaluate only: the JavaScript expression to run"},
			"query":      map[string]any{"type": "string", "description": "history only: a search over url and title"},
			"limit":      map[string]any{"type": "number", "description": "history only: how many visits to return (default 100, max 200)"},
			"url":        map[string]any{"type": "string", "description": "navigate only: the destination; its origin must be in the grant"},
			"method":     map[string]any{"type": "string", "description": "cdp only: the full protocol method name, e.g. Network.getAllCookies"},
			"params":     map[string]any{"type": "string", "description": "cdp only: the method's parameters as a JSON object, e.g. {\"urls\":true}"},
		},
	}
}

var browserFamily = Family{
	Name:         "browser",
	Instructions: "browser tool:\n- " + strings.Join(browserGuidelines, "\n- "),
	Tools: func(c *Caller) []Tool {
		return []Tool{{
			Name:        "browser",
			Description: browserDescription,
			InputSchema: browserSchema(),
			Call:        func(ctx context.Context, args json.RawMessage) Result { return browserCall(ctx, c, args, time.Now()) },
		}}
	},
}

func browserCall(ctx context.Context, c *Caller, args json.RawMessage, now time.Time) Result {
	var p struct {
		Verb       string   `json:"verb"`
		Since      *float64 `json:"since"`
		Expression string   `json:"expression"`
		Query      string   `json:"query"`
		Limit      *float64 `json:"limit"`
		URL        string   `json:"url"`
		Method     string   `json:"method"`
		Params     string   `json:"params"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return Fail("browser: arguments must be an object")
	}
	if _, ok := browser.VerbFor(p.Verb); !ok {
		return Fail("browser: unknown verb " + jsonString(p.Verb) + " — use " + strings.Join(browserVerbs(), ", "))
	}
	// cdp's parameters travel as the verb's params (ADR-0144); the daemon
	// decides whether this caller may have them at all.
	params := map[string]any{}
	if p.Since != nil {
		params["since"] = *p.Since
	}
	if p.Expression != "" {
		params["expression"] = p.Expression
	}
	if p.URL != "" {
		params["url"] = p.URL
	}
	if p.Method != "" {
		params["method"] = p.Method
	}
	if p.Query != "" {
		params["query"] = p.Query
	}
	if p.Limit != nil {
		params["limit"] = *p.Limit
	}
	if p.Params != "" {
		var cdp any
		if err := json.Unmarshal([]byte(p.Params), &cdp); err != nil {
			return Fail("browser: params must be a JSON object")
		}
		params["params"] = cdp
	}
	answer, err := c.post(ctx, "/api/browser/tool", map[string]any{"verb": p.Verb, "params": params}, "verb")
	if err != nil {
		return Fail("browser: " + err.Error())
	}
	return RenderBrowser(answer.Head, answer.Output, p.URL, c, now)
}

// RenderBrowser turns the daemon's output into the pi package's answer.
func RenderBrowser(verb string, output map[string]any, url string, c *Caller, now time.Time) Result {
	switch verb {
	case "screenshot":
		data := str(output, "data")
		if data == "" {
			return Fail("browser: the page could not be captured")
		}
		text := "Screenshot of the page on screen (attached)."
		if c != nil && c.Captures != "" {
			if path, err := SaveCapture(c.Captures, c.Identity.Principal(), "browser", "image/png", data, now); err == nil && path != "" {
				text += " Also saved as " + path + "."
			}
		}
		return Result{Content: []Content{{Type: "text", Text: text}, {Type: "image", Data: data, MimeType: "image/png"}}}
	case "history":
		return Text(SummarizeHistory(output, 60))
	case "events":
		lines := SummarizeEvents(output, 40)
		if len(lines) == 0 {
			return Text("The tab recorded nothing.")
		}
		return Text(strings.Join(lines, "\n"))
	case "evaluate":
		return Text(SummarizeEvaluate(output))
	case "navigate":
		return Text(SummarizeNavigate(output, url))
	case "cdp":
		return Text(SummarizeCdp(output, 4000))
	}
	lines, dropped := SummarizeAx(output, maxAxLines)
	if len(lines) == 0 {
		return Text("The page has no accessible content (it may still be loading).")
	}
	text := strings.Join(lines, "\n")
	if dropped > 0 {
		text += "\n… " + itoa(dropped) + " more node(s) not shown"
	}
	return Text(text)
}

const maxAxLines = 200

var axNamelessRoles = map[string]bool{"image": true, "textbox": true, "heading": true, "paragraph": true, "list": true, "table": true}

// SummarizeAx renders an accessibility tree as `role "name"` lines.
func SummarizeAx(output map[string]any, max int) (lines []string, dropped int) {
	for _, item := range arr(output, "nodes") {
		node, _ := item.(map[string]any)
		role := str(obj(node, "role"), "value")
		if role == "" || role == "none" || role == "generic" || role == "InlineTextBox" {
			continue
		}
		name := collapseSpace(stringOf(obj(node, "name")["value"]))
		if name == "" && !axNamelessRoles[role] {
			continue
		}
		if len(lines) >= max {
			dropped++
			continue
		}
		if name != "" {
			lines = append(lines, role+` "`+name+`"`)
		} else {
			lines = append(lines, role)
		}
	}
	return lines, dropped
}

// SummarizeEvents renders the tab's recorded ring for the events verb.
func SummarizeEvents(output map[string]any, max int) []string {
	events := arr(output, "events")
	if events == nil {
		return nil
	}
	if len(events) > max {
		events = events[len(events)-max:]
	}
	var lines []string
	for _, item := range events {
		ev, _ := item.(map[string]any)
		name, ok := ev["event"].(string)
		if !ok {
			continue
		}
		params := jsonString(ev["params"])
		if ev["params"] == nil {
			params = "{}"
		}
		if len([]rune(params)) > 160 {
			params = truncate(params, 160)
		}
		lines = append(lines, name+" "+params)
	}
	if last, ok := numOK(output, "last"); ok && len(lines) == 0 {
		lines = append(lines, "nothing since the cursor (last #"+num(last)+")")
	}
	return lines
}

// SummarizeEvaluate renders Runtime.evaluate's answer: the value, or the
// page's own exception, which is an answer about the page and not a failure.
func SummarizeEvaluate(output map[string]any) string {
	if exception := obj(output, "exceptionDetails"); exception != nil {
		detail, ok := obj(exception, "exception")["description"].(string)
		if !ok {
			detail, ok = exception["text"].(string)
		}
		if ok {
			return "the page threw: " + strings.SplitN(detail, "\n", 2)[0]
		}
		return "the page threw: an exception"
	}
	result := obj(output, "result")
	if result == nil {
		return "evaluate returned nothing"
	}
	if str(result, "type") == "undefined" {
		return "undefined"
	}
	if v, ok := result["value"]; ok {
		if s, isStr := v.(string); isStr {
			return s
		}
		return jsonString(v)
	}
	if d, ok := result["description"].(string); ok {
		return d
	}
	return jsonString(result)
}

// SummarizeNavigate renders Page.navigate's answer.
func SummarizeNavigate(output map[string]any, url string) string {
	if e := str(output, "errorText"); e != "" {
		return "navigate failed: " + e
	}
	if url != "" {
		return "navigated to " + url
	}
	return "navigated"
}

// SummarizeCdp renders a raw method's answer as JSON, capped.
func SummarizeCdp(output any, max int) string {
	text := jsonString(output)
	r := []rune(text)
	if len(r) > max {
		return string(r[:max]) + "… (" + itoa(len(r)-max) + " more chars)"
	}
	return text
}

// SummarizeHistory renders the visits the daemon returned, newest first.
func SummarizeHistory(output map[string]any, max int) string {
	visits := arr(output, "visits")
	if len(visits) == 0 {
		return "No visits match."
	}
	shown := visits
	if len(shown) > max {
		shown = shown[:max]
	}
	lines := make([]string, 0, len(shown))
	for _, item := range shown {
		v, _ := item.(map[string]any)
		when := ""
		if at := str(v, "visitedAt"); at != "" {
			if t, err := time.Parse(time.RFC3339Nano, at); err == nil {
				when = t.UTC().Format("2006-01-02 15:04")
			}
		}
		title := ""
		if t := str(v, "title"); t != "" && t != str(v, "url") {
			title = t + " — "
		}
		lines = append(lines, strings.TrimSpace(when+"  "+title+str(v, "url")))
	}
	extra := ""
	if len(visits) > len(shown) {
		extra = ", " + itoa(len(visits)-len(shown)) + " more"
	}
	plural := "s"
	if len(visits) == 1 {
		plural = ""
	}
	return strings.Join(lines, "\n") + "\n(" + itoa(len(visits)) + " visit" + plural + extra + ")"
}
