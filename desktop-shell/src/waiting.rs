// The waiting screen's live half (the pure half is picode_shell::waitstate).
// Until 2026-09-24 the main window picked its target once, at setup: with no
// daemon found it loaded a static offline page and nothing ever navigated
// away from it — the page promised a retry it never made — and with a daemon
// found, setup blocked up to 30 s on the readiness gate before the window
// existed. Now the window always opens at once on ui/waiting.html; the health
// loop publishes each stage here and navigates the window to /desktop/ the
// moment the daemon answers.

use picode_shell::waitstate::{self, Stage};
use std::sync::{Condvar, Mutex};
use std::time::{Duration, Instant};
use tauri::Manager;

struct Current {
    stage: Stage,
    since: Instant,
    detail: String,
}

static CURRENT: Mutex<Option<Current>> = Mutex::new(None);

/// When Restart PiCode last ran: an unanswered daemon right after it is
/// "restarting", not "not answering".
static RESTART_AT: Mutex<Option<Instant>> = Mutex::new(None);

/// The health loop's sleep, cut short by a button on the page.
static NAP: (Mutex<bool>, Condvar) = (Mutex::new(false), Condvar::new());

/// Publishes a stage. The elapsed time restarts only when the stage changes,
/// so "Starting PiCode… 14 s" keeps counting across ticks. `detail` is the
/// last action's error, shown under the line; a new stage clears it.
pub fn set(app: &tauri::AppHandle, stage: Stage) {
    {
        let mut cur = CURRENT.lock().expect("waiting");
        match cur.as_mut() {
            Some(c) if c.stage == stage => {}
            _ => {
                *cur = Some(Current {
                    stage,
                    since: Instant::now(),
                    detail: String::new(),
                })
            }
        }
    }
    push(app);
}

/// Publishes a stage only if none was published yet (the first lookup).
pub fn set_if_unset(app: &tauri::AppHandle, stage: Stage) {
    if CURRENT.lock().expect("waiting").is_none() {
        set(app, stage);
    }
}

pub fn note_restart(app: &tauri::AppHandle) {
    *RESTART_AT.lock().expect("waiting") = Some(Instant::now());
    set(app, Stage::Restarting);
}

/// The stage for a known address that does not answer, failing since
/// `since`; a recent restart restarts the clock.
pub fn unanswered(since: Instant) -> Stage {
    let restart = *RESTART_AT.lock().expect("waiting");
    let from = match restart {
        Some(r) if r > since => r,
        _ => since,
    };
    let recent = restart.is_some_and(|r| r.elapsed() < waitstate::START_GRACE * 2);
    waitstate::unanswered(from.elapsed(), recent)
}

/// Records an action's failure against the current stage.
pub fn fail(app: &tauri::AppHandle, detail: &str) {
    if let Some(c) = CURRENT.lock().expect("waiting").as_mut() {
        c.detail = detail.to_string();
    }
    push(app);
}

fn payload() -> String {
    match CURRENT.lock().expect("waiting").as_ref() {
        Some(c) => waitstate::payload(c.stage, c.since.elapsed(), &c.detail),
        None => waitstate::payload(Stage::Linux, Duration::ZERO, ""),
    }
}

/// The main content webview, when it is showing a bundled page.
fn waiting_webview(app: &tauri::AppHandle) -> Option<tauri::Webview> {
    let wv = app.get_webview("main-content")?;
    let url = wv.url().ok()?;
    waitstate::is_local(url.as_str()).then_some(wv)
}

pub fn on_waiting_page(app: &tauri::AppHandle) -> bool {
    waiting_webview(app).is_some()
}

fn push(app: &tauri::AppHandle) {
    if let Some(wv) = waiting_webview(app) {
        let _ = wv.eval(format!(
            "window.__picodeWaiting && window.__picodeWaiting({})",
            payload()
        ));
    }
}

/// Leaves the waiting page for the daemon's /desktop/.
pub fn navigate(app: &tauri::AppHandle, base: &str) {
    let Some(wv) = waiting_webview(app) else { return };
    let Ok(mut url) = tauri::Url::parse(base) else { return };
    url.set_path("/desktop/");
    url.set_query(None);
    if let Err(e) = wv.navigate(url) {
        eprintln!("waiting: cannot open PiCode ({e})");
    }
}

/// Sleeps up to `secs`, or until a button asks for the next tick now.
pub fn nap(secs: u64) {
    let (lock, cv) = &NAP;
    let mut woken = lock.lock().expect("nap");
    if !*woken {
        woken = cv
            .wait_timeout(woken, Duration::from_secs(secs))
            .expect("nap")
            .0;
    }
    *woken = false;
}

fn wake() {
    let (lock, cv) = &NAP;
    *lock.lock().expect("nap") = true;
    cv.notify_all();
}

/// The page asks once on load, then receives every change through eval.
#[tauri::command]
pub fn waiting_state() -> String {
    payload()
}

/// The page's buttons. Each runs off the event thread: every one of them
/// spawns wsl.exe or a Windows tool.
#[tauri::command]
pub fn waiting_action(app: tauri::AppHandle, action: String) -> Result<(), String> {
    // A new attempt clears the last one's error; a new failure writes it back.
    if let Some(c) = CURRENT.lock().expect("waiting").as_mut() {
        c.detail.clear();
    }
    match action.as_str() {
        "retry" => {
            // A lookup that found nothing is asked again from the top, and
            // says so at once rather than after the next tick.
            let not_found = matches!(
                CURRENT.lock().expect("waiting").as_ref(),
                Some(c) if c.stage == Stage::NotFound
            );
            if not_found {
                set(&app, Stage::Linux);
            }
            wake();
        }
        "start" => {
            std::thread::spawn(|| {
                crate::restart_flow();
                wake();
            });
        }
        "logs" => {
            std::thread::spawn(crate::logs_flow);
        }
        "trust" => {
            std::thread::spawn(move || {
                if let Err(e) = trust_ca() {
                    fail(&app, &e);
                }
                wake();
            });
        }
        other => return Err(format!("unknown action {other}")),
    }
    Ok(())
}

const NO_CA: &str = "PiCode's certificate is missing in Linux. Run PiCode Desktop setup again.";

/// Trusts the distro's mkcert root for this Windows user — the same export
/// the installer does (internal/desktop ExportCA), into CurrentUser\Root:
/// no administrator rights, and Windows itself asks the human to confirm.
fn trust_ca() -> Result<(), String> {
    let board = crate::board::get().ok_or("PiCode is still starting. Try again in a moment.")?;
    let distro = board
        .distro()
        .ok_or("Linux has not answered yet. Try again in a moment.")?;
    let caroot = crate::wsl_output(&distro, &["mkcert", "-CAROOT"])
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty() && !s.contains(char::is_whitespace))
        .ok_or(NO_CA)?;
    let pem = crate::wsl_output(&distro, &["cat", &format!("{caroot}/rootCA.pem")])
        .filter(|p| p.contains("BEGIN CERTIFICATE"))
        .ok_or(NO_CA)?;
    let path = std::env::temp_dir().join("picode-mkcert-rootCA.cer");
    std::fs::write(&path, pem).map_err(|e| format!("Could not save the certificate ({e})."))?;
    let mut cmd = std::process::Command::new("certutil.exe");
    cmd.arg("-user").arg("-addstore").arg("Root").arg(&path);
    crate::hide_console(&mut cmd);
    let out = cmd
        .output()
        .map_err(|e| format!("Windows could not add the certificate ({e})."))?;
    if !out.status.success() {
        return Err("Windows did not add the certificate. Was the prompt declined?".to_string());
    }
    Ok(())
}
