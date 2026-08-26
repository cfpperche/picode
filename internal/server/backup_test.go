package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/backup"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func TestBackupAPI(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "data")
	dest := filepath.Join(root, "backups")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(data, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	eng := &backup.Engine{Store: st, DataDir: data, PiDir: filepath.Join(root, "pi"), Version: "test"}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil),
		AgentCmd: "cat", DataDir: data, Backup: eng,
	}).Handler)
	t.Cleanup(ts.Close)

	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/backup"))
	var s backup.Settings
	if err := json.NewDecoder(got.Body).Decode(&s); err != nil || s.Enabled {
		t.Fatalf("default = %+v %v", s, err)
	}

	body, _ := json.Marshal(map[string]any{
		"dir": dest, "intervalMin": 60, "keepDays": 10, "sessions": true, "secrets": true,
	})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/backup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	put := do(t, ts.Client(), req)
	if put.StatusCode != http.StatusOK {
		t.Fatalf("put = %d", put.StatusCode)
	}

	now := do(t, ts.Client(), mustPost(t, ts.URL+"/api/backup/now"))
	if now.StatusCode != http.StatusOK {
		t.Fatalf("now = %d", now.StatusCode)
	}
	list := do(t, ts.Client(), mustGet(t, ts.URL+"/api/backup/snapshots"))
	var wrap struct {
		Snapshots []backup.Snapshot `json:"snapshots"`
	}
	_ = json.NewDecoder(list.Body).Decode(&wrap)
	if len(wrap.Snapshots) != 1 {
		t.Fatalf("list = %+v", wrap)
	}

	bad, _ := json.Marshal(map[string]any{"dir": filepath.Join(data, "x"), "intervalMin": 60, "keepDays": 10, "sessions": true, "secrets": true})
	req2, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/backup", bytes.NewReader(bad))
	req2.Header.Set("Content-Type", "application/json")
	if res := do(t, ts.Client(), req2); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("inside dest = %d", res.StatusCode)
	}
}
