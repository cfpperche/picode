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
