// input.rs — the mouse and the keyboard for the computer tool (ADR-0148),
// through SendInput: absolute moves over the virtual screen, buttons, the
// wheel, virtual keys with the extended flag where Windows wants it, and text
// as Unicode key events so any character types regardless of layout. UIPI
// silently drops input aimed at a higher-integrity window; SendInput's count
// is the only tell, and it is reported.
use std::time::Duration;

use picode_shell::keys::{self, Chord, Key};
use windows::Win32::UI::Input::KeyboardAndMouse::{
    GetDoubleClickTime, MapVirtualKeyW, SendInput, VkKeyScanW, INPUT, INPUT_0, INPUT_KEYBOARD, MAPVK_VK_TO_CHAR,
    INPUT_MOUSE, KEYBDINPUT, KEYBD_EVENT_FLAGS, KEYEVENTF_EXTENDEDKEY, KEYEVENTF_KEYUP,
    KEYEVENTF_UNICODE, MAPVK_VK_TO_VSC, MOUSEEVENTF_ABSOLUTE, MOUSEEVENTF_HWHEEL, MOUSEEVENTF_LEFTDOWN,
    MOUSEEVENTF_LEFTUP, MOUSEEVENTF_MIDDLEDOWN, MOUSEEVENTF_MIDDLEUP, MOUSEEVENTF_MOVE,
    MOUSEEVENTF_RIGHTDOWN, MOUSEEVENTF_RIGHTUP, MOUSEEVENTF_VIRTUALDESK, MOUSEEVENTF_WHEEL,
    MOUSEINPUT, MOUSE_EVENT_FLAGS, VIRTUAL_KEY,
};
use windows::Win32::UI::WindowsAndMessaging::{
    GetSystemMetrics, SetCursorPos, SM_CXVIRTUALSCREEN, SM_CYVIRTUALSCREEN, SM_XVIRTUALSCREEN,
    SM_YVIRTUALSCREEN,
};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Button {
    Left,
    Right,
    Middle,
}

/// The most characters one `type` sends in one call. With `TYPE_PACE`
/// between characters this is 10 s, inside the daemon's 20 s window for the
/// action; a longer text is split by the caller (the tool says so).
pub const MAX_TYPE: usize = 2_000;

/// The pause between two typed characters. Measured 2026-09-18 in Windows
/// 11 Notepad through the real line (Claude Code → picode mcp → daemon →
/// shell): `KEYEVENTF_UNICODE` events fired back to back arrive at the app
/// as the *last* character repeated — two characters survived, eleven did
/// not (" 0123456789" landed as " 9999999999"). The app translates each
/// packet message when it gets to it, and a queue that already holds later
/// packets hands it the newest one. A pause lets each packet be translated
/// before the next is posted.
pub const TYPE_PACE: Duration = Duration::from_millis(5);

fn send(inputs: &[INPUT]) -> Result<(), String> {
    let n = unsafe { SendInput(inputs, std::mem::size_of::<INPUT>() as i32) } as usize;
    if n != inputs.len() {
        return Err(format!(
            "input: Windows took {n} of {} events — an elevated window (UIPI) or a locked desktop refuses injected input",
            inputs.len()
        ));
    }
    Ok(())
}

fn mouse(dx: i32, dy: i32, data: u32, flags: MOUSE_EVENT_FLAGS) -> INPUT {
    INPUT {
        r#type: INPUT_MOUSE,
        Anonymous: INPUT_0 { mi: MOUSEINPUT { dx, dy, mouseData: data, dwFlags: flags, time: 0, dwExtraInfo: 0 } },
    }
}

fn keybd(vk: u16, scan: u16, flags: KEYBD_EVENT_FLAGS) -> INPUT {
    INPUT {
        r#type: INPUT_KEYBOARD,
        Anonymous: INPUT_0 { ki: KEYBDINPUT { wVk: VIRTUAL_KEY(vk), wScan: scan, dwFlags: flags, time: 0, dwExtraInfo: 0 } },
    }
}

/// Moves the pointer to a physical point on the virtual screen. The absolute
/// event is what apps see; SetCursorPos then pins the exact pixel, because
/// the 0..65535 normalisation can land one off.
pub fn move_to(px: i32, py: i32) -> Result<(), String> {
    unsafe {
        let vx = GetSystemMetrics(SM_XVIRTUALSCREEN);
        let vy = GetSystemMetrics(SM_YVIRTUALSCREEN);
        let vw = GetSystemMetrics(SM_CXVIRTUALSCREEN).max(2) as i64;
        let vh = GetSystemMetrics(SM_CYVIRTUALSCREEN).max(2) as i64;
        let nx = ((px - vx) as i64 * 65535 / (vw - 1)).clamp(0, 65535) as i32;
        let ny = ((py - vy) as i64 * 65535 / (vh - 1)).clamp(0, 65535) as i32;
        send(&[mouse(nx, ny, 0, MOUSEEVENTF_MOVE | MOUSEEVENTF_ABSOLUTE | MOUSEEVENTF_VIRTUALDESK)])?;
        let _ = SetCursorPos(px, py);
    }
    Ok(())
}

pub fn button(b: Button, down: bool) -> Result<(), String> {
    let flags = match (b, down) {
        (Button::Left, true) => MOUSEEVENTF_LEFTDOWN,
        (Button::Left, false) => MOUSEEVENTF_LEFTUP,
        (Button::Right, true) => MOUSEEVENTF_RIGHTDOWN,
        (Button::Right, false) => MOUSEEVENTF_RIGHTUP,
        (Button::Middle, true) => MOUSEEVENTF_MIDDLEDOWN,
        (Button::Middle, false) => MOUSEEVENTF_MIDDLEUP,
    };
    send(&[mouse(0, 0, 0, flags)])
}

/// `count` clicks at the pointer, spaced well inside the double-click time.
pub fn click(b: Button, count: u32) -> Result<(), String> {
    let gap = Duration::from_millis((unsafe { GetDoubleClickTime() } / 4).clamp(20, 120) as u64);
    for i in 0..count.max(1) {
        button(b, true)?;
        std::thread::sleep(Duration::from_millis(12));
        button(b, false)?;
        if i + 1 < count {
            std::thread::sleep(gap);
        }
    }
    Ok(())
}

/// Wheel notches: positive scrolls up (or left), negative down (or right).
pub fn wheel(notches: i32, horizontal: bool) -> Result<(), String> {
    let flags = if horizontal { MOUSEEVENTF_HWHEEL } else { MOUSEEVENTF_WHEEL };
    // WHEEL_DELTA is 120 per notch; a horizontal wheel reads positive as right.
    let delta = if horizontal { -notches * 120 } else { notches * 120 };
    send(&[mouse(0, 0, delta as u32, flags)])
}

fn scan_of(vk: u16) -> u16 {
    unsafe { MapVirtualKeyW(vk as u32, MAPVK_VK_TO_VSC) as u16 }
}

pub fn key(vk: u16, down: bool) -> Result<(), String> {
    let mut flags = KEYBD_EVENT_FLAGS(0);
    if keys::is_extended(vk) {
        flags |= KEYEVENTF_EXTENDEDKEY;
    }
    if !down {
        flags |= KEYEVENTF_KEYUP;
    }
    send(&[keybd(vk, scan_of(vk), flags)])
}

/// Resolves a chord key to a virtual key plus the layout's own modifiers
/// (Shift for '@' on most layouts, AltGr for some).
fn resolve(k: &Key) -> Result<(u16, Vec<u16>), String> {
    match k {
        Key::Vk(v) => Ok((*v, Vec::new())),
        Key::Char(c) => {
            let mut units = [0u16; 2];
            let encoded = c.encode_utf16(&mut units);
            if encoded.len() != 1 {
                return Err(format!("key: '{c}' is not a single key — use type for text"));
            }
            let r = unsafe { VkKeyScanW(encoded[0]) };
            if r == -1 {
                return Err(format!("key: this keyboard layout has no key for '{c}' — use type"));
            }
            let vk = (r & 0xFF) as u16;
            let state = (r >> 8) & 0xFF;
            let mut mods = Vec::new();
            if state & 1 != 0 {
                mods.push(keys::VK_SHIFT);
            }
            if state & 2 != 0 {
                mods.push(keys::VK_CONTROL);
            }
            if state & 4 != 0 {
                mods.push(keys::VK_MENU);
            }
            Ok((vk, mods))
        }
    }
}

/// Presses the chord: modifiers down, key down and up, modifiers up. Returns
/// only after everything is released, or releases what it pressed on error.
pub fn chord(ch: &Chord) -> Result<(), String> {
    let held = chord_down(ch)?;
    let r = key(held[held.len() - 1], false);
    for m in held[..held.len() - 1].iter().rev() {
        let _ = key(*m, false);
    }
    r
}

/// Presses and holds the chord; the returned keys (modifiers first, the key
/// last) are what `release` must let go.
pub fn chord_down(ch: &Chord) -> Result<Vec<u16>, String> {
    let (vk, extra) = resolve(&ch.key)?;
    let mut held = Vec::new();
    for m in ch.mods.iter().chain(extra.iter()) {
        if held.contains(m) {
            continue;
        }
        if let Err(e) = key(*m, true) {
            release(&held);
            return Err(e);
        }
        held.push(*m);
    }
    if let Err(e) = key(vk, true) {
        release(&held);
        return Err(e);
    }
    held.push(vk);
    Ok(held)
}

/// Lets go of held keys, last pressed first.
pub fn release(held: &[u16]) {
    for vk in held.iter().rev() {
        let _ = key(*vk, false);
    }
}

/// Types text as Unicode key events; newlines press Return and tabs press
/// Tab, so the text lands the way a person's would.
pub fn type_text(text: &str) -> Result<(), String> {
    let mut sent = 0usize;
    for ch in text.chars() {
        if sent >= MAX_TYPE {
            return Err(format!("type: stopped after {MAX_TYPE} characters — send the rest in another call"));
        }
        sent += 1;
        match ch {
            '\r' => continue,
            '\n' => {
                key(keys::VK_RETURN, true)?;
                key(keys::VK_RETURN, false)?;
            }
            '\t' => {
                key(0x09, true)?;
                key(0x09, false)?;
            }
            c => match layout_key(c) {
                // A layout key carries its character in the keystroke
                // itself, so a backlog in the app's queue cannot rewrite
                // it. Measured 2026-09-18 under load: Unicode packets paced
                // 5 ms apart still arrived as runs of one repeated
                // character whenever Notepad fell behind ("do PiCode"
                // became "dddddddde"); layout keystrokes did not.
                Some((vk, shift)) => {
                    if shift {
                        key(keys::VK_SHIFT, true)?;
                    }
                    let r = key(vk, true).and_then(|_| key(vk, false));
                    if shift {
                        let _ = key(keys::VK_SHIFT, false);
                    }
                    r?;
                }
                None => {
                    let mut units = [0u16; 2];
                    for u in c.encode_utf16(&mut units).iter() {
                        send(&[keybd(0, *u, KEYEVENTF_UNICODE), keybd(0, *u, KEYEVENTF_UNICODE | KEYEVENTF_KEYUP)])?;
                    }
                }
            },
        }
        std::thread::sleep(TYPE_PACE);
    }
    Ok(())
}

/// The key of the current layout that produces `c` with at most Shift, or
/// None when the character needs a Unicode packet: absent from the layout
/// (á on US, emoji), behind AltGr (Ctrl+Alt would also fire shortcuts), or
/// a dead key (´ ^ ~ on US-International wait for the next key).
fn layout_key(c: char) -> Option<(u16, bool)> {
    let mut units = [0u16; 2];
    let encoded = c.encode_utf16(&mut units);
    if encoded.len() != 1 || (c as u32) < 0x20 {
        return None;
    }
    let r = unsafe { VkKeyScanW(encoded[0]) };
    if r == -1 {
        return None;
    }
    let vk = (r & 0xFF) as u16;
    let state = (r >> 8) & 0xFF;
    if state & !1 != 0 {
        return None;
    }
    let dead = unsafe { MapVirtualKeyW(vk as u32, MAPVK_VK_TO_CHAR) } & 0x8000_0000 != 0;
    if dead {
        return None;
    }
    Some((vk, state & 1 != 0))
}
