package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestWorkspaceActivityOnlyExposesScopedSummaries(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	first, err := st.AddWorkspace("First", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	second, err := st.AddWorkspace("Second", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ kind, workspace, state string }{
		{"mission.changed", first.ID, "blocked"},
		{"mission.changed", second.ID, "ready"},
		{"inbox.created", first.ID, "unread"},
		{"inbox.updated", first.ID, "read"},
		{"inbox.updated", first.ID, store.InboxDone},
	} {
		body := map[string]string{"id": row.kind + row.state, "workspaceId": row.workspace, "title": "Example", "state": row.state, "body": "private text"}
		if err := st.AppendEvent(row.kind, nil, nil, body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+first.ID+"/activity", nil)
	req.SetPathValue("id", first.ID)
	w := httptest.NewRecorder()
	handleWorkspaceActivity(Deps{Store: st})(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "private text") {
		t.Fatal("raw event body escaped")
	}
	var got struct {
		Items []workspaceActivityItem `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 3 {
		t.Fatalf("items = %+v", got.Items)
	}
	if got.Items[0].Action != "resolved" || got.Items[1].Action != "created" || got.Items[2].Kind != "mission" {
		t.Fatalf("items = %+v", got.Items)
	}
	req.SetPathValue("id", "missing")
	w = httptest.NewRecorder()
	handleWorkspaceActivity(Deps{Store: st})(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing status %d", w.Code)
	}
}
