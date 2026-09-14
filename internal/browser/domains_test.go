package browser

import "testing"

// The decision table for the one verb with a destination. Every row is a
// condition that changes the answer, and each is covered: this is the
// security boundary the daemon owns (the shell checks navigation again).
func TestAllowsOrigin(t *testing.T) {
	rows := []struct {
		name    string
		domains []string
		url     string
		allow   bool
	}{
		{"no grant allows no destination", nil, "https://example.com/a", false},
		{"empty grant entry is ignored", []string{"  "}, "https://example.com/", false},
		{"exact host", []string{"example.com"}, "https://example.com/a?b=1", true},
		{"exact host is case-insensitive", []string{"Example.COM"}, "https://example.com/", true},
		{"trailing whitespace in the entry", []string{" example.com "}, "https://example.com/", true},
		{"another host", []string{"example.com"}, "https://evil.test/", false},
		{"subdomain needs the wildcard", []string{"example.com"}, "https://docs.example.com/", false},
		{"wildcard covers the host itself", []string{"*.example.com"}, "https://example.com/", true},
		{"wildcard covers subdomains", []string{"*.example.com"}, "https://docs.example.com/", true},
		{"dot form covers subdomains", []string{".example.com"}, "https://docs.example.com/", true},
		{"wildcard does not cover a name that merely ends the same", []string{"*.example.com"}, "https://notexample.com/", false},
		{"any port is the same destination", []string{"localhost"}, "http://localhost:5173/x", true},
		{"plain http is fine", []string{"example.com"}, "http://example.com/", true},
		{"file: is never allowed", []string{"example.com"}, "file:///etc/passwd", false},
		{"javascript: is never allowed", []string{"example.com"}, "javascript:alert(1)", false},
		{"data: is never allowed", []string{"*.example.com"}, "data:text/html,hi", false},
		{"hostless http is refused", []string{"example.com"}, "http:///path", false},
		{"garbage is refused", []string{"example.com"}, "://nope", false},
		{"empty url is refused", []string{"example.com"}, "", false},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			if got := AllowsOrigin(row.domains, row.url); got != row.allow {
				t.Fatalf("AllowsOrigin(%v, %q) = %v, want %v", row.domains, row.url, got, row.allow)
			}
		})
	}
}

// AllowsVerb is the whole gate: the tier and, for a destination verb, the
// origin. Half-checking a command is what it exists to prevent.
func TestAllowsVerb(t *testing.T) {
	nav, _ := VerbFor("navigate")
	eval, _ := VerbFor("evaluate")
	snap, _ := VerbFor("snapshot")

	rows := []struct {
		name   string
		policy Policy
		verb   Verb
		params map[string]any
		allow  bool
	}{
		{"read reaches a read verb", Policy{Tier: "read"}, snap, nil, true},
		{"read does not reach evaluate", Policy{Tier: "read"}, eval, nil, false},
		{"act reaches evaluate", Policy{Tier: "act"}, eval, nil, true},
		{"act without a domain cannot navigate", Policy{Tier: "act"}, nav,
			map[string]any{"url": "https://example.com/"}, false},
		{"act with the domain can navigate", Policy{Tier: "act", Domains: []string{"example.com"}}, nav,
			map[string]any{"url": "https://example.com/"}, true},
		{"act with another domain cannot", Policy{Tier: "act", Domains: []string{"example.com"}}, nav,
			map[string]any{"url": "https://evil.test/"}, false},
		{"a missing url cannot navigate", Policy{Tier: "act", Domains: []string{"example.com"}}, nav,
			map[string]any{}, false},
		{"a non-string url cannot navigate", Policy{Tier: "act", Domains: []string{"example.com"}}, nav,
			map[string]any{"url": 42}, false},
		{"full still needs the origin listed: a subdomain is not the host", Policy{Tier: "full", Domains: []string{"example.com"}}, nav,
			map[string]any{"url": "https://docs.example.com/"}, false},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			if got := row.policy.AllowsVerb(row.verb, row.params); got != row.allow {
				t.Fatalf("AllowsVerb = %v, want %v", got, row.allow)
			}
		})
	}
}
