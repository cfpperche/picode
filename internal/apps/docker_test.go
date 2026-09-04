package apps

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
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

func TestDockerGroupsUseProjectIdentity(t *testing.T) {
	for _, tc := range []struct {
		name       string
		containers []docker.Container
		titles     []string
		counts     []int
	}{
		{"empty", nil, []string{}, []int{}},
		{"same project with unrelated names", []docker.Container{{ID: "a", Name: "database", Project: "website"}, {ID: "b", Name: "other-name", Project: "website"}}, []string{"website"}, []int{2}},
		{"similar names are not a grouping rule", []docker.Container{{ID: "a", Name: "web-1", Project: "beta"}, {ID: "b", Name: "web-2", Project: "alpha"}}, []string{"alpha", "beta"}, []int{1, 1}},
		{"unlabeled containers stay standalone", []docker.Container{{ID: "a", Name: "web-1"}, {ID: "b", Name: "web-2"}}, []string{"Standalone containers"}, []int{2}},
		{"standalone last, projects sorted", []docker.Container{{ID: "a", Name: "one"}, {ID: "b", Name: "two", Project: "zulu"}, {ID: "c", Name: "three", Project: "alpha"}}, []string{"alpha", "zulu", "Standalone containers"}, []int{1, 1, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blocks := dockerContainerGroups(docker.Inventory{Endpoint: "unix:///tmp/one.sock", Containers: tc.containers}, nil)
			if err := (View{APIVersion: APIVersion, Blocks: blocks}).Validate(); err != nil {
				t.Fatal(err)
			}
			titles, counts := []string{}, []int{}
			for _, block := range blocks {
				titles = append(titles, block.Title)
				counts = append(counts, len(block.Items))
				if !block.Collapsible || block.ID == "" {
					t.Fatal("group lacks stable identity/disclosure")
				}
			}
			if !reflect.DeepEqual(titles, tc.titles) || !reflect.DeepEqual(counts, tc.counts) {
				t.Fatalf("groups %v %v", titles, counts)
			}
		})
	}
}

func TestDockerGroupSummaryAndStableFolds(t *testing.T) {
	inv := docker.Inventory{Endpoint: "unix:///tmp/one.sock", Containers: []docker.Container{
		{ID: "a", Name: "zulu", Project: "website", State: "running"},
		{ID: "b", Name: "alpha", Project: "website", State: "exited"},
		{ID: "c", Name: "created", Project: "website", State: "created"},
		{ID: "d", Name: "paused", Project: "website", State: "paused"},
		{ID: "e", Name: "retrying", Project: "website", State: "restarting"},
	}}
	b := dockerContainerGroups(inv, nil)[0]
	if !reflect.DeepEqual(b.Meta, []string{"1 running", "1 paused", "1 restarting", "2 stopped"}) {
		t.Fatalf("summary %v", b.Meta)
	}
	if b.Items[0].Title != "alpha" || b.Items[4].Title != "zulu" {
		t.Fatalf("order %+v", b.Items)
	}
	inv.Containers[0], inv.Containers[1] = inv.Containers[1], inv.Containers[0]
	if dockerContainerGroups(inv, nil)[0].ID != b.ID {
		t.Fatal("row reorder changed group identity")
	}
	inv.Endpoint = "unix:///tmp/other.sock"
	if dockerContainerGroups(inv, nil)[0].ID == b.ID {
		t.Fatal("different connections share fold preferences")
	}
}

func TestDockerGroupBusyMatchesTheExactConnection(t *testing.T) {
	inv := docker.Inventory{Endpoint: "unix:///tmp/one.sock", Containers: []docker.Container{{ID: "a", Name: "database", Project: "website", State: "running"}}}
	for _, tc := range []struct {
		endpoint, id, state string
		busy                bool
	}{
		{inv.Endpoint, "a", "running", true}, {inv.Endpoint, "a", "succeeded", false},
		{"unix:///tmp/other.sock", "a", "running", false}, {inv.Endpoint, "other", "running", false},
	} {
		b := dockerContainerGroups(inv, []store.DockerOperation{{Endpoint: tc.endpoint, ContainerID: tc.id, Action: "restart", State: tc.state}})[0]
		if b.Busy != tc.busy || b.Items[0].Busy != tc.busy {
			t.Fatalf("busy for %+v: %+v", tc, b)
		}
		if !reflect.DeepEqual(b.Meta, []string{"1 running"}) {
			t.Fatalf("job state replaced container state: %v", b.Meta)
		}
	}
}
