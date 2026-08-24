import { api } from "./api.js";

export function deviceId() {
  let id = localStorage.getItem("picode-device-id");
  if (!id) {
    id = crypto.randomUUID();
    localStorage.setItem("picode-device-id", id);
  }
  return id;
}

export function startPresence() {
  const ping = () => {
    const host = location.hostname === "localhost" || location.hostname === "127.0.0.1";
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
