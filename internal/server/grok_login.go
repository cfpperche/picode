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
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/clicreds"
	"github.com/cfpperche/picode/internal/credentials"
)

// Grok's logins through PiCode's GUI (ADR-0192). Grok has no sign-in API of
// its own, but `grok login --oauth` and `grok login --device-auth` run without
// a terminal (measured on 1.0.41): each prints the page to open — and, for the
// device flow, the code — then waits, and exits 0 once Grok has written its
// own auth.json. PiCode runs that command, shows the page and the code in the
// dialog, and reports the exit. A key is XAI_API_KEY at launch, which a
// signed-in session outranks, so choosing a key signs Grok out first.

const grokKeyEnv = "XAI_API_KEY"

// grokLoginWindow bounds one sign-in, as Codex's does.
const grokLoginWindow = 10 * time.Minute

var (
	grokANSI = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	grokURL  = regexp.MustCompile(`https://[^\s"'<>]+`)
	grokCode = regexp.MustCompile(`\b[A-Z0-9]{4}-[A-Z0-9]{4,6}\b`)
)

type grokLoginState struct {
	mu      sync.Mutex
	running bool
	done    bool
	err     string
	cancel  context.CancelFunc
}

var grokLogin grokLoginState

// grokLoginArgs maps the dialog's door to Grok's own flag.
var grokLoginArgs = map[string]string{"oauth": "--oauth", "device": "--device-auth"}

func grokBinary() (string, error) { return exec.LookPath("grok") }

func handleGrokLoginStart(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Type string `json:"type"`
		}
		if !readCLIJSON(w, r, &req) {
			return
		}
		flag, ok := grokLoginArgs[req.Type]
		if !ok {
			writeErr(w, http.StatusBadRequest, "Unknown sign-in method.")
			return
		}
		bin, err := grokBinary()
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Grok is not installed on this machine.")
			return
		}
		grokLogin.mu.Lock()
		if grokLogin.running {
			grokLogin.mu.Unlock()
			writeErr(w, http.StatusConflict, "A Grok sign-in is already in progress. Finish or cancel it first.")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), grokLoginWindow)
		grokLogin.running, grokLogin.done, grokLogin.err, grokLogin.cancel = true, false, "", cancel
		grokLogin.mu.Unlock()
		finish := func(errText string) {
			grokLogin.mu.Lock()
			grokLogin.running, grokLogin.done, grokLogin.err = false, true, errText
			grokLogin.mu.Unlock()
			cancel()
		}

		cmd := exec.CommandContext(ctx, bin, "login", flag)
		// The page opens in the person's browser from the dialog; Grok must
		// not also try to open one on the machine PiCode runs on.
		cmd.Env = append(os.Environ(), "BROWSER=true")
		pr, pw := io.Pipe()
		cmd.Stdout, cmd.Stderr = pw, pw
		if err := cmd.Start(); err != nil {
			finish(err.Error())
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		type found struct{ url, code, last string }
		first := make(chan found, 1)
		go func() {
			sc := bufio.NewScanner(pr)
			var f found
			sent := false
			for sc.Scan() {
				line := strings.TrimSpace(grokANSI.ReplaceAllString(sc.Text(), ""))
				if line != "" {
					f.last = line
				}
				if f.url == "" {
					f.url = grokURL.FindString(line)
				}
				if f.code == "" {
					f.code = grokCode.FindString(line)
				}
				ready := f.url != "" && (flag == "--oauth" || f.code != "")
				if ready && !sent {
					first <- f
					sent = true
				}
			}
			if !sent {
				first <- f
			}
		}()
		exited := make(chan error, 1)
		go func() {
			err := cmd.Wait()
			_ = pw.Close()
			exited <- err
		}()

		var got found
		select {
		case got = <-first:
		case <-time.After(30 * time.Second):
			cancel()
			finish("Grok did not show a sign-in page in time.")
			writeErr(w, http.StatusBadGateway, "Grok did not show a sign-in page in time.")
			return
		}
		if got.url == "" {
			err := <-exited
			msg := got.last
			if msg == "" && err != nil {
				msg = err.Error()
			}
			if msg == "" {
				msg = "Grok ended without a sign-in page."
			}
			finish(msg)
			writeErr(w, http.StatusBadGateway, msg)
			return
		}
		go func() {
			err := <-exited
			switch {
			case err == nil:
				// Grok's own session now outranks any key: the key is no
				// longer what it runs on.
				_ = setKeyInUse(deps, "grok", "", "")
				finish("")
			case errors.Is(ctx.Err(), context.DeadlineExceeded):
				finish("The sign-in expired. Start it again.")
			case errors.Is(ctx.Err(), context.Canceled):
				finish("The sign-in was cancelled.")
			default:
				finish("Grok could not finish the sign-in.")
			}
		}()
		out := map[string]any{"type": req.Type, "url": got.url}
		if got.code != "" {
			out["userCode"] = got.code
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleGrokLoginStatus(w http.ResponseWriter, r *http.Request) {
	grokLogin.mu.Lock()
	defer grokLogin.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"pending": grokLogin.running, "done": grokLogin.done, "error": grokLogin.err})
}

func handleGrokLoginCancel(w http.ResponseWriter, r *http.Request) {
	grokLogin.mu.Lock()
	cancel, running := grokLogin.cancel, grokLogin.running
	grokLogin.mu.Unlock()
	if running && cancel != nil {
		cancel()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// useGrokKey makes a vault key the login Grok runs on: the signed-in session
// Grok holds would outrank XAI_API_KEY, so it is filed in the vault first —
// Use brings it back — and Grok's own logout clears it.
func useGrokKey(deps Deps, provider, id string) (int, error) {
	if n := liveTerminalsFor(deps, "grok"); n > 0 {
		return http.StatusConflict, errors.New("Close the running Grok terminal(s) first — signing Grok out under a running session ends it.")
	}
	if login, ok := clicreds.DetectProvider("grok", "xai"); ok {
		if _, err := credentials.Default().Import(login.Provider, login.Cred, login.Label, "imported:grok", login.Identity); err != nil {
			return http.StatusInternalServerError, err
		}
		// Grok's file names its session "<issuer>::<client id>", which the
		// vault does not keep: the file itself is kept, so Use can write a
		// session back after the logout below removed it. It is read now and
		// kept only once the logout is proven — a refused switch must not
		// replace a good copy with a file Grok cannot read.
		raw, _ := os.ReadFile(clicreds.CredentialPath("grok", "xai"))
		bin, err := grokBinary()
		if err != nil {
			return http.StatusBadRequest, errors.New("Grok is not installed on this machine.")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, bin, "logout").CombinedOutput(); err != nil {
			msg := strings.TrimSpace(grokANSI.ReplaceAllString(string(out), ""))
			if msg == "" {
				msg = err.Error()
			}
			return http.StatusBadGateway, errors.New("Grok could not sign out: " + msg)
		}
		// Grok's logout answering 0 is not proof: a session it did not
		// recognise stays in the file, and would still outrank the key.
		if _, still := clicreds.DetectProvider("grok", "xai"); still {
			return http.StatusBadGateway, errors.New("Grok still holds its own session after signing out, so the key would not take effect. Sign out in Grok, then try again.")
		}
		if len(raw) > 0 {
			if err := writeInterceptFile(grokSessionCopy(deps), raw, 0o600); err != nil {
				return http.StatusInternalServerError, err
			}
		}
	}
	if err := setKeyInUse(deps, "grok", provider, id); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

// grokCredentialEnv is the env a Grok launch gets: the chosen key, unless the
// person's own launch config sets the variable.
func grokCredentialEnv(deps Deps, have map[string]string) [][2]string {
	if have[grokKeyEnv] != "" {
		return nil
	}
	row, _, ok := keyInUse(deps, "grok")
	if !ok {
		return nil
	}
	if key := rowKey(row); key != "" {
		return [][2]string{{grokKeyEnv, key}}
	}
	return nil
}

// markGrokInUse: the chosen key is the row in use, and key rows can always
// be chosen.
func markGrokInUse(deps Deps, providers []providerView) {
	key, provider, keyed := keyInUse(deps, "grok")
	// With Grok signed out for a key, its file is gone: a session row can
	// still be written back, into the copy kept at the logout.
	live, _ := os.ReadFile(clicreds.CredentialPath("grok", "xai"))
	copyDoc, _ := os.ReadFile(grokSessionCopy(deps))
	for i := range providers {
		// The "a session outranks this key" note is moot once the key is
		// what Grok runs on: Grok was signed out for it.
		if keyed && providers[i].ID == provider {
			providers[i].Note = ""
		}
		for j := range providers[i].Accounts {
			a := &providers[i].Accounts[j]
			if a.Type == "oauth" && len(live) == 0 && len(copyDoc) > 0 {
				if row, ok, err := credentials.Default().Row(providers[i].ID, a.ID); err == nil && ok {
					_, a.Activatable = clicreds.RenderLogin("grok", providers[i].ID, row.Cred, copyDoc)
				}
			}
			if a.Type == "api_key" {
				a.Activatable = !a.Paused
				a.Active = keyed && a.ID == key.ID && providers[i].ID == provider
				continue
			}
			if keyed {
				a.Active = false
			}
		}
	}
}

// grokSessionCopy is the last Grok auth.json PiCode signed out of.
func grokSessionCopy(deps Deps) string {
	return filepath.Join(deps.DataDir, "credfiles", "grok-session.json")
}
