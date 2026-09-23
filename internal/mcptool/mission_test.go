package mcptool

import (
	"encoding/json"
	"testing"
)

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
