// Package connectors manages guest agent CLIs' native MCP configuration
// (ADR-0150). Each CLI owns its config; PiCode edits it in place — merge by
// key, atomic writes, unknown keys and ${VAR} placeholders preserved
// verbatim — and keeps no second connection database (ADR-0075). Where a
// store is not a plain file (Claude Code's user scope inside ~/.claude.json)
// the vendor CLI command is the only read/write path. Drivers never invent
// live state: without a headless status signal a server is "configured"
// (Live stays empty).
package connectors

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
	"time"
	"unicode"

	"github.com/cfpperche/picode/internal/mcp"
)

// CLI timeouts: a hung vendor CLI must not hold an API request past this.
const cliTimeout = 30 * time.Second

// Paths locates the config a request touches. Home empty → os.UserHomeDir.
// Cwd is the workspace folder; empty skips the project scope.
type Paths struct {
	Home string
	Cwd  string
}

func (p Paths) home() string {
	if p.Home != "" {
		return p.Home
	}
	h, _ := os.UserHomeDir()
	return h
}

// Driver is one guest CLI's native MCP management surface. Method shapes
// mirror internal/mcp so the /api/mcp handlers can dispatch without
// translating payloads; reports reuse mcp.Report verbatim, so the pane sees
// the exact response shape Pi produces.
type Driver interface {
	ID() string
	List(p Paths) (mcp.Report, error)
	Add(p Paths, scope, name string, entry mcp.Entry) error
	Toggle(p Paths, scope, name string, disabled bool) error
	Remove(p Paths, scope, name string) error
}

// For returns the phase-1 guest driver for cli, or nil when cli is Pi (its
// driver is internal/mcp itself) or has no driver yet.
func For(cli string) Driver {
	switch cli {
	case "claude":
		return Claude{}
	case "codex":
		return Codex{}
	default:
		return nil
	}
}

// guestScope validates the scope a guest driver may write. Phase 1 has no
// agent layer: guests keep user (machine) and project (workspace) files.
func guestScope(scope string) (string, error) {
	switch strings.TrimSpace(scope) {
	case "", "user":
		return "user", nil
	case "project":
		return "project", nil
	case "agent":
		return "", fmt.Errorf("connectors save to this machine or this workspace — per-agent scope is not available yet")
	default:
		return "", fmt.Errorf("scope must be user or project")
	}
}

// validEntry enforces the guest dialect of the add form: a command or a URL,
// never both; env travels with commands, headers with URLs. A bearer token
// becomes the standard Authorization header — guests have no token vault of
// their own (ADR-0150: credentials stay in each CLI's store).
func validEntry(e mcp.Entry) error {
	cmd := strings.TrimSpace(e.Command)
	url := strings.TrimSpace(e.URL)
	if cmd == "" && url == "" {
		return fmt.Errorf("command or url is required")
	}
	if cmd != "" && url != "" {
		return fmt.Errorf("use command or url, not both")
	}
	if cmd != "" {
		if hasBad(cmd) {
			return fmt.Errorf("command has invalid characters")
		}
		for _, a := range e.Args {
			if hasBad(a) {
				return fmt.Errorf("args must be a single line")
			}
		}
		if len(e.Headers) > 0 {
			return fmt.Errorf("headers are for URL servers")
		}
	}
	if url != "" && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return fmt.Errorf("url must start with http:// or https://")
	}
	if e.Auth != "" && e.Auth != "oauth" && e.Auth != "bearer" {
		return fmt.Errorf("sign-in must be oauth or bearer")
	}
	if len(e.Env) > 0 && url != "" {
		return fmt.Errorf("environment is for command servers")
	}
	for k, v := range e.Env {
		if strings.TrimSpace(k) == "" || hasBad(v) {
			return fmt.Errorf("environment values must be a single line")
		}
	}
	for k, v := range e.Headers {
		if strings.TrimSpace(k) == "" || hasBad(v) || strings.Contains(v, "\n") {
			return fmt.Errorf("header values must be a single line")
		}
	}
	return nil
}

// headersOf maps the add form onto wire headers: the given headers plus the
// bearer token as the standard Authorization header. OAuth is dropped —
// the vendor CLI runs its own flow (`<cli> mcp login <name>`).
func headersOf(e mcp.Entry) map[string]string {
	out := map[string]string{}
	for k, v := range e.Headers {
		out[k] = v
	}
	if strings.TrimSpace(e.BearerToken) != "" {
		out["Authorization"] = "Bearer " + strings.TrimSpace(e.BearerToken)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// redact keeps keys and drops values so GET never echoes secrets, matching
// internal/mcp's report behavior.
func redact(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k := range m {
		out[k] = ""
	}
	return out
}

func hasBad(s string) bool {
	for _, r := range s {
		if r == ';' || r == '|' || r == '&' || r == '`' || r == '\n' || r == '\r' || unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func installed(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// runVendorCLI shells out to the guest CLI itself (ADR-0150 decision 2: the
// vendor binary stays the authority wherever no plain file exists). Stderr
// rides the error so a refusal from the CLI reaches the pane verbatim; the
// subcommand is named but not the full argv, which can carry header secrets.
func runVendorCLI(bin string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cliTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		sub := "mcp"
		if len(args) > 1 {
			sub = args[0] + " " + args[1]
		} else if len(args) > 0 {
			sub = args[0]
		}
		return stdout.String(), fmt.Errorf("%s %s failed: %s", bin, sub, msg)
	}
	return stdout.String(), nil
}

// writeJSONFile re-marshals the whole document with 2-space indent. Key order
// inside pre-existing objects can move (Go maps are unordered) — every key,
// known or unknown, survives; golden tests pin the content, not the order.
func writeJSONFile(path string, raw map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return atomicWrite(path, b)
}

// atomicWrite lands tmp+rename so a crash never leaves a half-written
// vendor config behind.
func atomicWrite(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// sortKeys pins map iteration order where output must be deterministic
// (vendor CLI argv, golden tests).
func sortKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
