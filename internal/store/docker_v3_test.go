package store

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func testDockerJob(key, plan string, targets ...string) DockerJob {
	j := DockerJob{RequestKey: key, PlanID: plan, Endpoint: "unix:///tmp/qa", Actor: "Requester", AgentID: "agent", ApprovedBy: "Owner"}
	for _, id := range targets {
		j.Steps = append(j.Steps, DockerStep{Kind: "container", Target: id, Action: "stop", State: "queued"})
	}
	return j
}

func TestDockerJobsShareAtomicReservations(t *testing.T) {
	for _, first := range []string{"individual", "project"} {
		t.Run(first, func(t *testing.T) {
			s := openTest(t)
			if first == "individual" {
				op, _, err := s.BeginDockerOperation(DockerOperation{Endpoint: "unix:///tmp/qa", ContainerID: "b", Action: "stop", RequestKey: "individual"})
				if err != nil {
					t.Fatal(err)
				}
				if _, _, err = s.BeginDockerJob(testDockerJob("project", "plan", "a", "b")); !errors.Is(err, ErrDockerConflict) {
					t.Fatalf("conflict = %v", err)
				}
				// The unsuccessful multi-target request must not retain target a.
				if _, _, err = s.BeginDockerOperation(DockerOperation{Endpoint: "unix:///tmp/qa", ContainerID: "a", Action: "stop", RequestKey: "other-individual"}); err != nil {
					t.Fatal("partial reservation", err)
				}
				if err = s.FinishDockerOperation(op.ID, "succeeded", "verified"); err != nil {
					t.Fatal(err)
				}
				if _, _, err = s.BeginDockerJob(testDockerJob("after-release", "plan-2", "b")); err != nil {
					t.Fatal(err)
				}
			} else {
				j, _, err := s.BeginDockerJob(testDockerJob("project", "plan", "a", "b"))
				if err != nil {
					t.Fatal(err)
				}
				for _, id := range []string{"a", "b"} {
					if _, _, err = s.BeginDockerOperation(DockerOperation{Endpoint: j.Endpoint, ContainerID: id, Action: "stop", RequestKey: "individual-" + id}); !errors.Is(err, ErrDockerConflict) {
						t.Fatal(err)
					}
				}
				j.State = "failed"
				if err = s.UpdateDockerJob(j); err != nil {
					t.Fatal(err)
				}
				if _, _, err = s.BeginDockerOperation(DockerOperation{Endpoint: j.Endpoint, ContainerID: "a", Action: "stop", RequestKey: "released"}); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestDockerJobIdempotencyAndRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	request := testDockerJob("same-request", "plan-a", "a", "b")
	j, created, err := s.BeginDockerJob(request)
	if err != nil || !created {
		t.Fatal(err)
	}
	old, created, err := s.BeginDockerJob(request)
	if err != nil || created || old.ID != j.ID {
		t.Fatalf("duplicate %+v %v", old, err)
	}
	other := request
	other.PlanID = "another-plan"
	if _, _, err = s.BeginDockerJob(other); !errors.Is(err, ErrDockerConflict) {
		t.Fatal(err)
	}
	other = request
	other.RequestKey = "another-key"
	if _, _, err = s.BeginDockerJob(other); !errors.Is(err, ErrDockerConflict) {
		t.Fatal(err)
	}
	j.Steps[0].State = "running"
	if err = s.UpdateDockerJob(j); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.RecoverDockerJobs(); err != nil {
		t.Fatal(err)
	}
	got, err := s.DockerJob(j.ID)
	if err != nil || got.State != "unknown" || got.Steps[0].State != "unknown" || got.Steps[1].State != "skipped" || got.ApprovedBy != "Owner" || got.Actor != "Requester" {
		t.Fatalf("recovered %+v %v", got, err)
	}
	j.State = "succeeded"
	if err = s.UpdateDockerJob(j); err != nil {
		t.Fatal(err)
	}
	got, _ = s.DockerJob(j.ID)
	if got.State != "unknown" {
		t.Fatal("late update rewrote recovery")
	}
	if _, _, err = s.BeginDockerJob(testDockerJob("new-request", "new-plan", "a", "b")); err != nil {
		t.Fatal("reservations not released", err)
	}
}

func TestDockerPlanReviewIsDeduplicatedAndDoesNotExecute(t *testing.T) {
	s := openTest(t)
	p, err := s.CreateDockerPlan(DockerPlan{Title: "Stop qa", Actor: "Agent", Input: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	expires, _ := time.Parse(time.RFC3339Nano, p.ExpiresAt)
	if time.Until(expires) < 4*time.Minute {
		t.Fatal("preview expiry missing")
	}
	first, err := s.RequestDockerReview(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.RequestDockerReview(p.ID)
	if err != nil || first.InboxID == "" || second.InboxID != first.InboxID {
		t.Fatal("duplicate review", err)
	}
	item, err := s.GetInboxItem(first.InboxID)
	if err != nil || item.Kind != InboxFYI {
		t.Fatal(err)
	}
	if _, err = s.SetInboxItemState(item.ID, InboxDone, nil); err != nil {
		t.Fatal(err)
	}
	jobs, err := s.DockerJobs()
	if err != nil || len(jobs) != 0 {
		t.Fatal("Inbox triage executed a job", err)
	}
}

func TestDockerHealthIncidentDecisionTable(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		states  []string
		offsets []int
		want    []string
	}{
		{"disabled", false, []string{"bad", "bad", "bad"}, nil, []string{}},
		{"consecutive", true, []string{"bad", "bad", "bad"}, nil, []string{"open"}},
		{"recovery", true, []string{"bad", "bad", "good", "good"}, nil, []string{"resolved"}},
		{"unknown is not healthy", true, []string{"bad", "bad", "unknown", "good"}, nil, []string{"open"}},
		{"unknown breaks pending", true, []string{"bad", "unknown", "bad"}, nil, []string{}},
		{"missing evidence stays open", true, []string{"bad", "bad", "missing", "missing"}, nil, []string{"open"}},
		{"rapid samples do not accelerate", true, []string{"bad", "bad", "bad"}, []int{0, 1, 2}, []string{}},
		{"reconnect breaks pending", true, []string{"bad", "bad"}, []int{0, 120}, []string{}},
		{"regression is a new incident", true, []string{"bad", "bad", "good", "good", "bad", "bad"}, nil, []string{"open", "resolved"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := openTest(t)
			m := DefaultDockerMonitor("unix:///tmp/qa", "demo")
			m.Enabled = tc.enabled
			m.BadSamples = 2
			m.IntervalSeconds = 30
			m, err := s.SaveDockerMonitor(m)
			if err != nil {
				t.Fatal(err)
			}
			base := time.Now().UTC()
			for i, state := range tc.states {
				offset := i * 30
				if tc.offsets != nil {
					offset = tc.offsets[i]
				}
				signals := []DockerSignal{{Key: "qa:health", Title: "qa health", State: state}}
				if state == "missing" {
					signals = nil
				}
				if err = s.RecordDockerHealth(m.Endpoint, m.Project, m.Revision, json.RawMessage(`{"containers":[]}`), signals, base.Add(time.Duration(offset)*time.Second)); err != nil {
					t.Fatal(err)
				}
			}
			incidents, err := s.DockerIncidents(m.Endpoint, m.Project)
			if err != nil || len(incidents) != len(tc.want) {
				t.Fatalf("incidents %+v %v", incidents, err)
			}
			for i, inc := range incidents {
				if inc.State != tc.want[i] {
					t.Fatalf("incidents %+v", incidents)
				}
			}
		})
	}
}

func TestDockerMonitorRevisionAndRetention(t *testing.T) {
	s := openTest(t)
	m := DefaultDockerMonitor("unix:///tmp/qa", "demo")
	m.Enabled = true
	m.BadSamples = 2
	m.IntervalSeconds = 30
	m, err := s.SaveDockerMonitor(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveDockerMonitor(DefaultDockerMonitor(m.Endpoint, m.Project)); !errors.Is(err, ErrDockerConflict) {
		t.Fatal("stale config accepted", err)
	}
	base := time.Now().UTC().Add(-9 * 24 * time.Hour)
	for i := 0; i < 2; i++ {
		err = s.RecordDockerHealth(m.Endpoint, m.Project, m.Revision, json.RawMessage(`{}`), []DockerSignal{{Key: "open", State: "bad"}, {Key: "old", State: "bad"}}, base.Add(time.Duration(i*30)*time.Second))
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 2; i < 4; i++ {
		err = s.RecordDockerHealth(m.Endpoint, m.Project, m.Revision, json.RawMessage(`{}`), []DockerSignal{{Key: "open", State: "unknown"}, {Key: "old", State: "good"}}, base.Add(time.Duration(i*30)*time.Second))
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = s.RecordDockerHealth(m.Endpoint, m.Project, m.Revision, json.RawMessage(`{}`), nil, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	incidents, _ := s.DockerIncidents(m.Endpoint, m.Project)
	if len(incidents) != 1 || incidents[0].Signal != "open" {
		t.Fatalf("retention %+v", incidents)
	}
	oldRevision := m.Revision
	m.Enabled = false
	m, err = s.SaveDockerMonitor(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordDockerHealth(m.Endpoint, m.Project, oldRevision, json.RawMessage(`{"late":true}`), nil, time.Now().UTC().Add(time.Minute)); !errors.Is(err, ErrDockerConflict) {
		t.Fatal("late sample after disable", err)
	}
	got, _ := s.DockerMonitor(m.Endpoint, m.Project)
	if string(got.Snapshot) != "{}" || got.Enabled {
		t.Fatalf("disabled monitor %+v", got)
	}
}
