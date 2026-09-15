// Native dialogs (ADR-0142 slice 2): the shell has no dialog plugin, and a
// decision as heavy as stopping the distro is asked with the one dialog
// Windows gives every process for free. Same flags the retired Go tray used
// (msgbox_windows.go): foreground, topmost, warning icon. Both calls block
// the calling thread — no answer, no compact.

/// Asks a yes/no question and means it: the flow behind it stops the distro
/// and every session in it.
#[cfg(windows)]
pub fn confirm(caption: &str, text: &str) -> bool {
    use windows::core::HSTRING;
    use windows::Win32::UI::WindowsAndMessaging::{
        MessageBoxW, IDYES, MB_ICONWARNING, MB_SETFOREGROUND, MB_TOPMOST, MB_YESNO,
    };

    let caption = HSTRING::from(caption);
    let text = HSTRING::from(text);
    let style = MB_YESNO | MB_ICONWARNING | MB_SETFOREGROUND | MB_TOPMOST;
    unsafe { MessageBoxW(None, &text, &caption, style) == IDYES }
}

/// Reports an outcome. Nothing behind it is waiting on the answer.
#[cfg(windows)]
pub fn alert(caption: &str, text: &str) {
    use windows::core::HSTRING;
    use windows::Win32::UI::WindowsAndMessaging::{
        MessageBoxW, MB_ICONWARNING, MB_OK, MB_SETFOREGROUND, MB_TOPMOST,
    };

    let caption = HSTRING::from(caption);
    let text = HSTRING::from(text);
    let style = MB_OK | MB_ICONWARNING | MB_SETFOREGROUND | MB_TOPMOST;
    unsafe {
        MessageBoxW(None, &text, &caption, style);
    }
}

#[cfg(not(windows))]
pub fn confirm(_caption: &str, text: &str) -> bool {
    // Headless builds never compact: the safe answer is no.
    eprintln!("confirm (declined off Windows): {text}");
    false
}

#[cfg(not(windows))]
pub fn alert(caption: &str, text: &str) {
    eprintln!("{caption}: {text}");
}
