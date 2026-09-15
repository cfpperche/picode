package preview

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestMintGetExpire(t *testing.T) {
	s := NewStore(time.Minute)
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }

	tk, err := s.Mint(Request{OwnerKind: "agent", OwnerID: "a1", Root: "/proj", Path: "site/index.html", SessionID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if tk.Token == "" || tk.Path != "site/index.html" || tk.Root != "/proj" || tk.SessionID != "s1" {
		t.Fatalf("ticket=%+v", tk)
	}
	if !tk.ExpiresAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("expiry=%v", tk.ExpiresAt)
	}
	got, ok := s.Get(tk.Token)
	if !ok || got.Token != tk.Token || got.OwnerID != "a1" {
		t.Fatalf("get=%+v ok=%v", got, ok)
	}
	if s.Len() != 1 {
		t.Fatalf("len=%d", s.Len())
	}

	now = now.Add(time.Minute)
	if _, ok := s.Get(tk.Token); ok {
		t.Fatal("expired ticket still resolves")
	}
	if s.Len() != 0 {
		t.Fatalf("expired tickets linger: len=%d", s.Len())
	}
}

func TestStoreLazySweepAndZeroTTL(t *testing.T) {
	s := NewStore(0)
	if s.ttl != DefaultTTL {
		t.Fatalf("ttl=%v", s.ttl)
	}
	now := time.Now()
	s.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if _, err := s.Mint(Request{OwnerKind: "term", OwnerID: "t1", Root: "/p", Path: "a.html", SessionID: "s"}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Len() != 3 {
		t.Fatalf("len=%d", s.Len())
	}
	now = now.Add(DefaultTTL)
	if _, err := s.Mint(Request{OwnerKind: "term", OwnerID: "t1", Root: "/p", Path: "a.html", SessionID: "s"}); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 1 {
		t.Fatalf("sweep left %d tickets", s.Len())
	}
}

func TestTokensAreUniqueAndOpaque(t *testing.T) {
	s := NewStore(time.Minute)
	seen := map[string]bool{}
	for i := 0; i < 64; i++ {
		tk, err := s.Mint(Request{OwnerKind: "agent", OwnerID: "a", Root: "/p", Path: "index.html", SessionID: ""})
		if err != nil {
			t.Fatal(err)
		}
		if seen[tk.Token] {
			t.Fatalf("duplicate token %q", tk.Token)
		}
		seen[tk.Token] = true
		if raw, err := base64.RawURLEncoding.DecodeString(tk.Token); err != nil || len(raw) != TokenBytes {
			t.Fatalf("token %q is not %d raw bytes: %v", tk.Token, TokenBytes, err)
		}
	}
}

func TestMIMETypeTable(t *testing.T) {
	rows := []struct {
		path string
		want string
		ok   bool
	}{
		{"index.html", "text/html; charset=utf-8", true},
		{"INDEX.HTM", "text/html; charset=utf-8", true},
		{"a/b.css", "text/css; charset=utf-8", true},
		{"m.js", "text/javascript; charset=utf-8", true},
		{"m.mjs", "text/javascript; charset=utf-8", true},
		{"d.json", "application/json; charset=utf-8", true},
		{"x.wasm", "application/wasm", true},
		{"f.woff2", "font/woff2", true},
		{"i.avif", "image/avif", true},
		{"v.webm", "video/webm", true},
		{"m.gltf", "model/gltf+json", true},
		{".env", "", false},
		{"id_rsa", "", false},
		{"cert.pem", "", false},
		{"notes.md", "", false},
		{"app.exe", "", false},
		{"archive.tar.gz", "", false},
	}
	for _, r := range rows {
		got, ok := MIMEType(r.path)
		if ok != r.ok || got != r.want {
			t.Fatalf("%s: got (%q,%v) want (%q,%v)", r.path, got, ok, r.want, r.ok)
		}
	}
}

func TestIsDocumentAndHidden(t *testing.T) {
	for _, p := range []string{"a.html", "b.HTM", "dir/c.html"} {
		if !IsDocument(p) {
			t.Fatalf("%s should be a document", p)
		}
	}
	for _, p := range []string{"a.xhtml", "a.md", "a", ".html"} {
		if IsDocument(p) {
			t.Fatalf("%s should not be a document", p)
		}
	}
	for _, p := range []string{".env", "a/.git/config", "dir/.hidden/x.js", "."} {
		if !Hidden(p) {
			t.Fatalf("%s should be hidden", p)
		}
	}
	for _, p := range []string{"index.html", "assets/app.js", "a.b/c"} {
		if Hidden(p) {
			t.Fatalf("%s should not be hidden", p)
		}
	}
}

func TestTouchAndWatched(t *testing.T) {
	s := NewStore(time.Minute)
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	tk, err := s.Mint(Request{OwnerKind: "agent", OwnerID: "a1", Root: "/proj", Path: "site/index.html", SessionID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Watched(tk.Token); len(got) != 0 {
		t.Fatalf("fresh ticket watches %v", got)
	}
	s.Touch(tk.Token, "/proj/site/index.html")
	s.Touch(tk.Token, "/proj/site/app.js")
	s.Touch(tk.Token, "/proj/site/app.js") // no duplicate
	s.Touch("nope", "/proj/site/x.js")
	s.Touch(tk.Token, "")
	got := s.Watched(tk.Token)
	if len(got) != 2 {
		t.Fatalf("watched=%v", got)
	}
	seen := map[string]bool{}
	for _, p := range got {
		seen[p] = true
	}
	if !seen["/proj/site/index.html"] || !seen["/proj/site/app.js"] {
		t.Fatalf("watched=%v", got)
	}
	if s.Watched("nope") != nil {
		t.Fatal("unknown token watches something")
	}

	now = now.Add(time.Minute)
	if got := s.Watched(tk.Token); got != nil {
		t.Fatalf("expired ticket watches %v", got)
	}
	s.Touch(tk.Token, "/proj/site/late.js") // expired: ignored, not resurrected
}

func TestTouchCap(t *testing.T) {
	s := NewStore(time.Minute)
	tk, err := s.Mint(Request{OwnerKind: "term", OwnerID: "t1", Root: "/p", Path: "index.html", SessionID: ""})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxWatch+50; i++ {
		s.Touch(tk.Token, fmt.Sprintf("/p/asset-%d.js", i))
	}
	if got := len(s.Watched(tk.Token)); got != MaxWatch {
		t.Fatalf("watched=%d want %d", got, MaxWatch)
	}
}

func TestOverlay(t *testing.T) {
	s := NewStore(time.Minute)
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	tk, err := s.Mint(Request{OwnerKind: "agent", OwnerID: "a1", Root: "/proj", Path: "site/index.html", SessionID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Overlay(tk.Token); ok {
		t.Fatal("fresh ticket has an overlay")
	}
	if !s.SetOverlay(tk.Token, "texto do editor") {
		t.Fatal("set overlay refused")
	}
	if got, ok := s.Overlay(tk.Token); !ok || got != "texto do editor" {
		t.Fatalf("overlay=%q ok=%v", got, ok)
	}
	if !s.SetOverlay(tk.Token, "") || func() bool { got, _ := s.Overlay(tk.Token); return got != "" }() {
		t.Fatal("an empty overlay must be a real, replaceable overlay")
	}
	if s.SetOverlay("nope", "x") {
		t.Fatal("unknown token accepted an overlay")
	}
	now = now.Add(time.Minute)
	if _, ok := s.Overlay(tk.Token); ok {
		t.Fatal("expired ticket kept its overlay")
	}
}

// TestLabel pins the DNS half of a ticket (ADR-0137): one lowercase base32
// label, unique per ticket, resolvable by name as well as by token, and gone
// the moment the ticket expires.
func TestLabel(t *testing.T) {
	s := NewStore(time.Minute)
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }

	first, err := s.Mint(Request{OwnerKind: "agent", OwnerID: "a1", Root: "/p", Path: "index.html"})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Label) != 26 {
		t.Fatalf("label=%q (len %d), want 26 base32 chars", first.Label, len(first.Label))
	}
	if first.Label != strings.ToLower(first.Label) {
		t.Fatalf("label %q is not lowercase", first.Label)
	}
	if strings.ContainsAny(first.Label, "0189") {
		t.Fatalf("label %q is not base32", first.Label)
	}
	second, err := s.Mint(Request{OwnerKind: "agent", OwnerID: "a1", Root: "/p", Path: "index.html"})
	if err != nil {
		t.Fatal(err)
	}
	if second.Label == first.Label || second.Token == first.Token {
		t.Fatalf("two tickets share an identity: %q %q", first.Label, second.Label)
	}

	got, ok := s.ByLabel(first.Label)
	if !ok || got.Token != first.Token {
		t.Fatalf("byLabel=%+v ok=%v", got, ok)
	}
	// DNS names are case-insensitive; the lookup must be too.
	if got, ok := s.ByLabel(strings.ToUpper(first.Label)); !ok || got.Token != first.Token {
		t.Fatalf("uppercase label did not resolve: %+v ok=%v", got, ok)
	}
	if _, ok := s.ByLabel("notaticket"); ok {
		t.Fatal("unknown label resolved")
	}
	if _, ok := s.ByLabel(""); ok {
		t.Fatal("empty label resolved")
	}

	// Expiry drops both names, so a stale host cannot keep serving.
	now = now.Add(time.Minute)
	if _, ok := s.ByLabel(first.Label); ok {
		t.Fatal("expired label still resolves")
	}
	if _, ok := s.Get(first.Token); ok {
		t.Fatal("expired token still resolves after the label lookup")
	}
	if s.Len() != 0 {
		t.Fatalf("a label lookup left a ticket behind: len=%d", s.Len())
	}
}

// TestClearOverlay: Save hands the document back to disk without dropping the
// ticket (ADR-0137 D5) — the origin, and with it the page's storage, stays.
func TestClearOverlay(t *testing.T) {
	s := NewStore(time.Minute)
	tk, err := s.Mint(Request{OwnerKind: "agent", OwnerID: "a1", Root: "/p", Path: "index.html"})
	if err != nil {
		t.Fatal(err)
	}
	if !s.SetOverlay(tk.Token, "buffer") {
		t.Fatal("set overlay refused")
	}
	if !s.ClearOverlay(tk.Token) {
		t.Fatal("clear overlay refused")
	}
	if _, ok := s.Overlay(tk.Token); ok {
		t.Fatal("overlay survived the clear")
	}
	if _, ok := s.Get(tk.Token); !ok {
		t.Fatal("clearing the overlay dropped the ticket")
	}
	if s.ClearOverlay("nope") {
		t.Fatal("unknown token cleared an overlay")
	}
}
