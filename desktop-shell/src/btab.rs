// btab.rs — work browser tabs (Phase 3 slice 1, docs/plans/desktop-v2.md):
// each browser page is a child WebView2 of the main window, living behind an
// editor tab ("w:<id>"). The React side renders the toolbar and reports the
// viewport rect of the page region (ResizeObserver); this module positions
// the native webview there. ADR-0128: one shared profile, no debug port in
// this path, popups adopt as new tabs.

use std::collections::HashMap;
use std::sync::Mutex;

use tauri::webview::{NewWindowResponse, WebviewBuilder};
use tauri::WebviewUrl;
use tauri::{AppHandle, Emitter, LogicalPosition, LogicalSize, Manager, State};

#[derive(Default)]
pub struct BtabState {
    // Bounds the UI reported before the webview existed (first navigate).
    pub pending: Mutex<HashMap<String, (f64, f64, f64, f64)>>,
}

fn label(id: &str) -> String {
    format!("btab-{}", id)
}

fn normalize(url: &str) -> Result<tauri::Url, String> {
    let trimmed = url.trim();
    if trimmed.is_empty() {
        return Err("empty URL".into());
    }
    let full = if trimmed.contains("://") {
        trimmed.to_string()
    } else {
        format!("https://{}", trimmed)
    };
    full.parse().map_err(|_| "invalid URL".to_string())
}

// ensure creates the webview on first navigate — a browser tab with no URL
// yet is pure UI (the start card), no native surface wasted on it.
fn ensure(app: &AppHandle, id: &str, url: &str) -> Result<(), String> {
    let label = label(id);
    if app.get_webview(label.as_str()).is_some() {
        return Ok(());
    }
    let win = app.get_window("main").ok_or("main window is gone")?;
    let parsed = normalize(url)?;
    let (mut x, mut y, mut w, mut h) = (120.0, 120.0, 900.0, 640.0);
    if let Some(b) = app.state::<BtabState>().pending.lock().unwrap().remove(id) {
        (x, y, w, h) = b;
    }
    let emitter = app.clone();
    let page = WebviewBuilder::new(label, WebviewUrl::External(parsed))
        .data_directory(super::browserlab::webview_profile())
        .on_new_window(move |url, _features| {
            // Popups adopt as new editor tabs (the UI listens on this event).
            let _ = emitter.emit("btab://new", url.to_string());
            NewWindowResponse::Deny
        });
    win.add_child(page, LogicalPosition::new(x, y), LogicalSize::new(w, h))
        .map(|_| ())
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn btab_navigate(
    app: AppHandle,
    state: State<'_, BtabState>,
    id: String,
    url: String,
    x: Option<f64>,
    y: Option<f64>,
    w: Option<f64>,
    h: Option<f64>,
) -> Result<(), String> {
    if let (Some(x), Some(y), Some(w), Some(h)) = (x, y, w, h) {
        state
            .pending
            .lock()
            .unwrap()
            .insert(id.clone(), (x, y, w, h));
    }
    ensure(&app, &id, &url)?;
    let wv = app.get_webview(&label(&id)).ok_or("webview vanished")?;
    wv.navigate(normalize(&url)?).map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn btab_bounds(
    app: AppHandle,
    state: State<'_, BtabState>,
    id: String,
    x: f64,
    y: f64,
    w: f64,
    h: f64,
) -> Result<(), String> {
    match app.get_webview(&label(&id)) {
        Some(wv) => wv
            .set_bounds(tauri::Rect {
                position: LogicalPosition::new(x, y).into(),
                size: LogicalSize::new(w, h).into(),
            })
            .map_err(|e| e.to_string()),
        None => {
            // No webview yet — remember where it goes for the first navigate.
            state.pending.lock().unwrap().insert(id, (x, y, w, h));
            Ok(())
        }
    }
}

#[tauri::command]
pub async fn btab_visibility(app: AppHandle, id: String, visible: bool) -> Result<(), String> {
    match app.get_webview(&label(&id)) {
        Some(wv) => {
            if visible {
                wv.show().map_err(|e| e.to_string())
            } else {
                wv.hide().map_err(|e| e.to_string())
            }
        }
        None => Ok(()),
    }
}

#[tauri::command]
pub async fn btab_back(app: AppHandle, id: String) -> Result<(), String> {
    app.get_webview(&label(&id))
        .ok_or("no page")?
        .eval("history.back()")
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn btab_forward(app: AppHandle, id: String) -> Result<(), String> {
    app.get_webview(&label(&id))
        .ok_or("no page")?
        .eval("history.forward()")
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn btab_reload(app: AppHandle, id: String) -> Result<(), String> {
    app.get_webview(&label(&id))
        .ok_or("no page")?
        .eval("location.reload()")
        .map_err(|e| e.to_string())
}

// The active tab polls this; title is the host for now (v1).
#[tauri::command]
pub async fn btab_meta(app: AppHandle, id: String) -> Result<serde_json::Value, String> {
    match app.get_webview(&label(&id)) {
        Some(wv) => {
            let url = wv.url().map(|u| u.to_string()).unwrap_or_default();
            let title = tauri::Url::parse(&url)
                .ok()
                .and_then(|u| u.host_str().map(|s| s.to_string()))
                .unwrap_or_default();
            Ok(serde_json::json!({ "url": url, "title": title }))
        }
        None => Ok(serde_json::json!({ "url": "", "title": "" })),
    }
}

// Slice 2.1 (read tier seed): "Take a screenshot". Native CapturePreview
// (not CDP) — the PNG comes back base64 for the UI to save.
#[tauri::command]
pub async fn btab_screenshot(app: AppHandle, id: String) -> Result<String, String> {
    use std::sync::mpsc;
    use webview2_com::CapturePreviewCompletedHandler;
    use webview2_com::Microsoft::Web::WebView2::Win32::COREWEBVIEW2_CAPTURE_PREVIEW_IMAGE_FORMAT_PNG;
    use windows::core::HSTRING;
    use windows::Win32::Storage::FileSystem::FILE_ATTRIBUTE_NORMAL;
    use windows::Win32::System::Com::{STGM_CREATE, STGM_READWRITE, STGM_SHARE_DENY_WRITE};

    let wv = app.get_webview(&label(&id)).ok_or("tab not open")?;
    let (tx, rx) = mpsc::channel::<Result<(), String>>();
    let tx_err = tx.clone();
    // Data-URL downloads are blocked inside WebView2, so the shell writes
    // the PNG straight to the user's Pictures folder and the UI shows the
    // saved path.
    let dir = std::env::var("USERPROFILE")
        .map(|home| std::path::PathBuf::from(home).join("Pictures").join("PiCode"))
        .unwrap_or_else(|_| std::env::temp_dir());
    let _ = std::fs::create_dir_all(&dir);
    let ms = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_millis())
        .unwrap_or_default();
    let path = dir.join(format!("picode-{id}-{ms}.png"));
    let shot_path = path.clone();

    wv.with_webview(move |platform| unsafe {
        let core = match platform.controller().CoreWebView2() {
            Ok(c) => c,
            Err(e) => {
                let _ = tx_err.send(Err(format!("CoreWebView2: {e}")));
                return;
            }
        };
        let stream = match windows::Win32::UI::Shell::SHCreateStreamOnFileEx(
            &HSTRING::from(shot_path.to_string_lossy().as_ref()),
            (STGM_CREATE.0 | STGM_READWRITE.0 | STGM_SHARE_DENY_WRITE.0) as u32,
            FILE_ATTRIBUTE_NORMAL.0,
            true,
            None,
        ) {
            Ok(s) => s,
            Err(e) => {
                let _ = tx_err.send(Err(format!("stream: {e}")));
                return;
            }
        };
        let tx = tx.clone();
        let handler = CapturePreviewCompletedHandler::create(Box::new(move |hr| {
            let _ = tx.send(hr.map_err(|e| format!("capture failed ({e})")));
            Ok(())
        }));
        if let Err(e) = core.CapturePreview(COREWEBVIEW2_CAPTURE_PREVIEW_IMAGE_FORMAT_PNG, &stream, &handler) {
            let _ = tx_err.send(Err(format!("CapturePreview: {e}")));
        }
    });

    match rx.recv_timeout(std::time::Duration::from_secs(8)) {
        Ok(Ok(())) => {}
        Ok(Err(e)) => return Err(e),
        Err(_) => return Err("capture timed out".into()),
    }
    Ok(path.to_string_lossy().to_string())
}

// Slice 2 UX: an HTML popover can never paint over the WebView2 surface
// (sibling HWNDs), so the menu flips the native Z order instead — the UI
// webview rises above the page while the options menu is open. This is
// how the ChatGPT Work shell gets its menu over the page too.
fn hwnd_of(wv: &tauri::Webview) -> Result<isize, String> {
    use std::sync::mpsc;
    let (tx, rx) = mpsc::channel::<Result<isize, String>>();
    wv.with_webview(move |platform| unsafe {
        let mut hwnd = windows::Win32::Foundation::HWND::default();
        let v = platform
            .controller()
            .ParentWindow(&mut hwnd)
            .map(|_| hwnd.0 as isize) // HWND( *mut c_void )
            .map_err(|e| e.to_string());
        let _ = tx.send(v);
    });
    rx.recv_timeout(std::time::Duration::from_secs(3))
        .map_err(|_| "hwnd timeout".to_string())?
}

fn place(page_hwnd: isize, above: isize) -> Result<(), String> {
    use windows::Win32::UI::WindowsAndMessaging::{SetWindowPos, SWP_NOACTIVATE, SWP_NOMOVE, SWP_NOSIZE};
    unsafe {
        SetWindowPos(
            windows::Win32::Foundation::HWND(page_hwnd as *mut _),
            Some(windows::Win32::Foundation::HWND(above as *mut _)),
            0,
            0,
            0,
            0,
            SWP_NOMOVE | SWP_NOSIZE | SWP_NOACTIVATE,
        )
        .map_err(|e| e.to_string())
    }
}

#[tauri::command]
pub async fn btab_layer(app: AppHandle, id: String, menu_open: bool) -> Result<(), String> {
    let page = app.get_webview(&label(&id)).ok_or("tab not open")?;
    let ui = app.get_webview("main").ok_or("main webview missing")?;
    let page_hwnd = hwnd_of(&page)?;
    let ui_hwnd = hwnd_of(&ui)?;
    // menu closed → page above the UI (normal browsing);
    // menu open   → UI above the page (the menu shows over it)
    if menu_open {
        place(ui_hwnd, page_hwnd)
    } else {
        place(page_hwnd, ui_hwnd)
    }
}

#[tauri::command]
pub async fn btab_close(app: AppHandle, id: String) -> Result<(), String> {
    if let Some(wv) = app.get_webview(&label(&id)) {
        wv.close().map_err(|e| e.to_string())?;
    }
    app.state::<BtabState>().pending.lock().unwrap().remove(&id);
    Ok(())
}
