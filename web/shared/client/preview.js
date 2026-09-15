// HTML preview tickets (ADR-0136). An authenticated mint returns a capability
// URL the pane embeds in a sandboxed frame; the ticket is a credential, so it
// travels only in the URL and is never logged, fed to the feed, or shared.
import { api } from "./api.js";

function ownerPair(owner) {
  return { kind: (owner && owner.kind) || "", id: (owner && owner.id) || "" };
}

// mintPreview asks the daemon for a fresh ticket. The pane mints on every
// open and every Reload: the ticket is what makes the sandbox's relative
// assets servable, and re-minting renews the hour. The answer carries both
// ways to open it (ADR-0137): `sandbox` (the path form, always present) and
// `origin` (the ticket's own `<label>.localhost` origin, present only when
// the minting browser is on this machine's loopback).
export async function mintPreview(owner, path, root, signal) {
  const { kind, id } = ownerPair(owner);
  return api("/api/previews", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ kind, id, path: path || "", root: root || "" }),
    signal,
  });
}

// previewForms lists the ways one mint can be opened, best first: the
// ticket's own origin (storage, workers, its own cookie) then the sandboxed
// path form. An empty answer means the daemon offered nothing usable.
export function previewForms(mint) {
  const out = [];
  const pick = (form, mode) => {
    if (form && form.url) out.push({ mode, url: form.url, events: form.events || "" });
  };
  pick(mint && mint.origin, "origin");
  pick(mint && mint.sandbox, "sandbox");
  return out;
}

// previewReachable proves the ticket really serves before the iframe points
// at it, so a refusal is the pane's own line + Retry instead of the route's
// plain-text notice rendered inside the frame. A proxy that refuses HEAD
// (405) is not a refusal of the preview itself.
export async function previewReachable(url, signal) {
  const res = await fetch(url, { method: "HEAD", signal });
  if (res.status === 405) return true;
  return res.ok;
}

// putPreviewOverlay hands the pane's unsaved editor buffer to the ticket.
// The frame then renders what the editor holds while the project's relative
// assets keep resolving; the file on disk is not touched. Only the ticket's
// own document can be overlaid, so a page replacing its own preview gains
// nothing it did not already have.
export async function putPreviewOverlay(url, text) {
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "text/plain; charset=utf-8" },
    body: text == null ? "" : String(text),
  });
  if (!res.ok) {
    const err = new Error("Can't update this preview.");
    err.status = res.status;
    throw err;
  }
  return true;
}

// clearPreviewOverlay hands the document back to disk without touching the
// ticket. The origin form calls it on Save (ADR-0137 D5): the buffer and the
// file are equal at that moment, the preview's storage survives on the same
// origin, and no overlay is left to mask a later change on disk.
export async function clearPreviewOverlay(url) {
  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) {
    const err = new Error("Can't update this preview.");
    err.status = res.status;
    throw err;
  }
  return true;
}
