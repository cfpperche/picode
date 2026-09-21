package clipkgs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/connectors"
)

// OpenCode is the one guest whose plugins PiCode reads from files instead of a
// roster command, and the one whose removal PiCode performs itself: the CLI
// exposes `opencode plugin <module> [-g]` to add and nothing to remove, list or
// disable — the `plugin` array of its own config is the store.
//
// The path rule (which files, in which precedence, with XDG honored) is
// asked of internal/connectors, which measured it against OpenCode 1.18.31
// on 2026-09-18. The module list itself was read from
// https://opencode.ai/docs/plugins/ on 2026-09-20: npm modules named in the
// config, plus `.js`/`.ts` files that load from a plugins/ directory by their
// presence.

// opencodeRoster lists both halves of OpenCode's plugin store: the npm modules
// its configs name, and the local files its two plugin directories hold.
func opencodeRoster(_ context.Context, p Paths, scope string, available bool) ([]Row, string, error) {
	if available {
		return nil, "", fmt.Errorf("%w: OpenCode has no plugin marketplace", ErrVerbAbsent)
	}
	rows := []Row{}
	seen := map[string]bool{}
	for _, file := range opencodeConfigFiles(p, scope) {
		mods, err := opencodeModules(file)
		if err != nil {
			return nil, "", err
		}
		for _, mod := range mods {
			key := strings.ToLower(mod)
			if seen[key] {
				continue
			}
			seen[key] = true
			rows = append(rows, Row{
				ID:          mod,
				Name:        packageName(mod),
				Version:     packageVersion(mod),
				Source:      mod,
				SourceKind:  "npm",
				Scope:       scope,
				Enabled:     true,
				Installed:   true,
				InstallPath: file,
			})
		}
	}
	for _, dir := range opencodePluginDirs(p, scope) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		names := []string{}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || strings.HasPrefix(name, ".") {
				continue
			}
			if ext := strings.ToLower(filepath.Ext(name)); ext != ".js" && ext != ".ts" && ext != ".mjs" {
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			path := filepath.Join(dir, name)
			if seen[strings.ToLower(path)] {
				continue
			}
			seen[strings.ToLower(path)] = true
			rows = append(rows, Row{
				ID:          path,
				Name:        name,
				Source:      path,
				SourceKind:  "local-file",
				Scope:       scope,
				Enabled:     true,
				Installed:   true,
				InstallPath: path,
				Note:        "Loads from its directory; OpenCode has no command to turn one off, so removing the file is the way.",
			})
		}
	}
	return rows, "", nil
}

// opencodeModules reads one config file's `plugin` array. A missing file is an
// empty list; a file that does not parse is an error naming it, never an empty
// roster (ADR-0150's rule for a malformed user file, applied to a read).
func opencodeModules(file string) ([]string, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(b)) == "" {
		return nil, nil
	}
	doc, err := connectors.LenientJSON(b)
	if err != nil {
		return nil, fmt.Errorf("%s is not valid JSON: %v", file, err)
	}
	switch v := doc["plugin"].(type) {
	case nil:
		return nil, nil
	case string:
		// Measured 2026-09-21: OpenCode refuses this file itself. Running its
		// own `opencode plugin <module>` against `"plugin": "is-odd"` prints
		// *"Configuration is invalid … Expected array | undefined, got
		// \"is-odd\" plugin"* and writes nothing. Listing the string as an
		// installed module would show a plugin the CLI never loads, so the read
		// fails loudly instead and the pane names the file and the vendor's
		// expectation.
		return nil, fmt.Errorf("%s declares a plugin as a string; OpenCode expects a list (\"plugin\": [\"name\"])", file)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out, nil
	}
	return nil, fmt.Errorf("%s: the plugin key is not a list of module names", file)
}

// opencodeConfigFiles is the candidate list for one scope, in the order a
// plugin is looked for: the files the *vendor's own* install command writes
// first, then the project config `internal/connectors` measured. The
// divergence was measured on 2026-09-20 by the live harness — `opencode plugin
// <module>` with no -g writes <cwd>/.opencode/opencode.json and
// `opencode debug config` lists it as a plugin origin — so a project-scope
// install was invisible to the roster, and a removal would have edited the
// wrong file. The path rule for the documented project file stays with
// connectors; only this extra location is declared here.
func opencodeConfigFiles(p Paths, scope string) []string {
	files := connectors.OpenCodeConfigFiles(p.home(), p.Cwd, scope)
	if scope != "project" || p.Cwd == "" {
		return files
	}
	dir := filepath.Join(p.Cwd, ".opencode")
	return append([]string{
		filepath.Join(dir, "opencode.jsonc"),
		filepath.Join(dir, "opencode.json"),
	}, files...)
}

// opencodePluginDirs returns the directories OpenCode loads local plugins from
// for one scope, in the vendor's documented order.
func opencodePluginDirs(p Paths, scope string) []string {
	if scope == "project" {
		if p.Cwd == "" {
			return nil
		}
		return []string{filepath.Join(p.Cwd, ".opencode", "plugins")}
	}
	return []string{filepath.Join(connectors.OpenCodeConfigDir(p.home()), "plugins")}
}

// errNoElement is the splice's "this file does not name it" signal; the caller
// moves on to the next config file and only reports it when none matched.
var errNoElement = errors.New("the config does not name this module")

// opencodeRemove deletes one npm module from the `plugin` array of the config
// file that names it (ADR-0167). The edit is surgical: only the element and
// its comma leave the file, and the result is re-parsed and compared before it
// is written, so a shape this scanner mis-reads is refused instead of saved.
func opencodeRemove(_ context.Context, p Paths, t Target) error {
	module := clean(firstText(t.Source, t.Name))
	if module == "" {
		return ErrBadTarget
	}
	files := opencodeConfigFiles(p, t.Scope)
	if len(files) == 0 {
		return ErrNoWorkspace
	}
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		next, err := removeArrayElement(b, "plugin", module)
		if errors.Is(err, errNoElement) {
			continue
		}
		if err != nil {
			return err
		}
		if err := writeAtomic(file, next); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("%w: no OpenCode config names %s", ErrStale, module)
}

// removeArrayElement returns src with one string element removed from the
// array under key. Offsets come from a comment-blanked copy of the same
// length, so every byte outside the removed element survives verbatim.
func removeArrayElement(src []byte, key, element string) ([]byte, error) {
	scan := blankComments(src)
	spans, err := arrayElementSpans(scan, key)
	if err != nil {
		return nil, err
	}
	if len(spans) == 0 {
		return nil, errNoElement
	}
	lit := marshalString(element)
	target := -1
	for i, sp := range spans {
		if string(scan[sp.start:sp.end]) == lit {
			target = i
			break
		}
	}
	if target < 0 {
		return nil, errNoElement
	}
	start, end := spans[target].start, spans[target].end
	switch {
	case target+1 < len(spans):
		end = spans[target+1].start
	case target > 0:
		start = spans[target-1].end
	}
	next := append(append([]byte{}, src[:start]...), src[end:]...)
	if err := verifySplice(src, next, key, lit); err != nil {
		return nil, err
	}
	return next, nil
}

// verifySplice re-parses both documents and refuses when the only difference
// is not exactly one element leaving the named array.
func verifySplice(before, after []byte, key, lit string) error {
	was, err := connectors.LenientJSON(before)
	if err != nil {
		return fmt.Errorf("the config is not valid JSON: %v", err)
	}
	now, err := connectors.LenientJSON(after)
	if err != nil {
		return fmt.Errorf("the edit would not parse: %v", err)
	}
	want := modulesOf(was[key])
	got := modulesOf(now[key])
	var found bool
	kept := make([]string, 0, len(want))
	for _, m := range want {
		if !found && marshalString(m) == lit {
			found = true
			continue
		}
		kept = append(kept, m)
	}
	if !found || !equalStrings(kept, got) {
		return fmt.Errorf("%w: the plugin list did not change as expected", ErrStale)
	}
	delete(was, key)
	delete(now, key)
	if !reflect.DeepEqual(was, now) {
		return fmt.Errorf("%w: the edit touched more than the plugin list", ErrStale)
	}
	return nil
}

func modulesOf(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// arrayElementSpans finds the string elements of the array under a top-level
// key, as [start,end) byte spans over scan (quotes included). Only string
// elements are returned: a value PiCode cannot splice is left for
// verifySplice to refuse.
func arrayElementSpans(scan []byte, key string) ([]span, error) {
	arrStart, err := topLevelArray(scan, key)
	if err != nil {
		return nil, err
	}
	if arrStart < 0 {
		return nil, nil
	}
	spans := []span{}
	depth := 0
	expectElement := true
	for i := arrStart; i < len(scan); {
		switch scan[i] {
		case '"':
			start := i
			end := skipString(scan, i)
			if depth == 1 && expectElement {
				spans = append(spans, span{start: start, end: end})
			}
			i = end
			continue
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return spans, nil
			}
		case ',':
			expectElement = depth == 1
		case ' ', '\t', '\n', '\r':
		default:
			if depth == 1 {
				expectElement = false
			}
		}
		i++
	}
	return nil, fmt.Errorf("the plugin list is not closed")
}

// topLevelArray returns the index of the '[' of a top-level key's array, or -1
// when the document has no such key.
func topLevelArray(scan []byte, key string) (int, error) {
	depth := 0
	for i := 0; i < len(scan); i++ {
		switch scan[i] {
		case '"':
			if depth == 1 {
				end := skipString(scan, i)
				if string(scan[i:end]) == marshalString(key) {
					j := end
					for j < len(scan) && (scan[j] == ' ' || scan[j] == '\t' || scan[j] == '\n' || scan[j] == '\r' || scan[j] == ':') {
						j++
					}
					if j < len(scan) && scan[j] == '[' {
						return j, nil
					}
				}
				i = end - 1
				continue
			}
			i = skipString(scan, i) - 1
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		}
	}
	return -1, nil
}

type span struct{ start, end int }

// skipString returns the index just past the closing quote of the string
// beginning at start (so b[start:skipString(b,start)] is the quoted literal).
func skipString(b []byte, start int) int {
	esc := false
	for i := start + 1; i < len(b); i++ {
		switch {
		case esc:
			esc = false
		case b[i] == '\\':
			esc = true
		case b[i] == '"':
			return i + 1
		}
	}
	return len(b)
}

// blankComments replaces the bytes of // and /* */ comments with spaces so a
// scan sees JSON with the original offsets. Strings are copied verbatim. The
// read-side stripper in internal/connectors drops the bytes instead, which is
// what a decode needs and what an edit cannot use.
func blankComments(b []byte) []byte {
	out := append([]byte{}, b...)
	inStr, esc := false, false
	for i := 0; i < len(out); i++ {
		c := out[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '/':
			if i+1 < len(out) && out[i+1] == '/' {
				for i < len(out) && out[i] != '\n' {
					out[i] = ' '
					i++
				}
			} else if i+1 < len(out) && out[i+1] == '*' {
				i += 2
				for i < len(out) && !(out[i] == '*' && i+1 < len(out) && out[i+1] == '/') {
					out[i] = ' '
					i++
				}
				if i+1 < len(out) {
					out[i], out[i+1] = ' ', ' '
					i++
				}
			}
		}
	}
	return out
}

func marshalString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

func modeOf(path string) os.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm()
	}
	return 0o644
}

// writeAtomic lands tmp+rename with the target's own mode, so a crash never
// leaves half a vendor config behind (the ADR-0150 rule, applied to the one
// plugin-list write PiCode performs itself).
func writeAtomic(path string, b []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(modeOf(path)); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// packageName strips an npm spec down to the module name the pane shows:
// "npm:@scope/name@1.2.3" → "@scope/name".
func packageName(spec string) string {
	s := strings.TrimPrefix(clean(spec), "npm:")
	s = strings.TrimPrefix(s, "bun:")
	if i := versionAt(s); i >= 0 {
		return s[:i]
	}
	return s
}

func packageVersion(spec string) string {
	s := strings.TrimPrefix(clean(spec), "npm:")
	s = strings.TrimPrefix(s, "bun:")
	if i := versionAt(s); i >= 0 {
		return strings.TrimPrefix(s[i+1:], "v")
	}
	return ""
}

// versionAt finds the '@' that separates a package name from its version: the
// last one whose next byte starts a version, which keeps a leading scope
// (@scope/name) intact.
func versionAt(s string) int {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] != '@' {
			continue
		}
		if i+1 < len(s) {
			c := s[i+1]
			if c >= '0' && c <= '9' || c == '^' || c == '~' || c == '*' || c == 'v' {
				return i
			}
		}
	}
	return -1
}
