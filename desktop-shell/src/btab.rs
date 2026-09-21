// btab.rs — work browser tabs (Phase 3 slice 1, docs/plans/desktop-v2.md):
// each browser page is a child WebView2 of the main window, living behind an
// editor tab ("w:<id>"). The React side renders the toolbar and reports the
// viewport rect of the page region (ResizeObserver); this module positions
// the native webview there. ADR-0128: one shared profile, no debug port in
// this path, `target=_blank` and unsized `window.open` adopt as new tabs — a
// sized popup opens as a real window (OAuth needs `window.opener`).

use std::cell::RefCell;
use std::collections::{HashMap, HashSet, VecDeque};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{mpsc, Arc, Mutex, OnceLock};
use std::time::Duration;

use picode_shell::{cdppolicy, origins, permissions, preview, responsive};
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
    ICoreWebView2Deferral, ICoreWebView2DevToolsProtocolEventReceiver,
    ICoreWebView2PermissionRequestedEventArgs, COREWEBVIEW2_DOWNLOAD_STATE,
    COREWEBVIEW2_DOWNLOAD_STATE_COMPLETED, COREWEBVIEW2_DOWNLOAD_STATE_INTERRUPTED,
    COREWEBVIEW2_PERMISSION_KIND, COREWEBVIEW2_PERMISSION_KIND_AUTOPLAY,
    COREWEBVIEW2_PERMISSION_KIND_CAMERA, COREWEBVIEW2_PERMISSION_KIND_CLIPBOARD_READ,
    COREWEBVIEW2_PERMISSION_KIND_FILE_READ_WRITE, COREWEBVIEW2_PERMISSION_KIND_GEOLOCATION,
    COREWEBVIEW2_PERMISSION_KIND_LOCAL_FONTS, COREWEBVIEW2_PERMISSION_KIND_MICROPHONE,
    COREWEBVIEW2_PERMISSION_KIND_MIDI_SYSTEM_EXCLUSIVE_MESSAGES,
    COREWEBVIEW2_PERMISSION_KIND_NOTIFICATIONS, COREWEBVIEW2_PERMISSION_KIND_OTHER_SENSORS,
    COREWEBVIEW2_PERMISSION_STATE_ALLOW, COREWEBVIEW2_PERMISSION_STATE_DEFAULT,
    COREWEBVIEW2_PERMISSION_STATE_DENY,
};

#[derive(Default)]
pub struct BtabState {
    // Bounds the UI reported before the webview existed (first navigate).
    pub pending: Mutex<HashMap<String, (f64, f64, f64, f64)>>,
    // Tabs created without a reported rect: born hidden, so a page can never
    // paint at the placeholder rect over the pane's own empty state until the
    // first bounds call places it (owner report 2026-09-17).
    pub unplaced: Mutex<HashSet<String>>,
    // The full pane rect each tab last reported through btab_bounds — the
    // raw one, before any responsive narrowing. The device toolbar's width
    // command centers the page inside it; reset hands the whole rect back.
    pub panes: Mutex<HashMap<String, (f64, f64, f64, f64)>>,
    // Per-tab device-toolbar state ("Responsive width", the native half):
    // the preset width the page is narrowed to and the zoom that rides with
    // it. Rides btab_meta back to the UI, so the strip survives tab switches
    // and route returns; an entry exists only while the toolbar is shown —
    // hiding it resets the tab (a hidden toolbar must not leave a narrowed
    // page behind with no control to fix it).
    pub responsive: Mutex<HashMap<String, responsive::Width>>,
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

    let wv = find_webview(app, id).ok_or("tab not open")?;
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

// find_webview resolves a tab id to its native webview. The app's own tab id
// may carry a prefix the shell never saw (`w:2` vs `2`) because `ensure` names
// the webview from the id it was given — so the id's tail is tried too. Every
// command that needs "the webview of this tab" goes through here: a caller
// that passed the prefixed id used to miss it and fail silently in a
// `.catch` (the annotation crop, 2026-09-19).
fn find_webview(app: &AppHandle, id: &str) -> Option<tauri::Webview> {
    let tail = id.rsplit(':').next().unwrap_or(id);
    app.get_webview(&label(id))
        .or_else(|| app.get_webview(&label(tail)))
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
    let placed = if let Some(b) = app.state::<BtabState>().pending.lock().unwrap().remove(id) {
        (x, y, w, h) = b;
        true
    } else {
        // No host rect yet. The decision table for the first frame:
        //
        //   rect reported before creation  → created at it, visible
        //   no rect yet                    → created hidden at the placeholder,
        //                                    shown by the first btab_bounds
        //   an explicit btab_visibility    → wins over that (the flag drops)
        //   tab closed before any of them  → nothing to show
        false
    };
    // A tab recreated under an active device-toolbar width is born at the
    // narrowed rect — the per-tab state survives the close of the webview,
    // and the first bounds push alone would flash the page full-width.
    if placed {
        let st = app
            .state::<BtabState>()
            .responsive
            .lock()
            .unwrap()
            .get(id)
            .copied()
            .unwrap_or_default();
        if st.narrowed() {
            let r = responsive::width_rect(responsive::Rect::new(x, y, w, h), st.w);
            (x, y, w, h) = (r.x, r.y, r.w, r.h);
        }
    }
    let emitter = app.clone();
    let page = WebviewBuilder::new(label, WebviewUrl::External(parsed))
        .data_directory(super::browserlab::profile_for(id))
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
    let child = win
        .add_child(page, LogicalPosition::new(x, y), LogicalSize::new(w, h))
        .map_err(|e| e.to_string())?;
    if !placed {
        let _ = child.hide();
        app.state::<BtabState>()
            .unplaced
            .lock()
            .unwrap()
            .insert(id.to_string());
    }
    crate::layers::raise(app);
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
    // The raw pane rect is remembered either way: the device toolbar centers
    // the page inside it, and reset hands the whole rect back.
    state.panes.lock().unwrap().insert(id.clone(), (x, y, w, h));
    let (x, y, w, h) = {
        let st = state
            .responsive
            .lock()
            .unwrap()
            .get(&id)
            .copied()
            .unwrap_or_default();
        if st.narrowed() {
            let r = responsive::width_rect(responsive::Rect::new(x, y, w, h), st.w);
            (r.x, r.y, r.w, r.h)
        } else {
            (x, y, w, h)
        }
    };
    match app.get_webview(&label(&id)) {
        Some(wv) => {
            wv.set_bounds(tauri::Rect {
                position: LogicalPosition::new(x, y).into(),
                size: LogicalSize::new(w, h).into(),
            })
            .map_err(|e| e.to_string())?;
            // The first placement of a tab that was born hidden: this is what
            // makes the page appear, now that it is where it belongs.
            if app.state::<BtabState>().unplaced.lock().unwrap().remove(&id) {
                let _ = wv.show();
                crate::layers::raise(&app);
            }
            Ok(())
        }
        None => {
            // No webview yet — remember where it goes for the first navigate.
            state.pending.lock().unwrap().insert(id, (x, y, w, h));
            Ok(())
        }
    }
}

#[tauri::command]
pub async fn btab_visibility(app: AppHandle, id: String, visible: bool) -> Result<(), String> {
    // An explicit call is the UI's will: it also settles a tab that was born
    // waiting for its first placement, so bounds never shows it behind the
    // caller's back afterwards.
    app.state::<BtabState>().unplaced.lock().unwrap().remove(&id);
    match app.get_webview(&label(&id)) {
        Some(wv) => {
            if visible {
                wv.show().map_err(|e| e.to_string())?;
                crate::layers::raise(&app);
                Ok(())
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

// The active tab polls this; title is the host for now (v1). The receipt
// also carries the tab's device-toolbar state (`responsive`), which is what
// makes the strip survive tab switches and route returns — the UI paints
// from this, never from its own memory of the last click.
#[tauri::command]
pub async fn btab_meta(
    app: AppHandle,
    state: State<'_, BtabState>,
    id: String,
) -> Result<serde_json::Value, String> {
    let responsive = state.responsive.lock().unwrap().get(&id).map(|st| {
        serde_json::json!({ "on": true, "width": st.w, "zoom": st.zoom })
    });
    match app.get_webview(&label(&id)) {
        Some(wv) => {
            let url = wv.url().map(|u| u.to_string()).unwrap_or_default();
            let title = tauri::Url::parse(&url)
                .ok()
                .and_then(|u| u.host_str().map(|s| s.to_string()))
                .unwrap_or_default();
            Ok(serde_json::json!({ "url": url, "title": title, "responsive": responsive }))
        }
        None => Ok(serde_json::json!({ "url": "", "title": "", "responsive": responsive })),
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
    raw: Option<bool>,
) -> Result<serde_json::Value, String> {
    use webview2_com::CallDevToolsProtocolMethodCompletedHandler;
    use windows::core::HSTRING;

    let tier = cdppolicy::Tier::parse(&tier)?;
    // Two doors, and only two. The catalog (ADR-0128) is the narrow one; a
    // raw call (ADR-0144) is the one the owner opened on this machine, and it
    // needs the full tier here as well — the daemon checked both, and neither
    // process grants what the other refuses.
    if raw.unwrap_or(false) {
        if !*developer_mode().lock().unwrap() {
            return Err(format!(
                "{method}: raw CDP is off — turn on Developer mode in Settings ▸ Browser"
            ));
        }
        if tier != cdppolicy::Tier::Full {
            return Err(format!(
                "{method}: raw CDP needs the full tier — this command carried {}",
                tier.name()
            ));
        }
    } else {
        cdppolicy::allows(tier, &method)?;
    };
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
    let _ = wv.with_webview(move |platform| unsafe {
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
    // The decision half (which answer, which silence means what) is
    // preview.rs's, table-tested on any host.
    match preview::settle(rx, Duration::from_secs(8)) {
        preview::Settled::Painted => Ok(()),
        preview::Settled::Failed(e) => Err(e),
        preview::Settled::TimedOut => Err("capture timed out".into()),
        preview::Settled::Gone => Err("capture: the tab closed while the capture was in flight".into()),
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
    // Read-back + the non-empty-pixels gate: preview.rs's, table-tested on
    // any host. A capture can resolve with zero bytes when the page has not
    // composited a frame yet, and the tab hides the live page behind
    // whatever an answered IPC hands it — an empty blob hides it behind
    // nothing (owner report 2026-09-16: uniform gray where x.com should
    // freeze).
    let bytes = preview::read_still(&path)?;
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
    read_zoom(&app, &id).ok_or_else(|| "zoom: no answer".to_string())
}

// The controller's current ZoomFactor, read on the UI thread. `None` when
// the tab is not open or the round-trip died — callers treat it as best
// effort (the device toolbar only inherits it when the strip first shows).
fn read_zoom(app: &AppHandle, id: &str) -> Option<f64> {
    let wv = app.get_webview(&label(id))?;
    let (tx, rx) = mpsc::channel::<f64>();
    let done = tx.clone();
    let sent = wv.with_webview(move |platform| unsafe {
        let controller = platform.controller();
        let mut z = 1.0f64;
        if controller.ZoomFactor(&mut z).is_ok() {
            let _ = done.send(z);
        }
    });
    sent.ok()?;
    rx.recv_timeout(Duration::from_secs(5)).ok()
}

// Apply a zoom factor to one tab's controller (the UI-thread half of both
// zoom commands).
fn write_zoom(app: &AppHandle, id: &str, factor: f64) -> Result<(), String> {
    let wv = app.get_webview(&label(id)).ok_or("tab not open")?;
    let (tx, rx) = mpsc::channel::<Result<(), String>>();
    let done = tx.clone();
    let sent = wv.with_webview(move |platform| unsafe {
        let controller = platform.controller();
        let _ = done.send(
            controller
                .SetZoomFactor(factor)
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
    write_zoom(&app, &id, factor)?;
    Ok(factor)
}

// --- device toolbar ("Responsive width", the native half) -------------------
//
// The strip narrows the page to a preset width (bounds arithmetic in
// `picode_shell::responsive`) and applies a ZoomFactor zoom — no CDP, so no
// ADR (owner 2026-09-19); mobile UA / touch / DPR would be that other half
// and is deliberately absent. The state is per tab in BtabState and rides
// btab_meta back, so it survives tab switches and route returns.

// One place for applying the toolbar's state to a live webview: center the
// narrowed rect in the pane's last raw rect, and (when the call carries a
// zoom) set the controller's factor. A tab with no webview yet, or no rect
// yet, simply picks the state up on its next bounds push.
fn apply_responsive(
    app: &AppHandle,
    state: &BtabState,
    id: &str,
    st: responsive::Width,
    with_zoom: bool,
) -> Result<(), String> {
    let Some(wv) = app.get_webview(&label(id)) else {
        return Ok(());
    };
    if st.narrowed() {
        if let Some((x, y, w, h)) = state.panes.lock().unwrap().get(id).copied() {
            let r = responsive::width_rect(responsive::Rect::new(x, y, w, h), st.w);
            wv.set_bounds(tauri::Rect {
                position: LogicalPosition::new(r.x, r.y).into(),
                size: LogicalSize::new(r.w, r.h).into(),
            })
            .map_err(|e| e.to_string())?;
        }
    }
    if with_zoom {
        write_zoom(app, id, st.zoom)?;
    }
    Ok(())
}

// The inverse, shared by reset and hide: the full pane rect and 100% zoom.
fn restore_full(app: &AppHandle, state: &BtabState, id: &str) -> Result<(), String> {
    let Some(wv) = app.get_webview(&label(id)) else {
        return Ok(());
    };
    if let Some((x, y, w, h)) = state.panes.lock().unwrap().get(id).copied() {
        wv.set_bounds(tauri::Rect {
            position: LogicalPosition::new(x, y).into(),
            size: LogicalSize::new(w, h).into(),
        })
        .map_err(|e| e.to_string())?;
    }
    write_zoom(app, id, responsive::ZOOM_RESET)
}

/// `btab_responsive_set` shows or hides one tab's device toolbar and sets
/// its width or zoom — the UI sends only what changed. First show inherits
/// the controller's live zoom (the options menu may have moved it before the
/// strip existed), so the strip never shows a stale percent. Hiding resets:
/// zoom to 100%, full pane bounds — a hidden toolbar must not leave a
/// narrowed page behind with no control to fix it. Returns the state as
/// `btab_meta` paints it.
#[tauri::command]
pub async fn btab_responsive_set(
    app: AppHandle,
    state: State<'_, BtabState>,
    id: String,
    on: Option<bool>,
    width: Option<f64>,
    zoom: Option<f64>,
) -> Result<serde_json::Value, String> {
    if on == Some(false) {
        state.responsive.lock().unwrap().remove(&id);
        restore_full(&app, &state, &id)?;
        return Ok(serde_json::json!({
            "on": false, "width": 0.0, "zoom": responsive::ZOOM_RESET
        }));
    }
    // First show inherits the controller's live zoom — the options menu may
    // have moved it before the strip existed. The read is a UI-thread
    // round-trip, so it happens outside the map lock.
    let fresh = !state.responsive.lock().unwrap().contains_key(&id);
    let inherited = if fresh { read_zoom(&app, &id) } else { None };
    let mut st = state
        .responsive
        .lock()
        .unwrap()
        .get(&id)
        .copied()
        .unwrap_or_default();
    if let Some(live) = inherited {
        st.zoom = live;
    }
    if let Some(w) = width {
        st.w = responsive::normalize_width(w);
    }
    if let Some(z) = zoom {
        st.zoom = responsive::clamp_zoom(z);
    }
    state.responsive.lock().unwrap().insert(id.clone(), st);
    apply_responsive(&app, &state, &id, st, zoom.is_some())?;
    Ok(serde_json::json!({ "on": true, "width": st.w, "zoom": st.zoom }))
}

/// `btab_responsive_reset` is the strip's reset: full pane width, zoom back
/// to 100%. The toolbar itself stays up — hiding it is
/// `btab_responsive_set { on: false }`.
#[tauri::command]
pub async fn btab_responsive_reset(
    app: AppHandle,
    state: State<'_, BtabState>,
    id: String,
) -> Result<serde_json::Value, String> {
    let mut map = state.responsive.lock().unwrap();
    let Some(st) = map.get_mut(&id) else {
        return Ok(serde_json::json!({
            "on": false, "width": 0.0, "zoom": responsive::ZOOM_RESET
        }));
    };
    st.w = 0.0;
    st.zoom = responsive::ZOOM_RESET;
    let st = *st;
    drop(map);
    restore_full(&app, &state, &id)?;
    Ok(serde_json::json!({ "on": true, "width": 0.0, "zoom": st.zoom }))
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
        ICoreWebView2Environment15, ICoreWebView2Find, ICoreWebView2FindOptions, ICoreWebView2_28,
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
            // FindOptions are minted by the environment (_15), not the
            // webview — the environment comes back through _2.
            let environment = match core28.Environment() {
                Ok(env) => env,
                Err(e) => {
                    let _ = done.send(Err(format!("find: {e}")));
                    return;
                }
            };
            let Ok(environment15) = environment.cast::<ICoreWebView2Environment15>() else {
                let _ = done.send(Err("this WebView2 runtime has no Find API".into()));
                return;
            };
            let options: ICoreWebView2FindOptions = match environment15.CreateFindOptions() {
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
    close_inner(&app, &state, &id)
}

// The body of btab_close, shared with the per-app data clear: a webview
// must be closed before its partition folder can be removed, and closing
// is the same ceremony either way.
fn close_inner(
    app: &AppHandle,
    state: &BtabState,
    id: &str,
) -> Result<(), String> {
    if let Some(wv) = app.get_webview(&label(id)) {
        // Drop this tab's event receivers on the UI thread before the webview
        // goes: a receiver kept past its page would keep the page alive. A
        // held Ask dies with it — every deferral is completed (denied) so a
        // closed page is not pinned by one.
        let tab = id.to_string();
        let _ = wv.with_webview(move |_| {
            RECEIVERS.with(|r| {
                r.borrow_mut().remove(&tab);
            });
            // The annotate channel dies with its page too: a receiver kept
            // past the webview is exactly what made a later tab under the
            // same id look dead (the arm path used to skip subscribing).
            ANNOTATE_RECEIVERS.with(|r| {
                r.borrow_mut().remove(&tab);
            });
            ANNOTATE_NAV.with(|r| {
                r.borrow_mut().remove(&tab);
            });
            if let Ok(mut tabs) = annotate_tabs().lock() {
                tabs.remove(&tab);
            }
            PENDING_PERMISSIONS.with(|p| {
                let mut map = p.borrow_mut();
                let asks: Vec<u64> = map
                    .iter()
                    .filter(|(_, v)| v.tab == tab)
                    .map(|(k, _)| *k)
                    .collect();
                for ask in asks {
                    if let Some(v) = map.remove(&ask) {
                        unsafe {
                            let _ = v.args.SetState(COREWEBVIEW2_PERMISSION_STATE_DENY);
                            let _ = v.deferral.Complete();
                        }
                    }
                }
            });
        });
        wv.close().map_err(|e| e.to_string())?;
    }
    state.pending.lock().unwrap().remove(id);
    state.unplaced.lock().unwrap().remove(id);
    state.rings.lock().unwrap().remove(id);
    state.panes.lock().unwrap().remove(id);
    state.responsive.lock().unwrap().remove(id);
    grants().lock().unwrap().remove(id);
    Ok(())
}

// Clear one installed web app's own storage (ADR-0153): close its webview
// (the folder is in use while it lives) and remove the partition folder.
// Only ids that name a partition qualify — browserlab::app_partition is
// the containment, so the folder removed is always the app's own.
#[tauri::command]
pub async fn btab_clear_app_data(
    app: AppHandle,
    state: State<'_, BtabState>,
    id: String,
) -> Result<(), String> {
    let folder = super::browserlab::app_partition(&id)
        .ok_or_else(|| "only installed web apps have their own data to clear".to_string())?;
    close_inner(&app, &state, &id)?;
    if !folder.exists() {
        return Ok(());
    }
    // Closing a webview tears its browser process down asynchronously; the
    // first removal attempt can still meet a locked file. Three tries, a
    // breath between, then the honest failure.
    // Closing a webview tears its browser process down asynchronously; the
    // folder stays locked past the close for a while (the app's tab is
    // usually open when someone asks for the clear — 2026-09-18, owner run:
    // three 300ms retries lost that race and the dialog swallowed the real
    // reason behind a generic message). Give the engine a real budget — ten
    // seconds — then the honest failure.
    let mut last = String::new();
    for attempt in 0..20 {
        match std::fs::remove_dir_all(&folder) {
            Ok(()) => return Ok(()),
            Err(e) => {
                last = e.to_string();
                if attempt < 19 {
                    std::thread::sleep(std::time::Duration::from_millis(500));
                }
            }
        }
    }
    Err(format!(
        "could not clear the app's data — its files were still in use: {last}"
    ))
}

// Open a URL in the system default browser — the "Default browser" side of
// the web/local open-destination prefs (slice 3). CREATE_NO_WINDOW keeps
// cmd's console from flashing; the URL is scheme-checked here so nothing
// else can ride this command into an arbitrary shell verb.
#[tauri::command]
pub async fn btab_open_external(url: String) -> Result<(), String> {
    // The allowlist and its decision table live in external.rs, where its
    // tests run without cargo: http(s) for links that leave the app, and
    // `ms-settings:` for the OS screens PiCode does not reimplement (v2d).
    let target = crate::external::external_target(&url).ok_or_else(|| {
        "only http, https and ms-settings: targets can be handed to the system".to_string()
    })?;
    use std::os::windows::process::CommandExt;
    std::process::Command::new("cmd")
        .args(["/C", "start", "", &target])
        .creation_flags(0x0800_0000) // CREATE_NO_WINDOW
        .spawn()
        .map(|_| ())
        .map_err(|e| format!("open external: {e}"))
}

// --- Annotate mode (v2c) ----------------------------------------------------

// The script is injected into the live page (never over it: WebView2 forbids
// painting HTML over a native child), so the page keeps running and the
// overlay is the page's own DOM. The mode's state lives in the page (the
// script instance), in the receiver maps below, and in the set of armed
// tabs — every one of them pruned on close, because a stale entry is what
// taught the arm path to skip a subscription and leave a page silent.

// Tabs whose annotate mode is on: a completed navigation means a new
// document, and the re-injection below reads this to know whether the mode
// should survive it.
fn annotate_tabs() -> &'static std::sync::Mutex<std::collections::HashSet<String>> {
    static TABS: std::sync::OnceLock<std::sync::Mutex<std::collections::HashSet<String>>> =
        std::sync::OnceLock::new();
    TABS.get_or_init(|| std::sync::Mutex::new(std::collections::HashSet::new()))
}

thread_local! {
    // One message receiver per annotate-enabled tab, with the token that
    // unsubscribes it. COM interfaces are not Send, so they live on the UI
    // thread beside RECEIVERS. Keying on the tab id alone was the bug: a
    // webview can be recreated under the same id (a closed pane reopened
    // restores its tab), and the arm path skipped the subscription because
    // the id was already in the map — the page then worked (card, chips)
    // while every message went nowhere, silently (owner 2026-09-18: a saved
    // chip and a Send that never lit up).
    static ANNOTATE_RECEIVERS: std::cell::RefCell<std::collections::HashMap<String, (webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2WebMessageReceivedEventHandler, i64)>> =
        std::cell::RefCell::new(std::collections::HashMap::new());

    // The navigation hook that keeps the mode alive across a full page load
    // (the script belongs to the document, and a new document has none):
    // REINJECT_SCRIPT, which is the same idempotent script, is executed on
    // every completed navigation while the tab is armed.
    static ANNOTATE_NAV: std::cell::RefCell<std::collections::HashMap<String, (webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2NavigationCompletedEventHandler, i64)>> =
        std::cell::RefCell::new(std::collections::HashMap::new());
}

/// btab_annotate_mode turns the in-page annotate mode on or off for one tab.
/// On: enable WebView2's message channel for the tab, (re)subscribe to its
/// messages, and inject the script into the page that is already loaded.
/// Off: tell the page to take its overlay down. Nothing here goes through
/// CDP, so no tier applies — this is the human's own UI.
#[tauri::command]
pub async fn btab_annotate_mode(app: AppHandle, id: String, on: bool) -> Result<(), String> {
    // A work tab is a child Webview under the main window (WebviewBuilder in
    // `ensure`), never a WebviewWindow: looking it up as a window answers "no
    // such tab" for every tab that exists. The app's own tab id may carry a
    // prefix the shell never saw (`w:2` vs `2`), so the id's tail is tried too
    // — `ensure` is what named the webview, and it is named once.
    let wv = find_webview(&app, &id).ok_or_else(|| format!("no such tab: {id}"))?;
    let emitter = app.clone();
    let tab = id.clone();
    let (tx, rx) = std::sync::mpsc::channel::<Result<(), String>>();
    let _ = wv.with_webview(move |platform| unsafe {
        use webview2_com::WebMessageReceivedEventHandler;
        use windows::core::{HSTRING, PWSTR};
        let core = match platform.controller().CoreWebView2() {
            Ok(core) => core,
            Err(e) => {
                let _ = tx.send(Err(format!("CoreWebView2: {e}")));
                return;
            }
        };
        if let Ok(settings) = core.Settings() {
            let _ = settings.SetIsWebMessageEnabled(true);
        }
        // (Re)subscribe every arm: the old registrations (if any) are removed
        // first, so a webview recreated under the same id gets a live channel
        // instead of inheriting the map's stale key, and a re-arm never
        // double-delivers. Removal on a dead core fails harmlessly.
        let previous = ANNOTATE_RECEIVERS.with(|r| r.borrow_mut().remove(&tab));
        if let Some((_handler, token)) = previous {
            let _ = core.remove_WebMessageReceived(token);
        }
        let previous_nav = ANNOTATE_NAV.with(|r| r.borrow_mut().remove(&tab));
        if let Some((_handler, token)) = previous_nav {
            let _ = core.remove_NavigationCompleted(token);
        }
        if on {
            let emitter = emitter.clone();
            let tab_for_handler = tab.clone();
            let handler = WebMessageReceivedEventHandler::create(Box::new(move |_sender, args| {
                if let Some(args) = args {
                    let mut raw = PWSTR::null();
                    if args.WebMessageAsJson(&mut raw).is_ok() {
                        let payload = unsafe { raw.to_string().unwrap_or_default() };
                        // The UI matches the tab: {id, raw} keeps the page's own
                        // JSON intact instead of re-encoding it here.
                        let outer = serde_json::json!({ "id": tab_for_handler, "raw": payload });
                        let _ = emitter.emit("btab://annotate", outer.to_string());
                    }
                }
                Ok(())
            }));
            let mut token = 0i64;
            if let Err(e) = core.add_WebMessageReceived(&handler, &mut token) {
                let _ = tx.send(Err(format!("add_WebMessageReceived: {e}")));
                return;
            }
            ANNOTATE_RECEIVERS.with(|r| {
                r.borrow_mut().insert(tab.clone(), (handler, token));
            });
            // The mode lives in the document, and a navigation is a new
            // document: without this the overlay vanished while the strip
            // still said it was on (the REINJECT_SCRIPT debt, 2026-09-19).
            // The script is idempotent and its enter() re-reports the state,
            // so re-executing it on every completed navigation is safe.
            let nav_core = core.clone();
            let nav_tab = tab.clone();
            let nav = webview2_com::NavigationCompletedEventHandler::create(Box::new(move |_sender, _args| {
                let armed = annotate_tabs()
                    .lock()
                    .map(|s| s.contains(&nav_tab))
                    .unwrap_or(false);
                if armed {
                    let _ = unsafe {
                        nav_core.ExecuteScript(&HSTRING::from(crate::annotate::REINJECT_SCRIPT), None)
                    };
                }
                Ok(())
            }));
            let mut nav_token = 0i64;
            if let Err(e) = core.add_NavigationCompleted(&nav, &mut nav_token) {
                let _ = tx.send(Err(format!("add_NavigationCompleted: {e}")));
                return;
            }
            ANNOTATE_NAV.with(|r| {
                r.borrow_mut().insert(tab.clone(), (nav, nav_token));
            });
        }
        let script = if on {
            crate::annotate::SCRIPT
        } else {
            crate::annotate::EXIT_SCRIPT
        };
        match core.ExecuteScript(&HSTRING::from(script), None) {
            Ok(_) => {}
            Err(e) => {
                let _ = tx.send(Err(format!("ExecuteScript: {e}")));
                return;
            }
        }
        let _ = tx.send(Ok(()));
    });
    let out = rx.recv().unwrap_or_else(|_| Err("the shell did not answer".into()));
    out?;
    // The set the navigation hook reads. Written only after the webview's
    // thread did its part, so a navigation that lands in between sees the
    // mode as still off and the next arm re-injects anyway.
    let mut tabs = annotate_tabs().lock().unwrap();
    if on {
        tabs.insert(id);
    } else {
        tabs.remove(&id);
    }
    Ok(())
}

/// btab_annotate_clear discards annotations in one tab's loaded document
/// without leaving the mode: everything (the strip's trash) or just the most
/// recent pin (the strip's undo, `last`). It reuses the mode's ExecuteScript
/// path (no CDP, no tier): both snippets are guarded no-ops when the document
/// never armed the mode, so a click after a navigation that dropped the
/// script clears nothing and fails nothing.
#[tauri::command]
pub async fn btab_annotate_clear(app: AppHandle, id: String, last: Option<bool>) -> Result<(), String> {
    let wv = find_webview(&app, &id).ok_or_else(|| format!("no such tab: {id}"))?;
    let (tx, rx) = std::sync::mpsc::channel::<Result<(), String>>();
    let _ = wv.with_webview(move |platform| unsafe {
        use windows::core::HSTRING;
        let core = match platform.controller().CoreWebView2() {
            Ok(core) => core,
            Err(e) => {
                let _ = tx.send(Err(format!("CoreWebView2: {e}")));
                return;
            }
        };
        match core.ExecuteScript(
            &HSTRING::from(if last.unwrap_or(false) {
                crate::annotate::DROP_LAST_SCRIPT
            } else {
                crate::annotate::CLEAR_SCRIPT
            }),
            None,
        ) {
            Ok(_) => {
                let _ = tx.send(Ok(()));
            }
            Err(e) => {
                let _ = tx.send(Err(format!("ExecuteScript: {e}")));
            }
        }
    });
    rx.recv().unwrap_or_else(|_| Err("the shell did not answer".into()))
}

/// btab_annotate_overlay hides or shows the in-page annotation overlay. The
/// chrome brackets a page capture with it: pins, chips and the card live in
/// the page, so without this the picture sent to the agent is a photograph of
/// our own UI over the element it is meant to show (owner 2026-09-19).
#[tauri::command]
pub async fn btab_annotate_overlay(app: AppHandle, id: String, on: bool) -> Result<(), String> {
    let wv = find_webview(&app, &id).ok_or_else(|| format!("no such tab: {id}"))?;
    let (tx, rx) = std::sync::mpsc::channel::<Result<(), String>>();
    let _ = wv.with_webview(move |platform| unsafe {
        use windows::core::HSTRING;
        let core = match platform.controller().CoreWebView2() {
            Ok(core) => core,
            Err(e) => {
                let _ = tx.send(Err(format!("CoreWebView2: {e}")));
                return;
            }
        };
        let script = if on {
            crate::annotate::SHOW_SCRIPT
        } else {
            crate::annotate::HIDE_SCRIPT
        };
        match core.ExecuteScript(&HSTRING::from(script), None) {
            Ok(_) => {
                let _ = tx.send(Ok(()));
            }
            Err(e) => {
                let _ = tx.send(Err(format!("ExecuteScript: {e}")));
            }
        }
    });
    rx.recv().unwrap_or_else(|_| Err("the shell did not answer".into()))
}

/// btab_annotate_state pulls one tab's annotation state through the host→page
/// direction, which needs no page-side bridge: ExecuteScript's return value
/// comes back on the command's own result. The strip polls this while the
/// mode is on, so Send lights up even in a document whose `postMessage`
/// channel is dead (owner 2026-09-19: the page worked, every message
/// vanished). Empty string when the document never armed the mode.
#[tauri::command]
pub async fn btab_annotate_state(app: AppHandle, id: String) -> Result<String, String> {
    let wv = find_webview(&app, &id).ok_or_else(|| format!("no such tab: {id}"))?;
    let (tx, rx) = std::sync::mpsc::channel::<Result<String, String>>();
    let _ = wv.with_webview(move |platform| unsafe {
        use webview2_com::ExecuteScriptCompletedHandler;
        use windows::core::HSTRING;
        let core = match platform.controller().CoreWebView2() {
            Ok(core) => core,
            Err(e) => {
                let _ = tx.send(Err(format!("CoreWebView2: {e}")));
                return;
            }
        };
        // The error path reports through a second sender: the handler takes
        // the first one by move.
        let fail = tx.clone();
        let handler = ExecuteScriptCompletedHandler::create(Box::new(move |_err, result| {
            // `result` is the script's value as JSON: the page returned a
            // JSON string, so it arrives quoted. Decode once here and the
            // app receives the payload itself.
            let payload: String = serde_json::from_str(&result).unwrap_or_default();
            let _ = tx.send(Ok(payload));
            Ok(())
        }));
        if let Err(e) = core.ExecuteScript(&HSTRING::from(crate::annotate::STATE_SCRIPT), &handler) {
            let _ = fail.send(Err(format!("ExecuteScript: {e}")));
        }
    });
    rx.recv().unwrap_or_else(|_| Err("the shell did not answer".into()))
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

// The policy the user set per kind (the every-site entry) or per site (an
// "Always allow" answer from the Ask prompt): see picode_shell::permissions
// for the decision table. A kind with no entry follows the platform's own
// default (which is to deny).
fn permission_policy() -> &'static Mutex<HashMap<String, String>> {
    static POLICY: OnceLock<Mutex<HashMap<String, String>>> = OnceLock::new();
    POLICY.get_or_init(|| Mutex::new(HashMap::new()))
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

// The platform's kind for the name the dialog and the Ask prompt write — the
// reverse of `permission_kind_name`, over the store's closed vocabulary.
fn permission_kind_of(name: &str) -> Option<COREWEBVIEW2_PERMISSION_KIND> {
    Some(match name.trim().to_ascii_lowercase().as_str() {
        "camera" => COREWEBVIEW2_PERMISSION_KIND_CAMERA,
        "microphone" => COREWEBVIEW2_PERMISSION_KIND_MICROPHONE,
        "location" => COREWEBVIEW2_PERMISSION_KIND_GEOLOCATION,
        "notifications" => COREWEBVIEW2_PERMISSION_KIND_NOTIFICATIONS,
        "clipboard" => COREWEBVIEW2_PERMISSION_KIND_CLIPBOARD_READ,
        "autoplay" => COREWEBVIEW2_PERMISSION_KIND_AUTOPLAY,
        "sensors" => COREWEBVIEW2_PERMISSION_KIND_OTHER_SENSORS,
        "midi" => COREWEBVIEW2_PERMISSION_KIND_MIDI_SYSTEM_EXCLUSIVE_MESSAGES,
        "fonts" => COREWEBVIEW2_PERMISSION_KIND_LOCAL_FONTS,
        "filesystem" => COREWEBVIEW2_PERMISSION_KIND_FILE_READ_WRITE,
        _ => return None,
    })
}

// The engine keeps its own per-origin memory of a decision (the permission
// manager behind `SetPermissionState`), so a standing we forget in our map
// would keep working until the app restarts — and the dialog's Reset would be
// a control that does not do what it says. Every write that names a concrete
// origin tells the engine too. The profile is shared, so one call covers
// every tab, including the ones created later; the main window's profile is
// reachable even with no browser tab open. A `*` origin has no engine
// equivalent — that policy is ours alone.
fn apply_engine_state(app: &AppHandle, kind: &str, origin: Option<&str>, state: &str) {
    let Some(origin) = origin
        .map(str::trim)
        .filter(|o| !o.is_empty() && *o != permissions::ANY_SITE)
    else {
        return;
    };
    let Some(platform_kind) = permission_kind_of(kind) else {
        return;
    };
    let target = match state.trim().to_ascii_lowercase().as_str() {
        "allow" => COREWEBVIEW2_PERMISSION_STATE_ALLOW,
        "deny" => COREWEBVIEW2_PERMISSION_STATE_DENY,
        _ => COREWEBVIEW2_PERMISSION_STATE_DEFAULT,
    };
    // WebView2 may report the requesting document URI, while
    // SetPermissionState addresses a site origin. Keep both operations on
    // the same scheme+authority key so Reset and Allow once actually clear
    // the profile entry that Allow previously created.
    let origin = permissions::site_of(origin);
    for (name, wv) in app.webviews() {
        if !name.starts_with("btab-") && name != "main-content" {
            continue;
        }
        let origin = origin.clone();
        let _ = wv.with_webview(move |platform| unsafe {
            let Ok(core) = platform.controller().CoreWebView2() else {
                return;
            };
            let Ok(core13) =
                core.cast::<webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2_13>()
            else {
                return;
            };
            let Ok(profile) = core13.Profile() else {
                return;
            };
            let Ok(profile4) = profile
                .cast::<webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2Profile4>()
            else {
                return;
            };
            let wide: Vec<u16> = origin.encode_utf16().chain(std::iter::once(0)).collect();
            // The engine completes it asynchronously; nothing waits on it —
            // the next request from that origin is the only thing that can
            // tell whether it landed, and it re-asks either way.
            let handler =
                webview2_com::SetPermissionStateCompletedHandler::create(Box::new(|_| Ok(())));
            let _ = profile4.SetPermissionState(
                platform_kind,
                windows::core::PCWSTR(wide.as_ptr()),
                target,
                &handler,
            );
        });
    }
}

/// How long an unanswered Ask holds the page's request before the shell
/// denies it. A held request with no answer hangs the site, so a prompt the
/// user never sees expires instead of pinning the deferral forever.
const PERMISSION_ASK_TTL: Duration = Duration::from_secs(60);

// One held request: the platform args and its deferral, plus what the prompt
// and the report need. COM interfaces are not Send, so this lives in a
// thread_local on the UI thread (the same rule as RECEIVERS), and the answer
// command is sync for the same reason.
struct PendingPermission {
    tab: String,
    origin: String,
    kind: String,
    args: ICoreWebView2PermissionRequestedEventArgs,
    deferral: ICoreWebView2Deferral,
}

thread_local! {
    static PENDING_PERMISSIONS: RefCell<HashMap<u64, PendingPermission>> = RefCell::new(HashMap::new());
}

fn next_ask_id() -> u64 {
    static SEQ: AtomicU64 = AtomicU64::new(0);
    SEQ.fetch_add(1, Ordering::Relaxed) + 1
}

// Every decision the shell makes is reported, so Settings ▸ Browser can list
// what each site got. `standing` marks a decision that is (or refreshes) a
// saved per-site standing rather than a one-off answer; `ask` names the
// prompt an answer belongs to, which the tab uses to drop its bar.
fn report_permission(
    app: &AppHandle,
    origin: &str,
    kind: &str,
    allow: bool,
    standing: bool,
    ask: Option<u64>,
) {
    let mut payload = serde_json::json!({
        "origin": origin,
        "kind": kind,
        "decision": if allow { "allow" } else { "deny" },
        "standing": standing,
    });
    if let Some(id) = ask {
        payload["ask"] = serde_json::json!(id);
    }
    let _ = app.emit("btab://permission", payload);
}

// The watchdog behind PERMISSION_ASK_TTL: sleep off the UI thread, then run
// the denial where the pending COM objects live.
fn expire_permission_ask(app: &AppHandle, ask: u64) {
    let app = app.clone();
    std::thread::spawn(move || {
        std::thread::sleep(PERMISSION_ASK_TTL);
        let main = app.clone();
        let _ = app.run_on_main_thread(move || {
            let pending = PENDING_PERMISSIONS.with(|p| p.borrow_mut().remove(&ask));
            let Some(p) = pending else {
                return;
            };
            unsafe {
                let _ = p.args.SetState(COREWEBVIEW2_PERMISSION_STATE_DENY);
                let _ = p.deferral.Complete();
            }
            report_permission(&main, &p.origin, &p.kind, false, false, Some(ask));
        });
    });
}

// What the user decided for a kind — for every site (no origin), or for one
// site (an Ask prompt's "Always", relayed back on load): allow, deny, ask,
// or "default" to forget the entry.
#[tauri::command]
pub async fn btab_set_permission_policy(
    app: AppHandle,
    kind: String,
    state: String,
    origin: Option<String>,
) -> Result<(), String> {
    {
        let mut policy = permission_policy().lock().unwrap();
        permissions::set(&mut policy, origin.as_deref(), &kind, &state)?;
    }
    // A site's entry is also the engine's: forgetting one must make the next
    // request ask again, instead of waiting for a restart.
    apply_engine_state(&app, &kind, origin.as_deref(), &state);
    Ok(())
}

// Answer a held Ask prompt. Sync on purpose: the pending deferrals are COM
// objects on the UI thread (a sync command runs there), and `remember` is
// the prompt's "Always" — it writes the site's standing so the next request
// from it does not ask again.
#[tauri::command]
pub fn btab_permission_answer(
    app: AppHandle,
    id: u64,
    state: String,
    remember: bool,
) -> Result<(), String> {
    let allow = match state.trim().to_ascii_lowercase().as_str() {
        "allow" => true,
        "deny" => false,
        other => return Err(format!("{other:?} is not a permission answer")),
    };
    let pending = PENDING_PERMISSIONS.with(|p| p.borrow_mut().remove(&id));
    let Some(p) = pending else {
        return Err("that request is no longer waiting".into());
    };
    unsafe {
        let _ = p.args.SetState(if allow {
            COREWEBVIEW2_PERMISSION_STATE_ALLOW
        } else {
            COREWEBVIEW2_PERMISSION_STATE_DENY
        });
        let _ = p.deferral.Complete();
    }
    if remember {
        {
            let mut policy = permission_policy().lock().unwrap();
            permissions::set(
                &mut policy,
                Some(&p.origin),
                &p.kind,
                if allow { permissions::ALLOW } else { permissions::DENY },
            )?;
        }
        // "Always" is a standing for the site, so the engine's own memory
        // agrees with ours — and survives a relaunch of the app.
        apply_engine_state(&app, &p.kind, Some(&p.origin), if allow { "allow" } else { "deny" });
    } else {
        // SetState answers this request, but WebView2 may retain the decision
        // in the profile when the page has already negotiated the capability.
        // Allow once must not turn into a site standing: explicitly restore
        // the profile's DEFAULT state after completing the deferral.
        apply_engine_state(&app, &p.kind, Some(&p.origin), "default");
    }
    report_permission(&app, &p.origin, &p.kind, allow, remember, Some(id));
    Ok(())
}

// Every tab answers permission requests from that policy and reports the
// outcome, so the Site settings dialog can list what each site got. The
// "ask" state holds the request through a deferral and prompts in the tab;
// every other state answers immediately.
fn attach_permission_handler(app: &AppHandle, id: &str) {
    let Some(wv) = app.get_webview(label(id).as_str()) else {
        return;
    };
    let emitter = app.clone();
    let tab = id.to_string();
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
            let (decided, standing) = {
                let policy = permission_policy().lock().unwrap();
                match permissions::decide(&policy, &origin, name) {
                    permissions::Decision::Site(state) => (Some(state.to_string()), true),
                    permissions::Decision::Kind(state) => (Some(state.to_string()), false),
                    permissions::Decision::None => (None, false),
                }
            };
            match decided.as_deref() {
                Some("allow") => {
                    let _ = args.SetState(COREWEBVIEW2_PERMISSION_STATE_ALLOW);
                    report_permission(&emitter, &origin, name, true, standing, None);
                }
                Some("deny") => {
                    let _ = args.SetState(COREWEBVIEW2_PERMISSION_STATE_DENY);
                    report_permission(&emitter, &origin, name, false, standing, None);
                }
                Some("ask") => match args.GetDeferral() {
                    Ok(deferral) => {
                        let ask = next_ask_id();
                        PENDING_PERMISSIONS.with(|p| {
                            p.borrow_mut().insert(
                                ask,
                                PendingPermission {
                                    tab: tab.clone(),
                                    origin: origin.clone(),
                                    kind: name.to_string(),
                                    args: args.clone(),
                                    deferral,
                                },
                            );
                        });
                        let _ = emitter.emit(
                            "btab://permission-ask",
                            serde_json::json!({
                                "id": ask,
                                "tab": tab,
                                "origin": origin,
                                "kind": name,
                            }),
                        );
                        expire_permission_ask(&emitter, ask);
                    }
                    // No deferral: the request cannot wait, so it is denied
                    // now instead of hanging the page.
                    Err(_) => {
                        let _ = args.SetState(COREWEBVIEW2_PERMISSION_STATE_DENY);
                        report_permission(&emitter, &origin, name, false, false, None);
                    }
                },
                // No policy: the request falls through to the platform's own
                // default, which denies. Reported so it is visible.
                _ => report_permission(&emitter, &origin, name, false, false, None),
            }
            Ok(())
        }));
        let mut token: i64 = 0;
        let _ = core.add_PermissionRequested(&handler, &mut token);
    });
}

// --- Raw CDP (ADR-0144) ----------------------------------------------------

// Developer mode, as the page last told us. The daemon owns the setting and
// refuses first; this copy is the shell's own gate, so a command that arrived
// with a raw flag while the owner has the mode off is refused here too.
// Off unless the page said otherwise.
fn developer_mode() -> &'static Mutex<bool> {
    static DEVELOPER: OnceLock<Mutex<bool>> = OnceLock::new();
    DEVELOPER.get_or_init(|| Mutex::new(false))
}

#[tauri::command]
pub async fn btab_set_developer_mode(enabled: bool) -> Result<(), String> {
    *developer_mode().lock().unwrap() = enabled;
    Ok(())
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
            // Web messages ON at creation, not only when annotate mode is
            // armed: the injected annotate script posts its picks and its
            // state to the host, and a document created before the setting
            // was applied never gets the channel back — the page worked
            // (card, chips) while every message vanished (owner 2026-09-19).
            let _ = settings.SetIsWebMessageEnabled(true);
        }
    });
}
