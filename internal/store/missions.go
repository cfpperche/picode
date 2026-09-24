package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Missions (ADR-0199) own outcomes, never native processes or conversations.
const MissionOwner = "owner"

var (
	ErrMissionConflict  = errors.New("mission changed; refresh before trying again")
	ErrMissionForbidden = errors.New("this action is not allowed for this assignment")
	ErrMissionCapacity  = errors.New("mission history limit reached; archive this mission and create a new one")
)

type MissionCriterion struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type MissionEvidence struct {
	ID           string `json:"id"`
	CriterionID  string `json:"criterionId"`
	Kind         string `json:"kind"`
	Value        string `json:"value"`
	Outcome      string `json:"outcome"`
	Revision     string `json:"revision,omitempty"`
	Digest       string `json:"digest,omitempty"`
	At           string `json:"at"`
	Actor        string `json:"actor"`
	ScopeVersion int    `json:"scopeVersion"`
}

type MissionAssignment struct {
	AgentID        string `json:"agentId"`
	AgentName      string `json:"agentName"`
	CLI            string `json:"cli"`
	Session        string `json:"session,omitempty"`
	WorkPath       string `json:"workPath"`
	Generation     int    `json:"generation"`
	Reserved       bool   `json:"reserved"`
	Delivery       string `json:"delivery"`
	StartedAt      string `json:"startedAt"`
	AcknowledgedBy string `json:"acknowledgedBy,omitempty"`
}

type Mission struct {
	LastGeneration int                `json:"lastGeneration"`
	LastUpdate     string             `json:"lastUpdate,omitempty"`
	ID             string             `json:"id"`
	Sequence       int64              `json:"sequence"`
	WorkspaceID    string             `json:"workspaceId"`
	WorkspaceName  string             `json:"workspaceName"`
	WorkspacePath  string             `json:"workspacePath"`
	Repository     string             `json:"repository,omitempty"`
	Title          string             `json:"title"`
	Objective      string             `json:"objective"`
	Context        string             `json:"context"`
	Criteria       []MissionCriterion `json:"criteria"`
	Evidence       []MissionEvidence  `json:"evidence"`
	State          string             `json:"state"`
	Version        int                `json:"version"`
	ScopeVersion   int                `json:"scopeVersion"`
	NextAction     string             `json:"nextAction"`
	Blocker        string             `json:"blocker"`
	Assignment     *MissionAssignment `json:"assignment,omitempty"`
	ReviewRevision string             `json:"reviewRevision,omitempty"`
	ReviewScope    int                `json:"reviewScope,omitempty"`
	AcceptedAt     string             `json:"acceptedAt,omitempty"`
	InboxID        string             `json:"inboxId,omitempty"`
	Archived       bool               `json:"archived"`
	CreatedAt      string             `json:"createdAt"`
	UpdatedAt      string             `json:"updatedAt"`
}

type MissionMutation struct {
	FilesReady      bool               `json:"filesReady,omitempty"`
	InboxID         string             `json:"inboxId,omitempty"`
	Action          string             `json:"action"`
	ID              string             `json:"id,omitempty"`
	RequestID       string             `json:"requestId"`
	ExpectedVersion int                `json:"expectedVersion,omitempty"`
	WorkspaceID     string             `json:"workspaceId,omitempty"`
	Title           string             `json:"title,omitempty"`
	Objective       string             `json:"objective,omitempty"`
	Context         string             `json:"context,omitempty"`
	Criteria        []MissionCriterion `json:"criteria,omitempty"`
	AgentID         string             `json:"agentId,omitempty"`
	Generation      int                `json:"generation,omitempty"`
	Note            string             `json:"note,omitempty"`
	NextAction      string             `json:"nextAction,omitempty"`
	SourceStopped   bool               `json:"sourceStopped,omitempty"`
	TargetReady     bool               `json:"targetReady,omitempty"`
	Evidence        *MissionEvidence   `json:"evidence,omitempty"`
	Archived        bool               `json:"archived,omitempty"`
}

// MissionObservation is supplied by the server, never decoded from client JSON.
type MissionObservation struct {
	EvidenceDigest string
	Repository     string
	Session        string
	Revision       string
	Clean          bool
	Agent          *Agent
	WorkPath       string
}

type MissionHistory struct {
	Sequence int64   `json:"sequence"`
	Version  int     `json:"version"`
	Action   string  `json:"action"`
	Actor    string  `json:"actor"`
	At       string  `json:"at"`
	Note     string  `json:"note"`
	Snapshot Mission `json:"snapshot"`
}

func missionRaw(m MissionMutation) string { b, _ := json.Marshal(m); return string(b) }

func (s *Store) ReplayMission(actor string, m MissionMutation) (Mission, bool, error) {
	var payload, result string
	err := s.db.QueryRow(`SELECT payload,result FROM mission_requests WHERE actor=? AND request_id=?`, actor, m.RequestID).Scan(&payload, &result)
	if errors.Is(err, sql.ErrNoRows) {
		return Mission{}, false, nil
	}
	if err != nil {
		return Mission{}, false, err
	}
	if payload != missionRaw(m) {
		return Mission{}, false, ErrMissionConflict
	}
	var v Mission
	err = json.Unmarshal([]byte(result), &v)
	return v, true, err
}

// ApplyMission commits state, reservations, history, receipt and events together.
func (s *Store) ApplyMission(actor string, m MissionMutation, obs MissionObservation) (Mission, error) {
	if actor == "" || m.RequestID == "" || len(m.RequestID) > 128 {
		return Mission{}, errors.New("actor and requestId are required")
	}
	if len(m.Note) > 8192 || len(m.NextAction) > 2000 || len(m.Context) > 32768 || len(m.Criteria) > 32 {
		return Mission{}, errors.New("mission text exceeds its limit")
	}
	s.missionMu.Lock()
	defer s.missionMu.Unlock()
	if v, ok, err := s.ReplayMission(actor, m); ok || err != nil {
		return v, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Mission{}, err
	}
	defer s.rollback(tx)
	var v Mission
	if m.Action == "create" {
		if actor != MissionOwner {
			return v, ErrMissionForbidden
		}
		var w Workspace
		err = tx.QueryRow(`SELECT id,name,path FROM workspaces WHERE id=?`, m.WorkspaceID).Scan(&w.ID, &w.Name, &w.Path)
		if err != nil || IsFree(w) {
			return v, errors.New("choose an existing workspace")
		}
		var count int
		if err = tx.QueryRow(`SELECT count(*) FROM missions WHERE json_extract(body,'$.archived')=0`).Scan(&count); err != nil {
			return v, err
		}
		if count >= 2000 {
			return v, ErrMissionCapacity
		}
		v = Mission{ID: "mission_" + rand.Text(), WorkspaceID: w.ID, WorkspaceName: w.Name, WorkspacePath: w.Path, Repository: obs.Repository, State: "ready", ScopeVersion: 1, Criteria: []MissionCriterion{}, Evidence: []MissionEvidence{}, CreatedAt: nowUTC()}
	} else {
		var raw string
		if err = tx.QueryRow(`SELECT body FROM missions WHERE id=?`, m.ID).Scan(&raw); errors.Is(err, sql.ErrNoRows) {
			return v, ErrNotFound
		}
		if err != nil {
			return v, err
		}
		if err = json.Unmarshal([]byte(raw), &v); err != nil {
			return v, err
		}
		if v.Version != m.ExpectedVersion {
			return v, ErrMissionConflict
		}
		if v.Version >= 2000 && m.Action != "cancel" && m.Action != "release" && m.Action != "archive" {
			return v, ErrMissionCapacity
		}
	}
	if actor != MissionOwner {
		if v.Assignment == nil || !v.Assignment.Reserved || v.Assignment.AgentID != actor || v.Assignment.Generation != m.Generation {
			return v, ErrMissionForbidden
		}
		if obs.Session == "" || (v.Assignment.Session != "" && v.Assignment.Session != obs.Session) {
			return v, ErrMissionForbidden
		}
		switch m.Action {
		case "acknowledge", "report", "block", "evidence", "request-review":
		default:
			return v, ErrMissionForbidden
		}
		if m.Action != "acknowledge" && v.Assignment.Delivery != "acknowledged" {
			return v, errors.New("acknowledge this assignment before reporting work")
		}
		if v.State == "paused" || v.State == "cancelled" || v.State == "completed" {
			return v, errors.New("mission is not accepting agent updates")
		}
	}
	// A folder may become a Git repository after the mission was created.
	// Bind the first observed identity and invalidate evidence from the old scope.
	if m.Action != "relink" && v.Repository == "" && obs.Repository != "" && (v.State != "completed" && v.State != "cancelled" || m.Action == "reopen") {
		v.Repository = obs.Repository
		v.ScopeVersion++
		v.ReviewScope = 0
		v.ReviewRevision = ""
		if v.State == "in-review" {
			v.State = "in-progress"
		}
	}
	if m.Action == "inbox-answer" {
		err = s.missionAnswerTx(tx, &v, actor, m)
	} else {
		err = applyMissionAction(tx, &v, actor, m, obs)
	}
	if err != nil {
		return Mission{}, err
	}
	v.Version++
	v.UpdatedAt = nowUTC()
	if m.Action == "create" {
		res, e := tx.Exec(`INSERT INTO missions(id,workspace_id,body) VALUES(?,?,?)`, v.ID, v.WorkspaceID, "{}")
		if e != nil {
			return v, e
		}
		v.Sequence, _ = res.LastInsertId()
	}
	if m.Action == "block" || m.Action == "request-review" {
		if err = s.missionNoticeTx(tx, &v, m); err != nil {
			return v, err
		}
	}
	raw, _ := json.Marshal(v)
	if _, err = tx.Exec(`UPDATE missions SET body=?,workspace_id=? WHERE id=?`, string(raw), v.WorkspaceID, v.ID); err != nil {
		return v, err
	}
	h := MissionHistory{Version: v.Version, Action: m.Action, Actor: actor, At: v.UpdatedAt, Note: m.Note, Snapshot: v}
	hraw, _ := json.Marshal(h)
	if _, err = tx.Exec(`INSERT INTO mission_history(mission_id,version,action,actor,at,body) VALUES(?,?,?,?,?,?)`, v.ID, v.Version, m.Action, actor, v.UpdatedAt, string(hraw)); err != nil {
		return v, err
	}
	if _, err = tx.Exec(`INSERT INTO mission_requests(actor,request_id,payload,result) VALUES(?,?,?,?)`, actor, m.RequestID, missionRaw(m), string(raw)); err != nil {
		return v, err
	}
	if err = s.AppendEventTx(tx, "mission.changed", nil, nil, map[string]any{"id": v.ID, "workspaceId": v.WorkspaceID, "version": v.Version, "title": v.Title, "state": v.State, "action": m.Action}); err != nil {
		return v, err
	}
	return v, s.commit(tx)
}

func (s *Store) GetMission(id string) (Mission, error) {
	var raw string
	err := s.db.QueryRow(`SELECT body FROM missions WHERE id=?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return Mission{}, ErrNotFound
	}
	var v Mission
	if err == nil {
		err = json.Unmarshal([]byte(raw), &v)
	}
	return v, err
}

func (s *Store) ListMissions(workspace string, before int64) ([]Mission, error) {
	rows, err := s.db.Query(`SELECT body FROM missions WHERE (?='' OR workspace_id=?) AND (?=0 OR seq<?) ORDER BY seq DESC LIMIT 101`, workspace, workspace, before, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Mission{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		var v Mission
		if err = json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) MissionHistory(id string, before int64) ([]MissionHistory, error) {
	rows, err := s.db.Query(`SELECT seq,body FROM mission_history WHERE mission_id=? AND (?=0 OR seq<?) ORDER BY seq DESC LIMIT 101`, id, before, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MissionHistory{}
	for rows.Next() {
		var seq int64
		var raw string
		if err = rows.Scan(&seq, &raw); err != nil {
			return nil, err
		}
		var h MissionHistory
		if err = json.Unmarshal([]byte(raw), &h); err != nil {
			return nil, err
		}
		h.Sequence = seq
		out = append(out, h)
	}
	return out, rows.Err()
}

// MissionContext is bounded, deterministic and readable without a model.
func MissionContext(v Mission, history ...MissionHistory) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Mission: %s\n\n%s\n\n## Acceptance criteria\n", v.Title, v.Objective)
	for _, c := range v.Criteria {
		fmt.Fprintf(&b, "- [%s] %s\n", c.ID, c.Text)
	}
	fmt.Fprintf(&b, "\n## Context and decisions\n%s\n\n## Last checkpoint\n%s\n\n## Next action\n%s\n\n## Blocker\n%s\n", v.Context, v.LastUpdate, v.NextAction, v.Blocker)
	b.WriteString("\n## Recent reported attempts\n")
	count := 0
	for _, h := range history {
		if h.Version > v.Version || h.Note == "" || (h.Action != "report" && h.Action != "block" && h.Action != "changes") {
			continue
		}
		if count >= 8 {
			break
		}
		note := h.Note
		if len(note) > 1024 {
			note = strings.ToValidUTF8(note[:1024], "") + "… [excerpt]"
		}
		fmt.Fprintf(&b, "- v%d %s (%s): %s\n", h.Version, h.Action, h.Actor, note)
		count++
	}
	if count == 0 {
		b.WriteString("No earlier attempts in this packet.\n")
	}
	b.WriteString("\n## Evidence references\n")
	omitted := 0
	for _, e := range v.Evidence {
		value := e.Value
		if len(value) > 256 {
			value = strings.ToValidUTF8(value[:256], "") + "… [excerpt]"
		}
		line := fmt.Sprintf("- %s: criterion=%s kind=%s outcome=%s scope=%d revision=%s — %s\n", e.ID, e.CriterionID, e.Kind, e.Outcome, e.ScopeVersion, e.Revision, value)
		if b.Len()+len(line) > 120<<10 {
			omitted++
			continue
		}
		b.WriteString(line)
	}
	if len(v.Evidence) == 0 {
		b.WriteString("No evidence recorded.\n")
	}
	fmt.Fprintf(&b, "\nOmissions: native transcripts, credentials and file contents; older attempts beyond the latest eight; %d evidence references beyond the packet limit. Read the mission for full evidence and the owner history for older attempts.\n", omitted)
	fmt.Fprintf(&b, "\nMission ID: %s\nVersion: %d\n", v.ID, v.Version)
	if v.Assignment != nil {
		fmt.Fprintf(&b, "Assignment generation: %d\nWorking folder: %s\n", v.Assignment.Generation, v.Assignment.WorkPath)
	}
	fmt.Fprintf(&b, "\nUse the mission MCP tool (action show, id %s) when available, or `picode mission show --id %s`, for the latest state. Acknowledge the assignment before reporting work. Native transcripts, credentials and file contents are not included in this packet. Referenced evidence remains available in the mission.\n", v.ID, v.ID)
	packet := b.String()
	fmt.Fprintf(&b, "\nPacket SHA-256: %s\n", missionDigest(packet))
	return b.String()
}

func missionDigest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Notifications link back to the owner action. Inbox triage never completes a mission.
func (s *Store) missionNoticeTx(tx *sql.Tx, v *Mission, m MissionMutation) error {
	id := "inb_mission_" + rand.Text()
	title := "Review: " + v.Title
	if m.Action == "block" {
		title = "Needs you: " + v.Title
	}
	body := m.Note + "\n\n[Open mission](#/mission/" + v.ID + ")"
	kind, blocking, allowed := InboxResult, 0, `["ignore"]`
	if m.Action == "block" {
		kind, blocking, allowed = InboxQuestion, 1, `["respond","ignore"]`
	}
	_, err := tx.Exec(`INSERT INTO inbox_items(id,kind,source_kind,source_id,workspace_id,reason,title,body,blocking,allowed_responses,state,created_at,updated_at,session_path) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, kind, InboxFromSystem, v.ID, v.WorkspaceID, "mission", title, body, blocking, allowed, InboxUnread, v.UpdatedAt, v.UpdatedAt, "")
	if err != nil {
		return err
	}
	if m.Action == "block" {
		generation, session := 0, ""
		if v.Assignment != nil {
			generation, session = v.Assignment.Generation, v.Assignment.Session
		}
		if _, err = tx.Exec(`INSERT INTO mission_questions(inbox_id,mission_id,generation,scope_version,session) VALUES(?,?,?,?,?)`, id, v.ID, generation, v.ScopeVersion, session); err != nil {
			return err
		}
	}
	v.InboxID = id
	return s.AppendEventTx(tx, "inbox.created", nil, nil, map[string]any{"id": id, "kind": kind, "sourceKind": InboxFromSystem, "sourceId": v.ID, "workspaceId": v.WorkspaceID, "title": title})
}

// An old question remains answerable as history; it cannot reactivate a new
// assignment. Answers are never pasted into a terminal implicitly.
func (s *Store) missionAnswerTx(tx *sql.Tx, v *Mission, actor string, m MissionMutation) error {
	if actor != MissionOwner || strings.TrimSpace(m.Note) == "" {
		return ErrMissionForbidden
	}
	var generation, scope int
	var session string
	if err := tx.QueryRow(`SELECT generation,scope_version,session FROM mission_questions WHERE inbox_id=? AND mission_id=?`, m.InboxID, v.ID).Scan(&generation, &scope, &session); err != nil {
		return errors.New("mission question is unavailable")
	}
	currentGeneration, currentSession := 0, ""
	if v.Assignment != nil {
		currentGeneration, currentSession = v.Assignment.Generation, v.Assignment.Session
	}
	current := v.State == "blocked" && v.InboxID == m.InboxID && generation == currentGeneration && session == currentSession && scope == v.ScopeVersion
	note := "Answer recorded in mission history. Review the mission before reusing it for another assignment."
	if current {
		if err := applyMissionAction(tx, v, actor, MissionMutation{Action: "decision", Note: m.Note, NextAction: m.NextAction}, MissionObservation{}); err != nil {
			return err
		}
		note = "Decision recorded in mission context. The agent reads it with mission show; no terminal input was sent."
	}
	result, err := tx.Exec(`UPDATE inbox_items SET state=?,response=?,updated_at=?,body=body || ? WHERE id=? AND state<>?`, InboxDone, m.Note, nowUTC(), "\n\n"+note, m.InboxID, InboxDone)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return errors.New("this question is already resolved")
	}
	return s.AppendEventTx(tx, "inbox.updated", nil, nil, map[string]any{"id": m.InboxID, "workspaceId": v.WorkspaceID, "title": v.Title, "state": InboxDone})
}

// AgentMissionID is navigation metadata, not a grant to update the mission.
func (s *Store) AgentMissionID(agentID string) string {
	var id string
	_ = s.db.QueryRow(`SELECT mission_id FROM mission_reservations WHERE agent_id=?`, agentID).Scan(&id)
	return id
}
