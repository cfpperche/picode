//go:build unix

package rpc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// writeRendezvous plants a `<name>.pid` + `<name>.stream` pair.
func writeRendezvous(t *testing.T, dir, name string, pid, port int) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name+".pid"), []byte(fmt.Sprintf("%d\n", pid)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".stream"), []byte(fmt.Sprintf("%d\n", port)), 0o600); err != nil {
		t.Fatal(err)
	}
}

func touchAt(t *testing.T, path string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

const deadPID = 1 << 30 // beyond every platform's pid_max: kill(pid,0) = ESRCH

// The decision table from ADR-0114: rendezvous conditions → outcome.
func TestFindBrowserStreamPort(t *testing.T) {
	live := os.Getpid()
	tests := []struct {
		name    string
		build   func(t *testing.T, root string) string // returns sessionBase
		want    int
		wantErr bool // want 0 (no port)
	}{
		{
			name: "exact session, live pid",
			build: func(t *testing.T, root string) string {
				writeRendezvous(t, root, "piab-app-aa11-bb22", live, 9223)
				return "piab-app-aa11-bb22"
			},
			want: 9223,
		},
		{
			name: "fresh rotation, newest mtime wins",
			build: func(t *testing.T, root string) string {
				writeRendezvous(t, root, "piab-app-aa11-bb22", live, 9001)
				writeRendezvous(t, root, "piab-app-aa11-bb22-fresh-xyz", live, 9002)
				touchAt(t, filepath.Join(root, "piab-app-aa11-bb22.stream"), time.Now().Add(-time.Hour))
				return "piab-app-aa11-bb22"
			},
			want: 9002,
		},
		{
			name: "stale newest rotation skipped, older live returned",
			build: func(t *testing.T, root string) string {
				writeRendezvous(t, root, "piab-app-aa11-bb22", live, 9001)
				writeRendezvous(t, root, "piab-app-aa11-bb22-fresh-xyz", deadPID, 9002)
				touchAt(t, filepath.Join(root, "piab-app-aa11-bb22-fresh-xyz.stream"), time.Now().Add(-time.Minute))
				return "piab-app-aa11-bb22"
			},
			want: 9001,
		},
		{
			name: "dead pid everywhere",
			build: func(t *testing.T, root string) string {
				writeRendezvous(t, root, "piab-app-aa11-bb22", deadPID, 9223)
				return "piab-app-aa11-bb22"
			},
			wantErr: true,
		},
		{
			name: "port above 65535 refused",
			build: func(t *testing.T, root string) string {
				writeRendezvous(t, root, "piab-app-aa11-bb22", live, 70000)
				return "piab-app-aa11-bb22"
			},
			wantErr: true,
		},
		{
			name: "missing pid file",
			build: func(t *testing.T, root string) string {
				if err := os.WriteFile(filepath.Join(root, "piab-app-aa11-bb22.stream"), []byte("9223\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				return "piab-app-aa11-bb22"
			},
			wantErr: true,
		},
		{
			name: "non-numeric stream file",
			build: func(t *testing.T, root string) string {
				writeRendezvous(t, root, "piab-app-aa11-bb22", live, 9223)
				if err := os.WriteFile(filepath.Join(root, "piab-app-aa11-bb22.stream"), []byte("not-a-port"), 0o600); err != nil {
					t.Fatal(err)
				}
				return "piab-app-aa11-bb22"
			},
			wantErr: true,
		},
		{
			name: "oversized stream file",
			build: func(t *testing.T, root string) string {
				writeRendezvous(t, root, "piab-app-aa11-bb22", live, 9223)
				if err := os.WriteFile(filepath.Join(root, "piab-app-aa11-bb22.stream"), []byte("9223\nand then a lot more text than sixteen bytes"), 0o600); err != nil {
					t.Fatal(err)
				}
				return "piab-app-aa11-bb22"
			},
			wantErr: true,
		},
		{
			name: "other sessions ignored",
			build: func(t *testing.T, root string) string {
				writeRendezvous(t, root, "piab-other-cc33-dd44", live, 9100)
				return "piab-app-aa11-bb22"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Chmod(root, 0o700); err != nil {
				t.Fatal(err)
			}
			base := tt.build(t, root)
			got := findBrowserStreamPort(root, base, "")
			if tt.wantErr && got != 0 {
				t.Fatalf("findBrowserStreamPort = %d, want 0", got)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("findBrowserStreamPort = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFindBrowserStreamPortRefusals(t *testing.T) {
	live := os.Getpid()
	base := "piab-app-aa11-bb22"

	t.Run("symlinked root refused", func(t *testing.T) {
		real := t.TempDir()
		writeRendezvous(t, real, base, live, 9223)
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(real, link); err != nil {
			t.Fatal(err)
		}
		if got := findBrowserStreamPort(link, base, ""); got != 0 {
			t.Fatalf("symlinked root gave port %d, want 0", got)
		}
	})

	t.Run("root mode 0755 refused", func(t *testing.T) {
		root := t.TempDir()
		writeRendezvous(t, root, base, live, 9223)
		if err := os.Chmod(root, 0o755); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(root, 0o700)
		if got := findBrowserStreamPort(root, base, ""); got != 0 {
			t.Fatalf("0755 root gave port %d, want 0", got)
		}
	})

	t.Run("symlinked stream file refused (O_NOFOLLOW)", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Chmod(root, 0o700); err != nil {
			t.Fatal(err)
		}
		realPort := filepath.Join(t.TempDir(), "real.stream")
		if err := os.WriteFile(realPort, []byte("9223\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, base+".pid"), []byte(fmt.Sprintf("%d\n", live)), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(realPort, filepath.Join(root, base+".stream")); err != nil {
			t.Fatal(err)
		}
		if got := findBrowserStreamPort(root, base, ""); got != 0 {
			t.Fatalf("symlinked .stream gave port %d, want 0", got)
		}
	})

	t.Run("namespace dir honored", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Chmod(root, 0o700); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(root, "namespaces", "ns1", "run")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeRendezvous(t, dir, base, live, 9310)
		if got := findBrowserStreamPort(root, base, "ns1"); got != 9310 {
			t.Fatalf("namespace discovery = %d, want 9310", got)
		}
		if got := findBrowserStreamPort(root, base, ""); got != 0 {
			t.Fatalf("same session found outside its namespace: %d", got)
		}
	})
}

// streamRootTrust table: uid ownership and the exact 0700 mask.
func TestStreamRootTrust(t *testing.T) {
	info := func(uid uint32, mode os.FileMode) os.FileInfo {
		fi, err := os.Lstat(filepath.Join(t.TempDir(), "missing"))
		if err == nil {
			t.Fatal("unexpected file")
		}
		_ = fi
		return &statTInfo{uid: uid, mode: mode}
	}
	if !streamRootTrust(info(uint32(os.Getuid()), 0o700)) {
		t.Error("own 0700 dir should be trusted")
	}
	if streamRootTrust(info(uint32(os.Getuid()), 0o755)) {
		t.Error("0755 dir should be refused")
	}
	if streamRootTrust(info(uint32(os.Getuid()+1), 0o700)) {
		t.Error("foreign-owned dir should be refused")
	}
}

// Full path through ManagedAgent.BrowserStreamPort with a primed cache.
func TestBrowserStreamPortWithPrimedCache(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PI_AGENT_BROWSER_SOCKET_DIR", root)
	cwd := t.TempDir()
	sessionID := "01234567-89ab-cdef-0123-456789abcdef"
	writeRendezvous(t, root, implicitSessionName(sessionID, cwd), os.Getpid(), 9223)

	ma := &ManagedAgent{AgentID: "a1", Path: cwd, capture: newCaptureState()}
	ma.capture.sessionID = sessionID
	ma.capture.sessionFile = filepath.Join(cwd, "session.jsonl")

	port, ok := ma.BrowserStreamPort(context.Background())
	if !ok || port != 9223 {
		t.Fatalf("BrowserStreamPort = (%d, %v), want (9223, true)", port, ok)
	}
}

// statTInfo is a minimal os.FileInfo carrying a syscall.Stat_t for the
// trust checks.
type statTInfo struct {
	uid  uint32
	mode os.FileMode
}

func (s *statTInfo) Name() string       { return "x" }
func (s *statTInfo) Size() int64        { return 0 }
func (s *statTInfo) Mode() os.FileMode  { return s.mode }
func (s *statTInfo) ModTime() time.Time { return time.Time{} }
func (s *statTInfo) IsDir() bool        { return false }
func (s *statTInfo) Sys() any {
	return &syscall.Stat_t{Uid: s.uid}
}
