import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import AttachComposer from "./AttachComposer.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { api } from "@picode/shared/client/api.js";
import { forkAgentSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { forkRequest, forkSlug, forkWorktreePath } from "@picode/shared/domain/forkAgent.js";

const json = (body) => ({ method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// How long the fork waits for git to make the worktree. The command runs in
// a visible terminal (ADR-0096); while another agent is writing the
// repository it is only typed there (ADR-0078) and waits for the person's
// Enter. Then the dialog steps aside — a modal over that terminal would hide
// the very key it asks for — and the wait goes on in the background.
const WORKTREE_WAIT_MS = 120_000;
const PREPARED_WAIT_MS = 600_000;

// ForkAgentDialog — Fork agent…: a new agent of the same CLI on a copy of
// this agent's conversation, with a task of its own. The source keeps
// running. The task composer is the terminal's attach composer, so the
// fork can start with photos, files and sketches. The default checkout is
// the field's "one worktree per agent session" (Conductor, Claude Squad —
// docs/benchmarks/2026-09-07-git-graph-write-actions.md §4), created in
// the open through the git door rather than behind the user's back.
//
// `deliverGit(command, root)` is App's git door (typeIntoTerminal with the
// run preference), resolving "ran" or "prepared": the dialog composes the
// worktree command, hands it over, and watches the graph for the folder, as
// "also start an agent here" does. `onBackground(name)` hides the dialog
// while a prepared command waits; `onBackgroundError` reports what goes
// wrong after that, when there is no dialog left to show it in.
export default function ForkAgentDialog({ open, agent, cliName, deliverGit, onClose, onDone, onBackground, onBackgroundError }) {
  const [name, setName] = useState("");
  const [where, setWhere] = useState("worktree");
  const [text, setText] = useState("");
  const [items, setItems] = useState([]);
  const [graph, setGraph] = useState(null); // null loading, false: not a repository
  const [phase, setPhase] = useState(""); // "" | worktree | waiting | starting
  const [error, setError] = useState("");
  const [nameError, setNameError] = useState("");
  const cancelled = useRef(false);
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

  async function makeWorktree(slug, run) {
    setPhase("worktree");
    const base = "/api/agents/" + encodeURIComponent(agent.id) + "/git";
    const composed = await api(base + "/compose", json({ action: "create-worktree-branch", target: "HEAD", name: slug, root: graph.root }));
    const outcome = await deliverGit(composed.command, graph.root);
    if (outcome === "prepared" && onBackground) {
      run.background = true;
      onBackground(name.trim());
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
    // The only field that can be wrong is the name: its message sits under it.
    if (!got.ok) { setNameError(got.error); return; }
    setNameError(""); setError("");
    // A backgrounded run outlives this dialog's state: it never writes to
    // it again (the dialog may be showing another agent by then).
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
        <Dialog.Content className="dlg dlg-handoff dlg-fork" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Fork {agent.name}</Dialog.Title>
          <Dialog.Description className="dlg-body">
            A new {cliName} agent starts on a copy of this conversation. {agent.name} keeps going.
          </Dialog.Description>

          <form className="fork-form" noValidate onSubmit={(e) => { e.preventDefault(); submit(); }}>
            <label className="fork-field">
              <span>Name</span>
              <input className="dlg-input" value={name} onChange={(e) => { setName(e.target.value); if (nameError) setNameError(""); }} maxLength={80} disabled={busy} autoComplete="off" spellCheck={false} aria-invalid={nameError ? "true" : undefined} aria-describedby={nameError ? "fork-name-error" : undefined} />
              {nameError ? <small id="fork-name-error" className="fork-field-error" role="alert">{nameError}</small> : null}
            </label>

            <fieldset className="handoff-options" disabled={busy}>
              <legend>Where</legend>
              <div className={"handoff-choices" + (graph === false ? " is-single" : "")}>
                {graph !== false ? (
                  <label className="handoff-choice">
                    <input type="radio" name="fork-where" value="worktree" checked={where === "worktree"} disabled={!graph} onChange={() => setWhere("worktree")} />
                    <span>
                      <strong>New worktree</strong>
                      <small>
                        {graph === null
                          ? "Checking the folder…"
                          : "A branch of its own from the last commit." + (uncommitted ? " " + uncommitted + (uncommitted === 1 ? "\u00a0uncommitted change stays" : "\u00a0uncommitted changes stay") + " with " + agent.name + "." : "")}
                      </small>
                    </span>
                  </label>
                ) : null}
                <label className="handoff-choice">
                  <input type="radio" name="fork-where" value="same" checked={where === "same"} onChange={() => setWhere("same")} />
                  <span>
                    <strong>Same folder</strong>
                    <small>{graph === false ? "This folder is not a git repository, so the fork shares it." : "Shares the files with " + agent.name + ". Best for looking, not editing."}</small>
                  </span>
                </label>
              </div>
            </fieldset>
          </form>

          <div className="fork-task">
            <span className="fork-task-label">Task</span>
            <AttachComposer
              className="attach-composer"
              text={text}
              setText={setText}
              items={items}
              setItems={setItems}
              onSubmit={submit}
              busy={busy}
              agentId={agent.id}
              placeholder="What should the fork do?"
              hideSend
              autoFocus={false}
            />
          </div>

          {status ? <p className="handoff-status" role="status"><PiSpinner /> {status}</p> : null}
          {error ? <p className="handoff-error" role="alert">{error}</p> : null}

          <div className="dlg-actions" data-align-row>
            <button type="button" className="btn btn-primary btn-sm" onClick={submit} disabled={busy || (where === "worktree" && !graph)}>
              {busy ? "Forking…" : "Fork"}
            </button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={close}>Cancel</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
