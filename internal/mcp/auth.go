package mcp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"regexp"
	"strings"
	"time"
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

// oauthAccount is the adapter keyring account (sha256-<hex> of the server name).
func oauthAccount(name string) string {
	sum := sha256.Sum256([]byte(name))
	return "sha256-" + hex.EncodeToString(sum[:])
}

// HasOAuthTokens reports whether the adapter keyring has tokens for name.
// Presence only — the secret is never returned or logged.
func HasOAuthTokens(name string) bool {
	if name == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "secret-tool", "lookup", "service", "pi-mcp-adapter.oauth", "username", oauthAccount(name))
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return false
	}
	return bytes.Contains(out, []byte(`"accessToken"`))
}

// ApplySigned marks OAuth servers that already have a login on this machine.
func ApplySigned(rep *Report) {
	if rep == nil {
		return
	}
	for i := range rep.Servers {
		if rep.Servers[i].Auth != "oauth" || rep.Servers[i].Disabled {
			continue
		}
		rep.Servers[i].SignedIn = HasOAuthTokens(rep.Servers[i].Name)
	}
}
