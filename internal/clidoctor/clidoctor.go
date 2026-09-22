// Package clidoctor answers two questions about a CLI's configuration in one
// folder, read-only: what is actually in force, and what is wrong with it
// (slice 4 of docs/benchmarks/2026-09-22-omp-helpers-placement.md).
//
// The resolved list is the CLI's own answer, never a transcription: omp
// publishes every setting it knows through `omp config list --json` — about
// five hundred keys with type and description, 0.6 s, and no write to either
// file (measured 2026-09-22 on 18.2.8). PiCode only adds where each value
// comes from, by reading the two files omp layers, and the findings below.
//
// A finding is one line and, where one exists, one action. Every finding is a
// fact PiCode measured; none is advice about taste.
package clidoctor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const readTimeout = 20 * time.Second

// Entry is one setting as the CLI resolves it in this folder.
type Entry struct {
	Key         string `json:"key"`
	Value       any    `json:"value"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	// Source is the layer that sets the key: "project", "user", or "" when
	// neither file does (the CLI's default, or a source PiCode does not read,
	// such as an overlay or a foreign settings file omp also merges).
	Source   string `json:"source,omitempty"`
	Redacted bool   `json:"redacted,omitempty"`
}

// Finding is one measured problem, with the file it is about when there is one.
type Finding struct {
	ID       string   `json:"id"`
	Severity string   `json:"severity"` // "warn" or "info"
	Text     string   `json:"text"`
	File     string   `json:"file,omitempty"`
	Keys     []string `json:"keys,omitempty"`
	// Setting names a Settings row that fixes it, when PiCode has one.
	Setting string `json:"setting,omitempty"`
}

// Report is one read.
type Report struct {
	CLI      string    `json:"cli"`
	Dir      string    `json:"dir,omitempty"`
	Entries  []Entry   `json:"entries"`
	Findings []Finding `json:"findings"`
}

var supported = []string{"omp"}

func Supports(cli string) bool {
	for _, id := range supported {
		if id == cli {
			return true
		}
	}
	return false
}

func Supported() []string { return append([]string{}, supported...) }

// Paths are the inputs a read needs. Home empty → os.UserHomeDir; Dir empty
// answers for the machine, with no workspace layer.
type Paths struct {
	Home string
	Dir  string
	// List runs the CLI's own listing; tests replace it.
	List func(ctx context.Context, dir string) ([]byte, error)
	// Tracked reports whether a file is committed in dir's repository; tests
	// replace it.
	Tracked func(dir, file string) bool
}

func (p Paths) agentDir() string {
	if v := os.Getenv("PI_CODING_AGENT_DIR"); v != "" {
		return v
	}
	h := p.Home
	if h == "" {
		h, _ = os.UserHomeDir()
	}
	return filepath.Join(h, ".omp", "agent")
}

func ompList(ctx context.Context, dir string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "omp", "config", "list", "--json")
	if dir != "" {
		cmd.Dir = dir
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("omp config list: %s", msg)
	}
	// omp exits 0 even when it quarantines a config it cannot parse (measured
	// 2026-09-22), and says so only on stderr — which the quarantine finding
	// catches from the files instead.
	return out.Bytes(), nil
}

func gitTracked(dir, file string) bool {
	if dir == "" {
		return false
	}
	cmd := exec.Command("git", "-C", dir, "ls-files", "--error-unmatch", file)
	return cmd.Run() == nil
}

// Read builds the report for one CLI in one folder.
func Read(ctx context.Context, cli string, p Paths) (Report, error) {
	if !Supports(cli) {
		return Report{}, fmt.Errorf("PiCode has no checks for %s", cli)
	}
	if p.List == nil {
		p.List = ompList
	}
	if p.Tracked == nil {
		p.Tracked = gitTracked
	}
	raw, err := p.List(ctx, p.Dir)
	if err != nil {
		return Report{}, err
	}
	var listed map[string]struct {
		Value       any    `json:"value"`
		Type        string `json:"type"`
		Description string `json:"description"`
		Redacted    bool   `json:"redacted"`
	}
	if err := json.Unmarshal(raw, &listed); err != nil {
		return Report{}, fmt.Errorf("omp config list answered something PiCode could not read: %w", err)
	}

	userFile := filepath.Join(p.agentDir(), "config.yml")
	projectFile := ""
	if p.Dir != "" {
		projectFile = filepath.Join(p.Dir, ".omp", "config.yml")
	}
	user := readLayer(userFile, "the global file")
	project := readLayer(projectFile, "this workspace's file")

	known := map[string]bool{}
	for k := range listed {
		known[k] = true
	}
	rep := Report{CLI: cli, Dir: p.Dir, Entries: make([]Entry, 0, len(listed)), Findings: []Finding{}}
	for k, v := range listed {
		e := Entry{Key: k, Value: v.Value, Type: v.Type, Description: v.Description, Redacted: v.Redacted}
		switch {
		case project.sets[k]:
			e.Source = "project"
		case user.sets[k]:
			e.Source = "user"
		}
		if e.Redacted || secretish.MatchString(k) {
			e.Redacted = true
			if e.Value != nil && e.Value != "" {
				e.Value = "••••••"
			}
		}
		rep.Entries = append(rep.Entries, e)
	}
	sort.Slice(rep.Entries, func(i, j int) bool { return rep.Entries[i].Key < rep.Entries[j].Key })

	for _, l := range []layerRead{project, user} {
		if l.path == "" {
			continue
		}
		rep.Findings = append(rep.Findings, l.findings(known)...)
	}

	if p.Dir != "" {
		if legacy := filepath.Join(p.Dir, ".omp", "settings.json"); exists(legacy) {
			rep.Findings = append(rep.Findings, Finding{ID: "legacy-project-json", Severity: "info", File: legacy,
				Text: "This workspace still has the older .omp/settings.json. Omp reads it, and .omp/config.yml wins where both set a key."})
		}
		if exists(filepath.Join(p.Dir, ".pi")) && !exists(filepath.Join(p.Dir, ".omp")) {
			rep.Findings = append(rep.Findings, Finding{ID: "pi-only", Severity: "info",
				Text: "This folder has Pi settings (.pi/), which Omp does not read."})
		}
		if p.Tracked(p.Dir, ".env") {
			rep.Findings = append(rep.Findings, Finding{ID: "env-tracked", Severity: "warn", File: filepath.Join(p.Dir, ".env"),
				Text: "A .env file is committed to this repository, and Omp reads API keys from it."})
		}
	}
	if v, ok := listed["tools.approvalMode"]; ok && v.Value == "yolo" && !project.sets["tools.approvalMode"] && !user.sets["tools.approvalMode"] {
		rep.Findings = append(rep.Findings, Finding{ID: "approval-unset", Severity: "warn", Setting: "tools.approvalMode",
			Text: "No approval mode is set, so Omp approves every tool call without asking (its default)."})
	}
	return rep, nil
}

// secretish masks a value whose key names a credential, on top of what the CLI
// itself marks redacted. It reads the key's last segment, because the naive
// match masked `compaction.maxTokens` and `composer.tokenRate` too: omp 18.2.8's
// own list holds eight credential keys (`auth.broker.token`,
// `hindsight.apiToken`, `mnemopi.llmApiKey`, `searxng.basicPassword`, …) and a
// dozen innocent ones with "token" in their name.
type secretMatcher struct{}

var (
	secretWhole  = regexp.MustCompile(`(?i)^(token|password|secret|credentials?|api[_-]?key)$`)
	secretSuffix = regexp.MustCompile(`(ApiKey|Token|Password|Secret)$`)
	secretish    secretMatcher
)

func (secretMatcher) MatchString(key string) bool {
	last := key
	if i := strings.LastIndex(key, "."); i >= 0 {
		last = key[i+1:]
	}
	return secretWhole.MatchString(last) || secretSuffix.MatchString(last)
}

type layerRead struct {
	path   string
	label  string
	sets   map[string]bool
	flat   []string // keys written as one dotted name
	both   []string // keys written both flat and nested
	keys   []string // every dotted path the file holds, for the unknown check
	broken []string // quarantined copies beside the file
	bad    string   // the file does not parse
}

func readLayer(path, label string) layerRead {
	l := layerRead{path: path, label: label, sets: map[string]bool{}}
	if path == "" {
		return l
	}
	if m, _ := filepath.Glob(path + ".broken-*"); len(m) > 0 {
		sort.Strings(m)
		l.broken = m
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return l
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		l.bad = err.Error()
		return l
	}
	nested := map[string]bool{}
	flatSeen := map[string]bool{}
	var walk func(prefix string, m map[string]any, viaFlat bool)
	walk = func(prefix string, m map[string]any, viaFlat bool) {
		for k, v := range m {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			flat := viaFlat || strings.Contains(k, ".")
			l.keys = append(l.keys, path)
			l.sets[path] = true
			if flat {
				flatSeen[path] = true
			} else {
				nested[path] = true
			}
			if sub, ok := v.(map[string]any); ok {
				walk(path, sub, flat)
			}
		}
	}
	walk("", doc, false)
	for k := range flatSeen {
		if nested[k] {
			l.both = append(l.both, k)
		} else {
			l.flat = append(l.flat, k)
		}
	}
	sort.Strings(l.both)
	sort.Strings(l.flat)
	return l
}

func (l layerRead) findings(known map[string]bool) []Finding {
	var out []Finding
	for _, b := range l.broken {
		when := ""
		if st, err := os.Stat(b); err == nil {
			when = " on " + st.ModTime().Format("2006-01-02 15:04")
		}
		out = append(out, Finding{ID: "quarantined", Severity: "warn", File: b,
			Text: "Omp could not read " + l.label + " and moved that copy aside" + when + "."})
	}
	if l.bad != "" {
		out = append(out, Finding{ID: "unparseable", Severity: "warn", File: l.path,
			Text: "The YAML in " + l.label + " does not parse, and Omp moves such a file aside the next time it starts."})
		return out
	}
	// A path is known when it is a setting, or sits inside a setting whose value
	// is a map or a list (a role name under modelRoles, a chain key under
	// retry.fallbackChains).
	isKnown := func(k string) bool {
		if known[k] {
			return true
		}
		for i := len(k) - 1; i > 0; i-- {
			if k[i] == '.' && known[k[:i]] {
				return true
			}
		}
		// A parent of known settings (`retry`, `composer`) is structure.
		for s := range known {
			if strings.HasPrefix(s, k+".") {
				return true
			}
		}
		return false
	}
	var unknown []string
	for _, k := range l.keys {
		if !isKnown(k) {
			unknown = append(unknown, k)
		}
	}
	// Report only the top of an unknown subtree.
	sort.Strings(unknown)
	var tops []string
	for _, k := range unknown {
		if len(tops) > 0 && strings.HasPrefix(k, tops[len(tops)-1]+".") {
			continue
		}
		tops = append(tops, k)
	}
	if len(tops) > 0 {
		out = append(out, Finding{ID: "unknown-keys", Severity: "info", File: l.path, Keys: tops,
			Text: fmt.Sprintf("%s has %d %s that this version of Omp does not know (renamed, retired, or a typo): %s.", upper(l.label), len(tops), plural(len(tops), "key", "keys"), list(tops))})
	}
	var both, flat []string
	for _, k := range l.both {
		if known[k] {
			both = append(both, k)
		}
	}
	for _, k := range l.flat {
		if known[k] {
			flat = append(flat, k)
		}
	}
	if len(both) > 0 {
		out = append(out, Finding{ID: "written-twice", Severity: "warn", File: l.path, Keys: both,
			Text: fmt.Sprintf("%s sets %s twice, as one dotted key and nested under %s. Omp uses the nested one.", upper(l.label), list(both), strings.SplitN(both[0], ".", 2)[0])})
	}
	if len(flat) > 0 {
		out = append(out, Finding{ID: "flat-keys", Severity: "info", File: l.path, Keys: flat,
			Text: fmt.Sprintf("%s writes %s as one dotted key. Omp reads it, but PiCode's rows show it as not set.", upper(l.label), list(flat))})
	}
	return out
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func upper(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func list(keys []string) string {
	shown := keys
	extra := ""
	if len(shown) > 5 {
		shown = shown[:5]
		extra = fmt.Sprintf(" and %d more", len(keys)-5)
	}
	return strings.Join(shown, ", ") + extra
}
