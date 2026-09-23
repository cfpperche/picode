package llamaservice

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

// The review's reproduction (2026-09-23): an unknown model job on the owned
// endpoint while the service is stopped refused every lifecycle action — the
// start that could answer it included. With the resolver (llamajob's
// InterruptStopped) installed, Preview settles such jobs first.
func TestStoppedServiceIsNotBlockedByItsOwnJobs(t *testing.T) {
	s := testService(t)
	configure(t, s)
	s.mu.Lock()
	s.doc.Current = &Release{Version: "b10809", Dir: "/nonexistent", Files: map[string]string{"llama-server": "x"}}
	s.doc.Applied = s.doc.Config
	s.mu.Unlock()
	endpoint := "http://127.0.0.1:" + strconv.Itoa(s.doc.Applied.Port)
	j, _, err := s.st.BeginLlamaJob(store.LlamaJob{RequestKey: "k", Endpoint: endpoint, ConnectionID: "c", Model: "m", Operation: "download"})
	if err != nil {
		t.Fatal(err)
	}
	j.State = "unknown"
	if _, err = s.st.UpdateLlamaJob(j); err != nil {
		t.Fatal(err)
	}
	if !s.Stopped(endpoint) {
		t.Fatal("a created service with no process is stopped")
	}
	// Without a resolver the guard still refuses (it never guesses).
	if _, err = s.Preview(Request{Action: "start", Revision: s.rev}); err == nil || !strings.Contains(err.Error(), "reconcile model operations") {
		t.Fatalf("no resolver: %v", err)
	}
	s.SetJobResolver(func() {
		jobs, _ := s.st.LlamaJobs()
		for _, x := range jobs {
			if x.Active() && s.Stopped(x.Endpoint) {
				x.State = "interrupted"
				_, _ = s.st.UpdateLlamaJob(x)
			}
		}
	})
	if _, err = s.Preview(Request{Action: "start", Revision: s.rev}); err != nil && strings.Contains(err.Error(), "reconcile model operations") {
		t.Fatalf("the stopped service is still blocked by its own job: %v", err)
	}
	if after, _ := s.st.LlamaJob(j.ID); after.State != "interrupted" {
		t.Fatalf("job = %s", after.State)
	}
}

// Leftovers of an install killed mid-way go at start; a complete unrecorded
// installation stays (unknown ownership is preserved, ADR-0090).
func TestSweepInterruptedInstalls(t *testing.T) {
	s := testService(t)
	root := s.root
	for _, d := range []string{"cache", "release-b1-cut", "release-b1-whole"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	must := func(p string) {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	must(filepath.Join(root, "cache", ".install-123"))
	must(filepath.Join(root, "cache", "kept.tar.gz"))
	must(filepath.Join(root, "release-b1-cut", "libggml.so"))
	must(filepath.Join(root, "release-b1-whole", "llama-server"))
	s.sweepInterrupted()
	gone := func(p string) bool { _, err := os.Lstat(filepath.Join(root, p)); return os.IsNotExist(err) }
	if !gone("cache/.install-123") || !gone("release-b1-cut") {
		t.Fatal("leftovers of an interrupted install were not swept")
	}
	if gone("cache/kept.tar.gz") || gone("release-b1-whole") {
		t.Fatal("the sweep removed something that was not a leftover")
	}
}
