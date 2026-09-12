// picode-shell is the Desktop v2 window (docs/plans/desktop-v2.md,
// ADR-0120): a thin Tauri 2 shell that renders the PiCode UI served by the
// daemon inside WSL, plus a tray. The daemon is the source of truth; this
// process is a client and a supervisor, never a second backend.
//
// Phase 1 scope, deliberately small: discover the server address, open the
// window on it, keep a tray with Open/Quit, and stay single-instance. The
// keepalive, the disk actions and the browser policy arrive in later phases —
// the Go tray keeps owning them until then.

mod clean;
mod disk;
mod wslconfig;

// The undecorated window's frame — drag handles and window controls — is
// injected into every page the main webview shows, so the shell does not
// depend on which UI build the daemon happens to serve. Local pages
// (tauri.localhost) own their chrome and are skipped. Kept as one plain
// script: no build step lives between the shell and its window frame.
use std::process::Command;
use tauri::{
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    window::WindowBuilder,
    LogicalPosition, LogicalSize, Manager, WebviewUrl, WindowEvent,
};
use tauri_plugin_notification::NotificationExt;

fn main() {
    // The shell loads its own bundle, not the launcher's pick: /desktop/ is
    // composed for the shell only (ADR-0122), /browser/ is what a browser gets.
    let url = discover_server_url().map(|mut u| {
        u.set_path("/desktop/");
        u
    });

    tauri::Builder::default()
        .plugin(tauri_plugin_notification::init())
        .invoke_handler(tauri::generate_handler![
            disk::disk_report,
            disk::disk_compact,
            disk::disk_compact_dry_run,
            clean::clean_list,
            clean::clean_apply,
            wslconfig::wslconfig_read,
            wslconfig::wslconfig_write,
            open_dashboard,
        ])
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            // A second launch means someone wanted PiCode on screen: focus the
            // window the first instance already owns instead of starting over.
            if let Some(win) = app.get_window("main") {
                show(&win);
            }
        }))
        .setup(move |app| {
            let target = match url {
                Some(u) => WebviewUrl::External(u),
                None => WebviewUrl::App("offline.html".into()),
            };

            // Undecorated, with our own app bar (ADR-0122): a 40px local
            // webview spans the window's top edge — brand, drag, caption
            // buttons, all local so the ACL trusts them and no served
            // bundle is ever a dependency. The daemon's UI loads in the
            // webview below it and never sees a browser difference.
            spawn_window(app.handle(), "main", "PiCode", target, 1360.0, 880.0)?;

            let open = MenuItem::with_id(app, "open", "Open PiCode", true, None::<&str>)?;
            let management =
                MenuItem::with_id(app, "management", "Management\u{2026}", true, None::<&str>)?;
            // Phase 1 spike (docs/plans/desktop-v2.md): prove native
            // notifications from the shell — the agent-finished notice of
            // Phase 2 hangs off this same door.
            let notify = MenuItem::with_id(app, "notify", "Test notification", true, None::<&str>)?;
            let quit = MenuItem::with_id(
                app,
                "quit",
                "Quit (the service keeps running)",
                true,
                None::<&str>,
            )?;
            let menu = Menu::with_items(app, &[&open, &management, &notify, &quit])?;

            TrayIconBuilder::with_id("picode")
                .icon(app.default_window_icon().expect("bundled icon").clone())
                .tooltip("PiCode")
                .menu(&menu)
                .show_menu_on_left_click(false)
                .on_menu_event(|app, ev| match ev.id.as_ref() {
                    "open" => show_main(app),
                    "management" => open_management_window(app),
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

// open_management_window opens the Management page on demand — the second
// window of the shell: the WSL disk view, the cache prunes and the
// .wslconfig form, local pages, Rust commands behind them.
// management_url points the Management webview at the served bundle.
fn management_url(app: &tauri::AppHandle) -> tauri::Url {
    let mut url = discover_server_url().unwrap_or_else(|| {
        tauri::Url::parse("https://localhost:8445/").expect("static fallback origin")
    });
    url.set_path("/desktop/management.html");
    let _ = app;
    url
}

fn open_management_window(app: &tauri::AppHandle) {
    if let Some(win) = app.get_window("management") {
        show(&win);
        return;
    }
    let _ = spawn_window(
        app,
        "management",
        "PiCode — Management",
        // The Management page is part of the desktop bundle now.
        WebviewUrl::External(management_url(app)),
        980.0,
        860.0,
    );
}

// show_main brings the window to the front from the tray — the tap and the
// Open item are the same gesture.
fn show_main(app: &tauri::AppHandle) {
    if let Some(win) = app.get_window("main") {
        show(&win);
    }
}

fn show(win: &tauri::window::Window) {
    let _ = win.unminimize();
    let _ = win.show();
    let _ = win.set_focus();
}

// open_dashboard drives the served UI's dashboard from the app bar: the
// wordmark in the bar replaces the sidebar's own wordmark button in shell
// mode, and the UI listens for this event the same way it listens for
// picode-open-file.
#[tauri::command]
fn open_dashboard(app: tauri::AppHandle) -> Result<(), String> {
    let wv = app
        .get_webview("main-content")
        .ok_or_else(|| "the main window is not open".to_string())?;
    wv.eval("window.dispatchEvent(new CustomEvent('picode-open-dashboard'));")
        .map_err(|e| e.to_string())
}

/// The app bar's height, in logical pixels — the strip every window carries
/// above its content webview.
const BAR_H: f64 = 40.0;

// spawn_window builds an undecorated window with two webviews: our local
// app bar on top and the page itself below it, then keeps both webviews
// stretched to the window as it resizes. The bar is local on purpose —
// the ACL trusts local pages by default, so the frame can never go dead
// because of what the daemon serves.
fn spawn_window(
    app: &tauri::AppHandle,
    label: &str,
    title: &str,
    content: WebviewUrl,
    w: f64,
    h: f64,
) -> tauri::Result<tauri::window::Window> {
    let win = WindowBuilder::new(app, label)
        .title(title)
        .inner_size(w, h)
        .min_inner_size(720.0, 480.0)
        .decorations(false)
        .build()?;
    let bar = tauri::webview::WebviewBuilder::new(
        format!("{label}-titlebar"),
        WebviewUrl::App("titlebar.html".into()),
    );
    let page = tauri::webview::WebviewBuilder::new(format!("{label}-content"), content);
    relayout(&win)?;
    win.add_child(
        bar,
        LogicalPosition::new(0.0, 0.0),
        LogicalSize::new(0.0, BAR_H),
    )?;
    win.add_child(
        page,
        LogicalPosition::new(0.0, BAR_H),
        LogicalSize::new(0.0, 0.0),
    )?;
    relayout(&win)?;

    let handle = app.clone();
    let label = label.to_string();
    win.on_window_event(move |e| {
        if matches!(e, WindowEvent::Resized(_)) {
            if let Some(w) = handle.get_window(&label) {
                let _ = relayout(&w);
            }
        }
    });
    Ok(win)
}

// relayout stretches the bar across the top and the page under it, in
// logical pixels — tao hands us the window size in physical ones.
fn relayout(win: &tauri::window::Window) -> tauri::Result<()> {
    let label = win.label();
    let scale = win.scale_factor()?;
    let size = win.inner_size()?;
    let w = size.width as f64 / scale;
    let h = size.height as f64 / scale;
    if let Some(b) = win.get_webview(&format!("{label}-titlebar")) {
        b.set_bounds(tauri::Rect {
            position: LogicalPosition::new(0.0, 0.0).into(),
            size: LogicalSize::new(w, BAR_H).into(),
        })?;
    }
    if let Some(c) = win.get_webview(&format!("{label}-content")) {
        c.set_bounds(tauri::Rect {
            position: LogicalPosition::new(0.0, BAR_H).into(),
            size: LogicalSize::new(w, h - BAR_H).into(),
        })?;
    }
    Ok(())
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
            .args([
                "-d",
                distro,
                "--",
                "sh",
                "-lc",
                "cat \"$HOME/.picode/server.json\" 2>/dev/null",
            ])
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
    let units: Vec<u16> = bytes
        .chunks_exact(2)
        .map(|c| u16::from_le_bytes([c[0], c[1]]))
        .collect();
    String::from_utf16_lossy(&units)
}
