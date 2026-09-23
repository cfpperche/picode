import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { workspaceSettingsSchema } from "@picode/shared/contracts/schemas.js";
import { blocksLanding, cleanChecks, draftFrom, inheritedLine, integrationSave, integrationSource } from "@picode/shared/domain/workspaceSettings.js";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import LandingRulesFields from "./LandingRulesFields.jsx";
import { toast } from "../lib/toast.js";

// The workspace's own settings, opened from the card menu. Three parts: the
// name on the card, how a delivered branch lands in this project (the
// integration declaration, ADR-0182 — until now writable only by an agent or
// the CLI), and the way to the Communication page, which keeps its own
// surface. The integration part is one choice — follow the machine, or rules
// of its own — so the owner never has to think in layers.
export default function WorkspaceSettings({ ws, open, onClose, returnFocus }) {
  const [name, setName] = useState(ws.name);
  const [page, setPage] = useState(null); // GET /api/delivery/integration answer
  const [loadError, setLoadError] = useState("");
  const [mode, setMode] = useState("machine");
  const [draft, setDraft] = useState({ ffOnly: true, checks: [] });
  // { at: "name" | "landing", text } — the line sits under the part it is about.
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);
  const nameRef = useRef(null);
  const ownRef = useRef(null);

  function load() {
    setLoadError("");
    api("/api/delivery/integration?workspace=" + encodeURIComponent(ws.id))
      .then((p) => {
        setPage(p);
        setMode(integrationSource(p) === "own" ? "own" : "machine");
        setDraft(draftFrom(p));
      })
      .catch((e) => setLoadError(e.message || "Could not read the settings."));
  }

  useEffect(() => {
    if (!open) return;
    setName(ws.name);
    setError(null);
    setPage(null);
    load();
    // Reopening always reads again: an agent may have changed the rules.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, ws.id]);

  async function save() {
    const parsed = workspaceSettingsSchema.safeParse({ name, ffOnly: draft.ffOnly, checks: cleanChecks(draft.checks) });
    if (!parsed.success) {
      const issue = parsed.error.issues[0];
      setError({ at: issue.path[0] === "name" ? "name" : "landing", text: issue.message });
      if (issue.path[0] === "name") nameRef.current?.focus();
      return;
    }
    const plan = page ? integrationSave(page, mode, draft) : { action: "none" };
    const renamed = parsed.data.name !== ws.name;
    if (!renamed && plan.action === "none") { onClose(); return; }
    setBusy(true);
    setError(null);
    let at = "name";
    try {
      if (renamed) {
        await api("/api/workspaces/" + encodeURIComponent(ws.id), {
          method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: parsed.data.name }),
        });
      }
      at = "landing";
      const q = "/api/delivery/integration?workspace=" + encodeURIComponent(ws.id);
      if (plan.action === "delete") await api(q, { method: "DELETE" });
      if (plan.action === "put") {
        await api(q, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(plan.body) });
      }
      toast.ok("Settings saved.");
      onClose();
    } catch (e) {
      // A 409 means someone else saved first: show the current rules rather
      // than keep a form that would overwrite them, and say so.
      if (e.status === 409) {
        setError({ at, text: "Someone saved these rules while this was open. The current ones are shown now; make your change again." });
        load();
      } else {
        setError({ at, text: e.message || "Could not save." });
      }
    } finally {
      setBusy(false);
    }
  }

  const own = mode === "own";
  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o && !busy) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content
          className="dlg dlg-wsset"
          onOpenAutoFocus={(e) => { e.preventDefault(); nameRef.current?.focus(); }}
          // Back to the card's "…": the menu that opened this handed focus to
          // nothing, and on the phone focus would otherwise drop to <body>.
          onCloseAutoFocus={(e) => { if (returnFocus) { e.preventDefault(); returnFocus(); } }}
        >
          <Dialog.Title className="dlg-title">Workspace settings</Dialog.Title>
          <Dialog.Description className="dlg-body wsset-path" title={ws.path}>{shortPath(ws.path)}</Dialog.Description>
          <form noValidate onSubmit={(e) => { e.preventDefault(); void save(); }}>
            <fieldset className="wsset-fields" disabled={busy}>
              <label className="wsset-field">
                <span className="wsset-label">Name</span>
                <input ref={nameRef} className="dlg-input" aria-invalid={error && error.at === "name" ? true : undefined} aria-describedby={error && error.at === "name" ? "wsset-name-err-" + ws.id : undefined} value={name} maxLength={120} autoComplete="off" onChange={(e) => { setName(e.target.value); if (error && error.at === "name") setError(null); }} />
                {error && error.at === "name" ? <span id={"wsset-name-err-" + ws.id} className="form-error" role="alert">{error.text}</span> : null}
              </label>

              <section className="wsset-section" aria-labelledby={"wsset-int-" + ws.id}>
                <h3 id={"wsset-int-" + ws.id} className="wsset-label">Landing work</h3>
                <p className="wsset-help">When you authorize a branch an agent delivered, PiCode runs these checks and then merges it.</p>
                {loadError ? (
                  <p className="wsset-state" role="alert">{loadError} <button type="button" className="btn btn-ghost btn-sm" onClick={load}>Retry</button></p>
                ) : !page ? (
                  <div className="wsset-skel" aria-label="Loading settings"><div className="skel-line w-70" /><div className="skel-line w-50" /></div>
                ) : (
                  <>
                    <div className="wsset-seg" role="radiogroup" aria-label="Which rules apply">
                      <label className="wsset-seg-opt">
                        <input type="radio" name={"wsset-mode-" + ws.id} checked={!own} onChange={() => setMode("machine")} />
                        <span className="wsset-seg-face">Same as this machine</span>
                      </label>
                      <label className="wsset-seg-opt">
                        <input ref={ownRef} type="radio" name={"wsset-mode-" + ws.id} checked={own} onChange={() => { if (!own) setDraft(draftFrom(page)); setMode("own"); }} />
                        <span className="wsset-seg-face">Only this workspace</span>
                      </label>
                    </div>
                    {!own ? (
                      <p className={"wsset-effect" + (integrationSource(page) === "default" || blocksLanding(page.effective) ? " is-blocked" : "")}>
                        {inheritedLine(page)}
                        {integrationSource(page) === "default" ? (
                          <> <button type="button" className="wsset-inline-act" onClick={() => { setDraft(draftFrom(page)); setMode("own"); requestAnimationFrame(() => ownRef.current?.focus()); }}>Set rules for this workspace</button></>
                        ) : null}
                      </p>
                    ) : (
                      <LandingRulesFields draft={draft} setDraft={setDraft} idPrefix={"wsset-" + ws.id} />
                    )}
                  </>
                )}
                {error && error.at === "landing" ? <p className="form-error wsset-error" role="alert">{error.text}</p> : null}
              </section>

              <section className="wsset-section wsset-link-row">
                <div>
                  <h3 className="wsset-label">Communication</h3>
                  <p className="wsset-help">Who can talk to whom in this workspace.</p>
                </div>
                <a className="btn btn-ghost btn-sm" href={"#/clis/messages/" + encodeURIComponent("workspace:" + ws.id)} onClick={() => onClose()}>Open</a>
              </section>
            </fieldset>
            <div className="dlg-actions">
              <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={onClose}>Cancel</button>
              <button type="submit" className="btn btn-primary btn-sm" disabled={busy || (!page && !loadError)}>{busy ? "Saving…" : "Save"}</button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
