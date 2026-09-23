package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/hostfs"
)

// runClean prunes the caches `picode disk` measures — and nothing else.
//
// The prune table IS the consumer table in internal/hostfs: one source of
// truth for what a cache is, where it lives and what command gives the space
// back. KindData rows (Pi sessions, the PiCode database) are refused even
// when asked for by name; cleanup of those belongs to the app and to backups,
// never to a delete.
//
// --list measures and prints. --apply runs the table's own How commands and
// refuses without --yes. Progress and the outcome travel as JSON lines in
// --json mode, the same contract `picode-desktop disk-compact` uses, so the
// shell streams both the same way.
func runClean(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "emit progress and the outcome as JSON lines")
	list := fs.Bool("list", false, "measure and print the prunable caches")
	apply := fs.String("apply", "", "comma-separated cache ids to prune")
	yes := fs.Bool("yes", false, "required for --apply; there is no prompt")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("clean: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("clean: %v", err)
	}

	switch {
	case *list:
		if err := cleanList(home, *asJSON); err != nil {
			log.Fatalf("clean: %v", err)
		}
	case *apply != "":
		if err := cleanApply(home, splitIDs(*apply), *yes, *asJSON, realRunner{}); err != nil {
			log.Fatalf("clean: %v", err)
		}
	default:
		if *asJSON {
			emit(map[string]string{"refused": "nothing to do — pass --list or --apply id1,id2"})
			return
		}
		fmt.Println("nothing to do — pass --list to measure, or --apply id1,id2 --yes to prune")
	}
}

// cleanList prints the prunable half of the consumer table with fresh sizes.
func cleanList(home string, asJSON bool) error {
	rep, err := hostfs.MeasureWith(hostfs.Exec{}, home, locatedConsumers(home))
	if err != nil {
		return err
	}
	out := struct {
		At        string           `json:"at"`
		Consumers []cleanCacheInfo `json:"consumers"`
	}{At: rep.Measured.UTC().Format(time.RFC3339)}

	for _, m := range rep.Consumers {
		if m.Kind == hostfs.KindData {
			continue
		}
		out.Consumers = append(out.Consumers, cleanCacheInfo{
			ID: m.ID, Title: m.Title, Kind: string(m.Kind), Bytes: m.Bytes, How: m.How, Note: m.Note,
		})
	}

	if asJSON {
		return json.NewEncoder(os.Stdout).Encode(out)
	}
	fmt.Printf("Prunable caches (measured %s)\n", rep.Measured.Format("15:04"))
	for _, c := range out.Consumers {
		fmt.Printf("  %-9s %-11s %-26s %s\n", hostfs.Bytes(c.Bytes), c.Kind, c.Title, c.How)
	}
	return nil
}

type cleanCacheInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
	Bytes int64  `json:"bytes"`
	How   string `json:"how"`
	Note  string `json:"note"`
}

type cleanResult struct {
	ID     string `json:"id"`
	Before int64  `json:"before"`
	After  int64  `json:"after"`
	Freed  int64  `json:"freed"`
	Error  string `json:"error,omitempty"`
}

// cleanApply prunes the named ids. Refusals print as JSON with exit zero in
// --json mode — the shell reads the fields, not the exit code.
func cleanApply(home string, ids []string, yes, asJSON bool, r runner) error {
	if !yes {
		return refuse(asJSON, "--yes is required for --apply")
	}
	if len(ids) == 0 {
		return refuse(asJSON, "no cache ids given")
	}

	table := map[string]hostfs.Consumer{}
	for _, c := range hostfs.Consumers(home) {
		table[c.ID] = c
	}
	chosen := make([]hostfs.Consumer, 0, len(ids))
	for _, id := range ids {
		c, ok := table[id]
		if !ok {
			return refuse(asJSON, fmt.Sprintf("unknown cache id %q", id))
		}
		if c.Kind == hostfs.KindData {
			// Defense in depth: the caller may only name caches. A data
			// delete routed here would be a bug wearing a flag.
			return refuse(asJSON, fmt.Sprintf("%s is not a cache — its cleanup is never a delete", id))
		}
		chosen = append(chosen, c)
	}

	results := r.apply(home, chosen, asJSON)

	var total int64
	for _, res := range results {
		total += res.Freed
	}

	if asJSON {
		return json.NewEncoder(os.Stdout).Encode(struct {
			Results    []cleanResult `json:"results"`
			TotalFreed int64         `json:"totalFreed"`
		}{Results: results, TotalFreed: total})
	}
	for _, res := range results {
		if res.Error != "" {
			fmt.Printf("  %-9s %s — failed: %s\n", hostfs.Bytes(res.Freed), res.ID, res.Error)
			continue
		}
		fmt.Printf("  %-9s %s\n", hostfs.Bytes(res.Freed), res.ID)
	}
	fmt.Printf("Freed ≈%s\n", hostfs.Bytes(total))
	return nil
}

func refuse(asJSON bool, msg string) error {
	if asJSON {
		emit(map[string]string{"refused": msg})
		return nil
	}
	return fmt.Errorf("refused: %s", msg)
}

// splitIDs splits a comma list and drops empties, so "a,,b" and a trailing
// comma behave the same.
func splitIDs(s string) []string {
	var ids []string
	for _, id := range strings.Split(s, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func emit(v any) { _ = json.NewEncoder(os.Stdout).Encode(v) }

// progressLine is the compact command's wire contract: one JSON object per
// line, the final outcome is the last line. Kept local — the desktop binary
// owns its own copy of the same one-liner.
func progressLine(step string) string {
	return fmt.Sprintf("{\"progress\":%q}", step)
}

// runner executes the table's How commands and measures the caches. Real
// runs shell out and du the disk; tests fake both.
type runner interface {
	// apply prunes each consumer in order, printing progress in --json mode,
	// then measures once and fills in after/freed.
	apply(home string, chosen []hostfs.Consumer, asJSON bool) []cleanResult
}

type realRunner struct{}

func (realRunner) apply(home string, chosen []hostfs.Consumer, asJSON bool) []cleanResult {
	// Located once: before and after are measured in the same place, or
	// "freed" compares two different folders.
	located := locatedConsumers(home)
	before := measure(home, located)
	results := make([]cleanResult, 0, len(chosen))
	for _, c := range chosen {
		if asJSON {
			fmt.Println(progressLine(fmt.Sprintf("cleaning %s…", c.Title)))
		} else {
			fmt.Printf("  cleaning %s…\n", c.Title)
		}
		res := cleanResult{ID: c.ID, Before: before[c.ID]}
		if err := runHow(c); err != nil {
			res.Error = err.Error()
		}
		results = append(results, res)
	}

	after := measure(home, located)
	for i := range results {
		results[i].After = after[results[i].ID]
		results[i].Freed = results[i].Before - results[i].After
		if results[i].Freed < 0 {
			results[i].Freed = 0 // a cache that grew mid-run is not negative space
		}
	}
	return results
}

// measure sizes exactly the named consumers by measuring the whole home once
// and keeping what was asked for. Two full sweeps per clean (before, after)
// is the price of numbers that are comparable — a per-path du would count a
// shared parent twice.
func measure(home string, table []hostfs.Consumer) map[string]int64 {
	rep, err := hostfs.MeasureWith(hostfs.Exec{}, home, table)
	if err != nil {
		// An unmeasurable cache keeps Before 0 and the freed count honest at
		// 0 — the prune itself still ran.
		return map[string]int64{}
	}
	sizes := make(map[string]int64, len(table))
	for _, m := range rep.Consumers {
		sizes[m.ID] = m.Bytes
	}
	return sizes
}

// runHow runs the table's How — a command a person would type. `rm -rf` is
// special-cased to an in-process remove whose path must be one of the
// consumer's own paths, so a malformed table row can never widen into an
// arbitrary delete.
func runHow(c hostfs.Consumer) error {
	how := strings.TrimSpace(c.How)
	if how == "" {
		return fmt.Errorf("no cleanup command for %s", c.ID)
	}
	if rest, ok := strings.CutPrefix(how, "rm -rf "); ok {
		path := strings.TrimSpace(rest)
		for _, allowed := range c.Paths {
			if path == allowed {
				return os.RemoveAll(path)
			}
		}
		return fmt.Errorf("rm path %q is not one of %s's own paths", path, c.ID)
	}
	argv := strings.Fields(how)
	home, _ := os.UserHomeDir()
	tool := toolPath(argv[0], home)
	if tool == "" {
		return fmt.Errorf("%s is not installed", argv[0])
	}
	cmd := exec.Command(tool, argv[1:]...)
	cmd.Env = withToolDir(os.Environ(), tool)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v: %s", c.How, err, strings.TrimSpace(string(out)))
	}
	return nil
}
