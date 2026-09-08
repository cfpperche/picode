package server

import (
	"net/http"
	"testing"
	"time"
)

func TestPinReminderRoutes(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	p := newPin(t, ts, "Standup notes")
	id := p["id"].(string)
	code, out := pinReqJSON(t, ts, http.MethodPut, "/api/pins/"+id+"/reminder", map[string]any{"kind": "interval", "intervalMin": 1, "tz": "UTC"}, nil)
	if code != http.StatusBadRequest || out["error"] == nil {
		t.Fatalf("1 min = %d %v", code, out)
	}
	code, out = pinReqJSON(t, ts, http.MethodPut, "/api/pins/"+id+"/reminder", map[string]any{"kind": "cron", "cron": "0 9 * * 1-5", "tz": "America/Sao_Paulo"}, nil)
	if code != http.StatusOK || out["label"] != "weekdays at 09:00" || out["nextAt"] == nil {
		t.Fatalf("set = %d %v", code, out)
	}
	var pin map[string]any
	getJSON(t, ts, "/api/pins/"+id, &pin)
	if rem, _ := pin["reminder"].(map[string]any); rem == nil || rem["label"] != "weekdays at 09:00" {
		t.Fatalf("pin.reminder = %v", pin["reminder"])
	}
	at := time.Now().Add(time.Hour).Format(time.RFC3339)
	code, out = pinReqJSON(t, ts, http.MethodPut, "/api/pins/"+id+"/reminder", map[string]any{"kind": "once", "at": at, "tz": "UTC"}, nil)
	if code != http.StatusOK || out["kind"] != "once" {
		t.Fatalf("replace = %d %v", code, out)
	}
	code, _ = pinReqJSON(t, ts, http.MethodPut, "/api/pins/nope-000000/reminder", map[string]any{"kind": "once", "at": at, "tz": "UTC"}, nil)
	if code != http.StatusNotFound {
		t.Fatalf("missing pin = %d", code)
	}
	del(t, ts, "/api/pins/"+id+"/reminder")
	pin = map[string]any{} // Unmarshal merges into a used map; start clean
	getJSON(t, ts, "/api/pins/"+id, &pin)
	if pin["reminder"] != nil {
		t.Fatalf("reminder survived delete: %v", pin["reminder"])
	}
}
