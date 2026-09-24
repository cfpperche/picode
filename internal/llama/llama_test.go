package llama

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"syscall"
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
	// The margins are wide on purpose: at a 1 ms budget the dial raced the
	// budget itself on a loaded runner, and the failure was classified as the
	// connection rather than the response (measured 2026-09-24, twice on the
	// ubuntu leg). What the test is about is the response timing out.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(200 * time.Millisecond) }))
	c, _ := New(server.URL, "")
	c.http.Timeout = 100 * time.Millisecond
	_, err := c.List()
	if ConnectionFailure(err).Code != "timeout" {
		t.Fatal(err)
	}
	server.Close()
	_, err = c.List()
	if ConnectionFailure(err).Code != "refused" {
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

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

// Each cause reads as itself, so the pane can say what to change.
func TestConnectionFailureNamesTheCause(t *testing.T) {
	wrap := func(err error) error { return &url.Error{Op: "Get", URL: "http://127.0.0.1:8080/models", Err: err} }
	refused := &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}
	cases := []struct {
		err  error
		code string
	}{
		{wrap(timeoutErr{}), "timeout"},
		{wrap(refused), "refused"},
		{wrap(&net.DNSError{Err: "no such host", Name: "llama.lan", IsNotFound: true}), "dns"},
		{wrap(&tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}), "tls"},
		{wrap(tls.RecordHeaderError{Msg: "first record does not look like a TLS handshake"}), "tls"},
		{wrap(context.Canceled), "cancelled"},
		{wrap(errors.New("something else")), "unreachable"},
		{&ConnectionError{"authentication", "The server rejected the API key."}, "authentication"},
	}
	for _, c := range cases {
		got := ConnectionFailure(c.err)
		if got.Code != c.code {
			t.Errorf("%v → %q, want %q", c.err, got.Code, c.code)
		}
		if strings.Contains(got.Message, "127.0.0.1") {
			t.Errorf("message leaks the endpoint: %q", got.Message)
		}
	}
}

func TestStartArgsReadIPv6(t *testing.T) {
	args := strings.Join(startArgs("llama-server", "/m", "http://[::1]:9090"), " ")
	if !strings.Contains(args, "--host ::1") || !strings.Contains(args, "--port 9090") {
		t.Fatalf("args = %s", args)
	}
}
