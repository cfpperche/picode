// Shell picker: viewport at boot, not User-Agent (iPad "desktop site" lies).
// Escape hatches: ?desktop=1 / ?mobile=1 (sticky). No live swap on rotate
// — remounting would drop agent websockets.

export function pickShell() {
  const q = new URLSearchParams(location.search);
  if (q.get("desktop") === "1") {
    localStorage.setItem("picode-shell", "desktop");
    return "desktop";
  }
  if (q.get("mobile") === "1") {
    localStorage.setItem("picode-shell", "mobile");
    return "mobile";
  }
  const saved = localStorage.getItem("picode-shell");
  if (saved === "desktop" || saved === "mobile") return saved;
  return window.matchMedia("(max-width: 767px)").matches ? "mobile" : "desktop";
}

export function setShell(name) {
  localStorage.setItem("picode-shell", name);
  const u = new URL(location.href);
  u.searchParams.delete("desktop");
  u.searchParams.delete("mobile");
  u.searchParams.set(name === "mobile" ? "mobile" : "desktop", "1");
  location.assign(u.toString());
}
