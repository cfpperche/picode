//! Geometry accepted by the trusted chrome bridge; runnable without Windows.
pub fn validate(rects: &[[f64; 4]]) -> Result<(), &'static str> {
    if rects.len() > 128 {
        return Err("Too many native layer regions");
    }
    for &[left, top, right, bottom] in rects {
        if [left, top, right, bottom]
            .iter()
            .any(|v| !v.is_finite() || v.abs() > 100_000.)
            || right < left
            || bottom < top
        {
            return Err("Invalid native layer region");
        }
    }
    Ok(())
}

pub fn physical(rect: [f64; 4], scale: f64, width: u32, height: u32) -> [i32; 4] {
    let [left, top, right, bottom] = rect;
    [
        (left * scale).floor().clamp(0., width as f64) as i32,
        (top * scale).floor().clamp(0., height as f64) as i32,
        (right * scale).ceil().clamp(0., width as f64) as i32,
        (bottom * scale).ceil().clamp(0., height as f64) as i32,
    ]
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn validates_empty_normal_offscreen_and_rejects_invalid_payloads() {
        for (rects, valid) in [
            (vec![], true),
            (vec![[0., 0., 10., 10.]], true),
            (vec![[-10., -10., 5., 5.]], true),
            (vec![[0., 0., 0., 0.]], true),
            (vec![[10., 0., 0., 10.]], false),
            (vec![[0., 10., 10., 0.]], false),
            (vec![[f64::NAN, 0., 10., 10.]], false),
            (vec![[0., 0., f64::INFINITY, 10.]], false),
            (vec![[0., 0., 100_001., 10.]], false),
            (vec![[0., 0., 10., 10.]; 129], false),
        ] {
            assert_eq!(validate(&rects).is_ok(), valid, "{rects:?}");
        }
    }
    #[test]
    fn scales_outwards_without_inverting_offscreen_regions() {
        for (rect, scale, expected) in [
            ([1., 1., 9., 9.], 1., [1, 1, 9, 9]),
            ([1., 1., 9., 9.], 1.25, [1, 1, 12, 12]),
            ([1., 1., 9., 9.], 1.5, [1, 1, 14, 14]),
            ([1., 1., 9., 9.], 2., [2, 2, 18, 18]),
            ([-10., -10., 5., 5.], 1.5, [0, 0, 8, 8]),
            ([200., 200., 300., 300.], 1.5, [100, 100, 100, 100]),
            ([-100., -100., -10., -10.], 1.5, [0, 0, 0, 0]),
        ] {
            assert_eq!(physical(rect, scale, 100, 100), expected);
        }
    }
}
