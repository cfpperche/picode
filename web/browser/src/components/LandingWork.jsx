import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { workspaceSettingsSchema } from "@picode/shared/contracts/schemas.js";
import { cleanChecks, draftFrom, followsRows, integrationSave, machinePage } from "@picode/shared/domain/workspaceSettings.js";
import LandingRulesFields from "./LandingRulesFields.jsx";
import { toast } from "../lib/toast.js";

// Asks a workspace card to open its Settings dialog (WorkspaceMenu listens).
export const OPEN_WORKSPACE_SETTINGS = "picode:workspace-settings";

// Preferences → Landing work: the machine's integration rules (ADR-0182) —
// the fallback every workspace without rules of its own follows — and the
// list of who follows what. The machine layer has no "inherit": it declares
// or it does not, and with none an authorized branch stays blocked.
export default function LandingWork({ hidden, workspaces, workspacesLoaded = true }) {
  const [layers, setLayers] = useState(null);
  // The machine layer the draft was built from. Save compares against it and
  // sends its version, so a save racing someone else's is a 409 — never a
  // silent overwrite with the newer version the feed just brought in.
  const [base, setBase] = useState(null);
  const dirtyRef = useRef(false);
  const formRef = useRef(null);
  const setRulesRef = useRef(null);
  const [loadError, setLoadError] = useState("");
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState({ ffOnly: true, checks: [] });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback((keepDraft) => {
    setLoadError("");
    api("/api/delivery/integrations")
      .then((l) => {
        setLayers(l);
        // An untouched form follows the new rules; an edited one keeps its
        // base, so its Save meets the change as a conflict.
        if (!keepDraft || !dirtyRef.current) {
          setBase(l);
          setDraft(draftFrom(machinePage(l)));
          // A feed refresh of a clean form retires a conflict line: the rules
          // it pointed at are the ones now shown. A 409's own reload keeps it.
          if (keepDraft) setError("");
          setEditing((was) => (keepDraft ? was || !!l.machine : !!l.machine));
        }
      })
      .catch((e) => setLoadError(e.message || "Could not read the rules."));
  }, []);

  // Fresh on every visit; while open, a workspace saving its own rules (the
  // Edit buttons below) or an agent changing them refreshes the list without
  // touching an unsaved machine draft.
  useEffect(() => {
    if (hidden) return undefined;
    load(false);
    let timer;
    const unsub = subscribeFeed((e) => {
      if (e.type === "delivery.changed" || /^workspace\./.test(e.type)) {
        clearTimeout(timer); timer = setTimeout(() => load(true), 120);
      }
    });
    return () => { unsub(); clearTimeout(timer); };
  }, [hidden, load]);

  const page = base ? machinePage(base) : null;
  const plan = page ? integrationSave(page, "own", draft) : { action: "none" };
  const dirty = editing && plan.action !== "none";
  dirtyRef.current = dirty;

  // Focus follows the control that replaced the one just clicked.
  function focusForm() { requestAnimationFrame(() => formRef.current?.querySelector("input")?.focus()); }

  async function save() {
    const parsed = workspaceSettingsSchema.pick({ ffOnly: true, checks: true }).safeParse({ ffOnly: draft.ffOnly, checks: cleanChecks(draft.checks) });
    if (!parsed.success) { setError(parsed.error.issues[0].message); return; }
    if (plan.action !== "put") return;
    setBusy(true);
    setError("");
    try {
      await api("/api/delivery/integration", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(plan.body) });
      toast.ok("Rules saved.");
      load(false);
    } catch (e) {
      if (e.status === 409) {
        setError("Someone saved these rules while this was open. The current ones are shown now; make your change again.");
        load(false);
      } else {
        setError(e.message || "Could not save.");
      }
    } finally {
      setBusy(false);
    }
  }

  function discard() {
    setError("");
    setDraft(draftFrom(page));
    setEditing(!!base.machine);
    if (base.machine) focusForm();
    else requestAnimationFrame(() => setRulesRef.current?.focus());
  }

  const rows = layers ? followsRows(workspaces, layers) : [];
  return (
    <section className="settings-section landing-work" hidden={hidden}>
      <h3>Landing work</h3>
      <p className="settings-desc">Rules for every workspace that has none of its own. When you authorize a branch an agent delivered, PiCode runs these checks and then merges it.</p>
      {loadError ? (
        <p className="wsset-state" role="alert">{loadError} <button type="button" className="btn btn-ghost btn-sm" onClick={() => load(false)}>Retry</button></p>
      ) : !layers || !base ? (
        <div className="wsset-skel" aria-label="Loading rules"><div className="skel-line w-70" /><div className="skel-line w-50" /></div>
      ) : (
        <>
          {!editing ? (
            <p className="wsset-effect is-blocked landing-none">
              No rules yet: workspaces without their own rules will not land authorized branches.{" "}
              <button ref={setRulesRef} type="button" className="btn btn-ghost btn-sm" onClick={() => { setEditing(true); focusForm(); }}>Set rules</button>
            </p>
          ) : (
            <form ref={formRef} noValidate className="landing-form" onSubmit={(e) => { e.preventDefault(); void save(); }}>
              <fieldset className="wsset-fields landing-fields" disabled={busy}>
                <LandingRulesFields
                  draft={draft}
                  setDraft={setDraft}
                  idPrefix="landing-machine"
                  checksHelp="These run for every project that follows these rules, so use commands every project has — such as git diff --check."
                />
              </fieldset>
              {error ? <p className="form-error" role="alert">{error}</p> : null}
              {dirty || !base.machine ? (
                <div className="landing-actions">
                  <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={discard}>{base.machine ? "Discard" : "Cancel"}</button>
                  <button type="submit" className="btn btn-primary btn-sm" disabled={busy || plan.action !== "put"}>{busy ? "Saving…" : "Save"}</button>
                </div>
              ) : null}
            </form>
          )}

          <h4 className="settings-sub landing-follows-h">Who follows these rules</h4>
          {!workspacesLoaded ? (
            <div className="wsset-skel" aria-label="Loading workspaces"><div className="skel-line w-80" /><div className="skel-line w-70" /></div>
          ) : rows.length === 0 ? (
            <p className="wsset-empty">No workspaces yet.</p>
          ) : (
            <ul className="landing-follows">
              {rows.map((r) => (
                <li key={r.id} className={"landing-follow is-" + r.state}>
                  <span className="landing-follow-name" title={r.name}>{r.name}</span>
                  <span className="landing-follow-state">{r.text}</span>
                  <button type="button" className="btn btn-ghost btn-sm" aria-label={"Edit " + r.name} onClick={() => window.dispatchEvent(new CustomEvent(OPEN_WORKSPACE_SETTINGS, { detail: r.id }))}>Edit</button>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  );
}
