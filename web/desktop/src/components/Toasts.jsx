import { useEffect, useState } from "react";
import { Toaster } from "sonner";
import { readToastPrefs } from "../lib/toastPrefs.js";
import { resolvedTheme, readThemeMode } from "@picode/shared/domain/theme.js";

// useRailInset measures the Inspector rail so right-hand toasts step left
// of it instead of covering its header actions (ADR-0078). The rail keeps
// its width while shown and measures 0 when hidden, so the offset follows.
function useRailInset() {
  const [inset, setInset] = useState(0);
  useEffect(() => {
    const rail = document.getElementById("inspector");
    if (!rail || typeof ResizeObserver === "undefined") return undefined;
    const measure = () => setInset(rail.hidden ? 0 : Math.round(rail.getBoundingClientRect().width));
    const observer = new ResizeObserver(measure);
    observer.observe(rail);
    const attrs = new MutationObserver(measure);
    attrs.observe(rail, { attributes: true, attributeFilter: ["hidden", "style"] });
    measure();
    return () => { observer.disconnect(); attrs.disconnect(); };
  }, []);
  return inset;
}

export default function Toasts() {
  const [prefs, setPrefs] = useState(readToastPrefs);
  const [theme, setTheme] = useState(() => resolvedTheme(readThemeMode()));
  const railInset = useRailInset();

  useEffect(() => {
    function onPrefs() { setPrefs(readToastPrefs()); }
    function onTheme() { setTheme(resolvedTheme(readThemeMode())); }
    window.addEventListener("picode-toast-prefs", onPrefs);
    window.addEventListener("storage", onPrefs);
    const mo = new MutationObserver(onTheme);
    mo.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
    return () => {
      window.removeEventListener("picode-toast-prefs", onPrefs);
      window.removeEventListener("storage", onPrefs);
      mo.disconnect();
    };
  }, []);

  return (
    <Toaster
      theme={theme}
      position={prefs.position}
      offset={railInset && String(prefs.position || "").endsWith("-right") ? { right: railInset + 24 } : undefined}
      expand={prefs.expand}
      richColors={prefs.richColors}
      closeButton={prefs.closeButton}
      duration={prefs.duration}
      visibleToasts={prefs.visibleToasts}
      className={"picode-toaster close-" + prefs.closePlace}
      toastOptions={{ className: "picode-toast" }}
    />
  );
}
