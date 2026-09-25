#![cfg_attr(
    not(debug_assertions),
    windows_subsystem = "windows"
)]
// picode-shell is the Desktop v2 window (docs/plans/desktop-v2.md,
// ADR-0120) and, since ADR-0142, the only Windows resident: a thin Tauri 2
// shell that renders the PiCode UI served by the daemon inside WSL, holds
// the distro open with a keepalive, and keeps one tray. The daemon is the
// source of truth; this process is a client and a supervisor, never a
// second backend. The keepalive and the Restart and Logs actions live here;
// the Go tray owned them until this shell took over. The disk line and the
// Give-back tray item moved into the Management window (its Disk tab runs
// the same compact flow), so the tray no longer carries them.

mod btab;
mod layers;
#[cfg(feature = "overlay-qa")]
mod overlayqa;
mod external;
mod annotate;
mod board;
mod hold;
mod daemon_acl;
mod browserlab;
mod capture;
mod clean;
mod clipboard;
mod computer;
mod computerlab;
mod embed;
mod desktop;
mod dialog;
mod disk;
mod health;
mod input;
mod keepalive;
mod status;
mod uia;
mod waiting;
mod wslconfig;

// The undecorated window's frame — drag handles and window controls — is
// injected into every page the main webview shows, so the shell does not
// depend on which UI build the daemon happens to serve. Local pages
// (tauri.localhost) own their chrome and are skipped. Kept as one plain
// script: no build step lives between the shell and its window frame.
use picode_shell::waitstate::{Cert, Stage};
use std::process::Command;
use std::sync::OnceLock;
use tauri::{
    menu::{Menu, MenuItem, PredefinedMenuItem, Submenu},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    Manager, WebviewUrl,
};

/// The main window, kept from the moment it is built. The registry lookup
/// (`get_webview_window("main")`) answered None on a resident started by the
/// logon task with `--hidden`, while the native window was alive and
/// perfectly showable — which is exactly the state where Open PiCode must
/// still work (2026-09-16, Tauri 2.11.5).
static MAIN_WINDOW: OnceLock<tauri::Window> = OnceLock::new();

/// The main window for modules that need its native handle (embed.rs):
/// the kept handle first, the registry as a fallback — see MAIN_WINDOW.
pub fn main_window(app: &tauri::AppHandle) -> Option<tauri::Window> {
    MAIN_WINDOW.get().cloned().or_else(|| app.get_window("main"))
}

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
    // The logon task starts the resident hidden: sign-in never pops a window,
    // the tray holds the service, and a second launch brings the window up
    // through the single-instance handler below.
    #[cfg(feature = "overlay-qa")]
    if let Some(url) = std::env::args().find_map(|a| a.strip_prefix("--overlay-qa=").map(str::to_owned)) {
        overlayqa::run(&url);
        return;
    }
    #[cfg(feature = "overlay-qa")]
    assert!(env!("CARGO_BIN_NAME") != "overlay_product_qa", "QA example requires --overlay-qa=<scratch URL>");
    let hidden = std::env::args().any(|a| a == "--hidden");
    // The shell loads its own bundle, not the launcher's pick: /desktop/ is
    // composed for the shell only (ADR-0122), /browser/ is what a browser gets.

    tauri::Builder::default()
        .plugin(tauri_plugin_notification::init())
        .invoke_handler(tauri::generate_handler![
            layers::chrome_layers,
            btab::btab_navigate,
            btab::btab_bounds,
            btab::btab_visibility,
            btab::btab_back,
            btab::btab_forward,
            btab::btab_reload,
            btab::btab_meta,
            btab::btab_screenshot,
            btab::btab_preview,
            btab::btab_print,
            btab::btab_zoom,
            btab::btab_set_zoom,
            btab::btab_responsive_set,
            btab::btab_responsive_reset,
            btab::btab_find,
            btab::btab_cdp_call,
            btab::btab_cdp_events,
            btab::btab_close,
            btab::btab_clear_app_data,
            btab::btab_open_external,
            btab::btab_annotate_mode,
            btab::btab_annotate_clear,
            btab::btab_annotate_state,
            btab::btab_annotate_overlay,
            btab::btab_set_prefs,
            btab::btab_download_dir,
            btab::btab_set_download_dir,
            btab::btab_set_ask_download,
            btab::btab_open_path,
            btab::btab_reveal_path,
            btab::btab_clear_data,
            btab::btab_set_permission_policy,
            btab::btab_permission_answer,
            btab::btab_set_scripts,
            btab::btab_set_developer_mode,
            browserlab::lab_open,
            browserlab::lab_navigate,
            browserlab::lab_back,
            browserlab::lab_forward,
            browserlab::lab_reload,
            browserlab::lab_current_url,
            computer::computer_displays,
            computer::computer_windows,
            computer::computer_call,
            computer::computer_set_grants,
            computer::computer_preview,
            clipboard::clipboard_files,
            computerlab::computerlab_open,
            embed::computer_embed,
            embed::computer_unembed,
            disk::disk_report,
            disk::disk_compact,
            disk::disk_compact_dry_run,
            disk::host_report,
            disk::wsl_restart,
            disk::wsl_update,
            disk::distro_conf_read,
            disk::distro_conf_write,
            disk::system_clean_apply,
            disk::places_report,
            disk::distro_move,
            disk::distro_backup,
            disk::history_report,
            clean::clean_list,
            clean::clean_apply,
            wslconfig::wslconfig_read,
            wslconfig::wslconfig_write,
            waiting::waiting_state,
            waiting::waiting_action,
        ])
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            // A second launch means someone wanted PiCode on screen: focus the
            // window the first instance already owns instead of starting over.
            // Routed through show_main so a missing window is rebuilt, not
            // silently ignored.
            show_main(app);
        }))
        .manage(browserlab::LabState::default())
        .manage(btab::BtabState::default())
        .manage(computer::ComputerState::default())
        .setup(move |app| {
            // The lab window is built here, hidden — never inside a tray
            // handler: creating a second webview mid-event-loop deadlocked
            // the whole app on Windows (frozen captions, blank page).
            browserlab::init(app);
            computerlab::init(app);
            // The window opens at once on the waiting page; the health loop
            // navigates it to /desktop/ when the daemon answers.
            let main_win = build_main_window(app.handle(), waiting_target())?;
            if hidden {
                let _ = main_win.hide();
            }
            let status_item =
                MenuItem::with_id(app, "status", "Starting…", false, None::<&str>)?;
            let status_sep = PredefinedMenuItem::separator(app)?;
            let open = MenuItem::with_id(app, "open", "Open PiCode", true, None::<&str>)?;
            let restart = MenuItem::with_id(
                app,
                "restart",
                "Restart PiCode",
                true,
                None::<&str>,
            )?;
            let logs =
                MenuItem::with_id(app, "logs", "View logs", true, None::<&str>)?;
            let management =
                MenuItem::with_id(app, "management", "Management\u{2026}", true, None::<&str>)?;
            let actions_sep = PredefinedMenuItem::separator(app)?;
            // The labs are owner-acceptance spikes (ADR-0148 M1), not
            // product: they sit one level down so the tray keeps its product
            // actions up front.
            let lab = MenuItem::with_id(app, "browserlab", "Browser lab", true, None::<&str>)?;
            let computerlab_item =
                MenuItem::with_id(app, "computerlab", "Computer lab", true, None::<&str>)?;
            let labs = Submenu::with_items(app, "Labs", true, &[&lab, &computerlab_item])?;
            let labs_sep = PredefinedMenuItem::separator(app)?;
            let quit = MenuItem::with_id(
                app,
                "quit",
                "Quit (the service keeps running)",
                true,
                None::<&str>,
            )?;
            let menu = Menu::with_items(
                app,
                &[
                    &status_item,
                    &status_sep,
                    &open,
                    &restart,
                    &logs,
                    &management,
                    &actions_sep,
                    &labs,
                    &labs_sep,
                    &quit,
                ],
            )?;

            TrayIconBuilder::with_id("picode")
                .icon(app.default_window_icon().expect("bundled icon").clone())
                .tooltip("PiCode")
                .menu(&menu)
                .show_menu_on_left_click(false)
                .on_menu_event(|app, ev| match ev.id.as_ref() {
                    "open" => show_main(app),
                    "restart" => {
                        std::thread::spawn(restart_flow);
                    }
                    "logs" => {
                        std::thread::spawn(logs_flow);
                    }
                    "browserlab" => browserlab::open(app),
                    "computerlab" => computerlab::open(app),
                    "management" => open_management_window(app),
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

            // The health thread reports to one board: one render composes
            // the status line and the tooltip, so no timer erases another's
            // write.
            board::init(board::Board::new(
                app.handle().clone(),
                status_item.clone(),
                open.clone(),
            ));
            std::thread::spawn(poll_loop);

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running picode-shell");
}

// How often the resident asks PiCode whether it is up: one curl to a
// loopback port, cheap enough for a timer. While the window shows the
// waiting page someone is looking at it, so the loop asks faster.
const POLL_EVERY_SECS: u64 = 5;
const WAITING_POLL_SECS: u64 = 2;

fn poll_loop() {
    let Some(board) = board::get() else { return };
    let app = board.app().clone();
    let mut url: Option<String> = None;
    let mut boot_id = String::new();
    // When the daemon stopped answering; drives Starting → Not answering.
    let mut failing_since: Option<std::time::Instant> = None;
    loop {
        let every = if waiting::on_waiting_page(&app) {
            WAITING_POLL_SECS
        } else {
            POLL_EVERY_SECS
        };
        if url.is_none() {
            // Discovery runs `wsl.exe -d <distro>`, which starts a stopped
            // distro — never while a flow holds it down.
            if hold::distro_held() {
                board.set_health(false, "paused while WSL work runs");
                waiting::set(&app, Stage::Paused);
                waiting::nap(every);
                continue;
            }
            // Only the first lookup is announced as "starting Linux": later
            // lookups re-read a moved port and must not flicker the stage.
            waiting::set_if_unset(&app, Stage::Linux);
            match discover_server() {
                Some((distro, found)) => {
                    // Before any window loads from it: a daemon off the
                    // default port needs its origin in the ACL.
                    daemon_acl::grant(board.app(), &found);
                    board.note_distro(&distro);
                    board.ensure_keepalive();
                    url = Some(found.to_string());
                }
                None => {
                    board.set_health(false, "PiCode has not started yet");
                    waiting::set(&app, Stage::NotFound);
                    waiting::nap(POLL_EVERY_SECS);
                    continue;
                }
            }
        }
        let base = url.clone().expect("discovered above");
        match health::fetch(&base) {
            Ok(h) => {
                failing_since = None;
                // The port can move inside its range (8445-8455), so a
                // failed probe invalidates the cached address rather than
                // being reported forever — and a changed boot id names the
                // restart.
                let restarted = !boot_id.is_empty() && boot_id != h.boot_id;
                boot_id = h.boot_id.clone();
                let mut detail = base.clone();
                if restarted {
                    detail.push_str(" (restarted)");
                }
                board.set_health(true, &detail);
                board.note_url(Some(base.clone()));
                disk::maybe_daily_sample();
                if waiting::on_waiting_page(&app) {
                    open_from_waiting(&app, &base);
                }
            }
            Err(_) => {
                let since = *failing_since.get_or_insert_with(std::time::Instant::now);
                url = None;
                board.note_url(None);
                board.set_health(false, "not answering");
                waiting::set(&app, waiting::unanswered(since));
            }
        }
        board.ensure_keepalive();
        waiting::nap(every);
    }
}

/// The daemon answers and the window still shows the waiting page: check
/// that Windows trusts the certificate (otherwise the webview lands on an
/// error page), wait for /desktop/ itself, then navigate.
fn open_from_waiting(app: &tauri::AppHandle, base: &str) {
    if health::cert(base) == Cert::Untrusted {
        waiting::set(app, Stage::Untrusted);
        return;
    }
    waiting::set(app, Stage::Opening);
    if await_desktop_ready(base) {
        waiting::navigate(app, base);
    }
}

pub(crate) fn restart_flow() {
    let Some(board) = board::get() else { return };
    let Some(distro) = board.distro() else {
        dialog::alert("PiCode", "The distro is not known yet — wait for the tray to come up.");
        return;
    };
    let user = resolve_user(&distro);
    let mut argv = vec!["-d".to_string(), distro];
    if let Some(user) = user {
        argv.push("-u".to_string());
        argv.push(user);
    }
    argv.extend(["--", "systemctl", "--user", "restart", "picode"].iter().map(|s| s.to_string()));
    let mut cmd = std::process::Command::new("wsl.exe");
    cmd.args(&argv);
    hide_console(&mut cmd);
    if let Err(e) = cmd.status() {
        dialog::alert("PiCode", &format!("Could not restart PiCode:\n{e}"));
        return;
    }
    // The next health tick reports the truth; until then say what is true.
    board.set_health(false, "restarting…");
    waiting::note_restart(board.app());
}

pub(crate) fn logs_flow() {
    let Some(board) = board::get() else { return };
    let Some(distro) = board.distro() else {
        dialog::alert("PiCode", "The distro is not known yet — wait for the tray to come up.");
        return;
    };
    // A log window is the one place a console is wanted, so this one
    // deliberately opens Windows Terminal instead of suppressing it.
    let user = resolve_user(&distro);
    let mut wt = vec!["wsl.exe".to_string(), "-d".to_string(), distro];
    if let Some(user) = user {
        wt.push("-u".to_string());
        wt.push(user);
    }
    wt.extend(["--", "journalctl", "--user", "-u", "picode", "-f"].iter().map(|s| s.to_string()));
    if std::process::Command::new("wt.exe").args(&wt).spawn().is_err() {
        let mut cmdline = vec!["/c".to_string(), "start".to_string(), String::new()];
        cmdline.extend(wt);
        let _ = std::process::Command::new("cmd").args(&cmdline).spawn();
    }
}

/// The Linux account the service runs as: what the distro logs in as. An
/// unreadable or root answer means "the distro default", which is what an
/// omitted -u already selects.
fn resolve_user(distro: &str) -> Option<String> {
    let mut cmd = std::process::Command::new("wsl.exe");
    cmd.args(["-d", distro, "--", "whoami"]);
    hide_console(&mut cmd);
    let out = cmd.output().ok()?;
    let name = console_string(&out.stdout);
    let name = name.trim();
    if name.is_empty() || name == "root" || name.chars().any(char::is_whitespace) {
        return None;
    }
    Some(name.to_string())
}

/// One command inside the distro, as the service's user, decoded from the
/// console's UTF-16. None when wsl.exe fails or the command exits non-zero.
pub(crate) fn wsl_output(distro: &str, argv: &[&str]) -> Option<String> {
    let mut cmd = std::process::Command::new("wsl.exe");
    cmd.args(["-d", distro]);
    if let Some(user) = resolve_user(distro) {
        cmd.args(["-u", &user]);
    }
    cmd.arg("--").args(argv);
    hide_console(&mut cmd);
    let out = cmd.output().ok()?;
    out.status.success().then(|| console_string(&out.stdout))
}

#[cfg(windows)]
pub(crate) fn hide_console(cmd: &mut std::process::Command) {
    use std::os::windows::process::CommandExt;
    const CREATE_NO_WINDOW: u32 = 0x0800_0000;
    cmd.creation_flags(CREATE_NO_WINDOW);
}

#[cfg(not(windows))]
pub(crate) fn hide_console(_cmd: &mut std::process::Command) {}

// open_management_window opens the Management page on demand — the second
// window of the shell: the WSL disk view, the cache prunes and the
// .wslconfig form, local pages, Rust commands behind them.
fn open_management_window(app: &tauri::AppHandle) {
    if let Some(win) = app.get_webview_window("management") {
        show(&win.as_ref().window());
        return;
    }
    // This runs on the tray's event thread. The health loop already knows
    // where PiCode answers, so the usual open costs no wsl.exe at all; only
    // before the first answer is the address looked up, and then on a
    // thread of its own — a slow WSL must not freeze the tray (the lookup
    // is hidden now, so a freeze would have nothing on screen to explain it).
    if let Some(base) = board::get().and_then(|b| b.url()) {
        if let Ok(url) = tauri::Url::parse(&base) {
            build_management_window(app, url);
            return;
        }
    }
    static LOOKING: std::sync::atomic::AtomicBool = std::sync::atomic::AtomicBool::new(false);
    if LOOKING.swap(true, std::sync::atomic::Ordering::SeqCst) {
        return; // a second click while the first lookup runs
    }
    let app = app.clone();
    std::thread::spawn(move || {
        let url = discover_server()
            .map(|(_, u)| u)
            .unwrap_or_else(|| tauri::Url::parse("https://localhost:8445/").expect("static origin"));
        daemon_acl::grant(&app, &url);
        let handle = app.clone();
        let posted = app.run_on_main_thread(move || {
            LOOKING.store(false, std::sync::atomic::Ordering::SeqCst);
            if let Some(win) = handle.get_webview_window("management") {
                show(&win.as_ref().window());
                return;
            }
            build_management_window(&handle, url);
        });
        if posted.is_err() {
            // The event loop is gone (shutting down); never leave the
            // latch set, or the item would stay dead for the process.
            LOOKING.store(false, std::sync::atomic::Ordering::SeqCst);
        }
    });
}

fn build_management_window(app: &tauri::AppHandle, mut url: tauri::Url) {
    url.set_path("/desktop/management.html");
    url.set_query(None);
    let _ = tauri::WebviewWindowBuilder::new(app, "management", WebviewUrl::External(url))
        .title("PiCode — Management")
        .inner_size(980.0, 860.0)
        .min_inner_size(720.0, 480.0)
        .decorations(false)
        .data_directory(browserlab::webview_profile())
        .build();
}

/// The restart gate: a daemon restart window can hold the port with a
/// listener that answers 404 for /desktop/, and the webview would park on
/// that dead body forever (2026-09-18: back-to-back deploys bricked the
/// resident until a manual restart). Wait — bounded — for a real 200; the
/// UI reloads itself on boot changes, so this gate only has to cover the
/// shell's navigations. False when the bound runs out: the waiting page
/// stays and the next tick tries again, instead of navigating onto the 404.
fn await_desktop_ready(base: &str) -> bool {
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(30);
    loop {
        if health::status_ok(base, "/desktop/") {
            return true;
        }
        if std::time::Instant::now() >= deadline {
            return false;
        }
        std::thread::sleep(std::time::Duration::from_millis(500));
    }
}

/// What the main window loads first: the bundled waiting page, always. It
/// costs no wsl.exe and no readiness wait on the thread that builds the
/// window, and the health loop navigates it to /desktop/ (waiting.rs).
fn waiting_target() -> WebviewUrl {
    WebviewUrl::App("waiting.html".into())
}

/// The main window's build, in one place: setup creates it, and Open PiCode
/// rebuilds it if it ever goes missing instead of silently no-op'ing
/// (2026-09-16: the logon task's --hidden launch left the resident with no
/// main window at all, and every Open path quietly did nothing).
fn build_main_window(
    app: &tauri::AppHandle,
    target: WebviewUrl,
) -> tauri::Result<tauri::Window> {
    // Undecorated window with a transparent chrome child: /desktop/ renders the
    // merged top row itself — brand, rail tabs, agent tabs and the
    // Windows caption buttons (ADR-0122). The shell only strips the
    // native frame.
    let win = tauri::window::WindowBuilder::new(app, "main")
        .title("PiCode")
        .inner_size(1360.0, 880.0)
        .min_inner_size(720.0, 480.0)
        .decorations(false)
        // One creation path for both starts: build visible, hide right after
        // when the logon task asked for --hidden (same setup turn, before
        // first paint, so sign-in stays pop-free). Measured 2026-09-16: at
        // logon the window ends up visible-but-parked off-screen either way;
        // what actually broke Open PiCode was the missed handle lookup, not
        // the build order.
        .visible(true)
        .build()?;
    win.add_child(
        tauri::webview::WebviewBuilder::new("main-content", target)
            .data_directory(browserlab::webview_profile())
            .disable_drag_drop_handler()
            .transparent(true)
            .initialization_script("window.__PICODE_LIVE_LAYERS__ = true;")
            .auto_resize(),
        tauri::LogicalPosition::new(0., 0.),
        win.inner_size()?.to_logical::<f64>(win.scale_factor()?),
    )?;
    disable_chrome_accelerators(app);

    // Remember the handle here, where both the first build and a rebuild
    // land: show_main drives this one, not a registry lookup that has been
    // observed to miss.
    let _ = MAIN_WINDOW.set(win.clone());
    // Close hides, exactly like the lab window below: without this
    // the X destroys "main", and tray click + Open PiCode silently
    // no-op on the missing window (2026-09-15: reopen dead in 0.3.0).
    {
        let hidden_main = win.clone();
        win.on_window_event(move |e| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = e {
                api.prevent_close();
                let _ = hidden_main.hide();
            }
        });
    }
    Ok(win)
}

/// WebView2's browser accelerators (Ctrl+R, F5, Ctrl+P, F12, …) fire on
/// whichever webview has focus. The chrome child is that webview whenever
/// the human is in PiCode UI — including a terminal pane — so Ctrl+R would
/// reload the whole app instead of reaching readline, and would reload
/// chrome instead of a work-browser page painted through a region hole.
/// Page webviews keep the engine default (on). Chrome handles reload in JS:
/// a visible work tab → `btab_reload`, else PiCode; a terminal is left alone.
fn disable_chrome_accelerators(app: &tauri::AppHandle) {
    let Some(wv) = app.get_webview("main-content") else {
        return;
    };

    let _ = wv.with_webview(|platform| unsafe {
        use webview2_com::Microsoft::Web::WebView2::Win32::ICoreWebView2Settings3;
        use windows::core::Interface;
        let Ok(core) = platform.controller().CoreWebView2() else {
            return;
        };
        let Ok(settings) = core.Settings() else {
            return;
        };
        let Ok(s3) = settings.cast::<ICoreWebView2Settings3>() else {
            return;
        };
        let _ = s3.SetAreBrowserAcceleratorKeysEnabled(false);
    });
}


fn show_main(app: &tauri::AppHandle) {
    // Open PiCode must never silently no-op. Order: the handle kept at build
    // time, then the registry, then a rebuild (which is what a resident that
    // somehow lost the window needs).
    if let Some(win) = MAIN_WINDOW.get() {
        show(win);
        return;
    }
    match app.get_window("main") {
        Some(win) => show(&win),
        None => match build_main_window(app, waiting_target()) {
            Ok(win) => show(&win),
            Err(e) => {
                eprintln!("open: cannot rebuild the main window ({e})");
                dialog::alert("PiCode", &format!("Could not open PiCode:\n{e}"));
            }
        },
    }
}

fn show(win: &tauri::Window) {
    let _ = win.unminimize();
    let _ = win.show();
    let _ = win.set_focus();
    // The Tauri calls can answer Ok and leave the window hidden — measured
    // 2026-09-16 on a resident started with `--hidden`: the native handle was
    // alive and ShowWindow brought it up instantly, while the app's own path
    // never did. Ask Windows directly as well; a window already visible costs
    // nothing here.
    if let Ok(hwnd) = win.hwnd() {
        use windows::Win32::UI::WindowsAndMessaging::{
            SetForegroundWindow, ShowWindow, SW_RESTORE, SW_SHOW,
        };
        unsafe {
            let _ = ShowWindow(hwnd, SW_RESTORE);
            let _ = ShowWindow(hwnd, SW_SHOW);
            let _ = SetForegroundWindow(hwnd);
        }
    }
}

/// discover_server finds the daemon the same way the Go tray did: the address
// lives in <data>/server.json inside the distro, and wsl.exe is the door to
// it. The first distro that answers wins — on every machine this product
// targets there is exactly one — and the winner's name comes back with the
// URL, because the keepalive must hold that same distro open.
fn discover_server() -> Option<(String, tauri::Url)> {
    // Every wsl.exe here runs without a console: the shell is a GUI app, so
    // a console child spawned plainly gets its own window — and this runs on
    // every Management open.
    let mut list = Command::new("wsl.exe");
    list.args(["--list", "--quiet"]);
    hide_console(&mut list);
    let out = list.output().ok()?;
    for distro in console_string(&out.stdout).lines().map(str::trim) {
        if distro.is_empty() {
            continue;
        }
        let mut cat = Command::new("wsl.exe");
        cat.args([
            "-d",
            distro,
            "--",
            "sh",
            "-lc",
            "cat \"$HOME/.picode/server.json\" 2>/dev/null",
        ]);
        hide_console(&mut cat);
        let out = cat.output().ok()?;
        let text = console_string(&out.stdout);
        if let Some(start) = text.find('{') {
            if let Ok(found) = serde_json::from_str::<ServerJson>(text[start..].trim_end()) {
                if let Ok(parsed) = tauri::Url::parse(&found.url) {
                    return Some((distro.to_string(), parsed));
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
