package pipkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidSource(t *testing.T) {
	if err := ValidSource("npm:pi-web-search"); err != nil {
		t.Fatal(err)
	}
	if err := ValidSource("git:github.com/badlogic/pi-skills"); err != nil {
		t.Fatal(err)
	}
	if err := ValidSource("npm:foo; rm -rf /"); err == nil {
		t.Fatal("expected reject")
	}
	if err := ValidSource(""); err == nil {
		t.Fatal("expected reject")
	}
}

func TestKindOf(t *testing.T) {
	if KindOf("npm:pi-web-search") != "npm" {
		t.Fatal(KindOf("npm:pi-web-search"))
	}
	if KindOf("https://github.com/user/repo") != "git" {
		t.Fatal("https")
	}
	if KindOf("/abs/path") != "path" {
		t.Fatal("path")
	}
	if KindOf("pi-web-search") != "npm" {
		t.Fatal("bare")
	}
}

func TestListParsesSettings(t *testing.T) {
	dir := t.TempDir()
	body := `{
	  "packages": [
	    "npm:pi-web-search",
	    {"source": "git:github.com/badlogic/pi-skills", "skills": ["brave-search"]}
	  ]
	}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := List(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Packages) != 2 {
		t.Fatalf("%+v", rep.Packages)
	}
	if rep.Packages[0].Source != "npm:pi-web-search" || rep.Packages[0].Kind != "npm" {
		t.Fatalf("%+v", rep.Packages[0])
	}
	if !rep.Packages[1].Filtered || !rep.Capabilities.WebSearch {
		t.Fatalf("%+v", rep)
	}
}

func TestListMissingIsEmpty(t *testing.T) {
	rep, err := List(t.TempDir(), "")
	if err != nil || len(rep.Packages) != 0 || rep.Capabilities.WebSearch {
		t.Fatalf("%v %+v", err, rep)
	}
}

func TestDetectWebSearch(t *testing.T) {
	if DetectWebSearch([]Pkg{{Source: "npm:foo"}}) {
		t.Fatal("false positive")
	}
	if !DetectWebSearch([]Pkg{{Source: "npm:pi-web-search"}}) {
		t.Fatal("pi-web-search")
	}
	if !DetectWebSearch([]Pkg{{Source: "git:github.com/x/brave-search"}}) {
		t.Fatal("brave-search")
	}
}
