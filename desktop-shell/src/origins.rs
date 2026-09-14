//! The origin half of a grant, mirrored from `internal/browser/domains.go`.
//! The daemon checks `navigate` before the command leaves; the shell checks
//! navigation itself (`NavigationStarting`) — the two must agree, so this is
//! the same deliberately small rule, written twice on purpose and tested
//! against the same table:
//!
//!   - only http and https: an agent may not follow file:, data: or
//!     javascript: anywhere, listed or not;
//!   - the entry `example.com` matches that host exactly;
//!   - the entry `*.example.com` (or `.example.com`) also matches its
//!     subdomains;
//!   - the port is ignored, so `localhost` covers a dev server on any port;
//!   - matching is case-insensitive on the host.
//!
//! An empty grant allows nothing: no domains means no destination of its own
//! (the default read policy still reads the tab the human has on screen —
//! that target is the tab, not a domain).

/// The host of a URL under the same reading as Go's `url.Parse` for the
/// cases the rule cares about: scheme must be http/https, the authority's
/// host (userinfo stripped, port dropped, IPv6 brackets stripped), lowercased.
fn host_of(raw_url: &str) -> Option<String> {
    let trimmed = raw_url.trim();
    let (scheme, rest) = trimmed.split_once("://")?;
    if !scheme.eq_ignore_ascii_case("http") && !scheme.eq_ignore_ascii_case("https") {
        return None;
    }
    let end = rest
        .find(|c| c == '/' || c == '?' || c == '#')
        .unwrap_or(rest.len());
    let authority = &rest[..end];
    let host_port = match authority.rsplit_once('@') {
        Some((_, h)) => h,
        None => authority,
    };
    if host_port.is_empty() {
        return None;
    }
    let host = if let Some(after_bracket) = host_port.strip_prefix('[') {
        after_bracket.split(']').next()?
    } else {
        host_port.split(':').next()?
    };
    if host.is_empty() {
        return None;
    }
    Some(host.to_ascii_lowercase())
}

/// May this grant load this URL? The shell's twin of `browser.AllowsOrigin`.
pub fn allows_origin(domains: &[String], raw_url: &str) -> bool {
    let host = match host_of(raw_url) {
        Some(h) => h,
        None => return false,
    };
    for entry in domains {
        let want = entry.trim().to_ascii_lowercase();
        if want.is_empty() {
            continue;
        }
        // Both spellings of "this host and its subdomains": *.example.com and
        // the leading-dot .example.com are the same promise.
        if let Some(suffix) = want.strip_prefix("*.") {
            if host == suffix || host.ends_with(&format!(".{suffix}")) {
                return true;
            }
            continue;
        }
        if let Some(suffix) = want.strip_prefix('.') {
            if suffix.is_empty() {
                continue;
            }
            if host == suffix || host.ends_with(&format!(".{suffix}")) {
                return true;
            }
            continue;
        }
        if want == host {
            return true;
        }
    }
    false
}

/// The whole navigation gate, one call so the COM handler cannot half-check:
/// no grant known → the shell enforces nothing (the tab was not driven by an
/// act-capable agent yet); a user-initiated load is always sovereign; an
/// agent-caused load must sit inside the grant. The NavigationStarting
/// handler in btab.rs is this function plus COM plumbing.
pub fn gate(domains: Option<&[String]>, user_initiated: bool, raw_url: &str) -> bool {
    match domains {
        None => true,
        Some(d) => user_initiated || allows_origin(d, raw_url),
    }
}

#[cfg(test)]
mod tests {
    use super::allows_origin;

    #[test]
    fn gate_decision_table() {
        let grant = vec!["example.com".to_string()];
        let rows: &[(Option<&[String]>, bool, &str, bool)] = &[
            // (grant, user-initiated, url, allow)
            (None, false, "https://evil.test/", true), // no grant armed: nothing to enforce
            (Some(&grant), true, "https://evil.test/", true), // the user is sovereign
            (Some(&grant), false, "https://example.com/a", true), // inside the grant
            (Some(&grant), false, "https://evil.test/", false), // agent roams out: cancel
            (Some(&[]), false, "https://example.com/", false), // empty grant: no destination
        ];
        for (domains, user, url, allow) in rows {
            assert_eq!(gate(*domains, *user, url), *allow, "gate({domains:?}, {user}, {url})");
        }
    }

    #[test]
    fn mirrors_the_go_table() {
        let rows: &[(&str, Vec<&str>, &str, bool)] = &[
            ("no grant allows no destination", vec![], "https://example.com/a", false),
            ("empty grant entry is ignored", vec!["  "], "https://example.com/", false),
            ("exact host", vec!["example.com"], "https://example.com/a?b=1", true),
            ("exact host is case-insensitive", vec!["Example.COM"], "https://example.com/", true),
            ("trailing whitespace in the entry", vec![" example.com "], "https://example.com/", true),
            ("another host", vec!["example.com"], "https://evil.test/", false),
            ("subdomain needs the wildcard", vec!["example.com"], "https://docs.example.com/", false),
            ("wildcard covers the host itself", vec!["*.example.com"], "https://example.com/", true),
            ("wildcard covers subdomains", vec!["*.example.com"], "https://docs.example.com/", true),
            ("dot form covers subdomains", vec![".example.com"], "https://docs.example.com/", true),
            ("wildcard does not cover a name that merely ends the same", vec!["*.example.com"], "https://notexample.com/", false),
            ("any port is the same destination", vec!["localhost"], "http://localhost:5173/x", true),
            ("plain http is fine", vec!["example.com"], "http://example.com/", true),
            ("file: is never allowed", vec!["example.com"], "file:///etc/passwd", false),
            ("javascript: is never allowed", vec!["example.com"], "javascript:alert(1)", false),
            ("data: is never allowed", vec!["*.example.com"], "data:text/html,hi", false),
            ("hostless http is refused", vec!["example.com"], "http:///path", false),
            ("garbage is refused", vec!["example.com"], "://nope", false),
            ("empty url is refused", vec!["example.com"], "", false),
            ("userinfo does not hide the host", vec!["example.com"], "https://user@example.com/", true),
            ("ipv6 literal, port ignored", vec!["::1"], "http://[::1]:5173/x", true),
        ];
        for (name, domains, url, allow) in rows {
            let owned: Vec<String> = domains.iter().map(|s| s.to_string()).collect();
            assert_eq!(
                allows_origin(&owned, url),
                *allow,
                "allows_origin({:?}, {:#?})",
                owned,
                url
            );
            let _ = name;
        }
    }
}
