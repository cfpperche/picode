// ADR-0072: explicit application paths win at every viewport size.
// The root launcher chooses by preference/viewport once per navigation.
export function resolveShell({ pathname = "/", search = "", saved = "", narrow = false } = {}) {
  const explicit = pathname.match(/^\/(desktop|mobile)(?:\/|$)/);
  if (explicit) return explicit[1];
  const query = new URLSearchParams(search);
  if (query.get("desktop") === "1") return "desktop";
  if (query.get("mobile") === "1") return "mobile";
  if (saved === "desktop" || saved === "mobile") return saved;
  return narrow ? "mobile" : "desktop";
}

function savedShell() {
  try { return localStorage.getItem("picode-shell") || ""; } catch { return ""; }
}

export function readShellPref() {
  const query = new URLSearchParams(location.search);
  if (query.get("desktop") === "1") return "desktop";
  if (query.get("mobile") === "1") return "mobile";
  const saved = savedShell();
  return saved === "desktop" || saved === "mobile" ? saved : "system";
}

export function pickShell() {
  const shell = resolveShell({ pathname: location.pathname, search: location.search, saved: savedShell(), narrow: window.matchMedia("(max-width: 767px)").matches });
  if (new URLSearchParams(location.search).get(shell) === "1") {
    try { localStorage.setItem("picode-shell", shell); } catch { /* Explicit paths also work without storage. */ }
  }
  return shell;
}

export function shellURL(href, shell) {
  const url = new URL(href);
  url.searchParams.delete("desktop");
  url.searchParams.delete("mobile");
  url.pathname = shell === "desktop" || shell === "mobile" ? "/" + shell + "/" : "/";
  return url.pathname + url.search + url.hash;
}

export function setShell(shell) {
  try {
    if (shell === "desktop" || shell === "mobile") localStorage.setItem("picode-shell", shell);
    else localStorage.removeItem("picode-shell");
  } catch { /* Navigation does not depend on persistent storage. */ }
  location.assign(shellURL(location.href, shell));
}
