package llamaservice

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

func TestMain(m *testing.M) { RunSupervisor(); os.Exit(m.Run()) }
func testService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(st, dir, func() ([]string, error) { return []string{"Configured test agent"}, nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close(); st.Close() })
	return s
}
func configure(t *testing.T, s *Service) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("Linux/WSL-only lifecycle; see docs/plans/llama-manager.md.")
	}
	_, err := s.Configure(Config{Port: 18080, Context: 4096, Threads: 2, Jinja: true}, s.rev)
	if err != nil {
		t.Fatal(err)
	}
}
func finish(t *testing.T, s *Service) Job {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Snapshot()
		if !v.Busy && len(v.Jobs) > 0 {
			return v.Jobs[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not finish")
	return Job{}
}
func TestConfigurationGuards(t *testing.T) {
	s := testService(t)
	if _, err := s.Configure(Config{}, 0); err == nil {
		t.Fatal("invalid settings accepted")
	}
	configure(t, s)
	if _, err := s.Configure(s.doc.Config, 0); err == nil {
		t.Fatal("stale revision accepted")
	}
	v := s.Snapshot()
	v.Config.Threads = 4
	if s.Snapshot().Config.Threads != 2 {
		t.Fatal("snapshot changed live state")
	}
	for _, action := range []string{"start", "stop", "restart", "rollback", "unknown", "cleanup"} {
		if _, err := s.Preview(Request{Action: action, Revision: s.rev}); err == nil {
			t.Fatalf("accepted %s without prerequisites", action)
		}
	}
	if _, err := s.Preview(Request{Action: "install", Version: "unverified", Revision: s.rev}); err == nil {
		t.Fatal("unverified release")
	}
	raw, rev, err := s.st.LlamaService()
	if err != nil || rev != s.rev {
		t.Fatal(err)
	}
	var doc Document
	if json.Unmarshal(raw, &doc) != nil || doc.Config != s.doc.Config {
		t.Fatal("configuration not persisted")
	}
}
func cacheArtifact(t *testing.T, s *Service, name string) {
	t.Helper()
	path := filepath.Join(s.root, "cache", name)
	if err := os.WriteFile(path, []byte("owned installer artifact"), 0600); err != nil {
		t.Fatal(err)
	}
	h, _, _ := hashRegular(path)
	s.doc.Archives[name] = h
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
}
func TestCacheReviewMatrix(t *testing.T) {
	for _, scenario := range []string{"eligible", "changed", "unowned", "symlink", "duplicate", "stale", "expired"} {
		t.Run(scenario, func(t *testing.T) {
			s := testService(t)
			configure(t, s)
			cacheArtifact(t, s, "archive")
			req := Request{Action: "cleanup", Files: []string{"archive"}, Revision: s.rev}
			if scenario == "unowned" {
				req.Files = []string{"unknown"}
			}
			if scenario == "duplicate" {
				req.Files = append(req.Files, "archive")
			}
			p, err := s.Preview(req)
			if scenario == "unowned" || scenario == "duplicate" {
				if err == nil {
					t.Fatal("unsafe selection accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "changed":
				if err = os.WriteFile(filepath.Join(s.root, "cache", "archive"), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				path := filepath.Join(s.root, "cache", "archive")
				if err = os.Rename(path, path+".target"); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink(path+".target", path); err != nil {
					t.Fatal(err)
				}
			case "stale":
				_, err = s.Configure(s.doc.Config, s.rev)
				if err != nil {
					t.Fatal(err)
				}
			case "expired":
				old := s.previews[p.Token]
				old.Expires = time.Now().Add(-time.Minute)
				s.previews[p.Token] = old
			}
			j, err := s.Execute(p.Token, false)
			if scenario != "eligible" {
				if err == nil {
					t.Fatal("stale or unsafe review executed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			same, err := s.Execute(p.Token, false)
			if err != nil || same.ID != j.ID {
				t.Fatal("retry not idempotent", err)
			}
			if done := finish(t, s); done.State != "succeeded" {
				t.Fatal(done)
			}
			if _, err = os.Stat(filepath.Join(s.root, "cache", "archive")); !os.IsNotExist(err) {
				t.Fatal("eligible file retained")
			}
		})
	}
}
func TestInterruptedRecoveryAndDiagnostics(t *testing.T) {
	s := testService(t)
	configure(t, s)
	s.doc.Jobs = []Job{{ID: "secret-id", Action: "install", State: "running", Message: "secret-path"}}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	next, err := New(s.st, filepath.Dir(s.root), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	if next.doc.Jobs[0].State != "interrupted" {
		t.Fatal(next.doc.Jobs)
	}
	raw, _ := json.Marshal(next.Diagnostics())
	var data map[string]any
	_ = json.Unmarshal(raw, &data)
	if len(data) != 6 {
		t.Fatal(string(raw))
	}
	for _, field := range []string{"host", "modelsDir", "current", "archives"} {
		if _, ok := data[field]; ok {
			t.Fatal("diagnostics exposed", field)
		}
	}
}
func TestArchiveRefusals(t *testing.T) {
	for _, scenario := range []string{"traversal", "absolute", "link escape", "cycle", "duplicate"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "input.gz")
			f, _ := os.Create(path)
			gz := gzip.NewWriter(f)
			tw := tar.NewWriter(gz)
			header := &tar.Header{Name: "llama-server", Mode: 0700, Typeflag: tar.TypeReg}
			switch scenario {
			case "traversal":
				header.Name = "../outside"
			case "absolute":
				header.Name = "/outside"
			case "link escape":
				header.Typeflag = tar.TypeSymlink
				header.Linkname = "../outside"
			case "cycle":
				header.Typeflag = tar.TypeSymlink
				header.Linkname = "llama-server"
			}
			if err := tw.WriteHeader(header); err != nil {
				t.Fatal(err)
			}
			if scenario == "duplicate" {
				if err := tw.WriteHeader(header); err != nil {
					t.Fatal(err)
				}
			}
			_ = tw.Close()
			_ = gz.Close()
			_ = f.Close()
			out := filepath.Join(dir, "out")
			_ = os.Mkdir(out, 0700)
			if _, err := extractRelease(path, out); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}

// Opt-in: docs/plans/llama-manager.md lists the official archive prerequisite.
// Runs CPU router only; never downloads or loads a model.
func TestLiveOwnedService(t *testing.T) {
	archive := os.Getenv("PICODE_LLAMA_SERVICE_ARCHIVE")
	if archive == "" {
		t.Skip("Set PICODE_LLAMA_SERVICE_ARCHIVE; see docs/plans/llama-manager.md.")
	}
	s := testService(t)
	configure(t, s)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s.doc.Config.Port = listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	if err = s.save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	name := "llama-b10809-bin-ubuntu-x64.tar.gz"
	if runtime.GOARCH == "arm64" {
		name = "llama-b10809-bin-ubuntu-arm64.tar.gz"
	}
	if err = os.WriteFile(filepath.Join(s.root, "cache", name), data, 0600); err != nil {
		t.Fatal(err)
	}
	act := func(action string, interrupt bool) Job {
		t.Helper()
		p, e := s.Preview(Request{Action: action, Version: "b10809", Revision: s.Snapshot().Revision})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.Execute(p.Token, interrupt); e != nil {
			t.Fatal(e)
		}
		return finish(t, s)
	}
	for _, action := range []string{"install", "start", "restart", "stop", "start"} {
		if j := act(action, true); j.State != "succeeded" {
			t.Fatal(action, j)
		}
	}
	// A verified candidate that cannot start must restore the running version.
	badDir := t.TempDir()
	badPath := filepath.Join(badDir, "llama-server")
	if err = os.WriteFile(badPath, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	h, _, _ := hashRegular(badPath)
	s.mu.Lock()
	s.doc.Previous = &Release{Version: "fixture-fails", Dir: badDir, Files: map[string]string{"llama-server": h}}
	err = s.save()
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if j := act("rollback", true); j.State != "failed" || !s.Snapshot().Running {
		t.Fatal("rollback recovery failed", j)
	}
	p, err := s.Preview(Request{Action: "stop", Revision: s.Snapshot().Revision})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Execute(p.Token, false); err == nil {
		t.Fatal("interruption guard missing")
	}
	if _, err = s.Execute(p.Token, true); err != nil {
		t.Fatal(err)
	}
	finish(t, s)
	if s.Snapshot().Running {
		t.Fatal("still running")
	}
	t.Log("verified install, start/restart/stop, failed-version recovery and interruption guard; no models loaded")
}

// A cleanup's hash runs outside the service lock: while one is in flight the
// status (Snapshot) still answers, and the review then completes from the
// hash already taken (2026-09-25; before, GGUF-sized hashes held the lock).
func TestCleanupHashesOutsideTheLock(t *testing.T) {
	s := testService(t)
	configure(t, s)
	cacheArtifact(t, s, "archive")
	hashMu.Lock()
	hashMemo = map[string]hashed{}
	hashMu.Unlock()
	entered, release := make(chan struct{}), make(chan struct{})
	var calls int
	old := hashFile
	hashFile = func(path string) (string, int64, error) {
		calls++
		if calls == 1 {
			close(entered)
			<-release
		}
		return hashRegular(path)
	}
	t.Cleanup(func() { hashFile = old })
	done := make(chan error, 1)
	go func() {
		_, err := s.Preview(Request{Action: "cleanup", Files: []string{"archive"}, Revision: s.rev})
		done <- err
	}()
	<-entered
	status := make(chan struct{})
	go func() { s.Snapshot(); close(status) }()
	select {
	case <-status:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("Snapshot waited on a cleanup hash: the lock was held")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("the file was hashed %d times; the check under the lock must answer from the first", calls)
	}
}

// A file rewritten in place with its modification time set back is not the
// file that was hashed: the change time moved, so the memo hashes it again.
func TestHashMemoSeesAnInPlaceRewrite(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("change time is read on Linux")
	}
	path := filepath.Join(t.TempDir(), "model.gguf")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	first, _, err := hashKnown(path)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(path, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	second, _, err := hashKnown(path)
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Fatal("the memo answered for a file rewritten in place")
	}
}
