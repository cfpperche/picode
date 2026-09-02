package gateway

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Backend is one person's daemon, as it is right now: read from their
// own server.json (the port can rebind) and token (it can rotate) on
// every request. Root reads two files in a home directory; that is the
// whole privilege the gateway uses.
type Backend struct {
	User  string
	Home  string
	URL   *url.URL
	Token string
}

// homeOf is the account boundary; tests replace it.
var homeOf = func(linux string) (string, error) {
	u, err := user.Lookup(linux)
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

// ErrNotRunning says the daemon has not published an address.
type ErrNotRunning struct{ User string }

func (e ErrNotRunning) Error() string {
	return fmt.Sprintf("PiCode is not running for %s — `picode provision --user %s --shared`", e.User, e.User)
}

// Resolve finds the daemon for a Linux user.
func Resolve(linux string) (Backend, error) {
	home, err := homeOf(linux)
	if err != nil {
		return Backend{}, fmt.Errorf("no Linux user %q", linux)
	}
	return ResolveHome(linux, home)
}

// ResolveHome is Resolve for a known home directory (the scratch seam:
// two daemons of one Linux user standing in for two members).
func ResolveHome(linux, home string) (Backend, error) {
	data := filepath.Join(home, ".picode")
	b, err := os.ReadFile(filepath.Join(data, "server.json"))
	if err != nil {
		return Backend{}, ErrNotRunning{User: linux}
	}
	var s struct {
		URL string `json:"url"`
	}
	if json.Unmarshal(b, &s) != nil || s.URL == "" {
		return Backend{}, ErrNotRunning{User: linux}
	}
	u, err := url.Parse(strings.TrimRight(s.URL, "/"))
	if err != nil || u.Host == "" {
		return Backend{}, ErrNotRunning{User: linux}
	}
	tok, _ := os.ReadFile(filepath.Join(data, "token"))
	return Backend{User: linux, Home: home, URL: u, Token: strings.TrimSpace(string(tok))}, nil
}
