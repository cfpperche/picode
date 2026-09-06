import { useEffect } from "react";
import { KEYBOARD_INSET_THRESHOLD, shellVars } from "@picode/shared/domain/keyboardInset.js";

// Size #m-app to the visual viewport so a bottom composer or extra-keys
// row sits above the software keyboard (ADR-0044). iOS 26 overlays the
// IME and may scroll the document; a fixed shell cannot be scrolled away.
// Chrome Android often already resizes the layout — writing the same
// height is then a no-op.

export function useVisualViewport() {
  useEffect(() => {
    const el = document.getElementById("m-app");
    if (!el) return undefined;
    const vv = window.visualViewport;

    const apply = () => {
      const innerHeight = window.innerHeight;
      const vvHeight = vv ? vv.height : innerHeight;
      const vvOffsetTop = vv ? vv.offsetTop : 0;
      const scale = vv ? vv.scale : 1;
      let vkHeight = 0;
      if (navigator.virtualKeyboard && navigator.virtualKeyboard.boundingRect) {
        vkHeight = Math.round(navigator.virtualKeyboard.boundingRect.height || 0);
      }
      const vars = shellVars({ innerHeight, vvHeight, vvOffsetTop, scale });
      if (!vars) return;
      const inset = Math.max(vars.inset, vkHeight);
      el.style.setProperty("--vv-height", vars.height + "px");
      el.style.setProperty("--vv-offset-top", vars.offsetTop + "px");
      el.style.setProperty("--kb-inset", inset + "px");
      el.classList.toggle("kb-open", inset >= KEYBOARD_INSET_THRESHOLD || vkHeight >= KEYBOARD_INSET_THRESHOLD);
    };

    apply();
    if (vv) {
      vv.addEventListener("resize", apply);
      vv.addEventListener("scroll", apply);
    }
    window.addEventListener("resize", apply);
    let onVk = null;
    if (navigator.virtualKeyboard && navigator.virtualKeyboard.addEventListener) {
      onVk = apply;
      navigator.virtualKeyboard.addEventListener("geometrychange", onVk);
    }
    return () => {
      if (vv) {
        vv.removeEventListener("resize", apply);
        vv.removeEventListener("scroll", apply);
      }
      window.removeEventListener("resize", apply);
      if (onVk) navigator.virtualKeyboard.removeEventListener("geometrychange", onVk);
      el.style.removeProperty("--vv-height");
      el.style.removeProperty("--vv-offset-top");
      el.style.removeProperty("--kb-inset");
      el.classList.remove("kb-open");
    };
  }, []);
}
