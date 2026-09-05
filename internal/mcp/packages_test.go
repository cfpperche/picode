package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConnectorPackageDiscovery(t *testing.T) {
	home := t.TempDir()
	user := filepath.Join(home, ".pi", "agent")
	pkg := filepath.Join(t.TempDir(), "connector")
	for _, dir := range []string{user, pkg} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	settings, _ := json.Marshal(map[string]any{"packages": []string{pkg}})
	if err := os.WriteFile(filepath.Join(user, "settings.json"), settings, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, manifest string
		count          int
	}{
		{"single", `{"name":"vendor-connector","pi":{"mcp":"mcp.json"}}`, 1},
		{"multiple", `{"name":"vendor-connector","pi":{"mcp":["mcp.json","extra.json"]}}`, 1},
		{"ordinary package", `{"name":"ordinary","pi":{"extensions":["index.ts"]}}`, 0},
		{"bad metadata", `{"name":"bad","pi":{"mcp":42}}`, 0},
		{"empty", `{"name":"empty","pi":{"mcp":[]}}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(pkg, "package.json"), []byte(tc.manifest), 0600); err != nil {
				t.Fatal(err)
			}
			rows := connectorPackages(Paths{Home: home})
			if len(rows) != tc.count {
				t.Fatalf("rows: %#v", rows)
			}
			if len(rows) > 0 && (rows[0].Source != pkg || rows[0].Scope != "user") {
				t.Fatalf("provenance: %#v", rows)
			}
		})
	}
	// Removing native package settings removes the connector: no parallel registry.
	if err := os.WriteFile(filepath.Join(user, "settings.json"), []byte(`{"packages":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if rows := connectorPackages(Paths{Home: home}); len(rows) != 0 {
		t.Fatal(rows)
	}
}
