package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Dev-server discovery: the parsers that read the machine, the decision table
// that names a port, and the list the panel consumes. Nothing here touches a
// real socket or /proc — the seams are stubbed, so the assertions are about
// the decisions, not the environment.

func TestProcNetTCPParse(t *testing.T) {
	table := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1435 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 111111 1 0000000000000000 100 0 0 10 0
   1: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 222222 1 0000000000000000 100 0 0 10 0
   2: 0100007F:1435 0A00000A:0050 01 00000000:00000000 00:00000000 00000000  1000        0 333333 1 0000000000000000 100 0 0 10 0
   3: 0A00000A:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 444444 1 0000000000000000 100 0 0 10 0
   4: 00000000:2328 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 555555 1 0000000000000000 100 0 0 10 0
   5: 0100007F:ZZZZ 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 666666 1 0000000000000000 100 0 0 10 0
   6: short row`
	got := parseProcNetTCP(table)
	// 0x1435 = 5173, 0x1F90 = 8080, 0x2328 = 9000.
	want := map[int]string{5173: "111111", 8080: "222222", 9000: "555555"}
	if len(got) != len(want) {
		t.Fatalf("ports=%v want %v", got, want)
	}
	for port, inode := range want {
		if got[port] != inode {
			t.Fatalf("port %d inode=%q want %q", port, got[port], inode)
		}
	}
	// An established row on the same port must not duplicate or replace it.
	if got[5173] != "111111" {
		t.Fatalf("established row overrode the listener: %v", got)
	}
}

func TestDevServerBindTable(t *testing.T) {
	cases := []struct {
		addr string
		ok   bool
	}{
		{"0100007F", true},                         // 127.0.0.1
		{"00000000", true},                         // 0.0.0.0
		{"0A00000A", false},                        // 10.0.0.10, the LAN
		{"0101A8C0", false},                        // 192.168.1.1
		{"00000000000000000000000001000000", true}, // ::1
		{"00000000000000000000000000000000", true}, // ::
	}
	for _, c := range cases {
		if got := loopbackBind(c.addr); got != c.ok {
			t.Fatalf("loopbackBind(%q)=%v want %v", c.addr, got, c.ok)
		}
	}
}

func TestDevServerPageFacts(t *testing.T) {
	cases := []struct {
		body      string
		title     string
		generator string
	}{
		{"<html><head><title>  Vite &amp; React </title></head></html>", "Vite & React", ""},
		{"<TITLE>Next</TITLE>", "Next", ""},
		{"<meta name=\"generator\" content=\"Astro v4\">", "", "Astro v4"},
		{"<meta name='generator' content='Remix'>", "", "Remix"},
		{"<html><body>no head at all</body></html>", "", ""},
		{"<title>" + strings.Repeat("x", 200) + "</title>", strings.Repeat("x", devServerTitleCap) + "…", ""},
	}
	for _, c := range cases {
		if got := htmlTitle(c.body); got != c.title {
			t.Fatalf("htmlTitle(%q)=%q want %q", c.body, got, c.title)
		}
		if got := generatorTool(c.body); got != c.generator {
			t.Fatalf("generatorTool(%q)=%q want %q", c.body, got, c.generator)
		}
	}
}

func TestDevServerToolNames(t *testing.T) {
	cases := []struct {
		argv []string
		want string
	}{
		{[]string{"node", "/w/node_modules/.bin/vite"}, "vite"},
		{[]string{"python3", "dev.py"}, "dev"},
		{[]string{"uvicorn", "app:app", "--reload"}, "uvicorn"},
		{[]string{"next-server (v14)"}, "next-server (v14)"},
		{[]string{"/usr/local/bin/astro", "dev"}, "astro"},
		{[]string{"sh", "-c", "npm run dev"}, ""},
		{[]string{"bash"}, ""},
		{[]string{}, ""},
	}
	for _, c := range cases {
		if got := devServerTool(c.argv); got != c.want {
			t.Fatalf("devServerTool(%v)=%q want %q", c.argv, got, c.want)
		}
	}
}

// TestJudgeDevServerAnswer is the decision table the panel's row verbs rest
// on: what a port answered decides whether it is a page you can open, or an
// API that only deserves a copy/stop/hide. Every row is a condition the
// panel's first version got wrong in one place — a control channel that
// answered 400 and was offered as a page.
func TestJudgeDevServerAnswer(t *testing.T) {
	cases := []struct {
		name         string
		answer       devServerAnswer
		wantKind     string
		wantTitle    string
		wantTool     string
		wantMismatch bool
	}{
		{
			name:      "html with a title is a page",
			answer:    devServerAnswer{scheme: "http", status: 200, contentType: "text/html; charset=utf-8", body: "<title>Vite + React</title>"},
			wantKind:  devServerKindPage,
			wantTitle: "Vite + React",
		},
		{
			name:     "html without a title is still a page",
			answer:   devServerAnswer{scheme: "http", status: 200, contentType: "text/html", body: "<div id=root></div>"},
			wantKind: devServerKindPage,
		},
		{
			name:     "an undeclared document with a title is a page",
			answer:   devServerAnswer{scheme: "http", status: 200, body: "<html><title>Dashboard</title></html>"},
			wantKind: devServerKindPage, wantTitle: "Dashboard",
		},
		{
			name:     "json is an api",
			answer:   devServerAnswer{scheme: "http", status: 200, contentType: "application/json", body: `{"ok":true}`},
			wantKind: devServerKindAPI,
		},
		{
			name:     "an html 404 is still something the tab can show",
			answer:   devServerAnswer{scheme: "http", status: 404, contentType: "text/html", body: "<title>Not found</title>"},
			wantKind: devServerKindPage, wantTitle: "Not found",
		},
		{
			name:     "a plain 404 is an api",
			answer:   devServerAnswer{scheme: "http", status: 404, contentType: "text/plain; charset=utf-8", body: "404 page not found"},
			wantKind: devServerKindAPI,
		},
		{
			name:     "an unhealthy server is still an api, not silence",
			answer:   devServerAnswer{scheme: "http", status: 500, contentType: "text/plain", body: "boom"},
			wantKind: devServerKindAPI,
		},
		{
			name:         "go's tls refusal is recognised, never a page",
			answer:       devServerAnswer{scheme: "http", status: 400, contentType: "text/plain; charset=utf-8", body: "Client sent an HTTP request to an HTTPS server.\n"},
			wantKind:     devServerKindAPI,
			wantMismatch: true,
		},
		{
			name:     "a plain 400 is not the tls refusal",
			answer:   devServerAnswer{scheme: "http", status: 400, contentType: "text/plain", body: "bad request"},
			wantKind: devServerKindAPI,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, title, tool := judgeDevServerAnswer(c.answer)
			if kind != c.wantKind || title != c.wantTitle || tool != c.wantTool {
				t.Fatalf("judge = (%q,%q,%q) want (%q,%q,%q)", kind, title, tool, c.wantKind, c.wantTitle, c.wantTool)
			}
			if got := tlsMismatch(c.answer); got != c.wantMismatch {
				t.Fatalf("tlsMismatch = %v want %v", got, c.wantMismatch)
			}
		})
	}
}

// TestProcessStartedAt covers the wall-clock half of a process's identity:
// start ticks since boot, plus the boot time the read cached.
func TestProcessStartedAt(t *testing.T) {
	if got := processStartedAt(""); !got.IsZero() {
		t.Fatalf("empty token = %v", got)
	}
	if got := processStartedAt("not a number"); !got.IsZero() {
		t.Fatalf("garbage token = %v", got)
	}
	boot := procBootTime()
	if boot.IsZero() {
		t.Skip("no /proc/stat btime here")
	}
	ticks := int64(time.Since(boot).Seconds() * procClockTicks)
	got := processStartedAt(strconv.FormatInt(ticks, 10))
	if got.IsZero() || got.After(time.Now()) || time.Since(got) > time.Hour {
		t.Fatalf("started at %v, now %v", got, time.Now())
	}
}

// stubDevServers replaces the three seams for one test: who owns a port, what
// a port answers, and which ports are listening right now. By default every
// port in the fixture is listening — a silent port is one nobody owns.
func stubDevServers(t *testing.T, owners map[int]devServerOwner, answers map[int]devServerProbe) *int {
	t.Helper()
	prevOwners, prevProbe, prevListening := devServerOwnersOfDeps, openDevServerProbe, devServerListening
	calls := 0
	devServerOwnersOfDeps = func(context.Context, Deps) (map[int]devServerOwner, bool) { return owners, true }
	devServerListening = func() map[int]string {
		out := map[int]string{}
		for port := range answers {
			out[port] = "inode"
		}
		for port := range owners {
			out[port] = "inode"
		}
		return out
	}
	openDevServerProbe = func(_ context.Context, port int) devServerProbe {
		calls++
		if probe, ok := answers[port]; ok {
			return probe
		}
		return devServerProbe{kind: devServerKindOpaque}
	}
	t.Cleanup(func() {
		devServerOwnersOfDeps, openDevServerProbe, devServerListening = prevOwners, prevProbe, prevListening
	})
	return &calls
}

func newDevServerDeps() Deps {
	return Deps{DevServers: newDevServerCache()}
}

func rowFor(t *testing.T, page DevServersPage, port int) DevServer {
	t.Helper()
	for _, row := range page.Servers {
		if row.Port == port {
			return row
		}
	}
	t.Fatalf("no row for port %d in %+v", port, page.Servers)
	return DevServer{}
}

// TestDevServersDecisionTable pins what becomes a row, what it says it is, and
// what the panel may then do with it.
func TestDevServersDecisionTable(t *testing.T) {
	owners := map[int]devServerOwner{
		9999:  {kind: "term", id: "t1", name: "api", workspace: "web", tool: "uvicorn", pid: 4242, startKey: "900", startedAt: time.Now().Add(-10 * time.Minute)},
		45683: {kind: "agent", id: "a1", name: "sidebar", workspace: "PiCode", tool: "agy", pid: 4243, startKey: "901", startedAt: time.Now().Add(-2 * time.Hour)},
	}
	answers := map[int]devServerProbe{
		5173:  {scheme: "http", kind: devServerKindPage, title: "Vite + React"},
		45683: {scheme: "https", kind: devServerKindAPI, tool: "agy"},
	}
	calls := stubDevServers(t, owners, answers)
	deps := newDevServerDeps()
	page := devServers(context.Background(), deps, false)
	if !page.Readable {
		t.Fatal("owners were readable in this stub")
	}
	if page.Total != 3 || len(page.Servers) != 3 {
		t.Fatalf("page = %+v", page)
	}

	// A page nobody in PiCode started is still a row: the usual dev port
	// answered, and that is what the panel is for.
	vite := rowFor(t, page, 5173)
	if vite.Kind != devServerKindPage || vite.Title != "Vite + React" || vite.OwnerKind != "" || vite.State != devServerStateLive {
		t.Fatalf("vite row = %+v", vite)
	}
	if vite.URL != "http://localhost:5173/" || vite.PID != 0 || vite.StartKey != "" {
		t.Fatalf("vite url/identity = %+v", vite)
	}

	// A port that never answered but a pane owns is still a row — with the
	// process identity Stop needs, and no "open" claim about it.
	uvicorn := rowFor(t, page, 9999)
	if uvicorn.Kind != devServerKindOpaque || uvicorn.State != devServerStateSilent {
		t.Fatalf("uvicorn row = %+v", uvicorn)
	}
	if uvicorn.PID != 4242 || uvicorn.StartKey != "900" || uvicorn.Tool != "uvicorn" {
		t.Fatalf("uvicorn identity = %+v", uvicorn)
	}

	// The HTTPS control channel is named as what it is: an API over https,
	// owned by an agent. This is the row the panel used to offer as a page.
	agy := rowFor(t, page, 45683)
	if agy.Kind != devServerKindAPI || agy.Scheme != "https" || agy.URL != "https://localhost:45683/" {
		t.Fatalf("agy row = %+v", agy)
	}
	if agy.OwnerKind != "agent" || agy.OwnerName != "sidebar" || agy.Workspace != "PiCode" || agy.Tool != "agy" {
		t.Fatalf("agy owner = %+v", agy)
	}
	if agy.StartedAt == "" {
		t.Fatalf("agy row has no start time: %+v", agy)
	}

	// A candidate port that answers nothing and nobody owns is not a server.
	for _, s := range page.Servers {
		if s.Port == 3000 {
			t.Fatalf("unanswered candidate was reported: %+v", s)
		}
	}
	first := *calls
	if first < 2 {
		t.Fatalf("probes=%d, want at least the two listeners", first)
	}

	// A poll within the TTL reuses what it learned: no new probes.
	if got := devServers(context.Background(), deps, false); len(got.Servers) != 3 {
		t.Fatalf("cached list=%+v", got)
	}
	if *calls != first {
		t.Fatalf("a poll inside the TTL probed again: %d → %d", first, *calls)
	}
}

// TestDevServerStartingState pins the one state that depends on the clock: a
// pane-owned port that answers nothing is "starting" while the process is
// young, and "silent" once it is not.
func TestDevServerStartingState(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name  string
		owner devServerOwner
		want  string
	}{
		{"young process", devServerOwner{pid: 9, startKey: "1", startedAt: now.Add(-5 * time.Second)}, devServerStateStarting},
		{"old process", devServerOwner{pid: 9, startKey: "1", startedAt: now.Add(-5 * time.Minute)}, devServerStateSilent},
		{"no start time", devServerOwner{pid: 9, startKey: "1"}, devServerStateSilent},
		{"answered", devServerOwner{}, devServerStateLive},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind := devServerKindOpaque
			if c.name == "answered" {
				kind = devServerKindAPI
			}
			if got := devServerState(kind, c.owner, now); got != c.want {
				t.Fatalf("state = %q want %q", got, c.want)
			}
		})
	}
}

// TestDevServerStaleCacheIsNotARow pins the fix a scratch panel found: a
// stopped server kept its row until the cache's TTL ran out (the TTL is about
// not re-probing a live page, never about believing a dead port).
func TestDevServerStaleCacheIsNotARow(t *testing.T) {
	answers := map[int]devServerProbe{5173: {scheme: "http", kind: devServerKindPage, title: "Acme"}}
	stubDevServers(t, nil, answers)
	deps := newDevServerDeps()
	if page := devServers(context.Background(), deps, false); len(page.Servers) != 1 {
		t.Fatalf("first read = %+v", page)
	}
	// The server is gone: nothing listens on the port any more, but the probe
	// answer is still cached for another two minutes.
	devServerListening = func() map[int]string { return map[int]string{} }
	page := devServers(context.Background(), deps, false)
	if len(page.Servers) != 0 {
		t.Fatalf("a port that stopped listening kept its row: %+v", page.Servers)
	}
}

// TestDevServersRefreshReprobes pins that the refresh button re-probes.
func TestDevServersRefreshReprobes(t *testing.T) {
	answers := map[int]devServerProbe{5173: {scheme: "http", kind: devServerKindPage, title: "before"}}
	stubDevServers(t, nil, answers)
	deps := newDevServerDeps()
	if page := devServers(context.Background(), deps, false); len(page.Servers) != 1 || page.Servers[0].Title != "before" {
		t.Fatalf("first read = %+v", page)
	}
	answers[5173] = devServerProbe{scheme: "http", kind: devServerKindPage, title: "after"}
	page := devServers(context.Background(), deps, true)
	if len(page.Servers) != 1 || page.Servers[0].Title != "after" {
		t.Fatalf("refresh did not re-probe: %+v", page)
	}
}

func TestDevServersEmptyAndRoute(t *testing.T) {
	stubDevServers(t, nil, nil)
	deps := newDevServerDeps()
	if page := devServers(context.Background(), deps, false); len(page.Servers) != 0 {
		t.Fatalf("empty machine answered %+v", page)
	}

	rec := httptest.NewRecorder()
	handleDevServers(deps)(rec, httptest.NewRequest(http.MethodGet, "/api/devservers", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var body DevServersPage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Servers == nil {
		t.Fatalf("servers must be an empty array, not null: %s", rec.Body.String())
	}

	// Anything but GET is refused here (the mux pattern says GET already).
	rec = httptest.NewRecorder()
	handleDevServers(deps)(rec, httptest.NewRequest(http.MethodPost, "/api/devservers", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d", rec.Code)
	}
}

// TestDevServerUnreadableOwners covers the machine where the walk finds
// nothing (a native Windows daemon, no /proc): the read says so, instead of
// looking like a machine with no servers.
func TestDevServerUnreadableOwners(t *testing.T) {
	prevOwners, prevProbe, prevListening := devServerOwnersOfDeps, openDevServerProbe, devServerListening
	devServerOwnersOfDeps = func(context.Context, Deps) (map[int]devServerOwner, bool) { return nil, false }
	devServerListening = func() map[int]string { return map[int]string{5173: "inode"} }
	openDevServerProbe = func(_ context.Context, port int) devServerProbe {
		if port == 5173 {
			return devServerProbe{scheme: "http", kind: devServerKindPage, title: "Acme"}
		}
		return devServerProbe{kind: devServerKindOpaque}
	}
	t.Cleanup(func() {
		devServerOwnersOfDeps, openDevServerProbe, devServerListening = prevOwners, prevProbe, prevListening
	})

	page := devServers(context.Background(), newDevServerDeps(), false)
	if page.Readable {
		t.Fatal("readable must be false when the walk cannot run")
	}
	if len(page.Servers) != 1 || page.Servers[0].Title != "Acme" {
		t.Fatalf("page = %+v", page)
	}
}

// TestDevServerCacheResolve covers the cache's own contract: unresolved ports
// get probed in parallel, a fresh entry is trusted, and an explicit refresh
// asks again.
func TestDevServerCacheResolve(t *testing.T) {
	prev := openDevServerProbe
	var calls int
	openDevServerProbe = func(_ context.Context, port int) devServerProbe {
		calls++
		if port == 5173 {
			return devServerProbe{scheme: "http", kind: devServerKindPage, title: "Acme", tool: "dev"}
		}
		return devServerProbe{kind: devServerKindOpaque}
	}
	t.Cleanup(func() { openDevServerProbe = prev })

	cache := newDevServerCache()
	ports := []int{5173, 3000}
	facts := cache.resolve(context.Background(), ports, false)
	if calls != 2 {
		t.Fatalf("probes=%d want one per port", calls)
	}
	if facts[5173].kind != devServerKindPage || facts[5173].title != "Acme" || facts[5173].tool != "dev" {
		t.Fatalf("5173 facts=%+v", facts[5173])
	}
	if facts[3000].kind != devServerKindOpaque {
		t.Fatalf("a silent port answered: %+v", facts[3000])
	}
	// Inside the TTL nothing is probed again, an explicit refresh probes both.
	cache.resolve(context.Background(), ports, false)
	if calls != 2 {
		t.Fatalf("a poll inside the TTL probed again: %d", calls)
	}
	cache.resolve(context.Background(), ports, true)
	if calls != 4 {
		t.Fatalf("refresh probes=%d want 4", calls)
	}
}

// TestDevServerCacheRetriesSilentPort pins the short retry: a port that
// answered nothing is asked again on the next poll (a dev server that is
// still compiling gets its title within seconds), while a port that answered
// is trusted for the long TTL.
func TestDevServerCacheRetriesSilentPort(t *testing.T) {
	prevProbe, prevRetry := openDevServerProbe, devServerRetryTTL
	devServerRetryTTL = 0
	var calls int
	answering := false
	openDevServerProbe = func(context.Context, int) devServerProbe {
		calls++
		if answering {
			return devServerProbe{scheme: "http", kind: devServerKindPage, title: "Acme", tool: "dev"}
		}
		return devServerProbe{kind: devServerKindOpaque}
	}
	t.Cleanup(func() { openDevServerProbe, devServerRetryTTL = prevProbe, prevRetry })

	cache := newDevServerCache()
	ports := []int{5173}
	cache.resolve(context.Background(), ports, false)
	cache.resolve(context.Background(), ports, false)
	if calls != 2 {
		t.Fatalf("silent port was not retried: calls=%d", calls)
	}
	// Once it answers, the next poll trusts the long TTL.
	answering = true
	facts := cache.resolve(context.Background(), ports, false)
	if facts[5173].kind != devServerKindPage || facts[5173].title != "Acme" {
		t.Fatalf("facts=%+v", facts[5173])
	}
	after := calls
	cache.resolve(context.Background(), ports, false)
	if calls != after {
		t.Fatalf("an answered port was re-probed inside the TTL: %d → %d", after, calls)
	}
}
