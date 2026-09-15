package tmux

// Machine-wide socket discovery (plan tmux-resilience, ADR-0139 follow-up):
// after the dedicated-socket move, an operator's first question is "which
// tmux servers are running on this machine, and what lives on each?" The tmux
// app answers it from here.
//
// What can be discovered honestly: every socket file in the user's tmux
// directory (the default socket and every -L name) plus this Manager's own
// -S path. A -S socket can live in any directory, so sockets created by
// other software outside these places are not discoverable — there is no
// registry, and scanning the filesystem would be worse than the gap.
//
// Liveness is probed with a plain unix-socket dial, never by running tmux:
// connecting to a dead socket file must not start a transient server.

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MachineSocket is one tmux server socket on this machine, as seen from this
// instance.
type MachineSocket struct {
	Path           string // full socket path; empty SocketPath in the caller's Manager means the default
	Name           string // file name: "default", "tachyon", "tmux.sock", …
	Running        bool   // something is listening on it
	Sessions       int    // sessions on it (0 when not running)
	PicodeSessions int    // sessions whose name carries the picode- hint
	Ours           bool   // this Manager's own socket (new sessions land here)
}

// MachineSockets lists every tmux socket this instance can see: the files in
// the user's tmux directory plus this Manager's own socket path. Running is
// probed with a unix dial — a dead socket file is reported as such, never
// resurrected by a probe.
func (m *Manager) MachineSockets(ctx context.Context) []MachineSocket {
	type entry struct {
		path, name string
		ours       bool
	}
	seen := map[string]bool{}
	var entries []entry

	add := func(path string, ours bool) {
		path = filepath.Clean(path)
		if seen[path] {
			return
		}
		seen[path] = true
		entries = append(entries, entry{path: path, name: filepath.Base(path), ours: ours})
	}
	if m.socket != "" {
		add(m.socket, true)
	}
	if dir := DefaultSocketDir(); dir != "" {
		if names, err := os.ReadDir(dir); err == nil {
			for _, f := range names {
				add(filepath.Join(dir, f.Name()), false)
			}
		}
	}

	out := make([]MachineSocket, 0, len(entries))
	for _, e := range entries {
		ms := MachineSocket{Path: e.path, Name: e.name, Ours: e.ours}
		ms.Running = unixSocketAlive(e.path)
		if ms.Running {
			ms.Sessions, ms.PicodeSessions = countSessionsOn(ctx, e.path)
		}
		out = append(out, ms)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Ours != out[j].Ours {
			return out[i].Ours
		}
		if out[i].Running != out[j].Running {
			return out[i].Running
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// unixSocketAlive reports whether something is listening on path. A plain
// dial: zero side effects — unlike a tmux client, which would start a
// transient server on a dead socket just to tell us it is dead.
func unixSocketAlive(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	conn, err := net.DialTimeout("unix", path, 750*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// countSessionsOn counts the sessions on a running server, and how many of
// them carry the picode- name hint (the cheap read the inventory uses).
func countSessionsOn(ctx context.Context, path string) (total, picoded int) {
	out, err := runSocket(ctx, path, "list-sessions", "-F", "#{session_name}")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		total++
		if strings.HasPrefix(line, "picode-") {
			picoded++
		}
	}
	return total, picoded
}

// runSocket runs one tmux command against an explicit socket path.
func runSocket(ctx context.Context, path string, args ...string) (string, error) {
	full := append([]string{"-S", path}, args...)
	cmd := exec.CommandContext(ctx, "tmux", full...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
