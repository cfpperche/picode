package gateway

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Login providers (ADR-0052), standard library only.
//
//	google  OpenID Connect: discovery, code + PKCE + state + nonce, ID
//	        token verified against the JWKS (RS256), login = verified email.
//	github  OAuth 2 (GitHub has no OIDC for users): code + state, then
//	        GET /user; login = "<login>@github" — the same spelling
//	        Tailscale uses, so one `users add` line serves both doors.

// HTTPClient talks to providers; tests replace it.
var HTTPClient = &http.Client{Timeout: 10 * time.Second}

type provider struct {
	name     string
	cfg      ProviderConfig
	secret   ProviderSecret
	authURL  string
	tokenURL string
	userURL  string // github
	jwksURL  string // google
	issuer   string // google: expected iss

	mu   sync.Mutex
	keys map[string]*rsa.PublicKey
	got  time.Time
}

func newProvider(name string, cfg ProviderConfig, sec ProviderSecret) (*provider, error) {
	if sec.ClientID == "" || sec.ClientSecret == "" {
		return nil, fmt.Errorf("gateway: %s has no client id/secret (picode gateway oidc set %s <id> <secret>)", name, name)
	}
	p := &provider{name: name, cfg: cfg, secret: sec, keys: map[string]*rsa.PublicKey{}}
	switch name {
	case "google":
		p.issuer = "https://accounts.google.com"
		if cfg.Issuer != "" {
			p.issuer = strings.TrimRight(cfg.Issuer, "/")
		}
	case "github":
		p.authURL, p.tokenURL, p.userURL = "https://github.com/login/oauth/authorize", "https://github.com/login/oauth/access_token", "https://api.github.com/user"
		if cfg.AuthURL != "" {
			p.authURL = cfg.AuthURL
		}
		if cfg.TokenURL != "" {
			p.tokenURL = cfg.TokenURL
		}
		if cfg.UserURL != "" {
			p.userURL = cfg.UserURL
		}
	default:
		return nil, fmt.Errorf("gateway: unknown provider %q", name)
	}
	return p, nil
}

// discover fills Google's endpoints from the issuer's discovery document
// (once per process; the values do not change).
func (p *provider) discover(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.authURL != "" {
		return nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, p.issuer+"/.well-known/openid-configuration", nil)
	res, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s discovery: %w", p.name, err)
	}
	defer res.Body.Close()
	var d struct {
		Issuer string `json:"issuer"`
		Auth   string `json:"authorization_endpoint"`
		Token  string `json:"token_endpoint"`
		JWKS   string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&d); err != nil || d.Auth == "" || d.Token == "" || d.JWKS == "" {
		return fmt.Errorf("%s discovery: unusable document", p.name)
	}
	p.authURL, p.tokenURL, p.jwksURL = d.Auth, d.Token, d.JWKS
	if d.Issuer != "" {
		p.issuer = d.Issuer
	}
	return nil
}

// pending is one login in flight, keyed by state; single use, 10 minutes.
type pending struct {
	provider, verifier, nonce, next string
	until                           time.Time
}

func pkce() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// authorizeURL builds the redirect to the provider.
func (p *provider) authorizeURL(ctx context.Context, redirect, state, nonce, challenge string) (string, error) {
	if err := p.discover(ctx); err != nil {
		return "", err
	}
	q := url.Values{
		"client_id": {p.secret.ClientID}, "redirect_uri": {redirect}, "response_type": {"code"}, "state": {state},
	}
	switch p.name {
	case "google":
		q.Set("scope", "openid email")
		q.Set("nonce", nonce)
		q.Set("code_challenge", challenge)
		q.Set("code_challenge_method", "S256")
		q.Set("prompt", "select_account")
	case "github":
		q.Set("scope", "read:user user:email")
	}
	sep := "?"
	if strings.Contains(p.authURL, "?") {
		sep = "&"
	}
	return p.authURL + sep + q.Encode(), nil
}

// exchange turns the callback's code into a login.
func (p *provider) exchange(ctx context.Context, code, redirect, verifier, nonce string) (string, error) {
	if err := p.discover(ctx); err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirect},
		"client_id": {p.secret.ClientID}, "client_secret": {p.secret.ClientSecret}}
	if p.name == "google" {
		form.Set("code_verifier", verifier)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s token: %w", p.name, err)
	}
	defer res.Body.Close()
	var tok struct {
		Access  string `json:"access_token"`
		IDToken string `json:"id_token"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&tok); err != nil || res.StatusCode != http.StatusOK || tok.Error != "" {
		return "", fmt.Errorf("%s token: %s %s", p.name, res.Status, tok.Error)
	}
	switch p.name {
	case "google":
		return p.loginFromIDToken(ctx, tok.IDToken, nonce)
	default:
		return p.loginFromGitHub(ctx, tok.Access)
	}
}

func (p *provider) loginFromGitHub(ctx context.Context, access string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, p.userURL, nil)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("github user: %w", err)
	}
	defer res.Body.Close()
	var u struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&u); err != nil || u.Login == "" {
		return "", fmt.Errorf("github user: no login (%s)", res.Status)
	}
	return strings.ToLower(u.Login) + "@github", nil
}

// loginFromIDToken verifies a Google ID token: RS256 signature against
// the JWKS, issuer, audience, expiry, nonce; login = verified email.
func (p *provider) loginFromIDToken(ctx context.Context, idToken, nonce string) (string, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return "", errors.New("google: malformed id_token")
	}
	var hdr struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(hb, &hdr) != nil || hdr.Alg != "RS256" {
		return "", errors.New("google: id_token is not RS256")
	}
	key, err := p.key(ctx, hdr.Kid)
	if err != nil {
		return "", err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", errors.New("google: bad signature encoding")
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig); err != nil {
		return "", errors.New("google: id_token signature does not verify")
	}
	var claims struct {
		Iss      string          `json:"iss"`
		Aud      json.RawMessage `json:"aud"`
		Exp      int64           `json:"exp"`
		Nonce    string          `json:"nonce"`
		Email    string          `json:"email"`
		Verified bool            `json:"email_verified"`
	}
	cb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(cb, &claims) != nil {
		return "", errors.New("google: bad claims")
	}
	if claims.Iss != p.issuer && claims.Iss != strings.TrimPrefix(p.issuer, "https://") {
		return "", fmt.Errorf("google: issuer %q is not %q", claims.Iss, p.issuer)
	}
	if !audienceHas(claims.Aud, p.secret.ClientID) {
		return "", errors.New("google: id_token is for another client")
	}
	if time.Now().Unix() >= claims.Exp {
		return "", errors.New("google: id_token expired")
	}
	if nonce == "" || claims.Nonce != nonce {
		return "", errors.New("google: nonce mismatch")
	}
	if !claims.Verified || claims.Email == "" {
		return "", errors.New("google: no verified email")
	}
	return strings.ToLower(claims.Email), nil
}

func audienceHas(raw json.RawMessage, want string) bool {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return one == want
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		for _, a := range many {
			if a == want {
				return true
			}
		}
	}
	return false
}

// key fetches the JWKS (cached an hour; refetched on an unknown kid).
func (p *provider) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	p.mu.Lock()
	k, ok := p.keys[kid]
	fresh := time.Since(p.got) < time.Hour
	p.mu.Unlock()
	if ok && fresh {
		return k, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, nil)
	res, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google jwks: %w", err)
	}
	defer res.Body.Close()
	var set struct {
		Keys []struct {
			Kty, Kid, N, E string
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&set); err != nil {
		return nil, fmt.Errorf("google jwks: %w", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.keys = map[string]*rsa.PublicKey{}
	for _, j := range set.Keys {
		if j.Kty != "RSA" {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(j.N)
		e, err2 := base64.RawURLEncoding.DecodeString(j.E)
		if err1 != nil || err2 != nil {
			continue
		}
		p.keys[j.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	p.got = time.Now()
	if k, ok := p.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("google jwks: no key %q", kid)
}
