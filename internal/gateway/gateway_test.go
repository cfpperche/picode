package gateway

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// backend records what reached it and answers like a daemon would.
type backend struct {
	name  string
	seen  chan *http.Request
	token string
}

func newBackend(t *testing.T, name string) (*backend, *httptest.Server) {
	b := &backend{name: name, seen: make(chan *http.Request, 16), token: "tok-" + name}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cp := r.Clone(context.Background())
		select {
		case b.seen <- cp:
		default:
		}
		switch {
		case r.URL.Path == "/api/auth/pairings" && r.Method == http.MethodPost:
			if r.Header.Get("Authorization") != "Bearer "+b.token {
				http.Error(w, `{"error":"pairing required","pair":true}`, http.StatusUnauthorized)
				return
			}
			fmt.Fprintf(w, `{"code":"code-%s"}`, name)
		case r.URL.Path == "/api/events":
			w.Header().Set("Content-Type", "text/event-stream")
			f := w.(http.Flusher)
			fmt.Fprint(w, "event: hello\ndata: {}\n\n")
			f.Flush()
			time.Sleep(50 * time.Millisecond)
			fmt.Fprint(w, "event: change\ndata: {}\n\n")
			f.Flush()
		case strings.HasPrefix(r.URL.Path, "/ws/"):
			hj, _ := w.(http.Hijacker)
			conn, rw, _ := hj.Hijack()
			defer conn.Close()
			fmt.Fprint(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
			rw.Flush()
			line, _ := rw.ReadString('\n')
			fmt.Fprint(rw, "echo:"+line)
			rw.Flush()
		default:
			fmt.Fprintf(w, "hello from %s, host=%s, xff=%s, auth=%q", name, r.Host, r.Header.Get("X-Forwarded-For"), r.Header.Get("Authorization"))
		}
	}))
	t.Cleanup(ts.Close)
	return b, ts
}

func newGateway(t *testing.T, fake map[string]string, backends map[string]*httptest.Server, tokens map[string]string) (*Server, *httptest.Server) {
	cfg := Config{Hostname: "box.tail1234.ts.net", Users: map[string]string{"alice@example.com": "alice", "Bob@GitHub": "bob"}}
	s := New(cfg)
	s.Fake = fake
	s.Resolve = func(linux string) (Backend, error) {
		ts, ok := backends[linux]
		if !ok {
			return Backend{}, ErrNotRunning{User: linux}
		}
		u, _ := url.Parse(ts.URL)
		return Backend{User: linux, URL: u, Token: tokens[linux]}, nil
	}
	gw := httptest.NewServer(s)
	t.Cleanup(gw.Close)
	return s, gw
}

// peer dials the gateway from a chosen "peer": httptest connections come
// from 127.0.0.1, so the fake identity map keys on that.
func get(t *testing.T, gw *httptest.Server, path string, hdr map[string]string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest("GET", gw.URL+path, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	return res, string(b)
}

func TestRoutesByIdentityAndStripsClaims(t *testing.T) {
	_, a := newBackend(t, "alice")
	_, b := newBackend(t, "bob")
	fake := map[string]string{"127.0.0.1": "alice@example.com"}
	s, gw := newGateway(t, fake, map[string]*httptest.Server{"alice": a, "bob": b}, nil)

	res, body := get(t, gw, "/api/workspaces", map[string]string{"Authorization": "Bearer stolen", "X-Forwarded-For": "1.2.3.4", "X-PiCode-User": "bob", "Cookie": "picode_session=x"})
	if res.StatusCode != 200 || !strings.HasPrefix(body, "hello from alice") {
		t.Fatalf("%d %s", res.StatusCode, body)
	}
	if !strings.Contains(body, "host=box.tail1234.ts.net") || !strings.Contains(body, "xff=127.0.0.1") || !strings.Contains(body, `auth=""`) {
		t.Fatalf("claims leaked or headers wrong: %s", body)
	}
	if res.Header.Get("Strict-Transport-Security") == "" {
		t.Fatal("no HSTS")
	}

	// Same peer, remapped identity → the other daemon, never alice's.
	s.Fake["127.0.0.1"] = "bob@github"
	if _, body := get(t, gw, "/api/workspaces", map[string]string{"Cookie": "picode_session=x"}); !strings.HasPrefix(body, "hello from bob") {
		t.Fatalf("bob got: %s", body)
	}
	// Unknown login → 403; whois failure → 503.
	s.Fake["127.0.0.1"] = "mallory@example.com"
	if res, _ := get(t, gw, "/", nil); res.StatusCode != http.StatusForbidden {
		t.Fatalf("unknown login = %d", res.StatusCode)
	}
	delete(s.Fake, "127.0.0.1")
	if res, _ := get(t, gw, "/", nil); res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("no identity = %d", res.StatusCode)
	}
	// Mapped but not running → 503 with the provision hint.
	s.Fake["127.0.0.1"] = "alice@example.com"
	s.Resolve = func(linux string) (Backend, error) { return Backend{}, ErrNotRunning{User: linux} }
	if res, body := get(t, gw, "/", nil); res.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "provision --user alice") {
		t.Fatalf("not running = %d %s", res.StatusCode, body)
	}
}

func TestFirstVisitAutoPairsOnlyWithoutACookie(t *testing.T) {
	_, a := newBackend(t, "alice")
	_, gw := newGateway(t, map[string]string{"127.0.0.1": "alice@example.com"}, map[string]*httptest.Server{"alice": a}, map[string]string{"alice": "tok-alice"})

	res, _ := get(t, gw, "/", map[string]string{"Sec-Fetch-Mode": "navigate"})
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/pair?code=code-alice" {
		t.Fatalf("no cookie: %d %q", res.StatusCode, res.Header.Get("Location"))
	}
	// Rate limited: a second navigation right away is proxied instead.
	if res, body := get(t, gw, "/", map[string]string{"Sec-Fetch-Mode": "navigate"}); res.StatusCode != 200 || !strings.HasPrefix(body, "hello") {
		t.Fatalf("second mint should not happen within 10s: %d %s", res.StatusCode, body)
	}
	// With a cookie (even a stale one) the request goes through.
	if res, body := get(t, gw, "/", map[string]string{"Sec-Fetch-Mode": "navigate", "Cookie": "picode_session=stale"}); res.StatusCode != 200 || !strings.HasPrefix(body, "hello") {
		t.Fatalf("cookie: %d %s", res.StatusCode, body)
	}
	// API calls (not navigations) are never redirected.
	if res, _ := get(t, gw, "/api/health", map[string]string{"Sec-Fetch-Mode": "cors"}); res.StatusCode != 200 {
		t.Fatalf("api: %d", res.StatusCode)
	}
	// The pairing page itself is proxied, never redirected.
	if res, _ := get(t, gw, "/pair?code=x", map[string]string{"Sec-Fetch-Mode": "navigate"}); res.StatusCode != 200 {
		t.Fatalf("pair page: %d", res.StatusCode)
	}
}

func TestStreamsAndUpgradesPassThrough(t *testing.T) {
	_, a := newBackend(t, "alice")
	_, gw := newGateway(t, map[string]string{"127.0.0.1": "alice@example.com"}, map[string]*httptest.Server{"alice": a}, nil)

	// SSE: the first event must arrive before the stream ends.
	req, _ := http.NewRequest("GET", gw.URL+"/api/events", nil)
	req.Header.Set("Cookie", "picode_session=x")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	br := bufio.NewReader(res.Body)
	start := time.Now()
	line, _ := br.ReadString('\n')
	if !strings.HasPrefix(line, "event: hello") || time.Since(start) > 40*time.Millisecond {
		t.Fatalf("first SSE frame %q after %v", line, time.Since(start))
	}

	// WebSocket-style upgrade: a raw 101 with an echo through the proxy.
	u, _ := url.Parse(gw.URL)
	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprint(conn, "GET /ws/term HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nCookie: picode_session=x\r\n\r\n")
	rd := bufio.NewReader(conn)
	status, _ := rd.ReadString('\n')
	if !strings.Contains(status, "101") {
		t.Fatalf("upgrade status %q", status)
	}
	for { // headers
		l, err := rd.ReadString('\n')
		if err != nil || l == "\r\n" {
			break
		}
	}
	fmt.Fprint(conn, "ping\n")
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	echo, _ := rd.ReadString('\n')
	if echo != "echo:ping\n" {
		t.Fatalf("echo %q", echo)
	}
}

func TestIdentityCachesAndRefusesNonTailnet(t *testing.T) {
	old := whoisCmd
	t.Cleanup(func() { whoisCmd = old })
	calls := 0
	whoisCmd = func(_ context.Context, ip string) ([]byte, error) {
		calls++
		return []byte(`{"UserProfile":{"LoginName":"Alice@Example.com"},"Node":{"Name":"phone.tail1234.ts.net."}}`), nil
	}
	id := NewIdentity(time.Minute)
	login, node, err := id.Whois(context.Background(), "100.64.0.9")
	if err != nil || login != "alice@example.com" || node != "phone.tail1234.ts.net" {
		t.Fatalf("%q %q %v", login, node, err)
	}
	if _, _, _ = id.Whois(context.Background(), "100.64.0.9"); calls != 1 {
		t.Fatalf("calls = %d, want cached", calls)
	}
	if _, _, err := id.Whois(context.Background(), "192.168.1.4"); err == nil || calls != 1 {
		t.Fatal("a LAN address must be refused before whois")
	}
	whoisCmd = func(context.Context, string) ([]byte, error) { return nil, errors.New("boom") }
	if _, _, err := id.Whois(context.Background(), "100.64.0.10"); err == nil {
		t.Fatal("whois failure must surface")
	}
}

func TestConfigRoundTripAndUsers(t *testing.T) {
	old := lookupUser
	t.Cleanup(func() { lookupUser = old })
	lookupUser = func(name string) (*user.User, error) {
		if name == "alice" || name == "bob" {
			return &user.User{Username: name}, nil
		}
		return nil, errors.New("no such user")
	}
	path := filepath.Join(t.TempDir(), "gateway.json")
	if _, err := Load(path); err == nil {
		t.Fatal("missing config must not load as empty")
	}
	c := Default("box.tail1234.ts.net")
	if err := c.AddUser("alice@example.com", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := c.AddUser("carol@example.com", "carol"); err == nil || !strings.Contains(err.Error(), "provision --user carol") {
		t.Fatalf("unknown account: %v", err)
	}
	if err := c.AddUser("x@y", "Bad User"); err == nil {
		t.Fatal("bad linux name accepted")
	}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if u, ok := back.UserFor("ALICE@example.com"); !ok || u != "alice" {
		t.Fatalf("lookup: %q %v", u, ok)
	}
	if back.Listen != ":443" || back.DataDir != "/etc/picode" {
		t.Fatalf("defaults: %+v", back)
	}
	back.RemoveUser("alice@example.com")
	if _, ok := back.UserFor("alice@example.com"); ok {
		t.Fatal("not removed")
	}
	if err := Save(path, Config{Users: map[string]string{}}); err == nil {
		t.Fatal("hostname is required")
	}
	_ = os.Remove(path)
}

func TestResolveReadsServerJSONAndToken(t *testing.T) {
	home := t.TempDir()
	old := homeOf
	t.Cleanup(func() { homeOf = old })
	homeOf = func(string) (string, error) { return home, nil }
	if _, err := Resolve("alice"); !errors.As(err, new(ErrNotRunning)) {
		t.Fatalf("no server.json: %v", err)
	}
	_ = os.MkdirAll(filepath.Join(home, ".picode"), 0o755)
	_ = os.WriteFile(filepath.Join(home, ".picode", "server.json"), []byte(`{"url":"http://localhost:8446/"}`), 0o644)
	_ = os.WriteFile(filepath.Join(home, ".picode", "token"), []byte("abc\n"), 0o600)
	be, err := Resolve("alice")
	if err != nil || be.URL.String() != "http://localhost:8446" || be.Token != "abc" {
		t.Fatalf("%+v %v", be, err)
	}
}
