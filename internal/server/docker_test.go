package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/apps"
	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/docker"
	"github.com/cfpperche/picode/internal/store"
)

type dockerRoundTrip func(*http.Request) (*http.Response, error)

func TestDockerRoutesRequirePairing(t *testing.T) {
	ts, st := newAuthServer(t)
	if err := st.SetSetting(auth.ModeSettingKey, auth.ModeAll); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/docker/containers", "/api/docker/operations", "/api/apps/docker/view"} {
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s: status %d", path, res.StatusCode)
		}
	}
	res, err := http.Post(ts.URL+"/api/docker/operations", "application/json", strings.NewReader(`{"action":"start"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("POST status %d", res.StatusCode)
	}
}

func (f dockerRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDockerRoutesUseSharedService(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	id := strings.Repeat("a", 64)
	c := &docker.Client{Endpoint: "unix:///tmp/test.sock", HTTP: &http.Client{Transport: dockerRoundTrip(func(r *http.Request) (*http.Response, error) {
		body := `[]`
		if r.URL.Path == "/version" {
			body = `{"ApiVersion":"1.44"}`
		} else if strings.HasSuffix(r.URL.Path, "/json") && strings.Contains(r.URL.Path, id) {
			body = `{"Id":"` + id + `","Name":"/qa","State":{"Status":"exited"},"Config":{"Image":"qa"}}`
		} else if strings.HasSuffix(r.URL.Path, "/logs") {
			body = ""
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
	s, err := docker.NewService(context.Background(), st, func(context.Context) (*docker.Client, error) { return c, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ts := httptest.NewServer(New("", Deps{Store: st, Docker: s, Apps: apps.NewRegistry(apps.BuiltIns(false)...)}).Handler)
	defer ts.Close()
	for _, path := range []string{"/api/docker/containers", "/api/docker/containers/" + id, "/api/docker/operations", "/api/apps/docker/view"} {
		var v any
		if code := getJSON(t, ts, path, &v); code != 200 {
			t.Fatalf("%s: %d", path, code)
		}
	}
	for _, body := range []string{`{`, `{"action":"delete","containerId":"` + id + `","requestKey":"request-123"}`, `{"action":"start","containerId":"bad","requestKey":"request-123"}`, `{"action":"start","containerId":"` + id + `","requestKey":"request-123","agentId":"missing"}`, `{"action":"start","containerId":"` + id + `","requestKey":"trailing-json"} {}`} {
		res, err := http.Post(ts.URL+"/api/docker/operations", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != 400 {
			t.Fatalf("invalid payload accepted: %d", res.StatusCode)
		}
	}
	payload, _ := json.Marshal(docker.Request{Action: "start", ContainerID: id, RequestKey: "request-accepted"})
	res, err := http.Post(ts.URL+"/api/docker/operations", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 202 {
		t.Fatalf("operation: %d", res.StatusCode)
	}
	var op store.DockerOperation
	if err = json.NewDecoder(res.Body).Decode(&op); err != nil {
		t.Fatal(err)
	}
	if op.Actor != "Local user" || op.ContainerID != id {
		t.Fatalf("operation %+v", op)
	}
	var record store.DockerOperation
	if code := getJSON(t, ts, "/api/docker/operations/"+op.ID, &record); code != 200 || record.ID != op.ID {
		t.Fatalf("history %d %+v", code, record)
	}
}
