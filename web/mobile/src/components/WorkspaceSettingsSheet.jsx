import { useEffect, useRef, useState } from "react";
import * as Sheet from "./MobileSheet.jsx";
import { api } from "@picode/shared/client/api.js";
import { workspaceSettingsSchema } from "@picode/shared/contracts/schemas.js";
import { blocksLanding, cleanChecks, draftFrom, inheritedLine, integrationSave, integrationSource } from "@picode/shared/domain/workspaceSettings.js";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import LandingRulesFields from "./LandingRulesFields.jsx";
import { toast } from "../lib/toast.js";

// The phone's copy of the desktop's workspace Settings dialog (the apps share
// only web/shared): the name on the card and how delivered work lands in the
// project (ADR-0182), as one choice — follow the machine, or rules of its
// own. Save follows the decision table in workspaceSettings.js, like the
// desktop's. Opened from Preferences → Landing work.
export default function WorkspaceSettingsSheet({ ws, open, onClose }) {
  const [name, setName] = useState(ws ? ws.name : "");
  const [page, setPage] = useState(null);
  const [loadError, setLoadError] = useState("");
  const [mode, setMode] = useState("machine");
  const [draft, setDraft] = useState({ ffOnly: true, checks: [] });
  // { at: "name" | "landing", text } — the line sits under the part it is about.
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);
  const ownRef = useRef(null);
  const id = ws ? ws.id : "";

  function load() {
    setLoadError("");
    api("/api/delivery/integration?workspace=" + encodeURIComponent(id))
      .then((p) => {
        setPage(p);
        setMode(integrationSource(p) === "own" ? "own" : "machine");
        setDraft(draftFrom(p));
      })
      .catch((e) => setLoadError(e.message || "Could not read the settings."));
  }

  useEffect(() => {
    if (!open || !ws) return;
    setName(ws.name);
    setError(null);
    setPage(null);
    load();
    // Focus goes into the sheet — the sheet itself, not the Name field, so
    // the keyboard stays down (MobileSheet's rule) while a screen reader
    // lands inside the dialog instead of on the Edit button behind it.
    requestAnimationFrame(() => {
      const sheets = document.querySelectorAll(".dlg-wsset.dlg-sheet");
      const node = sheets[sheets.length - 1];
      if (node && !node.contains(document.activeElement)) node.focus({ preventScroll: true });
    });
    // Reopening always reads again: an agent may have changed the rules.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, id]);

  async function save() {
    const parsed = workspaceSettingsSchema.safeParse({ name, ffOnly: draft.ffOnly, checks: cleanChecks(draft.checks) });
    if (!parsed.success) {
      const issue = parsed.error.issues[0];
      setError({ at: issue.path[0] === "name" ? "name" : "landing", text: issue.message });
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
        await api("/api/workspaces/" + encodeURIComponent(id), {
          method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: parsed.data.name }),
        });
      }
      at = "landing";
      const q = "/api/delivery/integration?workspace=" + encodeURIComponent(id);
      if (plan.action === "delete") await api(q, { method: "DELETE" });
      if (plan.action === "put") {
        await api(q, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(plan.body) });
      }
      toast.ok("Settings saved.");
      onClose();
    } catch (e) {
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

  if (!ws) return null;
  const own = mode === "own";
  const nameErr = error && error.at === "name";
  return (
    <Sheet.Root open={open} onOpenChange={(o) => { if (!o && !busy) onClose(); }}>
      <Sheet.Portal>
        <Sheet.Overlay className="dlg-overlay" />
        <Sheet.Content className="dlg dlg-wsset" aria-describedby={"wsset-path-" + id}>
          <Sheet.Title className="dlg-title">Workspace settings</Sheet.Title>
          <Sheet.Description id={"wsset-path-" + id} className="dlg-body wsset-path">{shortPath(ws.path)}</Sheet.Description>
          <form noValidate onSubmit={(e) => { e.preventDefault(); void save(); }}>
            <fieldset className="wsset-fields" disabled={busy}>
              <label className="wsset-field">
                <span className="wsset-label">Name</span>
                <input
                  className="dlg-input"
                  aria-invalid={nameErr ? true : undefined}
                  aria-describedby={nameErr ? "wsset-name-err-" + id : undefined}
                  value={name}
                  maxLength={120}
                  autoComplete="off"
                  onChange={(e) => { setName(e.target.value); if (nameErr) setError(null); }}
                />
                {nameErr ? <span id={"wsset-name-err-" + id} className="form-error" role="alert">{error.text}</span> : null}
              </label>

              <section className="wsset-section" aria-labelledby={"wsset-int-" + id}>
                <h3 id={"wsset-int-" + id} className="wsset-label">Landing work</h3>
                <p className="wsset-help">When you authorize a branch an agent delivered, PiCode runs these checks and then merges it.</p>
                {loadError ? (
                  <p className="wsset-state" role="alert">{loadError} <button type="button" className="btn btn-sm" onClick={load}>Retry</button></p>
                ) : !page ? (
                  <div className="wsset-skel" aria-label="Loading settings"><div className="skel-line w-70" /><div className="skel-line w-50" /></div>
                ) : (
                  <>
                    <div className="wsset-seg" role="radiogroup" aria-label="Which rules apply">
                      <label className="wsset-seg-opt">
                        <input type="radio" name={"wsset-mode-" + id} checked={!own} onChange={() => setMode("machine")} />
                        <span className="wsset-seg-face">Same as this machine</span>
                      </label>
                      <label className="wsset-seg-opt">
                        <input ref={ownRef} type="radio" name={"wsset-mode-" + id} checked={own} onChange={() => { if (!own) setDraft(draftFrom(page)); setMode("own"); }} />
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
                      <LandingRulesFields draft={draft} setDraft={setDraft} idPrefix={"wsset-" + id} />
                    )}
                  </>
                )}
                {error && error.at === "landing" ? <p className="form-error wsset-error" role="alert">{error.text}</p> : null}
              </section>
            </fieldset>
            <div className="dlg-actions">
              <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={onClose}>Cancel</button>
              <button type="submit" className="btn btn-primary btn-sm" disabled={busy || (!page && !loadError)}>{busy ? "Saving…" : "Save"}</button>
            </div>
          </form>
        </Sheet.Content>
      </Sheet.Portal>
    </Sheet.Root>
  );
}
