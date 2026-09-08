import { toast as sonner } from "sonner";
import { humanizeError } from "@picode/shared/client/api.js";
import { noticeDuration, plainNotice, suppressNotice } from "@picode/shared/domain/notice.js";
import { readToastPrefs } from "./toastPrefs.js";
import { renderNotice } from "../components/Notice.jsx";

// One door for everything the app announces (study:
// docs/benchmarks/2026-09-07-superset-notifications.md).
//
//   notify(notice)          the model: actor, status, metadata, actions
//   toast(text, kind)       the legacy one-liner — unchanged for its
//                           317 call sites, now a notice with no actor
//
// Three policies come from the shared model and are applied here, once:
// how long a notice lives, whether the user is already looking at what it
// would announce, and what makes two notices the same one.

// What surface the user is looking at, for suppression. The route is the
// hash: a finished-notice targeting #/agent/a1 is redundant while that
// agent's conversation is on screen and the window has focus.
function currentSurface() {
  if (typeof document === "undefined") return null;
  return {
    target: typeof location !== "undefined" ? location.hash : "",
    focused: typeof document.hasFocus === "function" ? document.hasFocus() : true,
  };
}

export function notify(n) {
  if (!n) return null;
  const prefs = readToastPrefs();
  if (suppressNotice(n, currentSurface())) return null;
  const opts = { duration: noticeDuration(n, prefs.duration) };
  // One live notice per source: sonner replaces a toast that carries an
  // id it already shows, so a chatty agent updates its card instead of
  // stacking a new one against `visibleToasts`.
  if (n.key) opts.id = n.key;
  return sonner.custom((id) => renderNotice(n, id, prefs), opts);
}

export function toast(message, kind = "err") {
  return notify(plainNotice(message, kind));
}

toast.error = (m) => toast(m, "err");
toast.ok = (m) => toast(m, "ok");
toast.info = (m) => toast(m, "info");
toast.warn = (m) => toast(m, "warn");

export function toastError(err) {
  const raw = err && typeof err === "object" ? err.message : err;
  return toast(humanizeError(raw || "Something went wrong."), "err");
}
