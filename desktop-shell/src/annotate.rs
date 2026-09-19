// annotate.rs — the in-page annotate mode's anchor on the Rust side (v2c).
//
// The mode does NOT freeze the page and never paints our HTML over the native
// view (which WebView2 forbids): the UI lives *inside* the page, injected
// through WebView2's own script API, and talks back through its WebMessage
// channel — no CDP, so no tier gate and nothing for the agent policy to
// referee. That is the shape the reference (ChatGPT Work) has and the owner
// asked for exactly: live page, highlight, pick, comment, styles, send.
//
// This module owns the scripts and the contract between them and the
// shell. Its tests run without cargo:
//   rustc --edition 2021 --test src/annotate.rs

/// SCRIPT enters the mode: hover highlight, click to pick (outline plus a
/// numbered pin), an anchored card per note (icon, input, trash-this, mic,
/// Cancel, Save), saved notes collapsing to chips (text, … menu, ×), Esc to
/// leave, and one message per event. Idempotent per document.
pub const SCRIPT: &str = include_str!("annotate.js");

/// EXIT_SCRIPT leaves the mode in whatever document is loaded now — the escape
/// hatch for the toolbar button, which cannot assume the script is still armed.
pub const EXIT_SCRIPT: &str =
    "(() => { const h = window.__picodeAnnotateV1; if (h && typeof h.exit === 'function') { h.exit(); } })();";

/// CLEAR_SCRIPT discards every annotation in the loaded document without
/// leaving the mode — the strip's trash. Guarded like the exit script: the
/// toolbar can be clicked in a tab whose document never armed the mode.
pub const CLEAR_SCRIPT: &str =
    "(() => { const h = window.__picodeAnnotateV1; if (h && typeof h.clear === 'function') { h.clear(); } })();";

/// DROP_LAST_SCRIPT removes the most recent annotation without leaving the
/// mode — the strip's undo. Guarded the same way as the clear script.
pub const DROP_LAST_SCRIPT: &str =
    "(() => { const h = window.__picodeAnnotateV1; if (h && typeof h.dropLast === 'function') { h.dropLast(); } })();";

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
            "\"saved\"",
            "\"removed\"",
            "\"state\"",
            "\"exit\"",
            "add a comment...",
            "reposition",
            "\"enter\"",
            ".clear",
            "dropLast",
            "cardOpen",
            "menuOpen",
            "Dictate the note",
            "Copy text",
        ] {
            assert!(SCRIPT.contains(needle), "the injected script lost: {needle}");
        }
    }

    // Re-arming is the recovery path: the chrome retries once when the page
    // does not answer, and the page must report what it holds on `enter` —
    // otherwise a recovered channel would show an empty strip with the pins
    // still on the page (owner 2026-09-19: a saved chip and a Send that
    // never lit up).
    #[test]
    fn entering_reports_the_state_it_already_holds() {
        let enter = SCRIPT
            .split("function enter()")
            .nth(1)
            .expect("enter() is gone");
        let body = enter.split("function exit()").next().unwrap_or(enter);
        assert!(body.contains("post({ kind: \"enter\""), "enter still announces itself");
        assert!(body.contains("sync();"), "enter must also sync the state");
        // The early return is what made a re-arm silent: the page kept its
        // pins and told the chrome nothing.
        for line in body.lines() {
            let code = line.trim();
            if code.starts_with("//") {
                continue;
            }
            assert!(
                !code.contains("if (on) return"),
                "enter must not return early: a re-arm has to report"
            );
        }
    }

    // While a card is open every other open path is locked: a stray click
    // off the card must not steal it onto a new pin and drop the unsaved
    // draft (owner 2026-09-18).
    #[test]
    fn picks_stay_locked_while_the_card_is_open() {
        assert!(
            SCRIPT.contains("if (!on || cardOpen || inHost(e)) return;"),
            "page clicks must not pick while a card is open"
        );
        for needle in [
            "if (cardOpen) return; // locked: finish the open note first"
        ] {
            assert!(SCRIPT.matches(needle).count() >= 3, "pins, chips and menus lock too: {needle}");
        }
    }

    // Keystrokes typed in the card must not reach the page: outside the
    // shadow root the event retargets to the host (a div, not a form field),
    // so a page's "ignore while typing" check fails and its single-letter
    // hotkey fires — GitHub's "s" opened its search over the card mid-word
    // (owner 2026-09-18, reproduced against @github/hotkey with a real key).
    // The shield stops propagation at the shadow root, after the input took
    // the key.
    #[test]
    fn the_script_shields_its_own_keys_and_pointers_from_the_page() {
        assert!(SCRIPT.contains("const SHIELD_EVENTS = ["), "the shield list is gone");
        for ev in ["\"keydown\"", "\"keypress\"", "\"keyup\""] {
            assert!(SCRIPT.contains(ev), "the shield lost {ev}");
        }
        assert!(
            SCRIPT.contains("root.addEventListener(type, (e) => e.stopPropagation(), false)"),
            "the shield must stop propagation at the shadow root"
        );
    }

    // Enter saves only through the open-card flag: a document-level listener
    // sees the event target retargeted to the host, so `e.target === input`
    // is never true there and Enter-to-save silently did nothing.
    #[test]
    fn enter_saves_through_the_card_flag_not_the_retargeted_target() {
        assert!(!SCRIPT.contains("e.target === input"), "retargeted target decides nothing");
        assert!(SCRIPT.contains("if (e.key === \"Enter\" && cardOpen)"));
    }

    // The wire contract: the page posts the message OBJECT and lets the
    // host serialize it once. A pre-stringified text risks coming back from
    // WebMessageAsJson JSON-encoded a second time, which the chrome parses
    // into a string with no kind and drops silently (owner 2026-09-18).
    #[test]
    fn the_page_posts_objects_not_strings() {
        assert!(
            SCRIPT.contains("postMessage(msg)"),
            "the page must post the object and let the host serialize it"
        );
        assert!(
            !SCRIPT.contains("postMessage(JSON.stringify"),
            "pre-stringified posts double-encode through WebMessageAsJson"
        );
    }
    // The exit script must be a no-op when the mode was never armed: the
    // toolbar can be clicked in a tab that never entered.
    #[test]
    fn the_exit_script_is_guarded() {
        assert!(EXIT_SCRIPT.contains("if (h && typeof h.exit === 'function')"));
        assert!(!EXIT_SCRIPT.contains(".exit()") || EXIT_SCRIPT.contains("typeof h.exit"));
        assert!(EXIT_SCRIPT.starts_with("(() => {"));
    }

    // Open/closed travels in flags, never in inline styles: reading
    // `node.style.display` as state inverts the mode (inline display starts
    // as "" — hover dead until the first pick, stuck on after it; owner
    // 2026-09-18). This asserts the pattern cannot creep back in.
    #[test]
    fn the_script_reads_no_state_from_inline_styles() {
        assert!(
            !SCRIPT.contains(".style.display !=="),
            "state travels in flags, not inline styles"
        );
    }
    // the document has no instance (navigation dropped it, say).
    #[test]
    fn the_clear_script_is_guarded() {
        assert!(CLEAR_SCRIPT.contains("if (h && typeof h.clear === 'function')"));
        assert!(CLEAR_SCRIPT.starts_with("(() => {"));
    }

    // Undo rides the same guard: dropping the last pin in a dead document
    // clears nothing and fails nothing.
    #[test]
    fn the_drop_last_script_is_guarded() {
        assert!(DROP_LAST_SCRIPT.contains("if (h && typeof h.dropLast === 'function')"));
        assert!(DROP_LAST_SCRIPT.starts_with("(() => {"));
    }

    // Re-injection is the same script: one contract, one asset.
    #[test]
    fn reinjection_reuses_the_same_script() {
        assert_eq!(REINJECT_SCRIPT, SCRIPT);
    }

    // The mode lives in the document, so the shell must re-inject it on a
    // completed navigation. The guard is static (btab.rs is Windows-only
    // code — it cannot run here), and it exists because the constant sat
    // unused while the overlay died on every full page load (2026-09-19).
    #[test]
    fn the_shell_reinjects_on_a_completed_navigation() {
        let shell = include_str!("btab.rs");
        assert!(
            shell.contains("add_NavigationCompleted"),
            "the navigation hook is gone: the mode dies on a full page load"
        );
        assert!(
            shell.contains("crate::annotate::REINJECT_SCRIPT"),
            "the hook must reinject the one script"
        );
        assert!(
            shell.contains("remove_NavigationCompleted"),
            "re-arming must drop the previous hook"
        );
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
