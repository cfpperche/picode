package server

// The Servers panel's two writes: the guards that decide what may be stopped,
// what a hide remembers, and how a stale hide is forgotten. The owner walk is
// stubbed, so the assertions are about the decisions, not the machine.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// actionDeps builds Deps around a temp store whose port owner is `owners`,
// with the port walk and the listener set stubbed: no /proc in an assertion.
func actionDeps(t *testing.T, owners map[int]devServerOwner) (Deps, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	prevOwners, prevListening := devServerOwnersOfDeps, devServerListening
	devServerOwnersOfDeps = func(context.Context, Deps) (map[int]devServerOwner, bool) { return owners, true }
	devServerListening = func() map[int]string {
		out := map[int]string{}
		for port := range owners {
			out[port] = "inode"
		}
		return out
	}
	t.Cleanup(func() { devServerOwnersOfDeps, devServerListening = prevOwners, prevListening })
	return Deps{Store: st, DevServers: newDevServerCache()}, st
}

// liveIdentity is this test process's own pid and start token: a process that
// is certainly alive, so the guards can be exercised without a child.
func liveIdentity(t *testing.T) (int, string) {
	t.Helper()
	pid := os.Getpid()
	token := processStartToken(pid)
	if token == "" {
		t.Skip("no /proc here: nothing to identify")
	}
	return pid, token
}

func postDevServerAction(t *testing.T, h http.HandlerFunc, body map[string]any, ctx context.Context) (int, map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

// TestDevServerStopGuards is the decision table under the Stop verb: nothing
// reaches a signal unless the port, the process and its start token all still
// agree — the refusal rows are the interesting ones.
func TestDevServerStopGuards(t *testing.T) {
	pid, token := liveIdentity(t)
	owners := map[int]devServerOwner{
		5173: {kind: "term", id: "t1", name: "web", workspace: "PiCode", tool: "vite", pid: pid, startKey: token},
	}
	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"a port PiCode does not own", map[string]any{"port": 9998, "pid": pid, "startKey": token}, http.StatusConflict},
		{"a port with no listening owner", map[string]any{"port": 8080, "pid": pid, "startKey": token}, http.StatusConflict},
		{"a process that is not the holder", map[string]any{"port": 5173, "pid": pid + 1, "startKey": token}, http.StatusConflict},
		{"a start token that does not match", map[string]any{"port": 5173, "pid": pid, "startKey": "1"}, http.StatusConflict},
		{"no start token at all", map[string]any{"port": 5173, "pid": pid}, http.StatusConflict},
		{"no port at all", map[string]any{"port": 0, "pid": pid, "startKey": token}, http.StatusConflict},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prev := signalDevServerProcessFn
			signalDevServerProcessFn = func(int, bool) error {
				t.Fatal("a refused stop reached the signal")
				return nil
			}
			t.Cleanup(func() { signalDevServerProcessFn = prev })

			deps, _ := actionDeps(t, owners)
			if code, _ := postDevServerAction(t, handleDevServerStop(deps), c.body, nil); code != c.want {
				t.Fatalf("status=%d want %d", code, c.want)
			}
		})
	}
}

// TestDevServerStopStillRunning covers the honest half of a Stop that did not
// take: the signal went out, the process did not end, and the answer says so
// instead of pretending. The process here is the test process itself, and the
// signal is the seam — nothing is signalled for real.
func TestDevServerStopStillRunning(t *testing.T) {
	pid, token := liveIdentity(t)
	prevSignal, prevWait := signalDevServerProcessFn, devServerStopWait
	devServerStopWait = 30 * time.Millisecond
	var sawForce bool
	signalDevServerProcessFn = func(got int, force bool) error {
		if got != pid {
			t.Fatalf("signalled pid %d want %d", got, pid)
		}
		sawForce = force
		return nil
	}
	t.Cleanup(func() { signalDevServerProcessFn, devServerStopWait = prevSignal, prevWait })

	owners := map[int]devServerOwner{5173: {kind: "term", id: "t1", name: "web", tool: "vite", pid: pid, startKey: token}}
	deps, st := actionDeps(t, owners)

	var events []string
	st.OnEvent = func(ev store.Event) { events = append(events, ev.Type) }

	code, out := postDevServerAction(t, handleDevServerStop(deps), map[string]any{"port": 5173, "pid": pid, "startKey": token, "force": true}, nil)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%v", code, out)
	}
	if out["stopped"] != false {
		t.Fatalf("a live process answered stopped=%v", out["stopped"])
	}
	if !sawForce {
		t.Fatal("force did not reach the signal")
	}
	if len(events) != 1 || events[0] != "devserver.stopped" {
		t.Fatalf("events = %v, want the one audit row", events)
	}
}

// TestDevServerHideRoundTrip covers the hide write, the read that marks the
// row, and the way back.
func TestDevServerHideRoundTrip(t *testing.T) {
	pid, token := liveIdentity(t)
	owners := map[int]devServerOwner{
		5173:  {kind: "term", id: "t1", name: "web", workspace: "PiCode", tool: "vite", pid: pid, startKey: token, startedAt: time.Now().Add(-time.Hour)},
		45683: {kind: "agent", id: "a1", name: "sidebar", workspace: "PiCode", tool: "agy", pid: pid + 1, startKey: token},
	}
	deps, _ := actionDeps(t, owners)
	stubProbe := map[int]devServerProbe{
		5173:  {scheme: "http", kind: devServerKindPage, title: "Acme"},
		45683: {scheme: "https", kind: devServerKindAPI, tool: "agy"},
	}
	prevProbe, prevListening := openDevServerProbe, devServerListening
	openDevServerProbe = func(_ context.Context, port int) devServerProbe {
		if p, ok := stubProbe[port]; ok {
			return p
		}
		return devServerProbe{kind: devServerKindOpaque}
	}
	devServerListening = func() map[int]string { return map[int]string{5173: "inode", 45683: "inode"} }
	t.Cleanup(func() { openDevServerProbe, devServerListening = prevProbe, prevListening })

	code, out := postDevServerAction(t, handleDevServerHide(deps), map[string]any{"port": 5173, "pid": pid, "startKey": token}, nil)
	if code != http.StatusOK || out["hideId"] == nil {
		t.Fatalf("hide status=%d body=%v", code, out)
	}
	// Hiding a port the walk does not own, or one whose identity moved, is a
	// refusal: a hide must mean one listener, not a port forever.
	if code, _ := postDevServerAction(t, handleDevServerHide(deps), map[string]any{"port": 8080, "pid": pid, "startKey": token}, nil); code != http.StatusConflict {
		t.Fatalf("hiding an unowned port answered %d", code)
	}
	if code, _ := postDevServerAction(t, handleDevServerHide(deps), map[string]any{"port": 5173, "pid": pid, "startKey": "1"}, nil); code != http.StatusConflict {
		t.Fatalf("hiding a moved process answered %d", code)
	}

	page := devServers(context.Background(), deps, false)
	if page.Hidden != 1 {
		t.Fatalf("hidden count = %d", page.Hidden)
	}
	row := rowFor(t, page, 5173)
	if !row.Hidden || row.HideID == 0 {
		t.Fatalf("row = %+v", row)
	}
	if other := rowFor(t, page, 45683); other.Hidden {
		t.Fatalf("a second row took the hide: %+v", other)
	}

	code, out = postDevServerAction(t, handleDevServerUnhide(deps), map[string]any{"hideId": row.HideID}, nil)
	if code != http.StatusOK || out["ok"] != true {
		t.Fatalf("unhide status=%d body=%v", code, out)
	}
	if page := devServers(context.Background(), deps, false); page.Hidden != 0 {
		t.Fatalf("row stayed hidden: %+v", page)
	}
	// Showing something twice is a 404, not a second write.
	if code, _ := postDevServerAction(t, handleDevServerUnhide(deps), map[string]any{"hideId": row.HideID}, nil); code != http.StatusNotFound {
		t.Fatalf("second unhide answered %d", code)
	}
}

// TestPruneDevServerHides covers the cleanup a hide triggers: a hide whose
// process is gone can never match a listener again, so it is forgotten — and
// the one whose process is alive stays.
func TestPruneDevServerHides(t *testing.T) {
	pid, token := liveIdentity(t)
	deps, st := actionDeps(t, nil)
	live, err := st.HideDevServer(5173, pid, token, `terminal "web"`, "vite")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.HideDevServer(45683, 999999, "1", "", "agy"); err != nil {
		t.Fatal(err)
	}
	pruneDevServerHides(deps)
	hides, err := st.ListDevServerHides()
	if err != nil {
		t.Fatal(err)
	}
	if len(hides) != 1 || hides[0].ID != live.ID {
		t.Fatalf("hides = %+v, want only the live one", hides)
	}
}
