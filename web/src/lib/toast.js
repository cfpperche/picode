import { humanizeError } from "./api.js";

let seq = 0;

export function toast(message, kind = "err") {
  const detail = { id: ++seq, message: String(message || ""), kind: kind || "err" };
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent("picode-toast", { detail }));
  }
  return detail.id;
}

toast.error = (m) => toast(m, "err");
toast.ok = (m) => toast(m, "ok");
toast.info = (m) => toast(m, "info");

export function toastError(err) {
  const raw = err && typeof err === "object" ? err.message : err;
  return toast(humanizeError(raw || "Something went wrong."), "err");
}
