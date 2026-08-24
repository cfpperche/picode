// Package presence tracks browsers connected to this PiCode instance
// (host machine vs phones/other devices on LAN or tailnet).
package presence

import (
	"net"
	"strings"
	"sync"
	"time"
)

const staleAfter = 45 * time.Second

// Device is one browser tab/app that has pinged recently.
type Device struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Host      bool   `json:"host"`
	Online    bool   `json:"online"`
	LastSeen  string `json:"lastSeen"`
	FirstSeen string `json:"firstSeen"`
}

type rec struct {
	id, name, ip string
	host         bool
	first, last  time.Time
}

// Registry is an in-memory set of live clients.
type Registry struct {
	mu    sync.Mutex
	items map[string]*rec
	local map[string]bool
}

// New builds a registry. localIPs are treated as the host machine.
func New(localIPs []string) *Registry {
	loc := map[string]bool{"127.0.0.1": true, "::1": true, "localhost": true}
	for _, ip := range localIPs {
		if ip != "" {
			loc[ip] = true
		}
	}
	return &Registry{items: map[string]*rec{}, local: loc}
}

// SetLocal replaces the host-IP set (call when interfaces change).
func (r *Registry) SetLocal(ips []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.local = map[string]bool{"127.0.0.1": true, "::1": true, "localhost": true}
	for _, ip := range ips {
		if ip != "" {
			r.local[ip] = true
		}
	}
}

// Ping records a heartbeat. id must be a client-generated opaque token.
func (r *Registry) Ping(id, ua, remote string, claimedHost bool) Device {
	id = strings.TrimSpace(id)
	if id == "" {
		return Device{}
	}
	ip := stripPort(remote)
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	it, ok := r.items[id]
	if !ok {
		it = &rec{id: id, first: now}
		r.items[id] = it
	}
	it.name = Label(ua)
	it.ip = ip
	it.host = claimedHost || r.local[ip]
	it.last = now
	return r.view(it, now)
}

// List returns devices, newest last-seen first. Stale ones stay until
// pruned, marked online=false.
func (r *Registry) List() []Device {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	out := make([]Device, 0, len(r.items))
	for id, it := range r.items {
		if now.Sub(it.last) > 10*time.Minute {
			delete(r.items, id)
			continue
		}
		out = append(out, r.view(it, now))
	}
	// host first, then online, then name — simple insertion order is fine
	return sortDevices(out)
}

func (r *Registry) view(it *rec, now time.Time) Device {
	return Device{
		ID:        it.id,
		Name:      it.name,
		IP:        it.ip,
		Host:      it.host,
		Online:    now.Sub(it.last) < staleAfter,
		LastSeen:  it.last.Format(time.RFC3339),
		FirstSeen: it.first.Format(time.RFC3339),
	}
}

// Label turns a User-Agent into a short device name.
func Label(ua string) string {
	u := strings.ToLower(ua)
	switch {
	case strings.Contains(u, "iphone"):
		return "iPhone"
	case strings.Contains(u, "ipad"):
		return "iPad"
	case strings.Contains(u, "android"):
		if strings.Contains(u, "mobile") {
			return "Android"
		}
		return "Android tablet"
	case strings.Contains(u, "macintosh"):
		return "Mac"
	case strings.Contains(u, "windows"):
		return "Windows"
	case strings.Contains(u, "linux"):
		return "Linux"
	default:
		if ua == "" {
			return "Unknown"
		}
		return "Browser"
	}
}

func stripPort(remote string) string {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return remote
	}
	if host == "::1" {
		return "::1"
	}
	return host
}

func sortDevices(in []Device) []Device {
	// host + online first
	out := make([]Device, 0, len(in))
	var host, on, off []Device
	for _, d := range in {
		switch {
		case d.Host:
			host = append(host, d)
		case d.Online:
			on = append(on, d)
		default:
			off = append(off, d)
		}
	}
	out = append(out, host...)
	out = append(out, on...)
	out = append(out, off...)
	return out
}
