package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// workspaceActivityItem exposes a small, safe summary of a durable event.
// Raw event bodies may contain agent settings or Inbox text and never leave
// the server through this route.
type workspaceActivityItem struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	EntityID  string `json:"entityId"`
	Title     string `json:"title"`
	Action    string `json:"action,omitempty"`
	State     string `json:"state,omitempty"`
	CreatedAt string `json:"createdAt"`
}

func activityItem(ev store.Event) (workspaceActivityItem, bool) {
	var data struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Title  string `json:"title"`
		Action string `json:"action"`
		State  string `json:"state"`
	}
	if json.Unmarshal(ev.Data, &data) != nil || data.ID == "" {
		return workspaceActivityItem{}, false
	}
	item := workspaceActivityItem{ID: ev.ID, EntityID: data.ID, CreatedAt: ev.CreatedAt}
	switch ev.Type {
	case "mission.changed":
		item.Kind, item.Title, item.Action, item.State = "mission", data.Title, data.Action, data.State
		if item.Title == "" {
			item.Title = "Mission"
		}
	case "inbox.created":
		item.Kind, item.Title, item.Action = "inbox", data.Title, "created"
	case "inbox.updated":
		if data.State != store.InboxDone {
			return workspaceActivityItem{}, false
		}
		item.Kind, item.Title, item.Action = "inbox", data.Title, "resolved"
	case "agent.added":
		item.Kind, item.Title, item.Action = "agent", data.Name, "created"
		if item.Title == "" {
			item.Title = "Agent"
		}
	default:
		return workspaceActivityItem{}, false
	}
	if item.Title == "" {
		item.Title = "Question"
	}
	return item, true
}

func handleWorkspaceActivity(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetWorkspace(id); err != nil {
			writeStoreErr(w, err)
			return
		}
		events, err := deps.Store.WorkspaceOverviewEvents(id, time.Now().Add(-7*24*time.Hour), 50)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		items := make([]workspaceActivityItem, 0, len(events))
		for _, ev := range events {
			if item, ok := activityItem(ev); ok {
				items = append(items, item)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "retentionDays": 7})
	}
}
