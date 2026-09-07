package llamaservice

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

func fixtureRelease(t *testing.T, s *Service, name string) *Release {
	t.Helper()
	dir := filepath.Join(s.root, name)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "llama-server")
	if err := os.WriteFile(path, []byte("verified synthetic release"), 0700); err != nil {
		t.Fatal(err)
	}
	h, _, err := hashRegular(path)
	if err != nil {
		t.Fatal(err)
	}
	r := &Release{Version: "b10809", Dir: dir, Files: map[string]string{"llama-server": h}}
	s.rememberRelease(r)
	return r
}

func TestReleaseCleanupMatrix(t *testing.T) {
	for _, scenario := range []string{"eligible", "current", "rollback", "unknown", "changed", "extra", "symlink", "outside", "running", "model-job", "stale", "expired", "changed-after-review", "promoted-after-review", "permission-failure"} {
		t.Run(scenario, func(t *testing.T) {
			s := testService(t)
			configure(t, s)
			r := fixtureRelease(t, s, "release-b10809-old")
			s.doc.Current = fixtureRelease(t, s, "release-b10809-current")
			s.doc.Previous = fixtureRelease(t, s, "release-b10809-rollback")
			name := filepath.Base(r.Dir)
			if err := s.save(); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "current":
				name = filepath.Base(s.doc.Current.Dir)
			case "rollback":
				name = filepath.Base(s.doc.Previous.Dir)
			case "unknown":
				delete(s.doc.Releases, name)
			case "changed":
				_ = os.WriteFile(filepath.Join(r.Dir, "llama-server"), []byte("changed"), 0700)
			case "extra":
				_ = os.WriteFile(filepath.Join(r.Dir, "keep"), []byte("keep"), 0600)
			case "symlink":
				if err := os.Remove(filepath.Join(r.Dir, "llama-server")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(s.doc.Current.Dir, "llama-server"), filepath.Join(r.Dir, "llama-server")); err != nil {
					t.Fatal(err)
				}
			case "outside":
				r.Dir = t.TempDir()
				s.doc.Releases[name] = *r
			case "running":
				s.process = &exec.Cmd{}
				defer func() { s.process = nil }()
			case "model-job":
				if _, _, err := s.st.BeginLlamaJob(store.LlamaJob{RequestKey: "busy", Endpoint: "http://127.0.0.1:18080", ConnectionID: "test", Model: "test", Operation: "download"}); err != nil {
					t.Fatal(err)
				}
			}
			p, err := s.Preview(Request{Action: "cleanup", Files: []string{name}, Revision: s.rev})
			good := scenario == "eligible" || scenario == "stale" || scenario == "expired" || scenario == "changed-after-review" || scenario == "promoted-after-review" || scenario == "permission-failure"
			if !good {
				if err == nil {
					t.Fatal("unsafe selection accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.Bytes != int64(len("verified synthetic release")) {
				t.Fatal(p.Bytes)
			}
			switch scenario {
			case "stale":
				if err = s.save(); err != nil {
					t.Fatal(err)
				}
			case "expired":
				old := s.previews[p.Token]
				old.Expires = time.Now().Add(-time.Minute)
				s.previews[p.Token] = old
			case "changed-after-review":
				_ = os.WriteFile(filepath.Join(r.Dir, "llama-server"), []byte("changed"), 0700)
			case "promoted-after-review":
				s.doc.Previous = r
			case "permission-failure":
				if os.Geteuid() == 0 {
					t.Skip("root bypasses directory permissions")
				}
				if err = os.Chmod(r.Dir, 0500); err != nil {
					t.Fatal(err)
				}
				defer os.Chmod(r.Dir, 0700)
			}
			_, err = s.Execute(p.Token, false)
			if scenario != "eligible" && scenario != "permission-failure" {
				if err == nil {
					t.Fatal("unsafe confirmation accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			job := finish(t, s)
			if scenario == "permission-failure" {
				if job.State != "failed" {
					t.Fatal(job)
				}
				return
			}
			if job.State != "succeeded" {
				t.Fatal(job)
			}
			if _, err = os.Stat(r.Dir); !os.IsNotExist(err) {
				t.Fatal("old installation survived", err)
			}
			for _, keep := range []*Release{s.doc.Current, s.doc.Previous} {
				if err = verifyRelease(keep); err != nil {
					t.Fatal("protected installation changed", err)
				}
			}
		})
	}
}

func TestReleaseCleanupInterrupted(t *testing.T) {
	s := testService(t)
	configure(t, s)
	r := fixtureRelease(t, s, "release-b10809-partial")
	remaining := filepath.Join(r.Dir, "library.so")
	if err := os.WriteFile(remaining, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	hash, _, err := hashRegular(remaining)
	if err != nil {
		t.Fatal(err)
	}
	r.Files["library.so"] = hash
	s.rememberRelease(r)
	s.doc.Jobs = []Job{{ID: "cleanup-interrupted", Action: "cleanup", State: "running"}}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	// Simulate termination between two removals. Startup must not replay deletion.
	if err := os.Remove(filepath.Join(r.Dir, "llama-server")); err != nil {
		t.Fatal(err)
	}
	next, err := New(s.st, filepath.Dir(s.root), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	if next.doc.Jobs[0].State != "interrupted" || next.busy {
		t.Fatal(next.doc.Jobs)
	}
	if _, err := os.Stat(remaining); err != nil {
		t.Fatal("remaining file removed", err)
	}
	if _, err := next.Preview(Request{Action: "cleanup", Files: []string{filepath.Base(r.Dir)}, Revision: next.rev}); err == nil {
		t.Fatal("partial manifest adopted")
	}
}
