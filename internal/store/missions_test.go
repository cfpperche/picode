package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func missionFixture(t *testing.T, s *Store) (Mission, Agent) {
	t.Helper()
	w, e := s.AddWorkspace("Missions", t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	a, e := s.AddAgent(w.ID, "Executor", "")
	if e != nil {
		t.Fatal(e)
	}
	m := MissionMutation{Action: "create", RequestID: "create-" + a.ID, WorkspaceID: w.ID, Title: "Recover a task", Objective: "Preserve the result across sessions", Criteria: []MissionCriterion{{Text: "History survives"}}}
	v, e := s.ApplyMission(MissionOwner, m, MissionObservation{})
	if e != nil {
		t.Fatal(e)
	}
	return v, a
}

func missionChange(t *testing.T, s *Store, v Mission, action string, edit func(*MissionMutation), o MissionObservation) Mission {
	t.Helper()
	m := MissionMutation{Action: action, ID: v.ID, ExpectedVersion: v.Version, RequestID: fmt.Sprintf("%s-%s-%d", v.ID, action, v.Version)}
	if edit != nil {
		edit(&m)
	}
	v, e := s.ApplyMission(MissionOwner, m, o)
	if e != nil {
		t.Fatal(action, e)
	}
	return v
}

func TestMissionRetryVersionAndHistory(t *testing.T) {
	s := openTest(t)
	w, _ := s.AddWorkspace("M", t.TempDir())
	m := MissionMutation{Action: "create", RequestID: "one", WorkspaceID: w.ID, Title: "Task", Objective: "Result", Criteria: []MissionCriterion{{Text: "Works"}}}
	v, e := s.ApplyMission(MissionOwner, m, MissionObservation{})
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.ApplyMission(MissionOwner, m, MissionObservation{})
	if e != nil || again.ID != v.ID || again.Version != 1 {
		t.Fatalf("retry %+v %v", again, e)
	}
	m.Title = "different"
	if _, e = s.ApplyMission(MissionOwner, m, MissionObservation{}); !errors.Is(e, ErrMissionConflict) {
		t.Fatal(e)
	}
	v = missionChange(t, s, v, "report", func(m *MissionMutation) { m.Note = "Checkpoint" }, MissionObservation{})
	_, e = s.ApplyMission(MissionOwner, MissionMutation{Action: "pause", ID: v.ID, ExpectedVersion: 1, RequestID: "stale"}, MissionObservation{})
	if !errors.Is(e, ErrMissionConflict) {
		t.Fatal(e)
	}
	h, e := s.MissionHistory(v.ID, 0)
	if e != nil || len(h) != 2 || h[0].Note != "Checkpoint" {
		t.Fatalf("history %+v %v", h, e)
	}
	events, _ := s.EventsOfType("mission.changed", 100)
	if len(events) != 2 {
		t.Fatalf("events %d", len(events))
	}
}

func TestMissionAssignmentDecisionTable(t *testing.T) {
	s := openTest(t)
	v, a := missionFixture(t, s)
	o := MissionObservation{Agent: &a, Session: "codex:one", WorkPath: t.TempDir()}
	v = missionChange(t, s, v, "assign", func(m *MissionMutation) { m.AgentID = a.ID; m.TargetReady = true }, o)
	for _, row := range []struct {
		name, actor, action, session string
		generation                   int
		want                         error
	}{
		{"foreign", "other", "report", "codex:one", 1, ErrMissionForbidden},
		{"generation", a.ID, "report", "codex:one", 2, ErrMissionForbidden},
		{"session rollover", a.ID, "report", "codex:two", 1, ErrMissionForbidden},
		{"agent cannot accept", a.ID, "accept", "codex:one", 1, ErrMissionForbidden},
		{"agent cannot assign", a.ID, "assign", "codex:one", 1, ErrMissionForbidden},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := s.ApplyMission(row.actor, MissionMutation{Action: row.action, ID: v.ID, ExpectedVersion: v.Version, RequestID: row.name, Generation: row.generation, Note: "work"}, MissionObservation{Session: row.session})
			if !errors.Is(err, row.want) {
				t.Fatalf("%v", err)
			}
		})
	}
	v = missionChange(t, s, v, "dispatch", nil, o)
	_, e := s.ApplyMission(MissionOwner, MissionMutation{Action: "dispatch", ID: v.ID, ExpectedVersion: v.Version, RequestID: "duplicate-send"}, o)
	if e == nil {
		t.Fatal("sent twice")
	}
	v = missionChange(t, s, v, "pause", nil, o)
	if !v.Assignment.Reserved || v.Assignment.Delivery != "unconfirmed" {
		t.Fatal("pause released uncertain writer")
	}
	_, e = s.ApplyMission(a.ID, MissionMutation{Action: "acknowledge", ID: v.ID, ExpectedVersion: v.Version, RequestID: "late", Generation: 1}, o)
	if e == nil {
		t.Fatal("late acknowledgement reactivated paused mission")
	}
	v = missionChange(t, s, v, "cancel", nil, o)
	if !v.Assignment.Reserved {
		t.Fatal("cancel released uncertain writer")
	}
	_, e = s.ApplyMission(MissionOwner, MissionMutation{Action: "reopen", ID: v.ID, ExpectedVersion: v.Version, RequestID: "reopen"}, o)
	if e == nil {
		t.Fatal("reopen with uncertain writer")
	}
	v = missionChange(t, s, v, "release", func(m *MissionMutation) { m.SourceStopped = true }, o)
	v = missionChange(t, s, v, "reopen", nil, o)
	if v.State != "ready" {
		t.Fatal(v.State)
	}
}

func TestMissionConcurrentReservationAndTransfer(t *testing.T) {
	s := openTest(t)
	one, a := missionFixture(t, s)
	w, _ := s.GetWorkspace(one.WorkspaceID)
	two, e := s.ApplyMission(MissionOwner, MissionMutation{Action: "create", RequestID: "second", WorkspaceID: w.ID, Title: "Second", Objective: "Second result", Criteria: []MissionCriterion{{Text: "Works"}}}, MissionObservation{})
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, v := range []Mission{one, two} {
		wg.Add(1)
		go func(v Mission) {
			defer wg.Done()
			_, err := s.ApplyMission(MissionOwner, MissionMutation{Action: "assign", ID: v.ID, ExpectedVersion: v.Version, RequestID: v.ID, AgentID: a.ID, TargetReady: true}, MissionObservation{Agent: &a, WorkPath: w.Path})
			results <- err
		}(v)
	}
	wg.Wait()
	close(results)
	wins := 0
	for err := range results {
		if err == nil {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("reserved %d times", wins)
	}
	rows, _ := s.ListMissions(w.ID, 0)
	var v Mission
	for _, r := range rows {
		if r.Assignment != nil {
			v = r
		}
	}
	b, _ := s.AddAgent(w.ID, "Replacement", "")
	m := MissionMutation{Action: "assign", ID: v.ID, ExpectedVersion: v.Version, RequestID: "transfer", AgentID: b.ID, TargetReady: true}
	if _, e = s.ApplyMission(MissionOwner, m, MissionObservation{Agent: &b, WorkPath: w.Path}); e == nil {
		t.Fatal("transferred without source confirmation")
	}
	m.SourceStopped = true
	v, e = s.ApplyMission(MissionOwner, m, MissionObservation{Agent: &b, Session: "new", WorkPath: w.Path})
	if e != nil || v.Assignment.Generation != 2 {
		t.Fatal(v, e)
	}
	if _, e = s.ApplyMission(a.ID, MissionMutation{Action: "report", ID: v.ID, ExpectedVersion: v.Version, RequestID: "late-old", Generation: 1, Note: "old result"}, MissionObservation{}); !errors.Is(e, ErrMissionForbidden) {
		t.Fatal(e)
	}
}

func TestMissionReviewEvidenceAndAcceptance(t *testing.T) {
	s := openTest(t)
	v, a := missionFixture(t, s)
	o := MissionObservation{Agent: &a, Session: "one", WorkPath: t.TempDir()}
	v = missionChange(t, s, v, "assign", func(m *MissionMutation) { m.AgentID = a.ID; m.TargetReady = true }, o)
	v = missionChange(t, s, v, "acknowledge", func(m *MissionMutation) { m.TargetReady = true }, o)
	if _, e := s.ApplyMission(MissionOwner, MissionMutation{Action: "request-review", ID: v.ID, ExpectedVersion: v.Version, RequestID: "no-evidence"}, o); e == nil {
		t.Fatal("review without evidence")
	}
	v = missionChange(t, s, v, "evidence", func(m *MissionMutation) {
		m.Evidence = &MissionEvidence{CriterionID: v.Criteria[0].ID, Kind: "note", Value: "Observed history after restart", Outcome: "pass"}
	}, o)
	v = missionChange(t, s, v, "request-review", nil, o)
	if v.InboxID == "" {
		t.Fatal("missing review notification")
	}
	v = missionChange(t, s, v, "changes", func(m *MissionMutation) { m.Note = "Include another restart" }, o)
	if v.State != "in-progress" {
		t.Fatal(v.State)
	}
	v = missionChange(t, s, v, "evidence", func(m *MissionMutation) {
		m.Evidence = &MissionEvidence{CriterionID: v.Criteria[0].ID, Kind: "note", Value: "Repeated restart passed", Outcome: "pass"}
	}, o)
	v = missionChange(t, s, v, "request-review", nil, o)
	v = missionChange(t, s, v, "accept", func(m *MissionMutation) { m.SourceStopped = true }, o)
	if v.State != "completed" || v.AcceptedAt == "" || v.Assignment.Reserved {
		t.Fatal(v)
	}
	if _, e := s.ApplyMission(MissionOwner, MissionMutation{Action: "report", ID: v.ID, ExpectedVersion: v.Version, RequestID: "after", Note: "changed"}, o); e == nil {
		t.Fatal("changed accepted result without reopening")
	}
	h, _ := s.MissionHistory(v.ID, 0)
	if h[0].Snapshot.AcceptedAt == "" {
		t.Fatal("acceptance not preserved")
	}
}

func TestMissionSurvivesRestartAndAgentRemoval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missions.db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	v, a := missionFixture(t, s)
	v = missionChange(t, s, v, "assign", func(m *MissionMutation) { m.AgentID = a.ID; m.TargetReady = true }, MissionObservation{Agent: &a, WorkPath: t.TempDir()})
	v = missionChange(t, s, v, "dispatch", nil, MissionObservation{})
	if e = s.DeleteAgent(a.ID); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	got, e := s.GetMission(v.ID)
	if e != nil || got.Version != v.Version || got.Assignment.AgentID != a.ID || got.Assignment.Delivery != "unconfirmed" {
		t.Fatal(got, e)
	}
	h, e := s.MissionHistory(v.ID, 0)
	if e != nil || len(h) != 3 {
		t.Fatal(h, e)
	}
}

func TestMissionReviewRevisionDecisionTable(t *testing.T) {
	v := Mission{Repository: "repo", ScopeVersion: 1, Criteria: []MissionCriterion{{ID: "c", Text: "Test passes"}}, Evidence: []MissionEvidence{{CriterionID: "c", ScopeVersion: 1, Outcome: "pass", Revision: "one"}}}
	for _, row := range []struct {
		name string
		o    MissionObservation
		ok   bool
	}{
		{"current", MissionObservation{Revision: "one", Clean: true}, true},
		{"dirty", MissionObservation{Revision: "one", Clean: false}, false},
		{"new revision", MissionObservation{Revision: "two", Clean: true}, false},
		{"unknown", MissionObservation{}, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			if e := missionReviewEvidence(v, row.o); (e == nil) != row.ok {
				t.Fatal(e)
			}
		})
	}
	v.ScopeVersion++
	if e := missionReviewEvidence(v, MissionObservation{Revision: "one", Clean: true}); e == nil {
		t.Fatal("accepted stale scope")
	}
}

func TestMissionInboxAnswerAfterTransfer(t *testing.T) {
	s := openTest(t)
	v, a := missionFixture(t, s)
	o := MissionObservation{Agent: &a, Session: "one", WorkPath: t.TempDir()}
	v = missionChange(t, s, v, "assign", func(m *MissionMutation) { m.AgentID = a.ID; m.TargetReady = true }, o)
	v = missionChange(t, s, v, "acknowledge", func(m *MissionMutation) { m.TargetReady = true }, o)
	v = missionChange(t, s, v, "block", func(m *MissionMutation) { m.Note = "Choose a format" }, o)
	question := v.InboxID
	b, _ := s.AddAgent(v.WorkspaceID, "Next", "")
	o.Agent = &b
	o.Session = "two"
	v = missionChange(t, s, v, "assign", func(m *MissionMutation) { m.AgentID = b.ID; m.TargetReady = true; m.SourceStopped = true }, o)
	v = missionChange(t, s, v, "inbox-answer", func(m *MissionMutation) { m.InboxID = question; m.Note = "Old answer" }, o)
	if v.State != "ready" || v.Context != "" {
		t.Fatal("old answer changed current work", v)
	}
	it, e := s.GetInboxItem(question)
	if e != nil || it.Response == nil || *it.Response != "Old answer" {
		t.Fatal(it, e)
	}
	h, _ := s.MissionHistory(v.ID, 0)
	if h[0].Note != "Old answer" {
		t.Fatal("lost answer provenance")
	}
}

func TestMissionCapacityAllowsExecutorRecovery(t *testing.T) {
	s := openTest(t)
	v, a := missionFixture(t, s)
	v = missionChange(t, s, v, "assign", func(m *MissionMutation) { m.AgentID = a.ID; m.TargetReady = true }, MissionObservation{Agent: &a, WorkPath: t.TempDir()})
	if _, err := s.db.Exec(`UPDATE missions SET body=json_set(body,'$.version',2000) WHERE id=?`, v.ID); err != nil {
		t.Fatal(err)
	}
	v, _ = s.GetMission(v.ID)
	if _, err := s.ApplyMission(MissionOwner, MissionMutation{Action: "report", ID: v.ID, RequestID: "limit", ExpectedVersion: v.Version, Note: "work"}, MissionObservation{}); !errors.Is(err, ErrMissionCapacity) {
		t.Fatal(err)
	}
	v = missionChange(t, s, v, "cancel", nil, MissionObservation{})
	v = missionChange(t, s, v, "release", func(m *MissionMutation) { m.SourceStopped = true }, MissionObservation{})
	v = missionChange(t, s, v, "archive", func(m *MissionMutation) { m.Archived = true }, MissionObservation{})
	if s.AgentMissionID(a.ID) != "" || !v.Archived {
		t.Fatal("capacity trapped the executor")
	}
}

func TestMissionRelinkKeepsGenerationAndInvalidatesEvidence(t *testing.T) {
	s := openTest(t)
	v, a := missionFixture(t, s)
	v = missionChange(t, s, v, "assign", func(m *MissionMutation) { m.AgentID = a.ID; m.TargetReady = true }, MissionObservation{Agent: &a, Session: "old", WorkPath: t.TempDir()})
	oldScope := v.ScopeVersion
	v = missionChange(t, s, v, "release", func(m *MissionMutation) { m.SourceStopped = true }, MissionObservation{})
	w, _ := s.AddWorkspace("Restored", t.TempDir())
	v = missionChange(t, s, v, "relink", func(m *MissionMutation) { m.WorkspaceID = w.ID; m.FilesReady = true }, MissionObservation{})
	b, _ := s.AddAgent(w.ID, "Replacement", "")
	v = missionChange(t, s, v, "assign", func(m *MissionMutation) { m.AgentID = b.ID; m.TargetReady = true }, MissionObservation{Agent: &b, Session: "new", WorkPath: w.Path})
	if v.Assignment.Generation != 2 || v.ScopeVersion <= oldScope || v.WorkspaceID != w.ID {
		t.Fatal(v)
	}
	rows, _ := s.ListMissions(w.ID, 0)
	if len(rows) != 1 {
		t.Fatal("workspace index not relinked")
	}
}
func TestMissionDirtyCandidateCannotProduceEvidence(t *testing.T) {
	s := openTest(t)
	w, _ := s.AddWorkspace("Repo", t.TempDir())
	v, e := s.ApplyMission(MissionOwner, MissionMutation{Action: "create", RequestID: "repo", WorkspaceID: w.ID, Title: "Task", Objective: "Outcome", Criteria: []MissionCriterion{{Text: "Passing result"}}}, MissionObservation{Repository: "repo"})
	if e != nil {
		t.Fatal(e)
	}
	for _, o := range []MissionObservation{{Revision: "head", Clean: false}, {Clean: true}} {
		_, e = s.ApplyMission(MissionOwner, MissionMutation{Action: "evidence", ID: v.ID, RequestID: "dirty", ExpectedVersion: v.Version, Evidence: &MissionEvidence{CriterionID: v.Criteria[0].ID, Kind: "note", Value: "Passing tests on dirty files", Outcome: "pass"}}, o)
		if e == nil {
			t.Fatal("dirty or unborn candidate produced revision-bound evidence")
		}
	}
}

func TestMissionConcurrentPauseAndDispatchOrAcknowledgement(t *testing.T) {
	for _, action := range []string{"dispatch", "acknowledge"} {
		t.Run(action, func(t *testing.T) {
			s := openTest(t)
			v, a := missionFixture(t, s)
			o := MissionObservation{Agent: &a, Session: "one", WorkPath: t.TempDir()}
			v = missionChange(t, s, v, "assign", func(m *MissionMutation) { m.AgentID = a.ID; m.TargetReady = true }, o)
			var wg sync.WaitGroup
			start := make(chan struct{})
			results := make(chan error, 2)
			for _, op := range []string{"pause", action} {
				wg.Add(1)
				go func(op string) {
					defer wg.Done()
					<-start
					actor := MissionOwner
					if op == "acknowledge" {
						actor = a.ID
					}
					_, err := s.ApplyMission(actor, MissionMutation{Action: op, ID: v.ID, ExpectedVersion: v.Version, RequestID: op, Generation: 1}, o)
					results <- err
				}(op)
			}
			close(start)
			wg.Wait()
			close(results)
			success, conflict := 0, 0
			for e := range results {
				if e == nil {
					success++
				} else if errors.Is(e, ErrMissionConflict) {
					conflict++
				} else {
					t.Fatal(e)
				}
			}
			if success != 1 || conflict != 1 {
				t.Fatalf("success=%d conflict=%d", success, conflict)
			}
			v, _ = s.GetMission(v.ID)
			if v.State != "paused" {
				v = missionChange(t, s, v, "pause", nil, o)
			}
			if !v.Assignment.Reserved {
				t.Fatal("pause released a possible writer")
			}
			_, err := s.ApplyMission(a.ID, MissionMutation{Action: "acknowledge", ID: v.ID, ExpectedVersion: v.Version, Generation: 1, RequestID: "late-ack"}, o)
			if err == nil {
				t.Fatal("late acknowledgement resumed paused work")
			}
		})
	}
}

func TestMissionNewGitIdentityInvalidatesPlainFolderEvidence(t *testing.T) {
	s := openTest(t)
	v, a := missionFixture(t, s)
	o := MissionObservation{Agent: &a, Session: "session", WorkPath: t.TempDir()}
	v = missionChange(t, s, v, "assign", func(m *MissionMutation) { m.AgentID = a.ID; m.TargetReady = true }, o)
	v = missionChange(t, s, v, "acknowledge", func(m *MissionMutation) { m.TargetReady = true }, o)
	add := func(m *MissionMutation) {
		m.Evidence = &MissionEvidence{CriterionID: v.Criteria[0].ID, Kind: "note", Value: "Observed result", Outcome: "pass"}
	}
	v = missionChange(t, s, v, "evidence", add, o)
	v = missionChange(t, s, v, "request-review", nil, o)
	o.Repository = "new-repo"
	o.Revision = "new-head"
	o.Clean = false
	for _, action := range []string{"evidence", "accept"} {
		m := MissionMutation{Action: action, ID: v.ID, ExpectedVersion: v.Version, RequestID: "git-init-" + action, SourceStopped: true}
		if action == "evidence" {
			add(&m)
		}
		if _, err := s.ApplyMission(MissionOwner, m, o); err == nil {
			t.Fatal("new dirty repository bypassed evidence identity", action)
		}
	}
	o.Clean = true
	v = missionChange(t, s, v, "evidence", add, o)
	if v.Repository != "new-repo" || v.ScopeVersion != 2 || v.Evidence[0].ScopeVersion == v.Evidence[1].ScopeVersion {
		t.Fatal(v)
	}
	v = missionChange(t, s, v, "request-review", nil, o)
	if v.ReviewRevision != "new-head" {
		t.Fatal("review did not bind the new candidate")
	}
}

func TestMissionContextIncludesBoundedAttemptsAndEvidence(t *testing.T) {
	v := Mission{ID: "mission-test", Title: "Task", Objective: "Outcome", Version: 12, Evidence: []MissionEvidence{{ID: "e1", CriterionID: "c1", Kind: "file", Value: "result.md"}}}
	history := []MissionHistory{{Version: 13, Action: "report", Note: "future-note"}}
	for i := 12; i > 0; i-- {
		history = append(history, MissionHistory{Version: i, Action: "report", Note: strings.Repeat("x", 2000)})
	}
	packet := MissionContext(v, history...)
	for _, want := range []string{"result.md", "v12 report", "v5 report", "[excerpt]", "Omissions:", "Packet SHA-256:"} {
		if !strings.Contains(packet, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(packet, "future-note") || strings.Contains(packet, "v4 report") || len(packet) > 120<<10 {
		t.Fatal("packet bounds violated")
	}
	if packet != MissionContext(v, history...) {
		t.Fatal("packet must be deterministic")
	}
}
