import { useEffect } from "react";
import { KEYBOARD_INSET_THRESHOLD, inputIsFocused, shellLayout } from "@picode/shared/domain/keyboardInset.js";
import { scheduleTermFit } from "@picode/shared/domain/termFit.js";
import { letterboxPx } from "../lib/layoutPrefs.js";
import { terms } from "../lib/terms.js";

// Pin #m-app to the visual viewport only while the software keyboard
// covers the page (ADR-0044). At rest the CSS 100dvh / top 0 stands, so
// iOS is not left with a black strip under the terminal.

export function useVisualViewport() {
  useEffect(() => {
    const el = document.getElementById("m-app");
    if (!el) return undefined;
    const vv = window.visualViewport;
    let restHeight = vv ? vv.height : window.innerHeight;

    const apply = () => {
      const innerHeight = window.innerHeight;
      const vvHeight = vv ? vv.height : innerHeight;
      const vvOffsetTop = vv ? vv.offsetTop : 0;
      const scale = vv ? vv.scale : 1;
      const focused = inputIsFocused(document.activeElement);
      if (!focused) restHeight = vvHeight;
      let vkHeight = 0;
      if (navigator.virtualKeyboard && navigator.virtualKeyboard.boundingRect) {
        vkHeight = Math.round(navigator.virtualKeyboard.boundingRect.height || 0);
      }
      const layout = shellLayout({
        innerHeight, vvHeight, vvOffsetTop, scale, restHeight, inputFocused: focused,
      });
      if (!layout) return;
      const open = layout.keyboardOpen || vkHeight >= KEYBOARD_INSET_THRESHOLD;
      if (!open) {
        el.style.removeProperty("--vv-height");
        el.style.removeProperty("--vv-offset-top");
        el.style.setProperty("--kb-inset", "0px");
        el.classList.remove("kb-open");
        document.documentElement.style.setProperty("--letterbox", letterboxPx(window.screen.height, innerHeight) + "px");
        return;
      }
      el.style.setProperty("--vv-height", layout.height + "px");
      el.style.setProperty("--vv-offset-top", layout.offsetTop + "px");
      el.style.setProperty("--kb-inset", Math.max(layout.inset, vkHeight) + "px");
      el.classList.add("kb-open");
      requestAnimationFrame(() => {
        for (const entry of terms.values()) scheduleTermFit(entry, true);
      });
    };

    apply();
    if (vv) {
      vv.addEventListener("resize", apply);
      vv.addEventListener("scroll", apply);
    }
    window.addEventListener("resize", apply);
    document.addEventListener("focusin", apply);
    document.addEventListener("focusout", apply);
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
      document.removeEventListener("focusin", apply);
      document.removeEventListener("focusout", apply);
      if (onVk) navigator.virtualKeyboard.removeEventListener("geometrychange", onVk);
      el.style.removeProperty("--vv-height");
      el.style.removeProperty("--vv-offset-top");
      el.style.removeProperty("--kb-inset");
      el.classList.remove("kb-open");
    };
  }, []);
}
