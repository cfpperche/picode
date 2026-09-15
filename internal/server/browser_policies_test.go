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
	"testing"

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
