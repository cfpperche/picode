//! Pure geometry for the computer tool (ADR-0148): the frame that maps the
//! model's coordinates — pixels of the last image it received — back to the
//! physical screen, and the fit that decides how large that image is. No
//! tauri, no Windows: `rustc --edition 2021 --test src/geometry.rs` runs the
//! tests on any host.

/// One returned image: where it came from on the virtual screen (physical
/// pixels, top-left origin) and the size it was scaled to. The model only
/// ever sees `img_w × img_h`; every coordinate it sends is in that space.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Frame {
    pub origin_x: i32,
    pub origin_y: i32,
    pub src_w: u32,
    pub src_h: u32,
    pub img_w: u32,
    pub img_h: u32,
}

impl Frame {
    /// Physical pixels per image pixel — 1.0 when nothing was scaled.
    pub fn scale(&self) -> f64 {
        if self.img_w == 0 {
            return 1.0;
        }
        self.src_w as f64 / self.img_w as f64
    }

    /// Maps an image point to the screen, aiming at the centre of the block
    /// of physical pixels that image pixel covers. None when the point lies
    /// outside the image: the model is pointing at something it never saw.
    pub fn to_physical(&self, x: i32, y: i32) -> Option<(i32, i32)> {
        if x < 0 || y < 0 || x as u32 >= self.img_w || y as u32 >= self.img_h {
            return None;
        }
        let s = self.scale();
        let px = self.origin_x + ((x as f64 + 0.5) * s).floor() as i32;
        let py = self.origin_y + ((y as f64 + 0.5) * s).floor() as i32;
        Some((px, py))
    }

    /// Maps a physical point into the image; None when it falls outside the
    /// frame (the cursor may sit on another monitor).
    pub fn to_image(&self, px: i32, py: i32) -> Option<(i32, i32)> {
        let rx = px - self.origin_x;
        let ry = py - self.origin_y;
        if rx < 0 || ry < 0 || rx as u32 >= self.src_w || ry as u32 >= self.src_h {
            return None;
        }
        let s = self.scale();
        Some(((rx as f64 / s).floor() as i32, (ry as f64 / s).floor() as i32))
    }

    /// The physical rectangle this frame covers, as (left, top, right, bottom).
    pub fn bounds(&self) -> (i32, i32, i32, i32) {
        (
            self.origin_x,
            self.origin_y,
            self.origin_x + self.src_w as i32,
            self.origin_y + self.src_h as i32,
        )
    }
}

/// The largest size that fits inside `max_w × max_h` keeping the aspect
/// ratio, never larger than the source: a small window is never blown up.
pub fn fit(src_w: u32, src_h: u32, max_w: u32, max_h: u32) -> (u32, u32) {
    if src_w == 0 || src_h == 0 {
        return (0, 0);
    }
    let mut w = src_w;
    let mut h = src_h;
    if w > max_w {
        h = ((h as u64 * max_w as u64 + w as u64 / 2) / w as u64) as u32;
        w = max_w;
    }
    if h > max_h {
        w = ((w as u64 * max_h as u64 + h as u64 / 2) / h as u64) as u32;
        h = max_h;
    }
    (w.max(1), h.max(1))
}

/// A frame for a whole source rectangle, scaled to fit.
pub fn frame_for(origin_x: i32, origin_y: i32, src_w: u32, src_h: u32, max_w: u32, max_h: u32) -> Frame {
    let (img_w, img_h) = fit(src_w, src_h, max_w, max_h);
    Frame { origin_x, origin_y, src_w, src_h, img_w, img_h }
}

/// The zoom action: a region `[x0, y0, x1, y1]` in the current image becomes
/// a new frame over the physical pixels underneath it, shown at up to 1:1.
/// None when the region is empty or outside the image.
pub fn zoom_frame(frame: &Frame, region: [i32; 4], max_w: u32, max_h: u32) -> Option<Frame> {
    let [x0, y0, x1, y1] = region;
    let (x0, x1) = (x0.min(x1), x0.max(x1));
    let (y0, y1) = (y0.min(y1), y0.max(y1));
    if x0 < 0 || y0 < 0 || x1 as u32 > frame.img_w || y1 as u32 > frame.img_h || x1 <= x0 || y1 <= y0 {
        return None;
    }
    let s = frame.scale();
    let px0 = frame.origin_x + (x0 as f64 * s).floor() as i32;
    let py0 = frame.origin_y + (y0 as f64 * s).floor() as i32;
    let px1 = frame.origin_x + (x1 as f64 * s).ceil() as i32;
    let py1 = frame.origin_y + (y1 as f64 * s).ceil() as i32;
    let src_w = (px1 - px0).max(1) as u32;
    let src_h = (py1 - py0).max(1) as u32;
    Some(frame_for(px0, py0, src_w, src_h, max_w, max_h))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fit_never_upscales_and_keeps_aspect() {
        assert_eq!(fit(1920, 1080, 1280, 1280), (1280, 720));
        assert_eq!(fit(800, 600, 1280, 1280), (800, 600));
        assert_eq!(fit(1080, 1920, 1280, 1280), (720, 1280));
        assert_eq!(fit(0, 10, 1280, 1280), (0, 0));
    }

    #[test]
    fn image_points_land_in_the_middle_of_their_physical_block() {
        let f = frame_for(2560, 0, 1920, 1080, 1280, 1280); // scale 1.5
        assert_eq!(f.scale(), 1.5);
        assert_eq!(f.to_physical(0, 0), Some((2560, 0)));
        assert_eq!(f.to_physical(1279, 719), Some((2560 + 1919, 1079)));
        assert_eq!(f.to_physical(412, 88), Some((2560 + 618, 132)));
        assert_eq!(f.to_physical(1280, 0), None, "outside the image");
        assert_eq!(f.to_physical(-1, 0), None);
    }

    #[test]
    fn physical_points_map_back_and_out_of_frame_is_none() {
        let f = frame_for(0, 0, 1920, 1080, 1280, 1280);
        assert_eq!(f.to_image(1919, 1079), Some((1279, 719)));
        assert_eq!(f.to_image(0, 0), Some((0, 0)));
        assert_eq!(f.to_image(-5, 10), None);
        assert_eq!(f.to_image(1920, 10), None);
        assert_eq!(f.bounds(), (0, 0, 1920, 1080));
    }

    #[test]
    fn zoom_covers_the_region_at_one_to_one_when_it_fits() {
        let f = frame_for(0, 0, 1920, 1080, 1280, 1280);
        let z = zoom_frame(&f, [100, 100, 300, 200], 1280, 1280).unwrap();
        assert_eq!((z.origin_x, z.origin_y), (150, 150));
        assert_eq!((z.src_w, z.src_h), (300, 150));
        assert_eq!((z.img_w, z.img_h), (300, 150), "no upscale, no downscale");
        assert_eq!(z.scale(), 1.0);
        assert!(zoom_frame(&f, [10, 10, 10, 50], 1280, 1280).is_none(), "empty region");
        assert!(zoom_frame(&f, [0, 0, 2000, 10], 1280, 1280).is_none(), "outside the image");
        // A swapped region is the same region.
        assert_eq!(zoom_frame(&f, [300, 200, 100, 100], 1280, 1280), Some(z));
    }
}
