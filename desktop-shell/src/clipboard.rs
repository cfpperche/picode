// clipboard.rs — the text clipboard for the computer tool (ADR-0148). Win32
// only: open, read or write CF_UNICODETEXT, close. Another app may hold the
// clipboard for a moment, so opening retries briefly instead of failing on
// the first busy answer.
use std::ffi::c_void;

use windows::Win32::Foundation::{HANDLE, HGLOBAL};
use windows::Win32::System::DataExchange::{
    CloseClipboard, EmptyClipboard, GetClipboardData, IsClipboardFormatAvailable, OpenClipboard,
    SetClipboardData,
};
use windows::Win32::System::Memory::{GlobalAlloc, GlobalLock, GlobalUnlock, GMEM_MOVEABLE};

const CF_UNICODETEXT: u32 = 13;
/// The most text one read hands the model.
pub const MAX_READ: usize = 200_000;

fn open() -> Result<(), String> {
    for _ in 0..8 {
        if unsafe { OpenClipboard(None) }.is_ok() {
            return Ok(());
        }
        std::thread::sleep(std::time::Duration::from_millis(25));
    }
    Err("clipboard: another program is holding the clipboard".into())
}

/// The clipboard's text, and how many characters were cut by MAX_READ.
pub fn read_text() -> Result<(String, usize), String> {
    open()?;
    let result = unsafe {
        (|| -> Result<(String, usize), String> {
            if IsClipboardFormatAvailable(CF_UNICODETEXT).is_err() {
                return Ok((String::new(), 0));
            }
            let h = GetClipboardData(CF_UNICODETEXT).map_err(|e| format!("clipboard: {e}"))?;
            let hg = HGLOBAL(h.0);
            let p = GlobalLock(hg) as *const u16;
            if p.is_null() {
                return Err("clipboard: the text could not be read".into());
            }
            let mut len = 0usize;
            while *p.add(len) != 0 && len < 8_000_000 {
                len += 1;
            }
            let units = std::slice::from_raw_parts(p, len);
            let text = String::from_utf16_lossy(units);
            let _ = GlobalUnlock(hg);
            let total = text.chars().count();
            if total > MAX_READ {
                let kept: String = text.chars().take(MAX_READ).collect();
                Ok((kept, total - MAX_READ))
            } else {
                Ok((text, 0))
            }
        })()
    };
    let _ = unsafe { CloseClipboard() };
    result
}

pub fn write_text(text: &str) -> Result<(), String> {
    let mut units: Vec<u16> = text.encode_utf16().collect();
    units.push(0);
    open()?;
    let result = unsafe {
        (|| -> Result<(), String> {
            EmptyClipboard().map_err(|e| format!("clipboard: {e}"))?;
            let bytes = units.len() * 2;
            let hg = GlobalAlloc(GMEM_MOVEABLE, bytes).map_err(|e| format!("clipboard: {e}"))?;
            let p = GlobalLock(hg) as *mut u16;
            if p.is_null() {
                return Err("clipboard: no memory for the text".into());
            }
            std::ptr::copy_nonoverlapping(units.as_ptr(), p, units.len());
            let _ = GlobalUnlock(hg);
            // The clipboard owns the memory from here on.
            SetClipboardData(CF_UNICODETEXT, Some(HANDLE(hg.0 as *mut c_void))).map_err(|e| format!("clipboard: {e}"))?;
            Ok(())
        })()
    };
    let _ = unsafe { CloseClipboard() };
    result
}
// Clipboard files (CF_HDROP): screenshots arrive as bytes through the web
// paste event, but files copied in Explorer arrive as references the
// browser never exposes (VSCode hit the same wall in #301603 and went
// native). This reads them on the Windows side and hands bytes back, so
// the existing drop door stages them unchanged — same caps (4 files,
// 4 MB each), same naming downstream. No files, or an unreadable entry:
// skip it and stage the rest; an empty clipboard answers an empty list,
// never an error.
use windows::Win32::UI::Shell::{DragQueryFileW, HDROP};

// CF_HDROP has no constant in the windows crate: the Win32 value is 15.
const CF_HDROP: u32 = 15;
const MAX_CLIP_FILES: usize = 4;
const MAX_CLIP_FILE_BYTES: usize = 4 * 1024 * 1024;

#[derive(serde::Serialize)]
pub struct ClipboardFile {
    pub name: String,
    pub mime: String,
    pub data: String,
}

fn clip_mime(name: &str) -> &'static str {
    match name
        .rsplit('.')
        .next()
        .map(|e| e.to_ascii_lowercase())
        .as_deref()
    {
        Some("png") => "image/png",
        Some("jpg") | Some("jpeg") => "image/jpeg",
        Some("gif") => "image/gif",
        Some("webp") => "image/webp",
        Some("txt") | Some("md") | Some("log") => "text/plain",
        Some("pdf") => "application/pdf",
        _ => "application/octet-stream",
    }
}

pub fn read_files() -> Result<Vec<ClipboardFile>, String> {
    use base64::Engine as _;
    open()?;
    let result = unsafe {
        (|| -> Result<Vec<ClipboardFile>, String> {
            if IsClipboardFormatAvailable(CF_HDROP).is_err() {
                return Ok(Vec::new());
            }
            let h = GetClipboardData(CF_HDROP).map_err(|e| format!("clipboard: {e}"))?;
            // Borrowed from the clipboard: DragQueryFile only reads, and the
            // memory stays the clipboard's until CloseClipboard below — never
            // DragFinish a handle GetClipboardData handed out.
            let hdrop = HDROP(h.0);
            let count = DragQueryFileW(hdrop, 0xFFFF_FFFF, None);
            let mut out = Vec::new();
            for i in 0..count.min(MAX_CLIP_FILES as u32) {
                let len = DragQueryFileW(hdrop, i, None);
                if len == 0 {
                    continue;
                }
                let mut buf = vec![0u16; (len + 1) as usize];
                DragQueryFileW(hdrop, i, Some(&mut buf));
                let path = String::from_utf16_lossy(&buf[..len as usize]);
                let raw = match std::fs::read(&path) {
                    Ok(raw) => raw,
                    Err(_) => continue,
                };
                if raw.len() > MAX_CLIP_FILE_BYTES {
                    continue;
                }
                let name = std::path::Path::new(&path)
                    .file_name()
                    .map(|s| s.to_string_lossy().into_owned())
                    .unwrap_or_else(|| "file".into());
                let mime = clip_mime(&name).to_string();
                let data = base64::engine::general_purpose::STANDARD.encode(&raw);
                out.push(ClipboardFile { name, mime, data });
            }
            Ok(out)
        })()
    };
    let _ = unsafe { CloseClipboard() };
    result
}

/// The `clipboard_files` Tauri command: file bytes for the attach bar.
/// Sync on purpose — local reads under the caps above finish in
/// milliseconds, and nothing here awaits.
#[tauri::command]
pub fn clipboard_files() -> Result<Vec<ClipboardFile>, String> {
    read_files()
}
