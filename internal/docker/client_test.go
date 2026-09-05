package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEndpointPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  map[string]string
		host string
		fail bool
		want string
	}{
		{"override", map[string]string{"PICODE_DOCKER_HOST": "unix:///tmp/own.sock", "DOCKER_CONTEXT": "remote"}, "ssh://remote", false, "unix:///tmp/own.sock"},
		{"context beats host", map[string]string{"DOCKER_CONTEXT": "local", "DOCKER_HOST": "tcp://remote"}, "unix:///tmp/context.sock", false, "unix:///tmp/context.sock"},
		{"env host", map[string]string{"DOCKER_HOST": "unix:///tmp/rootless.sock"}, "", false, "unix:///tmp/rootless.sock"},
		{"current context", nil, "unix:///var/run/docker.sock", false, "unix:///var/run/docker.sock"},
		{"remote", nil, "ssh://remote", false, ""}, {"relative", nil, "unix:relative", false, ""}, {"authority", nil, "unix://remote/tmp/a", false, ""}, {"query", nil, "unix:///tmp/a?x", false, ""},
		{"lookup error", nil, "", true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EndpointFrom(func(k string) string { return tc.env[k] }, func() (string, error) {
				if tc.fail {
					return "", errors.New("lookup failed")
				}
				return tc.host, nil
			})
			if got != tc.want || (err != nil) != (tc.want == "") {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}

func logFrame(stream byte, text string) []byte {
	head := make([]byte, 8)
	head[0] = stream
	binary.BigEndian.PutUint32(head[4:], uint32(len(text)))
	return append(head, []byte(text)...)
}

func TestReadLogs(t *testing.T) {
	for _, tc := range []struct {
		name            string
		data            []byte
		tty             bool
		want            string
		truncated, fail bool
	}{
		{"multiplex", append(logFrame(1, "one\n"), logFrame(2, "two\n")...), false, "one\ntwo\n", false, false},
		{"tty and control codes", []byte("\x1b[31mred\x1b[0m\x00\n<script>x</script>"), true, "red\n<script>x</script>", false, false},
		{"empty", nil, false, "", false, false},
		{"incomplete header", []byte{1, 0}, false, "", false, true},
		{"incomplete frame", append(logFrame(1, "abc")[:8], byte('a')), false, "", false, true},
		{"bounded tty", []byte(strings.Repeat("x", MaxLogBytes+20)), true, strings.Repeat("x", MaxLogBytes), true, false},
		{"bounded frame", logFrame(1, strings.Repeat("x", MaxLogBytes+20)), false, strings.Repeat("x", MaxLogBytes), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, truncated, err := ReadLogs(bytes.NewReader(tc.data), tc.tty)
			if got != tc.want || truncated != tc.truncated || (err != nil) != tc.fail {
				t.Fatalf("text length %d, truncated %v, err %v", len(got), truncated, err)
			}
		})
	}
}

func socketClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	// Do not include t.Name(): macOS Unix socket paths have a 104-byte limit,
	// and its temporary directory prefix already consumes much of that space.
	dir, err := os.MkdirTemp("", "pd-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "engine.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: handler}
	go srv.Serve(ln)
	t.Cleanup(func() { _ = srv.Close() })
	c, err := NewClient("unix://" + socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestSocketClientWithLongTestName(t *testing.T) {
	t.Run(strings.Repeat("long-name-", 20), func(t *testing.T) {
		c := socketClient(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `{"ApiVersion":"1.54","MinAPIVersion":"1.44"}`)
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := c.Check(ctx); err != nil {
			t.Fatal(err)
		}
	})
}

func TestVersionAndTransport(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
		fail       bool
	}{
		{"supported", `{"ApiVersion":"1.54","MinAPIVersion":"1.44"}`, 200, false},
		{"too old", `{"ApiVersion":"1.43"}`, 200, true},
		{"future minimum", `{"ApiVersion":"1.55","MinAPIVersion":"1.50"}`, 200, true},
		{"malformed", `not json`, 200, true},
		{"access denied", `{"message":"permission denied"}`, 403, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := socketClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/version" {
					t.Errorf("path %s", r.URL.Path)
				}
				w.WriteHeader(tc.code)
				_, _ = io.WriteString(w, tc.body)
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := c.Check(ctx); (err != nil) != tc.fail {
				t.Fatalf("Check = %v", err)
			}
		})
	}
	c, err := NewClient("unix://" + filepath.Join(t.TempDir(), "absent.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err = c.Check(context.Background()); err == nil {
		t.Fatal("missing daemon succeeded")
	}
}

func TestResourceSample(t *testing.T) {
	c := socketClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"cpu_stats":{"cpu_usage":{"total_usage":200},"system_cpu_usage":1000,"online_cpus":2},"precpu_stats":{"cpu_usage":{"total_usage":100},"system_cpu_usage":500},"memory_stats":{"usage":1000,"limit":4000,"stats":{"inactive_file":100}}}`)
	})
	s, err := c.Stats(context.Background(), strings.Repeat("a", 64))
	if err != nil || s.CPUPercent != 40 || s.MemoryBytes != 900 || s.LimitBytes != 4000 || s.SampledAt == "" {
		t.Fatalf("stats %+v, %v", s, err)
	}
}
