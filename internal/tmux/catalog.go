package tmux

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Option scopes, as tmux defines them. A PiCode terminal is one session with
// one window, so session and window options are both per-terminal; server
// options are one value for the whole tmux server this user runs.
const (
	ScopeServer  = "server"
	ScopeSession = "session"
	ScopeWindow  = "window"
)

// CatalogEntry is one tmux option as the running server reports it: its name,
// scope, current global value, and a coarse kind inferred from that value so
// a UI can pick an editor. tmux does not expose types, so the kind is a guess
// good enough for rendering — validation is tmux's own, at set-option time.
type CatalogEntry struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
	Value string `json:"value"`
	Kind  string `json:"kind"` // "bool" | "number" | "array" | "text"
}

var arrayIndex = regexp.MustCompile(`^([A-Za-z0-9-]+)\[\d+\]$`)
var numeric = regexp.MustCompile(`^-?\d+$`)

// OptionCatalog reads every option the running tmux server knows, across the
// three scopes. Values are the *global* layer — which for a session that sets
// nothing is exactly what it inherits, so the catalog doubles as the
// "inherited" column of a settings UI.
//
// Read live rather than compiled in: the catalog then always matches the tmux
// actually installed — options gained or retired between versions included —
// and the values reflect the user's own tmux.conf, which PiCode deliberately
// shares (the socket is the user's; see ADR-0024's follow-up discussion).
func (m *Manager) OptionCatalog(ctx context.Context) ([]CatalogEntry, error) {
	var out []CatalogEntry
	seen := map[string]bool{}
	for flag, scope := range map[string]string{
		"-sg": ScopeServer,
		"-g":  ScopeSession,
		"-wg": ScopeWindow,
	} {
		raw, err := m.run(ctx, "show-options", flag)
		if err != nil {
			return nil, fmt.Errorf("tmux show-options %s: %w", flag, err)
		}
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimRight(line, "\r")
			if line == "" {
				continue
			}
			name, value, _ := strings.Cut(line, " ")
			// Indexed entries (command-alias[0], status-format[1]...) fold
			// into one array-kind row; their elements are edited as a block.
			if mtch := arrayIndex.FindStringSubmatch(name); mtch != nil {
				base := mtch[1]
				if seen[scope+"/"+base] {
					continue
				}
				seen[scope+"/"+base] = true
				out = append(out, CatalogEntry{Name: base, Scope: scope, Kind: "array"})
				continue
			}
			if seen[scope+"/"+name] {
				continue
			}
			seen[scope+"/"+name] = true
			out = append(out, CatalogEntry{
				Name:  name,
				Scope: scope,
				Value: unquote(value),
				Kind:  inferKind(unquote(value)),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Scope != out[j].Scope {
			return out[i].Scope < out[j].Scope
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func inferKind(value string) string {
	switch {
	case value == "on" || value == "off":
		return "bool"
	case numeric.MatchString(value):
		return "number"
	default:
		return "text"
	}
}

// unquote strips the quoting show-options adds around values with spaces.
// tmux quotes with double quotes and escapes inner ones; for display and
// round-tripping through set-option the bare value is what we want.
func unquote(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		body := v[1 : len(v)-1]
		body = strings.ReplaceAll(body, `\"`, `"`)
		body = strings.ReplaceAll(body, `\\`, `\`)
		return body
	}
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		return v[1 : len(v)-1]
	}
	return v
}

// SetScopedOption writes one option at its scope. Session and window options
// target the session (a PiCode terminal is one window, so -w on the session
// reaches it); server options take no target — they are one value for the
// whole tmux server, and the caller is responsible for only doing that from
// a surface labelled as machine-wide.
func (m *Manager) SetScopedOption(ctx context.Context, scope, session, key, value string) error {
	switch scope {
	case ScopeServer:
		_, err := m.run(ctx, "set-option", "-s", key, value)
		return err
	case ScopeWindow:
		_, err := m.run(ctx, "set-option", "-w", "-t", session+":", key, value)
		return err
	default:
		_, err := m.run(ctx, "set-option", "-t", session+":", key, value)
		return err
	}
}

// UnsetScopedOption removes a session/window-level value so the option falls
// back to the layer above (the global, then tmux's default). Server options
// have no layer above; unsetting one restores tmux's compiled default.
func (m *Manager) UnsetScopedOption(ctx context.Context, scope, session, key string) error {
	switch scope {
	case ScopeServer:
		_, err := m.run(ctx, "set-option", "-su", key)
		return err
	case ScopeWindow:
		_, err := m.run(ctx, "set-option", "-wu", "-t", session+":", key)
		return err
	default:
		_, err := m.run(ctx, "set-option", "-u", "-t", session+":", key)
		return err
	}
}

// ScopedValue is one option resolved for application: which scope to write
// it at, and what to write. The server layer builds these; Bridge applies
// them on attach.
type ScopedValue struct {
	Scope string
	Key   string
	Value string
}
