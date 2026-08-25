import { useEffect, useState } from "react";
import { Toaster } from "sonner";
import { readToastPrefs } from "../lib/toastPrefs.js";
import { resolvedTheme, readThemeMode } from "../lib/theme.js";

export default function Toasts() {
  const [prefs, setPrefs] = useState(readToastPrefs);
  const [theme, setTheme] = useState(() => resolvedTheme(readThemeMode()));

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
      expand={prefs.expand}
      richColors={prefs.richColors}
      closeButton={prefs.closeButton}
      duration={prefs.duration}
      visibleToasts={prefs.visibleToasts}
      className="picode-toaster"
      toastOptions={{ className: "picode-toast" }}
    />
  );
}
