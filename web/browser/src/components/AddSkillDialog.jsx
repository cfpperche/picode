import { useEffect, useState } from "react";
import ScopeIcon from "./ScopeIcon.jsx";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { skillSourceSchema } from "@picode/shared/contracts/schemas.js";
import { conflictChoices, criticalFindings, installBlocker, installLine, sizeLabel, sourceLabel } from "@picode/shared/domain/cliSkills.js";

// Add a skill (ADR-0196 slice 2): name a source, read what it carries —
// files, spec problems, the advisory scan — then install one skill with
// explicit consent. PiCode never vouches: the findings are shown, not judged.
// With an agent (slice 4), the skill can go to that agent alone: a copy in
// PiCode's cache its CLI receives at the next start.
// A candidate chosen for the person (a Marketplace card) is scrolled into the
// list once, so the checked radio is the one in view.
function revealPick(el) {
  if (el && !el.dataset.seen) {
    el.dataset.seen = "1";
    el.scrollIntoView({ block: "nearest" });
  }
}

export default function AddSkillDialog({ open, onClose, workspaceId = "", workspaceName = "", agentId = "", agentName = "", initialScope = "", initialSource = "", pickName = "", onDone }) {
  const fallback = initialScope === "agent" && agentId ? "agent" : initialScope === "machine" || !workspaceId ? "machine" : "workspace";
  const [step, setStep] = useState("source");
  const [source, setSource] = useState("");
  const [scope, setScope] = useState(fallback);
  // Each opening starts where the pane's chip is.
  useEffect(() => { if (open) setScope(fallback); }, [open, fallback]);
  useEffect(() => {
    if (open && initialSource) {
      setSource(initialSource);
      read(initialSource, pickName);
    }
    // Only a new opening reads; typing afterwards is the person's.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, initialSource, pickName]);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState(null);
  const [pick, setPick] = useState("");
  const [accepted, setAccepted] = useState(false);
  const [question, setQuestion] = useState(null);

  const reset = () => {
    setStep("source"); setSource(""); setErr(""); setBusy(false); setPreview(null); setPick(""); setAccepted(false); setQuestion(null);
  };
  const close = () => { reset(); onClose(); };

  const look = (e) => {
    e.preventDefault();
    read(source);
  };
  // A Marketplace card opens the dialog on its source and its skill: the
  // preview, the scan and the consent are the same as for a typed source.
  const read = async (value, want = "") => {
    const parsed = skillSourceSchema.safeParse({ source: value });
    if (!parsed.success) { setErr(parsed.error.issues[0].message); return; }
    setBusy(true); setErr("");
    try {
      const p = await api("/api/skills/preview", { method: "POST", body: JSON.stringify({ source: parsed.data.source }) });
      setPreview(p);
      const wanted = want ? p.candidates.find((c) => c.name === want) : null;
      setPick(wanted ? wanted.path : p.candidates.length === 1 ? p.candidates[0].path : "");
      setAccepted(false); setQuestion(null);
      setStep("review");
    } catch (x) {
      setErr(x.message);
    } finally {
      setBusy(false);
    }
  };

  const cand = preview?.candidates.find((c) => c.path === pick) || null;
  const blocker = installBlocker(cand, accepted);
  const critical = criticalFindings(cand);

  const install = async (extra = {}) => {
    if (!cand) return;
    setBusy(true); setErr(""); setQuestion(null);
    const body = { preview: preview.id, path: cand.path, scope, acceptCritical: accepted, ...extra };
    if (scope === "workspace") body.workspace = workspaceId;
    if (scope === "agent") body.agent = agentId;
    try {
      const res = await api("/api/skills", { method: "POST", body: JSON.stringify(body) });
      onDone(res, scope);
      close();
    } catch (x) {
      const code = x.body?.code || "";
      if (conflictChoices(code).length) setQuestion({ code, message: x.message });
      else setErr(x.message);
      setBusy(false);
    }
  };

  // Where it installs: on both steps, so a card that opens on the review
  // step still lets the person choose.
  const scopeChips = (
    <div className="pkg-scope skill-add-scope" role="radiogroup" aria-label="Where to install">
      <button type="button" className="pkg-scope-btn" role="radio" aria-checked={scope === "machine"} onClick={() => setScope("machine")}><ScopeIcon scope="machine" />Global</button>
      {workspaceId ? (
        <button type="button" className="pkg-scope-btn" role="radio" aria-checked={scope === "workspace"} onClick={() => setScope("workspace")}><ScopeIcon scope="workspace" />{workspaceName || "This workspace"}</button>
      ) : null}
      {agentId ? (
        <button type="button" className="pkg-scope-btn" role="radio" aria-checked={scope === "agent"} onClick={() => setScope("agent")}><ScopeIcon scope="agent" />{agentName || "This agent"}</button>
      ) : null}
    </div>
  );

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) close(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create skill-add" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Add a skill</Dialog.Title>
          <Dialog.Description className="dlg-body">
            {step === "source" ? "Name where it comes from. Nothing is installed until you have read it." : "Read what this source carries before you install it."}
          </Dialog.Description>

          {step === "source" ? (
            <form className="cred-form" noValidate onSubmit={look}>
              <label className="cred-field">
                <span>Source</span>
                <input
                  className="cred-input"
                  autoComplete="off"
                  spellCheck={false}
                  placeholder="owner/repo, a URL or /path"
                  aria-invalid={err ? true : undefined}
                  value={source}
                  onChange={(e) => { setErr(""); setSource(e.target.value); }}
                />
              </label>
              {scopeChips}
              <p className="skill-add-note">{installLine(scope, workspaceName, agentName)}</p>
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={close}>Cancel</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{busy ? <span className="cred-waiting">Reading the source</span> : "Look inside"}</button>
              </div>
            </form>
          ) : (
            <div className="skill-add-review">
              {preview.candidates.length > 1 ? (
                <fieldset className="skill-add-list">
                  <legend title={preview.source.input}>{preview.candidates.length} skills in {sourceLabel(preview.source)}</legend>
                  {preview.candidates.map((c) => (
                    <label key={c.path || c.name} className={"skill-add-cand" + (pick === c.path ? " is-on" : "")} ref={pick === c.path ? revealPick : undefined}>
                      <input type="radio" name="skill-pick" checked={pick === c.path} onChange={() => { setPick(c.path); setAccepted(false); setQuestion(null); setErr(""); }} />
                      <span className="skill-add-cand-name">{c.name}</span>
                      <span className="skill-add-cand-desc">{c.description || "No description"}</span>
                      {criticalFindings(c).length ? <span className="skill-add-flag is-critical" title="The safety scan noticed something to read before installing">{criticalFindings(c).length === 1 ? "1 scan warning" : criticalFindings(c).length + " scan warnings"}</span> : null}
                    </label>
                  ))}
                </fieldset>
              ) : null}

              {cand ? (
                <div className="skill-add-detail">
                  <p className="skill-add-head"><strong>{cand.name}</strong> · {cand.files.length} {cand.files.length === 1 ? "file" : "files"} · {sizeLabel(cand.size)}</p>
                  <p className="skill-add-desc">{cand.description}</p>
                  {cand.folder ? <p className="skill-add-note">Its folder at the source is called {cand.folder}; it installs as {cand.name}.</p> : null}
                  {(cand.problems || []).length ? <p className="skill-add-problem">{cand.problems.join("; ")}.</p> : null}
                  {(cand.findings || []).length ? (
                    <ul className="skill-add-findings" aria-label="Scan findings">
                      {cand.findings.map((f) => (
                        <li key={f.rule + f.file} className={"is-" + f.severity}>
                          <span className="skill-add-sev">{f.severity === "critical" ? "Critical" : "Warning"}</span> {f.file}{f.line ? ":" + f.line : ""} {f.text}
                        </li>
                      ))}
                    </ul>
                  ) : <p className="skill-add-clean">The scan found nothing. It reads patterns; it is not a review.</p>}
                  <details className="skill-add-files">
                    <summary>Files</summary>
                    <ul>{cand.files.map((f) => <li key={f}>{f}</li>)}</ul>
                  </details>
                  {scopeChips}
                  <p className="skill-add-note">{installLine(scope, workspaceName, agentName)}</p>
                  {critical.length ? (
                    <label className="skill-add-accept">
                      <input type="checkbox" checked={accepted} onChange={(e) => setAccepted(e.target.checked)} />
                      <span>I read its files and still want to install it.</span>
                    </label>
                  ) : null}
                </div>
              ) : null}

              {(preview.notes || []).map((n) => <p key={n} className="skill-add-note">{n}</p>)}
              {question ? (
                <div className="cli-notice" role="alert">
                  <span>{question.message}</span>
                </div>
              ) : null}
              {blocker && cand ? <p className="skill-add-blocker">{blocker}</p> : null}
              <p className="form-error" role="alert" hidden={!err}>{err}</p>
              <div className="dlg-actions" data-align-row data-align-wrap>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setStep("source"); setErr(""); setQuestion(null); }}>Back</button>
                {question ? conflictChoices(question.code).map((c) => (
                  <button key={c.label} type="button" className={"btn btn-sm " + (c.danger ? "btn-danger" : "btn-primary")} disabled={busy} onClick={() => install(c.body)}>{c.label}</button>
                )) : (
                  <button type="button" className="btn btn-primary btn-sm" disabled={busy || !!blocker} onClick={() => install()}>{busy ? <span className="cred-waiting">Installing</span> : scope === "agent" ? "Add to " + (agentName || "this agent") : "Install"}</button>
                )}
              </div>
            </div>
          )}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
