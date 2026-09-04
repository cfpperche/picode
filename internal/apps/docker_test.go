package apps

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/docker"
	"github.com/cfpperche/picode/internal/store"
)

type dockerTransport func(*http.Request) (*http.Response, error)

func (f dockerTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDockerAppStates(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
		want       string
	}{
		{"empty", `[]`, 200, "No containers"},
		{"blocked", `{"message":"Docker access denied"}`, 403, "Docker access denied"},
		{"populated", `[{"Id":"` + strings.Repeat("a", 64) + `","Names":["/example"],"Image":"example:1","State":"running","Labels":{"com.docker.compose.project":"website"}}]`, 200, "example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, err := store.Open(filepath.Join(t.TempDir(), "db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			c := &docker.Client{Endpoint: "unix:///tmp/test.sock", HTTP: &http.Client{Transport: dockerTransport(func(r *http.Request) (*http.Response, error) {
				body, code := tc.body, tc.code
				if r.URL.Path == "/version" {
					body, code = `{"ApiVersion":"1.44"}`, 200
				}
				return &http.Response{StatusCode: code, Status: http.StatusText(code), Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}}
			s, err := docker.NewService(context.Background(), st, func(context.Context) (*docker.Client, error) { return c, nil }, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			v, err := (dockerApp{}).View(context.Background(), Host{Store: st, Docker: s}, "")
			if err != nil {
				t.Fatal(err)
			}
			if err = v.Validate(); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, b := range v.Blocks {
				if strings.Contains(b.Empty, tc.want) || (b.Text != nil && strings.Contains(*b.Text, tc.want)) {
					found = true
				}
				for _, it := range b.Items {
					if strings.Contains(it.Title, tc.want) {
						found = true
					}
				}
			}
			if !found {
				t.Fatalf("missing %q in %+v", tc.want, v)
			}
		})
	}
}

func TestDockerHistoryWorksWithoutDaemon(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for _, path := range []string{"", "history"} {
		v, err := (dockerApp{}).View(context.Background(), Host{Store: st}, path)
		if err != nil {
			t.Fatal(err)
		}
		if err = v.Validate(); err != nil {
			t.Fatal(err)
		}
		if len(v.Blocks) == 0 {
			t.Fatal("blank view")
		}
	}
}
