package gitinfo

import (
	"os/exec"
	"testing"
)

func TestWebRemote(t *testing.T) {
	cases := []struct {
		raw, url, host, kind string
		ok                   bool
	}{
		{"https://github.com/o/r.git", "https://github.com/o/r", "GitHub", "github", true},
		{"https://github.com/o/r", "https://github.com/o/r", "GitHub", "github", true},
		{"https://x-access-token:ghp_secret@github.com/o/r.git", "https://github.com/o/r", "GitHub", "github", true},
		{"git@github.com:o/r.git", "https://github.com/o/r", "GitHub", "github", true},
		{"ssh://git@github.com:22/o/r.git", "https://github.com/o/r", "GitHub", "github", true},
		{"git@gitlab.com:group/sub/r.git", "https://gitlab.com/group/sub/r", "GitLab", "gitlab", true},
		{"git@bitbucket.org:team/r.git", "https://bitbucket.org/team/r", "Bitbucket", "bitbucket", true},
		{"git@ssh.dev.azure.com:v3/org/proj/r", "https://dev.azure.com/org/proj/_git/r", "Azure DevOps", "azure", true},
		{"https://org@dev.azure.com/org/proj/_git/r", "https://dev.azure.com/org/proj/_git/r", "Azure DevOps", "azure", true},
		{"ssh://git@git.example.com:2222/team/r.git", "https://git.example.com/team/r", "git.example.com", "", true},
		{"http://git.local.lan/team/r", "http://git.local.lan/team/r", "git.local.lan", "", true},
		{"git://git.kernel.org/pub/scm/git/git.git", "https://git.kernel.org/pub/scm/git/git", "git.kernel.org", "", true},
		{"github-work:o/r.git", "", "", "", false}, // ssh config alias: no web host to guess
		{"/srv/git/r.git", "", "", "", false},
		{"../r", "", "", "", false},
		{`C:\repos\r`, "", "", "", false},
		{"file:///srv/git/r.git", "", "", "", false},
		{"https://github.com/", "", "", "", false},
		{"", "", "", "", false},
	}
	for _, c := range cases {
		got, ok := WebRemote(c.raw)
		if ok != c.ok || got.URL != c.url || got.Host != c.host || got.Kind != c.kind {
			t.Errorf("WebRemote(%q) = %+v, %v; want url=%q host=%q kind=%q ok=%v", c.raw, got, ok, c.url, c.host, c.kind, c.ok)
		}
	}
}

func TestPickRemotePrefersOrigin(t *testing.T) {
	cases := []struct{ config, name, raw string }{
		{"remote.upstream.url git@github.com:u/r.git\nremote.origin.url git@github.com:o/r.git", "origin", "git@github.com:o/r.git"},
		{"remote.upstream.url git@github.com:u/r.git\nremote.fork.url git@github.com:f/r.git", "upstream", "git@github.com:u/r.git"},
		{"", "", ""},
	}
	for _, c := range cases {
		if name, raw := pickRemote(c.config); name != c.name || raw != c.raw {
			t.Errorf("pickRemote(%q) = %q %q; want %q %q", c.config, name, raw, c.name, c.raw)
		}
	}
}

func TestRemoteOf(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git missing")
	}
	dir := t.TempDir()
	run(t, dir, "git", "init", "-b", "main")
	if RemoteOf(dir) != nil {
		t.Fatal("a repository without a remote has no web page")
	}
	run(t, dir, "git", "remote", "add", "origin", "git@github.com:o/r.git")
	run(t, dir, "git", "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	rem := RemoteOf(dir)
	if rem == nil || rem.URL != "https://github.com/o/r" || rem.Name != "origin" || rem.DefaultBranch != "trunk" {
		t.Fatalf("%+v", rem)
	}
	if RemoteOf(t.TempDir()) != nil {
		t.Fatal("a plain folder has no remote")
	}
}
