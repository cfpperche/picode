package browser

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// SessionDrive is the authority an identified principal has on the split
// bound to its own session (ADR-0172). The stored grant is not consulted:
// the open split is the consent, and closing it ends the binding. Tier act
// reaches navigate, evaluate and the input methods the shell catalog already
// allows. Domains "*" is any http(s) host; file: and javascript: stay
// refused by AllowsOrigin.
func SessionDrive() Policy {
	return Policy{Tier: "act", Domains: []string{"*"}}
}

// IsSessionDrive reports whether this verb runs in the principal's split
// rather than reading history or naming a raw CDP method.
func IsSessionDrive(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "snapshot", "screenshot", "events", "evaluate", "navigate", "open", "click", "type", "press":
		return true
	default:
		return false
	}
}

// Prepare turns a verb's parameters into the one call the shell runs.
// open is not CDP: the page opens the split. click, type and press are one
// Runtime.evaluate so a selector or a string is never concatenated into script.
func Prepare(name string, params map[string]any) (method string, raw json.RawMessage, err error) {
	verb, ok := VerbFor(name)
	if !ok {
		return "", nil, fmt.Errorf("unknown verb %q", name)
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "open":
		body := map[string]any{}
		if u := strings.TrimSpace(stringParam(params, "url")); u != "" {
			if !AllowsOrigin([]string{"*"}, u) {
				return "", nil, fmt.Errorf("open accepts only http and https URLs")
			}
			body["url"] = u
		}
		raw, err = json.Marshal(body)
		return "shell.open", raw, err
	case "click":
		expr, err := clickExpression(params)
		if err != nil {
			return "", nil, err
		}
		return "Runtime.evaluate", evalParams(expr), nil
	case "type":
		expr, err := typeExpression(params)
		if err != nil {
			return "", nil, err
		}
		return "Runtime.evaluate", evalParams(expr), nil
	case "press":
		expr, err := pressExpression(params)
		if err != nil {
			return "", nil, err
		}
		return "Runtime.evaluate", evalParams(expr), nil
	default:
		if params == nil {
			return verb.Method, json.RawMessage("{}"), nil
		}
		raw, err = json.Marshal(params)
		return verb.Method, raw, err
	}
}

func evalParams(expression string) json.RawMessage {
	raw, err := json.Marshal(map[string]any{"expression": expression, "returnByValue": true})
	if err != nil {
		return json.RawMessage("{}")
	}
	return raw
}

func stringParam(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	s, _ := params[key].(string)
	return s
}

func clickExpression(params map[string]any) (string, error) {
	if sel := strings.TrimSpace(stringParam(params, "selector")); sel != "" {
		quoted, _ := json.Marshal(sel)
		return `(() => { const sel = ` + string(quoted) + `; const el = document.querySelector(sel); if (!el) return {ok:false, error:"no element matches " + sel}; el.scrollIntoView({block:"center", inline:"center"}); el.click(); return {ok:true}; })()`, nil
	}
	x, xok := finite(params, "x")
	y, yok := finite(params, "y")
	if !xok || !yok {
		return "", fmt.Errorf("click needs a selector or x and y")
	}
	return `(() => { const el = document.elementFromPoint(` + strconv.FormatFloat(x, 'f', -1, 64) + `, ` + strconv.FormatFloat(y, 'f', -1, 64) + `); if (!el) return {ok:false, error:"no element at that point"}; el.click(); return {ok:true}; })()`, nil
}

func typeExpression(params map[string]any) (string, error) {
	text := stringParam(params, "text")
	if text == "" {
		return "", fmt.Errorf("type needs text")
	}
	quoted, _ := json.Marshal(text)
	sel := "null"
	if s := strings.TrimSpace(stringParam(params, "selector")); s != "" {
		b, _ := json.Marshal(s)
		sel = string(b)
	}
	return `(() => { const text = ` + string(quoted) + `; const sel = ` + sel + `; const el = sel ? document.querySelector(sel) : (document.activeElement || document.body); if (!el) return {ok:false, error: sel ? "no element matches " + sel : "nothing to type into"}; el.focus(); const field = !el.isContentEditable && (el.tagName === "INPUT" || el.tagName === "TEXTAREA"); if (field) { const proto = el instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype; const desc = Object.getOwnPropertyDescriptor(proto, "value"); const next = String(el.value || "") + text; if (desc && desc.set) desc.set.call(el, next); else el.value = next; el.dispatchEvent(new InputEvent("input", {bubbles:true, data:text, inputType:"insertText"})); el.dispatchEvent(new Event("change", {bubbles:true})); } else { document.execCommand("insertText", false, text); } return {ok:true}; })()`, nil
}

func pressExpression(params map[string]any) (string, error) {
	key, ok := pressKey(stringParam(params, "key"))
	if !ok {
		return "", fmt.Errorf("press does not know key %q", strings.TrimSpace(stringParam(params, "key")))
	}
	quoted, _ := json.Marshal(key)
	return `(() => { const key = ` + string(quoted) + `; const el = document.activeElement || document.body; if (!el) return {ok:false, error:"nothing is focused"}; const opts = {key, bubbles:true, cancelable:true}; el.dispatchEvent(new KeyboardEvent("keydown", opts)); el.dispatchEvent(new KeyboardEvent("keyup", opts)); if (key === "Enter" && el.form && typeof el.form.requestSubmit === "function") el.form.requestSubmit(); return {ok:true}; })()`, nil
}

func pressKey(key string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "enter", "return":
		return "Enter", true
	case "tab":
		return "Tab", true
	case "escape", "esc":
		return "Escape", true
	case "backspace":
		return "Backspace", true
	case "arrowup", "up":
		return "ArrowUp", true
	case "arrowdown", "down":
		return "ArrowDown", true
	case "arrowleft", "left":
		return "ArrowLeft", true
	case "arrowright", "right":
		return "ArrowRight", true
	case "space", " ":
		return " ", true
	}
	runes := []rune(strings.TrimSpace(key))
	if len(runes) == 1 {
		return string(runes), true
	}
	return "", false
}

func finite(params map[string]any, key string) (float64, bool) {
	if params == nil {
		return 0, false
	}
	n, ok := params[key].(float64)
	if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, false
	}
	return n, true
}
