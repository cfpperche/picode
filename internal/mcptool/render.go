package mcptool

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// The helpers below read the daemon's JSON the way the TypeScript renderers
// in packages/pi-*/src/logic.ts do, so both wires print the same words.

func itoa(n int) string { return strconv.Itoa(n) }

// num prints a JSON number the way JavaScript's template literal does:
// integers without a decimal point, everything else shortest round-trip.
func num(v any) string {
	switch n := v.(type) {
	case float64:
		if n == math.Trunc(n) && math.Abs(n) < 1e15 {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'g', -1, 64)
	case int:
		return strconv.Itoa(n)
	case json.Number:
		return n.String()
	case nil:
		return "undefined"
	}
	return stringOf(v)
}

// stringOf is `String(v)` for the shapes JSON can hold.
func stringOf(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case nil:
		return ""
	case bool:
		if s {
			return "true"
		}
		return "false"
	case float64, int, json.Number:
		return num(s)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func str(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func obj(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	o, _ := m[key].(map[string]any)
	return o
}

func arr(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	a, _ := m[key].([]any)
	return a
}

func numOK(m map[string]any, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	n, ok := m[key].(float64)
	return n, ok
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	}
	return true
}

// jsonString is JSON.stringify for a value already decoded from JSON.
func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return stringOf(v)
	}
	return string(b)
}

// truncate cuts a string at n characters (runes) and appends the ellipsis.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// collapseSpace is `.replace(/\s+/g, " ").trim()`.
func collapseSpace(s string) string { return strings.Join(strings.Fields(s), " ") }
