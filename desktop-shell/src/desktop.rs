// desktop.rs — the desktop as the computer tool sees it (ADR-0148): the
// monitors, the top-level windows, which one is in front, and bringing one to
// the front. Plain Win32 through the `windows` crate. Every function runs on
// the desk thread (computer.rs), which is per-monitor DPI aware, so every
// rectangle here is physical pixels on the virtual screen.
use std::ffi::c_void;

use serde::Serialize;
use windows::core::{BOOL, PWSTR};
use windows::Win32::Foundation::{CloseHandle, HWND, LPARAM, RECT};
use windows::Win32::Graphics::Dwm::{DwmGetWindowAttribute, DWMWA_CLOAKED, DWMWA_EXTENDED_FRAME_BOUNDS};
use windows::Win32::Graphics::Gdi::{
    EnumDisplayMonitors, GetMonitorInfoW, MonitorFromWindow, HDC, HMONITOR, MONITORINFO,
    MONITOR_DEFAULTTONEAREST,
};
use windows::Win32::System::Threading::{
    OpenProcess, QueryFullProcessImageNameW, PROCESS_NAME_WIN32, PROCESS_QUERY_LIMITED_INFORMATION,
};
use windows::Win32::UI::HiDpi::{GetDpiForMonitor, MDT_EFFECTIVE_DPI};
use windows::Win32::UI::WindowsAndMessaging::{
    EnumWindows, GetAncestor, GetForegroundWindow, GetWindowLongPtrW, GetWindowRect,
    GetWindowTextLengthW, GetWindowTextW, GetWindowThreadProcessId, IsIconic, IsWindow,
    IsWindowVisible, SetForegroundWindow, ShowWindow, GA_ROOT, GWL_EXSTYLE, SW_RESTORE,
    WS_EX_TOOLWINDOW,
};

/// MONITORINFOF_PRIMARY: the one monitor Windows calls primary.
const MONITOR_PRIMARY: u32 = 1;

/// One monitor. `index` is 1-based with the primary first — the number the
/// model names in `screenshot {display}`.
#[derive(Clone, Debug, Serialize)]
pub struct DisplayInfo {
    pub index: usize,
    pub left: i32,
    pub top: i32,
    pub right: i32,
    pub bottom: i32,
    pub primary: bool,
    pub dpi: u32,
    #[serde(skip)]
    handle: isize,
}

/// One top-level window worth naming: visible, uncloaked, titled, not a
/// tool window. `id` is the HWND as an integer — stable while the window
/// lives, meaningless after.
#[derive(Clone, Debug, Serialize)]
pub struct WindowInfo {
    pub id: isize,
    pub pid: u32,
    pub exe: String,
    pub title: String,
    pub left: i32,
    pub top: i32,
    pub right: i32,
    pub bottom: i32,
    pub display: usize,
    pub foreground: bool,
    pub minimized: bool,
}

pub fn hwnd(id: isize) -> HWND {
    HWND(id as *mut c_void)
}

pub fn displays() -> Vec<DisplayInfo> {
    unsafe extern "system" fn each(m: HMONITOR, _dc: HDC, _r: *mut RECT, lp: LPARAM) -> BOOL {
        let list = unsafe { &mut *(lp.0 as *mut Vec<HMONITOR>) };
        list.push(m);
        BOOL(1)
    }
    let mut mons: Vec<HMONITOR> = Vec::new();
    unsafe {
        let _ = EnumDisplayMonitors(None, None, Some(each), LPARAM(&mut mons as *mut _ as isize));
    }
    let mut out = Vec::new();
    for m in mons {
        let mut info = MONITORINFO { cbSize: std::mem::size_of::<MONITORINFO>() as u32, ..Default::default() };
        if !unsafe { GetMonitorInfoW(m, &mut info) }.as_bool() {
            continue;
        }
        let (mut dx, mut dy) = (96u32, 96u32);
        let _ = unsafe { GetDpiForMonitor(m, MDT_EFFECTIVE_DPI, &mut dx, &mut dy) };
        out.push(DisplayInfo {
            index: 0,
            left: info.rcMonitor.left,
            top: info.rcMonitor.top,
            right: info.rcMonitor.right,
            bottom: info.rcMonitor.bottom,
            primary: info.dwFlags & MONITOR_PRIMARY != 0,
            dpi: dx,
            handle: m.0 as isize,
        });
    }
    // The primary is display 1; the rest keep the enumeration order.
    out.sort_by_key(|d| !d.primary);
    for (i, d) in out.iter_mut().enumerate() {
        d.index = i + 1;
    }
    out
}

fn title_of(h: HWND) -> String {
    unsafe {
        let len = GetWindowTextLengthW(h);
        if len <= 0 {
            return String::new();
        }
        let mut buf = vec![0u16; len as usize + 1];
        let n = GetWindowTextW(h, &mut buf);
        String::from_utf16_lossy(&buf[..n.max(0) as usize])
    }
}

fn cloaked(h: HWND) -> bool {
    let mut c: u32 = 0;
    unsafe {
        DwmGetWindowAttribute(h, DWMWA_CLOAKED, &mut c as *mut u32 as *mut c_void, 4).is_ok() && c != 0
    }
}

/// The window's rectangle without the invisible resize borders when DWM
/// knows it, the plain window rect otherwise. Physical pixels.
pub fn window_rect(id: isize) -> Option<(i32, i32, i32, i32)> {
    let h = hwnd(id);
    let mut r = RECT::default();
    unsafe {
        if DwmGetWindowAttribute(h, DWMWA_EXTENDED_FRAME_BOUNDS, &mut r as *mut RECT as *mut c_void, std::mem::size_of::<RECT>() as u32).is_err()
            && GetWindowRect(h, &mut r).is_err()
        {
            return None;
        }
    }
    Some((r.left, r.top, r.right, r.bottom))
}

/// The plain window rect — what PrintWindow paints into.
pub fn outer_rect(id: isize) -> Option<(i32, i32, i32, i32)> {
    let mut r = RECT::default();
    unsafe { GetWindowRect(hwnd(id), &mut r).ok()? };
    Some((r.left, r.top, r.right, r.bottom))
}

/// The executable's file name (lowercase) for a process, or "" when the
/// process refuses the question (a higher-integrity one does).
pub fn exe_of(pid: u32) -> String {
    unsafe {
        let Ok(handle) = OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, pid) else {
            return String::new();
        };
        let mut buf = vec![0u16; 1024];
        let mut size = buf.len() as u32;
        let ok = QueryFullProcessImageNameW(handle, PROCESS_NAME_WIN32, PWSTR(buf.as_mut_ptr()), &mut size).is_ok();
        let _ = CloseHandle(handle);
        if !ok {
            return String::new();
        }
        let full = String::from_utf16_lossy(&buf[..size as usize]);
        full.rsplit(['\\', '/']).next().unwrap_or("").to_ascii_lowercase()
    }
}

pub fn foreground() -> isize {
    unsafe { GetForegroundWindow().0 as isize }
}

pub fn is_window(id: isize) -> bool {
    unsafe { IsWindow(Some(hwnd(id))).as_bool() }
}

fn info_of(h: HWND, displays: &[DisplayInfo], fg: isize) -> Option<WindowInfo> {
    unsafe {
        if !IsWindowVisible(h).as_bool() || cloaked(h) {
            return None;
        }
        let ex = GetWindowLongPtrW(h, GWL_EXSTYLE) as u32;
        if ex & WS_EX_TOOLWINDOW.0 != 0 {
            return None;
        }
        let title = title_of(h);
        if title.is_empty() {
            return None;
        }
        let mut pid = 0u32;
        GetWindowThreadProcessId(h, Some(&mut pid));
        let id = h.0 as isize;
        let (left, top, right, bottom) = window_rect(id)?;
        if right <= left || bottom <= top {
            return None;
        }
        let mon = MonitorFromWindow(h, MONITOR_DEFAULTTONEAREST);
        let display = displays.iter().find(|d| d.handle == mon.0 as isize).map(|d| d.index).unwrap_or(0);
        Some(WindowInfo {
            id,
            pid,
            exe: exe_of(pid),
            title,
            left,
            top,
            right,
            bottom,
            display,
            foreground: id == fg,
            minimized: IsIconic(h).as_bool(),
        })
    }
}

/// Every window the model may name, in Z order (front first).
pub fn windows(displays: &[DisplayInfo]) -> Vec<WindowInfo> {
    unsafe extern "system" fn each(h: HWND, lp: LPARAM) -> BOOL {
        let list = unsafe { &mut *(lp.0 as *mut Vec<HWND>) };
        list.push(h);
        BOOL(1)
    }
    let mut hwnds: Vec<HWND> = Vec::new();
    unsafe {
        let _ = EnumWindows(Some(each), LPARAM(&mut hwnds as *mut _ as isize));
    }
    let fg = foreground();
    hwnds.into_iter().filter_map(|h| info_of(h, displays, fg)).collect()
}

/// One window by id, or None when it is gone.
pub fn window(id: isize, displays: &[DisplayInfo]) -> Option<WindowInfo> {
    if !is_window(id) {
        return None;
    }
    info_of(hwnd(id), displays, foreground())
}

/// Brings a window to the front. Windows grants the foreground only to the
/// process that owns the last input or is already in front; when it refuses,
/// the error says so instead of pretending.
pub fn focus(id: isize) -> Result<(), String> {
    let h = hwnd(id);
    if !is_window(id) {
        return Err("no such window: it closed, or the id is stale — call windows again".into());
    }
    unsafe {
        if IsIconic(h).as_bool() {
            let _ = ShowWindow(h, SW_RESTORE);
        }
        let _ = SetForegroundWindow(h);
    }
    std::thread::sleep(std::time::Duration::from_millis(60));
    let front = unsafe { GetAncestor(GetForegroundWindow(), GA_ROOT).0 as isize };
    if front == id {
        Ok(())
    } else {
        Err("foreground_refused: Windows kept the focus where the human is working; click the window first, or try again".into())
    }
}
