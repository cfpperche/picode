// Package oauth runs Claude and Codex account login the way pi's TUI does:
// PKCE, loopback callback on the provider-registered ports, write auth.json.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
)

const (
	anthropicClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e" // pi CLI (base64 in pi-ai)
	anthropicAuth     = "https://claude.ai/oauth/authorize"
	anthropicToken    = "https://platform.claude.com/v1/oauth/token"
	anthropicPort     = "53692"
	anthropicPath     = "/callback"
	anthropicRedirect = "http://localhost:53692/callback"
	anthropicScopes   = "org:create_api_key user:profile user:inference user:sessions:claude_code user:mcp_servers user:file_upload"

	codexClientID  = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexAuth      = "https://auth.openai.com/oauth/authorize"
	codexToken     = "https://auth.openai.com/oauth/token"
	codexPort      = "1455"
	codexPath      = "/auth/callback"
	codexRedirect  = "http://localhost:1455/auth/callback"
	codexScopes    = "openid profile email offline_access"
	codexAccountNS = "https://api.openai.com/auth"
)

type pending struct {
	provider string
	verifier string
	state    string
	returnTo string
	ln       net.Listener
	cancel   context.CancelFunc
	done     chan result
}

type result struct {
	err error
}

var (
	mu       sync.Mutex
	cur      *pending
	lastDone bool
	lastErr  error
)

func pkce() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

// Start begins loopback OAuth. Returns the URL to open in the browser.
func Start(provider, returnTo string) (authorizeURL, userCode string, err error) {
	mu.Lock()
	defer mu.Unlock()
	if cur != nil {
		return "", "", fmt.Errorf("an account login is already in progress")
	}
	lastDone, lastErr = false, nil
	verifier, challenge, err := pkce()
	if err != nil {
		return "", "", err
	}
	var addr, path, authURL, clientID, scopes, redirect, state string
	switch provider {
	case "anthropic":
		addr, path = "127.0.0.1:"+anthropicPort, anthropicPath
		authURL, clientID, scopes, redirect = anthropicAuth, anthropicClientID, anthropicScopes, anthropicRedirect
		state = verifier
	case "openai-codex":
		addr, path = "127.0.0.1:"+codexPort, codexPath
		authURL, clientID, scopes, redirect = codexAuth, codexClientID, codexScopes, codexRedirect
		b := make([]byte, 16)
		if _, err = rand.Read(b); err != nil {
			return "", "", err
		}
		state = fmt.Sprintf("%x", b)
	case "github-copilot", "kimi-coding", "xai":
		dc, err := beginDevice(provider)
		if err != nil {
			return "", "", err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		p := &pending{provider: provider, returnTo: returnTo, cancel: cancel, done: make(chan result, 1)}
		cur = p
		go func() { p.finish(dc.poll(ctx)) }()
		return dc.url, dc.code, nil
	default:
		return "", "", fmt.Errorf("account login for this provider is not wired yet")
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", "", fmt.Errorf("callback port busy (%s): %w", addr, err)
	}
	if returnTo != "" {
		if u, err := url.Parse(returnTo); err != nil || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") {
			returnTo = ""
		}
	}
	p := &pending{provider: provider, verifier: verifier, state: state, returnTo: returnTo, ln: ln, done: make(chan result, 1)}
	cur = p
	go serve(p, path, redirect, clientID)

	q := url.Values{}
	if provider == "anthropic" {
		q.Set("code", "true")
	}
	q.Set("client_id", clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirect)
	q.Set("scope", scopes)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	return authURL + "?" + q.Encode(), "", nil
}

func serve(p *pending, path, redirect, clientID string) {
	defer func() {
		_ = p.ln.Close()
		mu.Lock()
		if cur == p {
			cur = nil
		}
		mu.Unlock()
	}()
	srv := &http.Server{ReadHeaderTimeout: 10 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("error") != "" {
			htmlFail(w, r.URL.Query().Get("error"))
			p.finish(fmt.Errorf("oauth: %s", r.URL.Query().Get("error")))
			return
		}
		code := r.URL.Query().Get("code")
		st := r.URL.Query().Get("state")
		if code == "" || st != p.state {
			htmlFail(w, "missing code or state mismatch")
			p.finish(fmt.Errorf("oauth callback invalid"))
			return
		}
		if err := exchange(p.provider, code, st, p.verifier, redirect, clientID); err != nil {
			htmlFail(w, "token exchange failed")
			p.finish(err)
			return
		}
		htmlOK(w)
		p.finish(nil)
	})}
	_ = srv.Serve(p.ln)
}

func (p *pending) finish(err error) {
	mu.Lock()
	lastDone, lastErr = true, err
	mu.Unlock()
	select {
	case p.done <- result{err: err}:
	default:
	}
	go func() {
		time.Sleep(400 * time.Millisecond)
		if p.ln != nil {
			_ = p.ln.Close()
		}
	}()
}

// Status is for GUI polling. Tokens are never included.
func Status() (pending, done bool, err string) {
	mu.Lock()
	defer mu.Unlock()
	if cur != nil && !lastDone {
		return true, false, ""
	}
	if lastDone {
		if lastErr != nil {
			return false, true, lastErr.Error()
		}
		return false, true, ""
	}
	return false, false, ""
}

func Cancel() {
	mu.Lock()
	p := cur
	mu.Unlock()
	if p != nil {
		if p.cancel != nil {
			p.cancel()
		}
		p.finish(fmt.Errorf("cancelled"))
		if p.ln != nil {
			_ = p.ln.Close()
		}
	}
}

func exchange(provider, code, state, verifier, redirect, clientID string) error {
	var access, refresh string
	var expires int64
	var extra map[string]any

	switch provider {
	case "anthropic":
		body, _ := json.Marshal(map[string]string{
			"grant_type":    "authorization_code",
			"client_id":     clientID,
			"code":          code,
			"state":         state,
			"redirect_uri":  redirect,
			"code_verifier": verifier,
		})
		raw, err := postJSON(anthropicToken, body)
		if err != nil {
			return err
		}
		var tok struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		}
		if err := json.Unmarshal(raw, &tok); err != nil {
			return err
		}
		access, refresh = tok.AccessToken, tok.RefreshToken
		expires = time.Now().Add(time.Duration(tok.ExpiresIn)*time.Second - 5*time.Minute).UnixMilli()
	case "openai-codex":
		form := url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {clientID},
			"code":          {code},
			"code_verifier": {verifier},
			"redirect_uri":  {redirect},
		}
		raw, err := postForm(codexToken, form)
		if err != nil {
			return err
		}
		var tok struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		}
		if err := json.Unmarshal(raw, &tok); err != nil {
			return err
		}
		access, refresh = tok.AccessToken, tok.RefreshToken
		expires = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second).UnixMilli()
		aid := accountID(tok.AccessToken)
		if aid == "" {
			return fmt.Errorf("codex token missing account id")
		}
		extra = map[string]any{"accountId": aid}
	default:
		return fmt.Errorf("unknown provider")
	}
	cred := map[string]any{
		"type":    "oauth",
		"access":  access,
		"refresh": refresh,
		"expires": expires,
	}
	for k, v := range extra {
		cred[k] = v
	}
	return catalog.PutOAuth(provider, cred)
}

func postJSON(u string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("token http %d", res.StatusCode)
	}
	return raw, nil
}

func postForm(u string, form url.Values) ([]byte, error) {
	res, err := http.PostForm(u, form)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("token http %d", res.StatusCode)
	}
	return raw, nil
}

func accountID(access string) string {
	parts := strings.Split(access, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		b, err2 := base64.StdEncoding.DecodeString(parts[1])
		if err2 != nil {
			return ""
		}
		payload = b
	}
	var m map[string]any
	if json.Unmarshal(payload, &m) != nil {
		return ""
	}
	auth, _ := m[codexAccountNS].(map[string]any)
	id, _ := auth["chatgpt_account_id"].(string)
	return id
}

func htmlOK(w http.ResponseWriter) {
	back := ""
	mu.Lock()
	if cur != nil {
		back = cur.returnTo
	}
	mu.Unlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, oauthPage(true, back))
}

func htmlFail(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = io.WriteString(w, oauthPage(false, ""))
}

func oauthPage(ok bool, back string) string {
	heading := "Authentication complete"
	msg := "Returning to PiCode…"
	if !ok {
		heading = "Authentication did not complete"
		msg = "You can close this tab."
	}
	script := ""
	if ok {
		script = `<script>(function(){var n=3,el=document.getElementById("n");function tick(){if(el)el.textContent=n;if(n<=0){try{if(window.opener)window.opener.focus()}catch(e){}window.close();` +
			`setTimeout(function(){` + backJS(back) + `},200);return}n--;setTimeout(tick,1000)}setTimeout(tick,400)})()</script>`
	}
	return `<!doctype html><html lang="en"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/><title>PiCode</title>
<style>:root{--text:#fafafa;--dim:#a1a1aa;--bg:#09090b}*{box-sizing:border-box}html{color-scheme:dark}body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;background:var(--bg);color:var(--text);font-family:ui-sans-serif,system-ui,sans-serif;text-align:center}main{max-width:480px}.logo{width:72px;height:72px;margin:0 auto 24px}h1{margin:0 0 10px;font-size:28px;font-weight:650}p{margin:0;color:var(--dim);font-size:15px;line-height:1.6}</style></head>
<body><main>
<svg class="logo" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 800" aria-hidden="true"><path fill="#fff" fill-rule="evenodd" d="M165.29 165.29H517.36V400H400V517.36H282.65V634.72H165.29ZM282.65 282.65V400H400V282.65Z"/><path fill="#fff" d="M517.36 400H634.72V634.72H517.36Z"/></svg>
<h1>` + heading + `</h1><p>` + msg + ` <span id="n"></span></p>
</main>` + script + `</body></html>`
}

func backJS(back string) string {
	if back == "" {
		return ""
	}
	return "location.replace(" + strconvQuote(back) + ")"
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
