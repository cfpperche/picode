import { humanizeError } from "./api.js";

export function extractPiError(msg) {
  if (!msg) return "";
  if (typeof msg === "string") return msg;
  if (msg.errorMessage) return String(msg.errorMessage);
  if (typeof msg.error === "string") return msg.error;
  if (msg.error && typeof msg.error === "object") {
    return msg.error.message || msg.error.type || JSON.stringify(msg.error);
  }
  if (msg.finalError) return String(msg.finalError);
  if (msg.stopReason === "error") return "The model returned an error.";
  if (msg.stopReason === "aborted") return "Stopped.";
  return "";
}

export function alertFromPi(ev) {
  if (!ev) return null;
  if (ev.type === "auto_retry_start") {
    const n = ev.attempt || 1;
    const max = ev.maxAttempts || "?";
    return { level: "warn", text: "Retrying (" + n + "/" + max + ")… " + humanizeError(extractPiError(ev) || ev.errorMessage || "") };
  }
  if (ev.type === "auto_retry_end" && ev.success === false) {
    return { level: "error", text: humanizeError(ev.finalError || "Retry failed.") };
  }
  if (ev.type === "compaction_end" && !ev.aborted && !ev.result && ev.errorMessage) {
    return { level: "error", text: humanizeError(ev.errorMessage) };
  }
  if (ev.type === "extension_error") {
    return { level: "error", text: humanizeError(ev.error || "Extension error") };
  }
  if (ev.type === "message_end" || ev.type === "turn_end") {
    const m = ev.message || {};
    const raw = extractPiError(m);
    if (m.stopReason === "aborted") return { level: "warn", text: "Stopped." };
    if (m.stopReason === "error" || raw) {
      return { level: "error", text: humanizeError(raw || "The model returned an error.") };
    }
  }
  if (ev.type === "agent_end" && !ev.willRetry && Array.isArray(ev.messages)) {
    for (const m of ev.messages) {
      if (m && m.role === "assistant" && (m.stopReason === "error" || m.errorMessage)) {
        return { level: "error", text: humanizeError(extractPiError(m) || "The model returned an error.") };
      }
    }
  }
  return null;
}
