//! Build-only QA entry: no tray, resident, keepalive, or production profile.
use tauri::Manager;

pub fn run(url: &str) {
    let url: tauri::Url = url.parse().expect("QA URL");
    assert!(
        url.scheme() == "http"
            && matches!(url.host_str(), Some("localhost" | "127.0.0.1"))
            && url.port().is_some_and(|p| p != 8445),
        "QA requires a scratch loopback HTTP port"
    );
    let profile =
        std::env::temp_dir().join(format!("picode-overlay-product-{}", std::process::id()));
    crate::browserlab::set_qa_profile(profile);
    tauri::Builder::default()
        .manage(crate::btab::BtabState::default())
        .invoke_handler(tauri::generate_handler![
            crate::layers::chrome_layers,
            crate::btab::btab_navigate,
            crate::btab::btab_bounds,
            crate::btab::btab_visibility,
            crate::btab::btab_back,
            crate::btab::btab_forward,
            crate::btab::btab_reload,
            crate::btab::btab_meta,
            crate::btab::btab_close,
            crate::btab::btab_zoom,
            crate::btab::btab_set_zoom,
            crate::btab::btab_find,
            crate::btab::btab_set_prefs,
            crate::btab::btab_set_scripts,
            crate::btab::btab_set_permission_policy,
            crate::btab::btab_permission_answer,
            crate::btab::btab_set_developer_mode,
            crate::btab::btab_cdp_call,
            crate::btab::btab_cdp_events,
        ])
        .setup(move |app| {
            let win =
                crate::build_main_window(app.handle(), tauri::WebviewUrl::External(url.clone()))?;
            win.set_title("PiCode — overlay product QA")?;
            // Only this QA binary exposes developer tools through the explicit
            // PICODE_CDP_PORT opt-in read by main before this entry.
            let _ = app.get_webview("main-content");
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("overlay QA failed");
}
