// Package presence tracks browsers connected to this PiCode instance
// (host machine vs phones/other devices on LAN or tailnet).
package presence

import (
	"context"
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
	Kind      string `json:"kind,omitempty"`
	Host      bool   `json:"host"`
	Online    bool   `json:"online"`
	LastSeen  string `json:"lastSeen"`
	FirstSeen string `json:"firstSeen"`
	Session   string `json:"session,omitempty"` // the paired session behind this ping (ADR-0049)
}

type rec struct {
	id, name, ip, kind string
	session            string
	host               bool
	first, last        time.Time
	online             bool // last announced state, so Expire fires once per transition
}

// Registry is an in-memory set of live clients.
type Registry struct {
	mu    sync.Mutex
	items map[string]*rec
	local map[string]bool

	// OnChange fires (outside the lock) when a device first appears, comes
	// back from stale, or goes stale (ADR-0048: ephemeral device.online /
	// device.offline notices; Device.Online says which).
	OnChange func(Device)
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
// kind "extension" is the Chrome side panel (ADR-0043 Track B).
func (r *Registry) Ping(id, ua, remote string, claimedHost bool, kind string) Device {
	return r.PingSession(id, ua, remote, claimedHost, kind, "")
}

// PingSession is Ping with the authenticated session behind the call,
// so the Devices page can join liveness onto identity.
func (r *Registry) PingSession(id, ua, remote string, claimedHost bool, kind, session string) Device {
	id = strings.TrimSpace(id)
	if id == "" {
		return Device{}
	}
	ip := stripPort(remote)
	now := time.Now().UTC()
	r.mu.Lock()
	it, ok := r.items[id]
	wasOnline := ok && r.view(it, now).Online
	if !ok {
		it = &rec{id: id, first: now}
		r.items[id] = it
	}
	it.kind = strings.TrimSpace(kind)
	if it.kind == "extension" {
		it.name = "Chrome extension"
	} else {
		it.name = Label(ua)
	}
	it.ip = ip
	it.host = claimedHost || r.local[ip]
	if session != "" {
		it.session = session
	}
	it.last = now
	d := r.view(it, now)
	it.online = true
	var changed func(Device)
	if !wasOnline {
		changed = r.OnChange
	}
	r.mu.Unlock()
	// Keep the callback outside the registry lock, but finish it before Ping
	// returns. The old detached goroutine could race a following Expire and
	// publish online after offline for a device that was already stale.
	if changed != nil {
		changed(d)
	}
	return d
}

// Expire announces every device that went stale since the last call —
// silence is the only signal a device has gone, so a ticker asks.
func (r *Registry) Expire() []Device {
	now := time.Now().UTC()
	r.mu.Lock()
	var gone []Device
	for _, it := range r.items {
		if it.online && now.Sub(it.last) >= staleAfter {
			it.online = false
			gone = append(gone, r.view(it, now))
		}
	}
	fn := r.OnChange
	r.mu.Unlock()
	if fn != nil {
		for _, d := range gone {
			fn(d)
		}
	}
	return gone
}

// Watch runs Expire every interval until ctx ends.
func (r *Registry) Watch(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.Expire()
		}
	}
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

// AnyHostOnline reports whether a browser on the host machine pinged within
// the staleness window — "the user is at the desk" (ADR-0047 keeps the
// phone quiet then, the way Claude Code's Remote Control does).
func (r *Registry) AnyHostOnline() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	for _, it := range r.items {
		if it.host && it.kind != "extension" && now.Sub(it.last) < staleAfter {
			return true
		}
	}
	return false
}

// SessionLive reports whether any device pinged under that session id
// within the staleness window — a browser holding this session's cookie
// is here right now. The auth gate asks before rotating a loopback
// session's secret (ADR-0049 amendment), so an active browser never has
// its cookie rotated out from under it.
func (r *Registry) SessionLive(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	for _, it := range r.items {
		if it.session == id && now.Sub(it.last) < staleAfter {
			return true
		}
	}
	return false
}

func (r *Registry) view(it *rec, now time.Time) Device {
	return Device{
		Session:   it.session,
		ID:        it.id,
		Name:      it.name,
		IP:        it.ip,
		Kind:      it.kind,
		Host:      it.host,
		Online:    now.Sub(it.last) < staleAfter,
		LastSeen:  it.last.Format(time.RFC3339),
		FirstSeen: it.first.Format(time.RFC3339),
	}
}

// Label turns a User-Agent into a short device name. Headless browsers
// (the QA fleets automation runs on this machine) say so instead of
// borrowing the OS name — a row of "Linux" machines that are really
// test profiles is how the Devices list filled with duplicates.
func Label(ua string) string {
	u := strings.ToLower(ua)
	switch {
	case strings.Contains(u, "headlesschrome") || strings.Contains(u, "headless"):
		return "Headless browser"
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
