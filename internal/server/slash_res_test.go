package server

import (
	"encoding/json"
	"testing"
)

func TestExtensionCommandsFromGetCommands(t *testing.T) {
	raw := []byte(`{
		"commands": [
			{"name": "roles", "description": "Pick a role", "source": "extension"},
			{"name": "fix-tests", "description": "Fix tests", "source": "prompt"},
			{"name": "skill:web", "description": "Search", "source": "skill"},
			{"name": "roles", "description": "dup", "source": "extension"},
			{"name": "", "source": "extension"},
			{"name": "vision", "source": "extension"}
		]
	}`)
	got := extensionCommandsFromGetCommands(raw)
	if len(got) != 2 {
		t.Fatalf("got %d items: %+v", len(got), got)
	}
	if got[0].Name != "roles" || got[0].Hint != "Pick a role" || got[0].Kind != "extension" {
		t.Fatalf("roles row: %+v", got[0])
	}
	if got[1].Name != "vision" || got[1].Hint != "Command" {
		t.Fatalf("vision row: %+v", got[1])
	}
}

func TestExtensionCommandsFromGetCommandsEmpty(t *testing.T) {
	if len(extensionCommandsFromGetCommands(nil)) != 0 {
		t.Fatal("nil")
	}
	if len(extensionCommandsFromGetCommands(json.RawMessage(`{`))) != 0 {
		t.Fatal("bad json")
	}
}
