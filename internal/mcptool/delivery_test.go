package mcptool

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// TestDeliveryPiPackageMatchesFamily is the seam between the two faces of one
// contract (ADR-0171): the pi package a Pi agent loads and the MCP family a
// guest CLI is launched with must offer the same actions and parameters.
// computer_test.go holds the same seam for its package.
func TestDeliveryPiPackageMatchesFamily(t *testing.T) {
	schema := deliveryFamily.Tools(&Caller{})[0].InputSchema
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		t.Fatal("the family declares no properties")
	}
	required := map[string]bool{}
	want, _ := schema["required"].([]string)
	for _, name := range want {
		required[name] = true
	}
	src, err := os.ReadFile(filepath.Join("..", "..", "packages", "pi-delivery", "extensions", "delivery.ts"))
	if err != nil {
		t.Skip("pi-delivery source not beside this package")
	}
	for name := range props {
		needle := name + ": Type.Optional("
		if required[name] {
			needle = name + ": Type."
		}
		if !strings.Contains(string(src), needle) {
			t.Errorf("pi-delivery no longer declares %s — the MCP schema must follow", name)
		}
	}
	// The actions are one list in two files: the enum here, ACTIONS in the
	// package's logic. A drift means one face offers what the other refuses.
	enum, _ := props["action"].(map[string]any)["enum"].([]string)
	logic, err := os.ReadFile(filepath.Join("..", "..", "packages", "pi-delivery", "src", "logic.ts"))
	if err != nil {
		t.Fatal(err)
	}
	block := regexp.MustCompile(`(?s)ACTIONS = \[(.*?)\]`).FindSubmatch(logic)
	if block == nil {
		t.Fatal("ACTIONS not found in packages/pi-delivery/src/logic.ts")
	}
	var js []string
	for _, part := range regexp.MustCompile(`"([a-z-]+)"`).FindAllStringSubmatch(string(block[1]), -1) {
		js = append(js, part[1])
	}
	if strings.Join(enum, ",") != strings.Join(js, ",") {
		t.Fatalf("the two faces disagree:\n  mcp: %v\n  pi:  %v", enum, js)
	}
}
