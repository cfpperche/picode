package osopen

import (
	"strings"
	"testing"
)

func TestWSLToWin(t *testing.T) {
	got, ok := WSLToWin("/mnt/e/picode-backups/x")
	if !ok || got != `E:\picode-backups\x` {
		t.Fatalf("got %q %v", got, ok)
	}
	if _, ok := WSLToWin("/home/goat"); ok {
		t.Fatal("linux path")
	}
}

func TestWindowsExplorerPath(t *testing.T) {
	if !RunningWSL() {
		t.Skip("not WSL")
	}
	if windowsExplorer() == "" {
		t.Fatal("explorer.exe not found under /mnt/c/Windows")
	}
}

// Decision table for URL hand-off: every row of ValidOpenURL's policy.
func TestValidOpenURLTable(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"https://example.com/a?b=c#d", true},
		{"http://localhost:8474/x", true},
		{"https://example.com/cb?code=x&state=y", true}, // OAuth query strings carry &
		{"  https://example.com/trim  ", true},
		{"https://sub_domain.auth0.com/authorize", true},
		{"javascript:alert(1)", false},
		{"file:///etc/passwd", false},
		{"data:text/html,<script>", false},
		{"ms-settings:signinoptions", false},
		{"https://example.com/\" & calc", false},
		{"https://example.com/a b", false},
		{"https://" + strings.Repeat("x", 3000), false},
		{"", false},
		{"   ", false},
		{"https://", false},
	}
	for _, tc := range cases {
		got, err := ValidOpenURL(tc.in)
		if (err == nil) != tc.want {
			t.Errorf("ValidOpenURL(%q) err=%v, want openable=%v", tc.in, err, tc.want)
		}
		if err == nil && got != strings.TrimSpace(tc.in) {
			t.Errorf("ValidOpenURL(%q) = %q, want the trimmed URL back", tc.in, got)
		}
	}
}

// OpenURL refuses bad targets before touching the OS — the exec boundary
// is untestable headless, so the refusal is the observable contract here.
func TestOpenURLRefusesBadTargets(t *testing.T) {
	if err := OpenURL("file:///etc/passwd"); err == nil {
		t.Fatal("file URL reached the opener")
	}
	if err := OpenURL(""); err == nil {
		t.Fatal("empty target reached the opener")
	}
}
