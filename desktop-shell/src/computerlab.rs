// computerlab.rs — the Computer lab (ADR-0148, M1): a local page in its own
// window that drives the computer tool's commands by hand, so the owner can
// accept capture, coordinates, input and the accessibility snapshot on a real
// Windows desktop before any agent is wired to them. Same shape as the
// Browser lab: built hidden at setup, shown from the tray, closing hides.
use tauri::{AppHandle, Manager, WebviewUrl, WebviewWindowBuilder};

pub fn init(app: &tauri::App) {
    let win = match WebviewWindowBuilder::new(app, "computerlab", WebviewUrl::App("computerlab.html".into()))
        .title("PiCode — Computer lab")
        .inner_size(1240.0, 900.0)
        .visible(false)
        .data_directory(crate::browserlab::webview_profile())
        .build()
    {
        Ok(w) => w,
        Err(e) => {
            eprintln!("computer lab: {e}");
            return;
        }
    };
    let hidden = win.clone();
    win.on_window_event(move |e| {
        if let tauri::WindowEvent::CloseRequested { api, .. } = e {
            api.prevent_close();
            let _ = hidden.hide();
        }
    });
}

pub fn open(app: &AppHandle) {
    if let Some(win) = app.get_webview_window("computerlab") {
        let _ = win.unminimize();
        let _ = win.show();
        let _ = win.set_focus();
    }
}

#[tauri::command]
pub fn computerlab_open(app: AppHandle) {
    open(&app);
}
