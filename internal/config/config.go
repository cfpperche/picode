// Package config resolves PiCode's runtime configuration.
//
// Port precedence: DB setting (editable in Settings UI) > PICODE_PORT env
// (headless/ops) > default range. See ADR-0007.
package config

import (
	"fmt"
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
	Host     string // bind host
	Port     PortConfig
	DataDir  string // db, certs, server.json
	Insecure bool   // disable TLS (dev / behind proxy)
}

// PortSettingKey is the DB settings key edited by the Settings UI.
const PortSettingKey = "server.port"

// Resolve builds the effective config. dbGet returns ("", false, nil) when
// the setting is absent.
func Resolve(dbGet func(key string) (string, bool, error)) (Config, error) {
	home := ""
	if u, err := user.Current(); err == nil && u.HomeDir != "" {
		home = u.HomeDir
	}
	cfg := Config{
		Host:     getenv("PICODE_HOST", "0.0.0.0"),
		DataDir:  getenv("PICODE_DATA", filepath.Join(home, ".picode")),
		Insecure: os.Getenv("PICODE_INSECURE") == "1",
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
