package server

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// stubGh points the three runners at a scenario and resets the cache, so
// tests never touch a real gh and never leak cache between cases.
func stubGh(t *testing.T, lookErr error, authErr error, list []ghRepo, listErr error) {
	t.Helper()
	origLook, origAuth, origList := ghLookPath, ghAuthStatus, ghRepoList
	ghLookPath = func(string) (string, error) { return "/usr/bin/gh", lookErr }
	ghAuthStatus = func() error { return authErr }
	ghRepoList = func() ([]ghRepo, error) {
		if listErr != nil {
			return nil, listErr
		}
		return list, nil
	}
	t.Cleanup(func() { ghLookPath, ghAuthStatus, ghRepoList = origLook, origAuth, origList })
	ghReposCache.Lock()
	ghReposCache.at = time.Time{}
	ghReposCache.resp = ghReposResponse{}
	ghReposCache.Unlock()
}

func getRepos(t *testing.T, path string) ghReposResponse {
	t.Helper()
	ts := newTestServer(t, "cat")
	var resp ghReposResponse
	if code := getJSON(t, ts, path, &resp); code != 200 {
		t.Fatalf("repos = %d", code)
	}
	return resp
}

func TestGithubReposDecisionTable(t *testing.T) {
	good := []ghRepo{
		{NameWithOwner: "octo/proj", URL: "https://github.com/octo/proj", UpdatedAt: time.Now()},
		{NameWithOwner: "octo/priv", URL: "https://github.com/octo/priv", Private: true},
		{NameWithOwner: "", URL: "https://github.com/octo/broken"}, // gh row missing fields
	}
	cases := []struct {
		name      string
		lookErr   error
		authErr   error
		list      []ghRepo
		listErr   error
		available bool
		reason    string
		wantLen   int
	}{
		{"gh missing", errors.New("not found"), nil, nil, nil, false, "not installed", 0},
		{"not logged in", nil, errors.New("exit 1"), nil, nil, false, "not logged in", 0},
		{"list fails", nil, nil, nil, errors.New("network"), false, "could not list", 0},
		{"ok filters broken rows", nil, nil, good, nil, true, "", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubGh(t, tc.lookErr, tc.authErr, tc.list, tc.listErr)
			resp := getRepos(t, "/api/github/repos")
			if resp.Available != tc.available {
				t.Fatalf("available = %v, want %v (reason %q)", resp.Available, tc.available, resp.Reason)
			}
			if tc.reason != "" && !strings.Contains(resp.Reason, tc.reason) {
				t.Fatalf("reason = %q, want it to contain %q", resp.Reason, tc.reason)
			}
			if tc.available && len(resp.Repos) != tc.wantLen {
				t.Fatalf("repos = %d, want %d", len(resp.Repos), tc.wantLen)
			}
			if !tc.available && resp.Reason == "" {
				t.Fatal("unavailable response must carry a visible reason")
			}
		})
	}
}

func TestGithubReposCache(t *testing.T) {
	stubGh(t, nil, nil, []ghRepo{{NameWithOwner: "octo/proj", URL: "https://github.com/octo/proj"}}, nil)

	getRepos(t, "/api/github/repos")
	// Failure the list runner after the first success must not matter: the
	// cached answer wins within the TTL.
	ghRepoList = func() ([]ghRepo, error) { return nil, errors.New("boom") }
	resp := getRepos(t, "/api/github/repos")
	if !resp.Available || len(resp.Repos) != 1 {
		t.Fatalf("cache miss: %+v", resp)
	}
	// refresh=1 bypasses the cache and surfaces the fresh failure.
	resp = getRepos(t, "/api/github/repos?refresh=1")
	if resp.Available {
		t.Fatal("refresh=1 must bypass the cache")
	}
	if resp.Reason == "" {
		t.Fatal("refresh failure must carry a reason")
	}
	// A stale cache (older than the TTL) misses too.
	ghReposCache.Lock()
	ghReposCache.at = time.Now().Add(-ghReposTTL - time.Second)
	ghReposCache.Unlock()
	resp = getRepos(t, "/api/github/repos")
	if resp.Available {
		t.Fatal("stale cache must be refreshed")
	}
}
