#![windows_subsystem = "windows"]
//! Isolated native composition gate. No resident, daemon, tray, or production profile.
//! Build: examples/overlay-lab/build.sh
use std::sync::{
    atomic::{AtomicU8, Ordering},
    Arc,
};
use tauri::{webview::WebviewBuilder, LogicalPosition, LogicalSize, Manager, WebviewUrl};
use windows::Win32::Foundation::HWND;
use windows::Win32::UI::WindowsAndMessaging::{
    SetWindowPos, HWND_TOP, SWP_NOACTIVATE, SWP_NOMOVE, SWP_NOSIZE,
};

fn front(view: &tauri::Webview) {
    let _ = view.with_webview(|native| unsafe {
        let mut parent = HWND::default();
        if native.controller().ParentWindow(&mut parent).is_ok() {
            let _ = SetWindowPos(
                parent,
                Some(HWND_TOP),
                0,
                0,
                0,
                0,
                SWP_NOMOVE | SWP_NOSIZE | SWP_NOACTIVATE,
            );
        }
    });
}

fn layout(app: &tauri::AppHandle, mode: u8) {
    let Some(win) = app.get_window("live-overlay-lab") else {
        return;
    };
    let Ok(size) = win.inner_size() else { return };
    let scale = win.scale_factor().unwrap_or(1.0);
    let (width, height) = (size.width as f64 / scale, size.height as f64 / scale);
    if let Some(page) = app.get_webview("overlay-lab-page") {
        let _ = page.set_size(LogicalSize::new(width, (height - 70.).max(1.)));
    }
    if let Some(bar) = app.get_webview("overlay-lab-bar") {
        let _ = bar.set_size(LogicalSize::new(width, 70.));
    }
    if let Some(overlay) = app.get_webview("overlay-lab-overlay") {
        let modal = mode == 3;
        let _ = overlay.set_position(LogicalPosition::new(if modal { 0. } else { 160. }, 70.));
        let _ = overlay.set_size(LogicalSize::new(
            if modal {
                width
            } else {
                (width - 160.).clamp(1., 600.)
            },
            if modal { (height - 70.).max(1.) } else { 200. },
        ));
        front(&overlay);
    }
}

fn main() {
    // Lab-only debugging, bounded to a separate process and profile.
    std::env::set_var(
        "WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS",
        "--remote-debugging-port=19473",
    );
    tauri::Builder::default()
        .setup(|app| {
            let active = Arc::new(AtomicU8::new(0));
            let profile = std::env::temp_dir()
                .join(format!("picode-live-overlay-lab-{}", std::process::id()));
            let win = tauri::window::WindowBuilder::new(app, "live-overlay-lab")
                .title("PiCode — isolated live overlay lab")
                .inner_size(1000., 720.)
                .build()?;
            let page = WebviewBuilder::new(
                "overlay-lab-page",
                WebviewUrl::External("about:blank".parse()?),
            )
            .data_directory(profile.clone());
            let page = win.add_child(
                page,
                LogicalPosition::new(0., 70.),
                LogicalSize::new(1000., 650.),
            )?;
            page.eval(include_str!("overlay-lab/page.js"))?;
            let overlay = WebviewBuilder::new(
                "overlay-lab-overlay",
                WebviewUrl::External("about:blank".parse()?),
            )
            .data_directory(profile.clone())
            .transparent(true);
            let overlay = win.add_child(
                overlay,
                LogicalPosition::new(160., 70.),
                LogicalSize::new(600., 200.),
            )?;
            overlay.eval(include_str!("overlay-lab/overlay.js"))?;
            overlay.hide()?;
            let active_bar = active.clone();
            let bar = WebviewBuilder::new(
                "overlay-lab-bar",
                WebviewUrl::External("about:blank".parse()?),
            )
            .data_directory(profile)
            .on_document_title_changed(move |view, title| {
                // A lab-only signal from this fixture; never installed on the page.
                let mode = title.split(':').nth(1).unwrap_or("");
                let Some(overlay) = view.app_handle().get_webview("overlay-lab-overlay") else {
                    return;
                };
                if mode == "close" {
                    active_bar.store(0, Ordering::Relaxed);
                    let _ = overlay.hide();
                    return;
                }
                if mode != "suggest" && mode != "modal" && mode != "menu" {
                    return;
                }
                let kind = match mode {
                    "modal" => 3,
                    "menu" => 2,
                    _ => 1,
                };
                active_bar.store(kind, Ordering::Relaxed);
                layout(view.app_handle(), kind);
                let _ = overlay.eval(&format!("window.mode({mode:?})"));
                let _ = overlay.show();
                front(&overlay);
            });
            let bar = win.add_child(
                bar,
                LogicalPosition::new(0., 0.),
                LogicalSize::new(1000., 70.),
            )?;
            bar.eval(include_str!("overlay-lab/bar.js"))?;
            let handle = app.handle().clone();
            win.on_window_event(move |event| {
                if matches!(
                    event,
                    tauri::WindowEvent::Resized(_) | tauri::WindowEvent::ScaleFactorChanged { .. }
                ) {
                    layout(&handle, active.load(Ordering::Relaxed));
                }
            });
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("overlay lab failed");
}
