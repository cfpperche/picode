package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/mcp"
	"github.com/cfpperche/picode/internal/mcptool"
)

// PiCode tools at launch (ADR-0154). A CLI agent has no per-agent scope of
// its own, so the families named in its launch settings become MCP servers
// the way ADR-0106 injects the peer-communication server: a private file
// for Claude Code, process-local overrides for Codex, inline configuration
// for OpenCode. Nothing is written to the CLI's own config files here —
// that is the Connectors pane's job (machine and workspace scope).

// hasToolLaunchMechanism says a CLI can receive MCP servers on its command
// line or environment. The others keep the file scopes only.
func hasToolLaunchMechanism(cli string) bool {
	switch cli {
	case "claude-code", "codex", "opencode":
		return true
	}
	return false
}

// managedCLILaunchOverrides is the launch default when a workspace binds a
// CLI as a principal (ADR-0159 Fatia 2). CLIs that take servers at launch
// get every picode-mcp family; the rest stay Connectors-only. Binding is
// the opt-in. Grants stay on Settings ▾ Computer / Browser (term:<id>).
func managedCLILaunchOverrides(cli string) clilaunch.Overrides {
	return fillManagedCLITools(clilaunch.Overrides{}, cli)
}

// fillManagedCLITools writes the catalog onto an unset Tools override.
// A non-nil Tools (including an empty list the person saved) is left alone.
func fillManagedCLITools(ov clilaunch.Overrides, cli string) clilaunch.Overrides {
	if ov.Tools != nil || !hasToolLaunchMechanism(cli) {
		return ov
	}
	tools := append([]string{}, mcptool.FamilyNames()...)
	ov.Tools = &tools
	return ov
}

// toolFamilies validates the families a config names against the catalog.
func toolFamilies(c clilaunch.Config) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	for _, name := range c.Tools {
		f, ok := mcptool.FamilyFor(name)
		if !ok {
			return nil, fmt.Errorf("PiCode has no tool family %q — choose from %s.", name, strings.Join(mcptool.FamilyNames(), ", "))
		}
		if !seen[f.Name] {
			seen[f.Name] = true
			out = append(out, f.Name)
		}
	}
	sort.Strings(out)
	return out, nil
}

// toolLaunchRefusal is the PUT guard: a config naming tools for a CLI that
// cannot take them at launch is refused with the way that works instead.
func toolLaunchRefusal(cli clilaunch.CLI, c clilaunch.Config) error {
	families, err := toolFamilies(c)
	if err != nil {
		return err
	}
	if len(families) > 0 && !hasToolLaunchMechanism(cli.ID) {
		return fmt.Errorf("PiCode tools at launch are not available for %s yet — add the PiCode connector in its Connectors pane instead.", cli.Name)
	}
	return nil
}

// toolServerName is the connector name in every scope, so a launch entry
// and a file entry collapse where the CLI merges by name.
func toolServerName(family string) string { return mcp.ToolPresetPrefix + family }

// toolBinary is the daemon's own executable: `picode mcp` lives in it.
func toolBinary() string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		return exe
	}
	return "picode"
}

// toolLaunchOptions folds the tool servers into the launch options the CLI
// already receives. dir is this launch's private directory.
func toolLaunchOptions(cli string, families []string, dir string, in communication.LaunchOptions, existingOpenCode string) (communication.LaunchOptions, error) {
	if len(families) == 0 {
		return in, nil
	}
	out := communication.LaunchOptions{Env: map[string]string{}, Args: append([]string{}, in.Args...)}
	for k, v := range in.Env {
		out.Env[k] = v
	}
	bin := toolBinary()
	switch cli {
	case "claude-code":
		// One --mcp-config per launch: the peer-communication file, when
		// present, is merged into ours rather than passed as a second flag.
		servers := map[string]any{}
		args := []string{}
		for i := 0; i < len(out.Args); i++ {
			if out.Args[i] == "--mcp-config" && i+1 < len(out.Args) {
				if err := mergeMCPServersFile(out.Args[i+1], servers); err != nil {
					return in, err
				}
				i++
				continue
			}
			args = append(args, out.Args[i])
		}
		for _, f := range families {
			servers[toolServerName(f)] = map[string]any{"command": bin, "args": []string{"mcp", f}}
		}
		raw, _ := json.Marshal(map[string]any{"mcpServers": servers})
		path := filepath.Join(dir, "tools.mcp.json")
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			return in, err
		}
		out.Args = append(args, "--mcp-config", path)
	case "codex":
		for _, f := range families {
			out.Args = append(out.Args, "-c", codexToolOverride(f, bin, "command"), "-c", codexToolOverride(f, bin, "args"))
		}
	case "opencode":
		servers := map[string]any{}
		for _, f := range families {
			servers[toolServerName(f)] = map[string]any{"type": "local", "command": []string{bin, "mcp", f}, "enabled": true}
		}
		existing := existingOpenCode
		if v, ok := out.Env["OPENCODE_CONFIG_CONTENT"]; ok {
			existing = v
		}
		merged, err := mergeOpenCodeServers(existing, servers)
		if err != nil {
			return in, err
		}
		out.Env["OPENCODE_CONFIG_CONTENT"] = merged
	default:
		return in, errors.New("PiCode tools at launch are not available for this CLI")
	}
	return out, nil
}

// codexToolOverride is one `-c` for Codex: `mcp_servers.<name>.command="…"`
// or `.args=["mcp","<family>"]` — TOML values, JSON-quoted strings.
func codexToolOverride(family, bin, key string) string {
	name := toolServerName(family)
	if key == "command" {
		return fmt.Sprintf("mcp_servers.%s.command=%s", name, tomlString(bin))
	}
	return fmt.Sprintf("mcp_servers.%s.args=[%s,%s]", name, tomlString("mcp"), tomlString(family))
}

// mergeMCPServersFile reads a Claude Code MCP file's servers into servers.
func mergeMCPServersFile(path string, servers map[string]any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc struct {
		Servers map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("%s is not a valid MCP file: %w", filepath.Base(path), err)
	}
	for k, v := range doc.Servers {
		servers[k] = v
	}
	return nil
}

// mergeOpenCodeServers adds servers to an inline OpenCode configuration,
// keeping every other key (the shape communication.MergeOpenCode uses).
func mergeOpenCodeServers(existing string, servers map[string]any) (string, error) {
	config := map[string]json.RawMessage{}
	if strings.TrimSpace(existing) != "" {
		if err := json.Unmarshal([]byte(existing), &config); err != nil || config == nil {
			return "", errors.New("OpenCode inline configuration must be a JSON object before adding PiCode tools")
		}
	}
	current := map[string]json.RawMessage{}
	if raw, ok := config["mcp"]; ok {
		if err := json.Unmarshal(raw, &current); err != nil || current == nil {
			return "", errors.New("OpenCode inline mcp configuration must be a JSON object before adding PiCode tools")
		}
	}
	for name, server := range servers {
		raw, err := json.Marshal(server)
		if err != nil {
			return "", err
		}
		current[name] = raw
	}
	raw, err := json.Marshal(current)
	if err != nil {
		return "", err
	}
	config["mcp"] = raw
	out, err := json.Marshal(config)
	return string(out), err
}

// toolLaunchPlan is the preview of what a launch will add for its tools.
func toolLaunchPlan(cli clilaunch.CLI, families []string, dir string) *clilaunch.IntegrationPlan {
	if len(families) == 0 {
		return nil
	}
	p := &clilaunch.IntegrationPlan{Branches: []clilaunch.Injection{}, Files: []string{}, Environment: map[string]string{}}
	names := make([]string, 0, len(families))
	for _, f := range families {
		names = append(names, toolServerName(f))
	}
	bin := toolBinary()
	switch cli.ID {
	case "claude-code":
		path := filepath.Join(dir, "tools.mcp.json")
		p.Summary = "PiCode tools via --mcp-config: " + strings.Join(families, ", ")
		p.Branches = append(p.Branches, clilaunch.Injection{When: "Every launch", Args: []string{"--mcp-config", path}})
		p.Files = append(p.Files, path)
	case "codex":
		p.Summary = "PiCode tools via -c overrides: " + strings.Join(families, ", ")
		args := []string{}
		for _, f := range families {
			args = append(args, "-c", codexToolOverride(f, bin, "command"), "-c", codexToolOverride(f, bin, "args"))
		}
		p.Branches = append(p.Branches, clilaunch.Injection{When: "Every launch", Args: args})
	case "opencode":
		p.Summary = "PiCode tools via inline configuration: " + strings.Join(families, ", ")
		p.Environment["OPENCODE_CONFIG_CONTENT"] = "mcp: " + strings.Join(names, ", ")
	default:
		p.Summary = "Not available at launch for " + cli.Name + " — add the PiCode connector in its Connectors pane (workspace scope)"
	}
	return p
}
