package gitclone

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CloneError carries the failure class and the tail of git's stderr.
type CloneError struct {
	Class  string // "auth" | "notfound" | "network" | ""
	Stderr string
}

func (e *CloneError) Error() string {
	if e.Stderr != "" {
		return e.Stderr
	}
	return "git clone failed"
}

const stderrTail = 4 << 10

// Clone runs git clone into dest. Credentials come from the host's own git
// setup; every interactive prompt is disabled so a missing credential fails
// fast instead of hanging the request.
func Clone(ctx context.Context, rem Remote, dest string) error {
	args := []string{"clone"}
	if rem.Branch != "" {
		args = append(args, "--branch", rem.Branch)
	}
	args = append(args, "--", rem.URL, dest)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=/bin/true",
		"SSH_ASKPASS=/bin/true",
		"SSH_ASKPASS_REQUIRE=never",
		"GIT_SSH_COMMAND=ssh -oBatchMode=yes",
	)
	var errb strings.Builder
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		tail := strings.TrimSpace(errb.String())
		if len(tail) > stderrTail {
			tail = tail[len(tail)-stderrTail:]
		}
		return &CloneError{Class: ClassifyStderr(tail), Stderr: tail}
	}
	return nil
}

// SameOrigin reports whether dir is already a clone of url (its origin
// remote names the same repository under NormalizeForCompare).
func SameOrigin(dir, url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	origin := strings.TrimSpace(string(out))
	return origin != "" && NormalizeForCompare(origin) == NormalizeForCompare(url)
}

// DirUsable stats dest: whether it exists and, if a directory, whether it
// is empty. A non-directory reports exists with empty=false.
func DirUsable(dest string) (exists, empty bool) {
	st, err := os.Stat(dest)
	if err != nil {
		return false, false
	}
	if !st.IsDir() {
		return true, false
	}
	ents, err := os.ReadDir(dest)
	if err != nil {
		return true, false
	}
	return true, len(ents) == 0
}
