package store

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func testLlamaJob(key, model string) LlamaJob {
	return LlamaJob{RequestKey: key, Endpoint: "http://localhost:8080", ConnectionID: "test-connection", Model: model, Operation: "load"}
}

func TestLlamaReservationsAndIdentity(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(map[bool]string{false: "model", true: "endpoint"}[replace], func(t *testing.T) {
			s, e := Open(filepath.Join(t.TempDir(), "db"))
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			req := testLlamaJob("request", "a")
			req.ReplaceOthers = replace
			j, created, e := s.BeginLlamaJob(req)
			if e != nil || !created {
				t.Fatal(j, created, e)
			}
			same, created, e := s.BeginLlamaJob(req)
			if e != nil || created || same.ID != j.ID {
				t.Fatal(same, created, e)
			}
			other := req
			other.Model = "b"
			if _, _, e = s.BeginLlamaJob(other); !errors.Is(e, ErrLlamaConflict) {
				t.Fatal(e)
			}
			other = testLlamaJob("other", "a")
			if _, _, e = s.BeginLlamaJob(other); !errors.Is(e, ErrLlamaConflict) {
				t.Fatal(e)
			}
			other.Model = "b"
			other.ReplaceOthers = true
			if _, _, e = s.BeginLlamaJob(other); !errors.Is(e, ErrLlamaConflict) {
				t.Fatal(e)
			}
			j.State = "unknown"
			j, e = s.UpdateLlamaJob(j)
			if e != nil {
				t.Fatal(e)
			}
			if _, _, e = s.BeginLlamaJob(other); !errors.Is(e, ErrLlamaConflict) {
				t.Fatal("unknown released reservation", e)
			}
			stale := j
			j.State = "succeeded"
			j, e = s.UpdateLlamaJob(j)
			if e != nil {
				t.Fatal(e)
			}
			stale.State = "running"
			if _, e = s.UpdateLlamaJob(stale); !errors.Is(e, ErrLlamaConflict) {
				t.Fatal("stale writer accepted")
			}
			if _, created, e = s.BeginLlamaJob(other); e != nil || !created {
				t.Fatal(created, e)
			}
		})
	}
}

func TestLlamaConcurrentReservation(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, key := range []string{"a", "b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, _, err := s.BeginLlamaJob(testLlamaJob(key, "same"))
			results <- err
		}(key)
	}
	wg.Wait()
	close(results)
	accepted := 0
	for e := range results {
		if e == nil {
			accepted++
		} else if !errors.Is(e, ErrLlamaConflict) {
			t.Fatal(e)
		}
	}
	if accepted != 1 {
		t.Fatal(accepted)
	}
}

func TestLlamaJobPersistsProgressAndConnection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	j, _, e := s.BeginLlamaJob(testLlamaJob("key", "model"))
	if e != nil {
		t.Fatal(e)
	}
	j.Progress = []LlamaProgress{{"one.gguf", 20, 100}, {"two.gguf", 4, 0}}
	j.State = "unknown"
	j, e = s.UpdateLlamaJob(j)
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	got, e := s.LlamaJob(j.ID)
	if e != nil || got.ConnectionID != j.ConnectionID || len(got.Progress) != 2 || got.Progress[1].Total != 0 {
		t.Fatal(got, e)
	}
}

func TestLlamaEndpointCapacity(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for _, id := range []string{"a", "b", "c", "d"} {
		if _, _, e = s.BeginLlamaJob(testLlamaJob(id, id)); e != nil {
			t.Fatal(e)
		}
	}
	if _, _, e = s.BeginLlamaJob(testLlamaJob("fifth", "fifth")); !errors.Is(e, ErrLlamaConflict) {
		t.Fatal(e)
	}
	other := testLlamaJob("other-endpoint", "a")
	other.Endpoint = "http://localhost:8081"
	if _, _, e = s.BeginLlamaJob(other); e != nil {
		t.Fatal(e)
	}
}

// History pruning (ADR-0083 amendment 2026-09-25): only finished jobs older
// than the cutoff and outside the newest `keep` go; an old active job stays.
func TestPruneLlamaJobs(t *testing.T) {
	s := testStore(t)
	mk := func(key, state string, age time.Duration) string {
		j, _, err := s.BeginLlamaJob(testLlamaJob(key, key))
		if err != nil {
			t.Fatal(err)
		}
		if state != "queued" {
			j.State = state
			if j, err = s.UpdateLlamaJob(j); err != nil {
				t.Fatal(err)
			}
		}
		at := time.Now().Add(-age).UTC().Format(time.RFC3339Nano)
		if _, err := s.db.Exec(`UPDATE llama_jobs SET created_at=? WHERE id=?`, at, j.ID); err != nil {
			t.Fatal(err)
		}
		return j.ID
	}
	oldDone := mk("old-done", "succeeded", 60*24*time.Hour)
	oldActive := mk("old-active", "unknown", 61*24*time.Hour)
	young := mk("young", "failed", 24*time.Hour)
	newestOld := mk("newest-old", "failed", 40*24*time.Hour)
	// keep=2: the two newest rows (young, newest-old) stay whatever their age.
	n, err := s.PruneLlamaJobs(time.Now().Add(-30*24*time.Hour), 2)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("pruned %d rows, want 1", n)
	}
	if _, err := s.LlamaJob(oldDone); err == nil {
		t.Error("an old finished job outside the kept rows was not pruned")
	}
	for _, id := range []string{oldActive, young, newestOld} {
		if _, err := s.LlamaJob(id); err != nil {
			t.Errorf("%s was pruned: %v", id, err)
		}
	}
}
