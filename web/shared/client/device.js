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

export function startPresence(client = document.documentElement.dataset.app) {
  const ping = () => {
    // Presence follows the mounted application, including explicit wide
    // mobile previews and narrow desktop views. Resizing never swaps apps.
    const host = client === "desktop";
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
