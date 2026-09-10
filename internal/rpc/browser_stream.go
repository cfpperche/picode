// Browser stream discovery (ADR-0114): locate the agent-browser session's
// loopback stream WebSocket so the daemon can proxy it to the browser
// surface. The algorithm mirrors the pi-browser-capture sidecar
// (packages/pi-browser-capture), which mirrors pi-agent-browser-native's
// implicit session naming: derive `piab-<slug>-<id12>-<cwdhash8>` from the
// pi session id and cwd, then read `<name>.pid` + `<name>.stream` from the
// socket root under the same safety checks (uid-owned 0700 dir, realpath,
// O_NOFOLLOW, ≤16-byte numeric files, pid liveness). Discovery only ever
// reads — the daemon never launches a browser and never enables a server.

package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const managedSessionNamePrefix = "piab-"
const maxProjectSlugLength = 24
const sessionNameCwdHashLength = 8
const sessionNameSessionIDLength = 12

var projectSlugRE = regexp.MustCompile(`[^a-z0-9]+`)
var numericFileRE = regexp.MustCompile(`^\d+\s*$`)

// ErrNoSession reports that the agent's pi process has no session file yet
// (nothing to attach a consent mirror to).
var ErrNoSession = errors.New("the agent has no session file yet")

// inputConsentTTL bounds how long the proxy may drive on a consent mirror
// read (ADR-0115): mouse-move bursts must not hammer the filesystem, and a
// consent flip reaches driving clients within one TTL.
const inputConsentTTL = 500 * time.Millisecond

// implicitSessionName mirrors the sidecar's createImplicitSessionName
// (which mirrors pi-agent-browser-native runtime.ts).
func implicitSessionName(sessionID, cwd string) string {
	normalized := strings.ToLower(strings.ReplaceAll(sessionID, "-", ""))
	if runtime.GOOS == "android" {
		digest := sha256.Sum256([]byte("session:" + normalized + ":cwd:" + cwd))
		return managedSessionNamePrefix + hex.EncodeToString(digest[:])[:sessionNameSessionIDLength*2]
	}
	slug := projectSlugRE.ReplaceAllString(strings.ToLower(filepath.Base(cwd)), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > maxProjectSlugLength {
		slug = slug[:maxProjectSlugLength]
	}
	if slug == "" {
		slug = "project"
	}
	stable := hashPrefix("session:"+normalized, sessionNameSessionIDLength)
	cwdHash := hashPrefix("cwd:"+cwd, sessionNameCwdHashLength)
	return managedSessionNamePrefix + slug + "-" + stable + "-" + cwdHash
}

func hashPrefix(value string, bytes int) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])[:bytes*2]
}

// browserSocketRoot mirrors the sidecar's socketRoot: env override first,
// then the per-uid /tmp directory; unsupported platforms return "".
func browserSocketRoot(override string) string {
	if override != "" {
		return override
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "android" {
		return ""
	}
	prefix := "/tmp/piab"
	if runtime.GOOS == "darwin" {
		prefix = "/private/tmp/piab"
	}
	return prefix + "-" + strconv.Itoa(os.Getuid())
}

// findBrowserStreamPort returns the newest live stream port for sessionBase
// (or one of its `-fresh-` rotations), or 0. Every check the sidecar applies
// is applied here; any failure just moves to the next candidate.
func findBrowserStreamPort(root, sessionBase, namespace string) int {
	if root == "" || !safeSegment(sessionBase) || (namespace != "" && !safeSegment(namespace)) {
		return 0
	}
	if !sameRealPath(root) {
		return 0
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return 0
	}
	if !streamRootTrust(rootInfo) {
		return 0
	}
	dir := root
	if namespace != "" {
		dir = filepath.Join(root, "namespaces", namespace, "run")
	}
	if !sameRealPath(dir) {
		return 0
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	type candidate struct {
		name  string
		mtime int64
	}
	var candidates []candidate
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".stream") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".stream")
		if base != sessionBase && !strings.HasPrefix(base, sessionBase+"-fresh-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		candidates = append(candidates, candidate{base, info.ModTime().UnixNano()})
	}
	// Newest first: the freshest rotation is the live session.
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].mtime > candidates[j].mtime })

	for _, c := range candidates {
		pid := readRendezvousNumber(filepath.Join(dir, c.name+".pid"))
		port := readRendezvousNumber(filepath.Join(dir, c.name+".stream"))
		if pid <= 0 || port <= 0 || port > 65535 {
			continue
		}
		if !pidAlive(pid) {
			continue
		}
		return port
	}
	return 0
}

func safeSegment(value string) bool {
	if value == "." || value == ".." || value == "" {
		return false
	}
	return filepath.Base(value) == value && !strings.Contains(value, "\\")
}

// sameRealPath rejects symlinked directories (Node: realpath(root) === resolve(root)).
func sameRealPath(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	realAbs, err := filepath.Abs(real)
	if err != nil {
		return false
	}
	return realAbs == filepath.Clean(abs)
}

// readRendezvousNumber reads a `.pid`/`.stream` rendezvous file: opened
// O_NOFOLLOW, must be a small regular numeric file owned by this uid.
// Returns 0 on any violation.
func readRendezvousNumber(path string) int {
	f, ok := openRendezvousFile(path)
	if !ok {
		return 0
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16 {
		return 0
	}
	if !rendezvousFileOwned(info) {
		return 0
	}
	buf := make([]byte, 32)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return 0
	}
	if !numericFileRE.Match(buf[:n]) {
		return 0
	}
	value, err := strconv.Atoi(strings.TrimRight(string(buf[:n]), " \t\n\r"))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// BrowserStreamPort resolves the live agent-browser stream port for this
// agent, or 0. The session identity comes from the capture bridge's cache
// (primed at spawn and after settled turns, so no RPC is needed while a
// browser tool runs — that is exactly when the surface is opened).
func (ma *ManagedAgent) BrowserStreamPort(ctx context.Context) (int, bool) {
	sessionID, sessionFile := ma.cachedSessionIdentity(ctx)
	if sessionID == "" || sessionFile == "" {
		return 0, false
	}
	root := browserSocketRoot(os.Getenv("PI_AGENT_BROWSER_SOCKET_DIR"))
	if root == "" {
		return 0, false
	}
	port := findBrowserStreamPort(root, implicitSessionName(sessionID, ma.Path), os.Getenv("AGENT_BROWSER_NAMESPACE"))
	return port, port > 0
}

// cachedSessionIdentity returns the capture bridge's cached session
// identity, resolving it with one get_state (short timeout) when idle.
func (ma *ManagedAgent) cachedSessionIdentity(ctx context.Context) (sessionID, sessionFile string) {
	if ma.capture == nil {
		return getSessionIdentity(ctx, ma)
	}
	ma.capture.mu.Lock()
	sessionID, sessionFile = ma.capture.sessionID, ma.capture.sessionFile
	ma.capture.mu.Unlock()
	if sessionID != "" && sessionFile != "" {
		return sessionID, sessionFile
	}
	resolvedID, resolvedFile := getSessionIdentity(ctx, ma)
	if resolvedID == "" && resolvedFile == "" {
		return sessionID, sessionFile
	}
	ma.capture.mu.Lock()
	if resolvedID != "" {
		ma.capture.sessionID = resolvedID
	}
	if resolvedFile != "" {
		ma.capture.sessionFile = resolvedFile
	}
	ma.capture.mu.Unlock()
	return resolvedID, resolvedFile
}

// getSessionIdentity asks the live pi process for its session identity.
// Failures stay silent: the browser surface is optional decoration, like
// the capture bridge.
func getSessionIdentity(ctx context.Context, ma *ManagedAgent) (string, string) {
	if ma == nil || ma.client == nil {
		return "", ""
	}
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := ma.GetState(callCtx)
	if err != nil || !resp.Success {
		return "", ""
	}
	var data struct {
		SessionID   string `json:"sessionId"`
		SessionFile string `json:"sessionFile"`
	}
	if json.Unmarshal(resp.Data, &data) != nil {
		return "", ""
	}
	return data.SessionID, data.SessionFile
}

// BrowserInputConsent reports whether the session's input mirror (ADR-0115)
// currently enables browser control. The mirror is written by the
// pi-browser-capture extension beside the session file; an absent or
// malformed mirror means off. Readings are cached for one TTL so input
// bursts do not hammer the filesystem.
func (ma *ManagedAgent) BrowserInputConsent(ctx context.Context) bool {
	_, sessionFile := ma.cachedSessionIdentity(ctx)
	if sessionFile == "" {
		return false
	}
	if ma.capture == nil {
		return readInputMirror(sessionFile)
	}
	cs := ma.capture
	cs.mu.Lock()
	if time.Since(cs.inputCheckedAt) < inputConsentTTL {
		on := cs.inputOn
		cs.mu.Unlock()
		return on
	}
	cs.mu.Unlock()
	on := readInputMirror(sessionFile)
	cs.mu.Lock()
	cs.inputOn = on
	cs.inputCheckedAt = time.Now()
	cs.mu.Unlock()
	return on
}

// readInputMirror reads the sidecar's consent mirror: `{"on":bool}` in the
// session's capture directory. Any absence or malformation is consent off.
func readInputMirror(sessionFile string) bool {
	raw, err := os.ReadFile(filepath.Join(sessionFile+".capture", "input.json"))
	if err != nil {
		return false
	}
	var mirror struct {
		On bool `json:"on"`
	}
	return json.Unmarshal(raw, &mirror) == nil && mirror.On
}

// WriteInputConsent flips the session's input mirror (ADR-0115) and primes
// the consent cache, so the surface's own next input is not refused by a
// stale TTL read.
func (ma *ManagedAgent) WriteInputConsent(ctx context.Context, on bool) error {
	_, sessionFile := ma.cachedSessionIdentity(ctx)
	if sessionFile == "" {
		return ErrNoSession
	}
	dir := sessionFile + ".capture"
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp := filepath.Join(dir, "input.json.tmp")
	if err := os.WriteFile(tmp, []byte(`{"on":`+strconv.FormatBool(on)+`,"ts":`+strconv.FormatInt(time.Now().UnixMilli(), 10)+`}`), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(dir, "input.json")); err != nil {
		return err
	}
	if ma.capture != nil {
		ma.capture.mu.Lock()
		ma.capture.inputOn = on
		ma.capture.inputCheckedAt = time.Now()
		ma.capture.mu.Unlock()
	}
	return nil
}
