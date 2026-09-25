package llamajob

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

type router struct {
	mu              sync.Mutex
	models          map[string]string
	posts           map[string]int
	cancelCompletes bool
	rejectUnload    bool
	offline         bool
	events          bool
}

func (f *router) handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/models/sse" {
		if !f.events {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.offline {
		http.Error(w, "private diagnostics", 503)
		return
	}
	if r.URL.Path == "/props" {
		json.NewEncoder(w).Encode(map[string]string{"role": "router", "build_info": "b10809-qa"})
		return
	}
	if r.Method == "POST" {
		var body struct {
			Model string `json:"model"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		f.posts[r.URL.Path]++
		switch r.URL.Path {
		case "/models":
			f.models[body.Model] = "downloading"
		case "/models/load":
			f.models[body.Model] = "loading"
		case "/models/unload":
			if f.rejectUnload {
				http.Error(w, "secret", 500)
				return
			}
			if f.models[body.Model] == "downloading" && !f.cancelCompletes {
				delete(f.models, body.Model)
			} else {
				f.models[body.Model] = "unloaded"
			}
		}
		w.Write([]byte(`{"success":true}`))
		return
	}
	data := []any{}
	for id, state := range f.models {
		data = append(data, map[string]any{"id": id, "status": map[string]any{"value": state, "progress": map[string]any{"https://host/one.gguf?secret=1": map[string]int{"done": 25, "total": 100}, "https://host/two.gguf": map[string]int{"done": 4, "total": 0}}}})
	}
	json.NewEncoder(w).Encode(map[string]any{"data": data})
}
func fixture(t *testing.T, events bool) (*router, *Service, *store.Store, string) {
	t.Helper()
	f := &router{models: map[string]string{"a": "unloaded"}, posts: map[string]int{}, events: events}
	upstream := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(upstream.Close)
	st, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	s, err := New(st, func() (string, string) { return upstream.URL, "key" })
	if err != nil {
		t.Fatal(err)
	}
	s.interval = 5 * time.Millisecond
	s.timeLimit = 2 * time.Second
	t.Cleanup(s.Close)
	return f, s, st, upstream.URL
}
func await(t *testing.T, st *store.Store, id string, check func(store.LlamaJob) bool) store.LlamaJob {
	t.Helper()
	end := time.Now().Add(4 * time.Second)
	for time.Now().Before(end) {
		j, e := st.LlamaJob(id)
		if e != nil {
			t.Fatal(e)
		}
		if check(j) {
			return j
		}
		time.Sleep(5 * time.Millisecond)
	}
	j, _ := st.LlamaJob(id)
	t.Fatalf("job did not settle: %+v", j)
	return j
}
func TestLoadDedupAndFallback(t *testing.T) {
	f, s, st, _ := fixture(t, false)
	j, e := s.Start("a", "load", "same", false)
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.Start("a", "load", "same", false)
	if e != nil || j.ID != again.ID {
		t.Fatal(again, e)
	}
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.Observed == "loading" })
	f.mu.Lock()
	f.models["a"] = "loaded"
	f.mu.Unlock()
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.State == "succeeded" })
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts["/models/load"] != 1 {
		t.Fatal(f.posts)
	}
}
func TestDownloadCancelAndCompletionRace(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(map[bool]string{false: "canceled", true: "completed"}[complete], func(t *testing.T) {
			f, s, st, _ := fixture(t, true)
			f.cancelCompletes = complete
			j, e := s.Start("new", "download", "download", false)
			if e != nil {
				t.Fatal(e)
			}
			observed := await(t, st, j.ID, func(j store.LlamaJob) bool { return j.Observed == "downloading" })
			if len(observed.Progress) != 2 || observed.Progress[0].File != "one.gguf" || observed.Progress[1].Total != 0 {
				t.Fatal(observed.Progress)
			}
			if _, e = s.Cancel(j.ID); e != nil {
				t.Fatal(e)
			}
			want := "canceled"
			if complete {
				want = "succeeded"
			}
			await(t, st, j.ID, func(j store.LlamaJob) bool { return j.State == want })
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.posts["/models/unload"] != 1 {
				t.Fatal(f.posts)
			}
		})
	}
}
func TestReplaceFailureNeverLoads(t *testing.T) {
	f, s, st, _ := fixture(t, false)
	f.models["old"] = "loaded"
	f.rejectUnload = true
	j, e := s.Start("a", "load", "replace", true)
	if e != nil {
		t.Fatal(e)
	}
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.State == "unknown" })
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts["/models/load"] != 0 {
		t.Fatal(f.posts)
	}
}
func TestRestartReconcilesWithoutReplay(t *testing.T) {
	f, s, st, url := fixture(t, false)
	j, e := s.Start("a", "load", "load", false)
	if e != nil {
		t.Fatal(e)
	}
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.Observed == "loading" })
	s.Close()
	f.mu.Lock()
	f.models["a"] = "loaded"
	f.mu.Unlock()
	next, e := New(st, func() (string, string) { return url, "key" })
	if e != nil {
		t.Fatal(e)
	}
	defer next.Close()
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.State == "succeeded" })
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts["/models/load"] != 1 {
		t.Fatal(f.posts)
	}
}
func TestRestartChangedConnectionAndCancellationNeverReplays(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(map[bool]string{false: "same", true: "changed"}[changed], func(t *testing.T) {
			f, s, st, url := fixture(t, true)
			s.Close()
			j, _, e := st.BeginLlamaJob(store.LlamaJob{RequestKey: "restore", Endpoint: url, ConnectionID: identity(url, "key"), Model: "new", Operation: "download"})
			if e != nil {
				t.Fatal(e)
			}
			j.State = "running"
			j.Stage = "cancelDispatch"
			j.CancelRequested = true
			j.CancelSupported = true
			if _, e = st.UpdateLlamaJob(j); e != nil {
				t.Fatal(e)
			}
			f.mu.Lock()
			f.models["new"] = "downloading"
			f.mu.Unlock()
			key := "key"
			if changed {
				key = "new-key"
			}
			next, e := New(st, func() (string, string) { return url, key })
			if e != nil {
				t.Fatal(e)
			}
			await(t, st, j.ID, func(j store.LlamaJob) bool {
				if changed {
					return j.Message == "Restore the original server connection, then check the result."
				}
				return j.Observed == "downloading"
			})
			next.Close()
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.posts) != 0 {
				t.Fatal("replayed mutation", f.posts)
			}
		})
	}
}
func TestCancelUnsupportedAndFailedDownload(t *testing.T) {
	f, s, st, _ := fixture(t, false)
	j, e := s.Start("new", "download", "download", false)
	if e != nil {
		t.Fatal(e)
	}
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.Observed == "downloading" })
	if _, e = s.Cancel(j.ID); e == nil {
		t.Fatal("unsupported cancellation accepted")
	}
	f.mu.Lock()
	f.models["new"] = "failed"
	f.mu.Unlock()
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.State == "failed" })
}

func TestPreflightMatrix(t *testing.T) {
	for _, tc := range []struct {
		name, model, state, operation, want string
		replace                             bool
	}{
		{"external busy", "a", "loading", "load", "failed", false},
		{"missing", "missing", "unloaded", "load", "failed", false},
		{"already loaded", "a", "loaded", "load", "succeeded", false},
		{"already unloaded", "a", "unloaded", "unload", "succeeded", false},
		{"replace busy", "a", "loading", "load", "failed", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, s, st, _ := fixture(t, false)
			f.models["a"] = tc.state
			j, e := s.Start(tc.model, tc.operation, tc.name, tc.replace)
			if e != nil {
				t.Fatal(e)
			}
			await(t, st, j.ID, func(j store.LlamaJob) bool { return j.State == tc.want })
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.posts) != 0 {
				t.Fatal(f.posts)
			}
		})
	}
}

func TestTimeoutAndQueuedRecovery(t *testing.T) {
	f, s, st, url := fixture(t, false)
	s.timeLimit = 50 * time.Millisecond
	j, e := s.Start("a", "load", "timeout", false)
	if e != nil {
		t.Fatal(e)
	}
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.State == "unknown" })
	s.Close()
	queued, _, e := st.BeginLlamaJob(store.LlamaJob{RequestKey: "queued", Endpoint: url, ConnectionID: identity(url, "key"), Model: "b", Operation: "load"})
	if e != nil {
		t.Fatal(e)
	}
	f.mu.Lock()
	f.models["a"] = "unloaded"
	f.mu.Unlock()
	next, e := New(st, func() (string, string) { return url, "key" })
	if e != nil {
		t.Fatal(e)
	}
	defer next.Close()
	await(t, st, queued.ID, func(j store.LlamaJob) bool { return j.State == "interrupted" })
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.State == "interrupted" })
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts["/models/load"] != 1 {
		t.Fatal(f.posts)
	}
}

func TestReplaceCompletesInOrder(t *testing.T) {
	f, s, st, _ := fixture(t, false)
	f.models["old"] = "loaded"
	j, e := s.Start("a", "load", "replace", true)
	if e != nil {
		t.Fatal(e)
	}
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.Observed == "loading" })
	f.mu.Lock()
	if f.models["old"] != "unloaded" || f.posts["/models/unload"] != 1 {
		t.Error(f.models, f.posts)
	}
	f.models["a"] = "loaded"
	f.mu.Unlock()
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.State == "succeeded" })
}

// ADR-0083 amendment (2026-09-23): the owner can abandon an unknown job; it
// releases the model, touches nothing on the server, and only an unknown job
// can be abandoned.
func TestAbandonReleasesAnUnknownJob(t *testing.T) {
	f, s, st, url := fixture(t, false)
	j, _, err := st.BeginLlamaJob(store.LlamaJob{RequestKey: "k1", Endpoint: url, ConnectionID: identity(url, "key"), Model: "a", Operation: "load"})
	if err != nil {
		t.Fatal(err)
	}
	j.State = "unknown"
	if j, err = st.UpdateLlamaJob(j); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.BeginLlamaJob(store.LlamaJob{RequestKey: "k0", Endpoint: url, ConnectionID: identity(url, "key"), Model: "a", Operation: "load"}); err == nil {
		t.Fatal("an unknown job must keep its model reserved")
	}
	done, err := s.Abandon(j.ID)
	if err != nil || done.State != "abandoned" || done.Active() {
		t.Fatalf("abandon = %+v, %v", done, err)
	}
	f.mu.Lock()
	if len(f.posts) != 0 {
		t.Fatalf("abandoning sent something to the server: %v", f.posts)
	}
	f.mu.Unlock()
	// The model is free again.
	if _, e := s.Start("a", "load", "k2", false); e != nil {
		t.Fatalf("the model stayed reserved: %v", e)
	}
	if _, err := s.Abandon(done.ID); err != store.ErrLlamaConflict {
		t.Fatalf("abandoning a finished job = %v", err)
	}
}

// ADR-0083/0090 amendments: a job on PiCode's own service while it is not
// running is not in flight — Check result, the worker and InterruptStopped
// all end it as interrupted instead of leaving it unknown forever.
func TestJobsOnAStoppedOwnedServiceAreInterrupted(t *testing.T) {
	f, s, st, url := fixture(t, false)
	stopped := false
	var mu sync.Mutex
	s.SetStopped(func(endpoint string) bool { mu.Lock(); defer mu.Unlock(); return stopped && endpoint == url })
	j, e := s.Start("fresh-model", "download", "d1", false)
	if e != nil {
		t.Fatal(e)
	}
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.Observed == "downloading" })
	// The owned service stops: the router goes away with it.
	f.mu.Lock()
	f.offline = true
	f.mu.Unlock()
	mu.Lock()
	stopped = true
	mu.Unlock()
	got := await(t, st, j.ID, func(j store.LlamaJob) bool { return !j.Active() })
	if got.State != "interrupted" {
		t.Fatalf("state = %s (%s)", got.State, got.Message)
	}
	// And a leftover unknown job is settled by InterruptStopped.
	left, _, err := st.BeginLlamaJob(store.LlamaJob{RequestKey: "x", Endpoint: url, ConnectionID: "c", Model: "b", Operation: "unload"})
	if err != nil {
		t.Fatal(err)
	}
	left.State = "unknown"
	if left, err = st.UpdateLlamaJob(left); err != nil {
		t.Fatal(err)
	}
	if err := s.InterruptStopped(); err != nil {
		t.Fatal(err)
	}
	if after, _ := st.LlamaJob(left.ID); after.State != "interrupted" {
		t.Fatalf("leftover = %s", after.State)
	}
}

// ADR-0083 amendment: after a restart, an unload whose model is still loaded
// did not happen — interrupted, mirroring the load rule.
func TestRestartedUnloadStillLoadedIsInterrupted(t *testing.T) {
	f, s, st, url := fixture(t, false)
	f.mu.Lock()
	f.models["a"] = "loaded"
	f.mu.Unlock()
	j, _, err := st.BeginLlamaJob(store.LlamaJob{RequestKey: "u", Endpoint: url, ConnectionID: identity(url, "key"), Model: "a", Operation: "unload"})
	if err != nil {
		t.Fatal(err)
	}
	j.State = "unknown"
	if _, err = st.UpdateLlamaJob(j); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reconcile(j.ID); err != nil {
		t.Fatal(err)
	}
	got := await(t, st, j.ID, func(j store.LlamaJob) bool { return !j.Active() })
	if got.State != "interrupted" {
		t.Fatalf("state = %s (%s)", got.State, got.Message)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts["/models/unload"] != 0 {
		t.Fatal("checking the result must not replay the unload")
	}
}

// The download owner hears about every ended download, not only a success:
// an abandoned one lets go of what it recorded at the start.
func TestAbandonedDownloadIsReported(t *testing.T) {
	_, _, st, url := fixture(t, false)
	got := make(chan store.LlamaJob, 1)
	s, err := New(st, func() (string, string) { return url, "key" }, func(j store.LlamaJob) { got <- j })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	j, _, err := st.BeginLlamaJob(store.LlamaJob{RequestKey: "d1", Endpoint: url, ConnectionID: identity(url, "key"), Model: "a", Operation: "download"})
	if err != nil {
		t.Fatal(err)
	}
	j.State = "unknown"
	if _, err = st.UpdateLlamaJob(j); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Abandon(j.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-got:
		if r.ID != j.ID || r.State != "abandoned" {
			t.Fatalf("reported %+v", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("an abandoned download was never reported")
	}
}

// A download PiCode lost track of that the server does not list at all is
// released as interrupted after a few reads (ADR-0083 amendment 2026-09-25),
// without sending anything; one the server still lists keeps waiting.
func TestLostDownloadTheServerDoesNotListIsReleased(t *testing.T) {
	f, s, st, url := fixture(t, false)
	lost, _, err := st.BeginLlamaJob(store.LlamaJob{RequestKey: "d-lost", Endpoint: url, ConnectionID: identity(url, "key"), Model: "gone", Operation: "download"})
	if err != nil {
		t.Fatal(err)
	}
	lost.State = "unknown"
	if _, err = st.UpdateLlamaJob(lost); err != nil {
		t.Fatal(err)
	}
	s.absentFor = 30 * time.Millisecond
	if _, err := s.Reconcile(lost.ID); err != nil {
		t.Fatal(err)
	}
	got := await(t, st, lost.ID, func(j store.LlamaJob) bool { return !j.Active() })
	if got.State != "interrupted" {
		t.Fatalf("state = %s (%s)", got.State, got.Message)
	}
	// The model is free again.
	if _, _, err := st.BeginLlamaJob(store.LlamaJob{RequestKey: "d-again", Endpoint: url, ConnectionID: identity(url, "key"), Model: "gone", Operation: "download"}); err != nil {
		t.Fatalf("the model stayed reserved: %v", err)
	}
	f.mu.Lock()
	posts := len(f.posts)
	f.mu.Unlock()
	if posts != 0 {
		t.Fatalf("checking the result sent something to the server: %v", f.posts)
	}
}

func TestLostDownloadTheServerStillRunsIsFollowed(t *testing.T) {
	f, s, st, url := fixture(t, false)
	f.mu.Lock()
	f.models["a"] = "downloading"
	f.mu.Unlock()
	j, _, err := st.BeginLlamaJob(store.LlamaJob{RequestKey: "d-live", Endpoint: url, ConnectionID: identity(url, "key"), Model: "a", Operation: "download"})
	if err != nil {
		t.Fatal(err)
	}
	j.State = "unknown"
	if _, err = st.UpdateLlamaJob(j); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reconcile(j.ID); err != nil {
		t.Fatal(err)
	}
	await(t, st, j.ID, func(j store.LlamaJob) bool { return j.State == "running" })
	time.Sleep(20 * s.interval)
	if got, _ := st.LlamaJob(j.ID); !got.Active() {
		t.Fatalf("a download the server still runs was released: %s (%s)", got.State, got.Message)
	}
}

// Inside the grace period a lost download stays unknown however many reads
// miss it: SSE wakes can pack three reads into a second, and a send PiCode
// saw time out may be listed by the server only later.
func TestLostDownloadKeepsItsGracePeriod(t *testing.T) {
	_, s, st, url := fixture(t, false)
	s.absentFor = time.Hour
	j, _, err := st.BeginLlamaJob(store.LlamaJob{RequestKey: "d-grace", Endpoint: url, ConnectionID: identity(url, "key"), Model: "gone", Operation: "download"})
	if err != nil {
		t.Fatal(err)
	}
	j.State = "unknown"
	if _, err = st.UpdateLlamaJob(j); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reconcile(j.ID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * s.interval) // many more than absentReads reads
	if got, _ := st.LlamaJob(j.ID); !got.Active() {
		t.Fatalf("released inside the grace period: %s (%s)", got.State, got.Message)
	}
}
