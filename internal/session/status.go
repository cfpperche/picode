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

// Bar is the composer status line (cwd, git, context, cost).
type Bar struct {
	Cwd            string   `json:"cwd"`
	Branch         string   `json:"branch,omitempty"`
	Worktree       string   `json:"worktree,omitempty"`
	Cost           float64  `json:"cost"`
	ContextTokens  *int     `json:"contextTokens,omitempty"`
	ContextWindow  *int     `json:"contextWindow,omitempty"`
	ContextPercent *float64 `json:"contextPercent,omitempty"`
}

// BuildBar assembles footer facts for a workspace. sessionPath may be empty.
func BuildBar(cwd, sessionPath string, contextWindow int) Bar {
	b := Bar{Cwd: formatCwd(cwd)}
	b.Branch, b.Worktree = gitInfo(cwd)
	if sessionPath != "" {
		cost, tokens := scanUsage(sessionPath)
		b.Cost = cost
		if tokens > 0 {
			b.ContextTokens = &tokens
			if contextWindow > 0 {
				b.ContextWindow = &contextWindow
				pct := 100 * float64(tokens) / float64(contextWindow)
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

func gitInfo(cwd string) (branch, worktree string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--is-inside-work-tree").Output(); err != nil || strings.TrimSpace(string(out)) != "true" {
		return "", ""
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		branch = strings.TrimSpace(string(out))
		if branch == "HEAD" {
			branch = "detached"
		}
	}
	gitDir, err := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--git-dir").Output()
	if err != nil {
		return branch, ""
	}
	gd := filepath.ToSlash(strings.TrimSpace(string(gitDir)))
	if i := strings.LastIndex(gd, "/worktrees/"); i >= 0 {
		worktree = gd[i+len("/worktrees/"):]
	}
	return branch, worktree
}

func scanUsage(path string) (cost float64, lastTokens int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var raw map[string]any
		if json.Unmarshal(sc.Bytes(), &raw) != nil || raw["type"] != "message" {
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
		if c, _ := u["cost"].(map[string]any); c != nil {
			if v, ok := c["total"].(float64); ok {
				cost += v
			}
		}
		if v, ok := u["totalTokens"].(float64); ok && int(v) > 0 {
			lastTokens = int(v)
		}
	}
	return cost, lastTokens
}
