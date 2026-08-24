// Shell picker: viewport at boot, not User-Agent (iPad "desktop site" lies).
// Escape hatches: ?desktop=1 / ?mobile=1 (sticky). No live swap on rotate
// — remounting would drop agent websockets.

export function readShellPref() {
  const q = new URLSearchParams(location.search);
  if (q.get("desktop") === "1") return "desktop";
  if (q.get("mobile") === "1") return "mobile";
  const saved = localStorage.getItem("picode-shell");
  if (saved === "desktop" || saved === "mobile") return saved;
  return "system";
}

export function pickShell() {
  const pref = readShellPref();
  if (pref === "desktop" || pref === "mobile") {
    if (new URLSearchParams(location.search).get(pref) === "1") {
      localStorage.setItem("picode-shell", pref);
    }
    return pref;
  }
  return window.matchMedia("(max-width: 767px)").matches ? "mobile" : "desktop";
}

export function setShell(name) {
  const u = new URL(location.href);
  u.searchParams.delete("desktop");
  u.searchParams.delete("mobile");
  if (name === "desktop" || name === "mobile") {
    localStorage.setItem("picode-shell", name);
  } else {
    localStorage.removeItem("picode-shell");
  }
  location.assign(u.pathname + u.search + u.hash);
}
