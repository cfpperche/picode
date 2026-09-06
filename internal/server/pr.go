package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

// Pull-request state through the host's gh (ADR-0078, phase 2). PiCode never
// holds a GitHub token (ADR-0034): gh's own login is the credential, and gh
// runs in the owner's folder so the repository and the branch are its to work
// out. "No gh", "not logged in", "no GitHub remote" and "no pull request" are
// states of a 200 page — as gitstatus treats "no repository" — and the rail
// draws each as one line and one action. Answers are cached for a minute per
// folder and branch; nothing polls GitHub in the background, and an explicit
// Refresh (`?refresh=1`) bypasses the cache. The type route pre-fills a
// command in the owner's terminal as literal keystrokes, never pressing Enter:
// the human submits it where they can see it (door, not cage).
func registerPRRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/agents/{id}/pr", handleAgentPR(deps))
	mux.HandleFunc("GET /api/terminals/{id}/pr", handleTerminalPR(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/pr", handleWorkspacePR(deps))
	mux.HandleFunc("POST /api/terminals/{id}/type", handleTerminalType(deps))
}

const (
	prFields   = "number,title,url,state,isDraft,reviewDecision,statusCheckRollup,headRefName,baseRefName,additions,deletions,changedFiles,author,updatedAt"
	prCacheTTL = time.Minute
	prTimeout  = 15 * time.Second
	// typeTextMax bounds a pre-typed command: it must stay readable in the
	// prompt before the human decides to press Enter.
	typeTextMax = 2000
)

type prEntry struct {
	at     time.Time
	branch string
	page   map[string]any
}

var prCache = struct {
	sync.Mutex
	pages  map[string]prEntry
	authAt time.Time
	authOK bool
}{pages: map[string]prEntry{}}

// resetPRCache forgets every cached answer (tests swap the gh on PATH).
func resetPRCache() {
	prCache.Lock()
	prCache.pages = map[string]prEntry{}
	prCache.authAt = time.Time{}
	prCache.authOK = false
	prCache.Unlock()
}

func handleAgentPR(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if !checkFileRoot(w, r, cwd) {
			return
		}
		writeJSON(w, http.StatusOK, prPage(cwd, r.URL.Query().Get("refresh") == "1"))
	}
}

func handleTerminalPR(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		cwd := liveTermCwd(deps, r, term)
		if !checkFileRoot(w, r, cwd) {
			return
		}
		writeJSON(w, http.StatusOK, prPage(cwd, r.URL.Query().Get("refresh") == "1"))
	}
}

func handleWorkspacePR(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		if !checkFileRoot(w, r, cwd) {
			return
		}
		writeJSON(w, http.StatusOK, prPage(cwd, r.URL.Query().Get("refresh") == "1"))
	}
}

// prPage answers from the cache when the folder is still on the same branch
// and the entry is fresh; otherwise it asks gh and remembers the answer.
func prPage(cwd string, refresh bool) map[string]any {
	branch := currentBranch(cwd)
	key := canonDir(cwd)
	if !refresh {
		prCache.Lock()
		e, ok := prCache.pages[key]
		prCache.Unlock()
		if ok && e.branch == branch && time.Since(e.at) < prCacheTTL {
			return e.page
		}
	}
	page := lookupPR(cwd, branch, refresh)
	prCache.Lock()
	prCache.pages[key] = prEntry{at: time.Now(), branch: branch, page: page}
	prCache.Unlock()
	return page
}

func prBlocked(branch, reason, message string) map[string]any {
	return map[string]any{"status": "blocked", "reason": reason, "message": message, "branch": branch}
}

func lookupPR(cwd, branch string, refresh bool) map[string]any {
	if branch == "" {
		return prBlocked("", "no-git", "Not a git repository.")
	}
	gh, err := exec.LookPath("gh")
	if err != nil {
		return prBlocked(branch, "gh-missing", "GitHub CLI (gh) is not installed.")
	}
	if !ghAuthOK(gh, refresh) {
		return prBlocked(branch, "gh-unauth", "GitHub CLI is not logged in.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), prTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gh, "pr", "view", "--json", prFields)
	cmd.Dir = cwd
	cmd.Env = ghEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := firstLine(stderr.String())
		low := strings.ToLower(msg)
		switch {
		case ctx.Err() != nil:
			return prBlocked(branch, "gh-timeout", "GitHub did not answer in time.")
		case strings.Contains(low, "no pull requests found"):
			return map[string]any{"status": "none", "branch": branch}
		case strings.Contains(low, "could not determine base repo"),
			strings.Contains(low, "not a git repository"),
			strings.Contains(low, "no git remotes"),
			strings.Contains(low, "none of the git remotes"):
			return prBlocked(branch, "no-remote", "This folder has no GitHub remote.")
		case strings.Contains(low, "authentication"),
			strings.Contains(low, "http 401"),
			strings.Contains(low, "not logged"):
			return prBlocked(branch, "gh-unauth", "GitHub CLI is not logged in.")
		}
		if msg == "" {
			msg = err.Error()
		}
		return prBlocked(branch, "gh-error", msg)
	}
	var raw prRaw
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		return prBlocked(branch, "gh-error", "gh returned an unreadable answer.")
	}
	return map[string]any{"status": "ok", "branch": branch, "pr": normalizePR(raw)}
}

// ghAuthOK remembers gh's login answer for a minute: every rail read would
// otherwise pay a second gh process, and a login does not flip that often.
func ghAuthOK(gh string, refresh bool) bool {
	prCache.Lock()
	cached := !refresh && !prCache.authAt.IsZero() && time.Since(prCache.authAt) < prCacheTTL
	ok := prCache.authOK
	prCache.Unlock()
	if cached {
		return ok
	}
	ctx, cancel := context.WithTimeout(context.Background(), prTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gh, "auth", "status")
	cmd.Env = ghEnv()
	ok = cmd.Run() == nil
	prCache.Lock()
	prCache.authAt, prCache.authOK = time.Now(), ok
	prCache.Unlock()
	return ok
}

func ghEnv() []string {
	return append(os.Environ(), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1", "NO_COLOR=1", "GH_PAGER=cat")
}

// currentBranch names the checkout — a branch, or the short HEAD when
// detached — and is "" outside a repository.
func currentBranch(cwd string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", cwd, "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	if b := strings.TrimSpace(string(out)); b != "" {
		return b
	}
	out, err = exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// prRaw mirrors the fields gh pr view --json emits; only what the rail shows.
type prRaw struct {
	Number         int    `json:"number"`
	Title          string `json:"title"`
	URL            string `json:"url"`
	State          string `json:"state"`
	IsDraft        bool   `json:"isDraft"`
	ReviewDecision string `json:"reviewDecision"`
	HeadRefName    string `json:"headRefName"`
	BaseRefName    string `json:"baseRefName"`
	Additions      int    `json:"additions"`
	Deletions      int    `json:"deletions"`
	ChangedFiles   int    `json:"changedFiles"`
	UpdatedAt      string `json:"updatedAt"`
	Author         struct {
		Login string `json:"login"`
	} `json:"author"`
	StatusCheckRollup []struct {
		TypeName   string `json:"__typename"`
		Name       string `json:"name"`
		Context    string `json:"context"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		State      string `json:"state"`
		DetailsURL string `json:"detailsUrl"`
		TargetURL  string `json:"targetUrl"`
	} `json:"statusCheckRollup"`
}

// normalizePR flattens gh's answer into what the rail draws. Checks come in
// two shapes — CheckRun (status + conclusion) and StatusContext (state) —
// and both fold into passed / failed / pending / skipped, naming the failures.
func normalizePR(raw prRaw) map[string]any {
	checks := map[string]any{"total": len(raw.StatusCheckRollup), "passed": 0, "failed": 0, "pending": 0, "skipped": 0}
	failing := []map[string]string{}
	for _, c := range raw.StatusCheckRollup {
		verdict := strings.ToUpper(c.State)
		if c.TypeName == "CheckRun" || c.Conclusion != "" || c.Status != "" {
			if strings.ToUpper(c.Status) == "COMPLETED" || c.Conclusion != "" {
				verdict = strings.ToUpper(c.Conclusion)
			} else {
				verdict = "PENDING"
			}
		}
		name := c.Name
		if name == "" {
			name = c.Context
		}
		url := c.DetailsURL
		if url == "" {
			url = c.TargetURL
		}
		switch verdict {
		case "SUCCESS":
			checks["passed"] = checks["passed"].(int) + 1
		case "NEUTRAL", "SKIPPED":
			checks["skipped"] = checks["skipped"].(int) + 1
		case "FAILURE", "ERROR", "TIMED_OUT", "CANCELLED", "ACTION_REQUIRED", "STARTUP_FAILURE", "STALE":
			checks["failed"] = checks["failed"].(int) + 1
			if len(failing) < 5 {
				failing = append(failing, map[string]string{"name": name, "url": url})
			}
		default:
			checks["pending"] = checks["pending"].(int) + 1
		}
	}
	checks["failing"] = failing
	return map[string]any{
		"number":         raw.Number,
		"title":          raw.Title,
		"url":            raw.URL,
		"state":          strings.ToLower(raw.State),
		"draft":          raw.IsDraft,
		"reviewDecision": raw.ReviewDecision,
		"head":           raw.HeadRefName,
		"base":           raw.BaseRefName,
		"additions":      raw.Additions,
		"deletions":      raw.Deletions,
		"changedFiles":   raw.ChangedFiles,
		"author":         raw.Author.Login,
		"updatedAt":      raw.UpdatedAt,
		"checks":         checks,
	}
}

func handleTerminalType(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		var req struct {
			Text string `json:"text"`
			Root string `json:"root"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if msg := typeTextProblem(req.Text); msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
		// An optional root is the rail's folder: a terminal that moved away
		// must not receive a command meant for it (ADR-0074's precondition).
		// Judged before the tmux check, like the run route: a stale root is
		// the caller's problem on any machine.
		if cwd := liveTermCwd(deps, r, t); req.Root != "" && req.Root != canonDir(cwd) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "This terminal moved to " + cwd + ".", "reason": "moved", "cwd": cwd})
			return
		}
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, http.StatusServiceUnavailable, "Need tmux to type into a terminal.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		// Keystrokes go to a shell prompt, never into an editor or a
		// running command.
		if cmd := paneCommandFn(ctx, deps, tmux.ShellSessionName(t.ID)); cmd != "" && !isShell(cmd) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "This terminal is running " + cmd + ".", "reason": "foreground"})
			return
		}
		if err := deps.Tmux.TypeText(ctx, tmux.ShellSessionName(t.ID), req.Text); err != nil {
			writeErr(w, http.StatusConflict, "Open the terminal first, then try again.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// typeTextProblem says why a pre-typed command is refused: it is literal
// keystrokes the human still has to submit, so no control characters, no
// newline, nothing that reads like a tmux flag, and short enough to read.
func typeTextProblem(text string) string {
	if strings.TrimSpace(text) == "" {
		return "nothing to type"
	}
	if len(text) > typeTextMax {
		return "that command is too long to type"
	}
	if strings.HasPrefix(text, "-") {
		return "a command cannot start with a dash"
	}
	for _, r := range text {
		if r < 0x20 || r == 0x7f {
			return "control characters and newlines are not typed"
		}
	}
	return ""
}
