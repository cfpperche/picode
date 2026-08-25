package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTwoAPIKeysSwitch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if err := PutAPIKey("xai", "sk-two"); err != nil {
		t.Fatal(err)
	}
	acc := AccountsFor("xai")
	if len(acc) != 2 {
		t.Fatalf("got %d accounts: %+v", len(acc), acc)
	}
	active := 0
	var firstID string
	for _, a := range acc {
		if a.Active {
			active++
			if a.Type != "api_key" {
				t.Fatalf("type %s", a.Type)
			}
		} else {
			firstID = a.ID
		}
	}
	if active != 1 {
		t.Fatalf("active %d", active)
	}
	if key := mustKey(t, home, "xai"); key != "sk-two" {
		t.Fatalf("auth.json %s", key)
	}
	if firstID == "" {
		t.Fatal("missing inactive")
	}
	if err := ActivateAccount("xai", firstID); err != nil {
		t.Fatal(err)
	}
	if key := mustKey(t, home, "xai"); key != "sk-one" {
		t.Fatalf("after switch %s", key)
	}
}

func TestRemoveActivePromotesOther(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if err := PutAPIKey("xai", "sk-two"); err != nil {
		t.Fatal(err)
	}
	var drop string
	for _, a := range AccountsFor("xai") {
		if a.Active {
			drop = a.ID
		}
	}
	if err := RemoveAccount("xai", drop); err != nil {
		t.Fatal(err)
	}
	acc := AccountsFor("xai")
	if len(acc) != 1 || !acc[0].Active {
		t.Fatalf("%+v", acc)
	}
	if key := mustKey(t, home, "xai"); key != "sk-one" {
		t.Fatalf("promoted %s", key)
	}
}

func TestRemoveLastClearsAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	id := AccountsFor("xai")[0].ID
	if err := RemoveAccount("xai", id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".pi", "agent", "auth.json")); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".pi", "agent", "auth.json"))
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	if _, ok := m["xai"]; ok {
		t.Fatal("xai still in auth.json")
	}
	if n := len(AccountsFor("xai")); n != 0 {
		t.Fatalf("vault %d", n)
	}
}

func TestSameKeyDoesNotDuplicate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if n := len(AccountsFor("xai")); n != 1 {
		t.Fatalf("got %d", n)
	}
}

func mustKey(t *testing.T, home, provider string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m[provider]["key"]
}
