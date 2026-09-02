// Package config resolves PiCode's runtime configuration.
//
// Port precedence: DB setting (editable in Settings UI) > PICODE_PORT env
// (headless/ops) > default range. See ADR-0007.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// PortConfig is a port or a range of ports to try, in order.
type PortConfig struct {
	Min int
	Max int
}

// String renders "8445" or "8445-8460".
func (p PortConfig) String() string {
	if p.Min == p.Max {
		return strconv.Itoa(p.Min)
	}
	return fmt.Sprintf("%d-%d", p.Min, p.Max)
}

// Validate checks basic sanity.
func (p PortConfig) Validate() error {
	if p.Min < 1 || p.Max < 1 || p.Min > 65535 || p.Max > 65535 {
		return fmt.Errorf("config: ports must be 1-65535")
	}
	if p.Max < p.Min {
		return fmt.Errorf("config: invalid range %s (min > max)", p)
	}
	if p.Max-p.Min > 100 {
		return fmt.Errorf("config: port range too wide (max 100 ports)")
	}
	return nil
}

// ParsePort parses "8445" or "8445-8460".
func ParsePort(s string) (PortConfig, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return PortConfig{}, fmt.Errorf("config: empty port")
	}
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		min, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		max, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil {
			return PortConfig{}, fmt.Errorf("config: bad port range %q", s)
		}
		return PortConfig{Min: min, Max: max}, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return PortConfig{}, fmt.Errorf("config: bad port %q", s)
	}
	return PortConfig{Min: n, Max: n}, nil
}

// DefaultPortRange is the default when nothing else is configured.
const DefaultPortRange = "8445-8455"

// Config is the resolved runtime configuration.
type Config struct {
	Host      string // bind host
	Port      PortConfig
	DataDir   string // db, certs, server.json
	Insecure  bool   // disable TLS (dev / behind proxy)
	PublicURL string // the origin other machines use; "" when none (ADR-0050)
}

// Settings keys edited by the Settings UI (ADR-0007, ADR-0050). Host,
// port and public URL share one precedence: DB > env > default. The env
// is how a provisioned member daemon (ADR-0051) is configured before it
// has a UI.
const (
	PortSettingKey      = "server.port"
	HostSettingKey      = "server.host"
	PublicURLSettingKey = "server.public_url"
)

// DefaultHost binds every interface (ADR-0007).
const DefaultHost = "0.0.0.0"

// ValidateHost accepts an unspecified address or an IP literal. Names are
// refused: a bind is an interface, and a name that resolves elsewhere
// tomorrow would silently take the server off the machine.
func ValidateHost(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("config: empty host")
	}
	if net.ParseIP(s) == nil {
		return "", fmt.Errorf("config: host must be an IP address (0.0.0.0 for every interface)")
	}
	return s, nil
}

// ValidatePublicURL normalises "https://host[:port]". Empty clears it.
// Plain http is only allowed when TLS is off (behind a proxy, dev).
func ValidatePublicURL(s string, insecure bool) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return "", fmt.Errorf("config: public URL must look like https://host:port")
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !insecure {
			return "", fmt.Errorf("config: public URL must be https (this server serves TLS)")
		}
	default:
		return "", fmt.Errorf("config: public URL must be https://")
	}
	if (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", fmt.Errorf("config: public URL is an origin — no path, query or credentials")
	}
	return u.Scheme + "://" + strings.ToLower(u.Host), nil
}

// Resolve builds the effective config. dbGet returns ("", false, nil) when
// the setting is absent.
func Resolve(dbGet func(key string) (string, bool, error)) (Config, error) {
	home := ""
	if u, err := user.Current(); err == nil && u.HomeDir != "" {
		home = u.HomeDir
	}
	cfg := Config{
		Host:     getenv("PICODE_HOST", DefaultHost),
		DataDir:  getenv("PICODE_DATA", filepath.Join(home, ".picode")),
		Insecure: os.Getenv("PICODE_INSECURE") == "1",
	}
	if v := os.Getenv("PICODE_PUBLIC_URL"); v != "" {
		if p, err := ValidatePublicURL(v, cfg.Insecure); err == nil {
			cfg.PublicURL = p
		}
	}
	if dbGet != nil {
		if v, ok, err := dbGet(HostSettingKey); err == nil && ok {
			if h, err := ValidateHost(v); err == nil {
				cfg.Host = h
			}
		}
		if v, ok, err := dbGet(PublicURLSettingKey); err == nil && ok && strings.TrimSpace(v) != "" {
			if p, err := ValidatePublicURL(v, cfg.Insecure); err == nil {
				cfg.PublicURL = p
			}
		}
	}

	// Precedence: DB (UI) > env > default.
	raw := DefaultPortRange
	if v := os.Getenv("PICODE_PORT"); v != "" {
		raw = v
	}
	if dbGet != nil {
		if v, ok, err := dbGet(PortSettingKey); err == nil && ok && strings.TrimSpace(v) != "" {
			raw = v
		}
	}
	p, err := ParsePort(raw)
	if err != nil {
		return cfg, err
	}
	if err := p.Validate(); err != nil {
		return cfg, err
	}
	cfg.Port = p
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
