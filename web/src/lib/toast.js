import { toast as sonner } from "sonner";
import { humanizeError } from "./api.js";

export function toast(message, kind = "err") {
  const m = String(message || "");
  if (kind === "ok") return sonner.success(m);
  if (kind === "info") return sonner(m);
  return sonner.error(m);
}

toast.error = (m) => toast(m, "err");
toast.ok = (m) => toast(m, "ok");
toast.info = (m) => toast(m, "info");

export function toastError(err) {
  const raw = err && typeof err === "object" ? err.message : err;
  return toast(humanizeError(raw || "Something went wrong."), "err");
}
