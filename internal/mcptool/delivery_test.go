package mcptool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type deliveryDaemon struct {
	status   int
	response []byte
	err      error
	body     []byte
	calls    int
}

func (d *deliveryDaemon) Post(_ context.Context, path string, body []byte) (int, []byte, error) {
	if path != "/api/delivery/tool" {
		panic(path)
	}
	d.body = body
	d.calls++
	return d.status, d.response, d.err
}
func (d *deliveryDaemon) Get(context.Context, string) (int, []byte, error) { panic("unexpected GET") }

func TestDeliveryMCPContract(t *testing.T) {
	for _, row := range []struct {
		name     string
		status   int
		response string
		err      error
		fail     bool
	}{
		{"success", 200, `{"schemaVersion":1}`, nil, false},
		{"refused", 409, `{"error":"stale revision"}`, nil, true},
		{"lost response", 0, "", errors.New("lost"), true},
		{"invalid reply", 200, "broken", nil, true},
	} {
		t.Run(row.name, func(t *testing.T) {
			d := &deliveryDaemon{status: row.status, response: []byte(row.response), err: row.err}
			c := &Caller{Identity: Identity{Agent: "inherited", Term: "terminal"}, Daemon: d}
			result := deliveryFamily.Tools(c)[0].Call(context.Background(), json.RawMessage(`{"action":"show","id":"d","agent":"forged","term":"forged"}`))
			if result.IsError != row.fail || d.calls != 1 {
				t.Fatalf("%+v calls %d", result, d.calls)
			}
			var p map[string]any
			_ = json.Unmarshal(d.body, &p)
			if p["agent"] != "inherited" || p["term"] != "terminal" {
				t.Fatal(p)
			}
		})
	}
	d := &deliveryDaemon{}
	if _, e := CallDelivery(context.Background(), &Caller{Daemon: d}, json.RawMessage(`{}`)); e == nil || d.calls != 0 {
		t.Fatal("missing identity dialed daemon")
	}
	for _, body := range []string{"null", "[]", "broken"} {
		if _, e := CallDelivery(context.Background(), &Caller{Daemon: d, Identity: Identity{Term: "t"}}, json.RawMessage(body)); e == nil || d.calls != 0 {
			t.Fatal(body)
		}
	}
}
