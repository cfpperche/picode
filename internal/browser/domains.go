package browser

import (
	"net/url"
	"strings"
)

// AllowsOrigin is the domain half of a grant (ADR-0128): may this agent load
// this URL? It is the daemon's own check on the one verb that names a
// destination (`navigate`); the shell checks navigation itself as well, and
// the two must agree, so the rule is deliberately small and written here in
// full:
//
//   - only http and https: an agent may not follow file:, data: or
//     javascript: anywhere, listed or not;
//   - the entry `example.com` matches that host exactly;
//   - the entry `*.example.com` (or `.example.com`) also matches its
//     subdomains;
//   - the port is ignored, so `localhost` covers a dev server on any port;
//   - matching is case-insensitive on the host.
//
// An empty grant allows nothing: no domains means no destination of its own
// (the default read policy still reads the tab the human has on screen — that
// target is the tab, not a domain).
func AllowsOrigin(domains []string, rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	for _, entry := range domains {
		want := strings.ToLower(strings.TrimSpace(entry))
		if want == "" {
			continue
		}
		// Both spellings of "this host and its subdomains": *.example.com and
		// the leading-dot .example.com are the same promise.
		if suffix, ok := strings.CutPrefix(want, "*."); ok {
			if host == suffix || strings.HasSuffix(host, "."+suffix) {
				return true
			}
			continue
		}
		if suffix, ok := strings.CutPrefix(want, "."); ok && suffix != "" {
			if host == suffix || strings.HasSuffix(host, "."+suffix) {
				return true
			}
			continue
		}
		if want == host {
			return true
		}
	}
	return false
}

// Allows is the whole policy for one verb: the tier reaches it, and, when the
// verb names a destination, the grant allows that origin. It is one call so a
// caller cannot half-check a command (the route used to check the tier alone).
func (p Policy) AllowsVerb(v Verb, params map[string]any) bool {
	if !p.Allows(v) {
		return false
	}
	if !v.NeedsURL {
		return true
	}
	raw, _ := params["url"].(string)
	return AllowsOrigin(p.Domains, raw)
}
