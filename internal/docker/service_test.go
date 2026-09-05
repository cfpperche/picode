package docker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

const testID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type fakeEngine struct {
	mu      sync.Mutex
	states  map[string]string
	started string
	calls   int
	mode    string
	entered chan struct{}
	release chan struct{}
}

func (f *fakeEngine) handler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if r.URL.Path == "/version" {
		_ = json.NewEncoder(w).Encode(map[string]string{"ApiVersion": "1.44"})
		return
	}
	if len(parts) < 5 {
		w.WriteHeader(404)
		return
	}
	id, verb := parts[3], parts[4]
	if r.Method == "GET" && verb == "json" {
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"Id": id, "Name": "/qa", "Config": map[string]any{"Image": "qa", "Env": []string{"SECRET=never-expose"}}, "State": map[string]string{"Status": f.states[id], "StartedAt": f.started}})
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(404)
		return
	}
	f.mu.Lock()
	f.calls++
	mode := f.mode
	f.mu.Unlock()
	if f.entered != nil {
		select {
		case f.entered <- struct{}{}:
		default:
		}
	}
	if mode == "wait" {
		select {
		case <-r.Context().Done():
			return
		case <-f.release:
		}
	}
	if mode == "error" {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "engine refused action"})
		return
	}
	f.mu.Lock()
	if mode != "unverified" {
		if verb == "stop" {
			f.states[id] = "exited"
		} else {
			f.states[id] = "running"
		}
		f.started = "2026-09-04T12:01:00Z"
	}
	f.mu.Unlock()
	w.WriteHeader(204)
}

func testService(t *testing.T, state, mode string) (*Service, *fakeEngine) {
	t.Helper()
	f := &fakeEngine{states: map[string]string{testID: state, strings.Repeat("b", 64): state}, started: "2026-09-04T12:00:00Z", mode: mode, entered: make(chan struct{}, 4), release: make(chan struct{})}
	c := socketClient(t, f.handler)
	st, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	s, err := NewService(context.Background(), st, func(context.Context) (*Client, error) { return c, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, f
}

func waitOperation(t *testing.T, s *Service, id string) store.DockerOperation {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		op, err := s.Store.DockerOperation(id)
		if err != nil {
			t.Fatal(err)
		}
		if op.State != "running" {
			return op
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("operation did not settle")
	return store.DockerOperation{}
}

func TestActionStateMatrix(t *testing.T) {
	for _, state := range []string{"created", "exited", "running", "paused", "restarting", "dead", "removing"} {
		for _, action := range []string{"start", "stop", "restart"} {
			t.Run(action+"_"+state, func(t *testing.T) {
				s, f := testService(t, state, "")
				op, err := s.Start(context.Background(), Request{Action: action, ContainerID: testID, RequestKey: "request-123"})
				want := (action == "start" && (state == "created" || state == "exited")) || (action != "start" && state == "running") || (action == "stop" && state == "restarting")
				if !want {
					if err == nil {
						t.Fatal("incompatible action accepted")
					}
					f.mu.Lock()
					defer f.mu.Unlock()
					if f.calls != 0 {
						t.Fatal("invalid state mutated Docker")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				done := waitOperation(t, s, op.ID)
				if done.State != "succeeded" {
					t.Fatalf("result %+v", done)
				}
			})
		}
	}
}

func TestOperationOutcomes(t *testing.T) {
	for _, tc := range []struct{ mode, want string }{{"error", "failed"}, {"unverified", "unknown"}} {
		t.Run(tc.mode, func(t *testing.T) {
			s, _ := testService(t, "exited", tc.mode)
			op, err := s.Start(context.Background(), Request{Action: "start", ContainerID: testID, RequestKey: "request-123"})
			if err != nil {
				t.Fatal(err)
			}
			done := waitOperation(t, s, op.ID)
			if done.State != tc.want || done.Message == "" {
				t.Fatalf("result %+v", done)
			}
		})
	}
}

func TestDuplicateAndConflictingOperations(t *testing.T) {
	s, f := testService(t, "running", "wait")
	req := Request{Action: "restart", ContainerID: testID, RequestKey: "request-123"}
	op, err := s.Start(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	<-f.entered
	again, err := s.Start(context.Background(), req)
	if err != nil || again.ID != op.ID {
		t.Fatalf("duplicate %+v %v", again, err)
	}
	other := req
	other.Action = "stop"
	if _, err = s.Start(context.Background(), other); !errors.Is(err, store.ErrDockerConflict) {
		t.Fatalf("key conflict %v", err)
	}
	other.RequestKey = "request-456"
	if _, err = s.Start(context.Background(), other); !errors.Is(err, store.ErrDockerConflict) {
		t.Fatalf("container conflict %v", err)
	}
	other.ContainerID = strings.Repeat("b", 64)
	second, err := s.Start(context.Background(), other)
	if err != nil {
		t.Fatal(err)
	}
	close(f.release)
	waitOperation(t, s, op.ID)
	waitOperation(t, s, second.ID)
	final, err := s.Start(context.Background(), req)
	if err != nil || final.ID != op.ID {
		t.Fatalf("settled duplicate %+v %v", final, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls != 2 {
		t.Fatalf("mutations %d", f.calls)
	}
}

func TestInvalidOperationRequests(t *testing.T) {
	s, f := testService(t, "exited", "")
	for _, req := range []Request{
		{Action: "delete", ContainerID: testID, RequestKey: "request-123"},
		{Action: "start", ContainerID: "../../etc", RequestKey: "request-123"},
		{Action: "start", ContainerID: testID, RequestKey: "short"},
		{Action: "start", ContainerID: testID, RequestKey: "request-123", AgentID: "missing"},
	} {
		if _, err := s.Start(context.Background(), req); err == nil {
			t.Fatalf("accepted %+v", req)
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls != 0 {
		t.Fatal("invalid request mutated Docker")
	}
}

func TestShutdownRecordsUnknown(t *testing.T) {
	s, f := testService(t, "running", "wait")
	op, err := s.Start(context.Background(), Request{Action: "stop", ContainerID: testID, RequestKey: "request-123"})
	if err != nil {
		t.Fatal(err)
	}
	<-f.entered
	s.Close()
	done := waitOperation(t, s, op.ID)
	if done.State != "unknown" {
		t.Fatalf("result %+v", done)
	}
}

func TestInspectDoesNotExposeEnvironment(t *testing.T) {
	s, _ := testService(t, "running", "")
	c, err := s.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	v, err := c.Inspect(context.Background(), testID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(v)
	if strings.Contains(string(data), "SECRET") {
		t.Fatal("environment exposed")
	}
}
