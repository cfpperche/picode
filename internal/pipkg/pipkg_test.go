package pipkg

import (
	"os"
	"path/filepath"
	"strings"
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

func TestWithAgent(t *testing.T) {
	rep := Report{Packages: []Pkg{{Source: "npm:user", Scope: "user"}}}
	rep = WithAgent(rep, []string{"npm:pi-web-search", ""})
	if len(rep.Packages) != 2 || rep.Packages[1].Scope != "agent" || !rep.Capabilities.WebSearch {
		t.Fatalf("%+v", rep)
	}
}

func TestListMissingIsEmpty(t *testing.T) {
	rep, err := List(t.TempDir(), "")
	if err != nil || len(rep.Packages) != 0 || rep.Capabilities.WebSearch {
		t.Fatalf("%v %+v", err, rep)
	}
}

func TestMutateArgs(t *testing.T) {
	got := MutateArgs("install", "npm:foo", false)
	want := []string{"install", "npm:foo", "--no-approve"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("%v", got)
	}
	// Project scope: -l and --approve — --no-approve would mark the
	// project untrusted and pi refuses to write its .pi/settings.json.
	got = MutateArgs("install", "npm:foo", true)
	if strings.Join(got, " ") != "install -l npm:foo --approve" {
		t.Fatalf("local: %v", got)
	}
	if strings.Join(MutateArgs("remove", "../pkg", true), " ") != "remove -l ../pkg --approve" {
		t.Fatalf("local remove: %v", MutateArgs("remove", "../pkg", true))
	}
}

func TestListMergesProject(t *testing.T) {
	user := t.TempDir()
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".pi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(user, "settings.json"), []byte(`{"packages":["npm:user-pkg"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, ".pi", "settings.json"), []byte(`{"packages":["npm:proj-pkg"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := List(user, proj)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Packages) != 2 {
		t.Fatalf("%+v", rep.Packages)
	}
	if rep.Packages[0].Scope != "user" || rep.Packages[1].Scope != "project" {
		t.Fatalf("%+v", rep.Packages)
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

func TestAbsPathSource(t *testing.T) {
	base := "/home/u/.pi/agent"
	cases := map[string]string{
		"../../picode/packages/pi-checklist": "/home/u/picode/packages/pi-checklist",
		"/abs/pkg":                           "/abs/pkg",
		"./local/pkg":                        "/home/u/.pi/agent/local/pkg",
		"npm:pi-web-search":                  "npm:pi-web-search",
		"git:github.com/x/y":                 "git:github.com/x/y",
	}
	for in, want := range cases {
		if got := AbsPathSource(in, base); got != want {
			t.Errorf("AbsPathSource(%q) = %q, want %q", in, got, want)
		}
	}
	if got := AbsPathSource("../x", ""); got != "../x" {
		t.Errorf("no base = %q", got)
	}
}

func TestListResolvesRelativePathInstall(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "picode", "packages", "pi-checklist")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	user := filepath.Join(root, ".pi", "agent")
	if err := os.MkdirAll(user, 0o755); err != nil {
		t.Fatal(err)
	}
	// pi writes the path relative to the settings directory.
	if err := os.WriteFile(filepath.Join(user, "settings.json"), []byte(`{"packages":["../../picode/packages/pi-checklist"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := List(user, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Packages) != 1 || rep.Packages[0].InstalledPath != pkg {
		t.Fatalf("packages = %+v, want InstalledPath %s", rep.Packages, pkg)
	}
}
