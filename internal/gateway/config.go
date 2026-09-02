// Package gateway is the front door of a shared PiCode box (ADR-0051):
// one listener on the tailnet, identifying the person by Tailscale and
// proxying to that person's own daemon, which runs as their own Linux
// user on loopback. Nothing here holds state about people beyond the
// login → user map; every daemon keeps its own sessions, files, agents.
package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// DefaultConfigPath is where `picode gateway` looks.
const DefaultConfigPath = "/etc/picode/gateway.json"

// Config is /etc/picode/gateway.json.
type Config struct {
	Listen   string            `json:"listen"`   // ":443"
	Hostname string            `json:"hostname"` // box.tailxxxx.ts.net — the name clients use and the cert covers
	DataDir  string            `json:"dataDir"`  // where the Tailscale leaf lives (default /etc/picode)
	Users    map[string]string `json:"users"`    // tailscale login → linux user
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

// Logins lists the map, sorted, for `picode users list`.
func (c Config) Logins() []string {
	out := make([]string, 0, len(c.Users))
	for l := range c.Users {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}
