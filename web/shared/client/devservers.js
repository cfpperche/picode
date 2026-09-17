// Dev servers on this machine. The panel polls one read; the daemon probes a
// page title once per port (never on every poll), so a dev server's own log
// does not fill with PiCode requests. The three writes are the panel's verbs:
// stop a listener PiCode started, hide one it did not (or that the human is
// done looking at), and bring a hidden one back.
import { api } from "./api.js";

export async function listDevServers({ refresh = false, signal } = {}) {
  return api("/api/devservers" + (refresh ? "?refresh=1" : ""), { signal });
}

// stopDevServer ends one listener. The identity travels with the request — the
// port, the pid and the process's start token — so a row that went stale can
// never signal a process that reused the id. force is the escape hatch after a
// SIGTERM that did not land: the panel only offers it when a stop said so.
export async function stopDevServer({ port, pid, startKey, force = false } = {}) {
  return api("/api/devservers/stop", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ port, pid, startKey, force: !!force }),
  });
}

export async function hideDevServer({ port, pid, startKey } = {}) {
  return api("/api/devservers/hide", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ port, pid, startKey }),
  });
}

export async function unhideDevServer(hideId) {
  return api("/api/devservers/unhide", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ hideId: Number(hideId) }),
  });
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
