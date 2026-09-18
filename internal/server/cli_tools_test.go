package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/communication"
)

// Decision table for ADR-0154's launch scope: CLI × families × what the
// launch already carries → what the CLI receives.
func TestToolLaunchOptionsDecisionTable(t *testing.T) {
	dir := t.TempDir()
	peerFile := filepath.Join(dir, "peer.json")
	if err := os.WriteFile(peerFile, []byte(`{"mcpServers":{"picode-messages":{"type":"http","url":"https://x/mcp"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		cli      string
		families []string
		in       communication.LaunchOptions
		existing string
		wantErr  string
		check    func(t *testing.T, out communication.LaunchOptions)
	}{
		{name: "no families is a no-op", cli: "claude-code", in: communication.LaunchOptions{Args: []string{"--x"}}, check: func(t *testing.T, out communication.LaunchOptions) {
			if strings.Join(out.Args, " ") != "--x" {
				t.Fatalf("args = %v", out.Args)
			}
		}},
		{name: "claude gets one generated file", cli: "claude-code", families: []string{"computer"}, check: func(t *testing.T, out communication.LaunchOptions) {
			if len(out.Args) != 2 || out.Args[0] != "--mcp-config" {
				t.Fatalf("args = %v", out.Args)
			}
			doc := readMCPFile(t, out.Args[1])
			srv := doc["picode-computer"].(map[string]any)
			if srv["command"] != toolBinary() || strings.Join(anyStrings(srv["args"]), " ") != "mcp computer" {
				t.Fatalf("server = %v", srv)
			}
		}},
		{name: "claude merges the peer file into the same flag", cli: "claude-code", families: []string{"browser", "computer"}, in: communication.LaunchOptions{Args: []string{"--mcp-config", peerFile}}, check: func(t *testing.T, out communication.LaunchOptions) {
			if len(out.Args) != 2 || out.Args[1] == peerFile {
				t.Fatalf("args = %v (one --mcp-config, ours)", out.Args)
			}
			doc := readMCPFile(t, out.Args[1])
			for _, name := range []string{"picode-messages", "picode-browser", "picode-computer"} {
				if _, ok := doc[name]; !ok {
					t.Fatalf("%s missing from %v", name, doc)
				}
			}
		}},
		{name: "codex gets -c overrides per family", cli: "codex", families: []string{"computer"}, in: communication.LaunchOptions{Args: []string{"--model", "x"}}, check: func(t *testing.T, out communication.LaunchOptions) {
			joined := strings.Join(out.Args, " ")
			if !strings.HasPrefix(joined, "--model x -c mcp_servers.picode-computer.command=") || !strings.HasSuffix(joined, `-c mcp_servers.picode-computer.args=["mcp","computer"]`) {
				t.Fatalf("args = %v", out.Args)
			}
		}},
		{name: "opencode merges inline configuration and keeps other keys", cli: "opencode", families: []string{"browser"}, existing: `{"theme":"dark","mcp":{"other":{"type":"remote","url":"https://o/"}}}`, check: func(t *testing.T, out communication.LaunchOptions) {
			var doc map[string]any
			if err := json.Unmarshal([]byte(out.Env["OPENCODE_CONFIG_CONTENT"]), &doc); err != nil {
				t.Fatal(err)
			}
			mcp := doc["mcp"].(map[string]any)
			if doc["theme"] != "dark" || mcp["other"] == nil {
				t.Fatalf("lost keys: %v", doc)
			}
			srv := mcp["picode-browser"].(map[string]any)
			if srv["type"] != "local" || srv["enabled"] != true || strings.Join(anyStrings(srv["command"]), " ") != toolBinary()+" mcp browser" {
				t.Fatalf("server = %v", srv)
			}
		}},
		{name: "opencode prefers the peer env over the launch env", cli: "opencode", families: []string{"computer"}, existing: `{"theme":"dark"}`, in: communication.LaunchOptions{Env: map[string]string{"OPENCODE_CONFIG_CONTENT": `{"mcp":{"picode-messages":{"type":"remote","url":"https://m/"}}}`}}, check: func(t *testing.T, out communication.LaunchOptions) {
			var doc map[string]any
			_ = json.Unmarshal([]byte(out.Env["OPENCODE_CONFIG_CONTENT"]), &doc)
			mcp := doc["mcp"].(map[string]any)
			if doc["theme"] != nil || mcp["picode-messages"] == nil || mcp["picode-computer"] == nil {
				t.Fatalf("doc = %v", doc)
			}
		}},
		{name: "opencode refuses a broken inline configuration", cli: "opencode", families: []string{"computer"}, existing: `{`, wantErr: "must be a JSON object"},
		{name: "a CLI without a mechanism is refused", cli: "grok", families: []string{"computer"}, wantErr: "not available"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := toolLaunchOptions(tc.cli, tc.families, dir, tc.in, tc.existing)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			tc.check(t, out)
		})
	}
}

func readMCPFile(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Servers map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Servers
}

func anyStrings(v any) []string {
	var out []string
	for _, x := range v.([]any) {
		out = append(out, x.(string))
	}
	return out
}

func TestToolFamiliesAndRefusals(t *testing.T) {
	if f, err := toolFamilies(clilaunch.Config{Tools: []string{"computer", "browser", "computer"}}); err != nil || strings.Join(f, ",") != "browser,computer" {
		t.Fatalf("families = %v %v", f, err)
	}
	if _, err := toolFamilies(clilaunch.Config{Tools: []string{"flying"}}); err == nil || !strings.Contains(err.Error(), "no tool family") {
		t.Fatalf("unknown family = %v", err)
	}
	claude, _ := clilaunch.Find("claude-code")
	grok, _ := clilaunch.Find("grok")
	if err := toolLaunchRefusal(claude, clilaunch.Config{Tools: []string{"computer"}}); err != nil {
		t.Fatal(err)
	}
	if err := toolLaunchRefusal(grok, clilaunch.Config{}); err != nil {
		t.Fatal("no tools, nothing to refuse")
	}
	if err := toolLaunchRefusal(grok, clilaunch.Config{Tools: []string{"computer"}}); err == nil || !strings.Contains(err.Error(), "Connectors pane") {
		t.Fatalf("grok = %v", err)
	}
	p := toolLaunchPlan(claude, []string{"computer"}, "/run")
	if p == nil || !strings.Contains(p.Summary, "--mcp-config") || len(p.Branches) != 1 || p.Files[0] != "/run/tools.mcp.json" {
		t.Fatalf("claude plan = %+v", p)
	}
	if p := toolLaunchPlan(grok, []string{"computer"}, "/run"); p == nil || !strings.Contains(p.Summary, "Not available at launch") {
		t.Fatalf("grok plan = %+v", p)
	}
	if toolLaunchPlan(claude, nil, "/run") != nil {
		t.Fatal("no families, no plan")
	}
}

// The routes refuse tools for a CLI that cannot take them at launch and
// keep them for one that can; the preview names what will be injected.
func TestToolLaunchRoutes(t *testing.T) {
	// cleanupServer gives the daemon a data dir: the PUT syncs the intercept
	// files there, never into the package directory.
	ts, _, _ := cleanupServer(t)
	cliRequest(t, ts, "PUT", "/api/clis/grok", map[string]any{"executable": "/bin/true", "args": []any{}, "env": map[string]any{}, "integration": false, "tools": []any{"computer"}}, 400)
	v := cliRequest(t, ts, "PUT", "/api/clis/claude-code", map[string]any{"executable": "/bin/true", "args": []any{}, "env": map[string]any{}, "integration": false, "tools": []any{"computer", "browser"}}, 200)
	if v["toolsCapable"] != true {
		t.Fatalf("claude-code toolsCapable = %v", v["toolsCapable"])
	}
	config := v["config"].(map[string]any)
	if strings.Join(anyStrings(config["tools"]), ",") != "computer,browser" {
		t.Fatalf("saved tools = %v", config["tools"])
	}
	plan := v["plan"].(map[string]any)
	inj := plan["toolInjection"].(map[string]any)
	if !strings.Contains(inj["summary"].(string), "browser, computer") {
		t.Fatalf("plan = %v", inj)
	}
	if strings.Join(anyStrings(plan["tools"]), ",") != "computer,browser" {
		t.Fatalf("snapshot tools = %v", plan["tools"])
	}
	rows := cliRequest(t, ts, "GET", "/api/clis", nil, 200)["clis"].([]any)
	for _, row := range rows {
		r := row.(map[string]any)
		want := r["id"] == "claude-code" || r["id"] == "codex" || r["id"] == "opencode"
		if r["toolsCapable"] != want {
			t.Fatalf("%s toolsCapable = %v", r["id"], r["toolsCapable"])
		}
	}
}
