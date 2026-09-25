package skillcatalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/skills"
)

// fakeGitHub serves one repository's API, tree and raw files.
func fakeGitHub(t *testing.T, files map[string]string, fail *atomic.Bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/repos/acme/skills", func(w http.ResponseWriter, r *http.Request) {
		if fail != nil && fail.Load() {
			http.Error(w, "limited", http.StatusForbidden)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"default_branch": "main"})
	})
	mux.HandleFunc("/api/repos/acme/skills/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
		var tree []map[string]string
		for p := range files {
			tree = append(tree, map[string]string{"path": p, "type": "blob"})
		}
		tree = append(tree, map[string]string{"path": "skills", "type": "tree"})
		_ = json.NewEncoder(w).Encode(map[string]any{"tree": tree})
	})
	mux.HandleFunc("/raw/acme/skills/main/", func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[strings.TrimPrefix(r.URL.Path, "/raw/acme/skills/main/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/sh/api/search", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"skills":[{"id":"anthropics/skills/pdf","source":"anthropics/skills","skillId":"pdf","name":"pdf","installs":200826},{"id":"bad","source":"nope","name":"x"}]}`))
	})
	mux.HandleFunc("/.well-known/agent-skills/index.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"skills":[{"name":"site-skill","description":"From  a\nsite"}]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func md(name, desc string) string {
	return "---\nname: " + name + "\ndescription: " + desc + "\nlicense: MIT\n---\nbody\n"
}

func testStore(t *testing.T, srv *httptest.Server, dir string) *Store {
	t.Helper()
	s := NewStore(dir)
	s.API, s.Raw, s.SkillsSH, s.HTTP = srv.URL+"/api", srv.URL+"/raw", srv.URL+"/sh", srv.Client()
	return s
}

// A repository lists every folder with a SKILL.md — hidden folders included
// (openai/skills keeps its curated set in skills/.curated) — with the header's
// name, description and license, the install input and the page; a folder
// path narrows it; a SKILL.md without a header is left out.
func TestGitHubSource(t *testing.T) {
	srv := fakeGitHub(t, map[string]string{
		"skills/pdf/SKILL.md":           md("pdf", "PDF tools"),
		"skills/.curated/deck/SKILL.md": md("deck", "Slides"),
		"skills/broken/SKILL.md":        "no header",
		"README.md":                     "hi",
	}, nil)
	s := testStore(t, srv, "")
	items, err := s.read(context.Background(), "acme/skills")
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Item{}
	for _, it := range items {
		byName[it.Name] = it
	}
	if len(items) != 2 || byName["pdf"].Install != "acme/skills/skills/pdf" || byName["pdf"].License != "MIT" ||
		byName["deck"].Install != "acme/skills/skills/.curated/deck" || !strings.HasSuffix(byName["pdf"].URL, "/acme/skills/tree/main/skills/pdf") {
		t.Fatalf("%+v", items)
	}
	sub, err := s.read(context.Background(), "acme/skills/skills/.curated")
	if err != nil || len(sub) != 1 || sub[0].Name != "deck" {
		t.Fatalf("%+v %v", sub, err)
	}
}

// Search answers from what is cached, starts the reads a source needs, says
// which sources are still being read, and ranks name matches first.
func TestSearchReadsInTheBackground(t *testing.T) {
	srv := fakeGitHub(t, map[string]string{"skills/pdf/SKILL.md": md("pdf", "PDF tools"), "skills/slides/SKILL.md": md("slides", "makes a pdf deck")}, nil)
	s := testStore(t, srv, "")
	items, states := s.Search("", []string{"acme/skills"})
	if len(items) != 0 {
		t.Fatalf("answered before reading: %+v", items)
	}
	last := states[len(states)-1]
	if last.Input != "acme/skills" || last.Builtin || !last.Reading {
		t.Fatalf("%+v", states)
	}
	if !states[0].Builtin || states[0].Input != Seeds[0] {
		t.Fatalf("seeds first: %+v", states)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		items, states = s.Search("pdf", []string{"acme/skills"})
		if len(items) == 2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(items) != 2 || items[0].Name != "pdf" || items[0].Origin != "yours" {
		t.Fatalf("%+v", items)
	}
	if st := states[len(states)-1]; st.Count != 2 || st.Reading || st.RefreshedAt == "" {
		t.Fatalf("%+v", st)
	}
}

// A source that fails keeps its last good list and says why; the cache on
// disk answers a new store before any read.
func TestFailureKeepsTheLastListAndTheCacheAnswers(t *testing.T) {
	var fail atomic.Bool
	srv := fakeGitHub(t, map[string]string{"skills/pdf/SKILL.md": md("pdf", "PDF tools")}, &fail)
	dir := t.TempDir()
	s := testStore(t, srv, dir)
	if err := s.Refresh(context.Background(), "acme/skills"); err != nil {
		t.Fatal(err)
	}
	fail.Store(true)
	if err := s.Refresh(context.Background(), "acme/skills"); err == nil {
		t.Fatal("a refused read reported success")
	}
	s.mu.Lock()
	c := s.sources["acme/skills"]
	s.mu.Unlock()
	if len(c.Items) != 1 || !strings.Contains(c.Error, "rate limit") {
		t.Fatalf("%+v", c)
	}
	warm := testStore(t, srv, dir)
	warm.mu.Lock()
	got := len(warm.sources["acme/skills"].Items)
	warm.mu.Unlock()
	if got != 1 {
		t.Fatalf("cache answered %d items", got)
	}
}

// skills.sh answers only when asked; its cards carry the source, the count
// and the page, never a description PiCode made up.
func TestSkillsSH(t *testing.T) {
	srv := fakeGitHub(t, nil, nil)
	s := testStore(t, srv, "")
	items, err := s.SearchSkillsSH(context.Background(), "pdf")
	if err != nil || len(items) != 1 {
		t.Fatalf("%+v %v", items, err)
	}
	it := items[0]
	if it.Origin != "skills.sh" || it.Install != "anthropics/skills" || it.Installs != 200826 || it.Description != "" || it.URL != "https://skills.sh/anthropics/skills/pdf" {
		t.Fatalf("%+v", it)
	}
	if none, err := s.SearchSkillsSH(context.Background(), "p"); err != nil || none != nil {
		t.Fatal("a one-letter query went out")
	}
	s.SkillsSH = srv.URL + "/missing"
	if _, err := s.SearchSkillsSH(context.Background(), "pdf"); err == nil || !strings.Contains(err.Error(), "skills.sh did not answer") {
		t.Fatalf("%v", err)
	}
}

func TestWellKnownIndex(t *testing.T) {
	srv := fakeGitHub(t, nil, nil)
	s := testStore(t, srv, "")
	items, err := s.readWellKnown(context.Background(), wellKnownSource(srv.URL))
	if err != nil || len(items) != 1 || items[0].Description != "From a site" || items[0].Install != srv.URL {
		t.Fatalf("%+v %v", items, err)
	}
}

func TestValidateSource(t *testing.T) {
	for in, want := range map[string]string{
		"acme/skills": "acme/skills",
		"https://github.com/acme/skills/tree/main/x": "acme/skills/x#main",
		"https://example.com":                        "https://example.com",
	} {
		got, err := ValidateSource(in, t.TempDir())
		if err != nil || got != want {
			t.Errorf("%s: %q %v, want %q", in, got, err, want)
		}
	}
	if _, err := ValidateSource("/tmp/skills", "/home/x"); err == nil {
		t.Error("a local folder became a marketplace source")
	}
}

func wellKnownSource(origin string) skills.Source {
	return skills.Source{Kind: "well-known", Origin: origin}
}
