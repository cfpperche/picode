package gitinfo

import (
	"net/url"
	"strings"
)

// Remote is where a repository lives on the web: the page a browser opens,
// the host's name for the menu label, and the branch the remote calls its
// default (so a "create pull request" link is never offered for it).
type Remote struct {
	Name          string `json:"name"`                    // the git remote: origin, upstream…
	URL           string `json:"url"`                     // https://github.com/owner/repo
	Host          string `json:"host"`                    // GitHub, GitLab, Bitbucket, Azure DevOps, or the hostname
	Kind          string `json:"kind,omitempty"`          // github | gitlab | bitbucket | azure; "" for any other host
	DefaultBranch string `json:"defaultBranch,omitempty"` // from refs/remotes/<name>/HEAD; "" when never fetched
}

// RemoteOf reads the repository's web page from its remotes: origin when it
// has one, the first remote otherwise. Nil for a folder with no remote, or
// one whose remote is not a web host (a local path, an ssh alias).
func RemoteOf(dir string) *Remote {
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	name, raw := pickRemote(git(dir, "config", "--get-regexp", `^remote\..*\.url$`))
	if raw == "" {
		return nil
	}
	rem, ok := WebRemote(raw)
	if !ok {
		return nil
	}
	rem.Name = name
	if head := git(dir, "symbolic-ref", "--short", "refs/remotes/"+name+"/HEAD"); head != "" {
		rem.DefaultBranch = strings.TrimPrefix(head, name+"/")
	}
	return &rem
}

// pickRemote reads `git config --get-regexp` lines ("remote.origin.url X")
// and prefers origin; any other remote in file order stands in for it.
func pickRemote(config string) (name, raw string) {
	for _, line := range strings.Split(config, "\n") {
		key, val, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		n := strings.TrimSuffix(strings.TrimPrefix(key, "remote."), ".url")
		if n == "origin" {
			return n, strings.TrimSpace(val)
		}
		if raw == "" {
			name, raw = n, strings.TrimSpace(val)
		}
	}
	return name, raw
}

// WebRemote turns a git remote URL into the repository's web page. It
// understands https, ssh (both spellings), git:// and scp-like remotes, and
// drops credentials and ports on the way: a token embedded in an https
// remote must never reach the UI. Local paths, file:// and ssh host aliases
// (no dot in the host) have no web page and answer false.
func WebRemote(raw string) (Remote, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Remote{}, false
	}
	scheme, host, path := "https", "", ""
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return Remote{}, false
		}
		switch u.Scheme {
		case "http":
			scheme = "http"
		case "https", "ssh", "git", "git+ssh", "ssh+git":
		default:
			return Remote{}, false
		}
		host, path = u.Hostname(), u.Path
	} else {
		// scp-like: [user@]host:path. A leading slash or a drive letter is a
		// local path, not a host (C:\repo fails the dotted-host check below).
		colon := strings.Index(raw, ":")
		if colon <= 0 || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, ".") {
			return Remote{}, false
		}
		host, path = raw[:colon], raw[colon+1:]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
	}
	host = strings.ToLower(host)
	if !strings.Contains(host, ".") && host != "localhost" {
		return Remote{}, false
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	if path == "" {
		return Remote{}, false
	}
	rem := Remote{Host: host}
	switch {
	case host == "github.com":
		rem.Kind, rem.Host = "github", "GitHub"
	case host == "gitlab.com":
		rem.Kind, rem.Host = "gitlab", "GitLab"
	case host == "bitbucket.org":
		rem.Kind, rem.Host = "bitbucket", "Bitbucket"
	case host == "ssh.dev.azure.com" || host == "dev.azure.com" || strings.HasSuffix(host, ".visualstudio.com"):
		rem.Kind, rem.Host = "azure", "Azure DevOps"
		// ssh: v3/org/project/repo → https://dev.azure.com/org/project/_git/repo
		if parts := strings.Split(path, "/"); host == "ssh.dev.azure.com" && len(parts) == 4 && parts[0] == "v3" {
			host, path = "dev.azure.com", parts[1]+"/"+parts[2]+"/_git/"+parts[3]
		}
	}
	rem.URL = scheme + "://" + host + "/" + path
	return rem, true
}
