import { api } from "./api.js";
import { deviceId } from "./device.js";
import { isStandalone } from "./install.js";

// Web Push client (ADR-0047). The browser subscribes through its own push
// service with the server's VAPID key; the server keeps the endpoint and
// posts encrypted messages later. Everything that can fail before the
// permission prompt is reported as a reason, never as a silent no-op.

export function urlBase64ToUint8Array(b64) {
  const s = String(b64 || "").replace(/-/g, "+").replace(/_/g, "/");
  const padded = s + "=".repeat((4 - (s.length % 4)) % 4);
  const raw = atob(padded);
  const out = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out;
}

function isIOS(nav) {
  const n = nav || navigator;
  return /iPhone|iPad|iPod/.test(n.userAgent || "") || (n.platform === "MacIntel" && (n.maxTouchPoints || 0) > 1);
}

// pushBlockedReason: "" when a subscription can be attempted, else one
// line the settings row shows in place of the Enable button.
export function pushBlockedReason(env) {
  const e = env || { win: window, nav: navigator, standalone: isStandalone() };
  const { win, nav } = e;
  const secure = win.isSecureContext !== false && (win.location.protocol === "https:" || win.location.hostname === "localhost");
  if (!secure) return "Push needs HTTPS. Open PiCode through its https:// address.";
  if (!("serviceWorker" in nav) || !("PushManager" in win) || !("Notification" in win)) {
    if (isIOS(nav) && !e.standalone) return "On iPhone, add PiCode to the Home Screen first (Share → Add to Home Screen), then enable push from there.";
    return "This browser does not support Web Push.";
  }
  if (isIOS(nav) && !e.standalone) return "On iPhone, add PiCode to the Home Screen first (Share → Add to Home Screen), then enable push from there.";
  if (win.Notification.permission === "denied") return "Notifications are blocked for this site in the browser settings.";
  return "";
}

export function pushSupported() {
  return pushBlockedReason() === "";
}

async function registration() {
  const reg = await navigator.serviceWorker.getRegistration();
  if (reg) return reg;
  return navigator.serviceWorker.register("/sw.js");
}

export async function currentSubscription() {
  if (!("serviceWorker" in navigator)) return null;
  const reg = await navigator.serviceWorker.getRegistration();
  if (!reg) return null;
  return reg.pushManager.getSubscription();
}

function subJSON(sub) {
  const j = sub.toJSON ? sub.toJSON() : {};
  return { endpoint: sub.endpoint, keys: { p256dh: (j.keys || {}).p256dh, auth: (j.keys || {}).auth } };
}

// subscribePush: permission → browser subscription → server record.
export async function subscribePush(prefs) {
  const { publicKey } = await api("/api/push/vapid");
  const reg = await registration();
  const perm = await Notification.requestPermission();
  if (perm !== "granted") throw new Error("Notifications were not allowed.");
  let sub = await reg.pushManager.getSubscription();
  if (!sub) {
    sub = await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: urlBase64ToUint8Array(publicKey) });
  }
  const body = { ...subJSON(sub), deviceId: deviceId(), prefs: prefs || { actions: true, finished: true } };
  return api("/api/push/subscriptions", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
}

export async function unsubscribePush() {
  const sub = await currentSubscription();
  if (!sub) return;
  const endpoint = sub.endpoint;
  try { await sub.unsubscribe(); } catch { /* the server record is what matters */ }
  await api("/api/push/subscriptions", { method: "DELETE", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ endpoint }) }).catch(() => {});
}

export async function setPushPrefs(prefs) {
  const sub = await currentSubscription();
  if (!sub) throw new Error("Push is not enabled on this device.");
  return api("/api/push/subscriptions", { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ endpoint: sub.endpoint, prefs }) });
}

export async function sendTestPush() {
  const sub = await currentSubscription();
  if (!sub) throw new Error("Push is not enabled on this device.");
  return api("/api/push/test", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ endpoint: sub.endpoint }) });
}
