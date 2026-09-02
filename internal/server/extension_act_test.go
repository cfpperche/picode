package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// newActServer is newTestServer with the handles the act tests need:
// the store (to plant batches) and the deps (to arm watchers directly).
func newActServer(t *testing.T) (*httptest.Server, *store.Store, Deps) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	deps := Deps{
		Store:    st,
		Tmux:     tmux.New(),
		Runtime:  rpc.NewRuntime("cat", st, nil),
		AgentCmd: "cat",
	}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	return ts, st, deps
}

type actNextBody struct {
	Watching bool                      `json:"watching"`
	Blocked  string                    `json:"blocked"`
	Batch    *browserhost.ActBatchWire `json:"batch"`
}

func decodeActNext(t *testing.T, res *http.Response) actNextBody {
	t.Helper()
	var b actNextBody
	if err := json.NewDecoder(res.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestActNextValidation(t *testing.T) {
	ts, _, _ := newActServer(t)

	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/extension/act/next"))
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("no agent = %d", res.StatusCode)
	}
	res = do(t, ts.Client(), mustGet(t, ts.URL+"/api/extension/act/next?agent=nope"))
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing agent = %d", res.StatusCode)
	}
}

func TestActNextEmpty(t *testing.T) {
	ts, _, _ := newActServer(t)
	wk := addWorkspaceWithAgent(t, ts, "Repo", t.TempDir())
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/extension/act/next?agent="+wk.Agents[0].ID))
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	if b := decodeActNext(t, res); b.Watching || b.Blocked != "" || b.Batch != nil {
		t.Fatalf("%+v", b)
	}
}

func TestActNextOriginGateAndClaim(t *testing.T) {
	ts, st, _ := newActServer(t)
	wk := addWorkspaceWithAgent(t, ts, "Repo", t.TempDir())
	id := wk.Agents[0].ID

	if _, err := st.CreateActBatch(id, "https://x.com",
		`[{"act":"read","selector":"main"}]`, 1); err != nil {
		t.Fatal(err)
	}

	// Wrong tab: blocked, not claimed.
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/extension/act/next?agent="+id+"&tab=https://other.io"))
	if b := decodeActNext(t, res); !b.Watching || b.Blocked != "origin" || b.Batch != nil {
		t.Fatalf("%+v", b)
	}

	// Matching tab (case-insensitive origin): claimed and returned.
	res = do(t, ts.Client(), mustGet(t, ts.URL+"/api/extension/act/next?agent="+id+"&tab=https://X.com"))
	b := decodeActNext(t, res)
	if b.Batch == nil || b.Batch.Origin != "https://x.com" || b.Batch.Rounds != browserhost.ActMaxRounds {
		t.Fatalf("%+v", b)
	}
	if len(b.Batch.Actions) != 1 || b.Batch.Actions[0].Act != "read" {
		t.Fatalf("actions %+v", b.Batch.Actions)
	}

	// The claimed batch comes back on further polls (panel may re-ask).
	res = do(t, ts.Client(), mustGet(t, ts.URL+"/api/extension/act/next?agent="+id))
	if b = decodeActNext(t, res); b.Batch == nil {
		t.Fatalf("%+v", b)
	}
}

func TestActResultNotFoundAndCap(t *testing.T) {
	ts, st, _ := newActServer(t)
	res := postJSON(t, ts, "/api/extension/act/act_none/result", map[string]any{})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing batch = %d", res.StatusCode)
	}

	wk := addWorkspaceWithAgent(t, ts, "Repo", t.TempDir())
	id := wk.Agents[0].ID
	b, err := st.CreateActBatch(id, "https://x.com", `[]`, browserhost.ActMaxRounds)
	if err != nil {
		t.Fatal(err)
	}
	// Round cap, and no live agent: the loop must end either way.
	res = postJSON(t, ts, "/api/extension/act/"+b.ID+"/result", map[string]any{
		"outcomes": []map[string]any{{"act": "read", "selector": "main", "ok": true}},
	})
	if body := decodeActNext(t, res); body.Watching {
		t.Fatal("cap round kept watching")
	}
	if row, _, _ := st.GetActBatchRow(b.ID); row.State != store.ActDone {
		t.Fatalf("state %q", row.State)
	}
}

func TestActWatchSettledCreatesBatch(t *testing.T) {
	ts, st, deps := newActServer(t)
	wk := addWorkspaceWithAgent(t, ts, "Repo", t.TempDir())
	id := wk.Agents[0].ID

	// addActWatch is the panel's signal; settled must consume it.
	addActWatch(id)
	w := actWatch{deps: deps, agentID: id, origin: "https://x.com"}
	w.settled("Answer.\n```picode-act\n{\"actions\":[{\"act\":\"click\",\"selector\":\"#go\"}]}\n```")
	batch, ok2, _ := st.PendingActBatch(id)
	if !ok2 || batch.Round != 1 {
		t.Fatalf("batch not created: %+v %v", batch, ok2)
	}
	if watchingAct(id) {
		t.Fatal("watch was not released")
	}
	if err := st.FinishActBatch(batch.ID); err != nil {
		t.Fatal(err)
	}

	// A plain answer creates nothing.
	addActWatch(id)
	w2 := actWatch{deps: deps, agentID: id, origin: "https://x.com"}
	w2.settled("Just text.")
	if _, ok, _ := st.PendingActBatch(id); ok {
		t.Fatal("plain answer created a batch")
	}
	if watchingAct(id) {
		t.Fatal("watch was not released")
	}

	// A malformed block ends the loop without a batch.
	addActWatch(id)
	w3 := actWatch{deps: deps, agentID: id, origin: "https://x.com"}
	w3.settled("```picode-act\nnope\n```")
	if _, ok, _ := st.PendingActBatch(id); ok {
		t.Fatal("bad block created a batch")
	}
}
