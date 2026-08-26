package mcp

import (
	"regexp"
	"strings"
)

var authURLRe = regexp.MustCompile(`https?://[^\s\]<>"'\\]+`)

// AuthURLFromUI pulls the first http(s) URL out of adapter UI copy.
func AuthURLFromUI(parts ...string) string {
	for _, p := range parts {
		if m := authURLRe.FindString(p); m != "" {
			return strings.TrimRight(m, ".,);")
		}
	}
	return ""
}
