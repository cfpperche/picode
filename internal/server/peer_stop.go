package server

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func peerStopPath(deps Deps, id string) string {
	return filepath.Join(deps.DataDir, "communication-stops", id+".json")
}
func peerStopPending(deps Deps, id string) (bool, error) {
	path := peerStopPath(deps, id)
	raw, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		return false, nil
	}
	if e != nil {
		return true, e
	}
	var owned map[int]string
	if json.Unmarshal(raw, &owned) != nil || len(owned) == 0 {
		return true, errors.New("unreadable process shutdown receipt")
	}
	for pid, token := range owned {
		if pid <= 0 || token == "" {
			return true, errors.New("invalid process shutdown receipt")
		}
		if processAlive(TermRuntime{PID: pid, ProcStart: token}) {
			return true, nil
		}
	}
	if e = os.Remove(path); e != nil && !os.IsNotExist(e) {
		return true, e
	}
	return false, nil
}
func savePeerStop(deps Deps, id string, owned map[int]string) error {
	path := peerStopPath(deps, id)
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	raw, e := json.Marshal(owned)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".stop-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, e = f.Write(raw); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
