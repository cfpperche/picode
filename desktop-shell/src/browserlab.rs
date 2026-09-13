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

fn lab_data_dir() -> PathBuf {
    std::env::var("LOCALAPPDATA")
        .map(|root| PathBuf::from(root).join("picode-shell").join("browserlab"))
        .unwrap_or_else(|_| std::env::temp_dir().join("picode-shell-browserlab"))
}

pub fn open(app: &AppHandle) {
    if let Some(win) = app.get_window("browserlab") {
        let _ = win.unminimize();
        let _ = win.show();
        let _ = win.set_focus();
        return;
    }
    let win = match tauri::window::WindowBuilder::new(app, "browserlab")
        .title("PiCode — Browser lab")
        .inner_size(1180.0, 840.0)
        .build()
    {
        Ok(w) => w,
        Err(e) => {
            eprintln!("browser lab: {e}");
            return;
        }
    };
    // Test 1 starts on GitHub's login page.
    let (w, h) = (1180.0, 840.0);
    let bar = WebviewBuilder::new("labbar", WebviewUrl::App("lab.html".into())).auto_resize();
    let page = WebviewBuilder::new(
        "labpage",
        WebviewUrl::External("https://github.com/login".parse().expect("static url")),
    )
    .data_directory(lab_data_dir())
    .auto_resize();
    let _ = win.add_child(bar, LogicalPosition::new(0.0, 0.0), LogicalSize::new(w, BAR_H));
    let _ = win.add_child(
        page,
        LogicalPosition::new(0.0, BAR_H),
        LogicalSize::new(w, h - BAR_H),
    );
}

// rebuild_page swaps the page webview — the only way to change the UA, which
// WebView2 fixes per-controller at creation. Same data_directory, so the
// cookies (the login) survive the swap: that is the whole point of test 3.
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
        .data_directory(lab_data_dir())
        .auto_resize();
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
