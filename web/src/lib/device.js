import { api } from "./api.js";

export function deviceId() {
  let id = localStorage.getItem("picode-device-id");
  if (!id) {
    // crypto.randomUUID needs a secure context; plain HTTP on a LAN or
    // tailnet address (PICODE_INSECURE) is not one. Fall back rather than
    // let the first effect throw and unmount the whole shell.
    id = typeof crypto.randomUUID === "function" ? crypto.randomUUID() : Array.from(crypto.getRandomValues(new Uint8Array(16)), (b) => b.toString(16).padStart(2, "0")).join("");
    localStorage.setItem("picode-device-id", id);
  }
  return id;
}

export function startPresence() {
  const ping = () => {
    // "host" = a desk browser (the desktop shell), whichever machine it is
    // on; the phone shell is never the desk. Loopback is no longer the
    // signal (ADR-0049: the daemon may live on another machine).
    const host = !/[?&]mobile=1/.test(location.search) && !window.matchMedia("(max-width: 767px)").matches;
    api("/api/devices/ping", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id: deviceId(), host }),
    }).catch(() => {});
  };
  ping();
  const t = setInterval(ping, 15000);
  return () => clearInterval(t);
}
