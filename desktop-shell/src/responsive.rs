//! The pure half of the work browser's device toolbar — the "Responsive
//! width" native half (owner's call 2026-09-19: resizing the view's bounds
//! plus `ZoomFactor` needs no ADR; CDP device emulation — mobile UA, touch,
//! DPR — is a later ADR and is deliberately absent here). `btab.rs` drives
//! the native half on the UI thread; this module is everything that is
//! arithmetic, so it runs on any host
//! (`rustc --edition 2021 --test src/responsive.rs`).
//!
//! The shape: a tab's page normally fills the pane rect the UI reports
//! through `btab_bounds`. With a width picked, the page is a **centered,
//! narrower rect of the same pane** — same top, same height — so
//! width-based media queries fire and the rest of the pane stays ours.
//! Zoom is the controller's `ZoomFactor`, clamped to WebView2's documented
//! limits (0.25–5.0, see learn.microsoft.com/microsoft-edge/webview2/
//! reference/win32/icorewebview2controller#zoomfactor). Reset is the plain
//! inverse: full pane rect, zoom 1.0.

/// The preset widths the device toolbar offers, device-widths ascending.
/// The UI renders its buttons from this list (the JS mirror in
/// `web/browser/src/lib/responsive.js` carries the labels); both sides must
/// agree or the strip highlights a preset the shell never stores.
pub const PRESETS: &[f64] = &[390.0, 768.0, 1024.0, 1280.0];

/// A page narrower than this is unusable rather than responsive; a pane
/// smaller than the floor keeps its own width (the pane wins).
pub const MIN_WIDTH: f64 = 320.0;

/// WebView2's documented `ZoomFactor` limits: below 0.25 or above 5.0 the
/// runtime refuses `SetZoomFactor` outright (E_INVALIDARG), so every zoom
/// this feature applies is clamped first.
pub const ZOOM_MIN: f64 = 0.25;
pub const ZOOM_MAX: f64 = 5.0;

/// Reset's zoom: 100%, the controller's default.
pub const ZOOM_RESET: f64 = 1.0;

/// One tab's device-toolbar state: which preset width the page is narrowed
/// to (`0.0` = the toolbar is up but the page keeps the full pane width)
/// and the zoom recorded with it. The shell keeps this per tab id and rides
/// it back to the UI on `btab_meta`, so the strip survives tab switches and
/// route returns.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct Width {
    /// The active preset width in logical px; `0.0` = full pane width.
    pub w: f64,
    /// The zoom the toolbar last applied or inherited.
    pub zoom: f64,
}

impl Default for Width {
    fn default() -> Self {
        Width { w: 0.0, zoom: ZOOM_RESET }
    }
}

impl Width {
    /// True when the page is actually narrowed to a preset (not merely
    /// showing the toolbar).
    pub fn narrowed(&self) -> bool {
        self.w > 0.0
    }
}

/// A rectangle in the UI's logical pixels — the shape `btab_bounds` already
/// speaks.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct Rect {
    pub x: f64,
    pub y: f64,
    pub w: f64,
    pub h: f64,
}

impl Rect {
    pub fn new(x: f64, y: f64, w: f64, h: f64) -> Self {
        Rect { x, y, w, h }
    }
}

/// Snap a requested width onto the preset ladder — the nearest preset wins,
/// so a stale or hand-edited value can never leave the shell holding a width
/// the strip cannot highlight. Anything non-finite lands on the first
/// (narrowest) preset.
pub fn normalize_width(requested: f64) -> f64 {
    if !requested.is_finite() {
        return PRESETS[0];
    }
    let mut best = PRESETS[0];
    let mut dist = (requested - best).abs();
    for &p in &PRESETS[1..] {
        let d = (requested - p).abs();
        if d < dist {
            best = p;
            dist = d;
        }
    }
    best
}

/// The page's rect for a chosen width: centered in the pane, full height,
/// never wider than the pane and never narrower than `MIN_WIDTH` — unless
/// the pane itself is smaller, in which case the pane wins (a centered
/// 390 px page inside a 300 px pane would just crop itself off-pane).
pub fn width_rect(pane: Rect, width: f64) -> Rect {
    let max = pane.w.max(0.0);
    let floor = MIN_WIDTH.min(max);
    let w = normalize_width(width).min(max).max(floor);
    Rect {
        x: pane.x + (pane.w - w) / 2.0,
        y: pane.y,
        w,
        h: pane.h,
    }
}

/// Clamp a zoom factor into WebView2's legal range. NaN fails the range
/// checks both ways, so it resets instead of poisoning the controller.
pub fn clamp_zoom(factor: f64) -> f64 {
    if factor.is_nan() {
        return ZOOM_RESET;
    }
    factor.clamp(ZOOM_MIN, ZOOM_MAX)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn close(a: f64, b: f64) -> bool {
        (a - b).abs() < 1e-9
    }

    #[test]
    fn presets_are_ascending_device_widths() {
        assert!(PRESETS.len() >= 3);
        for pair in PRESETS.windows(2) {
            assert!(pair[0] < pair[1]);
        }
        assert!(*PRESETS.first().unwrap() >= MIN_WIDTH);
    }

    #[test]
    fn normalize_snaps_to_the_nearest_preset() {
        assert!(close(normalize_width(390.0), 390.0));
        assert!(close(normalize_width(400.0), 390.0));
        assert!(close(normalize_width(500.0), 390.0));
        assert!(close(normalize_width(600.0), 768.0)); // 210 from 390 vs 168 from 768
        assert!(close(normalize_width(900.0), 1024.0)); // 132 from 768 vs 124 from 1024
        assert!(close(normalize_width(1200.0), 1280.0));
    }

    #[test]
    fn normalize_falls_back_on_garbage() {
        assert!(close(normalize_width(f64::NAN), PRESETS[0]));
        assert!(close(normalize_width(f64::INFINITY), PRESETS[0]));
        assert!(close(normalize_width(-40.0), PRESETS[0]));
    }

    #[test]
    fn width_rect_centers_a_narrower_page() {
        let pane = Rect::new(100.0, 50.0, 1000.0, 700.0);
        let r = width_rect(pane, 390.0);
        assert!(close(r.w, 390.0));
        assert!(close(r.h, 700.0));
        assert!(close(r.y, 50.0));
        assert!(close(r.x, 100.0 + (1000.0 - 390.0) / 2.0));
    }

    #[test]
    fn width_rect_never_exceeds_the_pane() {
        let pane = Rect::new(0.0, 0.0, 500.0, 400.0);
        let r = width_rect(pane, 1280.0);
        assert!(close(r.w, 500.0));
        assert!(close(r.x, 0.0));
    }

    #[test]
    fn width_rect_keeps_a_floor_but_a_tiny_pane_wins() {
        // A 400 px pane can host 390 px centered; the floor does not apply.
        let small = Rect::new(10.0, 10.0, 400.0, 300.0);
        let r = width_rect(small, 390.0);
        assert!(close(r.w, 390.0));
        assert!(close(r.x, 15.0));

        // A 300 px pane is under the floor: the pane's width wins whole.
        let tiny = Rect::new(10.0, 10.0, 300.0, 300.0);
        let r = width_rect(tiny, 390.0);
        assert!(close(r.w, 300.0));
        assert!(close(r.x, 10.0));
    }

    #[test]
    fn degenerate_panes_stay_degenerate() {
        let r = width_rect(Rect::new(5.0, 5.0, 0.0, 100.0), 768.0);
        assert!(close(r.w, 0.0));
        assert!(close(r.x, 5.0));
    }

    #[test]
    fn zoom_stays_inside_webview2_limits() {
        assert!(close(clamp_zoom(0.1), ZOOM_MIN));
        assert!(close(clamp_zoom(1.5), 1.5));
        assert!(close(clamp_zoom(9.0), ZOOM_MAX));
        assert_eq!(clamp_zoom(f64::NAN), ZOOM_RESET);
        assert_eq!(ZOOM_MIN, 0.25);
        assert_eq!(ZOOM_MAX, 5.0);
    }

    #[test]
    fn reset_is_zoom_100_and_no_width() {
        assert_eq!(ZOOM_RESET, 1.0);
        let st = Width::default();
        assert!(!st.narrowed());
        let st = Width { w: 768.0, zoom: 2.0 };
        assert!(st.narrowed());
    }
}
