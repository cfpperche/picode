package mcptool

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMissionMutationsRequireVersionedAssignmentFieldsBeforePost(t *testing.T) {
	valid := map[string]any{"action": "report", "id": "m", "requestId": "retry", "expectedVersion": 3, "generation": 1}
	for _, field := range []string{"generation", "expectedVersion", "requestId"} {
		t.Run("missing "+field, func(t *testing.T) {
			payload := make(map[string]any, len(valid))
			for key, value := range valid {
				payload[key] = value
			}
			delete(payload, field)
			args, _ := json.Marshal(payload)
			d := &fakeDaemon{status: 200, body: `{"mission":{"id":"m"}}`}
			_, err := CallMission(t.Context(), &Caller{Daemon: d, Identity: Identity{Agent: "a", Term: "t"}}, args)
			if err == nil || !strings.Contains(err.Error(), field) {
				t.Fatalf("error = %v, want a specific %s error", err, field)
			}
			if d.path != "" {
				t.Fatalf("missing %s reached daemon at %s", field, d.path)
			}
		})
	}
}

func TestMissionReadsNeedNoMutationFields(t *testing.T) {
	for _, action := range []string{"show", "context"} {
		t.Run(action, func(t *testing.T) {
			d := &fakeDaemon{status: 200, body: `{"mission":{"id":"m"}}`}
			args := json.RawMessage(`{"action":"` + action + `","id":"m"}`)
			if _, err := CallMission(t.Context(), &Caller{Daemon: d, Identity: Identity{Agent: "a", Term: "t"}}, args); err != nil {
				t.Fatal(err)
			}
			if d.path != "/api/missions/tool" {
				t.Fatalf("read did not reach daemon: %q", d.path)
			}
		})
	}
}

func TestMissionInheritedIdentityAndRetryPayload(t *testing.T) {
	d := &fakeDaemon{status: 200, body: `{"mission":{"id":"m"}}`}
	c := &Caller{Daemon: d, Identity: Identity{Agent: "actual", Term: "actual-terminal"}}
	_, err := CallMission(t.Context(), c, json.RawMessage(`{"action":"report","id":"m","agent":"forged","term":"forged","requestId":"retry","expectedVersion":3,"generation":2,"note":"checkpoint"}`))
	if err != nil || d.path != "/api/missions/tool" || d.got["agent"] != "actual" || d.got["term"] != "actual-terminal" || d.got["requestId"] != "retry" {
		t.Fatal(err, d.got)
	}
	if _, err = CallMission(t.Context(), &Caller{Daemon: d}, json.RawMessage(`{"action":"show","id":"m"}`)); err == nil {
		t.Fatal("unattributed caller accepted")
	}
	d.status = 409
	d.body = `{"error":"version changed"}`
	if _, err = CallMission(t.Context(), c, json.RawMessage(`{"action":"show","id":"m"}`)); err == nil || err.Error() != "version changed" {
		t.Fatal(err)
	}
}
