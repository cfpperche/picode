package server

// GET /api/github/repos — repositories behind the New-workspace repo picker.
// The list comes from the machine's own gh CLI (ADR-0034's credential
// model: the user's gh, never a PiCode-held token). Read-only metadata
// only; the clone itself still runs git with host credentials.

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ghRepo is the one repository row the picker needs.
type ghRepo struct {
	NameWithOwner string    `json:"nameWithOwner"`
	Description   string    `json:"description"`
	Private       bool      `json:"isPrivate"`
	URL           string    `json:"url"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ghReposResponse carries either the list or the visible reason it is
// unavailable — the UI shows the reason as text, never a dead control.
type ghReposResponse struct {
	Available bool     `json:"available"`
	Reason    string   `json:"reason,omitempty"`
	Repos     []ghRepo `json:"repos,omitempty"`
}

// ghLookPath / ghAuthStatus / ghRepoList are swapped in tests; a real run
// needs a gh binary and a logged-in account. Each runner bounds itself:
// a hung gh must not hang the dialog.
var (
	ghLookPath   = exec.LookPath
	ghAuthStatus = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), ghAuthTimeout)
		defer cancel()
		return exec.CommandContext(ctx, "gh", "auth", "status").Run()
	}
	ghRepoList = func() ([]ghRepo, error) {
		ctx, cancel := context.WithTimeout(context.Background(), ghReposTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", "repo", "list", "--limit", "100",
			"--json", "nameWithOwner,description,isPrivate,updatedAt,url").Output()
		if err != nil {
			return nil, err
		}
		var repos []ghRepo
		if err := json.Unmarshal(out, &repos); err != nil {
			return nil, err
		}
		return repos, nil
	}
)

const (
	ghReposTTL     = 5 * time.Minute
	ghReposTimeout = 20 * time.Second
	ghAuthTimeout  = 8 * time.Second
)

// ghReposCache memoizes the last successful list so reopening the dialog
// is instant; failures stay uncached so "I just ran gh auth login" retries
// for real.
var ghReposCache struct {
	sync.Mutex
	at   time.Time
	resp ghReposResponse
}

func handleGithubRepos(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refresh := r.URL.Query().Get("refresh") == "1"
		if !refresh {
			ghReposCache.Lock()
			cached, fresh := ghReposCache.resp, time.Since(ghReposCache.at) < ghReposTTL
			ghReposCache.Unlock()
			if fresh {
				writeJSON(w, http.StatusOK, cached)
				return
			}
		}
		resp := loadGhRepos()
		if resp.Available {
			ghReposCache.Lock()
			ghReposCache.at, ghReposCache.resp = time.Now(), resp
			ghReposCache.Unlock()
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func loadGhRepos() ghReposResponse {
	if _, err := ghLookPath("gh"); err != nil {
		return ghReposResponse{Reason: "GitHub CLI (gh) is not installed on this machine"}
	}
	if err := ghAuthStatus(); err != nil {
		return ghReposResponse{Reason: "GitHub CLI is not logged in — run gh auth login"}
	}
	repos, err := ghRepoList()
	if err != nil {
		return ghReposResponse{Reason: "could not list repositories — check your connection and gh login"}
	}
	kept := repos[:0]
	for _, r := range repos {
		if strings.TrimSpace(r.NameWithOwner) != "" && strings.TrimSpace(r.URL) != "" {
			kept = append(kept, r)
		}
	}
	return ghReposResponse{Available: true, Repos: kept}
}

func registerGithubReposRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/github/repos", handleGithubRepos(deps))
}
