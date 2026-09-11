// picode-shell is the Desktop v2 window (docs/plans/desktop-v2.md,
// ADR-0120): a thin Tauri 2 shell that renders the PiCode UI served by the
// daemon inside WSL, plus a tray. The daemon is the source of truth; this
// process is a client and a supervisor, never a second backend.
//
// Phase 1 scope, deliberately small: discover the server address, open the
// window on it, keep a tray with Open/Quit, and stay single-instance. The
// keepalive, the disk actions and the browser policy arrive in later phases —
// the Go tray keeps owning them until then.

use std::process::Command;
use tauri::{
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    Manager, WebviewUrl, WebviewWindowBuilder,
};
use tauri_plugin_notification::NotificationExt;

fn main() {
    let url = discover_server_url();

    tauri::Builder::default()
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            // A second launch means someone wanted PiCode on screen: focus the
            // window the first instance already owns instead of starting over.
            if let Some(win) = app.get_webview_window("main") {
                show(&win);
            }
        }))
        .setup(move |app| {
            let target = match url {
                Some(u) => WebviewUrl::External(u),
                None => WebviewUrl::App("offline.html".into()),
            };

            WebviewWindowBuilder::new(app, "main", target)
                .title("PiCode")
                .inner_size(1360.0, 880.0)
                .build()?;

            let open = MenuItem::with_id(app, "open", "Open PiCode", true, None::<&str>)?;
            // Phase 1 spike (docs/plans/desktop-v2.md): prove native
            // notifications from the shell — the agent-finished notice of
            // Phase 2 hangs off this same door.
            let notify = MenuItem::with_id(app, "notify", "Test notification", true, None::<&str>)?;
            let quit = MenuItem::with_id(app, "quit", "Quit (the service keeps running)", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&open, &notify, &quit])?;

            TrayIconBuilder::with_id("picode")
                .icon(app.default_window_icon().expect("bundled icon").clone())
                .tooltip("PiCode")
                .menu(&menu)
                .show_menu_on_left_click(false)
                .on_menu_event(|app, ev| match ev.id.as_ref() {
                    "open" => show_main(app),
                    "notify" => {
                        // Windows shows toasts for unpackaged apps only when a
                        // Start Menu shortcut with the app identity exists; a
                        // silent drop here is the spike saying the installer
                        // (Phase 2) must create that shortcut.
                        if let Err(e) = app
                            .notification()
                            .builder()
                            .title("PiCode")
                            .body("Native notifications work.")
                            .show()
                        {
                            eprintln!("notification: {e}");
                        }
                    }
                    "quit" => app.exit(0),
                    _ => {}
                })
                .on_tray_icon_event(|tray, ev| {
                    // Left-click is how tray apps open their window; the menu
                    // stays on the right click.
                    if let TrayIconEvent::Click {
                        button: MouseButton::Left,
                        button_state: MouseButtonState::Up,
                        ..
                    } = ev
                    {
                        show_main(tray.app_handle());
                    }
                })
                .build(app)?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running picode-shell");
}

// show_main brings the window to the front from the tray — the tap and the
// Open item are the same gesture.
fn show_main(app: &tauri::AppHandle) {
    if let Some(win) = app.get_webview_window("main") {
        show(&win);
    }
}

fn show(win: &tauri::WebviewWindow) {
    let _ = win.unminimize();
    let _ = win.show();
    let _ = win.set_focus();
}

// discover_server_url finds the daemon the same way the Go tray does: the
// address lives in <data>/server.json inside the distro, and wsl.exe is the
// door to it. The first distro that answers wins — on every machine this
// product targets there is exactly one.
fn discover_server_url() -> Option<tauri::Url> {
    let out = Command::new("wsl.exe")
        .args(["--list", "--quiet"])
        .output()
        .ok()?;
    for distro in console_string(&out.stdout).lines().map(str::trim) {
        if distro.is_empty() {
            continue;
        }
        let out = Command::new("wsl.exe")
            .args(["-d", distro, "--", "sh", "-lc", "cat \"$HOME/.picode/server.json\" 2>/dev/null"])
            .output()
            .ok()?;
        let text = console_string(&out.stdout);
        if let Some(start) = text.find('{') {
            if let Ok(found) = serde_json::from_str::<ServerJson>(text[start..].trim_end()) {
                if let Ok(parsed) = tauri::Url::parse(&found.url) {
                    return Some(parsed);
                }
            }
        }
    }
    None
}

#[derive(serde::Deserialize)]
struct ServerJson {
    url: String,
}

// console_string decodes what a Windows console program writes: UTF-16LE
// without a BOM. Same problem the Go half solves by inspecting the bytes —
// reading wsl.exe output as UTF-8 is how you end up with "P i C o d e".
fn console_string(bytes: &[u8]) -> String {
    if bytes.len() < 2 {
        return String::new();
    }
    if bytes[0] == 0xFF && bytes[1] == 0xFE {
        return decode_utf16(&bytes[2..]);
    }
    // Almost every odd byte zero is the UTF-16 signature; well-formed UTF-8
    // never has interior NUL bytes at all.
    let zeros = bytes[1..].iter().step_by(2).filter(|&&b| b == 0).count();
    let total = bytes.len() / 2;
    if total > 0 && zeros * 10 >= total * 7 {
        return decode_utf16(bytes);
    }
    String::from_utf8_lossy(bytes).into_owned()
}

fn decode_utf16(bytes: &[u8]) -> String {
    let units: Vec<u16> = bytes.chunks_exact(2).map(|c| u16::from_le_bytes([c[0], c[1]])).collect();
    String::from_utf16_lossy(&units)
}
