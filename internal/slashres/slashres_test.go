package slashres

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListSkillsAndTemplates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sk := filepath.Join(home, ".pi", "agent", "skills", "brave-search")
	if err := os.MkdirAll(sk, 0o700); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: brave-search\ndescription: Web search\n---\n# hi\n"
	if err := os.WriteFile(filepath.Join(sk, "SKILL.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	pr := filepath.Join(home, ".pi", "agent", "prompts")
	if err := os.MkdirAll(pr, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pr, "review.md"), []byte("---\ndescription: Review diff\n---\nGo.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	items := List("", false)
	var skill, tpl bool
	for _, it := range items {
		if it.Kind == "skill" && it.Name == "brave-search" {
			skill = true
		}
		if it.Kind == "template" && it.Name == "review" {
			tpl = true
		}
	}
	if !skill || !tpl {
		t.Fatalf("%+v", items)
	}
}

func TestProjectOnlyWhenTrusted(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cwd := t.TempDir()
	dir := filepath.Join(cwd, ".pi", "skills", "local")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\ndescription: Local only\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if n := len(List(cwd, false)); n != 0 {
		t.Fatalf("untrusted %d", n)
	}
	items := List(cwd, true)
	if len(items) != 1 || items[0].Name != "local" {
		t.Fatalf("%+v", items)
	}
}
