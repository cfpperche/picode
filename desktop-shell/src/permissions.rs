//! The site-permission policy half of the work browser (ADR-0128's shared
//! profile; slice 3, "Browser permissions"): the map the Settings dialog
//! writes (`btab_set_permission_policy`) and the `PermissionRequested`
//! handler in `btab.rs` reads. Pure, so the decision table below runs on any
//! host — the COM half only defers, answers and reports.
//!
//! The key is `"<site>|<kind>"` — the site is the origin's scheme and
//! authority, lowercased; the origin `*` is the every-site policy for a
//! kind. A lookup checks the site's own entry first (an "Always allow"
//! answer from the Ask prompt), then the kind's — the same order the
//! reference browser uses. No entry at all means the platform's own default
//! (deny) stands untouched.

use std::collections::HashMap;

/// The every-site origin: a kind's default policy.
pub const ANY_SITE: &str = "*";

/// The states a policy entry can hold — the words the dialog and the Ask
/// prompt write.
pub const ALLOW: &str = "allow";
pub const DENY: &str = "deny";
pub const ASK: &str = "ask";

/// The key for one entry: the site and the kind.
fn key(origin: &str, kind: &str) -> String {
    format!(
        "{}|{}",
        site_of(origin),
        kind.trim().to_ascii_lowercase()
    )
}

/// The site component of an origin string: the scheme and authority with the
/// path, query and fragment dropped, lowercased. The platform reports a
/// page's full URI, and a standing is about the site: a permission given on
/// one path must cover the next one on the same host. A string without
/// `://` is taken as the site itself (a bare host stays a bare host).
/// `pub` because the shell's `btab.rs` keys profile operations
/// (SetPermissionState / Reset) on the same value — a `pub(crate)` here
/// broke the cross-build the day allow-once landed (8117fd50).
pub fn site_of(origin: &str) -> String {
    let trimmed = origin.trim();
    let start = trimmed.find("://").map(|i| i + 3).unwrap_or(0);
    let tail = &trimmed[start..];
    let end = tail
        .find(|c| c == '/' || c == '?' || c == '#')
        .unwrap_or(tail.len());
    trimmed[..start + end].to_ascii_lowercase()
}

fn entry_key(origin: Option<&str>, kind: &str) -> String {
    match origin.map(str::trim).unwrap_or("") {
        "" => key(ANY_SITE, kind),
        other => key(other, kind),
    }
}

/// What the policy says for this origin and kind.
pub enum Decision<'a> {
    /// The site's own standing — remembered, and it outranks the kind.
    Site(&'a str),
    /// The every-site policy for that kind.
    Kind(&'a str),
    /// No entry: the platform's own default (deny) stands.
    None,
}

/// The decision table: site first, then kind, then nothing. The COM handler
/// maps `Allow`/`Deny` to `SetState` and `Ask` to a held deferral.
pub fn decide<'a>(policy: &'a HashMap<String, String>, origin: &str, kind: &str) -> Decision<'a> {
    let kind = kind.trim().to_ascii_lowercase();
    if let Some(state) = policy.get(&key(origin, &kind)) {
        return Decision::Site(state);
    }
    match policy.get(&key(ANY_SITE, &kind)) {
        Some(state) => Decision::Kind(state),
        None => Decision::None,
    }
}

/// Write, change or forget one entry. `origin` empty or `None` targets the
/// every-site policy; the state `"default"` removes the entry so the next
/// lookup falls through.
pub fn set(
    policy: &mut HashMap<String, String>,
    origin: Option<&str>,
    kind: &str,
    state: &str,
) -> Result<(), String> {
    let kind = kind.trim().to_ascii_lowercase();
    if kind.is_empty() {
        return Err("a permission kind is required".into());
    }
    let entry = entry_key(origin, &kind);
    match state.trim().to_ascii_lowercase().as_str() {
        "allow" => {
            policy.insert(entry, ALLOW.to_string());
        }
        "deny" => {
            policy.insert(entry, DENY.to_string());
        }
        "ask" => {
            policy.insert(entry, ASK.to_string());
        }
        "default" => {
            policy.remove(&entry);
        }
        other => return Err(format!("{other:?} is not a permission state")),
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::{decide, key, set, Decision, ALLOW, ANY_SITE, ASK, DENY};
    use std::collections::HashMap;

    fn word<'a>(d: &'a Decision<'_>) -> Option<&'a str> {
        match d {
            Decision::Site(s) | Decision::Kind(s) => Some(s),
            Decision::None => None,
        }
    }

    #[test]
    fn decision_table() {
        let mut policy = HashMap::new();
        // No entry: the platform default stands.
        assert!(matches!(decide(&policy, "meet.example.com", "camera"), Decision::None));

        // The kind policy applies to every site.
        set(&mut policy, None, "camera", ASK).unwrap();
        let kind = decide(&policy, "meet.example.com", "camera");
        assert!(matches!(kind, Decision::Kind(ASK)));
        assert_eq!(word(&kind), Some(ASK));

        // A site's own standing outranks the kind.
        set(&mut policy, Some("meet.example.com"), "camera", ALLOW).unwrap();
        let site = decide(&policy, "meet.example.com", "camera");
        assert!(matches!(site, Decision::Site(ALLOW)));
        assert_eq!(word(&site), Some(ALLOW));

        // Another site still reads the kind policy.
        assert!(matches!(decide(&policy, "other.test", "camera"), Decision::Kind(ASK)));
        // Another kind has no policy at all.
        assert!(matches!(decide(&policy, "meet.example.com", "microphone"), Decision::None));
    }

    #[test]
    fn rows_normalize_case_and_space() {
        let mut policy = HashMap::new();
        set(&mut policy, Some(" https://Meet.Example.COM/room?x=1 "), " Camera ", ALLOW).unwrap();
        assert_eq!(
            policy.get(&key("https://meet.example.com", "camera")).map(String::as_str),
            Some(ALLOW)
        );
        // One path's answer covers the site: the URI's path, query and case
        // do not enter the key.
        assert!(matches!(
            decide(&policy, "https://MEET.example.com/other", "CAMERA"),
            Decision::Site(ALLOW)
        ));
        // A different host is a different site.
        assert!(matches!(decide(&policy, "https://evil.test/room", "camera"), Decision::None));
    }

    #[test]
    fn a_default_forgets_only_its_own_entry() {
        let mut policy = HashMap::new();
        set(&mut policy, None, "camera", DENY).unwrap();
        set(&mut policy, Some("meet.example.com"), "camera", ALLOW).unwrap();
        set(&mut policy, Some("meet.example.com"), "camera", "default").unwrap();
        // The site entry is gone; the kind's deny is back in force.
        assert!(matches!(decide(&policy, "meet.example.com", "camera"), Decision::Kind(DENY)));
        assert!(policy.contains_key(&key(ANY_SITE, "camera")));
        set(&mut policy, None, "camera", "default").unwrap();
        assert!(policy.is_empty());
        assert!(matches!(decide(&policy, "meet.example.com", "camera"), Decision::None));
    }

    #[test]
    fn unknown_states_and_empty_kinds_are_refused() {
        let mut policy = HashMap::new();
        assert!(set(&mut policy, None, "camera", "sometimes").is_err());
        assert!(set(&mut policy, None, "  ", ALLOW).is_err());
        assert!(policy.is_empty());
    }
}
