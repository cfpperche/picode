// Chrome/Edge (Android + desktop) expose beforeinstallprompt.
// iOS Safari/Chrome has no install API — Add to Home Screen only.

let deferred = null;
const listeners = new Set();

function notify() {
  for (const fn of listeners) fn(canInstall());
}

export function canInstall() {
  return !!deferred && !isStandalone();
}

export function isStandalone() {
  return window.matchMedia("(display-mode: standalone)").matches
    || window.navigator.standalone === true;
}

export function onInstallChange(fn) {
  listeners.add(fn);
  fn(canInstall());
  return () => listeners.delete(fn);
}

export async function promptInstall() {
  if (deferred) {
    deferred.prompt();
    const choice = await deferred.userChoice;
    deferred = null;
    notify();
    return { ok: choice.outcome === "accepted", reason: choice.outcome };
  }
  // iOS: no A2HS API. The share sheet is the closest official hook —
  // the user taps “Add to Home Screen” there.
  if (typeof navigator.share === "function") {
    try {
      await navigator.share({
        title: "PiCode",
        url: location.origin + "/?mobile=1",
      });
      return { ok: true, reason: "share" };
    } catch (e) {
      if (e && e.name === "AbortError") return { ok: false, reason: "dismissed" };
      return { ok: false, reason: "share-failed" };
    }
  }
  return { ok: false, reason: "unavailable" };
}

if (typeof window !== "undefined") {
  window.addEventListener("beforeinstallprompt", (e) => {
    e.preventDefault();
    deferred = e;
    notify();
  });
  window.addEventListener("appinstalled", () => {
    deferred = null;
    notify();
  });
}
