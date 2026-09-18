// external.rs — what a PiCode button may hand to the operating system.
//
// `btab_open_external` runs `cmd /C start "" <target>`, so whatever reaches it
// is handed to the Windows shell: the guard is an allowlist of target shapes,
// and it refuses every character that could turn a target into a second
// command (quotes, spaces, `&`, `|`, `<`, `>`, `^`, backtick). It began as
// "http and https only" — the way out for links that leave the app (ADR-0122).
// The Windows Hello opener row (v2d) added one more class, `ms-settings:`, for
// the OS screens PiCode deliberately does not reimplement: saved passwords,
// passkeys and sign-in. Nothing else is handed over.
//
// Decision table (every row is a test below):
//
//   target                             | handed to the OS
//   -----------------------------------|-----------------------------------
//   https://example.com/a?b=c#d        | yes
//   http://localhost:8474/x            | yes
//   ms-settings:signinoptions          | yes
//   ms-settings:passkeys               | yes — Windows decides if it exists
//   ms-settings:                       | no — nothing to open
//   ms-settings:Sign-In                | no — upper case is not the shape
//   ms-settings:x?y=1                  | no — query shapes are undocumented
//   https://x/" & calc                 | no — the injection shape
//   a space, quote or backtick         | no
//   file://, javascript:, cmd:, data:  | no
//   ms-settings-evil:x                 | no — the prefix must match exactly
//   "" / whitespace only               | no
//
// Its tests run without cargo:  rustc --edition 2021 --test src/external.rs

/// external_target validates and normalizes an operating-system hand-off
/// target, returning what should actually be run — or None when it may not be
/// handed over at all.
pub fn external_target(url: &str) -> Option<String> {
    let raw = url.trim();
    if raw.is_empty() || raw.len() > 2048 {
        return None;
    }
    // A target is one argv element, never a second command.
    if raw.chars().any(|c| c.is_whitespace() || "\"'`&|<>^".contains(c)) {
        return None;
    }
    if let Some(rest) = raw.strip_prefix("ms-settings:") {
        let shape = !rest.is_empty()
            && rest
                .chars()
                .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '-');
        return if shape { Some(raw.to_string()) } else { None };
    }
    if raw.starts_with("http://") || raw.starts_with("https://") {
        return Some(raw.to_string());
    }
    None
}

#[cfg(test)]
mod tests {
    use super::external_target;

    #[test]
    fn the_allowlist_hands_over_web_and_settings_targets() {
        for target in [
            "https://example.com/a?b=c#d",
            "http://localhost:8474/x",
            "https://github.com/cfpperche/picode",
            "ms-settings:signinoptions",
            "ms-settings:passkeys",
            "ms-settings:privacy-microphone",
        ] {
            assert_eq!(
                external_target(target).as_deref(),
                Some(target),
                "should be handed to the OS: {target}"
            );
        }
    }

    #[test]
    fn the_allowlist_refuses_commands_and_undocumented_shapes() {
        for target in [
            "ms-settings:",
            "ms-settings:Sign-In",
            "ms-settings:x?y=1",
            "ms-settings:x&y",
            "ms-settings:x/../y",
            "ms-settings-evil:x",
            "https://x/\" & calc",
            "https://x/a b",
            "https://x/a`b",
            "file:///C:/windows",
            "javascript:alert(1)",
            "cmd:/c calc",
            "data:text/html,x",
            "",
            "   ",
        ] {
            assert_eq!(external_target(target), None, "must not be handed over: {target:?}");
        }
    }

    #[test]
    fn the_validated_string_is_what_runs() {
        assert_eq!(
            external_target("  ms-settings:signinoptions  ").as_deref(),
            Some("ms-settings:signinoptions")
        );
    }
}
