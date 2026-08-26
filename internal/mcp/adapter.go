package mcp

import "strings"

// AdapterConfigured is true when a package source is pi-mcp-adapter.
func AdapterConfigured(sources []string) bool {
	for _, s := range sources {
		if strings.Contains(strings.ToLower(s), "pi-mcp-adapter") {
			return true
		}
	}
	return false
}
