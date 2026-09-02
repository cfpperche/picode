package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Identity answers "who is at this address" from Tailscale itself:
// `tailscale whois` asks tailscaled about the peer that owns the
// connecting IP. Nothing the client sends is consulted — a header is a
// claim, the peer address is a fact of the tunnel.
type Identity struct {
	mu    sync.Mutex
	cache map[string]cached
	TTL   time.Duration
}

type cached struct {
	login, node string
	until       time.Time
}

// whoisCmd is the process boundary; tests replace it.
var whoisCmd = func(ctx context.Context, ip string) ([]byte, error) {
	return exec.CommandContext(ctx, "tailscale", "whois", "--json", ip).Output()
}

// NewIdentity caches answers for ttl (60 s is plenty: a peer does not
// change owner mid-session, and the CLI costs ~50 ms).
func NewIdentity(ttl time.Duration) *Identity {
	return &Identity{cache: map[string]cached{}, TTL: ttl}
}

// IsTailnet reports a Tailscale address (100.64.0.0/10 or fd7a:115c:a1e0::/48).
func IsTailnet(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		return v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127
	}
	return len(ip) == 16 && ip[0] == 0xfd && ip[1] == 0x7a && ip[2] == 0x11 && ip[3] == 0x5c && ip[4] == 0xa1 && ip[5] == 0xe0
}

// Whois resolves the login and node name behind ip.
func (id *Identity) Whois(ctx context.Context, ip string) (login, node string, err error) {
	if !IsTailnet(net.ParseIP(ip)) {
		return "", "", fmt.Errorf("%s is not a tailnet address", ip)
	}
	id.mu.Lock()
	if c, ok := id.cache[ip]; ok && time.Now().Before(c.until) {
		id.mu.Unlock()
		return c.login, c.node, nil
	}
	id.mu.Unlock()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	raw, err := whoisCmd(cctx, ip)
	if err != nil {
		return "", "", fmt.Errorf("tailscale whois %s: %w", ip, err)
	}
	var w struct {
		UserProfile struct {
			LoginName string `json:"LoginName"`
		} `json:"UserProfile"`
		Node struct {
			Name string `json:"Name"`
		} `json:"Node"`
	}
	if err := json.Unmarshal(raw, &w); err != nil || w.UserProfile.LoginName == "" {
		return "", "", fmt.Errorf("tailscale whois %s: no login in the answer", ip)
	}
	login = strings.ToLower(w.UserProfile.LoginName)
	node = strings.TrimSuffix(w.Node.Name, ".")
	id.mu.Lock()
	id.cache[ip] = cached{login: login, node: node, until: time.Now().Add(id.TTL)}
	id.mu.Unlock()
	return login, node, nil
}
