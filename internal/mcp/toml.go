package mcp

import (
	"os"
	"regexp"
	"strings"
)

var tomlMcpHeader = regexp.MustCompile(`(?m)^\[mcp_servers\.([^\]]+)\]\s*$`)

func serversFromTomlFile(path string) map[string]map[string]any {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return serversFromToml(string(b))
}

// serversFromToml reads [mcp_servers.name] tables only (Codex / Grok).
func serversFromToml(s string) map[string]map[string]any {
	out := map[string]map[string]any{}
	idxs := tomlMcpHeader.FindAllStringSubmatchIndex(s, -1)
	for i, loc := range idxs {
		name := strings.Trim(s[loc[2]:loc[3]], `"'`)
		if name == "" {
			continue
		}
		end := len(s)
		if i+1 < len(idxs) {
			end = idxs[i+1][0]
		}
		body := s[loc[1]:end]
		entry := map[string]any{}
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
				if strings.HasPrefix(line, "[") {
					break
				}
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key := strings.TrimSpace(k)
			val := strings.TrimSpace(v)
			val = strings.Trim(val, `"'`)
			if key == "url" || key == "command" {
				entry[key] = val
			}
		}
		out[name] = entry
	}
	return out
}

func toAnyServers(m map[string]map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}
