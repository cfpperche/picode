// Command picode-openapi prints the server's OpenAPI 3.1 spec to stdout.
//
// The spec is derived from the same registration list the real binary
// runs (server.Routes → registerAll), so it cannot drift from what
// picode actually serves. CI enforces freshness: scripts/docs-check.mjs
// re-runs this command and byte-compares against the committed
// www/public/api/openapi.json.
//
//	make openapi   # regenerates www/public/api/openapi.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/server"
	"github.com/cfpperche/picode/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "picode-openapi:", err)
		os.Exit(1)
	}
}

func run() error {
	const description = "The HTTP API the PiCode web UI itself speaks — same daemon, " +
		"same pairing-gated session. Generated from the server's route " +
		"registration by cmd/picode-openapi; do not edit the JSON by hand. " +
		"Authenticate with the picode_session cookie issued when you pair a " +
		"browser (Settings → Pairing)."

	spec := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "PiCode HTTP API",
			"version":     version.Version,
			"description": description,
		},
		"servers": []any{
			map[string]any{"url": "http://127.0.0.1:8445", "description": "local daemon"},
		},
		"tags":           tagsList(),
		"paths":          paths(),
		"components":     securitySchemes(),
		"security":       []any{map[string]any{"sessionCookie": []string{}}},
		"x-undocumented": []string{"/ (embedded UI)", "/assets/** (UI bundle)"},
	}

	out, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

// tagFor derives a stable group name from a route pattern.
func tagFor(path string) string {
	p := strings.TrimPrefix(strings.TrimPrefix(path, "/api"), "/ws")
	for _, seg := range strings.Split(p, "/") {
		if seg != "" {
			return seg
		}
	}
	return "misc"
}

func tagsList() []any {
	seen := map[string]bool{}
	for _, r := range server.Routes() {
		seen[tagFor(patternPath(r.Pattern))] = true
	}
	names := make([]string, 0, len(seen))
	for t := range seen {
		names = append(names, t)
	}
	sort.Strings(names)
	out := make([]any, len(names))
	for i, n := range names {
		out[i] = map[string]any{"name": n}
	}
	return out
}

func patternPath(pattern string) string {
	_, path, found := strings.Cut(pattern, " ")
	if !found {
		return pattern
	}
	return path
}

func paths() map[string]any {
	out := map[string]any{}
	for _, r := range server.Routes() {
		path := patternPath(r.Pattern)
		item, ok := out[path].(map[string]any)
		if !ok {
			item = map[string]any{}
			out[path] = item
		}
		op := map[string]any{
			"operationId": operationID(r.Method, path),
			"tags":        []string{tagFor(path)},
			"parameters":  pathParams(path),
			"responses": map[string]any{
				"200": map[string]any{"description": "Success"},
				"401": map[string]any{"description": "Not paired"},
			},
		}
		if strings.HasPrefix(path, "/ws") {
			op["summary"] = "WebSocket upgrade"
			op["description"] = "Upgrades to a WebSocket session; not request/response HTTP."
		}
		if r.Method != "GET" {
			op["requestBody"] = map[string]any{
				"content": map[string]any{
					"application/json": map[string]any{"schema": map[string]any{"type": "object"}},
				},
			}
		}
		item[strings.ToLower(r.Method)] = op
	}
	return out
}

func operationID(method, path string) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(method))
	for _, seg := range strings.Split(path, "/") {
		if seg == "" || strings.HasPrefix(seg, "{") {
			continue
		}
		b.WriteString(strings.ToUpper(seg[:1]) + seg[1:])
	}
	return b.String()
}

func pathParams(path string) []any {
	var out []any
	for _, seg := range strings.Split(path, "/") {
		if !strings.HasPrefix(seg, "{") || !strings.HasSuffix(seg, "}") {
			continue
		}
		out = append(out, map[string]any{
			"name":     strings.Trim(seg, "{}"),
			"in":       "path",
			"required": true,
			"schema":   map[string]any{"type": "string"},
		})
	}
	return out
}

func securitySchemes() map[string]any {
	return map[string]any{
		"securitySchemes": map[string]any{
			"sessionCookie": map[string]any{
				"type":        "apiKey",
				"in":          "cookie",
				"name":        "picode_session",
				"description": "Issued when a browser pairs with the daemon.",
			},
		},
	}
}
