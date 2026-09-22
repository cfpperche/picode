// Package climodels asks an agent CLI which models it can actually reach, so a
// pane that writes a model selector offers the CLI's own answer instead of a
// text box (ADR-0181).
//
// Only omp declares a reader today, and it is the vendor's own command — the
// bounded-subprocess exception ADR-0167 already makes for plugin verbs, with
// none of its consequences: this one is read-only and changes nothing.
//
// Two facts decide the shape, both measured on 2026-09-22 against omp 18.2.8:
//
//   - **The answer depends on the directory.** Run in a workspace whose
//     `.omp/config.yml` sets `disabledProviders: [openai]`, the same command
//     returned 0 models where the folder next to it returned 55. So the reader
//     always runs in the workspace the pane is editing, and a report says which
//     directory produced it.
//   - **Zero models is ambiguous.** A provider the project disabled and a
//     provider with no credential look identical from outside. The report says
//     the list is empty and lets the pane say why it cannot say why, rather
//     than inventing a reason.
package climodels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// readTimeout bounds the probe. Measured on this machine 2026-09-22: about two
// seconds against a warm home, eleven against a fresh one (the CLI builds its
// catalog cache on first use). The pane says "Asking the CLI…" meanwhile; past
// this bound it is better off saying the CLI did not answer.
const readTimeout = 25 * time.Second

// Model is one row of a CLI's catalog, in the CLI's own vocabulary.
type Model struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
	// Selector is what a role assignment holds. omp emits it itself, so the
	// pane writes the vendor's own string rather than one PiCode assembled.
	Selector string   `json:"selector"`
	Context  int      `json:"contextWindow,omitempty"`
	Thinking []string `json:"thinking,omitempty"`
	// Input lists what the model accepts beside text (`image`), in the
	// vendor's own words.
	Input []string `json:"input,omitempty"`
	// Cost is USD per million tokens, as the vendor's catalog states it. A
	// time-of-day multiplier the vendor also carries is not read: a pane
	// that printed one price for a model billed two ways would be wrong half
	// the day.
	Cost *Cost `json:"cost,omitempty"`
}

// Cost is the vendor's own per-million-token prices.
type Cost struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

// Report is one read of one CLI's catalog.
type Report struct {
	CLI    string  `json:"cli"`
	Dir    string  `json:"dir,omitempty"`
	Models []Model `json:"models"`
	// Kinds are the kinds present in this answer, in the vendor's own order,
	// so the pane's filter chips are the CLI's and not a guess.
	Kinds []string `json:"kinds,omitempty"`
	// CachedAt is when the CLI was asked; a cached answer carries the time it
	// was first given, so the pane can say how old it is.
	CachedAt string `json:"askedAt,omitempty"`
}

// supported are the CLIs PiCode has measured a read-only catalog command for.
var supported = []string{"omp"}

// Supports reports whether PiCode knows how to ask this CLI.
func Supports(cli string) bool {
	for _, id := range supported {
		if id == cli {
			return true
		}
	}
	return false
}

// Supported lists them, for the test that keeps the UI's list in step.
func Supported() []string { return append([]string{}, supported...) }

// The probe costs about eleven seconds (measured 2026-09-22), so an answer is
// kept while the files that decide it are unchanged: both config layers and the
// custom providers, by path, size and modification time. A save in PiCode's own
// panes changes one of them, so a write is never answered from before it.
//
// The credential store and the catalog cache are deliberately not in the
// fingerprint: omp rewrites both on every run — measured, the first version of
// this cache keyed on them and never hit once — and several omp terminals share
// them on the owner's machine. What they would have caught (a sign-in, a
// catalog refresh) is covered by the age bound and by the pane's Refresh, which
// asks again regardless (`fresh`).
const cacheFor = 10 * time.Minute

type cached struct {
	print string
	at    time.Time
	rep   Report
}

var (
	cacheMu sync.Mutex
	cache   = map[string]cached{}
	// home is the CLI's config root; tests point it at a fixture.
	home = func() string { h, _ := os.UserHomeDir(); return h }
)

func fingerprint(dir string) string {
	agent := filepath.Join(home(), ".omp", "agent")
	if v := os.Getenv("PI_CODING_AGENT_DIR"); v != "" {
		agent = v
	}
	files := []string{
		filepath.Join(agent, "config.yml"),
		filepath.Join(agent, "models.yml"),
	}
	if dir != "" {
		files = append(files, filepath.Join(dir, ".omp", "config.yml"), filepath.Join(dir, ".omp", "settings.json"))
	}
	var b strings.Builder
	for _, f := range files {
		if st, err := os.Stat(f); err == nil {
			fmt.Fprintf(&b, "%s:%d:%d;", f, st.Size(), st.ModTime().UnixNano())
		} else {
			fmt.Fprintf(&b, "%s:-;", f)
		}
	}
	return b.String()
}

// Read asks the CLI for its catalog, in dir. An empty dir runs where the daemon
// runs, which answers for the machine rather than for a project.
func Read(ctx context.Context, cli, dir string, fresh bool) (Report, error) {
	if !Supports(cli) {
		return Report{}, fmt.Errorf("PiCode cannot ask %s for its models", cli)
	}
	key := cli + "\x00" + dir
	if !fresh {
		cacheMu.Lock()
		c, ok := cache[key]
		cacheMu.Unlock()
		if ok && c.print == fingerprint(dir) && time.Since(c.at) < cacheFor {
			return c.rep, nil
		}
	}
	// Taken before the probe, so a config written while it runs makes the next
	// read ask again rather than keep an answer from before the write.
	print := fingerprint(dir)
	rep, err := probe(ctx, dir)
	if err != nil {
		return Report{}, err
	}
	rep.CachedAt = time.Now().UTC().Format(time.RFC3339)
	cacheMu.Lock()
	cache[key] = cached{print: print, at: time.Now(), rep: rep}
	cacheMu.Unlock()
	return rep, nil
}

func probe(ctx context.Context, dir string) (Report, error) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	// `--kind all` is the vendor's own word for "every kind", and the roles
	// that are not chat roles (image, speech, dictation, judge, web) need it.
	cmd := exec.CommandContext(ctx, "omp", "models", "--json", "--kind", "all")
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if ctx.Err() != nil {
			msg = "it did not answer in time"
		}
		return Report{}, fmt.Errorf("omp models: %s", bounded(msg, 400))
	}
	return parseOmp(stdout.Bytes(), dir)
}

func parseOmp(raw []byte, dir string) (Report, error) {
	var body struct {
		Models []Model `json:"models"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return Report{}, fmt.Errorf("omp models answered something PiCode could not read: %w", err)
	}
	rep := Report{CLI: "omp", Dir: dir, Models: body.Models}
	if rep.Models == nil {
		rep.Models = []Model{}
	}
	seen := map[string]bool{}
	for i := range rep.Models {
		// A row with no kind is a chat model: omp omits the field for the
		// default kind (`pi-catalog/src/types.ts`, `modelKind`).
		if rep.Models[i].Kind == "" {
			rep.Models[i].Kind = "chat"
		}
		// A selector the vendor did not emit is rebuilt the way it builds one,
		// so a row is never offered without the string a save would write.
		if rep.Models[i].Selector == "" {
			rep.Models[i].Selector = rep.Models[i].Provider + "/" + rep.Models[i].ID
		}
		if !seen[rep.Models[i].Kind] {
			seen[rep.Models[i].Kind] = true
			rep.Kinds = append(rep.Kinds, rep.Models[i].Kind)
		}
	}
	sort.Strings(rep.Kinds)
	return rep, nil
}

func bounded(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
