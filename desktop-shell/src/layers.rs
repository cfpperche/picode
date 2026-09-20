//! Trusted desktop chrome above live native pages (ADR-0161).
//! The chrome is a child WebView with a native region: page rectangles are
//! holes, floating HTML layers add their input/paint regions back. React
//! remains in one document; no site gets host UI or another IPC capability.
use serde::Deserialize;
use tauri::Manager;
use windows::Win32::Foundation::HWND;
use windows::Win32::Graphics::Gdi::{
    CombineRgn, CreateRectRgn, DeleteObject, SetWindowRgn, RGN_DIFF, RGN_OR,
};
use windows::Win32::UI::WindowsAndMessaging::{
    SetWindowPos, HWND_TOP, SWP_NOACTIVATE, SWP_NOMOVE, SWP_NOSIZE,
};

#[derive(Clone, Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Rect {
    pub left: f64,
    pub top: f64,
    pub right: f64,
    pub bottom: f64,
}

fn validate(rects: &[Rect]) -> Result<(), String> {
    picode_shell::layer_geometry::validate(
        &rects
            .iter()
            .map(|r| [r.left, r.top, r.right, r.bottom])
            .collect::<Vec<_>>(),
    )
    .map_err(str::to_owned)
}

pub fn raise(app: &tauri::AppHandle) {
    if let Some(view) = app.get_webview("main-content") {
        let _ = view.with_webview(|native| unsafe {
            let mut hwnd = HWND::default();
            if native.controller().ParentWindow(&mut hwnd).is_ok() {
                let _ = SetWindowPos(
                    hwnd,
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
}

#[tauri::command]
pub async fn chrome_layers(
    webview: tauri::Webview,
    pages: Vec<Rect>,
    overlays: Vec<Rect>,
    background: [u8; 3],
) -> Result<(), String> {
    // ACL and identity both apply. External page labels never control holes
    // in the application's trusted chrome, even on a trusted URL.
    if webview.label() != "main-content" {
        return Err("Not the desktop chrome".into());
    }
    validate(&pages)?;
    validate(&overlays)?;
    let window = webview.window();
    let size = window.inner_size().map_err(|e| e.to_string())?;
    let scale = window.scale_factor().map_err(|e| e.to_string())?;
    window
        .set_background_color(Some(tauri::window::Color(
            background[0],
            background[1],
            background[2],
            255,
        )))
        .map_err(|e| e.to_string())?;
    let (output, mut result) = tauri::async_runtime::channel(1);
    webview
        .with_webview(move |native| {
            let applied = (|| -> Result<(), String> {
                unsafe {
                    let mut hwnd = HWND::default();
                    native
                        .controller()
                        .ParentWindow(&mut hwnd)
                        .map_err(|e| e.to_string())?;
                    let region = CreateRectRgn(0, 0, size.width as i32, size.height as i32);
                    if region.0.is_null() {
                        return Err("Cannot allocate chrome region".into());
                    }
                    let edit = (|| -> Result<(), String> {
                        for (rects, op) in [(&pages, RGN_DIFF), (&overlays, RGN_OR)] {
                            for r in rects {
                                let [left, top, right, bottom] =
                                    picode_shell::layer_geometry::physical(
                                        [r.left, r.top, r.right, r.bottom],
                                        scale,
                                        size.width,
                                        size.height,
                                    );
                                let piece = CreateRectRgn(left, top, right, bottom);
                                if piece.0.is_null() {
                                    return Err("Cannot allocate layer region".into());
                                }
                                let combined =
                                    CombineRgn(Some(region), Some(region), Some(piece), op);
                                let _ = DeleteObject(piece.into());
                                if combined.0 == 0 {
                                    return Err("Cannot combine layer regions".into());
                                }
                            }
                        }
                        if SetWindowRgn(hwnd, Some(region), true) == 0 {
                            return Err("Cannot apply chrome region".into());
                        }
                        // Ownership of region transfers to Windows ONLY on success.
                        Ok(())
                    })();
                    if edit.is_err() {
                        let _ = DeleteObject(region.into());
                    }
                    edit?;
                    SetWindowPos(
                        hwnd,
                        Some(HWND_TOP),
                        0,
                        0,
                        0,
                        0,
                        SWP_NOMOVE | SWP_NOSIZE | SWP_NOACTIVATE,
                    )
                    .map_err(|e| e.to_string())?;
                    Ok(())
                }
            })();
            let _ = output.try_send(applied);
        })
        .map_err(|e| e.to_string())?;
    result.recv().await.ok_or("Native layer view closed")?
}
