package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Dev-server discovery: the parsers that read the machine, and the list the
// panel consumes. Nothing here touches a real socket or /proc — the three
// seams are stubbed, so the assertions are about the decision table, not the
// environment.

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

// stubDevServers replaces the two seams for one test: who owns a port, and
// what a page answers.
func stubDevServers(t *testing.T, owners map[int]devServerOwner, answers map[int]string) *int {
	t.Helper()
	prevOwners, prevProbe := devServerOwnersOfDeps, openDevServerProbe
	calls := 0
	devServerOwnersOfDeps = func(context.Context, Deps) map[int]devServerOwner { return owners }
	openDevServerProbe = func(_ context.Context, port int) (string, string, bool, error) {
		calls++
		title, ok := answers[port]
		if !ok {
			return "", "", false, nil
		}
		return title, "", true, nil
	}
	t.Cleanup(func() {
		devServerOwnersOfDeps, openDevServerProbe = prevOwners, prevProbe
	})
	return &calls
}

func newDevServerDeps() Deps {
	return Deps{DevServers: newDevServerCache()}
}

func TestDevServersDecisionTable(t *testing.T) {
	owners := map[int]devServerOwner{
		9999: {kind: "term", id: "t1", name: "api", workspace: "web", tool: "uvicorn"},
	}
	answers := map[int]string{5173: "Vite + React"}
	calls := stubDevServers(t, owners, answers)
	deps := newDevServerDeps()
	got := devServers(context.Background(), deps, false)
	if len(got) != 2 {
		t.Fatalf("servers=%v want two rows", got)
	}
	// Sorted by port: 5173 (probe only) then 9999 (owned, HTTP silent).
	if got[0].Port != 5173 || got[0].Title != "Vite + React" || got[0].OwnerKind != "" {
		t.Fatalf("row 0 = %+v", got[0])
	}
	if got[0].URL != "http://localhost:5173/" {
		t.Fatalf("url=%q", got[0].URL)
	}
	if got[1].Port != 9999 || got[1].OwnerName != "api" || got[1].Workspace != "web" || got[1].Tool != "uvicorn" {
		t.Fatalf("row 1 = %+v", got[1])
	}
	// A candidate port that answers nothing and nobody owns is not a server.
	for _, s := range got {
		if s.Port == 3000 {
			t.Fatalf("unanswered candidate was reported: %+v", s)
		}
	}
	// 5173 + 9999 + every silent candidate, once each.
	first := *calls
	if first < 2 {
		t.Fatalf("probes=%d, want at least the two listeners", first)
	}

	// A poll within the TTL reuses what it learned: no new probes.
	if got2 := devServers(context.Background(), deps, false); len(got2) != 2 {
		t.Fatalf("cached list=%v", got2)
	}
	if *calls != first {
		t.Fatalf("a poll inside the TTL probed again: %d → %d", first, *calls)
	}
}

func TestDevServersRefreshReprobes(t *testing.T) {
	answers := map[int]string{5173: "before"}
	calls := stubDevServers(t, nil, answers)
	deps := newDevServerDeps()
	if got := devServers(context.Background(), deps, false); len(got) != 1 || got[0].Title != "before" {
		t.Fatalf("first read = %v", got)
	}
	answers[5173] = "after"
	got := devServers(context.Background(), deps, true)
	if len(got) != 1 || got[0].Title != "after" {
		t.Fatalf("refresh did not re-probe: %v", got)
	}
	if *calls < 2 {
		t.Fatalf("probes=%d", *calls)
	}
}

func TestDevServersEmptyAndRoute(t *testing.T) {
	stubDevServers(t, nil, nil)
	deps := newDevServerDeps()
	if got := devServers(context.Background(), deps, false); len(got) != 0 {
		t.Fatalf("empty machine answered %v", got)
	}

	rec := httptest.NewRecorder()
	handleDevServers(deps)(rec, httptest.NewRequest(http.MethodGet, "/api/devservers", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var body struct {
		Servers []DevServer `json:"servers"`
	}
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

// TestDevServerCacheResolve covers the cache's own contract: unresolved
// ports get probed in parallel, a fresh entry is trusted, and an explicit
// refresh asks again.
func TestDevServerCacheResolve(t *testing.T) {
	prev := openDevServerProbe
	var calls int
	openDevServerProbe = func(_ context.Context, port int) (string, string, bool, error) {
		calls++
		if port == 5173 {
			return "Acme", "dev", true, nil
		}
		return "", "", false, nil
	}
	t.Cleanup(func() { openDevServerProbe = prev })

	cache := newDevServerCache()
	ports := []int{5173, 3000}
	facts := cache.resolve(context.Background(), ports, false)
	if calls != 2 {
		t.Fatalf("probes=%d want one per port", calls)
	}
	if !facts[5173].http || facts[5173].title != "Acme" || facts[5173].tool != "dev" {
		t.Fatalf("5173 facts=%+v", facts[5173])
	}
	if facts[3000].http {
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
	openDevServerProbe = func(context.Context, int) (string, string, bool, error) {
		calls++
		if answering {
			return "Acme", "dev", true, nil
		}
		return "", "", false, nil
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
	if !facts[5173].http || facts[5173].title != "Acme" {
		t.Fatalf("facts=%+v", facts[5173])
	}
	after := calls
	cache.resolve(context.Background(), ports, false)
	if calls != after {
		t.Fatalf("an answered port was re-probed inside the TTL: %d → %d", after, calls)
	}
}
