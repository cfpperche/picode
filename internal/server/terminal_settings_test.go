package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
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

// The whole catalog is writable now, so refusal is for what cannot be
// honest: a name this tmux does not know, a curated value outside its enum,
// and a non-curated value tmux itself rejects — in tmux's own words.
func TestTermSettingsRefuseWhatCannotBeApplied(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	for _, body := range []map[string]any{
		{"mouse-speed": "fast"}, // no such option
		{"mouse": "maybe"},      // curated flag, value outside its enum
		// A value tmux refuses. Chosen with care: tmux normalises yes/no
		// onto booleans, so "yes" on a bool is VALID — a number option fed
		// text is what actually errors ("value is invalid").
		{"display-time": "abc"},
	} {
		res := postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/settings", body)
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("patch %v = %d, want 400", body, res.StatusCode)
		}
		res.Body.Close()
	}
}

// And the counterpart: a real option outside the curated tier is accepted,
// stored, and applied to a live owned session at the right scope — including
// a window-scoped one, which needs `set-option -w`.
func TestANonCuratedOptionIsStoredAndApplied(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	id, session := newTerminal(t, ts)

	res := postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/"+id+"/settings", map[string]any{"mode-keys": "vi"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("patch mode-keys = %d", res.StatusCode)
	}
	res.Body.Close()

	out, err := exec.Command("tmux", "show-options", "-w", "-t", session+":", "-v", "mode-keys").Output()
	if err != nil || strings.TrimSpace(string(out)) != "vi" {
		t.Fatalf("mode-keys on the window = %q (%v), want vi", strings.TrimSpace(string(out)), err)
	}

	// Clearing it must UNSET the live value, not leave it pinned: mode-keys
	// has no PiCode default underneath, so nothing would ever overwrite it.
	postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/"+id+"/settings", map[string]any{"mode-keys": nil}).Body.Close()
	out, _ = exec.Command("tmux", "show-options", "-w", "-t", session+":", "-v", "mode-keys").Output()
	if strings.TrimSpace(string(out)) == "vi" {
		t.Fatal("cleared option is still set on the live window")
	}
}

// A server-wide option offered per terminal would be a lie — the panel labels
// it machine-wide and the API holds the same line.
func TestAServerScopedOptionIsRefusedPerTerminal(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	id, _ := newTerminal(t, ts)
	res := postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/"+id+"/settings", map[string]any{"escape-time": "50"})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("per-terminal escape-time = %d, want 400", res.StatusCode)
	}
	res.Body.Close()
}

func TestCatalogListsAllScopesWithWarnings(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	got := termSettings(t, ts, "/api/terminals/settings/catalog")
	rows, ok := got["catalog"].([]any)
	if !ok || len(rows) < 100 {
		t.Fatalf("catalog has %d rows, want the full option space (~159)", len(rows))
	}
	byName := map[string]map[string]any{}
	for _, r := range rows {
		row := r.(map[string]any)
		byName[row["name"].(string)] = row
	}
	if r := byName["destroy-unattached"]; r == nil || r["danger"] == "" || r["danger"] == nil {
		t.Error("destroy-unattached carries no warning — the panel must label it")
	}
	if r := byName["mouse"]; r == nil || r["curated"] != true {
		t.Error("mouse is not marked curated")
	}
	if r := byName["escape-time"]; r == nil || r["scope"] != "server" {
		t.Error("escape-time is not marked server scope")
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

// A global change must reach only the sessions this instance owns. tmux is
// machine-wide: another PiCode, an older session, or a test with its own
// database all carry the same `picode-` prefix. This is not hypothetical —
// before the fix, running the test above flipped `mouse` on the developer's
// own terminals, because the apply pass walked `tmux list-sessions`.
func TestAGlobalChangeLeavesForeignSessionsAlone(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux not installed")
	}
	ctx := context.Background()

	// A PiCode-owned name this server's store knows nothing about.
	foreign := tmux.ShellSessionName("foreign-" + t.Name())
	if err := tm.NewSession(ctx, foreign, t.TempDir(), "cat"); err != nil {
		t.Fatalf("seed foreign session: %v", err)
	}
	t.Cleanup(func() { _ = tm.KillSession(ctx, foreign) })
	if err := tm.SetOption(ctx, foreign, "mouse", "on"); err != nil {
		t.Fatalf("seed option: %v", err)
	}

	postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/settings", map[string]any{"mouse": "off"}).Body.Close()

	if got := tmuxOption(t, foreign, "mouse"); got != "on" {
		t.Fatalf("foreign session mouse = %q, want the untouched on — a global change reached a session this store does not own", got)
	}
}

// The counterpart to the test above, so scoping the apply cannot quietly
// become applying to nothing: a global change still reaches the live session
// of a terminal this instance does own.
func TestAGlobalChangeReachesAnOwnedLiveSession(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	_, session := newTerminal(t, ts)
	if got := tmuxOption(t, session, "mouse"); got != "on" {
		t.Fatalf("mouse = %q at creation, want the default on", got)
	}

	postJSONMethod(t, ts, http.MethodPatch, "/api/terminals/settings", map[string]any{"mouse": "off"}).Body.Close()

	if got := tmuxOption(t, session, "mouse"); got != "off" {
		t.Fatalf("mouse = %q, want the global change to reach an owned session with no reattach", got)
	}
}
