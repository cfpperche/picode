package pipkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDescriptorForMatchesCatalogByPackageAndSource(t *testing.T) {
	for _, s := range []string{"pi-web-search", "npm:pi-web-search", "/home/x/.pi/agent/npm/node_modules/pi-web-search"} {
		d := DescriptorFor(s, "")
		if d == nil || d.ID != "web-search" {
			t.Fatalf("DescriptorFor(%q) = %v, want the web-search descriptor", s, d)
		}
	}
	if d := DescriptorFor("pi-roles", ""); d != nil {
		t.Fatalf("pi-roles keeps its own adapter, got %v", d)
	}
	if d := DescriptorFor("pi-nowhere", ""); d != nil {
		t.Fatalf("DescriptorFor(pi-nowhere) = %v, want nil", d)
	}
}

func TestDescriptorFromManifestAndRemembered(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"name":"pi-example","picode":{"config":{"id":"example","title":"Example","files":[{"scope":"agent","path":"example.json","format":"json"}],"fields":[{"key":"token","label":"Token","type":"string","required":true}]}}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	d := DescriptorFor("", dir)
	if d == nil || d.ID != "example" {
		t.Fatalf("manifest descriptor = %v, want example", d)
	}
	if d.Match != "pi-example" {
		t.Fatalf("match = %q, want the package name", d.Match)
	}
	if again := DescriptorByID("example"); again == nil || again.Title != "Example" {
		t.Fatalf("DescriptorByID after manifest = %v, want the remembered descriptor", again)
	}
}

func TestDescriptorManifestWithoutEssentialsIsIgnored(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"name":"pi-thin","picode":{"config":{"id":"thin","title":"Thin"}}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	if d := DescriptorFor("", dir); d != nil {
		t.Fatalf("half a description must be no description, got %v", d)
	}
}

func TestValidateDescriptorValues(t *testing.T) {
	d := DescriptorFor("pi-web-search", "")
	if d == nil {
		t.Fatal("web-search descriptor missing")
	}
	if err := ValidateDescriptorValues(d, map[string]any{"provider": "xai", "model": "grok-4.6"}); err != nil {
		t.Fatalf("valid values rejected: %v", err)
	}
	for _, values := range []map[string]any{
		{},
		{"provider": "xai"},
		{"provider": "not-a-kind", "model": "m"},
		{"provider": "xai", "model": 7},
	} {
		if err := ValidateDescriptorValues(d, values); err == nil {
			t.Fatalf("values %v accepted", values)
		}
	}
	// Unknown keys are none of the descriptor's business.
	if err := ValidateDescriptorValues(d, map[string]any{"provider": "xai", "model": "m", "extra": true}); err != nil {
		t.Fatalf("unknown key rejected: %v", err)
	}
}

func TestWriteDescriptorFilePreservesUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "web-search.json")
	if err := os.WriteFile(abs, []byte(`{"provider":"xai","model":"m","keepMe":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteDescriptorFile(abs, map[string]any{"provider": "xai", "model": "grok-4.6"}, map[string]any{"provider": "xai", "model": "m", "keepMe": true}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("written file is not JSON: %v", err)
	}
	if doc["keepMe"] != true || doc["model"] != "grok-4.6" {
		t.Fatalf("written = %v", doc)
	}
	if !strings.HasSuffix(string(b), "\n") {
		t.Fatal("the written file should end with a newline, like every other config here")
	}
}

func TestDescriptorFileAbsResolvesAgentScope(t *testing.T) {
	dir := t.TempDir()
	old := UserDir
	UserDir = func() string { return dir }
	t.Cleanup(func() { UserDir = old })
	d := DescriptorFor("pi-web-search", "")
	abs, err := DescriptorFileAbs(d, "agent")
	if err != nil {
		t.Fatal(err)
	}
	if abs != filepath.Join(dir, "web-search.json") {
		t.Fatalf("abs = %q", abs)
	}
	if _, err := DescriptorFileAbs(d, "workspace"); err == nil {
		t.Fatal("a descriptor without a workspace file must refuse that scope")
	}
}
