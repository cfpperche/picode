// embed.rs — spike (2026-09-18, owner's call): host another process's
// top-level window inside the shell's main window with SetParent, so the
// owner can measure what breaks (focus, dialogs, resize, restore) before
// deciding whether Windows apps get a pane like the work browser. Reached
// only from the Computer lab; nothing in the product calls it.
//
// Cross-process parent/child is legal and unsupported (Raymond Chen, "Is it
// legal to have a cross-process parent/child window relationship?"): the
// two threads' input queues become attached, and the child dies with the
// parent. `computer_unembed` puts the window back; the lab must call it
// before the shell exits, or the app goes with the shell.
use serde_json::{json, Value};
use std::collections::HashMap;
use std::sync::{Mutex, OnceLock};
use tauri::AppHandle;
use windows::Win32::Foundation::{HWND, RECT};
use windows::Win32::UI::HiDpi::{SetThreadDpiHostingBehavior, DPI_HOSTING_BEHAVIOR_MIXED};
use windows::Win32::UI::WindowsAndMessaging::{
    GetClientRect, GetWindowLongPtrW, GetWindowRect, IsWindow, SetParent, SetWindowLongPtrW, SetWindowPos,
    GWL_STYLE, HWND_TOP, SWP_FRAMECHANGED, SWP_NOZORDER, SWP_SHOWWINDOW, WS_CAPTION, WS_CHILD, WS_MAXIMIZEBOX,
    WS_MINIMIZEBOX, WS_POPUP, WS_SYSMENU, WS_THICKFRAME,
};

struct Saved {
    style: isize,
    rect: RECT,
}

fn saved() -> &'static Mutex<HashMap<isize, Saved>> {
    static S: OnceLock<Mutex<HashMap<isize, Saved>>> = OnceLock::new();
    S.get_or_init(|| Mutex::new(HashMap::new()))
}

fn hwnd(id: isize) -> HWND {
    HWND(id as *mut core::ffi::c_void)
}

fn main_hwnd(app: &AppHandle) -> Result<HWND, String> {
    // The registry answers None on a resident the logon task started with
    // --hidden (2026-09-19, the first morning of the spike); the kept
    // handle in main.rs is what always works.
    let win = crate::main_window(app).ok_or("no main window")?;
    let h = win.hwnd().map_err(|e| format!("main hwnd: {e}"))?;
    Ok(hwnd(h.0 as isize))
}

/// Reparents `window` into the main window's right half. Answers the
/// rectangle used and what was saved for `computer_unembed`.
#[tauri::command]
pub fn computer_embed(app: AppHandle, window: i64) -> Result<Value, String> {
    let id = window as isize;
    let h = hwnd(id);
    if !unsafe { IsWindow(Some(h)) }.as_bool() {
        return Err("no such window: it closed, or the id is stale — call windows again".into());
    }
    let main = main_hwnd(&app)?;
    if id == main.0 as isize {
        return Err("embed: that is the PiCode window itself".into());
    }
    if saved().lock().unwrap().contains_key(&id) {
        return Err("embed: that window is already embedded — unembed it first".into());
    }
    let mut rect = RECT::default();
    unsafe { GetWindowRect(h, &mut rect) }.map_err(|e| format!("embed: GetWindowRect: {e}"))?;
    let style = unsafe { GetWindowLongPtrW(h, GWL_STYLE) };
    let mut client = RECT::default();
    unsafe { GetClientRect(main, &mut client) }.map_err(|e| format!("embed: GetClientRect: {e}"))?;
    let cw = client.right - client.left;
    let ch = client.bottom - client.top;
    let (x, y, w, hh) = (cw / 2, 80, cw / 2 - 16, ch - 96);
    // Mixed hosting lets a child of another DPI awareness live under us
    // (Windows 10 1803+); without it SetParent refuses across awareness.
    let _ = unsafe { SetThreadDpiHostingBehavior(DPI_HOSTING_BEHAVIOR_MIXED) };
    let strip = (WS_POPUP | WS_CAPTION | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX | WS_SYSMENU).0 as isize;
    let child = (style & !strip) | WS_CHILD.0 as isize;
    unsafe {
        SetWindowLongPtrW(h, GWL_STYLE, child);
    }
    let before = unsafe { SetParent(h, Some(main)) }.map_err(|e| {
        unsafe {
            SetWindowLongPtrW(h, GWL_STYLE, style);
        }
        format!("embed: SetParent refused ({e}) — an elevated app, or a window that cannot be a child")
    })?;
    unsafe { SetWindowPos(h, Some(HWND_TOP), x, y, w, hh, SWP_FRAMECHANGED | SWP_SHOWWINDOW) }
        .map_err(|e| format!("embed: SetWindowPos: {e}"))?;
    saved().lock().unwrap().insert(id, Saved { style, rect });
    Ok(json!({ "ok": true, "window": id, "rect": [x, y, w, hh], "parentBefore": before.0 as isize, "styleBefore": style }))
}

/// Puts an embedded window back on the desktop where it was.
#[tauri::command]
pub fn computer_unembed(window: i64) -> Result<Value, String> {
    let id = window as isize;
    let h = hwnd(id);
    let Some(s) = saved().lock().unwrap().remove(&id) else {
        return Err("unembed: that window is not embedded".into());
    };
    if !unsafe { IsWindow(Some(h)) }.as_bool() {
        return Err("unembed: the window is gone".into());
    }
    unsafe {
        SetWindowLongPtrW(h, GWL_STYLE, s.style);
    }
    unsafe { SetParent(h, None) }.map_err(|e| format!("unembed: SetParent: {e}"))?;
    let (w, hh) = (s.rect.right - s.rect.left, s.rect.bottom - s.rect.top);
    unsafe { SetWindowPos(h, None, s.rect.left, s.rect.top, w, hh, SWP_FRAMECHANGED | SWP_SHOWWINDOW | SWP_NOZORDER) }
        .map_err(|e| format!("unembed: SetWindowPos: {e}"))?;
    Ok(json!({ "ok": true, "window": id, "rect": [s.rect.left, s.rect.top, w, hh] }))
}

/// Everything still embedded, for the lab and for a shutdown that wants to
/// give the windows back.
pub fn embedded() -> Vec<isize> {
    saved().lock().unwrap().keys().copied().collect()
}
