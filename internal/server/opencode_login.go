package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cfpperche/picode/internal/clicreds"
)

// OpenCode's credentials through PiCode's GUI (ADR-0201). OpenCode ships
// the API its own apps sign in with: `opencode serve` answers GET /provider
// (models.dev's catalog, with each provider's variables), GET /provider/auth
// (the login methods a plugin adds, with their prompts), PUT /auth/{id} (a
// key, with the prompts' answers as metadata), and POST
// /provider/{id}/oauth/authorize|callback — "auto" flows finish inside
// OpenCode, "code" flows take the code the person pastes (measured on
// 1.18.32). PiCode runs that server on localhost while the dialog needs it;
// OpenCode writes its own auth.json, and PiCode files the result in the vault.

type opencodeServe struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	base  string
	idle  *time.Timer
	cache *opencodeCatalogView
	at    time.Time
}

var ocServe opencodeServe

const opencodeIdle = 10 * time.Minute

// ensure starts `opencode serve` when it is not running and returns its URL.
func (s *opencodeServe) ensure() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil && s.cmd.ProcessState == nil {
		s.touch()
		return s.base, nil
	}
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return "", errors.New("OpenCode is not installed on this machine.")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	cmd := exec.Command(bin, "serve", "--hostname", "127.0.0.1", "--port", fmt.Sprint(port))
	cmd.Env = os.Environ()
	if home, err := os.UserHomeDir(); err == nil {
		cmd.Dir = home
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	go func() { _ = cmd.Wait() }()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	// A request that reaches `opencode serve` while it boots is accepted and
	// never answered (measured: 5 of 5 cold starts); each probe gets its own
	// short timeout, and the next one answers once it is up.
	probe := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := probe.Get(base + "/provider/auth")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				s.cmd, s.base = cmd, base
				s.touch()
				return base, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	return "", errors.New("OpenCode's server did not start.")
}

// touch pushes the idle stop back; the server is stopped after a quiet spell.
func (s *opencodeServe) touch() {
	if s.idle != nil {
		s.idle.Stop()
	}
	s.idle = time.AfterFunc(opencodeIdle, s.stop)
}

func (s *opencodeServe) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.idle != nil {
		s.idle.Stop()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
	}
	s.cmd, s.base, s.cache = nil, "", nil
}

func (s *opencodeServe) call(ctx context.Context, method, path string, body any, out any) error {
	base, err := s.ensure()
	if err != nil {
		return err
	}
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		var e struct {
			Message string `json:"message"`
			Error   any    `json:"error"`
		}
		if json.Unmarshal(raw, &e) == nil && e.Message != "" {
			msg = e.Message
		}
		if msg == "" {
			msg = resp.Status
		}
		return errors.New("OpenCode: " + msg)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// opencodeMethod is one login door OpenCode offers for a provider.
type opencodeMethod struct {
	Index   int               `json:"index"`
	Type    string            `json:"type"` // oauth | api
	Label   string            `json:"label"`
	Prompts []json.RawMessage `json:"prompts,omitempty"`
}

type opencodeProviderView struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Env       []string         `json:"env,omitempty"`
	Methods   []opencodeMethod `json:"methods"`
	Connected bool             `json:"connected,omitempty"`
}

type opencodeCatalogView struct {
	Providers []opencodeProviderView `json:"providers"`
}

// catalog reads OpenCode's providers and their login methods, cached for a
// few minutes, and hands the rows to clicreds so the vault can file them.
func (s *opencodeServe) catalog(ctx context.Context) (*opencodeCatalogView, error) {
	s.mu.Lock()
	if s.cache != nil && time.Since(s.at) < 5*time.Minute {
		c := s.cache
		s.mu.Unlock()
		return c, nil
	}
	s.mu.Unlock()
	var all struct {
		All []struct {
			ID   string   `json:"id"`
			Name string   `json:"name"`
			Env  []string `json:"env"`
		} `json:"all"`
		Connected []string `json:"connected"`
	}
	if err := s.call(ctx, http.MethodGet, "/provider", nil, &all); err != nil {
		return nil, err
	}
	var auth map[string][]struct {
		Type    string            `json:"type"`
		Label   string            `json:"label"`
		Prompts []json.RawMessage `json:"prompts"`
	}
	if err := s.call(ctx, http.MethodGet, "/provider/auth", nil, &auth); err != nil {
		return nil, err
	}
	connected := map[string]bool{}
	for _, id := range all.Connected {
		connected[id] = true
	}
	view := &opencodeCatalogView{}
	var rows []clicreds.Provider
	for _, p := range all.All {
		v := opencodeProviderView{ID: p.ID, Name: p.Name, Env: p.Env, Connected: connected[p.ID]}
		if methods, ok := auth[p.ID]; ok && len(methods) > 0 {
			for i, m := range methods {
				v.Methods = append(v.Methods, opencodeMethod{Index: i, Type: m.Type, Label: m.Label, Prompts: m.Prompts})
			}
		} else {
			// No plugin: OpenCode's own default door is the key.
			v.Methods = []opencodeMethod{{Index: -1, Type: "api", Label: "API key"}}
		}
		view.Providers = append(view.Providers, v)
		row := clicreds.Provider{Provider: clicreds.OpencodeVaultID(p.ID), Name: p.Name}
		for _, m := range v.Methods {
			kind := clicreds.KindAPIKey
			if m.Type == "oauth" {
				kind = clicreds.KindOAuth
			}
			if !containsString(row.Kinds, kind) {
				row.Kinds = append(row.Kinds, kind)
			}
		}
		if len(p.Env) > 0 {
			row.Env = map[string]string{clicreds.KindAPIKey: p.Env[0]}
		}
		rows = append(rows, row)
	}
	clicreds.SetOpencodeCatalog(rows)
	s.mu.Lock()
	s.cache, s.at = view, time.Now()
	s.mu.Unlock()
	return view, nil
}

func handleOpencodeCatalog(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	view, err := ocServe.catalog(ctx)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// opencodeLogin is the one OAuth sign-in in flight.
var opencodeLogin struct {
	sync.Mutex
	running  bool
	done     bool
	err      string
	provider string
	method   int
	mode     string // auto | code
	cancel   context.CancelFunc
}

func handleOpencodeCredential(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Provider string            `json:"provider"`
			Method   int               `json:"method"`
			Type     string            `json:"type"`
			Key      string            `json:"key"`
			Inputs   map[string]string `json:"inputs"`
		}
		if !readCLIJSON(w, r, &req) {
			return
		}
		id := strings.TrimSpace(req.Provider)
		if id == "" || strings.ContainsAny(id, "/?#") {
			writeErr(w, http.StatusBadRequest, "Unknown provider.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		// The catalog rows are what lets the vault file a provider only
		// OpenCode knows; a PiCode restart since the dialog opened lost them.
		_, _ = ocServe.catalog(ctx)
		switch req.Type {
		case "api":
			key := strings.TrimSpace(req.Key)
			if key == "" {
				writeErr(w, http.StatusBadRequest, "An API key is required.")
				return
			}
			body := map[string]any{"type": "api", "key": key}
			if len(req.Inputs) > 0 {
				body["metadata"] = req.Inputs
			}
			if err := ocServe.call(ctx, http.MethodPut, "/auth/"+id, body, nil); err != nil {
				writeErr(w, http.StatusBadGateway, err.Error())
				return
			}
			fileCLILogin("opencode", clicreds.OpencodeVaultID(id))
			writeJSON(w, http.StatusOK, map[string]any{"done": true})
		case "oauth":
			startOpencodeOAuth(w, id, req.Method, req.Inputs)
		default:
			writeErr(w, http.StatusBadRequest, "Unknown credential kind.")
		}
	}
}

func startOpencodeOAuth(w http.ResponseWriter, id string, method int, inputs map[string]string) {
	opencodeLogin.Lock()
	if opencodeLogin.running {
		opencodeLogin.Unlock()
		writeErr(w, http.StatusConflict, "An OpenCode sign-in is already in progress. Finish or cancel it first.")
		return
	}
	opencodeLogin.running, opencodeLogin.done, opencodeLogin.err = true, false, ""
	opencodeLogin.Unlock()
	finish := func(errText string) {
		opencodeLogin.Lock()
		opencodeLogin.running, opencodeLogin.done, opencodeLogin.err = false, true, errText
		opencodeLogin.Unlock()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	body := map[string]any{"method": method}
	if len(inputs) > 0 {
		body["inputs"] = inputs
	}
	var auth struct {
		URL          string `json:"url"`
		Method       string `json:"method"`
		Instructions string `json:"instructions"`
	}
	err := ocServe.call(ctx, http.MethodPost, "/provider/"+id+"/oauth/authorize", body, &auth)
	cancel()
	if err != nil {
		finish(err.Error())
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	wait, stop := context.WithTimeout(context.Background(), 10*time.Minute)
	opencodeLogin.Lock()
	opencodeLogin.provider, opencodeLogin.method, opencodeLogin.mode, opencodeLogin.cancel = id, method, auth.Method, stop
	opencodeLogin.Unlock()
	// An "auto" flow finishes inside OpenCode (a device code, a local
	// callback): the callback call returns once it has.
	if auth.Method == "auto" {
		go func() {
			defer stop()
			var ok bool
			if err := ocServe.call(wait, http.MethodPost, "/provider/"+id+"/oauth/callback", map[string]any{"method": method}, &ok); err != nil {
				if errors.Is(wait.Err(), context.Canceled) {
					finish("The sign-in was cancelled.")
				} else if errors.Is(wait.Err(), context.DeadlineExceeded) {
					finish("The sign-in expired. Start it again.")
				} else {
					finish(err.Error())
				}
				return
			}
			if !ok {
				finish("OpenCode did not accept the sign-in.")
				return
			}
			fileCLILogin("opencode", clicreds.OpencodeVaultID(id))
			finish("")
		}()
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": auth.URL, "mode": auth.Method, "instructions": auth.Instructions})
}

// handleOpencodeCode passes the code a "code" flow's page shows.
func handleOpencodeCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if !readCLIJSON(w, r, &req) {
		return
	}
	code := strings.TrimSpace(req.Code)
	opencodeLogin.Lock()
	running, mode, id, method := opencodeLogin.running, opencodeLogin.mode, opencodeLogin.provider, opencodeLogin.method
	opencodeLogin.Unlock()
	if !running || mode != "code" {
		writeErr(w, http.StatusConflict, "The sign-in is not waiting for a code. Start it again.")
		return
	}
	if code == "" {
		writeErr(w, http.StatusBadRequest, "Paste the code the page shows.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	var ok bool
	err := ocServe.call(ctx, http.MethodPost, "/provider/"+id+"/oauth/callback", map[string]any{"method": method, "code": code}, &ok)
	opencodeLogin.Lock()
	opencodeLogin.running, opencodeLogin.done = false, true
	switch {
	case err != nil:
		opencodeLogin.err = err.Error()
	case !ok:
		opencodeLogin.err = "OpenCode did not accept that code."
	default:
		opencodeLogin.err = ""
	}
	errText := opencodeLogin.err
	opencodeLogin.Unlock()
	if errText != "" {
		writeErr(w, http.StatusBadRequest, errText)
		return
	}
	fileCLILogin("opencode", clicreds.OpencodeVaultID(id))
	writeJSON(w, http.StatusOK, map[string]any{"done": true})
}

func handleOpencodeLoginStatus(w http.ResponseWriter, r *http.Request) {
	opencodeLogin.Lock()
	defer opencodeLogin.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"pending": opencodeLogin.running, "done": opencodeLogin.done, "error": opencodeLogin.err, "mode": opencodeLogin.mode})
}

func handleOpencodeLoginCancel(w http.ResponseWriter, r *http.Request) {
	opencodeLogin.Lock()
	cancel, running := opencodeLogin.cancel, opencodeLogin.running
	opencodeLogin.Unlock()
	if running {
		if cancel != nil {
			cancel()
		}
		opencodeLogin.Lock()
		if opencodeLogin.mode == "code" {
			opencodeLogin.running, opencodeLogin.done, opencodeLogin.err = false, true, "The sign-in was cancelled."
		}
		opencodeLogin.Unlock()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
