// capture.rs — pixels for the computer tool (ADR-0148): a rectangle of the
// screen or one window, scaled down and encoded through WIC. GDI is enough
// for v1: BitBlt with CAPTUREBLT for a screen area (layered windows
// included), PrintWindow for a window even when another one covers it. The
// windows GDI paints black (hardware-composed video, apps that opt out of
// capture) fall back to the screen rectangle; Graphics Capture is the M4
// upgrade. Every call runs on the desk thread, which has COM initialised.
use std::ffi::c_void;

use picode_shell::geometry;
use windows::core::Interface;
use windows::Win32::Graphics::Gdi::{
    BitBlt, CreateCompatibleBitmap, CreateCompatibleDC, DeleteDC, DeleteObject, GetDC, GetDIBits,
    ReleaseDC, SelectObject, BITMAPINFO, BITMAPINFOHEADER, BI_RGB, CAPTUREBLT, DIB_RGB_COLORS,
    HBITMAP, HDC, HGDIOBJ, ROP_CODE, SRCCOPY,
};
use windows::Win32::Graphics::Imaging::{
    CLSID_WICImagingFactory, GUID_ContainerFormatJpeg, GUID_ContainerFormatPng,
    GUID_WICPixelFormat24bppBGR, GUID_WICPixelFormat32bppBGRA, IWICBitmapFrameEncode,
    IWICBitmapSource, IWICImagingFactory, WICBitmapDitherTypeNone, WICBitmapEncoderNoCache,
    WICBitmapInterpolationModeFant, WICBitmapPaletteTypeCustom,
};
use windows::Win32::Storage::Xps::{PrintWindow, PRINT_WINDOW_FLAGS};
use windows::Win32::System::Com::{
    CoCreateInstance, IStream, CLSCTX_INPROC_SERVER, STATFLAG_NONAME, STATSTG, STREAM_SEEK_SET,
};
use windows::Win32::UI::Shell::SHCreateMemStream;
use windows::Win32::UI::WindowsAndMessaging::{
    DrawIconEx, GetCursorInfo, GetIconInfo, CURSORINFO, CURSOR_SHOWING, DI_NORMAL, HICON, ICONINFO,
};

/// PW_RENDERFULLCONTENT: ask DWM for the whole composed window, not just what
/// GDI painted. Undocumented but the value every capture tool uses; the
/// `windows` crate does not name it.
const PW_RENDERFULLCONTENT: PRINT_WINDOW_FLAGS = PRINT_WINDOW_FLAGS(2);

/// One capture: 32-bit BGRA rows, top-down, and where it came from.
pub struct Shot {
    pub bgra: Vec<u8>,
    pub w: u32,
    pub h: u32,
    pub origin_x: i32,
    pub origin_y: i32,
}

/// A memory DC with a bitmap of the requested size selected into it. Drop
/// puts everything back.
struct Surface {
    screen: HDC,
    mem: HDC,
    bmp: HBITMAP,
    old: HGDIOBJ,
    w: u32,
    h: u32,
}

impl Surface {
    fn new(w: u32, h: u32) -> Result<Self, String> {
        if w == 0 || h == 0 || w > 32_000 || h > 32_000 {
            return Err(format!("capture_failed: {w}×{h} is not a size worth capturing"));
        }
        unsafe {
            let screen = GetDC(None);
            if screen.0.is_null() {
                return Err("capture_failed: no screen device context".into());
            }
            let mem = CreateCompatibleDC(Some(screen));
            let bmp = CreateCompatibleBitmap(screen, w as i32, h as i32);
            if mem.0.is_null() || bmp.0.is_null() {
                let _ = DeleteDC(mem);
                let _ = ReleaseDC(None, screen);
                return Err("capture_failed: no memory for the bitmap".into());
            }
            let old = SelectObject(mem, HGDIOBJ(bmp.0));
            Ok(Surface { screen, mem, bmp, old, w, h })
        }
    }

    /// Reads the pixels out as top-down BGRA. The bitmap must not be selected
    /// while GetDIBits runs, so it is swapped out first (and the surface is
    /// spent afterwards).
    fn pixels(&mut self) -> Result<Vec<u8>, String> {
        let mut info = BITMAPINFO::default();
        info.bmiHeader = BITMAPINFOHEADER {
            biSize: std::mem::size_of::<BITMAPINFOHEADER>() as u32,
            biWidth: self.w as i32,
            biHeight: -(self.h as i32),
            biPlanes: 1,
            biBitCount: 32,
            biCompression: BI_RGB.0,
            ..Default::default()
        };
        let mut buf = vec![0u8; self.w as usize * self.h as usize * 4];
        unsafe {
            SelectObject(self.mem, self.old);
            let lines = GetDIBits(self.mem, self.bmp, 0, self.h, Some(buf.as_mut_ptr() as *mut c_void), &mut info, DIB_RGB_COLORS);
            if lines <= 0 {
                return Err("capture_failed: the pixels could not be read".into());
            }
        }
        Ok(buf)
    }
}

impl Drop for Surface {
    fn drop(&mut self) {
        unsafe {
            SelectObject(self.mem, self.old);
            let _ = DeleteObject(HGDIOBJ(self.bmp.0));
            let _ = DeleteDC(self.mem);
            let _ = ReleaseDC(None, self.screen);
        }
    }
}

// The cursor, drawn where it is, so the model sees what the human sees and
// can tell where its last move landed.
fn draw_cursor(dc: HDC, origin_x: i32, origin_y: i32) {
    unsafe {
        let mut ci = CURSORINFO { cbSize: std::mem::size_of::<CURSORINFO>() as u32, ..Default::default() };
        if GetCursorInfo(&mut ci).is_err() || ci.flags != CURSOR_SHOWING {
            return;
        }
        let icon = HICON(ci.hCursor.0);
        let mut ii = ICONINFO::default();
        let (hx, hy) = if GetIconInfo(icon, &mut ii).is_ok() {
            let _ = DeleteObject(HGDIOBJ(ii.hbmMask.0));
            let _ = DeleteObject(HGDIOBJ(ii.hbmColor.0));
            (ii.xHotspot as i32, ii.yHotspot as i32)
        } else {
            (0, 0)
        };
        let _ = DrawIconEx(dc, ci.ptScreenPos.x - origin_x - hx, ci.ptScreenPos.y - origin_y - hy, icon, 0, 0, 0, None, DI_NORMAL);
    }
}

/// A rectangle of the virtual screen, in physical pixels.
pub fn screen_rect(x: i32, y: i32, w: u32, h: u32, with_cursor: bool) -> Result<Shot, String> {
    let mut s = Surface::new(w, h)?;
    unsafe {
        BitBlt(s.mem, 0, 0, w as i32, h as i32, Some(s.screen), x, y, ROP_CODE(SRCCOPY.0 | CAPTUREBLT.0))
            .map_err(|e| format!("capture_failed: {e}"))?;
        if with_cursor {
            draw_cursor(s.mem, x, y);
        }
    }
    let bgra = s.pixels()?;
    Ok(Shot { bgra, w, h, origin_x: x, origin_y: y })
}

fn all_black(bgra: &[u8]) -> bool {
    // Every 97th pixel: a real frame has some light somewhere.
    bgra.chunks_exact(4).step_by(97).all(|p| p[0] == 0 && p[1] == 0 && p[2] == 0)
}

/// One window, composed by DWM, covered or not. `rect` is its outer rect
/// (what PrintWindow paints). Falls back to the screen rectangle when the
/// window declines to be printed.
pub fn window(id: isize, rect: (i32, i32, i32, i32)) -> Result<Shot, String> {
    let (l, t, r, b) = rect;
    let w = (r - l).max(0) as u32;
    let h = (b - t).max(0) as u32;
    let mut s = Surface::new(w, h)?;
    let ok = unsafe { PrintWindow(crate::desktop::hwnd(id), s.mem, PW_RENDERFULLCONTENT) }.as_bool();
    let bgra = if ok { s.pixels()? } else { Vec::new() };
    drop(s);
    if !ok || all_black(&bgra) {
        return screen_rect(l, t, w, h, false);
    }
    Ok(Shot { bgra, w, h, origin_x: l, origin_y: t })
}

/// Scales the shot to fit `max_w × max_h` (never up) and encodes it — PNG for
/// the model, JPEG for the small preview. Returns the bytes and the encoded
/// size.
pub fn encode(shot: &Shot, max_w: u32, max_h: u32, jpeg: bool) -> Result<(Vec<u8>, u32, u32), String> {
    let (tw, th) = geometry::fit(shot.w, shot.h, max_w, max_h);
    unsafe {
        let factory: IWICImagingFactory = CoCreateInstance(&CLSID_WICImagingFactory, None, CLSCTX_INPROC_SERVER)
            .map_err(|e| format!("capture_failed: WIC {e}"))?;
        let bitmap = factory
            .CreateBitmapFromMemory(shot.w, shot.h, &GUID_WICPixelFormat32bppBGRA, shot.w * 4, &shot.bgra)
            .map_err(|e| format!("capture_failed: bitmap {e}"))?;
        let source: IWICBitmapSource = if (tw, th) != (shot.w, shot.h) {
            let scaler = factory.CreateBitmapScaler().map_err(|e| format!("capture_failed: {e}"))?;
            scaler.Initialize(&bitmap, tw, th, WICBitmapInterpolationModeFant).map_err(|e| format!("capture_failed: scale {e}"))?;
            scaler.cast().map_err(|e| format!("capture_failed: {e}"))?
        } else {
            bitmap.cast().map_err(|e| format!("capture_failed: {e}"))?
        };
        // 24-bit BGR: no alpha in a screenshot, and both encoders take it.
        let conv = factory.CreateFormatConverter().map_err(|e| format!("capture_failed: {e}"))?;
        conv.Initialize(&source, &GUID_WICPixelFormat24bppBGR, WICBitmapDitherTypeNone, None, 0.0, WICBitmapPaletteTypeCustom)
            .map_err(|e| format!("capture_failed: convert {e}"))?;
        let stream: IStream = SHCreateMemStream(None).ok_or("capture_failed: no memory stream")?;
        let container = if jpeg { &GUID_ContainerFormatJpeg } else { &GUID_ContainerFormatPng };
        let encoder = factory.CreateEncoder(container, std::ptr::null()).map_err(|e| format!("capture_failed: encoder {e}"))?;
        encoder.Initialize(&stream, WICBitmapEncoderNoCache).map_err(|e| format!("capture_failed: {e}"))?;
        let mut frame: Option<IWICBitmapFrameEncode> = None;
        let mut props = None;
        encoder.CreateNewFrame(&mut frame, &mut props).map_err(|e| format!("capture_failed: {e}"))?;
        let frame = frame.ok_or("capture_failed: no frame")?;
        frame.Initialize(props.as_ref()).map_err(|e| format!("capture_failed: {e}"))?;
        frame.SetSize(tw, th).map_err(|e| format!("capture_failed: {e}"))?;
        let mut fmt = GUID_WICPixelFormat24bppBGR;
        frame.SetPixelFormat(&mut fmt).map_err(|e| format!("capture_failed: {e}"))?;
        frame.WriteSource(&conv, std::ptr::null()).map_err(|e| format!("capture_failed: write {e}"))?;
        frame.Commit().map_err(|e| format!("capture_failed: {e}"))?;
        encoder.Commit().map_err(|e| format!("capture_failed: {e}"))?;
        let mut stat = STATSTG::default();
        stream.Stat(&mut stat, STATFLAG_NONAME).map_err(|e| format!("capture_failed: {e}"))?;
        let size = stat.cbSize as usize;
        stream.Seek(0, STREAM_SEEK_SET, None).map_err(|e| format!("capture_failed: {e}"))?;
        let mut out = vec![0u8; size];
        let mut read = 0u32;
        let hr = stream.Read(out.as_mut_ptr() as *mut c_void, size as u32, Some(&mut read));
        if hr.is_err() && read == 0 {
            return Err(format!("capture_failed: encoded bytes unreadable ({hr})"));
        }
        out.truncate(read as usize);
        if out.is_empty() {
            return Err("capture_failed: the encoder produced nothing".into());
        }
        Ok((out, tw, th))
    }
}
