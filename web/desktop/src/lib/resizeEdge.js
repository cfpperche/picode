import { useCallback, useRef, useState } from "react";

// One drag-to-resize idiom for side panels. The pure part is testable; the
// hook wires it to a sizer element. `edge` names the side of the PANEL the
// handle sits on: a handle on the panel's left edge (a right-hand rail) grows
// the panel as the pointer moves left.

export function clampWidth(value, min, max) {
  const v = Math.round(Number(value));
  if (!Number.isFinite(v)) return min;
  return Math.min(max, Math.max(min, v));
}

export function dragWidth(startWidth, startX, clientX, edge = "right") {
  const dx = clientX - startX;
  return edge === "left" ? startWidth - dx : startWidth + dx;
}

// stepWidth: keyboard resizing on a focused separator — arrows move the
// handle, so the panel grows toward the arrow's direction from its edge.
export function stepWidth(width, key, step = 20, edge = "right") {
  if (key !== "ArrowLeft" && key !== "ArrowRight") return null;
  const dir = key === "ArrowRight" ? 1 : -1;
  return edge === "left" ? width - dir * step : width + dir * step;
}

export function useEdgeResize({ width, min, max, edge = "right", onChange, onCommit }) {
  const [resizing, setResizing] = useState(false);
  const latest = useRef(width);
  const onPointerDown = useCallback((e) => {
    e.preventDefault();
    if (e.currentTarget.setPointerCapture) e.currentTarget.setPointerCapture(e.pointerId);
    const startX = e.clientX;
    const startW = width;
    latest.current = startW;
    setResizing(true);
    const move = (ev) => {
      latest.current = clampWidth(dragWidth(startW, startX, ev.clientX, edge), min, max);
      if (onChange) onChange(latest.current);
    };
    const stop = () => {
      setResizing(false);
      if (onCommit) onCommit(latest.current);
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", stop);
      window.removeEventListener("pointercancel", stop);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", stop);
    window.addEventListener("pointercancel", stop);
  }, [width, min, max, edge, onChange, onCommit]);
  const onKeyDown = useCallback((e) => {
    const next = stepWidth(width, e.key, 20, edge);
    if (next == null) return;
    e.preventDefault();
    if (onCommit) onCommit(clampWidth(next, min, max));
  }, [width, min, max, edge, onCommit]);
  return { resizing, onPointerDown, onKeyDown };
}
