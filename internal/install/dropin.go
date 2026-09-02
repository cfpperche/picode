package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DropInFile is the systemd drop-in that carries the unit's environment
// (ADR-0050). It lives beside the unit, in picode.service.d/, so `picode
// deploy` can rewrite the unit without touching what the operator set.
const DropInFile = "env.conf"

// DropInPath is ~/.config/systemd/user/picode.service.d/env.conf.
func DropInPath(home string) string {
	return filepath.Join(home, ".config", "systemd", "user", UnitName+".d", DropInFile)
}

// ReadEnvDropIn parses the drop-in's Environment= lines. A missing file is
// an empty map, not an error.
func ReadEnvDropIn(home string) (map[string]string, error) {
	b, err := os.ReadFile(DropInPath(home))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	return parseDropIn(string(b)), nil
}

// WriteEnvDropIn merges env over the existing drop-in (given keys win,
// others stay; an empty value removes the key) and writes it. Returns
// the path. The caller runs daemon-reload.
func WriteEnvDropIn(home string, env map[string]string) (string, error) {
	cur, err := ReadEnvDropIn(home)
	if err != nil {
		return "", err
	}
	for k, v := range env {
		if v == "" {
			delete(cur, k)
			continue
		}
		cur[k] = v
	}
	path := DropInPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, []byte(renderDropIn(cur)), 0o644)
}

// ParseEnvFlag splits KEY=VALUE; the key must look like a variable name.
func ParseEnvFlag(s string) (string, string, error) {
	k, v, ok := strings.Cut(s, "=")
	k = strings.TrimSpace(k)
	if !ok || k == "" || strings.ContainsAny(k, " \t\"'") {
		return "", "", fmt.Errorf("--env wants KEY=VALUE, got %q", s)
	}
	return k, v, nil
}

func renderDropIn(env map[string]string) string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# Written by picode install --env. Edit or delete freely; picode deploy keeps it.\n")
	b.WriteString("[Service]\n")
	for _, k := range keys {
		b.WriteString("Environment=" + quoteEnv(k+"="+env[k]) + "\n")
	}
	return b.String()
}

// quoteEnv follows systemd's quoting: bare when safe, else double-quoted
// with backslash escapes.
func quoteEnv(kv string) string {
	if !strings.ContainsAny(kv, " \t\"'\\#;") {
		return kv
	}
	return "\"" + strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(kv) + "\""
}

func parseDropIn(text string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Environment=") {
			continue
		}
		rest := strings.TrimPrefix(line, "Environment=")
		for _, kv := range splitQuoted(rest) {
			if k, v, ok := strings.Cut(kv, "="); ok && k != "" {
				out[k] = v
			}
		}
	}
	return out
}

// splitQuoted splits a systemd value list: whitespace-separated words,
// double-quoted words may hold spaces and backslash escapes.
func splitQuoted(s string) []string {
	var out []string
	var cur strings.Builder
	inQ, esc, have := false, false, false
	for _, r := range s {
		switch {
		case esc:
			cur.WriteRune(r)
			esc = false
		case inQ && r == '\\':
			esc = true
		case r == '"':
			inQ = !inQ
			have = true
		case !inQ && (r == ' ' || r == '\t'):
			if have || cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
				have = false
			}
		default:
			cur.WriteRune(r)
			have = true
		}
	}
	if have || cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}
