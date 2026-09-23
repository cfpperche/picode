package server

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/apps"
	"github.com/cfpperche/picode/internal/store"
)

func missionPost(t *testing.T, ts *httptest.Server, path string, m any, status int) map[string]any {
	t.Helper()
	b, _ := json.Marshal(m)
	res, out := inboxPost(t, ts, path, string(b))
	if res.StatusCode != status {
		t.Fatalf("%s: %d %+v", path, res.StatusCode, out)
	}
	return out
}
func missionResult(t *testing.T, out map[string]any) store.Mission {
	t.Helper()
	b, _ := json.Marshal(out["mission"])
	var v store.Mission
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestMissionHTTPRoundTripAndIdentity(t *testing.T) {
	_, s := newInboxServer(t)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: s, Apps: apps.NewRegistry(apps.BuiltIns(false)...)}).Handler)
	t.Cleanup(ts.Close)
	dir := t.TempDir()
	w, _ := s.AddWorkspace("Mission", dir)
	a, _ := s.AddAgent(w.ID, "Executor", "")
	path := filepath.Join(dir, "session.jsonl")
	if _, err := s.UpdateAgent(a.ID, store.AgentPatch{SessionPath: &path}); err != nil {
		t.Fatal(err)
	}
	v := missionResult(t, missionPost(t, ts, "/api/missions", store.MissionMutation{RequestID: "create", WorkspaceID: w.ID, Title: "Portable task", Objective: "Persist work", Criteria: []store.MissionCriterion{{Text: "History survives"}}}, 200))
	action := func(name string, fields map[string]any) map[string]any {
		t.Helper()
		if fields == nil {
			fields = map[string]any{}
		}
		fields["action"] = name
		fields["requestId"] = name + "-" + string(rune(v.Version+65))
		fields["expectedVersion"] = v.Version
		out := missionPost(t, ts, "/api/missions/"+v.ID+"/actions", fields, 200)
		v = missionResult(t, out)
		return out
	}
	action("assign", map[string]any{"agentId": a.ID, "targetReady": true})
	missionPost(t, ts, "/api/missions/tool", map[string]any{"action": "show", "id": v.ID}, 403)
	missionPost(t, ts, "/api/missions/tool", map[string]any{"action": "show", "id": v.ID, "agent": a.ID, "term": "foreign"}, 403)
	missionPost(t, ts, "/api/missions/tool", map[string]any{"action": "accept", "id": v.ID, "agent": a.ID, "generation": 1, "expectedVersion": v.Version, "requestId": "forbidden"}, 403)
	tool := func(name string, extra map[string]any) {
		t.Helper()
		if extra == nil {
			extra = map[string]any{}
		}
		extra["action"] = name
		extra["id"] = v.ID
		extra["agent"] = a.ID
		extra["generation"] = 1
		extra["expectedVersion"] = v.Version
		extra["requestId"] = "agent-" + name
		v = missionResult(t, missionPost(t, ts, "/api/missions/tool", extra, 200))
	}
	tool("acknowledge", nil)
	tool("block", map[string]any{"note": "Should the history include decisions?"})
	item := v.InboxID
	missionPost(t, ts, "/api/apps/inbox/action", map[string]any{"action": "respond", "path": "item/" + item, "args": map[string]string{"reply": "Yes, keep each decision."}}, 200)
	v, _ = s.GetMission(v.ID)
	if v.State != "in-progress" || v.Context == "" {
		t.Fatal(v)
	}
	tool("evidence", map[string]any{"evidence": map[string]string{"criterionId": v.Criteria[0].ID, "kind": "note", "value": "Reload preserved the history.", "outcome": "pass"}})
	tool("request-review", nil)
	action("accept", map[string]any{"sourceStopped": true})
	if v.State != "completed" || v.Assignment.Reserved {
		t.Fatal(v)
	}
}
func TestMissionEvidenceConfinement(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	_ = os.WriteFile(outside, []byte("outside"), 0600)
	_ = os.WriteFile(filepath.Join(root, "evidence"), []byte("one"), 0600)
	_ = os.Symlink(outside, filepath.Join(root, "link"))
	for _, name := range []string{"../outside", outside, "link", "."} {
		if _, err := missionFileDigest(root, name); err == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	first, err := missionFileDigest(root, "evidence")
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(root, "evidence"), []byte("two"), 0600)
	second, _ := missionFileDigest(root, "evidence")
	if first == second {
		t.Fatal("content changes were not detected")
	}
}

func TestMissionCanStartInUnbornRepository(t *testing.T) {
	ts, s := newInboxServer(t)
	dir := t.TempDir()
	gitOut(t, dir, "init")
	w, _ := s.AddWorkspace("New repository", dir)
	v := missionResult(t, missionPost(t, ts, "/api/missions", store.MissionMutation{RequestID: "unborn", WorkspaceID: w.ID, Title: "First implementation", Objective: "Start from an empty repository", Criteria: []store.MissionCriterion{{Text: "Implemented"}}}, 200))
	if v.Repository == "" {
		t.Fatal("lost repository identity")
	}
	missionPost(t, ts, "/api/missions/"+v.ID+"/actions", store.MissionMutation{Action: "evidence", RequestID: "unborn-evidence", ExpectedVersion: v.Version, Evidence: &store.MissionEvidence{CriterionID: v.Criteria[0].ID, Kind: "note", Value: "No commit", Outcome: "pass"}}, 409)
}

func TestMissionTransferRequiresPreparedCleanWorkingFolder(t *testing.T) {
	ts, s := newInboxServer(t)
	dir := gitRepo(t)
	other := filepath.Join(t.TempDir(), "other")
	gitOut(t, dir, "worktree", "add", "--detach", other)
	w, _ := s.AddWorkspace("Transfer", dir)
	a, _ := s.AddAgent(w.ID, "Source", "")
	b, _ := s.AddAgent(w.ID, "Target", other)
	v := missionResult(t, missionPost(t, ts, "/api/missions", store.MissionMutation{RequestID: "transfer-create", WorkspaceID: w.ID, Title: "Transfer", Objective: "Keep candidate identity", Criteria: []store.MissionCriterion{{Text: "Same candidate"}}}, 200))
	v = missionResult(t, missionPost(t, ts, "/api/missions/"+v.ID+"/actions", store.MissionMutation{Action: "assign", RequestID: "source", ExpectedVersion: v.Version, AgentID: a.ID, TargetReady: true}, 200))
	m := store.MissionMutation{Action: "assign", RequestID: "target", ExpectedVersion: v.Version, AgentID: b.ID, TargetReady: true, SourceStopped: true}
	missionPost(t, ts, "/api/missions/"+v.ID+"/actions", m, 409)
	m.FilesReady = true
	_ = os.WriteFile(filepath.Join(dir, "a"), []byte("dirty candidate"), 0600)
	missionPost(t, ts, "/api/missions/"+v.ID+"/actions", m, 409)
	_ = os.WriteFile(filepath.Join(dir, "a"), []byte("a"), 0600)
	v = missionResult(t, missionPost(t, ts, "/api/missions/"+v.ID+"/actions", m, 200))
	if v.Assignment.AgentID != b.ID {
		t.Fatal(v.Assignment)
	}
}

func TestMissionDeliveryReferenceDoesNotTransferAuthorship(t *testing.T) {
	ts, s := newInboxServer(t)
	dir := gitRepo(t)
	w, _ := s.AddWorkspace("Delivery", dir)
	v := missionResult(t, missionPost(t, ts, "/api/missions", store.MissionMutation{RequestID: "delivery-create", WorkspaceID: w.ID, Title: "Review result", Objective: "Link existing work", Criteria: []store.MissionCriterion{{Text: "Candidate matches"}}}, 200))
	rev := strings.TrimSpace(gitOut(t, dir, "rev-parse", "HEAD"))
	d, e := s.ApplyDelivery(v.Repository, "previous-executor", store.DeliveryMutation{Action: "register", RequestID: "delivery", Title: "Existing work", Branch: "main", Revision: rev, Target: "main"})
	if e != nil {
		t.Fatal(e)
	}
	missionPost(t, ts, "/api/missions/"+v.ID+"/actions", store.MissionMutation{Action: "evidence", RequestID: "link", ExpectedVersion: v.Version, Evidence: &store.MissionEvidence{CriterionID: v.Criteria[0].ID, Kind: "delivery", Value: d.ID, Outcome: "pass"}}, 200)
	if _, err := s.ApplyDelivery(v.Repository, "next-executor", store.DeliveryMutation{Action: "request-review", ID: d.ID, RequestID: "foreign-author", ExpectedVersion: d.Version}); err == nil {
		t.Fatal("mission link transferred delivery authorship")
	}
	got, _ := s.GetDelivery(v.Repository, d.ID)
	if got.Version != d.Version {
		t.Fatal("mission link mutated delivery")
	}
}
