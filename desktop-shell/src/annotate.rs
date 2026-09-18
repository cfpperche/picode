// annotate.rs — the in-page annotate mode's anchor on the Rust side (v2c).
//
// The mode does NOT freeze the page and never paints our HTML over the native
// view (which WebView2 forbids): the UI lives *inside* the page, injected
// through WebView2's own script API, and talks back through its WebMessage
// channel — no CDP, so no tier gate and nothing for the agent policy to
// referee. That is the shape the reference (ChatGPT Work) has and the owner
// asked for exactly: live page, highlight, pick, comment, styles, send.
//
// This module owns the two scripts and the contract between them and the
// shell. Its tests run without cargo:
//   rustc --edition 2021 --test src/annotate.rs

/// SCRIPT enters the mode: hint chip, hover highlight, click to pick (outline
/// plus pin), Esc to leave, and one message per event. Idempotent per document.
pub const SCRIPT: &str = include_str!("annotate.js");

/// EXIT_SCRIPT leaves the mode in whatever document is loaded now — the escape
/// hatch for the toolbar button, which cannot assume the script is still armed.
pub const EXIT_SCRIPT: &str =
    "(() => { const h = window.__picodeAnnotateV1; if (h && typeof h.exit === 'function') { h.exit(); } })();";

/// REINJECT_SCRIPT re-enters after a navigation when the mode is still on: the
/// new document has no script, and a hidden mode would look like a dead button.
pub const REINJECT_SCRIPT: &str = SCRIPT;

#[cfg(test)]
mod tests {
    use super::*;

    // The contract with the script: the shell relies on these pieces existing,
    // and a rename that breaks one of them silently kills the mode.
    #[test]
    fn the_script_keeps_the_pieces_the_shell_relies_on() {
        for needle in [
            "window.chrome?.webview?.postMessage",
            "attachShadow",
            "elementFromPoint",
            "Escape",
            "__picodeAnnotateV1",
            "\"pick\"",
            "\"comment\"",
            "add a comment...",
            "reposition",
            "\"enter\"",
            "\"exit\"",
        ] {
            assert!(SCRIPT.contains(needle), "the injected script lost: {needle}");
        }
    }

    // The exit script must be a no-op when the mode was never armed: the
    // toolbar can be clicked in a tab that never entered.
    #[test]
    fn the_exit_script_is_guarded() {
        assert!(EXIT_SCRIPT.contains("if (h && typeof h.exit === 'function')"));
        assert!(!EXIT_SCRIPT.contains(".exit()") || EXIT_SCRIPT.contains("typeof h.exit"));
        assert!(EXIT_SCRIPT.starts_with("(() => {"));
    }

    // Re-injection is the same script: one contract, one asset.
    #[test]
    fn reinjection_reuses_the_same_script() {
        assert_eq!(REINJECT_SCRIPT, SCRIPT);
    }

    // The pick payload carries what the note and the future editor need.
    #[test]
    fn the_pick_payload_names_its_fields() {
        for field in ["selector", "tag", "html", "rect", "styles"] {
            assert!(SCRIPT.contains(&format!("{field}:")), "pick payload lost {field}");
        }
        for prop in ["color", "background-color", "font-family", "font-size", "font-weight"] {
            assert!(SCRIPT.contains(&format!("\"{prop}\"")), "style list lost {prop}");
        }
        // opacity is not in the capture list yet (the editor adds it later):
        // the script must not pretend it is.
        assert!(!SCRIPT.contains("\"opacity\""));
    }
}
