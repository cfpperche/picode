// btab.rs — work browser tabs (Phase 3 slice 1, docs/plans/desktop-v2.md):
// each browser page is a child WebView2 of the main window, living behind an
// editor tab ("w:<id>"). The React side renders the toolbar and reports the
// viewport rect of the page region (ResizeObserver); this module positions
// the native webview there. ADR-0128: one shared profile, no debug port in
// this path, popups adopt as new tabs.

use std::cell::RefCell;
use std::collections::{HashMap, VecDeque};
use std::sync::{mpsc, Arc, Mutex, OnceLock};
use std::time::Duration;

use picode_shell::{cdppolicy, origins};
use windows::core::Interface;
use tauri::webview::{NewWindowResponse, WebviewBuilder};
use tauri::WebviewUrl;
use tauri::{AppHandle, Emitter, LogicalPosition, LogicalSize, Manager, State};
use webview2_com::{take_pwstr, NavigationStartingEventHandler};

/// The origin grant per webview id, armed by the last act-capable CDP call
/// the daemon relayed (tier act/full carries the agent's domain table).
/// The navigation gate attached in `ensure` reads it; a user-driven
/// `btab_navigate` disarms it (the user is sovereign); `btab_close` drops
/// the entry.
fn grants() -> &'static Mutex<HashMap<String, Vec<String>>> {
    static GRANTS: OnceLock<Mutex<HashMap<String, Vec<String>>>> = OnceLock::new();
    GRANTS.get_or_init(|| Mutex::new(HashMap::new()))
}
use webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2DevToolsProtocolEventReceiver;

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
        .on_new_window(move |url, _features| {
            // Popups adopt as new editor tabs (the UI listens on this event).
            let _ = emitter.emit("btab://new", url.to_string());
            NewWindowResponse::Deny
        });
    win.add_child(page, LogicalPosition::new(x, y), LogicalSize::new(w, h))
        .map_err(|e| e.to_string())?;
    attach_navigation_gate(app, id);
    apply_autofill(app, id);
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
