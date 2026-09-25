import { useEffect, useRef, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import WorkspaceAttach from "./WorkspaceAttach.jsx";
import { IconClip, IconFile, IconImage, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { forkAgentSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { forkRequest, forkSlug, forkWorktreePath } from "@picode/shared/domain/forkAgent.js";
import { planAttachFiles, readAttachFile, MAX_ATTACH } from "@picode/shared/domain/termPrompt.js";
import { deliverGitCommand } from "../lib/gitDelivery.js";
import { toast } from "../lib/toast.js";

const json = (body) => ({ method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// The desktop dialog's waits (ForkAgentDialog.jsx): git gets two minutes
// to make the worktree, ten while its command waits for the person's Enter.
const WORKTREE_WAIT_MS = 120_000;
const PREPARED_WAIT_MS = 600_000;

// ForkAgentSheet — Fork agent… on the phone: the desktop dialog's flow in a
// sheet (ADR-0072). A new agent of the same CLI starts on a copy of this
// agent's conversation with a task of its own; the source keeps running.
// The worktree is made in the open through the git door (ADR-0096,
// lib/gitDelivery.js). When another agent is writing the repository the
// command is only typed (ADR-0078): the sheet closes, `onPrepared` opens
// that terminal for the Enter, and the fork follows in the background.
export default function ForkAgentSheet({ open, agent, cliName, terminals = [], workspaceId = "", onClose, onDone, onPrepared, onBackgroundError }) {
  const imgPick = useRef(null);
  const filePick = useRef(null);
  const cancelled = useRef(false);
  const [name, setName] = useState("");
  const [where, setWhere] = useState("worktree");
  const [text, setText] = useState("");
  const [items, setItems] = useState([]);
  const [pick, setPick] = useState(false);
  const [graph, setGraph] = useState(null); // null loading, false: not a repository
  const [phase, setPhase] = useState(""); // "" | worktree | waiting | starting
  const [error, setError] = useState("");
  const [nameError, setNameError] = useState("");
  const busy = !!phase;

  useEffect(() => {
    if (!open || !agent) return;
    cancelled.current = false;
    setName(agent.name + " fork");
    setText(""); setItems([]); setError(""); setNameError(""); setPhase(""); setGraph(null); setWhere("worktree");
    let live = true;
    api("/api/agents/" + encodeURIComponent(agent.id) + "/git?limit=1")
      .then((g) => { if (live) setGraph(g && g.root ? g : false); })
      .catch(() => { if (live) setGraph(false); });
    return () => { live = false; };
  }, [open, agent && agent.id]);

  // Not a repository: the only place left is the source's own folder.
  useEffect(() => { if (graph === false) setWhere("same"); }, [graph]);

  if (!agent) return null;
  const uncommitted = graph && graph.uncommitted ? graph.uncommitted.count || 0 : 0;

  function close() {
    cancelled.current = true;
    onClose();
  }

  async function addFiles(list) {
    const plan = planAttachFiles(list, items.length);
    if (plan.tooMany) toast.error("Up to 4 files.");
    if (plan.tooLarge) toast.error("Each file must be under 4 MB.");
    const next = items.slice();
    for (const f of plan.files) {
      try {
        const row = await readAttachFile(f);
        next.push({ id: (crypto.randomUUID && crypto.randomUUID()) || String(Date.now()) + next.length, ...row });
      } catch (err) {
        if (err && err.message === "too-large") toast.error("Each file must be under 4 MB.");
      }
    }
    setItems(next);
  }

  function addFromFolder(hit) {
    setPick(false);
    if (!hit || !hit.path) return;
    if (items.length >= MAX_ATTACH) { toast.error("Up to 4 files."); return; }
    setItems((cur) => cur.concat([{ id: (crypto.randomUUID && crypto.randomUUID()) || String(Date.now()), name: hit.name, path: hit.path }]));
  }

  async function makeWorktree(slug, run) {
    setPhase("worktree");
    const base = "/api/agents/" + encodeURIComponent(agent.id) + "/git";
    const composed = await api(base + "/compose", json({ action: "create-worktree-branch", target: "HEAD", name: slug, root: graph.root }));
    const sent = await deliverGitCommand({ owner: { kind: "agent", id: agent.id }, root: graph.root, command: composed.command, run: true, terminals, workspaceId, api });
    if (!sent.ran) {
      run.background = true;
      if (onPrepared) onPrepared(sent, name.trim());
    }
    const deadline = Date.now() + (run.background ? PREPARED_WAIT_MS : WORKTREE_WAIT_MS);
    for (let i = 0; Date.now() < deadline; i += 1) {
      if (cancelled.current && !run.background) return "";
      if (i === 4 && !run.background) setPhase("waiting");
      await sleep(1500);
      try {
        const found = forkWorktreePath(await api(base + "?limit=1"), slug);
        if (found) return found;
      } catch {
        /* a poll that fails is not an answer; the deadline still applies */
      }
    }
    throw new Error("The worktree did not appear. Check the git terminal, or fork into the same folder.");
  }

  async function submit() {
    if (busy) return;
    const got = parseForm(forkAgentSchema, { name, where });
    if (!got.ok) { setNameError(got.error); return; }
    setNameError(""); setError("");
    // A backgrounded run outlives this sheet's state: it never writes to it
    // again (the sheet may be showing another agent by then).
    const run = { background: false };
    try {
      let workPath = "";
      if (where === "worktree") {
        workPath = await makeWorktree(forkSlug(got.value.name, graph), run);
        if (!workPath) return;
      }
      if (!run.background) setPhase("starting");
      const res = await api("/api/agents/" + encodeURIComponent(agent.id) + "/fork-agent", json(forkRequest({ name: got.value.name, text, items, workPath })));
      if (run.background) { onDone(res); return; }
      if (cancelled.current) return;
      setPhase("");
      onDone(res);
    } catch (e) {
      if (run.background) {
        if (onBackgroundError) onBackgroundError(e);
        return;
      }
      setPhase("");
      setError(e && e.message ? e.message : String(e));
    }
  }

  const status = phase === "worktree"
    ? "Creating the worktree…"
    : phase === "waiting"
      ? "Waiting for git. If the command is waiting in the git terminal, press Enter there."
      : phase === "starting"
        ? "Starting the fork…"
        : "";

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) close(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-sheet dlg-fork" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Fork {agent.name}</Dialog.Title>
          <Dialog.Description className="dlg-body">
            A new {cliName} agent starts on a copy of this conversation. {agent.name} keeps going.
          </Dialog.Description>

          <form className="fork-form" noValidate onSubmit={(e) => { e.preventDefault(); submit(); }}>
            <label className="fork-field">
              <span>Name</span>
              <input className="dlg-input" value={name} onChange={(e) => { setName(e.target.value); if (nameError) setNameError(""); }} maxLength={80} disabled={busy} autoComplete="off" spellCheck={false} aria-invalid={nameError ? "true" : undefined} aria-describedby={nameError ? "m-fork-name-error" : undefined} />
              {nameError ? <small id="m-fork-name-error" className="fork-field-error" role="alert">{nameError}</small> : null}
            </label>

            <fieldset className="handoff-options" disabled={busy}>
              <legend>Where</legend>
              <div className={"handoff-choices" + (graph === false ? " is-single" : "")}>
                {graph !== false ? (
                  <label className="handoff-choice">
                    <input type="radio" name="m-fork-where" value="worktree" checked={where === "worktree"} disabled={!graph} onChange={() => setWhere("worktree")} />
                    <span>
                      <strong>New worktree</strong>
                      <small>
                        {graph === null
                          ? "Checking the folder…"
                          : "A branch of its own from the last commit." + (uncommitted ? " " + uncommitted + (uncommitted === 1 ? " uncommitted change stays" : " uncommitted changes stay") + " with " + agent.name + "." : "")}
                      </small>
                    </span>
                  </label>
                ) : null}
                <label className="handoff-choice">
                  <input type="radio" name="m-fork-where" value="same" checked={where === "same"} onChange={() => setWhere("same")} />
                  <span>
                    <strong>Same folder</strong>
                    <small>{graph === false ? "This folder is not a git repository, so the fork shares it." : "Shares the files with " + agent.name + ". Best for looking, not editing."}</small>
                  </span>
                </label>
              </div>
            </fieldset>

            <div className="fork-task">
              <span className="fork-task-label">Task</span>
              <WorkspaceAttach open={pick} agentId={agent.id} onPick={addFromFolder} onClose={() => setPick(false)} />
              {items.length ? (
                <div className="term-attach-chips">
                  {items.map((it) => (
                    <span key={it.id} className="pin-att composer-pic">
                      <span className="pin-att-face" title={it.name}>
                        {it.image && it.url ? <img src={it.url} alt="" /> : <span className="pin-att-ext">{extOf(it.name)}</span>}
                      </span>
                      <button type="button" className="pin-att-x" title="Remove" onClick={() => setItems((cur) => cur.filter((x) => x.id !== it.id))}><IconX size={12} /></button>
                    </span>
                  ))}
                </div>
              ) : null}
              <textarea
                className="dlg-input fork-task-input"
                rows={3}
                data-vaul-no-drag
                value={text}
                disabled={busy}
                onChange={(e) => setText(e.target.value)}
                placeholder="What should the fork do? Left empty, it opens waiting."
                aria-label="Task"
              />
              <input ref={imgPick} type="file" accept="image/png,image/jpeg,image/gif,image/webp,image/*" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
              <input ref={filePick} type="file" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
              <div className="term-attach-row" data-align-row>
                <button type="button" className="icon-btn composer-attach term-attach-choice" disabled={busy} aria-label="Attach image" onClick={() => imgPick.current && imgPick.current.click()}><IconImage /><span>Photo</span></button>
                <button type="button" className="icon-btn composer-attach term-attach-choice" disabled={busy} aria-label="Attach file" onClick={() => filePick.current && filePick.current.click()}><IconFile /><span>File</span></button>
                <button type="button" className="icon-btn composer-attach term-attach-choice" disabled={busy} aria-label="Attach from folder" onClick={() => setPick(true)}><IconClip /><span>Folder</span></button>
              </div>
            </div>
          </form>

          {status ? <p className="handoff-status" role="status">{status}</p> : null}
          {error ? <p className="handoff-error" role="alert">{error}</p> : null}

          <div className="dlg-actions" data-align-row>
            <button type="button" className="btn btn-ghost" onClick={close}>Cancel</button>
            <button type="button" className="btn btn-primary" onClick={submit} disabled={busy || (where === "worktree" && !graph)}>
              {busy ? "Forking…" : "Fork"}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function extOf(name = "") {
  const m = /\.([a-z0-9]{1,5})$/i.exec(name);
  return m ? m[1].toUpperCase() : "FILE";
}
