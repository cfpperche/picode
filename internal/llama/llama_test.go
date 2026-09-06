package llama

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNormalizeURL(t *testing.T) {
	got, err := NormalizeURL("http://127.0.0.1:8080/v1/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:8080" {
		t.Fatalf("%s", got)
	}
	if _, err := NormalizeURL("file:///etc/passwd"); err == nil {
		t.Fatal("expected error")
	}
	got, err = NormalizeURL("")
	if err != nil || got != DefaultURL {
		t.Fatalf("%s %v", got, err)
	}
}

func TestConnectionResults(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		body, code string
	}{
		{"empty", 200, `{"data":[]}`, "ready"},
		{"auth", 401, `secret`, "authentication"},
		{"forbidden", 403, `secret`, "authentication"},
		{"missing", 404, ``, "unsupported"},
		{"rejected", 400, `secret`, "request_rejected"},
		{"malformed", 200, `oops`, "unsupported"},
		{"single model", 200, `{"data":[{"id":"model"}]}`, "unsupported"},
		{"server", 500, `secret`, "server_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
			defer server.Close()
			c, _ := New(server.URL, "secret")
			_, err := c.List()
			code := "ready"
			if err != nil {
				code = ConnectionFailure(err).Code
			}
			if code != tc.code {
				t.Fatalf("got %s: %v", code, err)
			}
		})
	}
}

func TestConnectionTimeoutAndTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(20 * time.Millisecond) }))
	c, _ := New(server.URL, "")
	c.http.Timeout = time.Millisecond
	_, err := c.List()
	if ConnectionFailure(err).Code != "timeout" {
		t.Fatal(err)
	}
	server.Close()
	_, err = c.List()
	if ConnectionFailure(err).Code != "unreachable" {
		t.Fatal(err)
	}
}

func TestWaitOutcomes(t *testing.T) {
	for _, tc := range []struct {
		want, state string
		pass        bool
	}{
		{"downloaded", "failed", false}, {"downloaded", "unloaded", true},
		{"downloaded", "unknown", false}, {"loaded", "loaded", true},
		{"loaded", "sleeping", true}, {"unloaded", "unloaded", true},
		{"unloaded", "failed", false}, {"unloaded", "loaded", false},
	} {
		t.Run(tc.want+"/"+tc.state, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"data":[{"id":"m","status":{"value":%q}}]}`, tc.state)
			}))
			defer server.Close()
			c, _ := New(server.URL, "")
			err := c.Wait("m", tc.want, time.Millisecond)
			if (err == nil) != tc.pass {
				t.Fatalf("pass=%v err=%v", tc.pass, err)
			}
		})
	}
}
