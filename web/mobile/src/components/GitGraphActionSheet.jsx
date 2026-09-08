import { useEffect, useRef, useState } from "react";
import * as Sheet from "./MobileSheet.jsx";
import { api } from "@picode/shared/client/api.js";
import { FIELD_LABELS, LABELS } from "@picode/shared/domain/graphActions.js";
import { gitURL, shortHash } from "../lib/git/model.js";
import { doorHint, doors, gate, sheetGroups, sheetMenu } from "../lib/git/graphSheet.js";

// The sheet over what a history row, a commit or the repository itself can
// do (ADR-0096 on mobile, ADR-0095's own presentation). Two steps: the rows
// the shared module offers for the target, then one action's form. It
// composes nothing — the exact command comes from the server and is the
// preview — and delivers through the host's doors: the terminal (prepare or
// run when idle), or an agent or pi terminal asked in its own turn.

function shortTarget(t) {
  const s = String(t || "");
  return /^[0-9a-f]{40,64}$/i.test(s) ? s.slice(0, 7) : s;
}

function useDebounced(value, ms = 250) {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}

export default function GitGraphActionSheet({ open, owner, root, target, graph, ctx, run = false, blocked, onClose, onOpenTerminal, onAskAgent, onOpen, onDelivered }) {
  const [item, setItem] = useState(null);
  const [name, setName] = useState("");
  const [message, setMessage] = useState("");
  const [door, setDoor] = useState("prepare");
  const [typed, setTyped] = useState("");
  const [composed, setComposed] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const submitting = useRef(false);

  const menu = open ? sheetMenu(target, graph, ctx) : null;
  const groups = sheetGroups(menu);
  const doorList = doors(menu ? menu.occupants : []);
  const who = door.startsWith("ask:") ? (doorList.find((d) => d.value === door) || {}).who || null : null;

  useEffect(() => {
    if (!open) return;
    setItem(null); setName(""); setMessage(""); setTyped(""); setComposed(null); setError(""); setNotice("");
    setDoor(run ? "run" : "prepare");
  }, [open, target, run]);

  const wantsName = !!(item && item.needs && item.needs.includes("name"));
  const wantsMessage = !!(item && item.needs && item.needs.includes("message"));
  const debouncedName = useDebounced(name);
  const debouncedMessage = useDebounced(message);
  useEffect(() => {
    if (!open || !item || !owner) return undefined;
    if ((wantsName && !debouncedName.trim()) || (wantsMessage && !debouncedMessage.trim())) { setComposed(null); return undefined; }
    let live = true;
    api(gitURL(owner, "git/compose"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: item.action, target: item.target || "", name: debouncedName.trim(), message: debouncedMessage, root }),
    }).then((res) => { if (live) { setComposed(res); setError(""); } })
      .catch((e) => { if (live) { setComposed(null); setError(e?.message || "That command cannot be composed."); } });
    return () => { live = false; };
  }, [open, item, owner, root, debouncedName, debouncedMessage, wantsName, wantsMessage]);

  const g = composed && !who ? gate({ tier: composed.tier, door, action: item.action, target: item.target || "", name: name.trim(), branch: composed.branch }) : { typed: "" };
  const ready = !!composed && (!g.typed || typed.trim() === g.typed) && !busy && !blocked;
  const preview = composed ? (who ? composed.prompt : composed.command) : "";

  const close = () => { if (!submitting.current) onClose(); };
  const pick = (row) => {
    if (row.kind === "copy") {
      navigator.clipboard?.writeText(row.value).then(() => setNotice("Copied.")).catch(() => setError("Clipboard blocked — copy it by hand."));
      return;
    }
    if (row.kind === "open-agent") { onOpen?.("agent", row.agentId); onClose(); return; }
    if (row.kind === "open-terminal") { onOpen?.("term", row.terminalId); onClose(); return; }
    if (row.kind === "action") { setItem(row); setName(row.name || ""); setError(""); setTyped(""); }
  };

  async function submit(e) {
    e.preventDefault();
    if (submitting.current || !ready) return;
    submitting.current = true;
    setBusy(true); setError("");
    try {
      if (who) {
        if (!onAskAgent) throw new Error("The agent channel is unavailable.");
        const result = await onAskAgent(who, composed.prompt, root, item.action, composed.verb);
        setNotice(result?.busy ? `Queued for ${who.name}: ${composed.verb} after its current turn.` : `Asked ${who.name} to ${composed.verb}.`);
      } else {
        if (!onOpenTerminal) throw new Error("The terminal channel is unavailable.");
        await onOpenTerminal(owner, root, composed.command, { run: door === "run" });
        setNotice(door === "run" ? "Sent to your terminal." : "Prepared in your terminal — press Enter there.");
      }
      onDelivered?.({ action: item.action, verb: composed.verb, door: who ? "ask" : door });
    } catch (err) {
      setError(err?.message || "Could not send this action. Try again.");
    } finally {
      submitting.current = false; setBusy(false);
    }
  }

  const title = item ? LABELS[item.action].replace(/…$/, "") : menu ? menu.title || "Git actions" : "Git actions";
  const label = (field) => (FIELD_LABELS[item.action] || {})[field] || (field === "name" ? "Name" : "Message");

  return <Sheet.Root open={!!open} onOpenChange={(v) => { if (!v) close(); }}>
    <Sheet.Portal><Sheet.Overlay className="dlg-overlay" /><Sheet.Content className="dlg m-git-sheet m-gg-sheet" aria-describedby="m-gg-sheet-description">
      <div className="m-git-sheet-heading"><Sheet.Title className="dlg-title">{title}</Sheet.Title><button type="button" className="btn btn-ghost" disabled={busy} onClick={close}>Close</button></div>
      <Sheet.Description id="m-gg-sheet-description" className="dlg-body">
        {item ? (item.target ? <>On <code>{shortTarget(item.target)}</code>{composed && composed.tier === "C" ? " — this one publishes or destroys work." : "."}</> : "In this repository.") : (menu && menu.state) || "What this can do."}
        {!item && menu && menu.busy ? <span className="m-gg-busy">{menu.busy}</span> : null}
      </Sheet.Description>
      {notice ? <div className="m-git-state" role="status"><p>{notice}</p><button type="button" className="btn btn-primary" onClick={close}>Done</button></div>
      : !item ? <div className="m-git-action-list">
        {groups.length ? groups.map((group, gi) => <div className="m-gg-group" key={group.label || gi}>
          {group.label ? <p className="m-gg-group-label">{group.label}{group.state ? <span>{group.state}</span> : null}</p> : null}
          {group.items.map((row) => <button type="button" className={"m-git-action" + (row.tier === "C" ? " is-danger" : "")} key={row.id} onClick={() => pick(row)}>{row.label}<span aria-hidden="true">›</span></button>)}
        </div>) : <div className="m-git-state"><p>Nothing to do here yet.</p><button className="btn" type="button" onClick={close}>Back</button></div>}
        {error ? <p className="form-error" role="alert">{error}</p> : null}
      </div>
      : <form noValidate onSubmit={submit}>
        <fieldset disabled={busy} className="m-git-action-fields">
          {wantsName ? <label className="m-git-field"><span>{label("name")}</span><input autoComplete="off" value={name} maxLength={200} onChange={(e) => setName(e.target.value)} /></label> : null}
          {wantsMessage ? <label className="m-git-field"><span>{label("message")}</span><input autoComplete="off" value={message} maxLength={400} placeholder="What changed, in one line" onChange={(e) => setMessage(e.target.value)} /></label> : null}
          <label className="m-git-field"><span>Send through</span><select value={door} onChange={(e) => { setDoor(e.target.value); setTyped(""); setError(""); }}>
            {doorList.map((d) => <option key={d.value} value={d.value}>{d.label}</option>)}
          </select></label>
          <p className="m-git-hint">{doorHint(door, who)}</p>
          {g.typed ? <label className="m-git-field m-gg-typed"><span>PiCode presses Enter on this one. Type <strong>{g.typed}</strong> to confirm.</span><input autoComplete="off" value={typed} onChange={(e) => setTyped(e.target.value)} aria-label={`Type ${g.typed} to confirm`} /></label> : null}
          <div className="m-git-preview m-gg-preview">{preview ? <pre>{preview}</pre> : <p className="m-git-hint">{wantsName || wantsMessage ? "Fill the field above to see the exact command." : "Composing…"}</p>}</div>
        </fieldset>
        {blocked ? <p className="form-error" role="alert">This folder changed. Close this sheet and follow the working folder before continuing.</p> : null}
        {busy ? <p className="m-git-pending" role="status"><span aria-hidden="true" />Sending this action…</p> : null}
        {error ? <p className="form-error" role="alert">{error}</p> : null}
        <div className="dlg-actions" data-align-row>
          <button type="button" className="btn btn-ghost" disabled={busy} onClick={() => { setItem(null); setError(""); }}>Back</button>
          <button type="submit" className={"btn " + (composed && composed.tier === "C" ? "btn-danger" : "btn-primary")} disabled={!ready}>{busy ? "Sending…" : who ? `Ask ${who.name}` : door === "run" ? "Run when idle" : "Prepare"}</button>
        </div>
      </form>}
    </Sheet.Content></Sheet.Portal>
  </Sheet.Root>;
}
