package rpc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImplicitSessionName(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		cwd       string
		want      string
	}{
		{
			name:      "slug, stable session id and cwd hash",
			sessionID: "01234567-89ab-cdef-0123-456789abcdef",
			cwd:       "/home/me/My App",
			want:      "piab-my-app-1fc4c2044601f36c2f73f62f-57ea8871bbbb81b1",
		},
		{
			name:      "empty basename falls back to project",
			sessionID: "",
			cwd:       "/",
			want:      "piab-project-0bc8506e4d853e42389963d5-3372d8acf829e958",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := implicitSessionName(tt.sessionID, tt.cwd); got != tt.want {
				t.Errorf("implicitSessionName(%q, %q) = %q, want %q", tt.sessionID, tt.cwd, got, tt.want)
			}
		})
	}
}

func TestBrowserSocketRootOverride(t *testing.T) {
	if got := browserSocketRoot("/custom/root"); got != "/custom/root" {
		t.Errorf("browserSocketRoot override = %q, want /custom/root", got)
	}
}

// safeSegment edge table for discovery inputs.
func TestSafeSegment(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"piab-my-app-1fc4-57ea", true},
		{"..", false},
		{".", false},
		{"", false},
		{"a/b", false},
		{`a\b`, false},
	}
	for _, tt := range tests {
		if got := safeSegment(tt.in); got != tt.want {
			t.Errorf("safeSegment(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// No rendezvous without an identity: neither the cache nor get_state has one.
func TestBrowserStreamPortWithoutIdentity(t *testing.T) {
	t.Setenv("PI_AGENT_BROWSER_SOCKET_DIR", t.TempDir())
	ma := &ManagedAgent{AgentID: "a1", Path: t.TempDir(), capture: newCaptureState()}
	if port, ok := ma.BrowserStreamPort(context.Background()); ok || port != 0 {
		t.Fatalf("BrowserStreamPort without identity = (%d, %v), want (0, false)", port, ok)
	}
}

// The consent mirror (ADR-0115): absent, malformed and off all mean off.
func TestBrowserInputConsent(t *testing.T) {
	t.Setenv("PI_AGENT_BROWSER_SOCKET_DIR", t.TempDir())
	cwd := t.TempDir()
	sessionID := "01234567-89ab-cdef-0123-456789abcdef"
	ma := &ManagedAgent{AgentID: "a1", Path: cwd, capture: newCaptureState()}
	ma.capture.sessionID = sessionID
	ma.capture.sessionFile = "/nonexistent/session.jsonl"
	if ma.BrowserInputConsent(context.Background()) {
		t.Fatal("consent without a mirror must be false")
	}

	ma.capture.sessionFile = filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.MkdirAll(ma.capture.sessionFile+".capture", 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(ma.capture.sessionFile+".capture", "input.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		ma.capture.mu.Lock()
		ma.capture.inputCheckedAt = time.Time{} // bust the TTL cache
		ma.capture.mu.Unlock()
	}
	write(`{"on":true,"ts":1}`)
	if !ma.BrowserInputConsent(context.Background()) {
		t.Fatal("consent mirror on must be true")
	}
	write(`{"on":false,"ts":2}`)
	if ma.BrowserInputConsent(context.Background()) {
		t.Fatal("consent mirror off must be false")
	}
	write(`not json`)
	if ma.BrowserInputConsent(context.Background()) {
		t.Fatal("malformed mirror must be false")
	}
}
