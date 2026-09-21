// Native dialogs (ADR-0142 slice 2): the shell has no dialog plugin, so
// outcomes are reported with the one dialog Windows gives every process for
// free. Same flags the retired Go tray used (msgbox_windows.go): foreground,
// topmost, warning icon. The call blocks the calling thread.

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
pub fn alert(caption: &str, text: &str) {
    eprintln!("{caption}: {text}");
}
