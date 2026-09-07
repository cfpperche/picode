package install

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
