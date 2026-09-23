package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/desktop"
)

// The root half of the Management window (ADR-0198): the distro's
// /etc/wsl.conf and its system caches, reached with `wsl.exe -u root
// --exec` and only through the closed list in internal/desktop/root.go.

// wslConfReport is the Distro tab: the file's owned values and the table the
// window renders from, so the keys live in one place.
type wslConfReport struct {
	Path   string              `json:"path"`
	Values []desktop.ConfValue `json:"values"`
	Keys   []desktop.ConfKey   `json:"keys"`
	Error  string              `json:"error,omitempty"`
}

// readWSLConf returns the file's text; a missing file is empty.
func readWSLConf(a app) (string, error) {
	out, err := a.runner.Output(desktop.WSLExe, desktop.ReadWSLConfArgs(a.distro)...)
	if err != nil {
		return "", fmt.Errorf("read %s as root: %w", desktop.WSLConfPath, err)
	}
	return string(out), nil
}

func runWSLConf(distroFlag, userFlag string) error {
	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
	}
	a.runner = timedRunner{d: 30 * time.Second}
	rep := wslConfReport{Path: desktop.WSLConfPath, Keys: desktop.WSLConfKeys, Values: []desktop.ConfValue{}}
	raw, err := readWSLConf(a)
	if err != nil {
		rep.Error = err.Error()
	} else {
		rep.Values = desktop.ParseWSLConf(raw)
	}
	return json.NewEncoder(os.Stdout).Encode(rep)
}

// runWSLConfWrite applies edits (a JSON list of {section,key,value}) to the
// current file and writes it back, keeping the old one as .bak. The edits
// are validated against the key table before anything runs as root.
//
// --yes is the tool's own gate, like system-clean's: the window asks the
// person, the tool refuses to write without the answer.
func runWSLConfWrite(distroFlag, userFlag, editsJSON string, yes bool) error {
	type result struct {
		Backup string `json:"backup,omitempty"`
		Error  string `json:"error,omitempty"`
	}
	emit := func(r result) error { return json.NewEncoder(os.Stdout).Encode(r) }
	if !yes {
		return emit(result{Error: "--yes is required to write " + desktop.WSLConfPath})
	}

	var edits []desktop.ConfEdit
	if err := json.Unmarshal([]byte(editsJSON), &edits); err != nil {
		return emit(result{Error: "edits JSON: " + err.Error()})
	}
	for _, e := range edits {
		if err := desktop.ValidateConf(e); err != nil {
			return emit(result{Error: err.Error()})
		}
	}
	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return emit(result{Error: err.Error()})
	}
	a.runner = timedRunner{d: 30 * time.Second}
	// A default account that does not exist makes WSL fall back to root.
	for _, e := range edits {
		if e.Section == "user" && e.Key == "default" && e.Value != "" {
			if err := a.runner.Run(desktop.WSLExe, desktop.RootArgs(a.distro, "getent", "passwd", e.Value)...); err != nil {
				return emit(result{Error: fmt.Sprintf("[user] default: there is no account %q in %s", e.Value, a.distro)})
			}
		}
	}
	raw, err := readWSLConf(a)
	if err != nil {
		return emit(result{Error: err.Error()})
	}
	next, err := desktop.EditWSLConf(raw, edits)
	if err != nil {
		return emit(result{Error: err.Error()})
	}
	if next == raw {
		return emit(result{})
	}
	if err := a.runner.Run(desktop.WSLExe, desktop.WriteWSLConfArgs(a.distro, next)...); err != nil {
		return emit(result{Error: "write " + desktop.WSLConfPath + " as root: " + err.Error()})
	}
	backup := ""
	if raw != "" {
		backup = desktop.WSLConfPath + ".bak"
	}
	return emit(result{Backup: backup})
}

type systemCleanResult struct {
	ID     string `json:"id"`
	Before int64  `json:"before"`
	After  int64  `json:"after"`
	Freed  int64  `json:"freed"`
	Error  string `json:"error,omitempty"`
}

// runSystemClean prunes the named system caches as root: measure, prune
// each in order (one progress line each), measure again. The outcome has
// the same shape as the person's own clean, so the window reads both alike.
func runSystemClean(distroFlag, userFlag, apply string, yes bool) error {
	emitJSON := func(v any) error { return json.NewEncoder(os.Stdout).Encode(v) }
	if !yes {
		return emitJSON(map[string]string{"refused": "--yes is required for --apply"})
	}
	var chosen []desktop.SystemCache
	for _, id := range splitList(apply) {
		c, ok := desktop.SystemCacheByID(id)
		if !ok {
			return emitJSON(map[string]string{"refused": fmt.Sprintf("unknown system cache %q", id)})
		}
		chosen = append(chosen, c)
	}
	if len(chosen) == 0 {
		return emitJSON(map[string]string{"refused": "no system cache ids given"})
	}
	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
	}
	a.runner = timedRunner{d: 5 * time.Minute}

	sizes := func() map[string]int64 {
		m := map[string]int64{}
		if ms, err := desktop.MeasureSystemCaches(a.runner, a.distro); err == nil {
			for _, c := range ms {
				m[c.ID] = c.Bytes
			}
		}
		return m
	}
	before := sizes()
	var results []systemCleanResult
	for _, c := range chosen {
		fmt.Println(progressLine("cleaning " + c.Title + "…"))
		res := systemCleanResult{ID: c.ID, Before: before[c.ID]}
		if err := desktop.PruneSystemCache(a.runner, a.distro, c); err != nil {
			res.Error = err.Error()
		}
		results = append(results, res)
	}
	after := sizes()
	var total int64
	for i := range results {
		results[i].After = after[results[i].ID]
		if f := results[i].Before - results[i].After; f > 0 {
			results[i].Freed = f
			total += f
		}
	}
	return emitJSON(struct {
		Results    []systemCleanResult `json:"results"`
		TotalFreed int64               `json:"totalFreed"`
	}{results, total})
}

// splitList splits a comma list, dropping empties.
func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
