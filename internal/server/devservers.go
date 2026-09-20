package server

// Dev servers: what is listening on this machine's loopback, who owns it,
// what the port calls itself, and what the panel may do about it. A dev
// server is the one thing the file preview cannot serve — it exists only
// while its process does, on a port — so this route answers a read: the
// browser tab talks to the server directly, never through the daemon.
//
// Three facts, three costs:
//   - which ports listen: /proc/net/tcp{,6} (Linux; empty elsewhere).
//   - who owns a port: the process tree of each PiCode terminal/agent pane —
//     the same /proc walk ADR-0062 already does for CLI presence — down to
//     the process that holds the socket, with its pid and start token.
//   - what a port is: one HTTP(S) GET for the content type and <title>, the
//     only reason this code ever speaks to the user's servers. Once per port,
//     plus an explicit refresh — never on every poll, or a panel left open
//     would spam the dev server's own log.
//
// What is reported stays narrow on purpose: ports a PiCode pane owns, plus
// the usual dev ports that actually answer. Everything else listening on the
// machine (a database, a private API) is none of this panel's business.
//
// A port that answers is not automatically a page. The first version offered
// "Open" for anything that answered below 500, which put an agent CLI's
// internal control channel — an HTTPS API with a bundled certificate — in the
// list as a page nobody could open (measured on the owner's machine,
// 2026-09-17). Every row now says what the port is (`kind`), which scheme
// answered (`scheme`), and, when PiCode can see the process, the identity an
// action needs (`pid` + `startKey`, the PID-reuse token the peer-stop path
// already uses).

import (
	"context"
	"crypto/tls"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

const (
	devServerBodyCap  = 64 << 10
	devServerTitleCap = 100
	devServerRows     = 40
	// devServerTTL is how long a probed answer is trusted before a poll asks
	// the port again: a dev server must not log a PiCode request every few
	// seconds because a panel is open.
	devServerTTL = 2 * time.Minute
	// devServerTimeout bounds one probe, headers to title.
	devServerTimeout = 900 * time.Millisecond
	// devServerTreeCap bounds one pane's process walk, like the reconciler's.
	devServerTreeCap = 96
	// devServerStarting is how long a pane-owned port that answers nothing is
	// still called "starting": the compile before a dev server takes its port.
	devServerStarting = 60 * time.Second
)

// devServerRetryTTL is the shorter trust for a port that answered nothing — a
// server still starting gets its answer within seconds, not minutes. A var so
// a test can collapse it.
var devServerRetryTTL = 10 * time.Second

// devServerStopWait bounds how long the Stop route waits for the signal to
// show before it answers "still running". A var so a test need not wait.
var devServerStopWait = 1500 * time.Millisecond

// devServerPorts are the ports a dev server usually takes when it was started
// outside a PiCode terminal (an IDE task, another tool). A port a pane owns
// is reported whatever its number; this list only widens detection.
var devServerPorts = []int{
	1234, 3000, 3001, 3002, 4173, 4200, 4321, 5000, 5173, 5174, 5175, 5176,
	5500, 6006, 8000, 8001, 8080, 8081, 8888, 9000, 9090,
}

// What a port turned out to be. A page is what PiCode's browser tab can show;
// an API answered but is not a page (JSON, plain text, a 404); opaque answered
// nothing at all, and exists as a row only because a PiCode pane holds it.
const (
	devServerKindPage   = "page"
	devServerKindAPI    = "api"
	devServerKindOpaque = "opaque"
)

// What a row is doing right now: answered in the last probe, still coming up,
// or listening without answering.
const (
	devServerStateLive     = "live"
	devServerStateStarting = "starting"
	devServerStateSilent   = "silent"
)

// DevServer is one listening port.
type DevServer struct {
	Port      int    `json:"port"`
	URL       string `json:"url"`
	Scheme    string `json:"scheme"` // http | https — what answered
	Kind      string `json:"kind"`   // page | api | opaque
	State     string `json:"state"`  // live | starting | silent
	Title     string `json:"title,omitempty"`
	Tool      string `json:"tool,omitempty"`
	OwnerKind string `json:"ownerKind,omitempty"` // agent | term
	OwnerID   string `json:"ownerId,omitempty"`
	OwnerName string `json:"ownerName,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	// PID is the process holding the listening socket, and StartKey its
	// start token — the pair an action echoes back so a Stop can never
	// signal a process that reused the id. Zero and empty when the port
	// belongs to something PiCode cannot see.
	PID       int    `json:"pid,omitempty"`
	StartKey  string `json:"startKey,omitempty"`
	StartedAt string `json:"startedAt,omitempty"` // RFC3339, for "since 21:40"
	// Hidden marks a row the human hid; HideID is what shows it again.
	Hidden bool  `json:"hidden,omitempty"`
	HideID int64 `json:"hideId,omitempty"`
}

// DevServersPage is the read's answer. The server reports every listener it
// can see, including the ones the panel keeps behind a disclosure; the panel
// decides what to put first, so revealing the rest costs no round trip.
type DevServersPage struct {
	Servers   []DevServer `json:"servers"`
	Hidden    int         `json:"hidden"`    // rows carrying a saved Hide
	Total     int         `json:"total"`     // rows before the cap
	Truncated bool        `json:"truncated"` // more listeners existed than the cap
	Readable  bool        `json:"readable"`  // whether port owners are readable here
}

// DevServerCache remembers the last answer per port, so a poll costs one
// /proc read and the user's servers hear from us at most once per TTL. The
// probes of one refresh run in parallel — a loopback port that silently drops
// a SYN costs its timeout, and twenty-one of those in series (measured: 17 s)
// would park the panel behind one unreachable port.
type DevServerCache struct {
	refresh sync.Mutex // one refresh at a time; late callers reuse its answer
	mu      sync.Mutex
	seen    map[int]devServerFacts
}

func newDevServerCache() *DevServerCache {
	return &DevServerCache{seen: map[int]devServerFacts{}}
}

// devServerFacts is what the probe learned about one port.
type devServerFacts struct {
	kind   string // page | api | opaque
	scheme string // http | https
	title  string
	tool   string
	at     time.Time
}

// devServerOwner names the pane a port belongs to, and the process inside it
// that holds the socket.
type devServerOwner struct {
	kind      string
	id        string
	name      string
	workspace string
	tool      string
	pid       int
	startKey  string
	startedAt time.Time
}

// devServerClaim is one process holding one socket: what the pane walk needs
// to answer "which process", "what is it" and "did it reuse a pid".
type devServerClaim struct {
	pid      int
	tool     string
	startKey string
}

// openDevServerProbe, devServerOwnersOfDeps and devServerListening are the
// seams a test replaces: no /proc, no sockets, no real dev server behind the
// assertions.
var (
	openDevServerProbe    = probeDevServerPort
	devServerOwnersOfDeps = devServerOwners
	devServerListening    = readLoopbackListenerInodes
)

// registerDevServerRoutes wires the read and the two writes the panel has.
// They are ordinary guarded APIs: the browser that fills the panel asked.
func registerDevServerRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/devservers", handleDevServers(deps))
	mux.HandleFunc("POST /api/devservers/stop", handleDevServerStop(deps))
	mux.HandleFunc("POST /api/devservers/hide", handleDevServerHide(deps))
	mux.HandleFunc("POST /api/devservers/unhide", handleDevServerUnhide(deps))
}

func handleDevServers(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, devServers(r.Context(), deps, r.URL.Query().Get("refresh") == "1"))
	}
}

// devServers answers the list. refresh re-probes every row (the panel's own
// Refresh button); a poll only probes what is new or stale.
func devServers(ctx context.Context, deps Deps, refresh bool) DevServersPage {
	page := DevServersPage{Servers: []DevServer{}}
	if deps.DevServers == nil {
		return page
	}
	owners, readable := devServerOwnersOfDeps(ctx, deps)
	page.Readable = readable

	ports := make([]int, 0, len(devServerPorts)+len(owners))
	seenPort := map[int]bool{}
	for _, port := range devServerPorts {
		if !seenPort[port] {
			seenPort[port] = true
			ports = append(ports, port)
		}
	}
	for port := range owners {
		if !seenPort[port] {
			seenPort[port] = true
			ports = append(ports, port)
		}
	}
	sort.Ints(ports)
	facts := deps.DevServers.resolve(ctx, ports, refresh)
	listening := devServerListening()
	hides := devServerHides(deps)
	now := time.Now()

	for _, port := range ports {
		owner := owners[port]
		fact := facts[port]
		// A port is a server while it is really there. A PiCode pane holding
		// the socket is enough — a non-HTTP server, or one still starting, is
		// still the thing the user just launched — but a port nobody owns must
		// be listening *now*: a cached title is not evidence, and without this
		// a stopped server kept its row for the rest of the cache's TTL (seen
		// on a scratch panel minutes after Stop said it was gone).
		_, live := listening[port]
		if owner.kind == "" && (!live || fact.kind == "" || fact.kind == devServerKindOpaque) {
			continue
		}
		scheme := firstNonEmpty(fact.scheme, "http")
		row := DevServer{
			Port:   port,
			URL:    scheme + "://localhost:" + strconv.Itoa(port) + "/",
			Scheme: scheme,
			Kind:   firstNonEmpty(fact.kind, devServerKindOpaque),
			Title:  fact.title,
			Tool:   firstNonEmpty(owner.tool, fact.tool),
			State:  devServerState(firstNonEmpty(fact.kind, devServerKindOpaque), owner, now),
		}
		if owner.kind != "" {
			row.OwnerKind, row.OwnerID, row.OwnerName, row.Workspace = owner.kind, owner.id, owner.name, owner.workspace
			row.PID, row.StartKey = owner.pid, owner.startKey
			if !owner.startedAt.IsZero() {
				row.StartedAt = owner.startedAt.UTC().Format(time.RFC3339)
			}
			if id, ok := hides[devServerHideKey(port, owner.pid, owner.startKey)]; ok {
				row.Hidden, row.HideID = true, id
				page.Hidden++
			}
		}
		page.Servers = append(page.Servers, row)
	}
	page.Total = len(page.Servers)
	if len(page.Servers) > devServerRows {
		page.Servers = page.Servers[:devServerRows]
		page.Truncated = true
	}
	return page
}

// devServerState says what a row is doing. A port that answered is live; one
// that answered nothing is either still coming up (the process is younger
// than the starting window) or listening without answering.
func devServerState(kind string, owner devServerOwner, now time.Time) string {
	if kind != devServerKindOpaque {
		return devServerStateLive
	}
	if owner.pid > 0 && !owner.startedAt.IsZero() && now.Sub(owner.startedAt) < devServerStarting {
		return devServerStateStarting
	}
	return devServerStateSilent
}

// devServerHides answers the saved hides, keyed the way a row looks for one.
func devServerHides(deps Deps) map[string]int64 {
	out := map[string]int64{}
	if deps.Store == nil {
		return out
	}
	hides, err := deps.Store.ListDevServerHides()
	if err != nil {
		return out
	}
	for _, h := range hides {
		out[devServerHideKey(h.Port, h.PID, h.StartKey)] = h.ID
	}
	return out
}

func devServerHideKey(port, pid int, startKey string) string {
	return strconv.Itoa(port) + ":" + strconv.Itoa(pid) + ":" + startKey
}

// resolve answers the probed facts for these ports, refreshing what is new,
// stale or forced. Concurrent calls serialise on one refresh: the second one
// waits for the first and reads its answer instead of probing again.
func (c *DevServerCache) resolve(ctx context.Context, ports []int, forced bool) map[int]devServerFacts {
	c.refresh.Lock()
	defer c.refresh.Unlock()
	now := time.Now()
	todo := make([]int, 0, len(ports))
	for _, port := range ports {
		c.mu.Lock()
		prev, had := c.seen[port]
		c.mu.Unlock()
		ttl := devServerTTL
		if had && prev.kind == devServerKindOpaque {
			ttl = devServerRetryTTL
		}
		if forced || !had || now.Sub(prev.at) > ttl {
			todo = append(todo, port)
		}
	}
	if len(todo) > 0 {
		// A port that silently drops a SYN costs its whole timeout, so the
		// candidates go out in batches wider than the list.
		const parallel = 16
		sem := make(chan struct{}, parallel)
		var wg sync.WaitGroup
		for _, port := range todo {
			if ctx.Err() != nil {
				break
			}
			wg.Add(1)
			sem <- struct{}{}
			go func(port int) {
				defer wg.Done()
				defer func() { <-sem }()
				probe := openDevServerProbe(ctx, port)
				facts := devServerFacts{kind: probe.kind, scheme: probe.scheme, title: probe.title, tool: probe.tool, at: time.Now()}
				c.mu.Lock()
				c.seen[port] = facts
				c.mu.Unlock()
			}(port)
		}
		wg.Wait()
	}
	out := make(map[int]devServerFacts, len(ports))
	c.mu.Lock()
	for _, port := range ports {
		if facts, ok := c.seen[port]; ok {
			out[port] = facts
		}
	}
	c.mu.Unlock()
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// devServerOwners maps a loopback port to the PiCode pane whose process tree
// listens on it; empty when /proc is unreadable (not Linux). The second
// answer says whether port owners are readable at all, which is what the
// panel needs to stay honest on a machine where the walk finds nothing.
func devServerOwners(ctx context.Context, deps Deps) (map[int]devServerOwner, bool) {
	out := map[int]devServerOwner{}
	if deps.Store == nil || deps.Tmux == nil {
		return out, false
	}
	snap := readProcSnapshot()
	if len(snap.ppid) == 0 {
		return out, false
	}
	workspaceNames := map[string]string{}
	if spaces, err := deps.Store.ListWorkspaces(); err == nil {
		for _, w := range spaces {
			workspaceNames[w.ID] = w.Name
		}
	}
	inodeOwner := map[string]devServerOwner{}
	claim := func(pid int, owner devServerOwner) {
		if pid <= 0 {
			return
		}
		for inode, held := range devServerSocketInodes(pid, snap) {
			if _, seen := inodeOwner[inode]; seen {
				continue
			}
			next := owner
			if next.tool == "" {
				next.tool = held.tool
			}
			next.pid, next.startKey = held.pid, held.startKey
			next.startedAt = processStartedAt(held.startKey)
			inodeOwner[inode] = next
		}
	}
	if terminals, err := deps.Store.ListTerminals(); err == nil {
		for _, t := range terminals {
			pid, err := deps.Tmux.PanePID(ctx, tmux.ShellSessionName(t.ID))
			if err != nil || pid <= 0 {
				continue
			}
			claim(pid, devServerOwner{kind: "term", id: t.ID, name: t.Name, workspace: workspaceNames[t.WorkspaceID]})
		}
	}
	if agents, err := deps.Store.ListAllAgents(); err == nil {
		for _, a := range agents {
			owner := devServerOwner{kind: "agent", id: a.ID, name: a.Name, workspace: workspaceNames[a.WorkspaceID]}
			for _, name := range []string{deps.agentSession(a.ID), tmux.ShellSessionName(a.ID)} {
				pid, err := deps.Tmux.PanePID(ctx, name)
				if err != nil || pid <= 0 {
					continue
				}
				claim(pid, owner)
			}
		}
	}
	if len(inodeOwner) == 0 {
		return out, true
	}
	for port, inode := range readLoopbackListenerInodes() {
		if owner, ok := inodeOwner[inode]; ok {
			if _, taken := out[port]; !taken {
				out[port] = owner
			}
		}
	}
	return out, true
}

// devServerSocketInodes walks a pane's process tree and answers the socket
// inodes its processes hold, each with the process that holds it and a tool
// hint from that process's own command line. A dev server holds its listening
// socket, so the inode is the join key between "who" and "which port".
func devServerSocketInodes(panePID int, snap *procSnapshot) map[string]devServerClaim {
	children := make(map[int][]int, len(snap.ppid))
	for pid, ppid := range snap.ppid {
		children[ppid] = append(children[ppid], pid)
	}
	out := map[string]devServerClaim{}
	queue := []int{panePID}
	visited := 0
	for len(queue) > 0 && visited < devServerTreeCap {
		pid := queue[0]
		queue = queue[1:]
		visited++
		queue = append(queue, children[pid]...)
		dir := "/proc/" + strconv.Itoa(pid) + "/fd"
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		held := devServerClaim{pid: pid, tool: devServerTool(snap.argv[pid]), startKey: processStartToken(pid)}
		for _, entry := range entries {
			target, err := os.Readlink(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			inode, ok := strings.CutPrefix(target, "socket:[")
			if !ok {
				continue
			}
			out[strings.TrimSuffix(inode, "]")] = held
		}
	}
	return out
}

// processStartedAt turns a process's start token (ticks since boot, field 22
// of /proc/<pid>/stat) into a wall clock. An unreadable /proc answers zero,
// and the row simply shows no "since" time.
func processStartedAt(startKey string) time.Time {
	ticks, err := strconv.ParseInt(startKey, 10, 64)
	if err != nil || ticks <= 0 {
		return time.Time{}
	}
	boot := procBootTime()
	if boot.IsZero() {
		return time.Time{}
	}
	return boot.Add(time.Duration(ticks) * time.Second / time.Duration(procClockTicks))
}

// procClockTicks is USER_HZ: /proc reports starttime in hundredths of a
// second, the value the Linux ABI fixes for every platform Go builds for.
const procClockTicks = 100

// procBootTime reads /proc/stat's btime once — the anchor that turns start
// ticks into a wall clock.
var procBootTime = sync.OnceValue(func() time.Time {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}
	}
	for _, line := range strings.Split(string(raw), "\n") {
		value, ok := strings.CutPrefix(line, "btime ")
		if !ok {
			continue
		}
		seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return time.Time{}
		}
		return time.Unix(seconds, 0)
	}
	return time.Time{}
})

// devServerAnswer is one HTTP answer, before it is judged.
type devServerAnswer struct {
	scheme      string
	status      int
	contentType string
	body        string
}

// devServerProbe is what one port turned out to be.
type devServerProbe struct {
	scheme string
	kind   string
	title  string
	tool   string
}

// The two clients the probe uses. The TLS one does not verify the certificate
// on purpose: a dev server's certificate is self-signed by definition (Vite
// --https, mkcert, a CLI that ships its own), and this client never trusts
// what it reads with it — it asks the port what it is, reads a content type
// and a title, and closes. The work browser still shows the browser's own
// warning when a human opens the page.
var (
	devServerPlainClient = &http.Client{Timeout: devServerTimeout}
	devServerTLSClient   = &http.Client{
		Timeout:   devServerTimeout,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec // loopback discovery only
	}
)

// probeDevServerPort asks one loopback port what it is. Plain HTTP first —
// most dev servers speak it — and one TLS attempt when the plain answer was
// nothing, or Go's own "you sent HTTP to an HTTPS server" page, which is how
// an HTTPS listener answers a client that guessed http.
func probeDevServerPort(ctx context.Context, port int) devServerProbe {
	answer, err := fetchDevServer(ctx, "http", port)
	if err != nil || tlsMismatch(answer) {
		secure, secureErr := fetchDevServer(ctx, "https", port)
		if secureErr != nil {
			return devServerProbe{kind: devServerKindOpaque}
		}
		answer = secure
	}
	kind, title, tool := judgeDevServerAnswer(answer)
	return devServerProbe{scheme: answer.scheme, kind: kind, title: title, tool: tool}
}

// fetchDevServer reads one loopback port over one scheme. Only a content type
// and a capped body are read, under a short deadline.
func fetchDevServer(ctx context.Context, scheme string, port int) (devServerAnswer, error) {
	ctx, cancel := context.WithTimeout(ctx, devServerTimeout)
	defer cancel()
	client := devServerPlainClient
	if scheme == "https" {
		client = devServerTLSClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scheme+"://127.0.0.1:"+strconv.Itoa(port)+"/", nil)
	if err != nil {
		return devServerAnswer{}, err
	}
	req.Header.Set("Accept", "text/html,*/*")
	res, err := client.Do(req)
	if err != nil {
		return devServerAnswer{}, err
	}
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(res.Body, devServerBodyCap))
	return devServerAnswer{
		scheme:      scheme,
		status:      res.StatusCode,
		contentType: res.Header.Get("Content-Type"),
		body:        string(body),
	}, nil
}

// judgeDevServerAnswer decides what an answer makes a port: a page (HTML is
// what the browser tab can show, whatever the status), or an API — it
// answered, and a 404 is an answer, but it is not a page.
func judgeDevServerAnswer(a devServerAnswer) (kind, title, tool string) {
	if isHTMLAnswer(a) {
		return devServerKindPage, htmlTitle(a.body), generatorTool(a.body)
	}
	return devServerKindAPI, "", generatorTool(a.body)
}

// isHTMLAnswer answers whether a response is a page: a declared HTML type, or
// a document that never declared one and still carries a title.
func isHTMLAnswer(a devServerAnswer) bool {
	if strings.Contains(strings.ToLower(a.contentType), "text/html") {
		return true
	}
	return a.contentType == "" && strings.Contains(strings.ToLower(a.body), "<title")
}

// tlsMismatch recognises Go's own answer to plain HTTP on a TLS listener:
// "Client sent an HTTP request to an HTTPS server". It is how an HTTPS dev
// server, or an agent CLI's control channel, announces itself to a client
// that guessed http.
func tlsMismatch(a devServerAnswer) bool {
	return a.status == http.StatusBadRequest && strings.Contains(a.body, "HTTP request to an HTTPS server")
}

var (
	titleTag      = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	generatorMeta = regexp.MustCompile(`(?is)<meta[^>]+name=["']?generator["']?[^>]+content=["']([^"']+)["']`)
)

func htmlTitle(body string) string {
	m := titleTag.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	title := strings.Join(strings.Fields(html.UnescapeString(m[1])), " ")
	if len(title) > devServerTitleCap {
		title = strings.TrimSpace(title[:devServerTitleCap]) + "…"
	}
	return title
}

func generatorTool(body string) string {
	m := generatorMeta.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// readLoopbackListenerInodes answers port → socket inode for every listening
// loopback socket: the same read answers "what is open" and, through
// /proc/<pid>/fd, "who holds it".
func readLoopbackListenerInodes() map[int]string {
	out := map[int]string{}
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for port, inode := range parseProcNetTCP(string(raw)) {
			out[port] = inode
		}
	}
	return out
}

// parseProcNetTCP reads the LISTEN rows of a /proc/net/tcp table: the local
// address and port in hex, state 0A, and the socket inode.
func parseProcNetTCP(table string) map[int]string {
	out := map[int]string{}
	lines := strings.Split(table, "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 10 || fields[3] != "0A" {
			continue
		}
		addr, portHex, ok := strings.Cut(fields[1], ":")
		if !ok || !loopbackBind(addr) {
			continue
		}
		port, err := strconv.ParseInt(portHex, 16, 32)
		if err != nil || port <= 0 {
			continue
		}
		if _, err := strconv.ParseUint(fields[9], 10, 64); err != nil {
			continue
		}
		out[int(port)] = fields[9]
	}
	return out
}

// loopbackBind reports the local addresses this panel cares about: the two
// loopback forms and the wildcard binds.
func loopbackBind(hexAddr string) bool {
	switch strings.ToUpper(hexAddr) {
	case "0100007F", // 127.0.0.1
		"00000000",                         // 0.0.0.0
		"00000000000000000000000001000000", // ::1
		"00000000000000000000000000000000", // ::
		"0000000000000000000000000000000000000000000000000000FFFF0100007F": // ::ffff:127.0.0.1
		return true
	default:
		return false
	}
}

// devServerTool names the program behind a socket: the binary, or the script
// an interpreter drives ("node …/vite" → vite). A bare shell is not a tool
// name, so it answers "".
func devServerTool(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	first := strings.TrimSuffix(filepath.Base(argv[0]), ".exe")
	if _, isInterp := interpreters[first]; isInterp {
		if len(argv) < 2 {
			return ""
		}
		second := strings.TrimSuffix(filepath.Base(argv[1]), ".exe")
		// `sh -c …` is a wrapper, not a tool.
		if strings.HasPrefix(second, "-") {
			return ""
		}
		second = strings.TrimSuffix(second, filepath.Ext(second))
		if second == "" || second == "sh" || second == "bash" {
			return ""
		}
		return second
	}
	switch first {
	case "sh", "bash", "dash", "zsh", "tmux", "su", "sudo":
		return ""
	}
	return first
}
