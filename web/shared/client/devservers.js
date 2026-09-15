// Dev servers on this machine. The panel polls one read; the daemon probes a
// page title once per port (never on every poll), so a dev server's own log
// does not fill with PiCode requests.
import { api } from "./api.js";

export async function listDevServers({ refresh = false, signal } = {}) {
  return api("/api/devservers" + (refresh ? "?refresh=1" : ""), { signal });
}

// isLoopbackUrl answers whether a URL points at this machine. Those are the
// addresses PiCode's own browser surface may show in a frame, and the ones a
// terminal link opens there instead of the system browser.
export function isLoopbackUrl(raw) {
  try {
    const u = new URL(String(raw || ""));
    if (u.protocol !== "http:" && u.protocol !== "https:") return false;
    const host = u.hostname.replace(/^\[|\]$/g, "").toLowerCase();
    return host === "localhost" || host === "::1" || host === "0.0.0.0" || host.endsWith(".localhost") || isIPv4Loopback(host);
  } catch {
    return false;
  }
}

// 127.0.0.0/8 — a dev server bound anywhere in the loopback range is still
// this machine (and the mixed-content carve-out covers the whole range).
function isIPv4Loopback(host) {
  const m = /^127\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(host);
  if (!m) return false;
  return m.slice(1).every((part) => Number(part) <= 255);
}
