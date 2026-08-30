package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cfpperche/picode/internal/termopts"
	"github.com/cfpperche/picode/internal/tmux"
)

func termSettings(t *testing.T, ts *httptest.Server, path string) map[string]any {
	t.Helper()
	res, err := ts.Client().Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d", path, res.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return out
}

func strMap(t *testing.T, v any, field string) map[string]string {
	t.Helper()
	raw, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("%s is %T, want an object", field, v)
	}
	out := map[string]string{}
	for k, x := range raw {
		s, ok := x.(string)
		if !ok {
			t.Fatalf("%s[%q] is %T, want a string", field, k, x)
		}
		out[k] = s
	}
	return out
}

// A fresh install has nothing stored, and must still answer with something the
// panel can render: the flag list, and the built-in defaults as the effective
// values.
func TestGlobalTermSettingsStartAtTheBuiltInDefaults(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	got := termSettings(t, ts, "/api/terminals/settings")
	if len(strMap(t, got["values"], "values")) != 0 {
		t.Fatalf("values = %v, want nothing set yet", got["values"])
	}
	if eff := strMap(t, got["effective"], "effective"); eff["mouse"] != "on" {
		t.Fatalf("effective mouse = %q, want the built-in default on", eff["mouse"])
	}
	flags, ok := got["flags"].([]any)
	if !ok || len(flags) == 0 {
		t.Fatalf("flags = %v, want the registry", got["flags"])
	}
}

func TestPatchGlobalTermSettings(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	res := postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/settings", map[string]any{"mouse": "off"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("patch = %d", res.StatusCode)
	}
	res.Body.Close()
	got := termSettings(t, ts, "/api/terminals/settings")
	if eff := strMap(t, got["effective"], "effective"); eff["mouse"] != "off" {
		t.Fatalf("effective mouse = %q, wanted the saved off", eff["mouse"])
	}
}

// Clearing a field has to return it to the default, not pin the value it was
// showing. This is the difference between "inherit" and "happens to match".
func TestClearingAGlobalFieldReturnsItToTheDefault(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/settings", map[string]any{"mouse": "off"}).Body.Close()
	postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/settings", map[string]any{"mouse": nil}).Body.Close()

	got := termSettings(t, ts, "/api/terminals/settings")
	if len(strMap(t, got["values"], "values")) != 0 {
		t.Fatalf("values = %v, want the field gone rather than pinned", got["values"])
	}
	if eff := strMap(t, got["effective"], "effective"); eff["mouse"] != "on" {
		t.Fatalf("effective mouse = %q, want the default back", eff["mouse"])
	}
}

func TestTermSettingsRefuseWhatTheRegistryDoesNotKnow(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	for _, body := range []map[string]any{
		{"history-limit": "50000"}, // a real tmux option, but not one we offer
		{"mouse": "maybe"},         // a value the flag does not take
	} {
		res := postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/settings", body)
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("patch %v = %d, want 400", body, res.StatusCode)
		}
		res.Body.Close()
	}
}

func newTerminal(t *testing.T, ts *httptest.Server) (id, session string) {
	t.Helper()
	res := postJSON(t, ts, "/api/terminals", map[string]any{"cwd": t.TempDir()})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create terminal = %d", res.StatusCode)
	}
	defer res.Body.Close()
	var view map[string]any
	if err := json.NewDecoder(res.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	id, _ = view["id"].(string)
	session, _ = view["session"].(string)
	if id == "" || session == "" {
		t.Fatalf("create terminal returned %+v", view)
	}
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), session) })
	return id, session
}

// The whole point of live inheritance: a terminal that has overridden nothing
// follows a later change to the global.
func TestATerminalWithoutOverridesFollowsTheGlobal(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	id, _ := newTerminal(t, ts)

	postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/settings", map[string]any{"mouse": "off"}).Body.Close()

	got := termSettings(t, ts, "/api/terminals/"+id+"/settings")
	if len(strMap(t, got["values"], "values")) != 0 {
		t.Fatalf("values = %v, want the terminal to still override nothing", got["values"])
	}
	if inh := strMap(t, got["inherited"], "inherited"); inh["mouse"] != "off" {
		t.Fatalf("inherited mouse = %q, want the new global off", inh["mouse"])
	}
	if eff := strMap(t, got["effective"], "effective"); eff["mouse"] != "off" {
		t.Fatalf("effective mouse = %q, want to follow the global", eff["mouse"])
	}
}

// And the other half: an override survives a global change instead of being
// overwritten by the pass that updates everyone else.
func TestAnOverrideSurvivesAGlobalChange(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	id, session := newTerminal(t, ts)

	postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/"+id+"/settings", map[string]any{"mouse": "off"}).Body.Close()
	if got := tmuxOption(t, session, "mouse"); got != "off" {
		t.Fatalf("mouse = %q on the live session, want the override applied without a reattach", got)
	}

	// The global moves the other way. Everyone else follows; this one must not.
	postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/settings", map[string]any{"mouse": "on"}).Body.Close()

	got := termSettings(t, ts, "/api/terminals/"+id+"/settings")
	if eff := strMap(t, got["effective"], "effective"); eff["mouse"] != "off" {
		t.Fatalf("effective mouse = %q, want the override to win over the global", eff["mouse"])
	}
	if live := tmuxOption(t, session, "mouse"); live != "off" {
		t.Fatalf("live session mouse = %q, want the global pass to leave an override alone", live)
	}
}

// A terminal called "settings" must reach its own panel, not the global one.
// The route only works because Go's mux prefers a literal segment to {id};
// this pins that the two cannot be confused.
func TestATerminalNamedSettingsGetsItsOwnScope(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	res := postJSON(t, ts, "/api/terminals", map[string]any{"name": "settings", "cwd": t.TempDir()})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", res.StatusCode)
	}
	var view map[string]any
	_ = json.NewDecoder(res.Body).Decode(&view)
	res.Body.Close()
	id, _ := view["id"].(string)
	session, _ := view["session"].(string)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), session) })

	postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/"+id+"/settings", map[string]any{"mouse": "off"}).Body.Close()

	if global := termSettings(t, ts, "/api/terminals/settings"); len(strMap(t, global["values"], "values")) != 0 {
		t.Fatalf("the global scope was written by a terminal's patch: %v", global["values"])
	}
	if scope, _ := termSettings(t, ts, "/api/terminals/"+id+"/settings")["scope"].(string); scope != id {
		t.Fatalf("scope = %q, want the terminal id %q", scope, id)
	}
	if scope, _ := termSettings(t, ts, "/api/terminals/settings")["scope"].(string); scope != termopts.GlobalScope {
		t.Fatalf("global scope = %q, want %q", scope, termopts.GlobalScope)
	}
}

// A new terminal is born with the global default, not with a copy of it: the
// session tmux actually runs has to match what the panel says.
func TestANewTerminalIsCreatedWithTheGlobalDefault(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/settings", map[string]any{"mouse": "off"}).Body.Close()

	_, session := newTerminal(t, ts)
	if got := tmuxOption(t, session, "mouse"); got != "off" {
		t.Fatalf("mouse = %q on a terminal created after the global changed, want off", got)
	}
}
