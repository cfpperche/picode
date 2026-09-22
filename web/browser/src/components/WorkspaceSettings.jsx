import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { workspaceSettingsSchema } from "@picode/shared/contracts/schemas.js";
import { cleanChecks, draftFrom, inheritedLine, integrationSave, integrationSource } from "@picode/shared/domain/workspaceSettings.js";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import { IconPlus, IconX } from "./Icons.jsx";
import { toast } from "../lib/toast.js";

const MAX_CHECKS = 8;

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
  const focusLast = useRef(false);
  const listRef = useRef(null);
  const [busy, setBusy] = useState(false);
  const nameRef = useRef(null);

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

  // A new check row takes the cursor, so Add check → type is one motion.
  useEffect(() => {
    if (!focusLast.current || !listRef.current) return;
    focusLast.current = false;
    const inputs = listRef.current.querySelectorAll("input");
    inputs[inputs.length - 1]?.focus();
  }, [draft.checks.length]);

  function setCheck(i, v) {
    setDraft((d) => ({ ...d, checks: d.checks.map((c, j) => (j === i ? v : c)) }));
  }

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
  const checks = draft.checks;
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
                <p className="wsset-help">How a branch an agent delivers gets merged into this project.</p>
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
                        <input type="radio" name={"wsset-mode-" + ws.id} checked={own} onChange={() => { if (!own) setDraft(draftFrom(page)); setMode("own"); }} />
                        <span className="wsset-seg-face">Only this workspace</span>
                      </label>
                    </div>
                    {!own ? (
                      <p className="wsset-effect">{inheritedLine(page)}</p>
                    ) : (
                      <div className="wsset-own">
                        <label className="dlg-choice wsset-choice">
                          <input type="checkbox" checked={draft.ffOnly} onChange={(e) => setDraft((d) => ({ ...d, ffOnly: e.target.checked }))} />
                          <span>Only land a branch that is up to date with the target <span className="wsset-muted">(fast-forward)</span></span>
                        </label>
                        <div className="wsset-checks">
                          <span className="wsset-sublabel">Checks that must pass first</span>
                          {checks.length === 0 ? (
                            <p className="wsset-empty">No checks: an approved branch lands right away.</p>
                          ) : (
                            <ol className="wsset-check-list" ref={listRef}>
                              {checks.map((c, i) => (
                                <li key={i} className="wsset-check">
                                  <input
                                    className="dlg-input wsset-check-input"
                                    value={c}
                                    autoComplete="off"
                                    spellCheck={false}
                                    placeholder={i === 0 ? "e.g. make ci" : "Another command"}
                                    aria-label={"Check " + (i + 1)}
                                    onChange={(e) => setCheck(i, e.target.value)}
                                  />
                                  <button type="button" className="ws-icon-btn wsset-check-remove" aria-label={"Remove check " + (i + 1)} title="Remove" onClick={() => setDraft((d) => ({ ...d, checks: d.checks.filter((_, j) => j !== i) }))}>
                                    <IconX size={13} />
                                  </button>
                                </li>
                              ))}
                            </ol>
                          )}
                          {checks.length < MAX_CHECKS ? (
                            <button type="button" className="btn btn-ghost btn-sm wsset-add" onClick={() => { focusLast.current = true; setDraft((d) => ({ ...d, checks: [...d.checks, ""] })); }}>
                              <IconPlus size={13} /> Add check
                            </button>
                          ) : null}
                        </div>
                      </div>
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
