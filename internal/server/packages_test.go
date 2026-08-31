package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPackagesUnknownAgentStillListsMachine(t *testing.T) {
	st := testStore(t)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)
	res, err := http.Get(ts.URL + "/api/packages?agent=not-an-agent")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	var body struct {
		Packages []any `json:"packages"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Packages == nil {
		t.Fatal("packages missing")
	}
}
