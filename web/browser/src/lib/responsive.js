// The device toolbar's pure half ("Responsive width", the native half the
// owner called on 2026-09-19): the preset ladder the strip renders, the zoom
// ladder its −/+ steps through, and the labels for both. The shell keeps its
// own copy of the preset list (`desktop-shell/src/responsive.rs`) for the
// bounds arithmetic and normalizes whatever arrives against it — the two
// lists must agree or the strip highlights a width the shell never stores.
// No CDP emulation lives here: mobile UA / touch / DPR is a later ADR.

// Preset widths, device-widths ascending; `name` is the strip button's
// tooltip (the reference's device classes at their common CSS widths).
export const PRESETS = [
  { width: 390, name: "Phone" },
  { width: 768, name: "Tablet" },
  { width: 1024, name: "Laptop" },
  { width: 1280, name: "Desktop" },
];

// The zoom ladder — the reference's own steps, 25% to 500% (WebView2's
// ZoomFactor limits; the shell clamps to the same range).
export const ZOOM_STEPS = [0.25, 0.33, 0.5, 0.67, 0.75, 0.8, 0.9, 1, 1.1, 1.25, 1.5, 1.75, 2, 2.5, 3, 4, 5];

// The step from `current` in `dir` (−1 out, +1 in), pinned to the ladder.
// A zoom between steps (an engine accelerator moved it) steps from the
// first rung at or above it, which is what the ⋮ menu's stepper always did.
export function stepZoom(current, dir) {
  const at = ZOOM_STEPS.findIndex((z) => z >= current - 0.001);
  const base = at < 0 ? ZOOM_STEPS.indexOf(1) : at;
  return ZOOM_STEPS[Math.min(ZOOM_STEPS.length - 1, Math.max(0, base + dir))];
}

// The strip's percent, as the zoom row prints it.
export function zoomLabel(factor) {
  const z = Number.isFinite(factor) ? factor : 1;
  return `${Math.round(z * 100)}%`;
}

// The active preset's row, for the strip's highlight: the shell stores
// `width: 0` for "toolbar up, full pane width" — nothing is highlighted.
export function activePreset(width) {
  if (!Number.isFinite(width) || width <= 0) return null;
  return PRESETS.find((p) => p.width === width) || null;
}

// The strip button's tooltip: the device class at its width.
export function presetTitle(preset) {
  return `${preset.name} · ${preset.width}px`;
}
