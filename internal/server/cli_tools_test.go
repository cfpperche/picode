package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/mcptool"
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
		env      []string
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
		{name: "delivery is injected like any other family", cli: "claude-code", families: []string{"delivery"}, check: func(t *testing.T, out communication.LaunchOptions) {
			if len(out.Args) != 2 || out.Args[0] != "--mcp-config" {
				t.Fatalf("args = %v", out.Args)
			}
			srv, _ := readMCPFile(t, out.Args[1])["picode-delivery"].(map[string]any)
			if srv == nil || srv["command"] != toolBinary() || strings.Join(anyStrings(srv["args"]), " ") != "mcp delivery" {
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
		{name: "codex gets -c overrides per family", cli: "codex", families: []string{"computer"}, in: communication.LaunchOptions{Args: []string{"--model", "x"}}, env: []string{"PICODE_TERM_ID=t1"}, check: func(t *testing.T, out communication.LaunchOptions) {
			joined := strings.Join(out.Args, " ")
			if !strings.HasPrefix(joined, "--model x -c mcp_servers.picode-computer.command=") || !strings.Contains(joined, `-c mcp_servers.picode-computer.args=["mcp","computer"]`) {
				t.Fatalf("args = %v", out.Args)
			}
		}},
		// Codex hands a stdio MCP server no environment (measured 2026-09-21),
		// so the identity the server resolves the principal from has to be
		// written into the config or every tool call answers `no identity`.
		{name: "codex carries the server identity", cli: "codex", families: []string{"delivery"}, env: []string{"PICODE_TERM_ID=t1", "PICODE_AGENT_ID=a1", "PICODE_DATA=/d"}, check: func(t *testing.T, out communication.LaunchOptions) {
			joined := strings.Join(out.Args, " ")
			for _, want := range []string{
				`-c mcp_servers.picode-delivery.command=`,
				`-c mcp_servers.picode-delivery.args=["mcp","delivery"]`,
				`-c mcp_servers.picode-delivery.env.PICODE_TERM_ID="t1"`,
				`-c mcp_servers.picode-delivery.env.PICODE_AGENT_ID="a1"`,
				`-c mcp_servers.picode-delivery.env.PICODE_DATA="/d"`,
			} {
				if !strings.Contains(joined, want) {
					t.Fatalf("missing %s in %v", want, out.Args)
				}
			}
		}},
		{name: "codex without an identity writes no env", cli: "codex", families: []string{"delivery"}, check: func(t *testing.T, out communication.LaunchOptions) {
			if strings.Contains(strings.Join(out.Args, " "), ".env.") {
				t.Fatalf("no identity, no env overrides: %v", out.Args)
			}
		}},
		// opencode passes the launch environment to its servers (measured the
		// same day), so its config stays free of identity values that would be
		// wrong the moment the same config is read outside this launch.
		{name: "opencode needs no identity in the config", cli: "opencode", families: []string{"delivery"}, env: []string{"PICODE_TERM_ID=t1", "PICODE_AGENT_ID=a1"}, check: func(t *testing.T, out communication.LaunchOptions) {
			if strings.Contains(out.Env["OPENCODE_CONFIG_CONTENT"], "PICODE_TERM_ID") {
				t.Fatalf("config carries an identity it should inherit: %s", out.Env["OPENCODE_CONFIG_CONTENT"])
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
			out, err := toolLaunchOptions(tc.cli, tc.families, dir, tc.in, tc.existing, tc.env)
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

func TestFillManagedCLITools(t *testing.T) {
	want := strings.Join(mcptool.FamilyNames(), ",")
	filled := fillManagedCLITools(clilaunch.Overrides{}, "claude-code")
	if filled.Tools == nil || strings.Join(*filled.Tools, ",") != want {
		t.Fatalf("claude default = %v", filled.Tools)
	}
	none := fillManagedCLITools(clilaunch.Overrides{}, "grok")
	if none.Tools != nil {
		t.Fatalf("grok should stay Connectors-only, got %v", none.Tools)
	}
	keep := []string{"computer"}
	kept := fillManagedCLITools(clilaunch.Overrides{Tools: &keep}, "claude-code")
	if kept.Tools == nil || strings.Join(*kept.Tools, ",") != "computer" {
		t.Fatalf("existing tools overwritten: %v", kept.Tools)
	}
	empty := []string{}
	off := fillManagedCLITools(clilaunch.Overrides{Tools: &empty}, "claude-code")
	if off.Tools == nil || len(*off.Tools) != 0 {
		t.Fatalf("explicit empty tools filled: %v", off.Tools)
	}
}

func TestToolFamiliesAndRefusals(t *testing.T) {
	if f, err := toolFamilies(clilaunch.Config{Tools: []string{"computer", "browser", "computer"}}); err != nil || strings.Join(f, ",") != "browser,computer" {
		t.Fatalf("families = %v %v", f, err)
	}
	if _, err := toolFamilies(clilaunch.Config{Tools: []string{"flying"}}); err == nil || !strings.Contains(err.Error(), "no tool family") {
		t.Fatalf("unknown family = %v", err)
	}
	// delivery joined the catalog with ADR-0171; the form learned it later.
	if f, err := toolFamilies(clilaunch.Config{Tools: []string{"delivery", "checklist", "delivery"}}); err != nil || strings.Join(f, ",") != "checklist,delivery" {
		t.Fatalf("families = %v %v", f, err)
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
	p := toolLaunchPlan(claude, []string{"computer"}, "/run", nil)
	if p == nil || !strings.Contains(p.Summary, "--mcp-config") || len(p.Branches) != 1 || p.Files[0] != "/run/tools.mcp.json" {
		t.Fatalf("claude plan = %+v", p)
	}
	if p := toolLaunchPlan(grok, []string{"computer"}, "/run", nil); p == nil || !strings.Contains(p.Summary, "Not available at launch") {
		t.Fatalf("grok plan = %+v", p)
	}
	if toolLaunchPlan(claude, nil, "/run", nil) != nil {
		t.Fatal("no families, no plan")
	}
	// The preview names the identity a launch will fill in: a person reading
	// "what will be injected" sees where the server learns who is asking.
	codex, _ := clilaunch.Find("codex")
	preview := toolLaunchPlan(codex, []string{"delivery"}, "/run", toolIdentityEnvPreview())
	if preview == nil || len(preview.Branches) != 1 {
		t.Fatalf("codex plan = %+v", preview)
	}
	joined := strings.Join(preview.Branches[0].Args, " ")
	for _, want := range []string{
		`mcp_servers.picode-delivery.env.PICODE_TERM_ID="{terminal}"`,
		`mcp_servers.picode-delivery.env.PICODE_AGENT_ID="{agent}"`,
		`mcp_servers.picode-delivery.env.PICODE_TERM_URL="{url}"`,
		`mcp_servers.picode-delivery.env.PICODE_DATA="{data}"`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("preview missing %s in %v", want, preview.Branches[0].Args)
		}
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

// TestToolFamiliesMatchTheForm is the seam between the daemon's catalog and
// the switches the launch form offers (PICODE_TOOL_FAMILIES). ADR-0171 put
// delivery in the catalog while the form kept offering four, so the family
// was reachable only by writing the config by hand. Two files, one truth
// (the shape clipkgs.TestCLIListMatchesJS established).
func TestToolFamiliesMatchTheForm(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "web", "shared", "domain", "cliLaunch.js"))
	if err != nil {
		t.Fatal(err)
	}
	block := regexp.MustCompile(`(?s)PICODE_TOOL_FAMILIES = \[(.*?)\n\]`).FindSubmatch(body)
	if block == nil {
		t.Fatal("PICODE_TOOL_FAMILIES not found in web/shared/domain/cliLaunch.js")
	}
	var js []string
	for _, m := range regexp.MustCompile(`id: "([a-z-]+)"`).FindAllStringSubmatch(string(block[1]), -1) {
		js = append(js, m[1])
	}
	if got := mcptool.FamilyNames(); strings.Join(got, ",") != strings.Join(js, ",") {
		t.Fatalf("the form and the catalog disagree:\n  daemon: %v\n  form:   %v", got, js)
	}
}
