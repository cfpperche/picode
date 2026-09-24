package server

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cfpperche/picode/internal/clicreds"
)

// Antigravity's Google sign-in through PiCode's GUI (ADR-0197). Antigravity has
// no login command. Measured on 1.2.9: in print mode (`agy -p`), with no login
// and a terminal on stdin, it prints Google's authorization page — whose
// callback, antigravity.google/oauth-callback, shows a code — then asks for
// that code on stdin, with a 60-second limit. PiCode runs it under a pseudo
// terminal (`script`), shows the page, passes the pasted code in, and stops
// agy the moment its token file is written: print mode would otherwise go on
// to run the prompt, so the prompt is the smallest one and a second of time.
//
// A login already in the file would skip the sign-in and run the prompt, so it
// is filed in the vault and set aside first, and put back if the sign-in fails.

const agyLoginWindow = 70 * time.Second

var agyAuthURL = regexp.MustCompile(`https://accounts\.google\.com/o/oauth2/auth\?[^\s"'<>]+`)

type agyLoginState struct {
	mu      sync.Mutex
	running bool
	done    bool
	err     string
	stdin   io.WriteCloser
	cancel  context.CancelFunc
	started time.Time
}

var agyLogin agyLoginState

func agyTokenPath() string { return clicreds.CredentialPath("agy", "google") }

// agyScriptArgs builds the `script(1)` invocation that runs the sign-in under
// a pty. util-linux takes the command with -c and flushes with -f; the BSD
// script on macOS has neither — given `-qfec` it prints its usage line
// (`script -p [-deq] [-T fmt] [file]`, which is what a macOS runner answered,
// 2026-09-24) — and takes the command after the file instead, so the shell
// line is handed to `sh -c` there.
func agyScriptArgs(goos, cmdline, devNull string) []string {
	if goos == "darwin" {
		return []string{"-q", devNull, "sh", "-c", cmdline}
	}
	return []string{"-qfec", cmdline, devNull}
}

func handleAgyLoginStart(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bin, err := exec.LookPath("agy")
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Antigravity is not installed on this machine.")
			return
		}
		script, err := exec.LookPath("script")
		if err != nil {
			writeErr(w, http.StatusBadRequest, "This machine has no `script` command (util-linux), which the sign-in needs. Sign in from a terminal instead.")
			return
		}
		if n := liveTerminalsFor(deps, "agy"); n > 0 {
			writeErr(w, http.StatusConflict, "Close the running Antigravity terminal(s) first — the sign-in replaces its login file.")
			return
		}
		agyLogin.mu.Lock()
		if agyLogin.running {
			agyLogin.mu.Unlock()
			writeErr(w, http.StatusConflict, "An Antigravity sign-in is already in progress. Finish or cancel it first.")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), agyLoginWindow)
		agyLogin.running, agyLogin.done, agyLogin.err, agyLogin.cancel, agyLogin.started = true, false, "", cancel, time.Now()
		agyLogin.mu.Unlock()

		// Set a current login aside (it is filed in the vault first).
		token := agyTokenPath()
		aside := ""
		if raw, err := os.ReadFile(token); err == nil && len(raw) > 0 {
			fileCLILogin("agy", "google")
			aside = token + ".picode-aside"
			if err := os.Rename(token, aside); err != nil {
				cancel()
				agyLogin.mu.Lock()
				agyLogin.running, agyLogin.done, agyLogin.err = false, true, err.Error()
				agyLogin.mu.Unlock()
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		restore := func() {
			if aside == "" {
				return
			}
			if _, err := os.Stat(token); errors.Is(err, os.ErrNotExist) {
				_ = os.Rename(aside, token)
			} else {
				_ = os.Remove(aside)
			}
		}
		finish := func(errText string) {
			if errText != "" {
				restore()
			} else if aside != "" {
				_ = os.Remove(aside)
			}
			agyLogin.mu.Lock()
			agyLogin.running, agyLogin.done, agyLogin.err, agyLogin.stdin = false, true, errText, nil
			agyLogin.mu.Unlock()
			cancel()
		}

		cmdline := shellQuote(bin) + " -p . --print-timeout 1s"
		cmd := exec.CommandContext(ctx, script, agyScriptArgs(runtime.GOOS, cmdline, "/dev/null")...)
		// agy opens the page itself through xdg-open (measured: a real browser
		// on the machine PiCode runs on); the dialog already opens it where
		// the person is, so agy gets openers that do nothing.
		cmd.Env = append(os.Environ(), "BROWSER=true", "NO_COLOR=1")
		if dir, err := agyNoOpenDir(deps.DataDir); err == nil {
			cmd.Env = append(cmd.Env, "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error { killAgyTree(cmd.Process.Pid); return nil }
		stdin, err := cmd.StdinPipe()
		if err != nil {
			finish(err.Error())
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		pr, pw := io.Pipe()
		cmd.Stdout, cmd.Stderr = pw, pw
		if err := cmd.Start(); err != nil {
			finish(err.Error())
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		agyLogin.mu.Lock()
		agyLogin.stdin = stdin
		agyLogin.mu.Unlock()

		urlCh := make(chan string, 1)
		lastLine := make(chan string, 1)
		go func() {
			sc := bufio.NewScanner(pr)
			sc.Buffer(make([]byte, 1<<16), 1<<20)
			sent, last, firstErr := false, "", ""
			for sc.Scan() {
				line := strings.TrimSpace(strings.ReplaceAll(grokANSI.ReplaceAllString(sc.Text(), ""), "\r", ""))
				if line != "" {
					last = line
				}
				// agy ends every failure on "authentication failed or timed
				// out": its first "Error:" line is the one that says which.
				if firstErr == "" && strings.HasPrefix(line, "Error:") {
					firstErr = line
				}
				if u := agyAuthURL.FindString(line); u != "" && !sent {
					urlCh <- u
					sent = true
				}
			}
			if !sent {
				urlCh <- ""
			}
			if firstErr != "" {
				last = firstErr
			}
			lastLine <- last
		}()

		// The token file appearing is the sign-in's end: agy is stopped
		// before print mode goes on to its prompt.
		signedIn := make(chan struct{})
		go func() {
			tick := time.NewTicker(150 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					if raw, err := os.ReadFile(token); err == nil && len(raw) > 0 {
						if _, ok := clicreds.DetectProvider("agy", "google"); ok {
							close(signedIn)
							killAgyTree(cmd.Process.Pid)
							return
						}
					}
				}
			}
		}()
		exited := make(chan error, 1)
		go func() {
			err := cmd.Wait()
			_ = pw.Close()
			exited <- err
		}()

		var url string
		select {
		case url = <-urlCh:
		case <-time.After(30 * time.Second):
			cancel()
			finish("Antigravity did not show a sign-in page in time.")
			writeErr(w, http.StatusBadGateway, "Antigravity did not show a sign-in page in time.")
			return
		}
		// The minute is Antigravity's, and it starts when the page is shown.
		agyLogin.mu.Lock()
		agyLogin.started = time.Now()
		agyLogin.mu.Unlock()
		if url == "" {
			<-exited
			msg := <-lastLine
			if msg == "" {
				msg = "Antigravity ended without a sign-in page."
			}
			finish(msg)
			writeErr(w, http.StatusBadGateway, msg)
			return
		}
		go func() {
			<-exited
			select {
			case <-signedIn:
				fileCLILogin("agy", "google")
				finish("")
				return
			default:
			}
			msg := ""
			select {
			case msg = <-lastLine:
			case <-time.After(time.Second):
			}
			low := strings.ToLower(msg)
			switch {
			case strings.Contains(low, "invalid_grant") || strings.Contains(low, "authentication failed:"):
				finish("Google did not accept that code. Get a new link and paste the code the page shows.")
			case errors.Is(ctx.Err(), context.DeadlineExceeded) || strings.Contains(low, "timed out"):
				finish("The sign-in window closed. Get a new link and paste the code within a minute.")
			case errors.Is(ctx.Err(), context.Canceled):
				finish("The sign-in was cancelled.")
			case msg != "":
				finish(msg)
			default:
				finish("Antigravity could not finish the sign-in.")
			}
		}()
		writeJSON(w, http.StatusOK, map[string]any{"url": url, "seconds": 60})
	}
}

// handleAgyLoginCode passes the code the person copied from Google's page.
func handleAgyLoginCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if !readCLIJSON(w, r, &req) {
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" || strings.ContainsAny(code, "\r\n") {
		writeErr(w, http.StatusBadRequest, "Paste the code the page shows.")
		return
	}
	agyLogin.mu.Lock()
	stdin, running := agyLogin.stdin, agyLogin.running
	agyLogin.mu.Unlock()
	if !running || stdin == nil {
		writeErr(w, http.StatusConflict, "The sign-in is no longer waiting for a code. Get a new link.")
		return
	}
	if _, err := io.WriteString(stdin, code+"\n"); err != nil {
		writeErr(w, http.StatusConflict, "The sign-in is no longer waiting for a code. Get a new link.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func handleAgyLoginStatus(w http.ResponseWriter, r *http.Request) {
	agyLogin.mu.Lock()
	defer agyLogin.mu.Unlock()
	left := 0
	if agyLogin.running {
		left = 60 - int(time.Since(agyLogin.started).Seconds())
		if left < 0 {
			left = 0
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"pending": agyLogin.running, "done": agyLogin.done, "error": agyLogin.err, "seconds": left})
}

func handleAgyLoginCancel(w http.ResponseWriter, r *http.Request) {
	agyLogin.mu.Lock()
	cancel, running := agyLogin.cancel, agyLogin.running
	agyLogin.mu.Unlock()
	if running && cancel != nil {
		cancel()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// killAgyTree stops a sign-in: `script` starts agy in a session of its own
// (measured), so killing script's group misses it. agy's children are found
// by parent id and each one's group is killed, then script's own group.
func killAgyTree(scriptPid int) {
	if scriptPid <= 0 {
		return
	}
	for _, child := range childPids(scriptPid) {
		_ = syscall.Kill(-child, syscall.SIGKILL)
		_ = syscall.Kill(child, syscall.SIGKILL)
	}
	_ = syscall.Kill(-scriptPid, syscall.SIGKILL)
}

// childPids lists the processes whose parent is pid, from /proc.
func childPids(pid int) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []int
	for _, e := range entries {
		n, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		raw, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		// The command sits in parentheses and may hold spaces: the fields
		// after the last ")" are state, ppid, ….
		i := strings.LastIndexByte(string(raw), ')')
		if i < 0 {
			continue
		}
		fields := strings.Fields(string(raw)[i+1:])
		if len(fields) > 1 {
			if ppid, err := strconv.Atoi(fields[1]); err == nil && ppid == pid {
				out = append(out, n)
			}
		}
	}
	return out
}

// agyNoOpenDir holds do-nothing xdg-open / sensible-browser / www-browser for
// agy's PATH, in PiCode's own data dir.
func agyNoOpenDir(dataDir string) (string, error) {
	if dataDir == "" {
		return "", errors.New("no data dir")
	}
	dir := filepath.Join(dataDir, "agy-noopen")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for _, name := range []string{"xdg-open", "sensible-browser", "www-browser", "gio"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			return "", err
		}
	}
	return dir, nil
}
