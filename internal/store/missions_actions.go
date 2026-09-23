package store

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"
)

func applyMissionAction(tx *sql.Tx, v *Mission, actor string, m MissionMutation, o MissionObservation) error {
	terminal := v.State == "completed" || v.State == "cancelled"
	if terminal && m.Action != "reopen" && m.Action != "archive" && m.Action != "release" {
		return errors.New("reopen the mission before changing it")
	}
	switch m.Action {
	case "create", "edit":
		if actor != MissionOwner {
			return ErrMissionForbidden
		}
		if v.Assignment != nil && v.Assignment.Reserved {
			return errors.New("release the current assignment before changing the objective")
		}
		if strings.TrimSpace(m.Title) == "" || len(m.Title) > 200 || strings.ContainsAny(m.Title, "\r\n") || strings.TrimSpace(m.Objective) == "" || len(m.Objective) > 8192 {
			return errors.New("provide a title and objective within the text limits")
		}
		if len(m.Criteria) == 0 {
			return errors.New("add at least one acceptance criterion")
		}
		m.Criteria = append([]MissionCriterion(nil), m.Criteria...)
		seen := map[string]bool{}
		for i, c := range m.Criteria {
			if strings.TrimSpace(c.Text) == "" || len(c.Text) > 1000 {
				return errors.New("each criterion needs text of at most 1000 bytes")
			}
			if c.ID == "" {
				m.Criteria[i].ID = "criterion_" + rand.Text()
			}
			if len(m.Criteria[i].ID) > 128 || seen[m.Criteria[i].ID] {
				return errors.New("criterion IDs must be unique")
			}
			seen[m.Criteria[i].ID] = true
		}
		v.Title = strings.TrimSpace(m.Title)
		v.Objective = m.Objective
		v.Context = m.Context
		v.Criteria = m.Criteria
		v.NextAction = m.NextAction
		v.ReviewRevision = ""
		v.ReviewScope = 0
		v.State = "ready"
		v.Blocker = ""
		if m.Action == "edit" {
			v.ScopeVersion++
		}
	case "relink":
		if actor != MissionOwner || (v.Assignment != nil && v.Assignment.Reserved) {
			return errors.New("release the previous executor before restoring the workspace")
		}
		var w Workspace
		if err := tx.QueryRow(`SELECT id,name,path FROM workspaces WHERE id=?`, m.WorkspaceID).Scan(&w.ID, &w.Name, &w.Path); err != nil || IsFree(w) {
			return errors.New("choose an existing workspace")
		}
		if (v.Repository != "" && o.Repository != v.Repository) || (v.Repository == "" && w.Path != v.WorkspacePath && !m.FilesReady) {
			return errors.New("restore the original repository or confirm the prepared working files")
		}
		if v.Assignment != nil {
			v.LastGeneration = max(v.LastGeneration, v.Assignment.Generation)
		}
		v.WorkspaceID, v.WorkspaceName, v.WorkspacePath = w.ID, w.Name, w.Path
		if v.Repository == "" {
			v.Repository = o.Repository
		}
		v.Assignment = nil
		v.State = "ready"
		v.ReviewRevision = ""
		v.ReviewScope = 0
		v.ScopeVersion++
	case "assign":
		if actor != MissionOwner {
			return ErrMissionForbidden
		}
		if !m.TargetReady {
			return errors.New("confirm the target agent is ready and has no unrelated input draft")
		}
		if o.Agent == nil || o.Agent.ID != m.AgentID || o.Agent.WorkspaceID != v.WorkspaceID {
			return errors.New("choose an agent in this workspace")
		}
		generation := v.LastGeneration + 1
		if v.Assignment != nil {
			generation = max(generation, v.Assignment.Generation+1)
			if v.Assignment.Reserved && !m.SourceStopped {
				return errors.New("confirm the current agent and its child processes have stopped writing")
			}
		}
		var held string
		err := tx.QueryRow(`SELECT mission_id FROM mission_reservations WHERE agent_id=?`, m.AgentID).Scan(&held)
		if err == nil && held != v.ID {
			return errors.New("this agent already has a mission")
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err = tx.Exec(`DELETE FROM mission_reservations WHERE mission_id=?`, v.ID); err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO mission_reservations(mission_id,agent_id) VALUES(?,?)`, v.ID, m.AgentID); err != nil {
			return err
		}
		v.LastGeneration = generation
		v.Assignment = &MissionAssignment{AgentID: m.AgentID, AgentName: o.Agent.Name, CLI: o.Agent.CLI, Session: o.Session, WorkPath: o.WorkPath, Generation: generation, Reserved: true, Delivery: "prepared", StartedAt: nowUTC()}
		v.State = "ready"
		v.Blocker = ""
		v.ReviewRevision = ""
		v.ReviewScope = 0
	case "dispatch":
		if actor != MissionOwner {
			return ErrMissionForbidden
		}
		if v.State != "ready" || v.Assignment == nil || !v.Assignment.Reserved || v.Assignment.Delivery != "prepared" {
			return errors.New("this assignment cannot be sent again; check the agent before recovery")
		}
		v.Assignment.Delivery = "unconfirmed"
	case "acknowledge":
		if v.Assignment == nil || !v.Assignment.Reserved {
			return errors.New("assign an agent first")
		}
		if v.State != "ready" && v.State != "in-progress" {
			return errors.New("resume the mission before acknowledging work")
		}
		if actor == MissionOwner && !m.TargetReady {
			return errors.New("confirm receipt in the target agent")
		}
		if o.Session != "" && v.Assignment.Session == "" {
			v.Assignment.Session = o.Session
		}
		v.Assignment.Delivery = "acknowledged"
		v.Assignment.AcknowledgedBy = actor
		v.State = "in-progress"
	case "report", "block":
		v.LastUpdate = m.Note
		if strings.TrimSpace(m.Note) == "" {
			return errors.New("provide an update")
		}
		if m.NextAction != "" {
			v.NextAction = m.NextAction
		}
		if m.Action == "block" {
			v.Blocker = m.Note
			v.State = "blocked"
		}
	case "decision":
		if actor != MissionOwner || strings.TrimSpace(m.Note) == "" {
			return errors.New("provide the owner's decision")
		}
		entry := "\n\nOwner decision: " + m.Note
		if len(v.Context)+len(entry) > 32768 {
			return errors.New("context limit reached; shorten the existing context before adding a decision")
		}
		v.Context += entry
		v.Blocker = ""
		v.NextAction = m.NextAction
		if v.State == "blocked" {
			v.State = "in-progress"
		}
	case "evidence":
		if v.Repository != "" && (!o.Clean || o.Revision == "") {
			return errors.New("commit the candidate and keep its working folder clean before adding evidence")
		}
		if m.Evidence == nil {
			return errors.New("provide evidence")
		}
		e := *m.Evidence
		if len(v.Evidence) >= 128 {
			return errors.New("evidence limit reached")
		}
		found := false
		for _, c := range v.Criteria {
			if c.ID == e.CriterionID {
				found = true
			}
		}
		if !found || strings.TrimSpace(e.Value) == "" || len(e.Value) > 8192 {
			return errors.New("evidence needs a current criterion and at most 8192 bytes")
		}
		if e.Kind != "note" && e.Kind != "file" && e.Kind != "delivery" {
			return errors.New("evidence kind must be note, file or delivery")
		}
		if e.Outcome != "pass" && e.Outcome != "fail" {
			return errors.New("evidence outcome must be pass or fail")
		}
		e.ID = "evidence_" + rand.Text()
		e.At = nowUTC()
		e.Actor = actor
		e.ScopeVersion = v.ScopeVersion
		e.Revision = o.Revision
		e.Digest = o.EvidenceDigest
		if e.Kind == "note" {
			e.Digest = missionDigest(e.Value)
		}
		v.Evidence = append(v.Evidence, e)
		if v.State == "in-review" {
			v.State = "in-progress"
			v.ReviewScope = 0
			v.ReviewRevision = ""
		}
	case "request-review":
		if v.State != "in-progress" && v.State != "blocked" {
			return errors.New("start the mission before requesting review")
		}
		if err := missionReviewEvidence(*v, o); err != nil {
			return err
		}
		v.ReviewRevision = o.Revision
		v.ReviewScope = v.ScopeVersion
		v.State = "in-review"
		v.Blocker = ""
	case "accept":
		if actor != MissionOwner {
			return ErrMissionForbidden
		}
		if v.State != "in-review" || v.ReviewScope != v.ScopeVersion || v.ReviewRevision != o.Revision {
			return errors.New("review changed; request a new review")
		}
		if err := missionReviewEvidence(*v, o); err != nil {
			return err
		}
		if v.Assignment != nil && v.Assignment.Reserved && !m.SourceStopped {
			return errors.New("confirm the executor has stopped writing before accepting")
		}
		if err := releaseMissionReservation(tx, v); err != nil {
			return err
		}
		v.State = "completed"
		v.AcceptedAt = nowUTC()
		v.NextAction = ""
	case "changes":
		if actor != MissionOwner || v.State != "in-review" || strings.TrimSpace(m.Note) == "" {
			return errors.New("provide the changes needed for this review")
		}
		entry := "\n\nChanges requested: " + m.Note
		if len(v.Context)+len(entry) > 32768 {
			return errors.New("context limit reached")
		}
		v.Context += entry
		v.ScopeVersion++
		v.State = "in-progress"
		v.Blocker = m.Note
		v.NextAction = m.Note
		v.ReviewScope = 0
		v.ReviewRevision = ""
	case "pause":
		if actor != MissionOwner {
			return ErrMissionForbidden
		}
		v.State = "paused"
	case "resume":
		if actor != MissionOwner || v.State != "paused" {
			return errors.New("only a paused mission can resume")
		}
		v.State = "ready"
		if v.Assignment != nil && v.Assignment.Reserved && v.Assignment.Delivery == "acknowledged" {
			v.State = "in-progress"
		}
	case "cancel":
		if actor != MissionOwner {
			return ErrMissionForbidden
		}
		v.State = "cancelled"
	case "release":
		if actor != MissionOwner || !m.SourceStopped {
			return errors.New("confirm the executor and its children have stopped writing")
		}
		if err := releaseMissionReservation(tx, v); err != nil {
			return err
		}
		if !terminal {
			v.State = "ready"
		}
	case "reopen":
		if actor != MissionOwner || !terminal {
			return errors.New("only a completed or cancelled mission can reopen")
		}
		if v.Assignment != nil && v.Assignment.Reserved {
			return errors.New("resolve the previous executor before reopening")
		}
		v.State = "ready"
		v.Archived = false
		v.ReviewScope = 0
		v.ReviewRevision = ""
		v.AcceptedAt = ""
	case "archive":
		if actor != MissionOwner || !terminal || (v.Assignment != nil && v.Assignment.Reserved) {
			return errors.New("complete or cancel the mission and release its executor before archiving")
		}
		v.Archived = m.Archived
	default:
		return errors.New("unsupported mission action")
	}
	return nil
}

func releaseMissionReservation(tx *sql.Tx, v *Mission) error {
	if _, err := tx.Exec(`DELETE FROM mission_reservations WHERE mission_id=?`, v.ID); err != nil {
		return err
	}
	if v.Assignment != nil {
		v.Assignment.Reserved = false
	}
	return nil
}

func missionReviewEvidence(v Mission, o MissionObservation) error {
	if v.Repository != "" && (!o.Clean || o.Revision == "") {
		return errors.New("commit the candidate changes and keep its working folder clean before review")
	}
	for _, c := range v.Criteria {
		var latest *MissionEvidence
		for i := range v.Evidence {
			e := &v.Evidence[i]
			if e.CriterionID == c.ID && e.ScopeVersion == v.ScopeVersion && (v.Repository == "" || e.Revision == o.Revision) {
				latest = e
			}
		}
		if latest == nil || latest.Outcome != "pass" {
			return errors.New("every criterion needs passing evidence for the current scope and revision")
		}
	}
	return nil
}
