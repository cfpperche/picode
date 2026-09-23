package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/clicreds"
	"github.com/cfpperche/picode/internal/credentials"
)

// Hermes's credentials through PiCode's GUI (ADR-0193). Hermes adds a
// credential to its own pool with `hermes auth add <provider>`: a key read
// from stdin (never argv, where the process list would show it), or an OAuth
// sign-in that — for every provider PiCode offers — prints a page and a code
// and waits (measured on 0.21.4). The new credential goes last in Hermes's
// pool, Hermes's own default (the owner's call, 2026-09-23). PiCode runs the
// command, shows the page and code, and files what Hermes stored in the vault.

type hermesLoginState struct {
	mu       sync.Mutex
	running  bool
	done     bool
	err      string
	provider string
	cancel   context.CancelFunc
}

var hermesLogin hermesLoginState

func hermesBinary() (string, error) { return exec.LookPath("hermes") }

func hermesEnv() []string {
	// Hermes is Python: unbuffered, or its page never reaches the pipe before
	// it blocks waiting for the sign-in (measured).
	return append(os.Environ(), "PYTHONUNBUFFERED=1", "NO_COLOR=1")
}

// hermesProvider checks the provider and kind against Hermes's roster.
func hermesProvider(provider, kind string) (clicreds.Provider, error) {
	spec, ok := clicreds.For("hermes")
	if !ok {
		return clicreds.Provider{}, errors.New("Hermes is not declared.")
	}
	for _, p := range spec.Providers {
		if p.Provider == provider {
			if !containsString(p.Kinds, kind) {
				return clicreds.Provider{}, errors.New("Hermes does not take that kind of credential for this provider.")
			}
			return p, nil
		}
	}
	return clicreds.Provider{}, errors.New("Hermes does not know that provider.")
}

func handleHermesCredential(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Provider string `json:"provider"`
			Type     string `json:"type"`
			Key      string `json:"key"`
		}
		if !readCLIJSON(w, r, &req) {
			return
		}
		provider := strings.TrimSpace(req.Provider)
		kind := strings.TrimSpace(req.Type)
		if _, err := hermesProvider(provider, kind); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		bin, err := hermesBinary()
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Hermes is not installed on this machine.")
			return
		}
		addID := clicreds.HermesAddID(provider, kind)
		switch kind {
		case clicreds.KindAPIKey:
			key := strings.TrimSpace(req.Key)
			if key == "" {
				writeErr(w, http.StatusBadRequest, "An API key is required.")
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, "auth", "add", addID, "--type", "api-key")
			cmd.Env = hermesEnv()
			cmd.Stdin = strings.NewReader(key + "\n")
			out, err := cmd.CombinedOutput()
			if err != nil {
				writeErr(w, http.StatusBadGateway, hermesLastLine(string(out), "Hermes refused the key."))
				return
			}
			cred, _ := json.Marshal(map[string]string{"type": catalog.LoginAPIKey, "key": key})
			if _, err := credentials.Default().Import(provider, cred, "", "imported:hermes", ""); err != nil {
				writeVaultErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"done": true, "message": hermesLastLine(string(out), "")})
		case clicreds.KindOAuth:
			startHermesOAuth(w, bin, provider, addID)
		default:
			writeErr(w, http.StatusBadRequest, "Unknown credential kind.")
		}
	}
}

// startHermesOAuth runs Hermes's device-code sign-in, answers with its page
// and code, and files the login when Hermes exits 0.
func startHermesOAuth(w http.ResponseWriter, bin, provider, addID string) {
	hermesLogin.mu.Lock()
	if hermesLogin.running {
		hermesLogin.mu.Unlock()
		writeErr(w, http.StatusConflict, "A Hermes sign-in is already in progress. Finish or cancel it first.")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	hermesLogin.running, hermesLogin.done, hermesLogin.err, hermesLogin.provider, hermesLogin.cancel = true, false, "", provider, cancel
	hermesLogin.mu.Unlock()
	finish := func(errText string) {
		hermesLogin.mu.Lock()
		hermesLogin.running, hermesLogin.done, hermesLogin.err = false, true, errText
		hermesLogin.mu.Unlock()
		cancel()
	}
	cmd := exec.CommandContext(ctx, bin, "auth", "add", addID, "--type", "oauth", "--no-browser")
	cmd.Env = hermesEnv()
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
		sent, openSeen := false, false
		for sc.Scan() {
			line := strings.TrimSpace(grokANSI.ReplaceAllString(sc.Text(), ""))
			if line != "" {
				f.last = line
			}
			// The page to open follows an "Open…" line (or sits on it); a
			// "Portal:" line before it names the vendor, not the page.
			if strings.Contains(strings.ToLower(line), "open") {
				openSeen = true
			}
			if f.url == "" && openSeen {
				f.url = grokURL.FindString(line)
			}
			if f.code == "" {
				f.code = grokCode.FindString(line)
			}
			if f.url != "" && f.code != "" && !sent {
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
	case <-time.After(45 * time.Second):
		cancel()
		finish("Hermes did not show a sign-in page in time.")
		writeErr(w, http.StatusBadGateway, "Hermes did not show a sign-in page in time.")
		return
	}
	if got.url == "" {
		<-exited
		msg := got.last
		if msg == "" {
			msg = "Hermes ended without a sign-in page."
		}
		finish(msg)
		writeErr(w, http.StatusBadGateway, msg)
		return
	}
	go func() {
		err := <-exited
		switch {
		case err == nil:
			if login, ok := clicreds.DetectProvider("hermes", provider); ok {
				_, _ = credentials.Default().Import(provider, login.Cred, login.Label, "imported:hermes", login.Identity)
			}
			finish("")
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			finish("The sign-in expired. Start it again.")
		case errors.Is(ctx.Err(), context.Canceled):
			finish("The sign-in was cancelled.")
		default:
			finish("Hermes could not finish the sign-in.")
		}
	}()
	writeJSON(w, http.StatusOK, map[string]any{"url": got.url, "userCode": got.code})
}

func handleHermesLoginStatus(w http.ResponseWriter, r *http.Request) {
	hermesLogin.mu.Lock()
	defer hermesLogin.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"pending": hermesLogin.running, "done": hermesLogin.done, "error": hermesLogin.err, "provider": hermesLogin.provider})
}

func handleHermesLoginCancel(w http.ResponseWriter, r *http.Request) {
	hermesLogin.mu.Lock()
	cancel, running := hermesLogin.cancel, hermesLogin.running
	hermesLogin.mu.Unlock()
	if running && cancel != nil {
		cancel()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// hermesLastLine is the last meaningful line Hermes printed, for a message.
func hermesLastLine(out, fallback string) string {
	lines := strings.Split(grokANSI.ReplaceAllString(out, ""), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l != "" && !strings.HasPrefix(l, "Warning:") && !strings.Contains(l, "getpass") {
			return l
		}
	}
	return fallback
}
