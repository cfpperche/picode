package gateway

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// The gateway's own session (ADR-0052): who logged in through a
// provider, signed so the gateway needs no session table. It is the
// first factor for people off the tailnet; the daemon's picode_session
// (a paired device) stays the second.
const (
	SessionCookie = "picode_gateway"
	SessionTTL    = 30 * 24 * time.Hour
)

type signer struct{ key []byte }

func newSigner(hexKey string) (signer, error) {
	k, err := hex.DecodeString(hexKey)
	if err != nil || len(k) < 32 {
		return signer{}, fmt.Errorf("gateway: cookie key must be 32 random bytes in hex (run `picode gateway oidc set`)")
	}
	return signer{key: k}, nil
}

// issue encodes login|exp and signs it: base64(payload).base64(mac).
func (s signer) issue(login string, exp time.Time) string {
	payload := login + "|" + strconv.FormatInt(exp.Unix(), 10)
	p := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return p + "." + s.mac(p)
}

// verify returns the login for a valid, unexpired cookie value.
func (s signer) verify(value string, now time.Time) (string, bool) {
	p, m, ok := strings.Cut(value, ".")
	if !ok || subtle.ConstantTimeCompare([]byte(m), []byte(s.mac(p))) != 1 {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(p)
	if err != nil {
		return "", false
	}
	login, expS, ok := strings.Cut(string(raw), "|")
	if !ok || login == "" {
		return "", false
	}
	exp, err := strconv.ParseInt(expS, 10, 64)
	if err != nil || now.Unix() >= exp {
		return "", false
	}
	return login, true
}

func (s signer) mac(p string) string {
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(p))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func setSessionCookie(w http.ResponseWriter, value string, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: value, Path: "/", HttpOnly: true, Secure: secure,
		SameSite: http.SameSiteLaxMode, MaxAge: int(SessionTTL / time.Second)})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
}
