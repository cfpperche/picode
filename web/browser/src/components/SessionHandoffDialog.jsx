import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { handoffRequest, handoffSessionLabel, handoffSummaryLine } from "@picode/shared/domain/sessionHandoff.js";
import { sessionHandoffSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { askConfirm } from "../lib/confirm.js";
import { toast, toastError } from "../lib/toast.js";
import PiSpinner from "./PiSpinner.jsx";

const json = (body) => ({ method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

// SessionHandoffDialog (ADR-0088): "Continue in <CLI>". It previews what
// would travel (counts, what is left behind, whether the source is still
// live) through the read-only preview endpoint, re-previews on every
// choice, and only the primary action writes anything. The same plan
// serves preview and execution on the server, so what the dialog shows is
// what happens.
export default function SessionHandoffDialog({ open, session, sourceCli, sourceName, target, onClose, onDone }) {
  const [form, setForm] = useState({ to: target ? target.id : "", mode: "", landing: target && target.landing ? target.landing : target && target.landings && target.landings.length > 1 ? "agent" : "", window: "recent", tools: "native" });
  const [preview, setPreview] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [showBrief, setShowBrief] = useState(false);
  const seq = useRef(0);

  useEffect(() => {
    if (!open || !target) return;
    setForm({ to: target.id, mode: "", landing: target.landing || (target.landings && target.landings.length > 1 ? "agent" : ""), window: "recent", tools: "native" });
    setPreview(null); setError(""); setShowBrief(false);
  }, [open, target]);

  useEffect(() => {
    if (!open || !session || !form.to) return;
    const mine = ++seq.current;
    setLoading(true);
    const timer = setTimeout(async () => {
      try {
        const p = await api("/api/clis/" + encodeURIComponent(sourceCli) + "/sessions/handoff/preview", json(handoffRequest(session, form)));
        if (mine !== seq.current) return;
        setPreview(p); setError("");
        if (!form.mode && p.mode) setForm((f) => ({ ...f, mode: p.mode }));
      } catch (e) {
        if (mine !== seq.current) return;
        const modes = e && e.body && Array.isArray(e.body.modes) ? e.body.modes : [];
        if (e && e.status === 413 && modes.includes("brief") && form.mode !== "brief") {
          setError("");
          setForm((f) => ({ ...f, mode: "brief" }));
          return;
        }
        setPreview(null); setError((e && e.message) || "Could not prepare the handoff.");
      } finally {
        if (mine === seq.current) setLoading(false);
      }
    }, 150);
    return () => clearTimeout(timer);
  }, [open, session, sourceCli, form.to, form.mode, form.window, form.tools]);

  if (!target) return null;
  const modes = (preview && preview.modes) || target.modes || [];
  const canNative = modes.includes("native");
  const canBrief = modes.includes("brief");
  const tooLarge = !!(preview && preview.tooLarge);
  const mode = form.mode || (preview && preview.mode) || modes[0] || "";

  async function run(force) {
    const body = { ...form, mode };
    if (!body.landing) delete body.landing; // single-landing targets: the server's default decides
    const parsed = parseForm(sessionHandoffSchema, body);
    if (!parsed.ok) { toast(parsed.error || "Check the handoff options."); return; }
    setBusy(true);
    try {
      const res = await api("/api/clis/" + encodeURIComponent(sourceCli) + "/sessions/handoff", json(handoffRequest(session, parsed.value, { force })));
      onDone && onDone(res, target);
    } catch (e) {
      const live = e && e.body && e.body.live;
      if (!force && live && e.status === 409) {
        setBusy(false);
        const ok = await askConfirm({
          title: "Session still active",
          message: live.name + " is still writing to this session. Continue anyway? The newest turns may be missing.",
          confirmLabel: "Continue anyway",
        });
        if (ok) return run(true);
        return;
      }
      toastError(e);
    } finally {
      setBusy(false);
    }
  }

  const counts = preview && preview.counts;
  const manifest = preview && preview.manifest;
  const warnings = (manifest && manifest.warnings) || [];
  const shown = warnings.slice(0, 3);
  const landings = (target && target.landings) || [];
  const opensAgent = form.landing === "agent";

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o && !busy) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-handoff" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Continue in {target.name}</Dialog.Title>
          <Dialog.Description className="dlg-body">
            {sourceName} session {handoffSessionLabel(session)} continues {opensAgent ? "as a " + target.name + " agent in this app" : "in " + target.name + ", in a new terminal"}, in the same folder.
          </Dialog.Description>

          {error ? <p className="handoff-error" role="alert">{error}</p> : null}
          {loading && !preview ? <p className="handoff-status"><PiSpinner /> Preparing the handoff…</p> : null}

          {preview ? (
            <div className="handoff-summary" aria-live="polite">
              <p className="handoff-line">{handoffSummaryLine(counts, manifest) || "Nothing to carry over yet."}{loading ? <span className="handoff-refreshing"> · updating…</span> : null}</p>
              {shown.length ? (
                <ul className="handoff-warnings">
                  {shown.map((w, i) => <li key={i}>{w}</li>)}
                  {warnings.length > shown.length ? <li>and {warnings.length - shown.length} more</li> : null}
                </ul>
              ) : null}
              {preview.live ? <p className="handoff-live">{preview.live.name} is still using this session. You can continue anyway; the newest turns may be missing.</p> : null}
            </div>
          ) : null}

          {landings.length > 1 ? (
            <fieldset className="handoff-options" disabled={busy}>
              <legend>Where</legend>
              <div className="handoff-choices">
                <label className="handoff-choice" title="A stopped agent in this app; it needs no installed CLI.">
                  <input type="radio" name="handoff-landing" value="agent" checked={form.landing === "agent"} onChange={() => setForm((f) => ({ ...f, landing: "agent" }))} />
                  <span><strong>{target.name} agent · in the app</strong><small>The conversation opens as a stopped {target.name} agent here, not in a terminal.</small></span>
                </label>
                <label className="handoff-choice" title={target.installed ? "A terminal running the " + target.name + " CLI." : target.name + " is not installed."}>
                  <input type="radio" name="handoff-landing" value="terminal" checked={form.landing === "terminal"} disabled={!target.installed} onChange={() => setForm((f) => ({ ...f, landing: "terminal" }))} />
                  <span><strong>{target.name} CLI · in a terminal</strong><small>{target.installed ? "A new terminal starts " + target.name + " on this conversation." : target.name + " is not installed."}</small></span>
                </label>
              </div>
            </fieldset>
          ) : null}

          <fieldset className="handoff-options" disabled={busy}>
            <legend>How</legend>
            <div className="handoff-choices">
              <label className="handoff-choice" title="Written in the other CLI's own session format and resumed there.">
                <input type="radio" name="handoff-mode" value="native" checked={mode === "native"} disabled={!canNative} onChange={() => setForm((f) => ({ ...f, mode: "native" }))} />
                <span><strong>Native session</strong><small>{canNative ? "Turns and tool calls become a real " + target.name + " session." : tooLarge ? "This conversation is too large to import." : target.name + " cannot import sessions yet."}</small></span>
              </label>
              <label className="handoff-choice" title="A short summary the other CLI reads first.">
                <input type="radio" name="handoff-mode" value="brief" checked={mode === "brief"} disabled={!canBrief} onChange={() => setForm((f) => ({ ...f, mode: "brief" }))} />
                <span><strong>Brief</strong><small>{canBrief ? target.name + " starts from a short summary of this conversation." : target.name + " cannot start from a prompt."}</small></span>
              </label>
            </div>
          </fieldset>

          {preview && preview.hasCompaction ? (
            <fieldset className="handoff-options" disabled={busy}>
              <legend>How much</legend>
              <div className="handoff-choices">
                <label className="handoff-choice">
                  <input type="radio" name="handoff-window" value="recent" checked={form.window === "recent"} onChange={() => setForm((f) => ({ ...f, window: "recent" }))} />
                  <span><strong>Since the last summary</strong><small>What the previous agent still had in view, plus its summary of the rest.</small></span>
                </label>
                <label className="handoff-choice">
                  <input type="radio" name="handoff-window" value="all" checked={form.window === "all"} onChange={() => setForm((f) => ({ ...f, window: "all" }))} />
                  <span><strong>Whole conversation</strong><small>Every turn, including those before the summary.</small></span>
                </label>
              </div>
            </fieldset>
          ) : null}

          {mode === "native" && canNative ? (
            <label className="handoff-inline" title="Native keeps tool calls as tool calls; text turns them into plain messages, for a CLI that rejects unknown tools.">
              <input type="checkbox" checked={form.tools === "text"} disabled={busy} onChange={(e) => setForm((f) => ({ ...f, tools: e.target.checked ? "text" : "native" }))} />
              Turn tool calls into plain text
            </label>
          ) : null}

          {mode === "brief" && preview && preview.briefPreview ? (
            <div className="handoff-brief">
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setShowBrief((v) => !v)} aria-expanded={showBrief}>
                {showBrief ? "Hide brief" : "Show brief"} ({Math.round((preview.briefBytes || preview.briefPreview.length) / 1024 * 10) / 10} KB)
              </button>
              {showBrief ? <pre className="handoff-brief-text">{preview.briefPreview}</pre> : null}
            </div>
          ) : null}

          <div className="dlg-actions" data-align-row>
            <button type="button" className="btn btn-primary btn-sm" onClick={() => run(false)} disabled={busy || loading || !preview || !mode || (form.landing !== "agent" && !target.installed)} title={!target.installed && form.landing !== "agent" ? target.name + " is not installed" : ""}>
              {busy ? "Continuing…" : "Continue"}
            </button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} disabled={busy}>Cancel</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
