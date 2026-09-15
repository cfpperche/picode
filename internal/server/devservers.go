package server

// Dev servers: what is listening on this machine's loopback, who owns it and
// what the page calls itself. A dev server is the one thing the file preview
// cannot serve — it exists only while its process does, on a port — so this
// route answers a read: the browser tab talks to the server directly, never
// through the daemon.
//
// Three facts, three costs:
//   - which ports listen: /proc/net/tcp{,6} (Linux; empty elsewhere).
//   - who owns a port: the process tree of each PiCode terminal/agent pane —
//     the same /proc walk ADR-0062 already does for CLI presence.
//   - what the page is called: an HTTP GET for <title>, the only reason this
//     code ever speaks to the user's servers. Once per port, plus an explicit
//     refresh — never on every poll, or a panel left open would spam the dev
//     server's own log.
//
// What is reported stays narrow on purpose: ports a PiCode pane owns, plus
// the usual dev ports that actually answer. Everything else listening on the
// machine (a database, a private API) is none of this panel's business.

import (
	"context"
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
	// devServerTTL is how long a probed title is trusted before a poll asks
	// the page again: a dev server must not log a PiCode request every few
	// seconds because a panel is open.
	devServerTTL = 2 * time.Minute
	// devServerTimeout bounds one probe, headers to title.
	devServerTimeout = 900 * time.Millisecond
	// devServerTreeCap bounds one pane's process walk, like the reconciler's.
	devServerTreeCap = 96
)

// devServerRetryTTL is the shorter trust for a port that answered nothing — a
// server still starting gets its title within seconds, not minutes. A var so
// a test can collapse it.
var devServerRetryTTL = 10 * time.Second

// devServerPorts are the ports a dev server usually takes when it was started
// outside a PiCode terminal (an IDE task, another tool). A port a pane owns
// is reported whatever its number; this list only widens detection.
var devServerPorts = []int{
	1234, 3000, 3001, 3002, 4173, 4200, 4321, 5000, 5173, 5174, 5175, 5176,
	5500, 6006, 8000, 8001, 8080, 8081, 8888, 9000, 9090,
}

// DevServer is one running server.
type DevServer struct {
	Port      int    `json:"port"`
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	Tool      string `json:"tool,omitempty"`
	OwnerKind string `json:"ownerKind,omitempty"` // agent | term
	OwnerID   string `json:"ownerId,omitempty"`
	OwnerName string `json:"ownerName,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

// DevServerCache remembers the last probe per port, so a poll costs one
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
	title string
	tool  string
	http  bool // something answered HTTP there
	at    time.Time
}

// devServerOwner names the pane a port belongs to.
type devServerOwner struct {
	kind      string
	id        string
	name      string
	workspace string
	tool      string
}

// openDevServerProbe and devServerOwnersOfDeps are the seams a test replaces:
// no /proc, no sockets, no real dev server behind the assertions.
var (
	openDevServerProbe    = probeDevServerTitle
	devServerOwnersOfDeps = devServerOwners
)

// registerDevServerRoutes wires the one read this feature needs. The route is
// an ordinary guarded API: the browser that fills the panel is the one that
// asked for it.
func registerDevServerRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/devservers", handleDevServers(deps))
}

func handleDevServers(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		servers := devServers(r.Context(), deps, r.URL.Query().Get("refresh") == "1")
		writeJSON(w, http.StatusOK, map[string]any{"servers": servers})
	}
}

// devServers answers the list. refresh re-probes every row (the panel's own
// Refresh button); a poll only probes what is new or stale.
func devServers(ctx context.Context, deps Deps, refresh bool) []DevServer {
	out := []DevServer{}
	if deps.DevServers == nil {
		return out
	}
	owners := devServerOwnersOfDeps(ctx, deps)

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

	for _, port := range ports {
		owner := owners[port]
		fact := facts[port]
		// A port is a server when it answered HTTP, or when a PiCode pane
		// owns its socket (a non-HTTP server, or one still starting, is still
		// the thing the user just launched).
		if !fact.http && owner.kind == "" {
			continue
		}
		srv := DevServer{
			Port:  port,
			URL:   "http://localhost:" + strconv.Itoa(port) + "/",
			Title: fact.title,
			Tool:  firstNonEmpty(owner.tool, fact.tool),
		}
		if owner.kind != "" {
			srv.OwnerKind, srv.OwnerID, srv.OwnerName, srv.Workspace = owner.kind, owner.id, owner.name, owner.workspace
		}
		out = append(out, srv)
	}
	if len(out) > devServerRows {
		out = out[:devServerRows]
	}
	return out
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
		if had && !prev.http {
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
				title, tool, ok, err := openDevServerProbe(ctx, port)
				facts := devServerFacts{title: title, tool: tool, http: ok && err == nil, at: time.Now()}
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
// listens on it; empty when /proc is unreadable (not Linux).
func devServerOwners(ctx context.Context, deps Deps) map[int]devServerOwner {
	out := map[int]devServerOwner{}
	if deps.Store == nil || deps.Tmux == nil {
		return out
	}
	snap := readProcSnapshot()
	if len(snap.ppid) == 0 {
		return out
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
		for inode, tool := range devServerSocketInodes(pid, snap) {
			if _, seen := inodeOwner[inode]; seen {
				continue
			}
			next := owner
			if next.tool == "" {
				next.tool = tool
			}
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
			for _, name := range []string{tmux.SessionName(a.ID), tmux.ShellSessionName(a.ID)} {
				pid, err := deps.Tmux.PanePID(ctx, name)
				if err != nil || pid <= 0 {
					continue
				}
				claim(pid, owner)
			}
		}
	}
	if len(inodeOwner) == 0 {
		return out
	}
	for port, inode := range readLoopbackListenerInodes() {
		if owner, ok := inodeOwner[inode]; ok {
			if _, taken := out[port]; !taken {
				out[port] = owner
			}
		}
	}
	return out
}

// devServerSocketInodes walks a pane's process tree and answers the socket
// inodes its processes hold, each with a tool hint from that process's own
// command line. A dev server holds its listening socket, so the inode is the
// join key between "who" and "which port".
func devServerSocketInodes(panePID int, snap *procSnapshot) map[string]string {
	children := make(map[int][]int, len(snap.ppid))
	for pid, ppid := range snap.ppid {
		children[ppid] = append(children[ppid], pid)
	}
	out := map[string]string{}
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
		tool := devServerTool(snap.argv[pid])
		for _, entry := range entries {
			target, err := os.Readlink(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			inode, ok := strings.CutPrefix(target, "socket:[")
			if !ok {
				continue
			}
			out[strings.TrimSuffix(inode, "]")] = tool
		}
	}
	return out
}

// probeDevServerTitle asks one loopback port for its page. Only the title and
// a generator tag are read, capped, under a short deadline.
func probeDevServerTitle(ctx context.Context, port int) (string, string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, devServerTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(port)+"/", nil)
	if err != nil {
		return "", "", false, err
	}
	req.Header.Set("Accept", "text/html,*/*")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", false, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 500 {
		// A dev server that is up but unhealthy is still a server; anything
		// else that answers non-2xx proves the port too (a 404 is an answer).
		return "", "", false, nil
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, devServerBodyCap))
	text := string(body)
	return htmlTitle(text), generatorTool(text), true, nil
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
