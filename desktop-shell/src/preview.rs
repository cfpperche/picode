//! The decision half of a work-tab capture (ADR-0161's legacy hide/capture
//! path): `capture_png` and `btab_preview` in `btab.rs` drive the COM half on
//! the UI thread and learn the outcome through a channel; this module is
//! everything that channel answer and the file on disk decide, so the rows
//! below run on any host (`rustc --edition 2021 --test src/preview.rs`).
//! The COM calls themselves — CoreWebView2, the file stream, CapturePreview —
//! stay Windows-only; what they report lands here as a `Result`.
//!
//! The rows the handoff names, and where each lives:
//! - **valid**        — `Settled::Painted` + `read_still` returns the bytes
//! - **failed**       — `Settled::Failed` (CoreWebView2, stream, call, handler hr)
//! - **timeout**      — `Settled::TimedOut` (the 8 s budget elapsed)
//! - **gone**         — `Settled::Gone` (every sender dropped: the webview
//!   closed mid-capture). The old code answered "capture timed out" for this
//!   row, which named the wrong failure; a gone tab is not a slow capture.
//! - **not painted**  — `read_still` refuses zero bytes: CapturePreview can
//!   resolve successfully with an empty file when the page has not composited
//!   a frame yet, and a still of nothing is the uniform gray the menu once
//!   hid the live page behind (owner report 2026-09-16). Pixels are required.

use std::path::Path;
use std::sync::mpsc::{Receiver, RecvTimeoutError};
use std::time::Duration;

/// How one capture ended. `Painted` means a PNG sits on disk — possibly a
/// zero-byte one; the empty row is `read_still`'s to refuse.
#[derive(Debug, PartialEq, Eq)]
pub enum Settled {
    /// The completed handler reported success.
    Painted,
    /// A step failed and said why: `CoreWebView2`, `stream`, `CapturePreview`
    /// or the completed handler's `hr`.
    Failed(String),
    /// Nothing came back inside the budget.
    TimedOut,
    /// Every sender dropped before a word: the closure never ran, so the
    /// webview (or the tab) is gone.
    Gone,
}

/// Wait one capture out. The UI thread sends exactly one `Ok(())` on success
/// or one `Err(reason)` from any failed step; this reads that one word and
/// classifies silence.
pub fn settle(rx: Receiver<Result<(), String>>, budget: Duration) -> Settled {
    match rx.recv_timeout(budget) {
        Ok(Ok(())) => Settled::Painted,
        Ok(Err(e)) => Settled::Failed(e),
        Err(RecvTimeoutError::Timeout) => Settled::TimedOut,
        Err(RecvTimeoutError::Disconnected) => Settled::Gone,
    }
}

/// The still a preview may serve: the capture's file read back, the temp
/// file taken down either way, and the pixels required non-empty. An empty
/// answer is an error, not a still — the tab hides the live page behind
/// whatever an answered IPC hands it, and an empty blob hides it behind
/// nothing.
pub fn read_still(path: &Path) -> Result<Vec<u8>, String> {
    let bytes = std::fs::read(path).map_err(|e| format!("preview: {e}"))?;
    let _ = std::fs::remove_file(path);
    if bytes.is_empty() {
        return Err("preview: empty capture — the page has not painted".into());
    }
    Ok(bytes)
}

#[cfg(test)]
mod tests {
    use super::{read_still, settle, Settled};
    use std::sync::mpsc;
    use std::time::Duration;

    // The real PNG signature: a "valid" row that is not four plausible bytes.
    const PNG: &[u8] = &[0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A];

    fn temp_still(name: &str, bytes: &[u8]) -> std::path::PathBuf {
        let path = std::env::temp_dir().join(format!("picode-preview-test-{}-{name}", std::process::id()));
        std::fs::write(&path, bytes).unwrap();
        path
    }

    #[test]
    fn capture_outcomes_classify() {
        // Valid: the completed handler reports success.
        let (tx, rx) = mpsc::channel();
        tx.send(Ok(())).unwrap();
        assert_eq!(settle(rx, Duration::from_secs(1)), Settled::Painted);

        // Failed: every failing step — CoreWebView2, the stream, the call,
        // the handler's hr — reports through the same Err, reason attached.
        let (tx, rx) = mpsc::channel();
        tx.send(Err("capture failed (0x8007139f)".into())).unwrap();
        assert_eq!(
            settle(rx, Duration::from_secs(1)),
            Settled::Failed("capture failed (0x8007139f)".into())
        );

        // Timeout: a capture that answers late still counts as timed out —
        // the budget is the decision, not the eventual word.
        let (tx, rx) = mpsc::channel();
        {
            let tx = tx.clone();
            std::thread::spawn(move || {
                std::thread::sleep(Duration::from_millis(300));
                let _ = tx.send(Ok(()));
            });
        }
        assert_eq!(settle(rx, Duration::from_millis(50)), Settled::TimedOut);

        // Gone: the closure never ran, every sender dropped — the tab closed
        // mid-capture, which is not a timeout and must not say it is.
        let (tx, rx) = mpsc::channel::<Result<(), String>>();
        drop(tx);
        assert_eq!(settle(rx, Duration::from_secs(1)), Settled::Gone);
    }

    #[test]
    fn a_still_requires_pixels() {
        // Valid: the bytes come back whole and the temp file is taken down.
        let path = temp_still("ok.png", PNG);
        let bytes = read_still(&path).unwrap();
        assert_eq!(bytes, PNG);
        assert!(!path.exists(), "the temp file must not outlive the read");

        // Not painted: a successful capture with zero bytes is refused —
        // hiding the live page behind it is the uniform gray of 2026-09-16.
        let path = temp_still("empty.png", &[]);
        let err = read_still(&path).unwrap_err();
        assert!(err.contains("not painted"), "{err}");
        assert!(!path.exists(), "an empty capture still cleans its file");

        // Failed read-back: the file never landed (disk, lock, gone).
        let err = read_still(&std::env::temp_dir().join(format!("picode-preview-test-missing-{}", std::process::id()))).unwrap_err();
        assert!(err.starts_with("preview:"), "{err}");
    }
}
