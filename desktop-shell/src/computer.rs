// computer.rs — the computer tool's actuator (ADR-0148). One dedicated
// "desk" thread owns everything that touches the desktop: it is COM
// initialised (MTA, for UI Automation), per-monitor DPI aware (so every
// rectangle is physical), and it runs one job at a time — one action on the
// machine at any moment, and no COM object ever crosses a thread. The Tauri
// commands are the doors: they hand a job to the desk and wait for the
// answer, the way btab_cdp_call waits on the UI thread.
//
// Coordinates: the model sees images; every coordinate it sends is in the
// pixel space of the last image this desk returned to that principal
// (geometry::Frame). Nothing here knows tiers or bindings — the grant is one
// bit per principal, mirrored from the daemon by the page.
use std::collections::{HashMap, HashSet};
use std::sync::{mpsc, Mutex, OnceLock};
use std::time::{Duration, Instant};

use picode_shell::geometry::{self, Frame};
use picode_shell::{axfmt, b64, keys};
use serde_json::{json, Value};
use tauri::State;
use windows::Win32::System::Com::{CoInitializeEx, COINIT_MULTITHREADED};
use windows::Win32::UI::HiDpi::{SetThreadDpiAwarenessContext, DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2};
use windows::Win32::UI::WindowsAndMessaging::GetCursorPos;

use crate::{capture, clipboard, desktop, input, uia};

/// The image the model receives is at most this wide and tall (Anthropic's
/// guidance: 1280×720 fits the training resolution; larger costs tokens and
/// gets downscaled anyway).
pub const MAX_IMAGE_W: u32 = 1280;
pub const MAX_IMAGE_H: u32 = 1280;
/// The small JPEG the UI shows as "Last capture": capped so it does not bloat
/// the session log.
const PREVIEW_W: u32 = 480;
const PREVIEW_H: u32 = 480;
/// How long a door waits for the desk. Captures take tens of milliseconds;
/// `wait` and `hold_key` are capped below it.
const CALL_TIMEOUT: Duration = Duration::from_secs(20);
const MAX_WAIT_SECS: f64 = 5.0;
const MAX_SCROLL: i64 = 50;
const MAX_REPEAT: u64 = 100;
const MAX_DEPTH: u16 = 24;

/// The 23 actions, the order the refusal lists them in.
pub const ACTIONS: &[&str] = &[
    "screenshot", "zoom", "snapshot", "cursor_position", "wait", "windows", "focus",
    "left_click", "right_click", "middle_click", "double_click", "triple_click",
    "left_click_drag", "mouse_move", "left_mouse_down", "left_mouse_up", "scroll",
    "type", "key", "hold_key", "clipboard_read", "clipboard_write", "open",
];

pub enum Reply {
    Json(Value),
    Bytes(Vec<u8>),
}

pub struct Job {
    principal: String,
    action: String,
    params: Value,
    reply: mpsc::Sender<Result<Reply, String>>,
}

#[derive(Default)]
pub struct ComputerState {
    jobs: Mutex<Option<mpsc::Sender<Job>>>,
}

/// The grants, as the page last told us: the daemon owns the setting and
/// refuses first; this copy is the shell's own gate. Empty until told.
fn grants() -> &'static Mutex<HashSet<String>> {
    static G: OnceLock<Mutex<HashSet<String>>> = OnceLock::new();
    G.get_or_init(|| Mutex::new(HashSet::new()))
}

#[derive(Default)]
struct Desk {
    frames: HashMap<String, Frame>,
    shots: HashMap<String, Vec<u8>>,
    seq: u64,
    uia: Option<uia::Uia>,
    held_keys: Vec<u16>,
    held_buttons: Vec<input::Button>,
}

impl Desk {
    fn release_all(&mut self) {
        input::release(&self.held_keys);
        self.held_keys.clear();
        for b in self.held_buttons.drain(..) {
            let _ = input::button(b, false);
        }
    }
}

fn start_desk() -> mpsc::Sender<Job> {
    let (tx, rx) = mpsc::channel::<Job>();
    let spawned = std::thread::Builder::new().name("computer-desk".into()).spawn(move || {
        unsafe {
            let _ = CoInitializeEx(None, COINIT_MULTITHREADED);
            let _ = SetThreadDpiAwarenessContext(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2);
        }
        let mut desk = Desk::default();
        for job in rx {
            let started = Instant::now();
            let mut result = run(&mut desk, &job);
            if let Ok(Reply::Json(Value::Object(ref mut o))) = result {
                if let Some(Value::Object(meta)) = o.get_mut("meta") {
                    meta.insert("ms".into(), json!(started.elapsed().as_millis() as u64));
                }
            }
            if result.is_err() {
                // A refusal or a failure mid-action never leaves a key or a
                // button held: the human would inherit it.
                desk.release_all();
            }
            let _ = job.reply.send(result);
        }
    });
    if let Err(e) = spawned {
        eprintln!("computer desk: {e}");
    }
    tx
}

fn jobs(state: &ComputerState) -> mpsc::Sender<Job> {
    let mut guard = state.jobs.lock().unwrap();
    if guard.is_none() {
        *guard = Some(start_desk());
    }
    guard.as_ref().expect("set above").clone()
}

fn call(state: &ComputerState, principal: &str, action: &str, params: Value) -> Result<Reply, String> {
    let (reply, rx) = mpsc::channel();
    jobs(state)
        .send(Job { principal: principal.to_string(), action: action.to_string(), params, reply })
        .map_err(|_| "computer: the desk thread is gone".to_string())?;
    match rx.recv_timeout(CALL_TIMEOUT) {
        Ok(r) => r,
        Err(_) => Err(format!("timeout: {action} did not finish in {}s", CALL_TIMEOUT.as_secs())),
    }
}

fn json_call(state: &ComputerState, principal: &str, action: &str, params: Value) -> Result<Value, String> {
    match call(state, principal, action, params)? {
        Reply::Json(v) => Ok(v),
        Reply::Bytes(_) => Err("computer: unexpected binary answer".into()),
    }
}

#[tauri::command]
pub async fn computer_set_grants(keys: Vec<String>) -> Result<(), String> {
    *grants().lock().unwrap() = keys.into_iter().map(|k| k.trim().to_string()).filter(|k| !k.is_empty()).collect();
    Ok(())
}

#[tauri::command]
pub async fn computer_displays(state: State<'_, ComputerState>) -> Result<Value, String> {
    json_call(&state, "", "displays", json!({}))
}

#[tauri::command]
pub async fn computer_windows(state: State<'_, ComputerState>) -> Result<Value, String> {
    json_call(&state, "", "windows", json!({}))
}

/// The door the daemon's `computer.command` frames (and the lab) come
/// through: one action for one principal, the grant checked here too.
#[tauri::command]
pub async fn computer_call(
    state: State<'_, ComputerState>,
    principal: String,
    action: String,
    params_json: String,
) -> Result<Value, String> {
    let principal = principal.trim().to_string();
    if principal.is_empty() || !grants().lock().unwrap().contains(&principal) {
        return Err("disabled: this agent may not use the computer — turn it on in Settings ▸ Computer".into());
    }
    let params: Value = if params_json.trim().is_empty() {
        json!({})
    } else {
        serde_json::from_str(&params_json).map_err(|e| format!("params: {e}"))?
    };
    json_call(&state, &principal, &action, params)
}

/// The last image the desk returned to a principal, as raw PNG bytes for the
/// UI (the btab_preview shape).
#[tauri::command]
pub async fn computer_preview(state: State<'_, ComputerState>, principal: String) -> Result<tauri::ipc::Response, String> {
    match call(&state, principal.trim(), "preview", json!({}))? {
        Reply::Bytes(b) => Ok(tauri::ipc::Response::new(b)),
        Reply::Json(_) => Err("computer: no image yet".into()),
    }
}

// ---- the desk ---------------------------------------------------------------

fn num(v: &Value, key: &str) -> Option<f64> {
    v.get(key).and_then(|x| x.as_f64())
}

fn int(v: &Value, key: &str) -> Option<i64> {
    v.get(key).and_then(|x| x.as_i64().or_else(|| x.as_f64().map(|f| f.round() as i64)))
}

fn text(v: &Value, key: &str) -> Option<String> {
    v.get(key).and_then(|x| x.as_str()).map(|s| s.to_string())
}

fn point(v: &Value, key: &str) -> Result<Option<(i32, i32)>, String> {
    let Some(arr) = v.get(key) else { return Ok(None) };
    if arr.is_null() {
        return Ok(None);
    }
    let a = arr.as_array().ok_or_else(|| format!("bad_coordinate: {key} must be [x, y]"))?;
    if a.len() != 2 {
        return Err(format!("bad_coordinate: {key} must be [x, y]"));
    }
    let x = a[0].as_f64().ok_or("bad_coordinate: x is not a number")?;
    let y = a[1].as_f64().ok_or("bad_coordinate: y is not a number")?;
    Ok(Some((x.round() as i32, y.round() as i32)))
}

fn bounds_json(f: &Frame) -> Value {
    let (l, t, r, b) = f.bounds();
    json!({ "left": l, "top": t, "right": r, "bottom": b })
}

fn cursor_in(frame: Option<&Frame>) -> Value {
    let mut p = windows::Win32::Foundation::POINT::default();
    if unsafe { GetCursorPos(&mut p) }.is_err() {
        return Value::Null;
    }
    match frame.and_then(|f| f.to_image(p.x, p.y)) {
        Some((x, y)) => json!([x, y]),
        None => Value::Null,
    }
}

/// The current frame's physical point for an image coordinate, or the
/// reasons it cannot be one.
fn physical(desk: &Desk, principal: &str, (x, y): (i32, i32)) -> Result<(i32, i32), String> {
    let frame = desk
        .frames
        .get(principal)
        .ok_or("bad_coordinate: take a screenshot first — coordinates are pixels of the last image")?;
    frame.to_physical(x, y).ok_or_else(|| {
        format!("bad_coordinate: [{x}, {y}] is outside the last image ({}×{})", frame.img_w, frame.img_h)
    })
}

/// Encodes a shot, records it as the principal's frame and answers with the
/// image and its meta.
fn finish_frame(desk: &mut Desk, principal: &str, shot: capture::Shot, window: Option<&desktop::WindowInfo>, display: usize) -> Result<Value, String> {
    let (png, w, h) = capture::encode(&shot, MAX_IMAGE_W, MAX_IMAGE_H, false)?;
    let (jpg, _, _) = capture::encode(&shot, PREVIEW_W, PREVIEW_H, true)?;
    let frame = Frame { origin_x: shot.origin_x, origin_y: shot.origin_y, src_w: shot.w, src_h: shot.h, img_w: w, img_h: h };
    desk.seq += 1;
    desk.frames.insert(principal.to_string(), frame);
    desk.shots.insert(principal.to_string(), png.clone());
    Ok(json!({
        "ok": true,
        "image": b64::encode(&png),
        "mime": "image/png",
        "meta": {
            "display": display,
            "window": window,
            "width": w,
            "height": h,
            "scale": frame.scale(),
            "bounds": bounds_json(&frame),
            "seq": desk.seq,
            "cursor": cursor_in(Some(&frame)),
            "preview": format!("data:image/jpeg;base64,{}", b64::encode(&jpg)),
        }
    }))
}

fn display_of(displays: &[desktop::DisplayInfo], x: i32, y: i32) -> usize {
    displays.iter().find(|d| x >= d.left && x < d.right && y >= d.top && y < d.bottom).map(|d| d.index).unwrap_or(0)
}

fn screenshot(desk: &mut Desk, principal: &str, params: &Value) -> Result<Value, String> {
    let displays = desktop::displays();
    if let Some(id) = int(params, "window") {
        let info = desktop::window(id as isize, &displays)
            .ok_or("no such window: it closed, or the id is stale — call windows again")?;
        let rect = desktop::outer_rect(id as isize).ok_or("capture_failed: the window has no rectangle")?;
        let shot = capture::window(id as isize, rect)?;
        let display = info.display;
        return finish_frame(desk, principal, shot, Some(&info), display);
    }
    let index = int(params, "display").unwrap_or(1).max(1) as usize;
    let d = displays
        .iter()
        .find(|d| d.index == index)
        .ok_or_else(|| format!("bad_display: there is no display {index}; displays are 1..={}", displays.len().max(1)))?;
    let shot = capture::screen_rect(d.left, d.top, (d.right - d.left) as u32, (d.bottom - d.top) as u32, true)?;
    finish_frame(desk, principal, shot, None, index)
}

/// After an action: the same area again, so the model sees what changed.
fn recapture(desk: &mut Desk, principal: &str) -> Result<Value, String> {
    let Some(frame) = desk.frames.get(principal).copied() else {
        return screenshot(desk, principal, &json!({}));
    };
    let shot = capture::screen_rect(frame.origin_x, frame.origin_y, frame.src_w, frame.src_h, true)?;
    let displays = desktop::displays();
    let display = display_of(&displays, frame.origin_x, frame.origin_y);
    finish_frame(desk, principal, shot, None, display)
}

fn zoom(desk: &mut Desk, principal: &str, params: &Value) -> Result<Value, String> {
    let frame = desk.frames.get(principal).copied().ok_or("bad_coordinate: take a screenshot first, then zoom into a region of it")?;
    let region = params.get("region").and_then(|r| r.as_array()).ok_or("zoom: region must be [x0, y0, x1, y1] in the last image")?;
    if region.len() != 4 {
        return Err("zoom: region must be [x0, y0, x1, y1] in the last image".into());
    }
    let mut r = [0i32; 4];
    for (i, v) in region.iter().enumerate() {
        r[i] = v.as_f64().ok_or("zoom: region values must be numbers")?.round() as i32;
    }
    let target = geometry::zoom_frame(&frame, r, MAX_IMAGE_W, MAX_IMAGE_H)
        .ok_or_else(|| format!("bad_coordinate: the region {r:?} is empty or outside the last image ({}×{})", frame.img_w, frame.img_h))?;
    let shot = capture::screen_rect(target.origin_x, target.origin_y, target.src_w, target.src_h, true)?;
    let displays = desktop::displays();
    let display = display_of(&displays, target.origin_x, target.origin_y);
    finish_frame(desk, principal, shot, None, display)
}

fn snapshot(desk: &mut Desk, principal: &str, params: &Value) -> Result<Value, String> {
    if desk.uia.is_none() {
        desk.uia = Some(uia::Uia::new()?);
    }
    let displays = desktop::displays();
    let id = match int(params, "window") {
        Some(id) => id as isize,
        None => desktop::foreground(),
    };
    let info = desktop::window(id, &displays).ok_or("no such window: it closed, or the id is stale — call windows again")?;
    let depth = int(params, "depth").unwrap_or(MAX_DEPTH as i64).clamp(0, MAX_DEPTH as i64) as u16;
    let nodes = desk.uia.as_ref().expect("set above").snapshot(id, axfmt::MAX_NODES * 2, depth)?;
    let rendered = axfmt::render(&nodes, desk.frames.get(principal), axfmt::MAX_NODES);
    Ok(json!({
        "ok": true,
        "window": info,
        "lines": rendered.lines,
        "dropped": rendered.dropped,
        "nodes": nodes.len(),
        "meta": {}
    }))
}

fn modifiers(params: &Value) -> Result<Vec<u16>, String> {
    let Some(t) = text(params, "text") else { return Ok(Vec::new()) };
    let mut out = Vec::new();
    for part in t.split('+').map(|p| p.trim().to_ascii_lowercase()).filter(|p| !p.is_empty()) {
        let m = keys::modifier(&part).ok_or_else(|| format!("text: '{part}' is not a modifier (shift, ctrl, alt, super)"))?;
        if !out.contains(&m) {
            out.push(m);
        }
    }
    Ok(out)
}

fn click(desk: &mut Desk, principal: &str, params: &Value, b: input::Button, count: u32) -> Result<Value, String> {
    if let Some(p) = point(params, "coordinate")? {
        let (px, py) = physical(desk, principal, p)?;
        input::move_to(px, py)?;
    }
    let mods = modifiers(params)?;
    for m in &mods {
        input::key(*m, true)?;
        desk.held_keys.push(*m);
    }
    let r = input::click(b, count);
    input::release(&desk.held_keys);
    desk.held_keys.clear();
    r?;
    std::thread::sleep(Duration::from_millis(120));
    recapture(desk, principal)
}

fn drag(desk: &mut Desk, principal: &str, params: &Value) -> Result<Value, String> {
    let start = point(params, "start_coordinate")?.ok_or("left_click_drag: start_coordinate is required")?;
    let end = point(params, "coordinate")?.ok_or("left_click_drag: coordinate is required")?;
    let (sx, sy) = physical(desk, principal, start)?;
    let (ex, ey) = physical(desk, principal, end)?;
    input::move_to(sx, sy)?;
    input::button(input::Button::Left, true)?;
    desk.held_buttons.push(input::Button::Left);
    std::thread::sleep(Duration::from_millis(80));
    for i in 1..=8 {
        let x = sx + (ex - sx) * i / 8;
        let y = sy + (ey - sy) * i / 8;
        input::move_to(x, y)?;
        std::thread::sleep(Duration::from_millis(20));
    }
    input::button(input::Button::Left, false)?;
    desk.held_buttons.clear();
    std::thread::sleep(Duration::from_millis(120));
    recapture(desk, principal)
}

fn run(desk: &mut Desk, job: &Job) -> Result<Reply, String> {
    let p = &job.params;
    let principal = job.principal.as_str();
    let json = |v: Result<Value, String>| v.map(Reply::Json);
    match job.action.as_str() {
        "displays" => json(Ok(json!({ "displays": desktop::displays() }))),
        "windows" => {
            let displays = desktop::displays();
            json(Ok(json!({ "windows": desktop::windows(&displays), "displays": displays })))
        }
        "preview" => match desk.shots.get(principal) {
            Some(b) => Ok(Reply::Bytes(b.clone())),
            None => Err("no image yet: take a screenshot first".into()),
        },
        "screenshot" => json(screenshot(desk, principal, p)),
        "zoom" => json(zoom(desk, principal, p)),
        "snapshot" => json(snapshot(desk, principal, p)),
        "cursor_position" => {
            let mut pt = windows::Win32::Foundation::POINT::default();
            unsafe { GetCursorPos(&mut pt) }.map_err(|e| format!("cursor: {e}"))?;
            let frame = desk.frames.get(principal);
            let inside = frame.and_then(|f| f.to_image(pt.x, pt.y));
            json(Ok(json!({
                "ok": true,
                "coordinate": inside.map(|(x, y)| json!([x, y])).unwrap_or(Value::Null),
                "outside": inside.is_none(),
                "screen": [pt.x, pt.y],
                "meta": {}
            })))
        }
        "wait" => {
            let secs = num(p, "duration").unwrap_or(1.0).clamp(0.0, MAX_WAIT_SECS);
            std::thread::sleep(Duration::from_secs_f64(secs));
            json(recapture(desk, principal))
        }
        "focus" => {
            let id = int(p, "window").ok_or("focus: window (an id from windows) is required")?;
            desktop::focus(id as isize)?;
            let displays = desktop::displays();
            json(Ok(json!({ "ok": true, "window": desktop::window(id as isize, &displays), "meta": {} })))
        }
        "left_click" => json(click(desk, principal, p, input::Button::Left, 1)),
        "right_click" => json(click(desk, principal, p, input::Button::Right, 1)),
        "middle_click" => json(click(desk, principal, p, input::Button::Middle, 1)),
        "double_click" => json(click(desk, principal, p, input::Button::Left, 2)),
        "triple_click" => json(click(desk, principal, p, input::Button::Left, 3)),
        "left_click_drag" => json(drag(desk, principal, p)),
        "mouse_move" => {
            let pt = point(p, "coordinate")?.ok_or("mouse_move: coordinate is required")?;
            let (px, py) = physical(desk, principal, pt)?;
            input::move_to(px, py)?;
            json(Ok(json!({ "ok": true, "screen": [px, py], "meta": {} })))
        }
        "left_mouse_down" | "left_mouse_up" => {
            if let Some(pt) = point(p, "coordinate")? {
                let (px, py) = physical(desk, principal, pt)?;
                input::move_to(px, py)?;
            }
            let down = job.action == "left_mouse_down";
            input::button(input::Button::Left, down)?;
            if down {
                desk.held_buttons.push(input::Button::Left);
            } else {
                desk.held_buttons.retain(|b| *b != input::Button::Left);
            }
            json(Ok(json!({ "ok": true, "meta": {} })))
        }
        "scroll" => {
            if let Some(pt) = point(p, "coordinate")? {
                let (px, py) = physical(desk, principal, pt)?;
                input::move_to(px, py)?;
            }
            let amount = int(p, "scroll_amount").unwrap_or(3).clamp(1, MAX_SCROLL) as i32;
            let dir = text(p, "scroll_direction").unwrap_or_else(|| "down".into()).to_ascii_lowercase();
            let (notches, horizontal) = match dir.as_str() {
                "up" => (amount, false),
                "down" => (-amount, false),
                "left" => (amount, true),
                "right" => (-amount, true),
                other => return Err(format!("scroll: scroll_direction '{other}' is not up, down, left or right")),
            };
            input::wheel(notches, horizontal)?;
            std::thread::sleep(Duration::from_millis(150));
            json(recapture(desk, principal))
        }
        "type" => {
            let t = text(p, "text").ok_or("type: text is required")?;
            if t.chars().count() > input::MAX_TYPE {
                return Err(format!("type: {} characters is more than one call types ({}) — split it", t.chars().count(), input::MAX_TYPE));
            }
            input::type_text(&t)?;
            std::thread::sleep(Duration::from_millis(120));
            json(recapture(desk, principal))
        }
        "key" => {
            let t = text(p, "text").ok_or("key: text is required (for example ctrl+s, Return, alt+F4)")?;
            let chord = keys::parse_chord(&t)?;
            let repeat = int(p, "repeat").unwrap_or(1).clamp(1, MAX_REPEAT as i64);
            for i in 0..repeat {
                input::chord(&chord)?;
                if i + 1 < repeat {
                    std::thread::sleep(Duration::from_millis(30));
                }
            }
            std::thread::sleep(Duration::from_millis(120));
            json(recapture(desk, principal))
        }
        "hold_key" => {
            let t = text(p, "text").ok_or("hold_key: text is required")?;
            let chord = keys::parse_chord(&t)?;
            let secs = num(p, "duration").unwrap_or(1.0).clamp(0.0, MAX_WAIT_SECS);
            let held = input::chord_down(&chord)?;
            desk.held_keys = held.clone();
            std::thread::sleep(Duration::from_secs_f64(secs));
            input::release(&held);
            desk.held_keys.clear();
            json(recapture(desk, principal))
        }
        "clipboard_read" => {
            let (t, dropped) = clipboard::read_text()?;
            json(Ok(json!({ "ok": true, "text": t, "dropped": dropped, "meta": {} })))
        }
        "clipboard_write" => {
            let t = text(p, "text").ok_or("clipboard_write: text is required")?;
            clipboard::write_text(&t)?;
            json(Ok(json!({ "ok": true, "chars": t.chars().count(), "meta": {} })))
        }
        "open" => {
            let target = text(p, "target").map(|s| s.trim().to_string()).filter(|s| !s.is_empty())
                .ok_or("open: target is required (an app name, a path or a URL)")?;
            open_target(&target)?;
            json(Ok(json!({ "ok": true, "target": target, "meta": {} })))
        }
        other => Err(format!("unknown_action: '{other}' — use one of {}", ACTIONS.join(", "))),
    }
}

/// `start` resolves apps on the PATH, documents by association and URLs by
/// scheme — the same door btab_open_path uses.
fn open_target(target: &str) -> Result<(), String> {
    use std::os::windows::process::CommandExt;
    if target.contains('"') || target.contains('\n') {
        return Err("open: the target must not contain quotes or newlines".into());
    }
    std::process::Command::new("cmd")
        .args(["/C", "start", "", target])
        .creation_flags(0x0800_0000) // CREATE_NO_WINDOW
        .spawn()
        .map(|_| ())
        .map_err(|e| format!("open: {e}"))
}
