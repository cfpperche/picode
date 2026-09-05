package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cfpperche/picode/internal/pipkg"
)

// ConnectorPackage is installation metadata, not a claim of live tools or
// authentication. The adapter owns pi.mcp loading and namespacing. Keeping
// this read-only avoids a competing package/config loader in PiCode.
type ConnectorPackage struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Scope  string `json:"scope"`
}

func connectorPackages(p Paths) []ConnectorPackage {
	out := []ConnectorPackage{}
	cwd := p.Cwd
	if p.AgentCwd != "" {
		cwd = p.AgentCwd
	}
	rep, err := pipkg.List(filepath.Dir(p.PiGlobal()), cwd)
	if err != nil {
		return out
	}
	for _, pkg := range rep.Packages {
		if pkg.InstalledPath == "" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(pkg.InstalledPath, "package.json"))
		if err != nil {
			continue
		}
		var manifest struct {
			Name string `json:"name"`
			Pi   struct {
				MCP json.RawMessage `json:"mcp"`
			} `json:"pi"`
		}
		if json.Unmarshal(raw, &manifest) != nil || manifest.Name == "" {
			continue
		}
		var one string
		var many []string
		if json.Unmarshal(manifest.Pi.MCP, &one) == nil && one != "" || json.Unmarshal(manifest.Pi.MCP, &many) == nil && len(many) > 0 {
			out = append(out, ConnectorPackage{Name: manifest.Name, Source: pkg.Source, Scope: pkg.Scope})
		}
	}
	return out
}
