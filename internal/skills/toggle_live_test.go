package skills

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestSwitchLiveVendorReadBack flips one skill off and back on through
// SetEnabled and asks each installed CLI, in its own words, whether it still
// loads the skill (ADR-0196 slice 3's live gate). Every command runs in a
// throwaway HOME with the vendors' own home variables cleared, and PiCode's
// session intercepts are skipped so the real binary answers. Opt-in with
// PICODE_SKILLS_LIVE=1; a CLI that is not installed, or cannot start without
// credentials, skips with the reason.
func TestSwitchLiveVendorReadBack(t *testing.T) {
	if os.Getenv("PICODE_SKILLS_LIVE") != "1" {
		t.Skip("set PICODE_SKILLS_LIVE=1")
	}
	const probe = "picode-probe"
	cases := []struct {
		cli    string
		folder string // machine folder under HOME the CLI reads
		bin    string
		// state runs the vendor's own read-back: on reports whether the CLI
		// loads the probe, evidence is the line that says so.
		state func(ctx context.Context, h liveHome) (on bool, evidence string, err error)
	}{
		{"claude-code", ".claude/skills", "claude", claudeLoads},
		{"codex", ".agents/skills", "codex", codexLoads},
		{"opencode", ".config/opencode/skills", "opencode", openCodeLoads},
		{"omp", ".omp/agent/skills", "omp", ompLoads},
		{"grok", ".grok/skills", "grok", grokLoads},
		{"hermes", ".hermes/skills", "hermes", hermesLoads},
		{"muse", ".agents/skills", "muse", museLoads},
	}
	for _, c := range cases {
		t.Run(c.cli, func(t *testing.T) {
			bin, path := realBinary(c.bin)
			if bin == "" {
				t.Skipf("%s is not installed", c.bin)
			}
			root := t.TempDir()
			h := liveHome{home: filepath.Join(root, "home"), ws: filepath.Join(root, "ws"), bin: bin, path: path}
			for _, d := range []string{h.home, h.ws} {
				if err := os.MkdirAll(d, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			// runMuse (and anything else SetEnabled shells out to) inherits
			// the process environment: point it at the throwaway HOME too.
			t.Setenv("HOME", h.home)
			t.Setenv("PATH", path)
			for _, k := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME", "CODEX_HOME", "HERMES_HOME", "CLAUDE_CONFIG_DIR", "PICODE_TERM_ID"} {
				t.Setenv(k, "")
				os.Unsetenv(k)
			}
			writeSkill(t, filepath.Join(h.home, c.folder), probe, "---\nname: "+probe+"\ndescription: a probe PiCode switches off and on\n---\nbody\n")

			rowFor := func() Row {
				rep, err := Read(Query{CLI: c.cli, Home: h.home, Workspace: h.ws})
				if err != nil {
					t.Fatal(err)
				}
				for _, r := range rep.Rows {
					if r.Name == probe && r.Scope == Machine {
						return r
					}
				}
				t.Fatalf("no machine row for %s in %+v", probe, rep.Rows)
				return Row{}
			}
			check := func(want bool) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				on, ev, err := c.state(ctx, h)
				var skip errSkip
				if errors.As(err, &skip) {
					t.Skipf("%s: %s", c.cli, skip.why)
				}
				if err != nil {
					t.Fatalf("%s read-back: %v", c.cli, err)
				}
				t.Logf("%s says %v: %s", c.cli, on, ev)
				if on != want {
					t.Fatalf("%s: want loaded=%v, vendor says %v (%s)", c.cli, want, on, ev)
				}
			}

			check(true) // the vendor sees the probe before any switch
			for _, want := range []bool{false, true} {
				row := rowFor()
				res, err := SetEnabled(context.Background(), ToggleReq{CLI: c.cli, Home: h.home, Workspace: h.ws, Row: row, Enabled: want})
				if err != nil {
					t.Fatalf("SetEnabled(%v): %v", want, err)
				}
				if res.Enabled != want {
					t.Fatalf("SetEnabled(%v) read back %+v", want, res)
				}
				check(want)
			}
		})
	}
}

type liveHome struct {
	home, ws  string
	bin, path string
}

type errSkip struct{ why string }

func (e errSkip) Error() string { return e.why }

// env is the process environment with HOME at the throwaway home, the
// vendors' home overrides and PiCode's session variables gone.
func (h liveHome) env(extra ...string) []string {
	drop := []string{"HOME=", "PATH=", "XDG_CONFIG_HOME=", "XDG_DATA_HOME=", "XDG_STATE_HOME=", "XDG_CACHE_HOME=",
		"CODEX_HOME=", "HERMES_HOME=", "OPENCODE_", "PICODE_", "CLAUDE", "ANTHROPIC_", "GROK_", "OMP_", "MUSE_"}
	var out []string
next:
	for _, kv := range os.Environ() {
		for _, p := range drop {
			if strings.HasPrefix(kv, p) {
				continue next
			}
		}
		out = append(out, kv)
	}
	out = append(out, "HOME="+h.home, "PATH="+h.path, "NO_COLOR=1", "TERM=dumb")
	return append(out, extra...)
}

func (h liveHome) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, h.bin, args...)
	cmd.Dir = h.ws
	cmd.Env = h.env()
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(errb.String() + "\n" + out.String())
		if authFailure(msg) {
			return "", errSkip{why: "needs credentials: " + firstLine(msg)}
		}
		return "", fmt.Errorf("%s %s: %v\n%s", filepath.Base(h.bin), strings.Join(args, " "), err, msg)
	}
	return out.String(), nil
}

func authFailure(msg string) bool {
	m := strings.ToLower(msg)
	for _, w := range []string{"not logged in", "login required", "please log in", "unauthorized", "api key", "authenticate"} {
		if strings.Contains(m, w) {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	return s
}

// realBinary finds name on PATH, skipping PiCode's session intercepts (they
// add hooks that report to the running PiCode), and returns PATH without the
// intercept folders.
func realBinary(name string) (bin, path string) {
	var keep []string
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if isIntercept(filepath.Join(d, name)) {
			continue
		}
		keep = append(keep, d)
		if bin != "" {
			continue
		}
		p := filepath.Join(d, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			bin = p
		}
	}
	return bin, strings.Join(keep, string(os.PathListSeparator))
}

func isIntercept(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	b := make([]byte, 256)
	n, _ := f.Read(b)
	return bytes.Contains(b[:n], []byte("PiCode intercept"))
}

// claudeLoads reads the init event's skills: Claude Code leaves an "off"
// skill out. The API points at a closed port, so nothing leaves the machine;
// the process is killed once the init line is in.
func claudeLoads(ctx context.Context, h liveHome) (bool, string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, h.bin, "-p", "hi", "--output-format", "stream-json", "--verbose")
	cmd.Dir = h.ws
	cmd.Env = h.env("ANTHROPIC_BASE_URL=http://127.0.0.1:9", "ANTHROPIC_API_KEY=sk-dummy")
	var errb bytes.Buffer
	cmd.Stderr = &errb
	out, err := cmd.StdoutPipe()
	if err != nil {
		return false, "", err
	}
	if err := cmd.Start(); err != nil {
		return false, "", err
	}
	defer func() { cancel(); _ = cmd.Wait() }()
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		var ev struct {
			Type    string   `json:"type"`
			Subtype string   `json:"subtype"`
			Skills  []string `json:"skills"`
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil || ev.Type != "system" || ev.Subtype != "init" {
			continue
		}
		for _, s := range ev.Skills {
			if s == "picode-probe" {
				return true, fmt.Sprintf("init.skills has picode-probe (%d skills)", len(ev.Skills)), nil
			}
		}
		return false, fmt.Sprintf("init.skills lacks picode-probe: %v", ev.Skills), nil
	}
	return false, "", fmt.Errorf("no init event; stderr: %s", strings.TrimSpace(errb.String()))
}

// codexLoads reads the skills block of the prompt Codex would send.
func codexLoads(ctx context.Context, h liveHome) (bool, string, error) {
	out, err := h.run(ctx, "debug", "prompt-input", "hi")
	if err != nil {
		return false, "", err
	}
	if !strings.Contains(out, "<skills_instructions>") {
		return false, "", fmt.Errorf("no skills block in codex prompt-input:\n%.2000s", out)
	}
	if m := regexp.MustCompile(`- picode-probe: [^(]*\([^)]*\)`).FindString(out); m != "" {
		return true, "skills block: " + m, nil
	}
	return false, "skills block has no `- picode-probe:` line", nil
}

// openCodeLoads reads the merged permission.skill rule for the probe.
func openCodeLoads(ctx context.Context, h liveHome) (bool, string, error) {
	out, err := h.run(ctx, "debug", "config")
	if err != nil {
		return false, "", err
	}
	var cfg struct {
		Permission map[string]json.RawMessage `json:"permission"`
	}
	if i := strings.Index(out, "{"); i >= 0 {
		out = out[i:]
	}
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		return false, "", fmt.Errorf("opencode debug config: %v\n%.2000s", err, out)
	}
	raw, ok := cfg.Permission["skill"]
	if !ok {
		return true, "merged config has no permission.skill", nil
	}
	var all string
	if json.Unmarshal(raw, &all) == nil {
		return all != "deny", "permission.skill = " + string(raw), nil
	}
	var per map[string]string
	if err := json.Unmarshal(raw, &per); err != nil {
		return false, "", err
	}
	v, ok := per["picode-probe"]
	return !ok || v != "deny", "permission.skill = " + string(raw), nil
}

// ompLoads reads the effective skills.ignoredSkills.
func ompLoads(ctx context.Context, h liveHome) (bool, string, error) {
	out, err := h.run(ctx, "config", "get", "skills.ignoredSkills", "--json")
	if err != nil {
		return false, "", err
	}
	var got struct {
		Value []string `json:"value"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		return false, "", fmt.Errorf("omp config get: %v\n%s", err, out)
	}
	for _, g := range got.Value {
		if globMatch(g, "picode-probe") {
			return false, fmt.Sprintf("skills.ignoredSkills = %q", got.Value), nil
		}
	}
	return true, fmt.Sprintf("skills.ignoredSkills = %q", got.Value), nil
}

// grokLoads reads the probe's entry in grok inspect.
func grokLoads(ctx context.Context, h liveHome) (bool, string, error) {
	out, err := h.run(ctx, "inspect", "--json")
	if err != nil {
		return false, "", err
	}
	var rep struct {
		Skills []map[string]any `json:"skills"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		return false, "", fmt.Errorf("grok inspect: %v", err)
	}
	for _, s := range rep.Skills {
		if s["name"] == "picode-probe" {
			b, _ := json.Marshal(s)
			return s["disabled"] != true, "skills[] entry " + string(b), nil
		}
	}
	return false, "", fmt.Errorf("grok inspect lists no picode-probe skill")
}

// hermesLoads reads the probe's row of hermes skills list.
func hermesLoads(ctx context.Context, h liveHome) (bool, string, error) {
	out, err := h.run(ctx, "skills", "list")
	if err != nil {
		return false, "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, " picode-probe ") {
			continue
		}
		line = strings.TrimSpace(line)
		switch {
		case strings.Contains(line, "disabled"):
			return false, line, nil
		case strings.Contains(line, "enabled"):
			return true, line, nil
		}
		return false, "", fmt.Errorf("hermes row without a state: %s", line)
	}
	return false, "", fmt.Errorf("hermes skills list has no picode-probe row:\n%s", out)
}

// museLoads reads the probe's activation in muse skills list.
func museLoads(ctx context.Context, h liveHome) (bool, string, error) {
	out, err := h.run(ctx, "skills", "list", "--json")
	if err != nil {
		return false, "", err
	}
	var rep struct {
		Skills []struct {
			Name       string `json:"name"`
			Activation string `json:"activation"`
			Path       string `json:"path"`
			Scope      string `json:"scope"`
		} `json:"skills"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		return false, "", fmt.Errorf("muse skills list: %v", err)
	}
	for _, s := range rep.Skills {
		if s.Name == "picode-probe" {
			return s.Activation != "off", fmt.Sprintf("%s %s activation=%s", s.Scope, s.Path, s.Activation), nil
		}
	}
	return false, "", fmt.Errorf("muse lists no picode-probe skill")
}
