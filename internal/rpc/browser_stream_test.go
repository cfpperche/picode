package rpc

import (
	"context"
	"testing"
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
