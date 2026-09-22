package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/pkgs"
	"github.com/cfpperche/picode/internal/store"
)

// The unified read answers for any CLI with a driver, and refuses one without
// by naming what exists (ADR-0176).
func TestPackageReportAnswersEveryDriver(t *testing.T) {
	userDir := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return userDir }
	t.Cleanup(func() { pipkg.UserDir = old })
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "settings.json"), []byte(`{"packages":["npm:pi-web-search"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st := testStore(t)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)

	res, err := http.Get(ts.URL + "/api/packages/report")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d %s", res.StatusCode, body)
	}
	for _, want := range []string{`"cli":"pi"`, `"catalog":"gallery"`, `"install":true`, `"config":true`, `"id":"machine"`, `pi-web-search`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("report %s lacks %s", body, want)
		}
	}
	// A read that named no workspace and no agent declares neither layer: a
	// radio that cannot work is not offered (the plan's decision table).
	for _, unwanted := range []string{`"id":"workspace"`, `"id":"agent"`} {
		if strings.Contains(string(body), unwanted) {
			t.Fatalf("machine report offers a layer it cannot honour: %s", body)
		}
	}

	res, err = http.Get(ts.URL + "/api/packages/report?cli=nope")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ = io.ReadAll(res.Body)
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "omp") {
		t.Fatalf("unknown CLI = %d %s, want 400 naming the drivers", res.StatusCode, body)
	}
}

// A pane asks the engine which CLIs it can render, so every driver the engine
// lists answers this route — a vendor PiCode cannot reach is the vendor's own
// failure (502), never "no packages driver". The JS list this replaced
// (CLI_PACKAGES/GUEST_PACKAGES) is gone: the engine is the source (ADR-0176
// slice 3).
func TestPackageReportAnswersEveryEngineDriver(t *testing.T) {
	st := testStore(t)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)
	// The vendor binaries are not installed here, so a guest read may fail —
	// what must never happen is the route refusing a CLI it has a driver for.
	for _, id := range pkgs.CLIs() {
		res, err := http.Get(ts.URL + "/api/packages/report?cli=" + id)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "no packages driver") {
			t.Fatalf("%s: %d %s", id, res.StatusCode, body)
		}
	}
}

// The isolation switch rides the unified report: a pane that draws its checkbox
// reads the fact here, not from the legacy route (ADR-0176 slice 3, the debt in
// docs/handoff/open/packages.md).
func TestPackageReportCarriesTheIsolationSwitch(t *testing.T) {
	userDir := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return userDir }
	t.Cleanup(func() { pipkg.UserDir = old })
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	st := testStore(t)
	ws, err := st.AddWorkspace("Isolated", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.AddAgent(ws.ID, "gpt", "")
	if err != nil {
		t.Fatal(err)
	}
	on := true
	if _, err := st.UpdateAgent(agent.ID, store.AgentPatch{PackagesIsolated: &on}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)
	res, err := http.Get(ts.URL + "/api/packages/report?agent=" + agent.ID + "&workspace=" + ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d %s", res.StatusCode, body)
	}
	if !strings.Contains(string(body), `"isolated":true`) {
		t.Fatalf("report drops the isolation switch: %s", body)
	}
	// The workspace radio names the folder the install lands in, and the agent
	// scope spells the agent out — both are the caller's own words for the
	// layers PiCode holds, not the CLI's.
	for _, want := range []string{`"label":"Isolated"`, `"note":"Only gpt, every session"`, `"agentName":"gpt"`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("report lacks %s: %s", want, body)
		}
	}
}
