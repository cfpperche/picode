package session

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Bar is the composer status line (cwd, git, context, tokens, cost, session).
type Bar struct {
	Cwd            string   `json:"cwd"`
	Branch         string   `json:"branch,omitempty"`
	Worktree       string   `json:"worktree,omitempty"`
	Dirty          bool     `json:"dirty"`
	SessionName    string   `json:"sessionName,omitempty"`
	Cost           float64  `json:"cost"`
	Input          int      `json:"input,omitempty"`
	Output         int      `json:"output,omitempty"`
	CacheRead      int      `json:"cacheRead,omitempty"`
	CacheWrite     int      `json:"cacheWrite,omitempty"`
	CacheHit       *float64 `json:"cacheHit,omitempty"`
	ContextTokens  *int     `json:"contextTokens,omitempty"`
	ContextWindow  *int     `json:"contextWindow,omitempty"`
	ContextPercent *float64 `json:"contextPercent,omitempty"`
	AutoCompact    bool     `json:"autoCompact"`
}

// BuildBar assembles footer facts for a workspace. sessionPath may be empty.
func BuildBar(cwd, sessionPath string, contextWindow int) Bar {
	b := Bar{Cwd: formatCwd(cwd), AutoCompact: autoCompactEnabled()}
	b.Branch, b.Worktree, b.Dirty = gitInfo(cwd)
	if sessionPath != "" {
		u := scanUsage(sessionPath)
		b.Cost = u.cost
		b.Input = u.input
		b.Output = u.output
		b.CacheRead = u.cacheRead
		b.CacheWrite = u.cacheWrite
		b.CacheHit = u.cacheHit
		b.SessionName = u.name
		if u.lastTokens > 0 {
			t := u.lastTokens
			b.ContextTokens = &t
			if contextWindow > 0 {
				b.ContextWindow = &contextWindow
				pct := 100 * float64(t) / float64(contextWindow)
				b.ContextPercent = &pct
			}
		}
	}
	if contextWindow > 0 && b.ContextWindow == nil {
		b.ContextWindow = &contextWindow
	}
	return b
}

// ParseContextWindow turns catalog strings like "200K" / "1M" into tokens.
func ParseContextWindow(s string) int {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0
	}
	mult := 1
	switch {
	case strings.HasSuffix(s, "M"):
		mult = 1_000_000
		s = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "K"):
		mult = 1_000
		s = strings.TrimSuffix(s, "K")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int(f * float64(mult))
}

func formatCwd(cwd string) string {
	clean := filepath.Clean(cwd)
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if clean == home {
			return "~"
		}
		if strings.HasPrefix(clean, home+string(os.PathSeparator)) {
			return "~" + clean[len(home):]
		}
	}
	return clean
}

func gitInfo(cwd string) (branch, worktree string, dirty bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--is-inside-work-tree").Output(); err != nil || strings.TrimSpace(string(out)) != "true" {
		return "", "", false
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		branch = strings.TrimSpace(string(out))
		if branch == "HEAD" {
			branch = "detached"
		}
	}
	if gitDir, err := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--git-dir").Output(); err == nil {
		gd := filepath.ToSlash(strings.TrimSpace(string(gitDir)))
		if i := strings.LastIndex(gd, "/worktrees/"); i >= 0 {
			worktree = gd[i+len("/worktrees/"):]
		}
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", cwd, "status", "--porcelain").Output(); err == nil {
		dirty = len(strings.TrimSpace(string(out))) > 0
	}
	return branch, worktree, dirty
}

func autoCompactEnabled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return true
	}
	b, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "settings.json"))
	if err != nil {
		return true
	}
	var s struct {
		Compaction *struct {
			Enabled *bool `json:"enabled"`
		} `json:"compaction"`
	}
	if json.Unmarshal(b, &s) != nil || s.Compaction == nil || s.Compaction.Enabled == nil {
		return true
	}
	return *s.Compaction.Enabled
}

func scanUsage(path string) (out struct {
	cost                                 float64
	lastTokens                           int
	input, output, cacheRead, cacheWrite int
	cacheHit                             *float64
	name                                 string
}) {
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var raw map[string]any
		if json.Unmarshal(sc.Bytes(), &raw) != nil {
			continue
		}
		if raw["type"] == "session_info" {
			if n, _ := raw["name"].(string); n != "" {
				out.name = n
			}
			continue
		}
		if raw["type"] != "message" {
			continue
		}
		msg, _ := raw["message"].(map[string]any)
		if msg == nil {
			continue
		}
		u, _ := msg["usage"].(map[string]any)
		if u == nil {
			continue
		}
		add := func(key string) int {
			if v, ok := u[key].(float64); ok {
				return int(v)
			}
			return 0
		}
		in, ou := add("input"), add("output")
		cr, cw := add("cacheRead"), add("cacheWrite")
		out.input += in
		out.output += ou
		out.cacheRead += cr
		out.cacheWrite += cw
		if c, _ := u["cost"].(map[string]any); c != nil {
			if v, ok := c["total"].(float64); ok {
				out.cost += v
			}
		}
		if v, ok := u["totalTokens"].(float64); ok && int(v) > 0 {
			out.lastTokens = int(v)
		}
		prompt := in + cr
		if prompt > 0 && cr > 0 {
			h := 100 * float64(cr) / float64(prompt)
			out.cacheHit = &h
		}
	}
	return out
}
