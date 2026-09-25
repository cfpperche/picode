//! The waiting screen's pure half: which stage the main window shows while
//! the daemon is not ready to be loaded, how a strict TLS probe is read, and
//! the JSON the page receives. No tauri, so the tests run on any host; the
//! statics, commands and navigation live in the binary (waiting.rs).
//!
//! The copy lives in the page (ui/waiting.html), keyed by `Stage::key`:
//! Rust says what is true, the page says it in words.

use std::time::Duration;

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum Stage {
    /// wsl.exe is being asked where PiCode answers; on a cold boot this is
    /// the distro starting.
    Linux,
    /// server.json was found but nothing answers yet, inside the grace
    /// window a starting service needs.
    Starting,
    /// A move, backup or WSL update holds the distro down (crate::hold).
    Paused,
    /// No distro reported a PiCode server at all.
    NotFound,
    /// Known address, no answer past the grace window.
    NotAnswering,
    /// Restart PiCode was asked for; the next answer ends it.
    Restarting,
    /// The daemon answers, but Windows does not trust its certificate: the
    /// webview would land on an error page.
    Untrusted,
    /// Healthy; waiting for /desktop/ to answer 200 before navigating.
    Opening,
}

impl Stage {
    pub fn key(self) -> &'static str {
        match self {
            Stage::Linux => "linux",
            Stage::Starting => "starting",
            Stage::Paused => "paused",
            Stage::NotFound => "not-found",
            Stage::NotAnswering => "not-answering",
            Stage::Restarting => "restarting",
            Stage::Untrusted => "untrusted",
            Stage::Opening => "opening",
        }
    }

    /// A wait in progress (the page shows motion and the elapsed time), as
    /// opposed to a stop that needs the human.
    pub fn busy(self) -> bool {
        matches!(
            self,
            Stage::Linux | Stage::Starting | Stage::Paused | Stage::Restarting | Stage::Opening
        )
    }
}

/// How long an unanswered daemon counts as "starting" before the screen
/// says it is not answering. systemd brings the service up in a few seconds
/// once the distro runs; a cold distro plus a first boot has been seen near
/// 15 s, so the window is generous rather than alarming.
pub const START_GRACE: Duration = Duration::from_secs(25);

/// The stage for a daemon whose address is known but that does not answer.
pub fn unanswered(failing_for: Duration, restarting: bool) -> Stage {
    if restarting && failing_for < START_GRACE * 2 {
        Stage::Restarting
    } else if failing_for < START_GRACE {
        Stage::Starting
    } else {
        Stage::NotAnswering
    }
}

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum Cert {
    Trusted,
    Untrusted,
    /// The strict probe failed for a reason that is not the certificate
    /// (timeout, reset): say nothing about trust and let the webview try.
    Unknown,
}

/// Reads one strict probe (`curl` without `-k`, `--ssl-no-revoke`: an mkcert
/// root has no revocation list and schannel would otherwise refuse it for
/// that alone). curl reports an untrusted peer as 60; older schannel builds
/// fold it into 35 and name the reason on stderr.
pub fn cert_verdict(exit: Option<i32>, stderr: &str) -> Cert {
    match exit {
        Some(0) => Cert::Trusted,
        Some(60) => Cert::Untrusted,
        Some(35) | Some(51) => {
            let s = stderr.to_ascii_uppercase();
            if s.contains("UNTRUSTED_ROOT")
                || s.contains("CERT_E_")
                || s.contains("SEC_E_CERT")
                || s.contains("CERTIFICATE")
            {
                Cert::Untrusted
            } else {
                Cert::Unknown
            }
        }
        _ => Cert::Unknown,
    }
}

/// Whether a webview URL is one of the shell's own bundled pages (WebView2
/// serves them from http://tauri.localhost; other platforms use tauri://).
pub fn is_local(url: &str) -> bool {
    url.starts_with("http://tauri.localhost") || url.starts_with("tauri://")
}

/// What the page receives: `{"stage":…,"busy":…,"elapsed":…,"detail":…}`.
/// Hand-written because this half has no serde.
pub fn payload(stage: Stage, elapsed: Duration, detail: &str) -> String {
    format!(
        "{{\"stage\":\"{}\",\"busy\":{},\"elapsed\":{},\"detail\":{}}}",
        stage.key(),
        stage.busy(),
        elapsed.as_secs(),
        json_string(detail)
    )
}

fn json_string(s: &str) -> String {
    let mut out = String::with_capacity(s.len() + 2);
    out.push('"');
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            // U+2028/2029 end a line inside a JS string literal: the payload
            // is spliced into eval() source, not only parsed as JSON.
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            '<' => out.push_str("\\u003c"),
            c if (c as u32) < 0x20 => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn unanswered_moves_from_starting_to_not_answering() {
        assert_eq!(unanswered(Duration::from_secs(0), false), Stage::Starting);
        assert_eq!(unanswered(START_GRACE - Duration::from_secs(1), false), Stage::Starting);
        assert_eq!(unanswered(START_GRACE, false), Stage::NotAnswering);
    }

    #[test]
    fn a_restart_gets_a_longer_window_then_says_so() {
        assert_eq!(unanswered(START_GRACE, true), Stage::Restarting);
        assert_eq!(unanswered(START_GRACE * 2, true), Stage::NotAnswering);
    }

    #[test]
    fn busy_is_every_wait_and_no_stop() {
        for s in [Stage::Linux, Stage::Starting, Stage::Paused, Stage::Restarting, Stage::Opening] {
            assert!(s.busy(), "{s:?} is a wait");
        }
        for s in [Stage::NotFound, Stage::NotAnswering, Stage::Untrusted] {
            assert!(!s.busy(), "{s:?} needs the human");
        }
    }

    #[test]
    fn cert_verdicts() {
        let table: [(Option<i32>, &str, Cert); 8] = [
            (Some(0), "", Cert::Trusted),
            (Some(60), "", Cert::Untrusted),
            (Some(35), "schannel: SEC_E_UNTRUSTED_ROOT (0x80090325)", Cert::Untrusted),
            (Some(35), "schannel: CertGetCertificateChain trust error CERT_E_UNTRUSTEDROOT", Cert::Untrusted),
            (Some(35), "schannel: failed to receive handshake, SSL/TLS connection failed", Cert::Unknown),
            (Some(28), "Operation timed out", Cert::Unknown),
            (Some(7), "Failed to connect", Cert::Unknown),
            (None, "", Cert::Unknown),
        ];
        for (exit, stderr, want) in table {
            assert_eq!(cert_verdict(exit, stderr), want, "exit {exit:?} stderr {stderr:?}");
        }
    }

    #[test]
    fn local_pages_are_recognised() {
        assert!(is_local("http://tauri.localhost/waiting.html"));
        assert!(is_local("tauri://localhost/waiting.html"));
        assert!(!is_local("https://localhost:8445/desktop/"));
    }

    #[test]
    fn payload_is_json_safe_for_eval() {
        let p = payload(Stage::Untrusted, Duration::from_secs(12), "a \"b\"\n</script>\u{2028}");
        assert_eq!(
            p,
            "{\"stage\":\"untrusted\",\"busy\":false,\"elapsed\":12,\"detail\":\"a \\\"b\\\"\\n\\u003c/script>\\u2028\"}"
        );
    }
}
