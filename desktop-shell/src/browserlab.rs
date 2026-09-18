// browserlab.rs — the work-browser spike (docs/plans/desktop-v2.md, Phase 3).
// A throwaway two-webview window that answers the ENGINE questions before the
// product decides panel-vs-tab. Top strip (local page): address bar,
// back/forward/reload and the Chrome-UA toggle. Below: the browsed page in
// its own persistent profile (%LOCALAPPDATA%\picode-shell\browserlab), so
// logins survive restarts — the ChatGPT-Work behavior we benchmark.
//
// The five tests this window exists for:
//   1. GitHub login persists across close/reopen of the lab.
//   2. Google login with WebView2's own UA (expect the block).
//   3. Google login under the Chrome-UA override (the decisive one).
//   4. OAuth popup (NewWindow) handling.
//   5. CDP reachability of the same logged-in view (follow-up).

use std::path::PathBuf;
use std::sync::Mutex;

use tauri::webview::WebviewBuilder;
use tauri::{AppHandle, LogicalPosition, LogicalSize, Manager, WebviewUrl};

// Plain-Chrome UA — no Edg/ token, no WebView tokens. The same trick the
// ChatGPT-Work browser plays by bundling its own Chromium: the experiment is
// whether Google's embedded-browser block keys on those tokens at all.
const CHROME_UA: &str = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36";

// The strip's height in logical pixels; the page gets the rest.
const BAR_H: f64 = 54.0;

#[derive(Default)]
pub struct LabState(pub Mutex<bool>);

// Every webview of the shell shares ONE explicit profile under the app's
// own folder — %LOCALAPPDATA%\PiCode\WebView2 — instead of letting each
// surface grow its own app.picode.shell/PicodeShell/picode-shell sprawl
// (three folders, 2026-09-13). One folder to back up, one to wipe.
pub fn webview_profile() -> PathBuf {
    std::env::var("LOCALAPPDATA")
        .map(|root| PathBuf::from(root).join("PiCode").join("WebView2"))
        .unwrap_or_else(|_| std::env::temp_dir().join("PiCode-WebView2"))
}

// An installed web app (ADR-0153) gets its own user-data folder: logins
// and clear-data stay inside the app. The folder derives from the tab id
// the UI already sends (`app-<webappId>`, the shell half of the stable
// `w:app-<id>`); ids that do not match the store's minting shape fall
// back to the shared profile — a webview is never pointed at a path
// built from an unvalidated string.
pub fn webapp_profile(webapp_id: &str) -> Option<PathBuf> {
    let mut chars = webapp_id.chars();
    match chars.next() {
        Some(first) if first.is_ascii_lowercase() || first.is_ascii_digit() => {}
        _ => return None,
    }
    if webapp_id.len() > 80
        || !webapp_id
            .chars()
            .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '-')
    {
        return None;
    }
    std::env::var("LOCALAPPDATA")
        .map(|root| {
            PathBuf::from(root)
                .join("PiCode")
                .join("WebView2")
                .join("webapps")
                .join(webapp_id)
        })
        .ok()
}

// The folder an id runs in: installed web apps partition (ADR-0153);
// every other tab — work browser, lab, management — shares the profile.
pub fn profile_for(id: &str) -> PathBuf {
    match id.strip_prefix("app-") {
        Some(webapp_id) => webapp_profile(webapp_id).unwrap_or_else(webview_profile),
        None => webview_profile(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn installed_app_tabs_partition_by_webapp_id() {
        let p = profile_for("app-mail-abc123");
        assert!(p.ends_with(std::path::Path::new("webapps").join("mail-abc123")));
        assert!(webapp_profile("mail-abc123")
            .unwrap()
            .ends_with(std::path::Path::new("webapps").join("mail-abc123")));
    }

    #[test]
    fn work_tabs_and_bad_ids_share_the_work_profile() {
        assert_eq!(profile_for("7"), webview_profile());
        assert_eq!(profile_for("app-Bad/Path"), webview_profile());
        assert_eq!(profile_for("app-"), webview_profile());
        assert_eq!(profile_for("app-has space"), webview_profile());
        assert_eq!(profile_for("app-.."), webview_profile());
    }
}

pub fn init(app: &tauri::App) {
    let win = match tauri::window::WindowBuilder::new(app, "browserlab")
        .title("PiCode — Browser lab")
        .inner_size(1180.0, 840.0)
        .visible(false)
        .build()
    {
        Ok(w) => w,
        Err(e) => {
            eprintln!("browser lab: {e}");
            return;
        }
    };
    // Closing hides; the tray reopens the same webviews (and their logins).
    {
        let hidden = win.clone();
        win.on_window_event(move |e| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = e {
                api.prevent_close();
                let _ = hidden.hide();
            }
        });
    }
    // Test 1 starts on GitHub's login page.
    let (w, h) = (1180.0, 840.0);
    let bar = WebviewBuilder::new("labbar", WebviewUrl::App("lab.html".into()))
        .data_directory(webview_profile())
        .auto_resize();
    let page = WebviewBuilder::new(
        "labpage",
        WebviewUrl::External("https://github.com/login".parse().expect("static url")),
    )
    .data_directory(webview_profile())
    .auto_resize();
    let _ = win.add_child(bar, LogicalPosition::new(0.0, 0.0), LogicalSize::new(w, BAR_H));
    let _ = win.add_child(
        page,
        LogicalPosition::new(0.0, BAR_H),
        LogicalSize::new(w, h - BAR_H),
    );
}

pub fn open(app: &AppHandle) {
    if let Some(win) = app.get_window("browserlab") {
        let _ = win.unminimize();
        let _ = win.show();
        let _ = win.set_focus();
    }
}

// rebuild_page swaps the page webview — the only way to change the UA, which
// WebView2 fixes per-controller at creation. Same profile (the app's default
// user-data folder), so the cookies — the login — survive the swap: that is
// the whole point of test 3.
fn rebuild_page(app: &AppHandle, url: &str, chrome_ua: bool) -> Result<(), String> {
    let win = app.get_window("browserlab").ok_or("lab window is gone")?;
    // Webview::close detaches it from the window; the strip stays put.
    if let Some(old) = app.get_webview("labpage") {
        old.close().map_err(|e| e.to_string())?;
    }
    let parsed: tauri::Url = if url.contains("://") {
        url.parse().map_err(|_| "invalid URL".to_string())?
    } else {
        format!("https://{url}")
            .parse()
            .map_err(|_| "invalid URL".to_string())?
    };
    let mut page = WebviewBuilder::new("labpage", WebviewUrl::External(parsed))
        .data_directory(webview_profile());
    page = page.auto_resize();
    if chrome_ua {
        page = page.user_agent(CHROME_UA);
    }
    let sf = win.scale_factor().unwrap_or(1.0);
    let size = win.inner_size().map_err(|e| e.to_string())?;
    let (lw, lh) = (size.width as f64 / sf, size.height as f64 / sf);
    win.add_child(
        page,
        LogicalPosition::new(0.0, BAR_H),
        LogicalSize::new(lw, lh - BAR_H),
    )
    .map(|_| ())
    .map_err(|e| e.to_string())
}

fn page_eval(app: &AppHandle, js: &str) -> Result<(), String> {
    let wv = app.get_webview("labpage").ok_or("no page yet")?;
    wv.eval(js).map_err(|e| e.to_string())
}

#[tauri::command]
pub fn lab_open(app: AppHandle) {
    open(&app);
}

#[tauri::command]
pub fn lab_navigate(
    app: AppHandle,
    state: tauri::State<'_, LabState>,
    url: String,
    chrome_ua: bool,
) -> Result<(), String> {
    let switching = {
        let mut mode = state.0.lock().unwrap();
        let switching = *mode != chrome_ua;
        *mode = chrome_ua;
        switching
    };
    if switching || app.get_webview("labpage").is_none() {
        rebuild_page(&app, &url, chrome_ua)
    } else {
        page_eval(&app, &format!("location.href = {url:?};"))
    }
}

#[tauri::command]
pub fn lab_back(app: AppHandle) -> Result<(), String> {
    page_eval(&app, "history.back()")
}

#[tauri::command]
pub fn lab_forward(app: AppHandle) -> Result<(), String> {
    page_eval(&app, "history.forward()")
}

#[tauri::command]
pub fn lab_reload(app: AppHandle) -> Result<(), String> {
    page_eval(&app, "location.reload()")
}

#[tauri::command]
pub fn lab_current_url(app: AppHandle) -> Result<String, String> {
    match app.get_webview("labpage") {
        Some(wv) => Ok(wv.url().map(|u| u.to_string()).unwrap_or_default()),
        None => Ok(String::new()),
    }
}
