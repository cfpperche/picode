// picode-shell is the Desktop v2 window (docs/plans/desktop-v2.md,
// ADR-0120): a thin Tauri 2 shell that renders the PiCode UI served by the
// daemon inside WSL, plus a tray. The daemon is the source of truth; this
// process is a client and a supervisor, never a second backend.
//
// Phase 1 scope, deliberately small: discover the server address, open the
// window on it, keep a tray with Open/Quit, and stay single-instance. The
// keepalive, the disk actions and the browser policy arrive in later phases —
// the Go tray keeps owning them until then.

mod btab;
mod browserlab;
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
    Emitter, Manager, WebviewUrl, WebviewWindowBuilder,
};
use tauri_plugin_notification::NotificationExt;

fn main() {
    // ADR-0128: no debug port exists in default operation. The shell bridges
    // CDP through the host API (btab_cdp_call), so the loopback port is an
    // explicit opt-in for external tooling instead of the way in. When it is
    // on, any local process can attach and the agent policy binds only what
    // flows through the daemon — the cost the toggle states.
    if let Ok(port) = std::env::var("PICODE_CDP_PORT") {
        match port.trim().parse::<u16>() {
            Ok(port) if port > 0 => std::env::set_var(
                "WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS",
                format!("--remote-debugging-port={port}"),
            ),
            _ => eprintln!("PICODE_CDP_PORT is not a port number; the debug port stays off"),
        }
    }
    // The shell loads its own bundle, not the launcher's pick: /desktop/ is
    // composed for the shell only (ADR-0122), /browser/ is what a browser gets.
    let url = discover_server_url().map(|mut u| {
        u.set_path("/desktop/");
        u
    });

    tauri::Builder::default()
        .plugin(tauri_plugin_notification::init())
        .invoke_handler(tauri::generate_handler![
            btab::btab_navigate,
            btab::btab_bounds,
            btab::btab_visibility,
            btab::btab_back,
            btab::btab_forward,
            btab::btab_reload,
            btab::btab_meta,
            btab::btab_screenshot,
            btab::btab_cdp_call,
            btab::btab_cdp_events,
            btab::btab_close,
            btab::btab_open_external,
            btab::btab_set_prefs,
            btab::btab_download_dir,
            btab::btab_set_download_dir,
            btab::btab_set_ask_download,
            btab::btab_open_path,
            btab::btab_reveal_path,
            btab::btab_clear_data,
            btab::btab_set_permission_policy,
            browserlab::lab_open,
            browserlab::lab_navigate,
            browserlab::lab_back,
            browserlab::lab_forward,
            browserlab::lab_reload,
            browserlab::lab_current_url,
            disk::disk_report,
            disk::disk_compact,
            disk::disk_compact_dry_run,
            clean::clean_list,
            clean::clean_apply,
            wslconfig::wslconfig_read,
            wslconfig::wslconfig_write,
        ])
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            // A second launch means someone wanted PiCode on screen: focus the
            // window the first instance already owns instead of starting over.
            if let Some(win) = app.get_webview_window("main") {
                show(&win);
            }
        }))
        .manage(browserlab::LabState::default())
        .manage(btab::BtabState::default())
        .setup(move |app| {
            // The lab window is built here, hidden — never inside a tray
            // handler: creating a second webview mid-event-loop deadlocked
            // the whole app on Windows (frozen captions, blank page).
            browserlab::init(app);
            let target = match url {
                Some(u) => WebviewUrl::External(u),
                None => WebviewUrl::App("offline.html".into()),
            };

            // Undecorated, single webview: the /desktop/ bundle renders the
            // merged top row itself — brand, rail tabs, agent tabs and the
            // Windows caption buttons (ADR-0122). The shell only strips the
            // native frame.
            WebviewWindowBuilder::new(app, "main", target)
                .title("PiCode")
                .inner_size(1360.0, 880.0)
                .min_inner_size(720.0, 480.0)
                .decorations(false)
                .data_directory(browserlab::webview_profile())
                .build()?;
            let open = MenuItem::with_id(app, "open", "Open PiCode", true, None::<&str>)?;
            let lab = MenuItem::with_id(app, "browserlab", "Browser lab", true, None::<&str>)?;
            let newbtab = MenuItem::with_id(app, "newbtab", "New browser tab", true, None::<&str>)?;
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
            let menu = Menu::with_items(app, &[&open, &lab, &newbtab, &management, &notify, &quit])?;

            TrayIconBuilder::with_id("picode")
                .icon(app.default_window_icon().expect("bundled icon").clone())
                .tooltip("PiCode")
                .menu(&menu)
                .show_menu_on_left_click(false)
                .on_menu_event(|app, ev| match ev.id.as_ref() {
                    "open" => show_main(app),
                    "browserlab" => browserlab::open(app),
                    "newbtab" => {
                        let _ = app.emit("btab://new", "");
                    }
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
    if let Some(win) = app.get_webview_window("management") {
        show(&win);
        return;
    }
    let mut url = discover_server_url()
        .unwrap_or_else(|| tauri::Url::parse("https://localhost:8445/").expect("static origin"));
    url.set_path("/desktop/management.html");
    let _ = tauri::WebviewWindowBuilder::new(app, "management", WebviewUrl::External(url))
        .title("PiCode — Management")
        .inner_size(980.0, 860.0)
        .min_inner_size(720.0, 480.0)
        .decorations(false)
        .data_directory(browserlab::webview_profile())
        .build();
}

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

/// discover_server_url finds the daemon the same way the Go tray does: the
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
