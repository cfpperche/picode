// btab.rs — work browser tabs (Phase 3 slice 1, docs/plans/desktop-v2.md):
// each browser page is a child WebView2 of the main window, living behind an
// editor tab ("w:<id>"). The React side renders the toolbar and reports the
// viewport rect of the page region (ResizeObserver); this module positions
// the native webview there. ADR-0128: one shared profile, no debug port in
// this path, `target=_blank` and unsized `window.open` adopt as new tabs — a
// sized popup opens as a real window (OAuth needs `window.opener`).

use std::cell::RefCell;
use std::collections::{HashMap, VecDeque};
use std::sync::{mpsc, Arc, Mutex, OnceLock};
use std::time::Duration;

use picode_shell::{cdppolicy, origins};
use windows::core::Interface;
use tauri::webview::{NewWindowResponse, WebviewBuilder};
use tauri::WebviewUrl;
use tauri::{AppHandle, Emitter, LogicalPosition, LogicalSize, Manager, State};
use webview2_com::{
    take_pwstr, DownloadStartingEventHandler, NavigationStartingEventHandler,
    PermissionRequestedEventHandler, StateChangedEventHandler,
};

/// The origin grant per webview id, armed by the last act-capable CDP call
/// the daemon relayed (tier act/full carries the agent's domain table).
/// The navigation gate attached in `ensure` reads it; a user-driven
/// `btab_navigate` disarms it (the user is sovereign); `btab_close` drops
/// the entry.
fn grants() -> &'static Mutex<HashMap<String, Vec<String>>> {
    static GRANTS: OnceLock<Mutex<HashMap<String, Vec<String>>>> = OnceLock::new();
    GRANTS.get_or_init(|| Mutex::new(HashMap::new()))
}
use webview2_com::Microsoft::Web::WebView2::Win32::{
    ICoreWebView2DevToolsProtocolEventReceiver, COREWEBVIEW2_DOWNLOAD_STATE,
    COREWEBVIEW2_DOWNLOAD_STATE_COMPLETED, COREWEBVIEW2_DOWNLOAD_STATE_INTERRUPTED,
    COREWEBVIEW2_PERMISSION_KIND, COREWEBVIEW2_PERMISSION_KIND_AUTOPLAY,
    COREWEBVIEW2_PERMISSION_KIND_CAMERA, COREWEBVIEW2_PERMISSION_KIND_CLIPBOARD_READ,
    COREWEBVIEW2_PERMISSION_KIND_FILE_READ_WRITE, COREWEBVIEW2_PERMISSION_KIND_GEOLOCATION,
    COREWEBVIEW2_PERMISSION_KIND_LOCAL_FONTS, COREWEBVIEW2_PERMISSION_KIND_MICROPHONE,
    COREWEBVIEW2_PERMISSION_KIND_MIDI_SYSTEM_EXCLUSIVE_MESSAGES,
    COREWEBVIEW2_PERMISSION_KIND_NOTIFICATIONS, COREWEBVIEW2_PERMISSION_KIND_OTHER_SENSORS,
    COREWEBVIEW2_PERMISSION_STATE_ALLOW, COREWEBVIEW2_PERMISSION_STATE_DENY,
};

#[derive(Default)]
pub struct BtabState {
    // Bounds the UI reported before the webview existed (first navigate).
    pub pending: Mutex<HashMap<String, (f64, f64, f64, f64)>>,
    // Per-tab CDP event rings. The event handlers run on the UI thread and
    // append here; btab_cdp_events drains them by sequence number.
    rings: Arc<Mutex<HashMap<String, Ring>>>,
}

#[derive(Default)]
struct Ring {
    events: VecDeque<serde_json::Value>,
    seq: u64,
    dropped: u64,
    subscribed: bool,
}

/// How many events one tab keeps for a poller that fell behind.
const RING_CAP: usize = 256;

/// The events a tab records — read-tier only, every entry a notification
/// about what the page did. Deny by default holds here too: a name this table
/// does not carry is never subscribed.
const EVENTS: &[(&str, &str)] = &[
    ("Page.frameNavigated", "Page"),
    ("Page.loadEventFired", "Page"),
    ("Page.domContentEventFired", "Page"),
    ("Page.javascriptDialogOpening", "Page"),
    ("Runtime.consoleAPICalled", "Runtime"),
    ("Runtime.exceptionThrown", "Runtime"),
    ("Network.requestWillBeSent", "Network"),
    ("Network.responseReceived", "Network"),
    ("Network.loadingFailed", "Network"),
    ("Log.entryAdded", "Log"),
];

thread_local! {
    // The receivers own the event registrations — dropping one tears its
    // subscription down. They are COM interfaces (not Send), so they live on
    // the UI thread where with_webview runs instead of in Tauri state.
    static RECEIVERS: RefCell<HashMap<String, Vec<ICoreWebView2DevToolsProtocolEventReceiver>>> =
        RefCell::new(HashMap::new());
}

fn push_event(rings: &Arc<Mutex<HashMap<String, Ring>>>, tab: &str, event: &str, payload: String) {
    let params = serde_json::from_str::<serde_json::Value>(&payload)
        .unwrap_or(serde_json::Value::String(payload));
    let mut map = rings.lock().unwrap();
    let ring = map.entry(tab.to_string()).or_default();
    ring.seq += 1;
    let entry = serde_json::json!({ "seq": ring.seq, "event": event, "params": params });
    ring.events.push_back(entry);
    while ring.events.len() > RING_CAP {
        ring.events.pop_front();
        ring.dropped += 1;
    }
}

// subscribe registers this tab's read-tier event receivers, once, and enables
// the domains the table names — WebView2 delivers a CDP event only when its
// domain is enabled. Runs on the UI thread (with_webview).
fn subscribe(
    app: &AppHandle,
    id: &str,
    rings: Arc<Mutex<HashMap<String, Ring>>>,
) -> Result<(), String> {
    use webview2_com::{
        CallDevToolsProtocolMethodCompletedHandler, CoTaskMemPWSTR,
        DevToolsProtocolEventReceivedEventHandler,
    };
    use windows::core::{HSTRING, PWSTR};

    let wv = app.get_webview(&label(id)).ok_or("tab not open")?;
    let (tx, rx) = mpsc::channel::<Result<(), String>>();
    let tab = id.to_string();
    let sent = tx.clone();
    let installed = wv.with_webview(move |platform| unsafe {
        let core = match platform.controller().CoreWebView2() {
            Ok(core) => core,
            Err(e) => {
                let _ = sent.send(Err(format!("CoreWebView2: {e}")));
                return;
            }
        };
        let mut kept = Vec::new();
        let mut errors = Vec::new();
        for (event, _domain) in EVENTS {
            let receiver = match core.GetDevToolsProtocolEventReceiver(&HSTRING::from(*event)) {
                Ok(receiver) => receiver,
                Err(e) => {
                    errors.push(format!("{event}: {e}"));
                    continue;
                }
            };
            let sink = rings.clone();
            let tab = tab.clone();
            let name = (*event).to_string();
            let handler = DevToolsProtocolEventReceivedEventHandler::create(Box::new(
                move |_source, args| {
                    if let Some(args) = args {
                        let mut raw = PWSTR::null();
                        if args.ParameterObjectAsJson(&mut raw).is_ok() {
                            push_event(&sink, &tab, &name, CoTaskMemPWSTR::from(raw).to_string());
                        }
                    }
                    Ok(())
                },
            ));
            let mut token = 0i64;
            match receiver.add_DevToolsProtocolEventReceived(&handler, &mut token) {
                Ok(()) => kept.push(receiver),
                Err(e) => errors.push(format!("{event}: {e}")),
            }
        }
        for domain in ["Page", "Runtime", "Network", "Log"] {
            let noop =
                CallDevToolsProtocolMethodCompletedHandler::create(Box::new(|_hr, _json| Ok(())));
            let _ = core.CallDevToolsProtocolMethod(
                &HSTRING::from(format!("{domain}.enable")),
                &HSTRING::from("{}"),
                &noop,
            );
        }
        RECEIVERS.with(|r| {
            r.borrow_mut().insert(tab.clone(), kept);
        });
        let _ = if errors.is_empty() {
            sent.send(Ok(()))
        } else {
            sent.send(Err(errors.join("; ")))
        };
    });
    if let Err(e) = installed {
        return Err(format!("with_webview: {e}"));
    }
    match rx.recv_timeout(Duration::from_secs(8)) {
        Ok(result) => result,
        Err(mpsc::RecvTimeoutError::Disconnected) => Err("the page went away".into()),
        Err(_) => Err("subscribing to page events timed out".into()),
    }
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
        .on_new_window(move |url, features| {
            // A `window.open` that names a size or position is a popup
            // window, not a tab — this is where OAuth lives: Google Identity
            // Services opens `accounts.google.com` sized and posts the
            // credential back through `window.opener`, and adopting that URL
            // as a tab severs the opener, leaving the flow dead-ended on a
            // blank bridge page (x.com login, 2026-09-15). Allow leaves the
            // request to the runtime, which opens its own popup window on
            // the same profile. A request without features (`target=_blank`,
            // plain `window.open`) still adopts as an editor tab.
            if features.size().is_some() || features.position().is_some() {
                return NewWindowResponse::Allow;
            }
            let _ = emitter.emit("btab://new", url.to_string());
            NewWindowResponse::Deny
        });
    win.add_child(page, LogicalPosition::new(x, y), LogicalSize::new(w, h))
        .map_err(|e| e.to_string())?;
    attach_navigation_gate(app, id);
    attach_download_handler(app, id);
    attach_permission_handler(app, id);
    apply_autofill(app, id);
    apply_scripts(app, id);
    Ok(())
}

// The shell-side half of the browser grant (the mirror of the daemon's
// browser.AllowsOrigin — the two must agree): once an act-capable agent has
// driven this tab, agent-caused loads must sit inside the grant. That covers
// not only the CDP navigations the daemon pre-checks but the script
// redirects it never sees; user-initiated loads are always sovereign. The
// decision table is tested in picode_shell::origins::gate.
fn attach_navigation_gate(app: &AppHandle, id: &str) {
    let Some(wv) = app.get_webview(label(id).as_str()) else {
        return;
    };
    let tab = id.to_string();
    let _ = wv.with_webview(move |platform| unsafe {
        let Ok(core) = platform.controller().CoreWebView2() else {
            return;
        };
        let handler = NavigationStartingEventHandler::create(Box::new(move |_, args| {
            let Some(args) = args else {
                return Ok(());
            };
            let uri = {
                let mut uri = windows::core::PWSTR::null();
                args.Uri(&mut uri)?;
                take_pwstr(uri)
            };
            let user = {
                let mut flag = windows::core::BOOL::default();
                args.IsUserInitiated(&mut flag)?;
                flag.as_bool()
            };
            let allowed = {
                let map = grants().lock().unwrap();
                origins::gate(map.get(&tab).map(|v| v.as_slice()), user, &uri)
            };
            if !allowed {
                args.SetCancel(true)?;
            }
            Ok(())
        }));
        let mut token: i64 = 0;
        let _ = core.add_NavigationStarting(&handler, &mut token);
    });
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
    // A toolbar/start-card navigate is the user's will, not the agent's:
    // disarm the grant; the agent's next act-tier command re-arms it.
    grants().lock().unwrap().remove(&id);
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

// Slice 2 (ADR-0128): the host-API CDP bridge. No debug port exists, so
// these two commands are the only path to a page's CDP. The daemon resolves
// an agent's tier; the catalog in picode_shell::cdppolicy re-checks every
// method before delivery, and refuses anything it does not name.
//
// The call shape is `invoke("btab_cdp_call", { id, method, paramsJson, tier })`
// and `invoke("btab_cdp_events", { id, since })`.
#[tauri::command]
pub async fn btab_cdp_call(
    app: AppHandle,
    id: String,
    method: String,
    params_json: Option<String>,
    tier: String,
    domains: Option<Vec<String>>,
) -> Result<serde_json::Value, String> {
    use webview2_com::CallDevToolsProtocolMethodCompletedHandler;
    use windows::core::HSTRING;

    let tier = cdppolicy::Tier::parse(&tier)?;
    cdppolicy::allows(tier, &method)?;
    let name = method.trim().to_string();
    let params = params_json.unwrap_or_else(|| "{}".to_string());
    if serde_json::from_str::<serde_json::Value>(&params).is_err() {
        return Err(format!("{name}: params_json is not valid JSON"));
    }
    let wv = app.get_webview(&label(&id)).ok_or("tab not open")?;
    // Arm the navigation gate: an act-capable tier carries the agent's
    // domain table (absent on legacy envelopes — leave the map alone).
    if matches!(tier, cdppolicy::Tier::Act | cdppolicy::Tier::Full) {
        if let Some(domains) = domains {
            grants().lock().unwrap().insert(id.clone(), domains);
        }
    }
    let (tx, rx) = mpsc::channel::<Result<String, String>>();
    let failed = tx.clone();
    let method_c = HSTRING::from(name.as_str());
    let params_c = HSTRING::from(params.as_str());
    let sent = wv.with_webview(move |platform| unsafe {
        let core = match platform.controller().CoreWebView2() {
            Ok(core) => core,
            Err(e) => {
                let _ = failed.send(Err(format!("CoreWebView2: {e}")));
                return;
            }
        };
        let handler = CallDevToolsProtocolMethodCompletedHandler::create(Box::new(move |hr, json| {
            let _ = tx.send(match hr {
                Ok(()) => Ok(json),
                Err(e) => Err(format!("{e}")),
            });
            Ok(())
        }));
        if let Err(e) = core.CallDevToolsProtocolMethod(&method_c, &params_c, &handler) {
            let _ = failed.send(Err(format!("{e}")));
        }
    });
    if let Err(e) = sent {
        return Err(format!("with_webview: {e}"));
    }
    let json = match rx.recv_timeout(Duration::from_secs(20)) {
        Ok(Ok(json)) => json,
        Ok(Err(e)) => return Err(format!("{name}: {e}")),
        Err(mpsc::RecvTimeoutError::Disconnected) => {
            return Err(format!("{name}: the page went away"))
        }
        Err(_) => return Err(format!("{name}: CDP call timed out")),
    };
    if json.trim().is_empty() {
        return Ok(serde_json::Value::Null);
    }
    serde_json::from_str(&json).map_err(|e| format!("{name}: CDP returned invalid JSON ({e})"))
}

// Poll one tab's recorded events. `since` is the last sequence number the
// caller saw; the reply carries the new events plus the tab's last number, so
// a caller can never mistake an overflowed ring for a quiet page.
#[tauri::command]
pub async fn btab_cdp_events(
    app: AppHandle,
    state: State<'_, BtabState>,
    id: String,
    since: Option<u64>,
) -> Result<serde_json::Value, String> {
    let subscribed = state
        .rings
        .lock()
        .unwrap()
        .get(&id)
        .map(|ring| ring.subscribed)
        .unwrap_or(false);
    if !subscribed {
        subscribe(&app, &id, state.rings.clone())?;
        state.rings.lock().unwrap().entry(id.clone()).or_default().subscribed = true;
    }
    let rings = state.rings.lock().unwrap();
    let Some(ring) = rings.get(&id) else {
        return Ok(serde_json::json!({ "events": [], "last": 0, "dropped": 0 }));
    };
    let since = since.unwrap_or(0);
    let events: Vec<serde_json::Value> = ring
        .events
        .iter()
        .filter(|e| e["seq"].as_u64().unwrap_or(0) > since)
        .cloned()
        .collect();
    Ok(serde_json::json!({ "events": events, "last": ring.seq, "dropped": ring.dropped }))
}

// Slice 2.1 (read tier seed): "Take a screenshot". Native CapturePreview
// (not CDP) — the PNG comes back base64 for the UI to save.
#[tauri::command]
// Capture the tab's visible page to a PNG file. "Take a screenshot" (saved
// to Pictures) and the menu's still (a temp file read back as bytes) both go
// through here. Data-URL downloads are blocked inside WebView2, so the PNG
// always lands on disk first.
async fn capture_png(app: &AppHandle, id: &str, path: &std::path::Path) -> Result<(), String> {
    use webview2_com::CapturePreviewCompletedHandler;
    use webview2_com::Microsoft::Web::WebView2::Win32::COREWEBVIEW2_CAPTURE_PREVIEW_IMAGE_FORMAT_PNG;
    use windows::core::HSTRING;
    use windows::Win32::Storage::FileSystem::FILE_ATTRIBUTE_NORMAL;
    use windows::Win32::System::Com::{STGM_CREATE, STGM_READWRITE, STGM_SHARE_DENY_WRITE};

    let wv = app.get_webview(&label(id)).ok_or("tab not open")?;
    let (tx, rx) = mpsc::channel::<Result<(), String>>();
    let tx_err = tx.clone();
    let shot_path = path.to_path_buf();
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
    match rx.recv_timeout(Duration::from_secs(8)) {
        Ok(Ok(())) => Ok(()),
        Ok(Err(e)) => Err(e),
        Err(_) => Err("capture timed out".into()),
    }
}

#[tauri::command]
pub async fn btab_screenshot(app: AppHandle, id: String) -> Result<String, String> {
    // The PNG goes to the user's Pictures folder and the UI shows the saved
    // path.
    let dir = std::env::var("USERPROFILE")
        .map(|home| std::path::PathBuf::from(home).join("Pictures").join("PiCode"))
        .unwrap_or_else(|_| std::env::temp_dir());
    let _ = std::fs::create_dir_all(&dir);
    let ms = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_millis())
        .unwrap_or_default();
    let path = dir.join(format!("picode-{id}-{ms}.png"));
    capture_png(&app, &id, &path).await?;
    Ok(path.to_string_lossy().to_string())
}

// The options menu opens over the page: a native WebView2 is a sibling that
// paints over HTML, so the tab hides it and shows this PNG in its place. Raw
// bytes, not a data URL — base64 would inflate them by a third across IPC,
// and Tauri hands the frontend an ArrayBuffer.
#[tauri::command]
pub async fn btab_preview(app: AppHandle, id: String) -> Result<tauri::ipc::Response, String> {
    let path = std::env::temp_dir().join(format!(
        "picode-preview-{}-{}.png",
        std::process::id(),
        id
    ));
    capture_png(&app, &id, &path).await?;
    let bytes = std::fs::read(&path).map_err(|e| format!("preview: {e}"))?;
    let _ = std::fs::remove_file(&path);
    Ok(tauri::ipc::Response::new(bytes))
}

// The runtime's own print dialog for this tab.
#[tauri::command]
pub async fn btab_print(app: AppHandle, id: String) -> Result<(), String> {
    use webview2_com::Microsoft::Web::WebView2::Win32::{
        COREWEBVIEW2_PRINT_DIALOG_KIND_BROWSER, ICoreWebView2_16,
    };
    let wv = app.get_webview(&label(&id)).ok_or("tab not open")?;
    let (tx, rx) = mpsc::channel::<Result<(), String>>();
    let done = tx.clone();
    let sent = wv.with_webview(move |platform| unsafe {
        let Ok(core) = platform.controller().CoreWebView2() else {
            let _ = done.send(Err("the page's webview is gone".into()));
            return;
        };
        let core16: ICoreWebView2_16 = match core.cast() {
            Ok(v) => v,
            Err(e) => {
                let _ = done.send(Err(format!("this WebView2 runtime cannot print ({e})")));
                return;
            }
        };
        let _ = done.send(
            core16
                .ShowPrintUI(COREWEBVIEW2_PRINT_DIALOG_KIND_BROWSER)
                .map_err(|e| format!("print: {e}")),
        );
    });
    sent.map_err(|e| format!("with_webview: {e}"))?;
    match rx.recv_timeout(Duration::from_secs(10)) {
        Ok(result) => result,
        Err(_) => Ok(()),
    }
}

// The tab's zoom factor, as the options menu shows it (1.0 = 100%).
#[tauri::command]
pub async fn btab_zoom(app: AppHandle, id: String) -> Result<f64, String> {
    let wv = app.get_webview(&label(&id)).ok_or("tab not open")?;
    let (tx, rx) = mpsc::channel::<Result<f64, String>>();
    let done = tx.clone();
    let sent = wv.with_webview(move |platform| unsafe {
        let controller = platform.controller();
        let mut z = 1.0f64;
        let _ = done.send(
            controller
                .ZoomFactor(&mut z)
                .map(|_| z)
                .map_err(|e| format!("zoom: {e}")),
        );
    });
    sent.map_err(|e| format!("with_webview: {e}"))?;
    match rx.recv_timeout(Duration::from_secs(5)) {
        Ok(result) => result,
        Err(_) => Err("zoom: no answer".into()),
    }
}

// Set the tab's zoom factor. The range matches the reference's own steps.
#[tauri::command]
pub async fn btab_set_zoom(app: AppHandle, id: String, factor: f64) -> Result<f64, String> {
    if !(0.25..=5.0).contains(&factor) {
        return Err(format!("{factor} is outside the zoom range (25%–500%)"));
    }
    let wv = app.get_webview(&label(&id)).ok_or("tab not open")?;
    let (tx, rx) = mpsc::channel::<Result<f64, String>>();
    let done = tx.clone();
    let sent = wv.with_webview(move |platform| unsafe {
        let controller = platform.controller();
        let _ = done.send(
            controller
                .SetZoomFactor(factor)
                .map(|_| factor)
                .map_err(|e| format!("zoom: {e}")),
        );
    });
    sent.map_err(|e| format!("with_webview: {e}"))?;
    match rx.recv_timeout(Duration::from_secs(5)) {
        Ok(result) => result,
        Err(_) => Err("zoom: no answer".into()),
    }
}

// Find in page through the runtime's find session. `forward` = None starts a
// fresh search (the input changed), Some(true|false) steps next/previous, and
// an empty query stops the session and clears the highlights. Returns the
// match state the menu shows as "2/7".
#[tauri::command]
pub async fn btab_find(
    app: AppHandle,
    id: String,
    query: String,
    forward: Option<bool>,
) -> Result<serde_json::Value, String> {
    use webview2_com::FindStartCompletedHandler;
    use webview2_com::Microsoft::Web::WebView2::Win32::{
        ICoreWebView2Find, ICoreWebView2FindOptions, ICoreWebView2_28,
    };
    use windows::core::HSTRING;

    let wv = app.get_webview(&label(&id)).ok_or("tab not open")?;
    let query = query.trim().to_string();
    let fresh = forward.is_none() && !query.is_empty();
    let (tx, rx) = mpsc::channel::<Result<(), String>>();
    let done = tx.clone();
    let (start_tx, start_rx) = mpsc::channel::<Result<(), String>>();
    let sent = wv.with_webview(move |platform| unsafe {
        let Ok(core) = platform.controller().CoreWebView2() else {
            let _ = done.send(Err("the page's webview is gone".into()));
            return;
        };
        let core28: ICoreWebView2_28 = match core.cast() {
            Ok(v) => v,
            Err(e) => {
                let _ = done.send(Err(format!("this WebView2 runtime has no Find API ({e})")));
                return;
            }
        };
        let find: ICoreWebView2Find = match core28.Find() {
            Ok(v) => v,
            Err(e) => {
                let _ = done.send(Err(format!("find: {e}")));
                return;
            }
        };
        let result = if query.is_empty() {
            find.Stop().map_err(|e| format!("find: {e}"))
        } else if let Some(forward) = forward {
            (if forward { find.FindNext() } else { find.FindPrevious() })
                .map_err(|e| format!("find: {e}"))
        } else {
            let options: ICoreWebView2FindOptions = match core28.CreateFindOptions() {
                Ok(v) => v,
                Err(e) => {
                    let _ = done.send(Err(format!("find: {e}")));
                    return;
                }
            };
            let _ = options.SetFindTerm(&HSTRING::from(query.as_str()));
            let _ = options.SetShouldHighlightAllMatches(true);
            let handler = FindStartCompletedHandler::create(Box::new(move |hr| {
                let _ = start_tx.send(hr.map_err(|e| format!("find: {e}")));
                Ok(())
            }));
            find.Start(&options, &handler)
                .map_err(|e| format!("find: {e}"))
        };
        let _ = done.send(result);
    });
    sent.map_err(|e| format!("with_webview: {e}"))?;
    match rx.recv_timeout(Duration::from_secs(5)) {
        Ok(Ok(())) => {}
        Ok(Err(e)) => return Err(e),
        Err(_) => return Err("find: no answer".into()),
    }
    if fresh {
        match start_rx.recv_timeout(Duration::from_secs(3)) {
            Ok(Ok(())) => {}
            Ok(Err(e)) => return Err(e),
            Err(_) => return Err("find: the search did not finish".into()),
        }
    }
    if forward.is_some() {
        // The match counters update on the runtime's own schedule; a short
        // beat is enough for the menu's read.
        std::thread::sleep(Duration::from_millis(80));
    }
    // The COM objects are thread-bound, so the state is read in a second
    // pass on the UI thread — this command already runs off it.
    let (tx2, rx2) = mpsc::channel::<Result<serde_json::Value, String>>();
    let done2 = tx2.clone();
    let sent2 = wv.with_webview(move |platform| unsafe {
        let Ok(core) = platform.controller().CoreWebView2() else {
            let _ = done2.send(Err("the page's webview is gone".into()));
            return;
        };
        let Ok(core28) = core.cast::<ICoreWebView2_28>() else {
            let _ = done2.send(Ok(serde_json::json!({ "count": 0, "active": 0 })));
            return;
        };
        let Ok(find) = core28.Find() else {
            let _ = done2.send(Ok(serde_json::json!({ "count": 0, "active": 0 })));
            return;
        };
        let mut count = 0i32;
        let mut active = 0i32;
        let _ = find.MatchCount(&mut count);
        let _ = find.ActiveMatchIndex(&mut active);
        let _ = done2.send(Ok(serde_json::json!({ "count": count, "active": active })));
    });
    sent2.map_err(|e| format!("with_webview: {e}"))?;
    match rx2.recv_timeout(Duration::from_secs(3)) {
        Ok(result) => result,
        Err(_) => Ok(serde_json::json!({ "count": 0, "active": 0 })),
    }
}

#[tauri::command]
pub async fn btab_close(
    app: AppHandle,
    state: State<'_, BtabState>,
    id: String,
) -> Result<(), String> {
    if let Some(wv) = app.get_webview(&label(&id)) {
        // Drop this tab's event receivers on the UI thread before the webview
        // goes: a receiver kept past its page would keep the page alive.
        let tab = id.clone();
        let _ = wv.with_webview(move |_| {
            RECEIVERS.with(|r| {
                r.borrow_mut().remove(&tab);
            });
        });
        wv.close().map_err(|e| e.to_string())?;
    }
    state.pending.lock().unwrap().remove(&id);
    state.rings.lock().unwrap().remove(&id);
    grants().lock().unwrap().remove(&id);
    Ok(())
}

// Open a URL in the system default browser — the "Default browser" side of
// the web/local open-destination prefs (slice 3). CREATE_NO_WINDOW keeps
// cmd's console from flashing; the URL is scheme-checked here so nothing
// else can ride this command into an arbitrary shell verb.
#[tauri::command]
pub async fn btab_open_external(url: String) -> Result<(), String> {
    let ok = url.starts_with("http://") || url.starts_with("https://");
    if !ok {
        return Err("only http and https URLs can be handed to the system browser".into());
    }
    use std::os::windows::process::CommandExt;
    std::process::Command::new("cmd")
        .args(["/C", "start", "", &url])
        .creation_flags(0x0800_0000) // CREATE_NO_WINDOW
        .spawn()
        .map(|_| ())
        .map_err(|e| format!("open external: {e}"))
}

// The work profile's autofill prefs (slice 3): password autosave and
// general (contact info) autofill. The UI pushes them (btab_set_prefs);
// the latest values are applied to every webview at creation. Defaults
// match WebView2's own: both on.
fn autofill() -> &'static Mutex<(bool, bool)> {
    static AUTOFILL: OnceLock<Mutex<(bool, bool)>> = OnceLock::new();
    AUTOFILL.get_or_init(|| Mutex::new((true, true)))
}

#[tauri::command]
pub async fn btab_set_prefs(
    app: AppHandle,
    password_autosave: Option<bool>,
    general_autofill: Option<bool>,
) -> Result<(), String> {
    {
        let mut cur = autofill().lock().unwrap();
        if let Some(v) = password_autosave {
            cur.0 = v;
        }
        if let Some(v) = general_autofill {
            cur.1 = v;
        }
    }
    let (pw, gen) = *autofill().lock().unwrap();
    for (name, wv) in app.webview_windows() {
        if !name.starts_with("btab-") {
            continue;
        }
        let _ = wv.with_webview(move |platform| unsafe {
            let Ok(core) = platform.controller().CoreWebView2() else { return };
            let Ok(settings) = core.Settings() else { return };
            let s9: webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2Settings9 = match settings.cast() { Ok(v) => v, Err(_) => return };
            let _ = s9.SetIsPasswordAutosaveEnabled(pw);
            let _ = s9.SetIsGeneralAutofillEnabled(gen);
        });
    }
    Ok(())
}

// Apply the current autofill prefs to a freshly created webview (called
// from ensure's attach step).
fn apply_autofill(app: &AppHandle, id: &str) {
    let Some(wv) = app.get_webview(label(id).as_str()) else { return };
    let _ = wv.with_webview(move |platform| unsafe {
        let Ok(core) = platform.controller().CoreWebView2() else { return };
        let Ok(settings) = core.Settings() else { return };
        let s9: webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2Settings9 = match settings.cast() { Ok(v) => v, Err(_) => return };
        let (pw, gen) = *autofill().lock().unwrap();
        let _ = s9.SetIsPasswordAutosaveEnabled(pw);
        let _ = s9.SetIsGeneralAutofillEnabled(gen);
    });
}

// Clear the work profile's browsing data (slice 3): the Clear browsing
// data dialog sends the kinds it checked and, for a time-range pick, how
// far back since is (unix seconds; None means all time). "passwords" is
// deliberately absent from that dialog's checklist — the Password manager
// dialog is the one place that clears them.
fn wipe_mask(kind: &str) -> i32 {
    // ICoreWebView2BrowsingDataKinds values (bindings 0.38.2).
    const FILE_SYSTEMS: i32 = 1;
    const INDEXED_DB: i32 = 2;
    const LOCAL_STORAGE: i32 = 4;
    const WEB_SQL: i32 = 8;
    const CACHE_STORAGE: i32 = 16;
    const ALL_DOM_STORAGE: i32 = 32;
    const COOKIES: i32 = 64;
    const ALL_SITE: i32 = 128;
    const DISK_CACHE: i32 = 256;
    const DOWNLOAD_HISTORY: i32 = 512;
    const GENERAL_AUTOFILL: i32 = 1024;
    const PASSWORD_AUTOSAVE: i32 = 2048;
    const BROWSING_HISTORY: i32 = 4096;
    const SETTINGS: i32 = 8192;
    const SERVICE_WORKERS: i32 = 32768;
    match kind {
        "history" => BROWSING_HISTORY,
        "cookies" => FILE_SYSTEMS | INDEXED_DB | LOCAL_STORAGE | WEB_SQL | ALL_DOM_STORAGE | COOKIES | ALL_SITE | SERVICE_WORKERS,
        "cache" => CACHE_STORAGE | DISK_CACHE,
        "downloads" => DOWNLOAD_HISTORY,
        "autofill" => GENERAL_AUTOFILL,
        "passwords" => PASSWORD_AUTOSAVE,
        "siteSettings" => SETTINGS,
        _ => 0,
    }
}

#[tauri::command]
pub async fn btab_clear_data(
    app: AppHandle,
    kinds: Vec<String>,
    since: Option<f64>,
) -> Result<(), String> {
    let mask = kinds.iter().fold(0, |acc, k| acc | wipe_mask(k));
    if mask == 0 {
        return Err("nothing was selected to clear".into());
    }
    let (_, wv) = app
        .webview_windows()
        .into_iter()
        .find(|(name, _)| name.starts_with("btab-"))
        .ok_or("no browser tab is open")?;
    let (tx, rx) = mpsc::channel::<Result<(), String>>();
    let done = tx.clone();
    let sent = wv.with_webview(move |platform| unsafe {
        let Ok(core) = platform.controller().CoreWebView2() else {
            let _ = done.send(Err("the page's webview is gone".into()));
            return;
        };
        let core13: webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2_13 = match core.cast() {
            Ok(v) => v,
            Err(e) => {
                let _ = done.send(Err(format!("{e}")));
                return;
            }
        };
        let Ok(profile) = core13.Profile() else {
            let _ = done.send(Err("profile unavailable".into()));
            return;
        };
        let p2: webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2Profile2 = match profile.cast() {
            Ok(v) => v,
            Err(e) => {
                let _ = done.send(Err(format!("{e}")));
                return;
            }
        };
        let mask = webview2_com::Microsoft::Web::WebView2::Win32::COREWEBVIEW2_BROWSING_DATA_KINDS(mask);
        let handler_done = done.clone();
        let handler = webview2_com::ClearBrowsingDataCompletedHandler::create(Box::new(move |hr| {
            let _ = match hr {
                Ok(()) => handler_done.send(Ok(())),
                Err(e) => handler_done.send(Err(format!("{e}"))),
            };
            Ok(())
        }));
        let result = match since {
            // A range: from `since` to now, in unix seconds.
            Some(start) => {
                let end = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .map(|d| d.as_secs_f64())
                    .unwrap_or(start);
                p2.ClearBrowsingDataInTimeRange(mask, start, end, &handler)
            }
            None => p2.ClearBrowsingData(mask, &handler),
        };
        if let Err(e) = result {
            let _ = done.send(Err(format!("{e}")));
        }
    });
    sent.map_err(|e| format!("with_webview: {e}"))?;
    match rx.recv_timeout(Duration::from_secs(30)) {
        Ok(Ok(())) => Ok(()),
        Ok(Err(e)) => Err(e),
        Err(_) => Err("clearing browsing data timed out".into()),
    }
}

// --- downloads (slice 3.3d) -------------------------------------------------

// Ask where to save: false writes the file straight into the profile's
// download folder without the runtime's own UI; true leaves the runtime UI
// in place, which is where a save prompt can appear. Default matches the
// reference and the platform: off.
fn ask_where() -> &'static Mutex<bool> {
    static ASK: OnceLock<Mutex<bool>> = OnceLock::new();
    ASK.get_or_init(|| Mutex::new(false))
}

// The profile's download folder. An empty string is the platform's answer
// for "the system Downloads folder", which is exactly what the settings row
// shows in that case.
#[tauri::command]
pub async fn btab_download_dir(app: AppHandle) -> Result<String, String> {
    let (_, wv) = app
        .webview_windows()
        .into_iter()
        .find(|(name, _)| name.starts_with("btab-"))
        .ok_or("no browser tab is open")?;
    let (tx, rx) = mpsc::channel::<Result<String, String>>();
    let done = tx.clone();
    let sent = wv.with_webview(move |platform| unsafe {
        let Ok(core) = platform.controller().CoreWebView2() else {
            let _ = done.send(Err("the page's webview is gone".into()));
            return;
        };
        let core13: webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2_13 = match core.cast() {
            Ok(v) => v,
            Err(e) => {
                let _ = done.send(Err(format!("{e}")));
                return;
            }
        };
        let Ok(profile) = core13.Profile() else {
            let _ = done.send(Err("profile unavailable".into()));
            return;
        };
        let mut value = windows::core::PWSTR::null();
        match profile.DefaultDownloadFolderPath(&mut value) {
            Ok(()) => {
                let _ = done.send(Ok(take_pwstr(value)));
            }
            Err(e) => {
                let _ = done.send(Err(format!("{e}")));
            }
        }
    });
    sent.map_err(|e| format!("with_webview: {e}"))?;
    match rx.recv_timeout(Duration::from_secs(10)) {
        Ok(Ok(path)) => Ok(path),
        Ok(Err(e)) => Err(e),
        Err(_) => Err("reading the download folder timed out".into()),
    }
}

// Point the shared profile at another folder. An empty path hands it back to
// the platform default. Applied to every live tab and remembered for the
// ones created later.
#[tauri::command]
pub async fn btab_set_download_dir(app: AppHandle, path: String) -> Result<(), String> {
    let wide: Vec<u16> = path.encode_utf16().chain(std::iter::once(0)).collect();
    for (name, wv) in app.webview_windows() {
        if !name.starts_with("btab-") {
            continue;
        }
        let wide = wide.clone();
        let _ = wv.with_webview(move |platform| unsafe {
            let Ok(core) = platform.controller().CoreWebView2() else {
                return;
            };
            let Ok(core13) = core.cast::<webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2_13>() else {
                return;
            };
            let Ok(profile) = core13.Profile() else {
                return;
            };
            let _ = profile.SetDefaultDownloadFolderPath(windows::core::PCWSTR(wide.as_ptr()));
        });
    }
    Ok(())
}

#[tauri::command]
pub async fn btab_set_ask_download(ask: bool) -> Result<(), String> {
    *ask_where().lock().unwrap() = ask;
    Ok(())
}

// Hand a downloaded file to the system (the row menu's Open), and show it in
// Explorer (Show in folder). Both refuse anything that is not an absolute
// path, so a stray string cannot turn into a command.
fn check_path(path: &str) -> Result<(), String> {
    let looks_absolute = path.starts_with("\\\\") || (path.len() > 2 && path.as_bytes()[1] == b':');
    if !looks_absolute || path.contains('"') {
        return Err("that is not an absolute Windows path".into());
    }
    Ok(())
}

#[tauri::command]
pub async fn btab_open_path(path: String) -> Result<(), String> {
    check_path(&path)?;
    use std::os::windows::process::CommandExt;
    std::process::Command::new("cmd")
        .args(["/C", "start", "", &path])
        .creation_flags(0x0800_0000) // CREATE_NO_WINDOW
        .spawn()
        .map(|_| ())
        .map_err(|e| format!("open path: {e}"))
}

#[tauri::command]
pub async fn btab_reveal_path(path: String) -> Result<(), String> {
    check_path(&path)?;
    use std::os::windows::process::CommandExt;
    std::process::Command::new("explorer")
        .arg(format!("/select,{path}"))
        .creation_flags(0x0800_0000) // CREATE_NO_WINDOW
        .spawn()
        .map(|_| ())
        .map_err(|e| format!("reveal path: {e}"))
}

// Every tab reports its downloads: the start (source, destination, expected
// size) and the outcome, so Settings ▸ Browser ▸ Download history can show
// what happened. The UI receives "btab://download" and writes the row
// through the daemon (the shell never talks to the store itself).
fn attach_download_handler(app: &AppHandle, id: &str) {
    let Some(wv) = app.get_webview(label(id).as_str()) else {
        return;
    };
    let emitter = app.clone();
    let _ = wv.with_webview(move |platform| unsafe {
        let Ok(core) = platform.controller().CoreWebView2() else {
            return;
        };
        // DownloadStarting lives on the _4 interface, not the base one.
        let Ok(core4) = core.cast::<webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2_4>() else {
            return;
        };
        let started = emitter.clone();
        let handler = DownloadStartingEventHandler::create(Box::new(move |_, args| {
            let Some(args) = args else {
                return Ok(());
            };
            let ask = *ask_where().lock().unwrap();
            // Silent save into the profile's folder unless the user asked to
            // be asked; with the runtime UI left in place the download can be
            // renamed or moved before it starts.
            let _ = args.SetHandled(!ask);
            let Ok(op) = args.DownloadOperation() else {
                return Ok(());
            };
            let url = {
                let mut value = windows::core::PWSTR::null();
                if op.Uri(&mut value).is_ok() {
                    take_pwstr(value)
                } else {
                    String::new()
                }
            };
            let path = {
                let mut value = windows::core::PWSTR::null();
                if op.ResultFilePath(&mut value).is_ok() {
                    take_pwstr(value)
                } else {
                    String::new()
                }
            };
            let mut total: i64 = 0;
            let _ = op.TotalBytesToReceive(&mut total);
            let _ = started.emit(
                "btab://download",
                serde_json::json!({
                    "status": "started",
                    "url": url,
                    "path": path,
                    "total": total,
                }),
            );
            // The outcome rides the same event, matched by destination path.
            let done = started.clone();
            let state_handler = StateChangedEventHandler::create(Box::new(move |op, _| {
                let Some(op) = op else {
                    return Ok(());
                };
                let mut state = COREWEBVIEW2_DOWNLOAD_STATE(0);
                if op.State(&mut state).is_err() {
                    return Ok(());
                }
                let status = if state.0 == COREWEBVIEW2_DOWNLOAD_STATE_COMPLETED.0 {
                    "completed"
                } else if state.0 == COREWEBVIEW2_DOWNLOAD_STATE_INTERRUPTED.0 {
                    "interrupted"
                } else {
                    return Ok(());
                };
                let path = {
                    let mut value = windows::core::PWSTR::null();
                    if op.ResultFilePath(&mut value).is_ok() {
                        take_pwstr(value)
                    } else {
                        String::new()
                    }
                };
                let mut received: i64 = 0;
                let _ = op.BytesReceived(&mut received);
                let _ = done.emit(
                    "btab://download",
                    serde_json::json!({
                        "status": status,
                        "path": path,
                        "received": received,
                    }),
                );
                Ok(())
            }));
            let mut token: i64 = 0;
            let _ = op.add_StateChanged(&state_handler, &mut token);
            Ok(())
        }));
        let mut token: i64 = 0;
        let _ = core4.add_DownloadStarting(&handler, &mut token);
    });
}

// --- site permissions (slice 3, Browser permissions) ------------------------

// The policy the user set per kind, in the Settings dialog: kind name →
// allow. A kind with no entry follows the platform's own default (which is
// to deny). The prompt ("ask") lands with the Site settings dialog; until
// then a request without a policy is answered the way an unhandled request
// always was, and the outcome is reported so the dialog can show it.
fn permission_policy() -> &'static Mutex<std::collections::HashMap<String, bool>> {
    static POLICY: OnceLock<Mutex<std::collections::HashMap<String, bool>>> = OnceLock::new();
    POLICY.get_or_init(|| Mutex::new(std::collections::HashMap::new()))
}

// The platform's kind as the daemon names it (the store's closed list).
fn permission_kind_name(kind: i32) -> &'static str {
    match kind {
        x if x == COREWEBVIEW2_PERMISSION_KIND_CAMERA.0 => "camera",
        x if x == COREWEBVIEW2_PERMISSION_KIND_MICROPHONE.0 => "microphone",
        x if x == COREWEBVIEW2_PERMISSION_KIND_GEOLOCATION.0 => "location",
        x if x == COREWEBVIEW2_PERMISSION_KIND_NOTIFICATIONS.0 => "notifications",
        x if x == COREWEBVIEW2_PERMISSION_KIND_CLIPBOARD_READ.0 => "clipboard",
        x if x == COREWEBVIEW2_PERMISSION_KIND_AUTOPLAY.0 => "autoplay",
        x if x == COREWEBVIEW2_PERMISSION_KIND_OTHER_SENSORS.0 => "sensors",
        x if x == COREWEBVIEW2_PERMISSION_KIND_MIDI_SYSTEM_EXCLUSIVE_MESSAGES.0 => "midi",
        x if x == COREWEBVIEW2_PERMISSION_KIND_LOCAL_FONTS.0 => "fonts",
        x if x == COREWEBVIEW2_PERMISSION_KIND_FILE_READ_WRITE.0 => "filesystem",
        _ => "unknown",
    }
}

// What the user decided for a kind: allow, deny, or the platform default.
#[tauri::command]
pub async fn btab_set_permission_policy(kind: String, state: String) -> Result<(), String> {
    let kind = kind.trim().to_lowercase();
    if kind.is_empty() {
        return Err("a permission kind is required".into());
    }
    let mut policy = permission_policy().lock().unwrap();
    match state.trim().to_lowercase().as_str() {
        "allow" => {
            policy.insert(kind, true);
        }
        "deny" => {
            policy.insert(kind, false);
        }
        "default" => {
            policy.remove(&kind);
        }
        other => return Err(format!("{other:?} is not a permission state")),
    }
    Ok(())
}

// Every tab answers permission requests from that policy and reports the
// outcome, so the Site settings dialog can list what each site got.
fn attach_permission_handler(app: &AppHandle, id: &str) {
    let Some(wv) = app.get_webview(label(id).as_str()) else {
        return;
    };
    let emitter = app.clone();
    let _ = wv.with_webview(move |platform| unsafe {
        let Ok(core) = platform.controller().CoreWebView2() else {
            return;
        };
        let handler = PermissionRequestedEventHandler::create(Box::new(move |_, args| {
            let Some(args) = args else {
                return Ok(());
            };
            let mut kind = COREWEBVIEW2_PERMISSION_KIND(0);
            let _ = args.PermissionKind(&mut kind);
            let name = permission_kind_name(kind.0);
            let origin = {
                let mut value = windows::core::PWSTR::null();
                if args.Uri(&mut value).is_ok() {
                    take_pwstr(value)
                } else {
                    String::new()
                }
            };
            let decided = permission_policy().lock().unwrap().get(name).copied();
            let allow = match decided {
                Some(v) => {
                    let _ = args.SetState(if v {
                        COREWEBVIEW2_PERMISSION_STATE_ALLOW
                    } else {
                        COREWEBVIEW2_PERMISSION_STATE_DENY
                    });
                    v
                }
                // No policy: the request falls through to the platform's own
                // default, which denies. Reported so it is visible.
                None => false,
            };
            let _ = emitter.emit(
                "btab://permission",
                serde_json::json!({
                    "origin": origin,
                    "kind": name,
                    "decision": if allow { "allow" } else { "deny" },
                }),
            );
            Ok(())
        }));
        let mut token: i64 = 0;
        let _ = core.add_PermissionRequested(&handler, &mut token);
    });
}

// --- JavaScript (slice 3, Browser permissions) ------------------------------

// Sites may use JavaScript: applied to every tab at creation (next to the
// autofill settings) and to the live ones when the switch moves. On by
// default, like the platform.
fn scripts_enabled() -> &'static Mutex<bool> {
    static SCRIPTS: OnceLock<Mutex<bool>> = OnceLock::new();
    SCRIPTS.get_or_init(|| Mutex::new(true))
}

#[tauri::command]
pub async fn btab_set_scripts(app: AppHandle, enabled: bool) -> Result<(), String> {
    *scripts_enabled().lock().unwrap() = enabled;
    for (name, wv) in app.webview_windows() {
        if !name.starts_with("btab-") {
            continue;
        }
        let _ = wv.with_webview(move |platform| unsafe {
            let Ok(core) = platform.controller().CoreWebView2() else { return };
            if let Ok(settings) = core.Settings() {
                let _ = settings.SetIsScriptEnabled(enabled);
            }
        });
    }
    Ok(())
}

// The creation-time half of the switch (the live half is btab_set_scripts):
// a new tab is born with the setting in force.
fn apply_scripts(app: &AppHandle, id: &str) {
    let Some(wv) = app.get_webview(label(id).as_str()) else {
        return;
    };
    let enabled = *scripts_enabled().lock().unwrap();
    let _ = wv.with_webview(move |platform| unsafe {
        let Ok(core) = platform.controller().CoreWebView2() else {
            return;
        };
        if let Ok(settings) = core.Settings() {
            let _ = settings.SetIsScriptEnabled(enabled);
        }
    });
}
