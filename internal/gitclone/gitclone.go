// Package gitclone validates remote repository URLs and runs the one git
// write the server allows: cloning into a fresh directory (ADR-0034).
package gitclone

import (
	"fmt"
	"strings"
	"unicode"
)

// Remote is a validated clone source.
type Remote struct {
	URL    string // normalized clone URL (user's form minus /tree/<b> and trailing /)
	Name   string // repository name, last path segment without .git
	Branch string // branch from a pasted /tree/<branch> web URL, or ""
}

// ParseRemote validates a user-supplied repository URL. The URL becomes a
// subprocess argument, so rejection is strict: shell metacharacters and a
// leading dash (argv injection like --upload-pack) are refused, and local
// paths (including file://) are not remotes.
func ParseRemote(raw string) (Remote, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Remote{}, fmt.Errorf("repository URL is required")
	}
	if len(s) > 400 {
		return Remote{}, fmt.Errorf("repository URL is too long")
	}
	if strings.HasPrefix(s, "-") {
		return Remote{}, fmt.Errorf("repository URL cannot start with a dash")
	}
	for _, r := range s {
		if r == ';' || r == '|' || r == '&' || r == '$' || r == '`' || r == '<' || r == '>' ||
			r == '\'' || r == '"' || r == '\\' || unicode.IsSpace(r) || unicode.IsControl(r) {
			return Remote{}, fmt.Errorf("repository URL has invalid characters")
		}
	}

	scpLike := false
	switch {
	case strings.HasPrefix(s, "https://"), strings.HasPrefix(s, "http://"),
		strings.HasPrefix(s, "ssh://"), strings.HasPrefix(s, "git://"):
	case strings.HasPrefix(s, "file://"):
		return Remote{}, fmt.Errorf("local paths are not remote repositories — use Local folder instead")
	case isScpLike(s):
		scpLike = true
	default:
		return Remote{}, fmt.Errorf("that doesn't look like a git URL (https://, ssh://, or git@host:org/repo)")
	}

	branch := ""
	// Honor a URL pasted from a repo's web page: .../org/repo/tree/<branch>.
	if !scpLike {
		if i := strings.Index(s, "/tree/"); i > 0 {
			branch = strings.Trim(s[i+len("/tree/"):], "/")
			s = s[:i]
		}
	}
	s = strings.TrimRight(s, "/")

	// A remote needs a path after the host: https://host alone names nothing.
	rest := s
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	if scpLike {
		rest = rest[strings.Index(rest, ":")+1:]
	} else if i := strings.Index(rest, "/"); i >= 0 {
		rest = rest[i+1:]
	} else {
		rest = ""
	}
	if strings.TrimRight(rest, "/") == "" {
		return Remote{}, fmt.Errorf("could not tell the repository name from that URL")
	}

	name := s
	if i := strings.LastIndexAny(name, "/:"); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimSuffix(name, ".git")
	if name == "" || name == "." || name == ".." {
		return Remote{}, fmt.Errorf("could not tell the repository name from that URL")
	}
	return Remote{URL: s, Name: name, Branch: branch}, nil
}

func isScpLike(s string) bool {
	// user@host:path with no scheme; the colon must come before any slash.
	at := strings.Index(s, "@")
	colon := strings.Index(s, ":")
	slash := strings.Index(s, "/")
	return at > 0 && colon > at && (slash == -1 || colon < slash)
}

// NormalizeForCompare reduces a clone URL to host/org/repo so the same
// repository compares equal across https, ssh:// and scp-like spellings.
func NormalizeForCompare(url string) string {
	s := strings.TrimSpace(strings.ToLower(url))
	for _, p := range []string{"https://", "http://", "ssh://", "git://", "file://"} {
		s = strings.TrimPrefix(s, p)
	}
	if i := strings.Index(s, "@"); i >= 0 {
		s = s[i+1:]
	}
	// scp-like host:path → host/path
	if i := strings.Index(s, ":"); i >= 0 {
		rest := s[i+1:]
		if !strings.Contains(s[:i], "/") {
			// strip a numeric port; otherwise the colon separates host from path
			j := 0
			for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
				j++
			}
			if j > 0 && (j == len(rest) || rest[j] == '/') {
				rest = strings.TrimPrefix(rest[j:], "/")
			}
			s = s[:i] + "/" + rest
		}
	}
	s = strings.TrimRight(s, "/")
	s = strings.TrimSuffix(s, ".git")
	return s
}

// ClassifyStderr maps git clone stderr to a coarse failure class:
// "auth", "notfound", "network", or "" for anything else.
func ClassifyStderr(s string) string {
	l := strings.ToLower(s)
	switch {
	case strings.Contains(l, "authentication failed"),
		strings.Contains(l, "could not read username"),
		strings.Contains(l, "could not read password"),
		strings.Contains(l, "permission denied (publickey"),
		strings.Contains(l, "terminal prompts disabled"),
		strings.Contains(l, "host key verification failed"):
		return "auth"
	case strings.Contains(l, "not found"),
		strings.Contains(l, "does not exist"),
		strings.Contains(l, "does not appear to be a git repository"):
		return "notfound"
	case strings.Contains(l, "could not resolve host"),
		strings.Contains(l, "unable to access"),
		strings.Contains(l, "connection timed out"),
		strings.Contains(l, "connection refused"),
		strings.Contains(l, "operation timed out"):
		return "network"
	}
	return ""
}
