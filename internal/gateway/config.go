// Package gateway is the front door of a shared PiCode box (ADR-0051):
// one listener on the tailnet, identifying the person by Tailscale and
// proxying to that person's own daemon, which runs as their own Linux
// user on loopback. Nothing here holds state about people beyond the
// login → user map; every daemon keeps its own sessions, files, agents.
package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// DefaultConfigPath is where `picode gateway` looks.
const DefaultConfigPath = "/etc/picode/gateway.json"

// Config is /etc/picode/gateway.json. No secrets here (0644): client
// secrets and the cookie key live in gateway.secret.json (0600).
type Config struct {
	Listen   string            `json:"listen"`   // ":443" — the tailnet front door (Tailscale leaf, whois)
	Hostname string            `json:"hostname"` // box.tailxxxx.ts.net — the name clients use and the cert covers
	DataDir  string            `json:"dataDir"`  // where the Tailscale leaf lives (default /etc/picode)
	Users    map[string]string `json:"users"`    // login → linux user (tailscale login, email, or name@github)

	// Public access (ADR-0052): a second, plain-HTTP front door behind a
	// TLS proxy, and the logins that door accepts.
	PlainListen    string                    `json:"plainListen,omitempty"`    // "127.0.0.1:8480" — behind Caddy / Cloudflare Tunnel
	PublicURL      string                    `json:"publicUrl,omitempty"`      // https://picode.example.com — the origin the proxy serves
	TrustedProxies []string                  `json:"trustedProxies,omitempty"` // CIDRs whose X-Forwarded-For is believed
	OIDC           map[string]ProviderConfig `json:"oidc,omitempty"`           // "google", "github"
}

// ProviderConfig is the public half of a login provider. Endpoints are
// filled from the provider's well-known values when empty; setting them
// is the test seam (a local fake provider) and is refused on a listener
// that is not loopback.
type ProviderConfig struct {
	Issuer   string `json:"issuer,omitempty"`   // OIDC: discovery base (google)
	AuthURL  string `json:"authUrl,omitempty"`  // OAuth: authorize endpoint (github)
	TokenURL string `json:"tokenUrl,omitempty"` // OAuth: token endpoint (github)
	UserURL  string `json:"userUrl,omitempty"`  // OAuth: identity endpoint (github)
}

// Overridden reports whether any endpoint was set by hand.
func (p ProviderConfig) Overridden() bool {
	return p.Issuer != "" || p.AuthURL != "" || p.TokenURL != "" || p.UserURL != ""
}

// SecretFile sits beside gateway.json, 0600.
const SecretFile = "gateway.secret.json"

// Secrets is gateway.secret.json.
type Secrets struct {
	CookieKey string                    `json:"cookieKey"` // hex, 32 bytes — signs the gateway session cookie
	Providers map[string]ProviderSecret `json:"providers"`
}

// ProviderSecret is a provider's client credentials.
type ProviderSecret struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

// SecretPath is the secret file beside a config path.
func SecretPath(cfgPath string) string { return filepath.Join(filepath.Dir(cfgPath), SecretFile) }

// LoadSecrets reads the secret file; a missing file yields empty secrets
// (the cookie key is minted on first Save).
func LoadSecrets(path string) (Secrets, error) {
	var s Secrets
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Secrets{Providers: map[string]ProviderSecret{}}, nil
		}
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("%s: %w", path, err)
	}
	if s.Providers == nil {
		s.Providers = map[string]ProviderSecret{}
	}
	return s, nil
}

// SaveSecrets writes the file 0600, minting the cookie key if absent.
func SaveSecrets(path string, s Secrets) error {
	if s.CookieKey == "" {
		k, err := randomHex(32)
		if err != nil {
			return err
		}
		s.CookieKey = k
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// TrustedProxyNets parses TrustedProxies; a bad CIDR is a config error.
func (c Config) TrustedProxyNets() ([]*net.IPNet, error) {
	var out []*net.IPNet
	for _, s := range c.TrustedProxies {
		_, n, err := net.ParseCIDR(strings.TrimSpace(s))
		if err != nil {
			return nil, fmt.Errorf("gateway: trustedProxies %q: %w", s, err)
		}
		out = append(out, n)
	}
	return out, nil
}

// Default is a fresh config for this box; hostname is filled by the caller.
func Default(hostname string) Config {
	return Config{Listen: ":443", Hostname: hostname, DataDir: filepath.Dir(DefaultConfigPath), Users: map[string]string{}}
}

// Load reads path. A missing file is an error: the gateway must not run
// with an empty map by accident.
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if c.Users == nil {
		c.Users = map[string]string{}
	}
	if c.Listen == "" {
		c.Listen = ":443"
	}
	if c.DataDir == "" {
		c.DataDir = filepath.Dir(path)
	}
	return c, c.Validate()
}

// Save writes path atomically (0644: no secrets inside).
func Save(path string, c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

var linuxUser = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)

// Validate checks shapes, not existence (AddUser checks the account).
func (c Config) Validate() error {
	if strings.TrimSpace(c.Hostname) == "" {
		return fmt.Errorf("gateway: hostname is required (the tailnet name of this box)")
	}
	if c.PublicURL != "" {
		u, err := url.Parse(c.PublicURL)
		if err != nil || u.Scheme != "https" || u.Host == "" || (u.Path != "" && u.Path != "/") {
			return fmt.Errorf("gateway: publicUrl must be an https origin, got %q", c.PublicURL)
		}
	}
	if _, err := c.TrustedProxyNets(); err != nil {
		return err
	}
	for name := range c.OIDC {
		if name != "google" && name != "github" {
			return fmt.Errorf("gateway: unknown login provider %q (google, github)", name)
		}
	}
	for login, u := range c.Users {
		if strings.TrimSpace(login) == "" || !linuxUser.MatchString(u) {
			return fmt.Errorf("gateway: bad mapping %q → %q", login, u)
		}
	}
	return nil
}

// UserFor resolves a Tailscale login (case-insensitive) to a Linux user.
func (c Config) UserFor(login string) (string, bool) {
	login = strings.ToLower(strings.TrimSpace(login))
	for l, u := range c.Users {
		if strings.ToLower(l) == login {
			return u, true
		}
	}
	return "", false
}

// lookupUser is the account boundary; tests replace it.
var lookupUser = func(name string) (*user.User, error) { return user.Lookup(name) }

// AddUser maps login → linuxUser; the account must exist.
func (c *Config) AddUser(login, linux string) error {
	login = strings.TrimSpace(login)
	if login == "" {
		return fmt.Errorf("gateway: empty login")
	}
	if !linuxUser.MatchString(linux) {
		return fmt.Errorf("gateway: %q is not a Linux user name", linux)
	}
	if _, err := lookupUser(linux); err != nil {
		return fmt.Errorf("gateway: no Linux user %q — create it with `picode provision --user %s --shared`", linux, linux)
	}
	if c.Users == nil {
		c.Users = map[string]string{}
	}
	c.Users[login] = linux
	return nil
}

// RemoveUser drops a login; unknown is not an error.
func (c *Config) RemoveUser(login string) {
	for l := range c.Users {
		if strings.EqualFold(l, strings.TrimSpace(login)) {
			delete(c.Users, l)
		}
	}
}

// IsMember reports whether a Linux user is the target of some login.
func (c Config) IsMember(linux string) bool {
	for _, u := range c.Users {
		if u == linux {
			return true
		}
	}
	return false
}

// Logins lists the map, sorted, for `picode users list`.
func (c Config) Logins() []string {
	out := make([]string, 0, len(c.Users))
	for l := range c.Users {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}
