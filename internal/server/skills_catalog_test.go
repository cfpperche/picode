package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/skillcatalog"
)

// The Marketplace routes: sources are added and removed (the seeds stay, a
// duplicate or a folder on disk is refused), a new source is read in the
// background and appears in the catalog, and skills.sh answers only when
// switched on — its failure is one line beside the cards, never an error.
func TestSkillCatalogRoutes(t *testing.T) {
	mux := http.NewServeMux()
	var shDown atomic.Bool
	mux.HandleFunc("/api/repos/acme/skills", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"default_branch":"main"}`))
	})
	mux.HandleFunc("/api/repos/acme/skills/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tree":[{"path":"skills/pdf/SKILL.md","type":"blob"}]}`))
	})
	mux.HandleFunc("/raw/acme/skills/main/skills/pdf/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("---\nname: pdf\ndescription: PDF tools\n---\n"))
	})
	mux.HandleFunc("/sh/api/search", func(w http.ResponseWriter, r *http.Request) {
		if shDown.Load() {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"skills":[{"id":"x/y/pdfx","source":"x/y","skillId":"pdfx","name":"pdfx","installs":7}]}`))
	})
	fake := httptest.NewServer(mux)
	t.Cleanup(fake.Close)
	prev := newSkillCatalog
	newSkillCatalog = func(dir string) *skillcatalog.Store {
		s := prev(dir)
		s.API, s.Raw, s.SkillsSH, s.HTTP = fake.URL+"/api", fake.URL+"/raw", fake.URL+"/sh", fake.Client()
		return s
	}
	t.Cleanup(func() { newSkillCatalog = prev })

	st := testStore(t)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat", DataDir: t.TempDir()}).Handler)
	t.Cleanup(ts.Close)
	type catalog struct {
		Items         []skillcatalog.Item        `json:"items"`
		Sources       []skillcatalog.SourceState `json:"sources"`
		SkillsSH      bool                       `json:"skillssh"`
		SkillsSHError string                     `json:"skillsshError"`
	}
	get := func(q string) catalog {
		t.Helper()
		res, err := http.Get(ts.URL + "/api/skills/catalog?q=" + q)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var c catalog
		_ = json.NewDecoder(res.Body).Decode(&c)
		return c
	}
	settled := func(q string) catalog {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for {
			c := get(q)
			reading := false
			for _, s := range c.Sources {
				reading = reading || s.Reading
			}
			if !reading || time.Now().After(deadline) {
				return c
			}
			time.Sleep(20 * time.Millisecond)
		}
	}

	if code, _ := skillsCall(t, "POST", ts.URL+"/api/skills/sources", map[string]string{"input": "https://github.com/acme/skills"}); code != 200 {
		t.Fatalf("add: %d", code)
	}
	for in, want := range map[string]int{"acme/skills": http.StatusConflict, "anthropics/skills": http.StatusConflict, "/tmp/x": http.StatusBadRequest} {
		if code, _ := skillsCall(t, "POST", ts.URL+"/api/skills/sources", map[string]string{"input": in}); code != want {
			t.Errorf("%s: %d, want %d", in, code, want)
		}
	}
	c := settled("pdf")
	if len(c.Items) != 1 || c.Items[0].Origin != "yours" || c.Items[0].Install != "acme/skills/skills/pdf" || c.SkillsSH {
		t.Fatalf("%+v", c)
	}
	var seedErr string
	for _, s := range c.Sources {
		if s.Builtin && s.Error != "" {
			seedErr = s.Error
		}
	}
	if seedErr == "" {
		t.Fatalf("a seed that could not be read says nothing: %+v", c.Sources)
	}

	if code, _ := skillsCall(t, "PUT", ts.URL+"/api/skills/skillssh", map[string]bool{"on": true}); code != 200 {
		t.Fatal("switch")
	}
	c = get("pdf")
	if !c.SkillsSH || len(c.Items) != 2 || c.Items[1].Origin != "skills.sh" {
		t.Fatalf("%+v", c)
	}
	shDown.Store(true)
	if c = get("pdf"); !strings.Contains(c.SkillsSHError, "skills.sh did not answer") || len(c.Items) != 1 {
		t.Fatalf("%+v", c)
	}

	if code, _ := skillsCall(t, "DELETE", ts.URL+"/api/skills/sources", map[string]string{"input": "anthropics/skills"}); code != http.StatusBadRequest {
		t.Fatalf("seed removed: %d", code)
	}
	if code, _ := skillsCall(t, "DELETE", ts.URL+"/api/skills/sources", map[string]string{"input": "acme/skills"}); code != 200 {
		t.Fatalf("remove: %d", code)
	}
	if c = settled(""); len(c.Sources) != len(skillcatalog.Seeds) {
		t.Fatalf("%+v", c.Sources)
	}
}
