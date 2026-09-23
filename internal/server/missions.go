package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/store"
)

// The effect lock serializes a mission's owner actions with prompt submission.
// Durable reservations in the store remain authoritative after a restart.
func registerMissionRoutes(mux Registrar, deps Deps) {
	var locks sync.Map
	lock := func(id string) func() {
		l, _ := locks.LoadOrStore(id, &sync.Mutex{})
		m := l.(*sync.Mutex)
		m.Lock()
		return m.Unlock
	}
	mux.HandleFunc("GET /api/missions", func(w http.ResponseWriter, r *http.Request) {
		before, err := missionCursor(r)
		if err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		rows, err := deps.Store.ListMissions(r.URL.Query().Get("workspace"), before)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"missions": rows})
	})
	mux.HandleFunc("GET /api/missions/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, err := deps.Store.GetMission(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		before, err := missionCursor(r)
		if err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		h, err := deps.Store.MissionHistory(v.ID, before)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		o, e := missionObservation(r.Context(), deps, v, "")
		observation := map[string]any{"available": e == nil, "revision": o.Revision, "clean": o.Clean, "session": o.Session, "agentAvailable": o.Agent != nil}
		if e != nil {
			observation["reason"] = e.Error()
		}
		if o.Agent != nil {
			busy, reason := agentBusy(r.Context(), deps, *o.Agent)
			observation["canDispatch"] = o.Agent.IsPi() || doorReaderCLI[o.Agent.CLI]
			observation["agentBusy"] = busy
			observation["activityReason"] = reason
		}
		writeJSON(w, 200, map[string]any{"mission": v, "context": store.MissionContext(v, h...), "history": h, "observation": observation})
	})
	owner := func(w http.ResponseWriter, r *http.Request) {
		var m store.MissionMutation
		if !decodeMission(w, r, &m) {
			return
		}
		if r.PathValue("id") == "" {
			m.Action = "create"
		} else {
			m.ID = r.PathValue("id")
		}
		defer lock(m.ID)()
		missionApply(w, r, deps, store.MissionOwner, m)
	}
	mux.HandleFunc("POST /api/missions", owner)
	mux.HandleFunc("POST /api/missions/{id}/actions", owner)
	mux.HandleFunc("POST /api/missions/tool", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Agent string `json:"agent"`
			Term  string `json:"term"`
			store.MissionMutation
		}
		if !decodeMission(w, r, &req) {
			return
		}
		var a store.Agent
		var err error
		if req.Agent != "" {
			a, err = deps.Store.GetAgent(req.Agent)
			if err == nil && req.Term != "" && (a.TerminalID == nil || *a.TerminalID != req.Term) {
				writeErr(w, 403, "agent and terminal identities disagree")
				return
			}
		} else if req.Term != "" {
			a, err = deps.Store.AgentByTerminal(req.Term)
		} else {
			writeErr(w, 403, "open this CLI as a PiCode agent")
			return
		}
		if err != nil {
			writeErr(w, 403, "agent identity is unavailable")
			return
		}
		defer lock(req.ID)()
		v, err := deps.Store.GetMission(req.ID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if v.Assignment == nil || v.Assignment.AgentID != a.ID || !v.Assignment.Reserved {
			writeErr(w, 403, "this mission is not assigned to this agent")
			return
		}
		if req.Action == "show" || req.Action == "context" {
			h, e := deps.Store.MissionHistory(v.ID, 0)
			if e != nil {
				writeStoreErr(w, e)
				return
			}
			if req.Action == "context" {
				writeJSON(w, 200, map[string]any{"id": v.ID, "version": v.Version, "assignment": v.Assignment, "context": store.MissionContext(v, h...)})
				return
			}
			writeJSON(w, 200, map[string]any{"mission": v, "context": store.MissionContext(v, h...)})
			return
		}
		missionApply(w, r, deps, a.ID, req.MissionMutation)
	})
}

func missionCursor(r *http.Request) (int64, error) {
	s := r.URL.Query().Get("before")
	if s == "" {
		return 0, nil
	}
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil || n < 0 {
		return 0, errors.New("invalid history cursor")
	}
	return n, nil
}

func decodeMission(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 80<<10))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		writeErr(w, 400, "invalid mission request")
		return false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		writeErr(w, 400, "expected one JSON object")
		return false
	}
	return true
}

func missionSession(deps Deps, a store.Agent) string {
	if a.TerminalID != nil {
		if rt, ok := deps.TermRuntimes.Get(*a.TerminalID); ok && rt.SessionID != "" {
			return a.CLI + ":" + rt.SessionID
		}
		if launch, e := deps.Store.TerminalLaunch(*a.TerminalID); e == nil && launch != nil && launch.LastSession != nil {
			ls := launch.LastSession
			if ls.SessionID != "" {
				return a.CLI + ":" + ls.SessionID
			}
			if ls.Path != "" {
				return a.CLI + ":" + ls.Path
			}
		}
	}
	if a.SessionPath != nil && *a.SessionPath != "" {
		return a.CLI + ":" + *a.SessionPath
	}
	return ""
}

func missionObservation(ctx context.Context, deps Deps, v store.Mission, target string) (store.MissionObservation, error) {
	o := store.MissionObservation{}
	w, err := deps.Store.GetWorkspace(v.WorkspaceID)
	if err != nil || w.Path != v.WorkspacePath {
		return o, errors.New("the mission's workspace is unavailable")
	}
	cwd := w.Path
	if target == "" && v.Assignment != nil {
		target = v.Assignment.AgentID
		cwd = v.Assignment.WorkPath
	}
	if target != "" {
		a, e := deps.Store.GetAgent(target)
		if e == nil {
			if a.WorkspaceID != v.WorkspaceID {
				return o, errors.New("choose an agent in the mission's workspace")
			}
			o.Agent = &a
			o.Session = missionSession(deps, a)
			cwd = store.AgentCwd(w, a)
			if v.Assignment != nil && v.Assignment.AgentID == target && v.Assignment.WorkPath != cwd {
				return o, errors.New("the assigned agent changed working folder; assign it again")
			}
		} else if v.Assignment == nil || target != v.Assignment.AgentID {
			return o, errors.New("target agent is unavailable")
		}
	}
	if info, e := os.Stat(cwd); e != nil || !info.IsDir() {
		return o, errors.New("the working folder is unavailable")
	}
	o.WorkPath = cwd
	probe, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if _, e := deliveryGit(probe, cwd, "rev-parse", "--show-toplevel"); e == nil {
		o.Repository = gitgraph.Key(cwd)
		if v.Repository != "" && v.Repository != o.Repository {
			return o, errors.New("the working folder belongs to another repository")
		}
		o.Revision, err = deliveryGit(probe, cwd, "rev-parse", "HEAD")
		if err != nil {
			o.Revision = ""
		}
		status, e := deliveryGit(probe, cwd, "status", "--porcelain", "--untracked-files=normal")
		o.Clean = e == nil && status == ""
	} else if v.Repository != "" {
		return o, errors.New("the mission repository cannot be verified")
	}
	return o, nil
}

func missionApply(w http.ResponseWriter, r *http.Request, deps Deps, actor string, m store.MissionMutation) {
	if m.RequestID == "" {
		writeErr(w, 400, "requestId is required")
		return
	}
	// Replay before source revalidation: an old receipt never resends a prompt.
	if v, ok, err := deps.Store.ReplayMission(actor, m); err != nil {
		missionError(w, err)
		return
	} else if ok {
		writeJSON(w, 200, map[string]any{"mission": v, "replayed": true})
		return
	}
	var v store.Mission
	var err error
	if m.Action == "create" {
		ws, e := deps.Store.GetWorkspace(m.WorkspaceID)
		if e != nil {
			writeStoreErr(w, e)
			return
		}
		v = store.Mission{WorkspaceID: ws.ID, WorkspacePath: ws.Path}
	} else {
		v, err = deps.Store.GetMission(m.ID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
	}
	target := ""
	if m.Action == "assign" {
		target = m.AgentID
	}
	observed := v
	if m.Action == "relink" {
		ws, e := deps.Store.GetWorkspace(m.WorkspaceID)
		if e != nil {
			writeStoreErr(w, e)
			return
		}
		observed.WorkspaceID, observed.WorkspacePath, observed.Assignment = ws.ID, ws.Path, nil
	}
	o, obsErr := missionObservation(r.Context(), deps, observed, target)
	needsSource := m.Action == "relink" || m.Action == "create" || m.Action == "assign" || m.Action == "dispatch" || m.Action == "evidence" || m.Action == "accept" || m.Action == "request-review" || actor != store.MissionOwner
	if needsSource && obsErr != nil {
		writeErr(w, 409, obsErr.Error())
		return
	}
	if actor != store.MissionOwner && o.Agent == nil {
		writeErr(w, 403, "assigned agent is unavailable")
		return
	}
	if m.Action == "assign" || m.Action == "dispatch" || m.Action == "accept" || m.Action == "release" {
		if v.Assignment != nil && v.Assignment.Reserved {
			if a, e := deps.Store.GetAgent(v.Assignment.AgentID); e == nil {
				if busy, reason := agentBusy(r.Context(), deps, a); busy {
					writeErr(w, 409, "current agent is "+reason+"; open it before changing responsibility")
					return
				}
			}
		}
		if o.Agent != nil {
			if busy, reason := agentBusy(r.Context(), deps, *o.Agent); busy {
				writeErr(w, 409, "target agent is "+reason)
				return
			}
		}
	}
	if m.Action == "assign" && v.Assignment != nil && v.Assignment.WorkPath != o.WorkPath {
		if !m.FilesReady {
			writeErr(w, 409, "confirm the required files are already present in the target working folder")
			return
		}
		source, sourceErr := missionObservation(r.Context(), deps, v, "")
		if sourceErr != nil || (v.Repository != "" && (!source.Clean || source.Revision != o.Revision)) {
			writeErr(w, 409, "transfer the source changes explicitly and open the same candidate revision in the target folder first")
			return
		}
	}
	if m.Action == "evidence" && m.Evidence != nil {
		if (v.Repository != "" || o.Repository != "") && (!o.Clean || o.Revision == "") {
			writeErr(w, 409, "commit the candidate and keep its working folder clean before adding evidence")
			return
		}
		// Observed evidence metadata cannot be supplied by an agent.
		e := *m.Evidence
		switch e.Kind {
		case "file":
			digest, e2 := missionFileDigest(o.WorkPath, e.Value)
			if e2 != nil {
				writeErr(w, 400, e2.Error())
				return
			}
			o.EvidenceDigest = digest
		case "delivery":
			d, e2 := deps.Store.GetDelivery(o.Repository, e.Value)
			if e2 != nil || d.Revision != o.Revision {
				writeErr(w, 409, "delivery must belong to this repository and candidate revision")
				return
			}
		}
	}
	if m.Action == "accept" || m.Action == "request-review" {
		if err := missionCheckEvidence(deps, v, o); err != nil {
			writeErr(w, 409, err.Error())
			return
		}
	}
	var dispatchHistory []store.MissionHistory
	if m.Action == "dispatch" {
		dispatchHistory, err = deps.Store.MissionHistory(v.ID, 0)
		if err != nil {
			missionError(w, err)
			return
		}
		if o.Agent == nil || v.Assignment == nil {
			writeErr(w, 409, "assign an available agent first")
			return
		}
		if !o.Agent.IsPi() && !doorReaderCLI[o.Agent.CLI] {
			writeErr(w, 409, "copy mission context and continue manually in this CLI; automatic submission is not supported")
			return
		}
		if o.Session != v.Assignment.Session {
			writeErr(w, 409, "the native session changed; assign it again")
			return
		}
		if deps.runMode(r, o.Agent.ID) == modeStopped {
			writeErr(w, 409, "open and start the assigned agent first")
			return
		}
	}
	updated, err := deps.Store.ApplyMission(actor, m, o)
	if err != nil {
		missionError(w, err)
		return
	}
	out := map[string]any{"mission": updated}
	if (m.Action == "pause" || m.Action == "cancel") && updated.Assignment != nil && updated.Assignment.Reserved {
		out["stop"] = "Open the agent to stop it and confirm its child processes have stopped writing."
		if deps.Runtime != nil {
			if ma := deps.Runtime.Get(updated.Assignment.AgentID); ma != nil {
				ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
				err := ma.Abort(ctx)
				cancel()
				if err == nil {
					out["stop"] = "Interruption requested. Confirm the agent and its child processes have stopped writing before releasing responsibility."
				} else {
					out["stop"] = "Interruption could not be confirmed. Open the agent to stop it."
				}
			}
		}
	}

	if m.Action == "dispatch" {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		text := store.MissionContext(updated, dispatchHistory...)
		if deps.Runtime != nil && deps.Runtime.Active(o.Agent.ID) {
			ma := deps.Runtime.Get(o.Agent.ID)
			if ma == nil {
				writeJSON(w, 200, map[string]any{"mission": updated, "submission": map[string]any{"error": "agent stopped before submission", "delivery": "unconfirmed"}})
				return
			}
			err = ma.SendTurn("prompt", text, nil)
			out["submission"] = map[string]any{"submitted": err == nil}
			if err != nil {
				out["submission"] = map[string]any{"error": err.Error(), "delivery": "unconfirmed"}
			}
		} else if o.Agent.TerminalID != nil {
			t, e := deps.Store.GetTerminal(*o.Agent.TerminalID)
			if e == nil {
				_, result := doorDeliverUnattended(deps, ctx, t, text)
				out["submission"] = result
			} else {
				out["submission"] = map[string]any{"error": "terminal unavailable", "delivery": "unconfirmed"}
			}
		} else {
			_, result := deps.deliverToInteractiveAgent(ctx, *o.Agent, text, tuiDeliverPrompt)
			out["submission"] = result
		}
	}
	writeJSON(w, 200, out)
}

func missionFileDigest(root, rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) {
		return "", errors.New("choose a file relative to the mission's working folder")
	}
	dir, err := os.OpenRoot(root)
	if err != nil {
		return "", errors.New("working folder unavailable")
	}
	defer dir.Close()
	f, err := dir.Open(rel)
	if err != nil {
		return "", errors.New("evidence must be an available file inside the working folder")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return "", errors.New("evidence must be a regular file of at most 4 MB")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, (4<<20)+1))
	if err != nil {
		return "", err
	}
	if n > 4<<20 {
		return "", errors.New("evidence exceeds 4 MB")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func missionCheckEvidence(deps Deps, v store.Mission, o store.MissionObservation) error {
	// Only the latest evidence for each criterion at this scope/revision counts.
	for _, c := range v.Criteria {
		var latest *store.MissionEvidence
		for i := range v.Evidence {
			e := &v.Evidence[i]
			if e.CriterionID == c.ID && e.ScopeVersion == v.ScopeVersion && (v.Repository == "" || e.Revision == o.Revision) {
				latest = e
			}
		}
		if latest == nil {
			continue
		}
		switch latest.Kind {
		case "file":
			d, e := missionFileDigest(o.WorkPath, latest.Value)
			if e != nil || d != latest.Digest {
				return fmt.Errorf("evidence for %q changed or is unavailable", c.Text)
			}
		case "delivery":
			d, e := deps.Store.GetDelivery(o.Repository, latest.Value)
			if e != nil || d.Revision != o.Revision {
				return errors.New("linked delivery is unavailable or changed")
			}
		}
	}
	return nil
}

func missionError(w http.ResponseWriter, err error) {
	code := 400
	if errors.Is(err, store.ErrMissionConflict) {
		code = 409
	}
	if errors.Is(err, store.ErrMissionForbidden) {
		code = 403
	}
	if errors.Is(err, store.ErrNotFound) {
		code = 404
	}
	if errors.Is(err, store.ErrMissionCapacity) {
		code = 409
	}
	writeErr(w, code, err.Error())
}
