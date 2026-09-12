package install

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Readiness reads the daemon's discovery file and asks the route; a missing
// daemon or an older one without the route answers "nothing to protect".
func TestReadinessDecisionTable(t *testing.T) {
	busy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/deploy/readiness" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ready":false,"busy":[{"kind":"agent","id":"a1","name":"Pi","why":"mid-turn"}]}`))
	}))
	defer busy.Close()
	old := httptest.NewServer(http.NotFoundHandler())
	defer old.Close()

	cases := []struct {
		name string
		url  string // "" = no server.json
		want int
	}{
		{"no server.json", "", 0},
		{"daemon reports one busy owner", busy.URL, 1},
		{"older daemon without the route", old.URL, 0},
		{"nothing listening", "http://127.0.0.1:1", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := t.TempDir()
			if c.url != "" {
				if err := os.WriteFile(filepath.Join(data, "server.json"), []byte(`{"url":"`+c.url+`"}`), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := Readiness(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != c.want {
				t.Fatalf("busy = %v, want %d", got, c.want)
			}
		})
	}
}

// The pane running the deploy is always mid-turn — asking is working — so
// the guard drops it before deciding. Every other owner still blocks.
func TestGuardIgnoresTheCallersOwnPane(t *testing.T) {
	self := Busy{Kind: "terminal", ID: "t-self", Name: "matrix", Why: "working"}
	other := Busy{Kind: "terminal", ID: "t-other", Name: "tabs", Why: "working"}
	agent := Busy{Kind: "agent", ID: "a-self", Name: "Pi", Why: "mid-turn"}

	cases := []struct {
		name    string
		env     map[string]string
		busy    []Busy
		wantIDs []string
	}{
		{"no env: nothing is dropped", nil, []Busy{self, other}, []string{"t-self", "t-other"}},
		{"own terminal only", map[string]string{termIDEnv: "t-self"}, []Busy{self}, nil},
		{"own terminal among others", map[string]string{termIDEnv: "t-self"}, []Busy{self, other}, []string{"t-other"}},
		{"own agent pane", map[string]string{agentIDEnv: "a-self"}, []Busy{agent, other}, []string{"t-other"}},
		{"terminal id that matches nobody", map[string]string{termIDEnv: "t-gone"}, []Busy{other}, []string{"t-other"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := dropSelf(c.busy, func(k string) string { return c.env[k] })
			var ids []string
			for _, o := range got {
				ids = append(ids, o.ID)
			}
			if len(ids) != len(c.wantIDs) {
				t.Fatalf("kept %v, want %v", ids, c.wantIDs)
			}
			for i, id := range ids {
				if id != c.wantIDs[i] {
					t.Fatalf("kept %v, want %v", ids, c.wantIDs)
				}
			}
		})
	}
}

// End to end through guardDeploy: a fleet whose only busy pane is the
// caller's deploys, and the refusal never names the caller to itself.
func TestGuardDeployWithOnlySelfBusy(t *testing.T) {
	prev := Readiness
	t.Cleanup(func() { Readiness = prev })
	t.Setenv(termIDEnv, "t-self")
	t.Setenv(agentIDEnv, "")

	Readiness = func(string) ([]Busy, error) {
		return []Busy{{Kind: "terminal", ID: "t-self", Name: "matrix", Why: "working"}}, nil
	}
	if err := guardDeploy(t.TempDir(), false); err != nil {
		t.Fatalf("the caller must not block itself: %v", err)
	}

	Readiness = func(string) ([]Busy, error) {
		return []Busy{
			{Kind: "terminal", ID: "t-self", Name: "matrix", Why: "working"},
			{Kind: "terminal", ID: "t-other", Name: "tabs", Why: "working"},
		}, nil
	}
	err := guardDeploy(t.TempDir(), false)
	if !errors.Is(err, ErrDeployBusy) {
		t.Fatalf("got %v, want ErrDeployBusy", err)
	}
	if !strings.Contains(err.Error(), "end 1 turn(s)") || !strings.Contains(err.Error(), `"tabs"`) {
		t.Fatalf("refusal must count and name only the others: %v", err)
	}
	if strings.Contains(err.Error(), `"matrix"`) {
		t.Fatalf("refusal named the caller to itself: %v", err)
	}
}
