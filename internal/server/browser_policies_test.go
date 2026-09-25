package server

// The grants editor's endpoints, table-tested: the read side lists every
// agent with its effective grant (saved or ADR-0134 default), the write
// side round-trips a grant, refuses bad tiers and unknown agents, and
// normalizes the domain table (trim, lowercase, drop empties).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
)

type policiesFixtureT struct {
	ts  *httptest.Server
	ids map[string]string // agent name -> store id
}

func policiesFixture(t *testing.T) *policiesFixtureT {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ws, err := st.AddWorkspace("main", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scout, err := st.AddAgent(ws.ID, "scout", "")
	if err != nil {
		t.Fatal(err)
	}
	scribe, err := st.AddAgent(ws.ID, "scribe", "")
	if err != nil {
		t.Fatal(err)
	}
	deps := Deps{Store: st}
	return &policiesFixtureT{
		ts:  httptest.NewServer(New("127.0.0.1:0", deps).Handler),
		ids: map[string]string{"scout": scout.ID, "scribe": scribe.ID},
	}
}

func TestPoliciesListDefaultsWhenNothingSaved(t *testing.T) {
	f := policiesFixture(t)
	resp, err := http.Get(f.ts.URL + "/api/browser/policies")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET status %d", resp.StatusCode)
	}
	var body struct {
		Policies []struct {
			AgentID string   `json:"agentId"`
			Name    string   `json:"name"`
			Tier    string   `json:"tier"`
			Domains []string `json:"domains"`
			Saved   bool     `json:"saved"`
		} `json:"policies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Policies) != 2 { // scout + scribe
		t.Fatalf("want 2 grants, got %d", len(body.Policies))
	}
	for _, g := range body.Policies {
		if g.Tier != "read" || g.Saved || len(g.Domains) != 0 {
			t.Fatalf("%s: want the untouched default, got tier=%q saved=%v domains=%v", g.AgentID, g.Tier, g.Saved, g.Domains)
		}
	}
}

func TestPolicySaveRoundTripsThroughTheList(t *testing.T) {
	f := policiesFixture(t)
	payload, _ := json.Marshal(map[string]any{"agent": f.ids["scout"], "tier": "act", "domains": []string{" Example.COM ", "", "*.example.org"}})
	resp, err := http.Post(f.ts.URL+"/api/browser/policy", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var b bytes.Buffer
		_, _ = b.ReadFrom(resp.Body)
		t.Fatalf("POST status %d body %s", resp.StatusCode, b.String())
	}
	policies := listPolicies(t, f.ts)
	var saved map[string]any
	for _, p := range policies {
		if g := p.(map[string]any); g["agentId"] == f.ids["scout"] {
			saved = g
		}
	}
	if saved == nil || saved["tier"] != "act" || saved["saved"] != true {
		t.Fatalf("scout's grant did not persist: %v", saved)
	}
	if doms := saved["domains"].([]any); len(doms) != 2 || doms[0] != "example.com" || doms[1] != "*.example.org" {
		t.Fatalf("domains not normalized: %v", doms)
	}
}

func TestPolicySaveRefusesBadTierAndUnknownAgent(t *testing.T) {
	f := policiesFixture(t)
	rows := []struct {
		name string
		body map[string]any
		want int
	}{
		{"bad tier", map[string]any{"agent": f.ids["scout"], "tier": "root"}, http.StatusBadRequest},
		{"unknown agent", map[string]any{"agent": "ghost", "tier": "read"}, http.StatusNotFound},
		{"empty agent", map[string]any{"agent": " ", "tier": "read"}, http.StatusBadRequest},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			payload, _ := json.Marshal(row.body)
			resp, err := http.Post(f.ts.URL+"/api/browser/policy", "application/json", bytes.NewReader(payload))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != row.want {
				t.Fatalf("status %d, want %d", resp.StatusCode, row.want)
			}
		})
	}
}

// TestPolicySaveTerminalWireRows pins ADR-0143's terminal-principal rows on
// the grants API's wire — the debt the work-browser board carries: the
// resolver's decision rows were covered (TestResolveCallerIsTheHouseIdentity),
// these were not. A known terminal saves under the term:<id> key and shows
// up as the listing's terminal row; an unknown one is a 404 that names it;
// agent+term together is refused — one principal per request, so a term id
// can never widen an agent's grant or the other way round.
func TestPolicySaveTerminalWireRows(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ws, err := st.AddWorkspace("main", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scout, err := st.AddAgent(ws.ID, "scout", "")
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(store.FreeWorkspaceID, "cli", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(term.ID, "pi", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st}).Handler)
	t.Cleanup(ts.Close)

	// ADR-0184: an unbound terminal holds no grant and has no row; bound
	// to an agent, a term id edits that agent's grant.
	if code, page := postRaw(t, ts, "/api/browser/policy", `{"term":"`+term.ID+`","tier":"act"}`); code != http.StatusBadRequest {
		t.Fatalf("shell term = %d %v", code, page)
	}
	cli, err := st.AddAgentWithCLI(ws.ID, "pi", "cli", "")
	if err != nil {
		t.Fatal(err)
	}
	tid := term.ID
	if _, err := st.UpdateAgent(cli.ID, store.AgentPatch{TerminalID: &tid}); err != nil {
		t.Fatal(err)
	}
	code, page := postRaw(t, ts, "/api/browser/policy", `{"term":"`+term.ID+`","tier":"act"}`)
	if code != http.StatusOK || page["saved"] != true {
		t.Fatalf("bound term = %d %v", code, page)
	}
	var agentRow map[string]any
	for _, p := range listPolicies(t, ts) {
		g := p.(map[string]any)
		if g["kind"] == "terminal" {
			t.Fatalf("terminal row listed: %v", g)
		}
		if g["agentId"] == cli.ID {
			agentRow = g
		}
	}
	if agentRow == nil || agentRow["tier"] != "act" || agentRow["saved"] != true {
		t.Fatalf("agent row did not carry the grant: %v", agentRow)
	}

	refusals := []struct {
		name  string
		body  string
		want  int
		inErr string
	}{
		{"unknown term", `{"term":"no-such-term","tier":"read"}`, http.StatusNotFound, "unknown terminal"},
		{"agent+term together", `{"agent":"` + scout.ID + `","term":"` + term.ID + `","tier":"read"}`, http.StatusBadRequest, "not both"},
	}
	for _, row := range refusals {
		t.Run(row.name, func(t *testing.T) {
			code, page := postRaw(t, ts, "/api/browser/policy", row.body)
			if code != row.want || !strings.Contains(page["error"].(string), row.inErr) {
				t.Fatalf("%s = %d %v, want %d with %q", row.name, code, page, row.want, row.inErr)
			}
		})
	}

	// The refusals saved nothing: the agent row is still the untouched
	// default, and no phantom row exists for the refused term.
	for _, p := range listPolicies(t, ts) {
		g := p.(map[string]any)
		if g["agentId"] == scout.ID && (g["saved"] != false || g["tier"] != "read") {
			t.Fatalf("refusal leaked a grant onto the agent: %v", g)
		}
		if g["kind"] == "terminal" && g["termId"] == "no-such-term" {
			t.Fatalf("refused term id grew a row: %v", g)
		}
	}
}

// listPolicies drains GET /api/browser/policies into a generic form.
func listPolicies(t *testing.T, ts *httptest.Server) []any {
	t.Helper()
	resp, err := http.Get(ts.URL + "/api/browser/policies")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Policies []any `json:"policies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Policies
}
