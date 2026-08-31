import { Fragment, memo, useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import "katex/dist/katex.min.css";
import { basename, statLabel, groupHunks, undoHunkInText } from "../lib/diff.js";
import DiffLine from "./DiffLine.jsx";
import { groupTurns, fmtWorked, fmtElapsed, stepLabel, turnDurationMs, firstTs, dayKey, fmtDayMark, workingIndex, pathsFromTurn } from "../lib/turns.js";
import { IconCopy, IconFile } from "./Icons.jsx";
import { ProviderFace } from "./ProviderFaces.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { isSearchTool, hitsFromTool, searchQuery } from "../lib/searchCards.js";
import { mdComponents } from "./SourceBlock.jsx";
import { api } from "../lib/api.js";
import ImageLightbox from "./ImageLightbox.jsx";
import FileCard from "./FileCard.jsx";
import SearchCombo from "./SearchCombo.jsx";
import { fieldLabel, summaryParts, BACK } from "../lib/askForm.js";

const WINDOW_STEP = 60;

function Conversation({ items, onToggleTool, onToggleFiles, convRef, onScroll, hidden, streaming, agentId, compactSince, onAbortBash, onReplyAsk, onPrefill, onQueueRemove, onQueueEdit, onQueueSave, onQueueCancelEdit, onOpenTab, after, earlierRemaining, onFetchEarlier }) {
  const [preview, setPreview] = useState("");
  const [limit, setLimit] = useState(WINDOW_STEP);
  const growLock = useRef(false);
  const turns = groupTurns((items || []).filter((it) => it.kind !== "sys" || it.err));
  const busy = workingIndex(turns, !!streaming);
  const hiddenCount = Math.max(0, turns.length - limit);
  const serverRemaining = Math.max(0, earlierRemaining || 0);
  const canGrow = hiddenCount > 0 || serverRemaining > 0;
  function growEarlier() {
    if (growLock.current) return;
    if (hiddenCount > 0) {
      growLock.current = true;
      const el = convRef && convRef.current;
      const keep = el ? el.scrollHeight - el.scrollTop : 0;
      setLimit((l) => l + WINDOW_STEP);
      requestAnimationFrame(() => {
        const el2 = convRef && convRef.current;
        if (el2) el2.scrollTop = Math.max(0, el2.scrollHeight - keep);
        growLock.current = false;
      });
      return;
    }
    if (serverRemaining > 0 && onFetchEarlier) {
      growLock.current = true;
      const el = convRef && convRef.current;
      const keep = el ? el.scrollHeight - el.scrollTop : 0;
      Promise.resolve(onFetchEarlier()).finally(() => {
        requestAnimationFrame(() => {
          const el2 = convRef && convRef.current;
          if (el2) el2.scrollTop = Math.max(0, el2.scrollHeight - keep);
          growLock.current = false;
        });
      });
    }
  }
  function onScrollWrap(ev) {
    const el = ev && ev.currentTarget;
    if (el && el.scrollTop < 80 && canGrow) growEarlier();
    if (onScroll) onScroll(ev);
  }
  return (
    <div id="conversation" className="conversation" ref={convRef} onScroll={onScrollWrap} style={{ visibility: hidden ? "hidden" : "visible" }}>
      <div className="conv-col">
        {canGrow ? (
          <button type="button" className="conv-load-earlier" onClick={growEarlier}>
            {hiddenCount > 0 ? "Load earlier (" + hiddenCount + ")" : "Load earlier"}
          </button>
        ) : null}
        {turns.reduce((acc, t, i) => {
          if (i < hiddenCount && i !== busy) return acc;
          if (t.kind === "loose") {
            acc.nodes.push(<Loose key={"l" + i} it={t.item} items={items} onToggleFiles={onToggleFiles} onAbortBash={onAbortBash} onReplyAsk={onReplyAsk} onPrefill={onPrefill} agentId={agentId} onOpenTab={onOpenTab} />);

          } else {
            const n = acc.n++;
            const live = i === busy;
            const chip = t.user && t.user.chip;
            const queued = !live && !!t.user && !t.user.dropped && t.replies.length === 0 && t.work.length === 0 && (chip === "steer" || (chip === "follow_up" && t.user.pending));
            const ts = firstTs(t);
            const day = dayKey(ts);
            if (day && day !== acc.day) {
              acc.nodes.push(<div key={"d" + n} className="day-mark">{fmtDayMark(ts)}</div>);
              acc.day = day;
            }
            acc.nodes.push(<Turn key={"t" + n} turn={t} i={n} live={live} queued={queued} onToggleTool={onToggleTool} agentId={agentId} onPreview={setPreview} onQueueRemove={onQueueRemove} onQueueEdit={onQueueEdit} onQueueSave={onQueueSave} onQueueCancelEdit={onQueueCancelEdit} onOpenTab={onOpenTab} />);
          }
          return acc;
        }, { n: 0, day: "", nodes: [] }).nodes}
        {compactSince ? <CompactLive since={compactSince} /> : null}
        {after}
      </div>
      <ImageLightbox src={preview} onClose={() => setPreview("")} />
    </div>
  );
}

function CompactLive({ since }) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [since]);
  return (
    <div
      className="compact-live"
      role="status"
      title="Older turns are being summarized into a compact. Large sessions can take minutes."
    >
      <span className="work-dot" aria-hidden="true" />
      <span className="compact-live-lab">Compacting session…</span>
      <span className="compact-live-time">{fmtElapsed(now - since)}</span>
    </div>
  );
}

function Loose({ it, items, onToggleFiles, onAbortBash, onReplyAsk, onPrefill, agentId, onOpenTab }) {
  if (it.kind === "sys") {
    return <div className={"sys-line" + (it.err ? " err" : "")}>{it.text}</div>;
  }
  if (it.kind === "ask") {
    return <AskCard it={it} onReply={onReplyAsk} onPrefill={onPrefill} />;
  }
  if (it.kind === "note") {
    return <NoteLine it={it} onPrefill={onPrefill} />;
  }
  if (it.kind === "bash") {
    return <BashBlock it={it} onAbort={onAbortBash} />;
  }
  if (it.kind === "files") {
    return <FilesChanged it={it} items={items} onToggleFiles={onToggleFiles} agentId={agentId} onOpenTab={onOpenTab} />;
  }
  if (it.kind === "compaction") {
    return <CompactionCard it={it} />;
  }
  return null;
}

function CompactionCard({ it }) {
  const [open, setOpen] = useState(false);
  const words = (it.text || "").trim().split(/\s+/).filter(Boolean).length;
  return (
    <div className={"compaction-card" + (open ? " open" : "")}>
      <button type="button" className="compaction-head" onClick={() => setOpen((v) => !v)}>
        <span className="tp-chevron" aria-hidden="true">›</span>
        Session compacted
        <span className="compaction-meta">{words ? words + "-word summary of earlier turns" : "earlier turns summarized"}</span>
      </button>
      {open ? (
        <div className="block-content md compaction-body">
          <Markdown remarkPlugins={[remarkGfm, remarkMath]} rehypePlugins={[rehypeKatex]}>{it.text}</Markdown>
        </div>
      ) : null}
    </div>
  );
}

/** "/roles clear agent" → { name: "roles", args: "clear agent" }. */
function cmdParts(cmd) {
  const t = String(cmd || "").trim();
  if (!t.startsWith("/")) return null;
  const words = t.slice(1).split(/\s+/).filter(Boolean);
  if (!words.length || !words[0]) return null;
  return { name: words[0], args: words.slice(1).join(" ") };
}

function looksLikePath(text) {
  const t = String(text || "").trim();
  return t.includes("/") && /^[\w.~/-]+$/.test(t);
}

const OUTCOME_MARK = {
  definition: "✓", role: "✓", cleared: "⌫", kept: "○",
  empty: "!", cancelled: "⊘", timeout: "◔", text: "✓",
};

/** One finished-flow line: mark + command badge + typed content chips. */
function AskOutcome({ cmd, parts, onPrefill }) {
  const kind = parts.kind;
  let body = null;
  if (kind === "definition") {
    body = (
      <>
        {parts.role ? <span className="ask-oc-role">{parts.role}</span> : null}
        <span className="ask-chip ask-chip-model" title={parts.model}>
          {parts.provider ? <ProviderFace id={parts.provider} /> : null}
          {parts.modelId}
        </span>
        {parts.thinking ? <span className="ask-chip">{parts.thinking}</span> : null}
        {parts.scope ? <span className="ask-chip ask-chip-scope">{parts.scope}</span> : null}
      </>
    );
  } else if (kind === "role") {
    body = (
      <>
        <span className="ask-oc-role">{parts.role}</span>
        {parts.text ? <span className="ask-oc-sub">{parts.text}</span> : null}
      </>
    );
  } else if (kind === "cleared" || kind === "kept") {
    body = (
      <>
        <span className="ask-oc-verb">{kind}</span>
        <span className="ask-chip ask-chip-file"><IconFile />{parts.file}</span>
        {kind === "kept" ? <span className="ask-oc-sub">nothing deleted</span> : null}
      </>
    );
  } else if (kind === "empty") {
    body = (
      <>
        <span className="ask-oc-text">{parts.text}</span>
        {cmd && cmd.name === "roles" && onPrefill ? (
          <button type="button" className="ask-chip ask-chip-action" onClick={() => onPrefill("/roles add")}>
            Set one up → /roles add
          </button>
        ) : null}
      </>
    );
  } else if (kind === "cancelled" || kind === "timeout") {
    body = <span className="ask-oc-text">{kind === "timeout" ? "timed out" : "cancelled"}</span>;
  } else {
    body = <span className="ask-oc-text">{parts.text}</span>;
  }
  return (
    <div className={"ask-outcome oc-" + kind}>
      <span className="ask-oc-mark" aria-hidden="true">{OUTCOME_MARK[kind] || "✓"}</span>
      {cmd ? <span className="ask-oc-badge">{cmd.name}</span> : null}
      {body}
    </div>
  );
}

const NOTE_MARK = { info: "✓", warning: "!", error: "✗" };

/**
 * A slash command's one-line result (its only notify): mark + command
 * badge + the message. A "/roles …" fragment in the message becomes a
 * chip that prefills the composer — the empty state carries its action.
 */
function NoteLine({ it, onPrefill }) {
  const cmd = cmdParts(it.cmd);
  const level = NOTE_MARK[it.level] ? it.level : "info";
  const text = String(it.text || "");
  const m = /(\/roles(?:\s+(?:edit|add)(?:\s+[\w-]+)?)?)/.exec(text);
  const action = m && onPrefill ? m[1] : "";
  const before = action ? text.slice(0, m.index).trim() : text;
  const after = action ? text.slice(m.index + action.length).trim() : "";
  return (
    <div className={"ask-outcome oc-note oc-" + level}>
      <span className="ask-oc-mark" aria-hidden="true">{NOTE_MARK[level]}</span>
      {cmd ? <span className="ask-oc-badge">{cmd.name}</span> : null}
      <span className="ask-oc-text">{before}</span>
      {action ? (
        <button type="button" className="ask-chip ask-chip-action" onClick={() => onPrefill(action)}>{action}</button>
      ) : null}
      {after ? <span className="ask-oc-text">{after}</span> : null}
    </div>
  );
}

function AskCard({ it, onReply, onPrefill }) {
  const [text, setText] = useState(it.prefill || "");
  const open = it.status === "open";
  const steps = it.steps && it.steps.length ? it.steps : [it];
  const current = steps.find((s) => s.status === "open") || null;
  const extra = (current && current.message) || "";
  const method = current ? current.method : "";
  const options = (current && current.options) || [];
  const answered = steps.filter((s) => s.answer && s.status === "answered");
  const cmd = cmdParts(it.cmd);
  const currentLab = fieldLabel((current && current.title) || it.title || "");
  // Pills go back only when the extension can (its select offers BACK).
  const canBack = options.includes(BACK);

  function send(body) {
    if (!open || !onReply) return;
    const openId = (current && current.id) || it.id;
    onReply(openId, body);
  }

  function onKey(e) {
    if (!open) return;
    if (e.key === "Escape") {
      e.preventDefault();
      send({ cancelled: true });
    }
    if (e.key === "Enter" && (method === "input" || method === "editor") && !e.shiftKey) {
      e.preventDefault();
      send({ value: text });
    }
  }

  const comboOpts = options.filter((opt) => opt !== BACK).map((opt) => ({ id: opt, label: opt }));

  if (!open) {
    if (it.status === "answered") {
      const parts = summaryParts(steps, it.note);
      if (parts) return <AskOutcome cmd={cmd} parts={parts} onPrefill={onPrefill} />;
      return <p className="ask-done">Answered</p>;
    }
    if (it.status === "cancelled" || it.status === "timeout") {
      return <AskOutcome cmd={cmd} parts={{ kind: it.status }} />;
    }
    return null;
  }

  const confirmDanger = method === "confirm" && /^(delete|remove)/i.test((current && current.title) || "");
  const pills = answered.map((s, i) => {
    const lab = fieldLabel(s.title);
    // Decorated role options ("vision — xai/… · medium") pill as the name.
    const shown = lab === "Role" || lab === "Name" ? String(s.answer).split(" — ")[0] : s.answer;
    return (
      <Fragment key={s.id}>
        {i > 0 ? <span className="ask-arrow" aria-hidden="true">›</span> : null}
        <div className="ask-step">
          <span className="ask-step-lab">{lab}</span>
          {canBack ? (
            <button type="button" className="ask-pill" title="Go back to this step" onClick={() => send({ backTo: i })}>{shown}</button>
          ) : (
            <span className="ask-pill ask-pill-static">{shown}</span>
          )}
        </div>
      </Fragment>
    );
  });

  return (
    <div className="ask-card open" onKeyDown={onKey}>
      <div className="ask-head">
        <span className="ask-head-cmd">{cmd ? cmd.name : "agent"}</span>
        {cmd && cmd.args ? <span className="ask-head-args">{cmd.args}</span> : null}
        <span className="ask-head-gap" />
        <button type="button" className="ask-cancel" onClick={() => send({ cancelled: true })}>Cancel</button>
      </div>
      {extra && method !== "confirm" ? <p className="ask-msg">{extra}</p> : null}
      {method === "confirm" ? (
        <div className="ask-confirm">
          {answered.length ? <div className="ask-actions">{pills}</div> : null}
          <p className="ask-q">{(current && current.title) || "Confirm?"}</p>
          {extra ? (
            looksLikePath(extra)
              ? <span className="ask-chip ask-chip-file"><IconFile />{extra}</span>
              : <p className="ask-msg">{extra}</p>
          ) : null}
          <div className="ask-actions" data-align-row>
            <button type="button" className={"btn btn-sm " + (confirmDanger ? "btn-danger" : "btn-primary")} onClick={() => send({ confirmed: true })}>
              {confirmDanger ? "Delete" : "Yes"}
            </button>
            <button type="button" className="btn btn-sm" onClick={() => send({ confirmed: false })}>
              {confirmDanger ? "Keep" : "No"}
            </button>
          </div>
        </div>
      ) : (
        <div className="ask-actions">
          {pills}
          {answered.length && (method === "select" || method === "input") ? <span className="ask-arrow" aria-hidden="true">›</span> : null}
          {method === "select" ? (
            <div className="ask-step ask-step-open">
              <span className="ask-step-lab">{currentLab}</span>
              {comboOpts.length ? (
                <SearchCombo
                  value=""
                  onChange={(id) => send({ value: id })}
                  options={comboOpts}
                  label={currentLab !== "Choose" ? "Choose " + currentLab.toLowerCase() + "…" : "Select"}
                  searchPlaceholder="Filter"
                  triggerClassName="ask-combo"
                  side="bottom"
                />
              ) : (
                <span className="ask-empty">No options.</span>
              )}
            </div>
          ) : null}
          {method === "input" ? (
            <>
              <input className="ask-input" value={text} placeholder={it.placeholder || ""} autoFocus onChange={(e) => setText(e.target.value)} />
              <button type="button" className="btn btn-primary btn-sm" onClick={() => send({ value: text })}>Send</button>
            </>
          ) : null}
        </div>
      )}
      {method === "editor" ? (
        <>
          <textarea className="ask-input ask-area" rows={4} value={text} placeholder={it.placeholder || ""} autoFocus onChange={(e) => setText(e.target.value)} />
          <div className="ask-actions" data-align-row>
            <button type="button" className="btn btn-primary btn-sm" onClick={() => send({ value: text })}>Send</button>
          </div>
        </>
      ) : null}
    </div>
  );
}

function BashBlock({ it, onAbort }) {
  const running = it.status === "run";
  return (
    <div className={"bash-block " + (it.status || "")}>
      <div className="bash-head">
        <span className="bash-mark" aria-hidden="true">
          {running ? <PiSpinner title="Running" /> : it.status === "ok" ? "✓" : it.status === "cancelled" ? "⊘" : "✗"}
        </span>
        <code className="bash-cmd">$ {it.command}</code>
        {it.exit != null && !running ? <span className="bash-exit">exit {it.exit}</span> : null}
        {running && onAbort ? (
          <button type="button" className="btn btn-ghost btn-sm" onClick={onAbort}>Stop</button>
        ) : it.output ? (
          <CopyBtn text={it.output} />
        ) : null}
      </div>
      {it.output ? <pre className="bash-out">{it.output}</pre> : running ? <pre className="bash-out bash-pending">…</pre> : null}
    </div>
  );
}

function openCard(setCards, p) {
  if (!p) return;
  setCards((c) => (c.includes(p) ? c : [...c, p]));
}

function TurnFileCards({ agentId, cards, setCards, onOpenTab }) {
  if (!cards.length) return null;
  return cards.map((p) => (
    <FileCard key={p} agentId={agentId} path={p} onClose={() => setCards((c) => c.filter((x) => x !== p))} onOpenTab={onOpenTab} />
  ));
}

function FilesChanged({ it, items, onToggleFiles, agentId, onOpenTab }) {
  const [cards, setCards] = useState([]);
  const i = items.indexOf(it);
  return (
    <div className={"files-changed" + (it.expanded ? " expanded" : "")}>
      <button type="button" className="files-changed-head" onClick={() => onToggleFiles(i)}>
        <span className="tp-chevron">›</span>
        {it.paths.length} {it.paths.length === 1 ? "file" : "files"} changed
      </button>
      {it.expanded ? (
        <ul className="files-changed-list">
          {it.paths.map((p) => (
            <li key={p}>
              {agentId ? (
                <button type="button" className="files-changed-open" onClick={() => openCard(setCards, p)}>{p}</button>
              ) : p}
            </li>
          ))}
        </ul>
      ) : null}
      <TurnFileCards agentId={agentId} cards={cards} setCards={setCards} onOpenTab={onOpenTab} />
    </div>
  );
}

function Turn({ turn, i, live, queued, onToggleTool, agentId, onPreview, onQueueRemove, onQueueEdit, onQueueSave, onQueueCancelEdit, onOpenTab }) {
  const [userOpen, setUserOpen] = useState(false);
  const [now, setNow] = useState(Date.now());
  const [cards, setCards] = useState([]);
  const liveFrom = useRef(0);
  const replyN = turn.replies.length;
  useEffect(() => {
    if (replyN > 0) setUserOpen(false);
  }, [replyN]);
  useEffect(() => {
    if (!live) { liveFrom.current = 0; return; }
    if (!liveFrom.current) liveFrom.current = Date.now();
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [live]);
  const shown = !!live || userOpen;
  const label = live
    ? "Working… " + fmtElapsed(now - (liveFrom.current || now))
    : queued
      ? "Queued"
      : fmtWorked(turnDurationMs(turn));
  const showWork = turn.work.length > 0 || live || queued;
  const files = pathsFromTurn(turn);
  const open = (p) => { if (agentId) openCard(setCards, p); };
  return (
    <div className="turn" id={"turn-" + i}>
      {turn.user ? <Block it={turn.user} railId={"turn-" + i + "-user"} onPreview={onPreview} onQueueRemove={onQueueRemove} onQueueEdit={onQueueEdit} onQueueSave={onQueueSave} onQueueCancelEdit={onQueueCancelEdit} /> : null}
      {showWork ? (
        <div className={"work" + (shown ? " open" : "") + (live ? " live" : "")}>
          <button type="button" className="work-head" onClick={() => !live && setUserOpen((v) => !v)}>
            <span className="work-dot" aria-hidden="true" />
            <span>{label}</span>
            {!live ? <span className="tp-chevron">›</span> : null}
          </button>
          {shown && turn.work.length > 0 ? (
            <ol className="work-steps">
              {turn.work.map((it, j) => (
                <li key={it.id || j} className={"work-step" + (live && j === turn.work.length - 1 ? " current" : "")}>
                  <span className="work-step-lab">{stepLabel(it)}</span>
                  {it.kind === "tool" ? <Tool it={it} onToggle={onToggleTool} onOpenFile={open} agentId={agentId} /> : null}
                </li>
              ))}
            </ol>
          ) : null}
        </div>
      ) : null}
      {files.length || cards.length ? (
        <div className="turn-files-wrap">
          {files.length ? (
            <ul className="turn-files">
              {files.map((p) => (
                <li key={p}>
                  <button type="button" className={"turn-files-open" + (cards.includes(p) ? " open" : "")} title={p} onClick={() => open(p)}>{basename(p)}</button>
                </li>
              ))}
            </ul>
          ) : null}
          <TurnFileCards agentId={agentId} cards={cards} setCards={setCards} onOpenTab={onOpenTab} />
        </div>
      ) : null}
      {turn.replies.map((it, j) => it.kind === "alert"
        ? <Alert key={j} it={it} />
        : <Block key={j} it={it} railId={j === 0 ? "turn-" + i + "-agent" : undefined} agentId={agentId} />)}
    </div>
  );
}

function Alert({ it }) {
  return <div className={"chat-alert " + (it.level || "error")}>{it.text}</div>;
}

function Block({ it, railId, agentId, onPreview, onQueueRemove, onQueueEdit, onQueueSave, onQueueCancelEdit }) {
  const user = it.cls === "user";
  const md = !user && it.cls !== "thinking";
  const pendingFu = user && it.pending && it.chip === "follow_up" && !it.dropped;
  const [edit, setEdit] = useState(it.text || "");
  useEffect(() => { if (it.editing) setEdit(it.text || ""); }, [it.editing, it.text]);
  async function onRun(lang, code) {
    if (!agentId) throw new Error("Select an agent.");
    return api("/api/agents/" + agentId + "/snippet", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ lang, code }),
    });
  }
  return (
    <div className={"block " + (it.cls || "")} data-rail={railId || undefined}>
      {md && it.text ? <div className="block-tools"><CopyBtn text={it.text} /></div> : null}
      <div className={"block-content" + (md ? " md" : "")}>
        {it.images && it.images.length ? (
          <div className="block-pics">
            {it.images.map((src, i) => (
              <button key={i} type="button" className="block-pic" onClick={() => onPreview && onPreview(src)} title="View image">
                <img src={src} alt="" />
              </button>
            ))}
          </div>
        ) : null}
        {user && it.chip && it.chip !== "prompt" ? (
          <span className={"msg-kind" + (it.dropped ? " dropped" : "")}>
            {it.dropped ? "Dropped" : it.chip === "steer" ? "Steer" : "Follow-up"}
          </span>
        ) : null}
        {pendingFu && it.editing ? (
          <>
            <textarea className="ask-input ask-area" rows={3} value={edit} onChange={(e) => setEdit(e.target.value)} autoFocus />
            <div className="ask-actions" data-align-row>
              <button type="button" className="btn btn-primary btn-sm" disabled={!edit.trim()} onClick={() => onQueueSave && onQueueSave(it.qid, edit)}>Save</button>
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => onQueueCancelEdit && onQueueCancelEdit(it.qid)}>Cancel</button>
            </div>
          </>
        ) : md ? <Markdown remarkPlugins={[remarkGfm, remarkMath]} rehypePlugins={[rehypeKatex]} components={mdComponents({ CopyBtn, onRun: agentId ? onRun : null })}>{it.text || ""}</Markdown> : it.text}
        {pendingFu && !it.editing ? (
          <div className="ask-actions" data-align-row>
            <button type="button" className="btn btn-sm" onClick={() => onQueueEdit && onQueueEdit(it.qid)}>Edit</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => onQueueRemove && onQueueRemove(it.qid)}>Remove</button>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function Tool({ it, onToggle, onOpenFile, agentId }) {
  const ch = it.change;
  const search = isSearchTool(it.name);
  const hits = search ? hitsFromTool(it) : [];
  const q = search ? (searchQuery(it.toolArgs) || it.args) : "";
  return (
    <div className={"tool-pill" + (it.status === "ok" ? " ok" : "") + (it.status === "error" ? " err" : "") + (it.expanded ? " expanded" : "") + (search ? " search" : "")}>
      <div className="tool-pill-head" onClick={() => onToggle(it.id)}>
        <span className="tp-chevron">›</span>
        <span className="tp-name">{search ? "search" : it.name}</span>
        {ch && onOpenFile ? (
          <button type="button" className="tp-file" onClick={(e) => { e.stopPropagation(); onOpenFile(ch.path); }}>{basename(ch.path) || it.args}</button>
        ) : (
          <span className="tp-args">{ch ? basename(ch.path) || it.args : (q || it.args)}</span>
        )}
        {ch ? <span className="tp-stat"><span className="add">{ch.add ? "+" + ch.add : ""}</span>{ch.add && ch.del ? " " : ""}<span className="del">{ch.del ? "−" + ch.del : ""}</span></span> : null}
        <span className="tp-status">{hits.length ? hits.length : (it.status || "···")}</span>
      </div>
      <div className={"tp-detail" + (ch ? " tp-diff" : "") + (hits.length ? " tp-search" : "")}>
        {ch ? <DiffHunks hunks={ch.hunks} path={ch.path} agentId={agentId} onOpenFile={onOpenFile} /> : hits.length ? <SearchHits hits={hits} /> : it.detail}
      </div>
    </div>
  );
}

function SearchHits({ hits }) {
  if (!hits.length) return <p className="side-empty">No sources.</p>;
  return (
    <ul className="search-hits">
      {hits.map((h) => (
        <li key={h.url}>
          <a href={h.url} target="_blank" rel="noreferrer">
            <span className="search-hit-title">{h.title}</span>
            <span className="search-hit-url">{h.url}</span>
            {h.snippet ? <span className="search-hit-snip">{h.snippet}</span> : null}
          </a>
        </li>
      ))}
    </ul>
  );
}

function CopyBtn({ text }) {
  const [done, setDone] = useState(false);
  return (
    <button
      type="button"
      className="copy-btn"
      title="Copy"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setDone(true);
          setTimeout(() => setDone(false), 1500);
        } catch { /* ignore */ }
      }}
    >
      <IconCopy />
      {done ? "Copied" : "Copy"}
    </button>
  );
}

function DiffHunks({ hunks, path, agentId, onOpenFile }) {
  const groups = groupHunks(hunks);
  const [mark, setMark] = useState({});
  const [busy, setBusy] = useState(-1);
  if (!hunks || !hunks.length) return <div className="diff-empty">No diff</div>;
  async function undo(i, g) {
    if (!agentId || !path || busy >= 0) return;
    if (!g.dels.length && !g.ctxBefore.length && !g.ctxAfter.length) {
      setMark((m) => ({ ...m, [i]: "nowrite" }));
      return;
    }
    setBusy(i);
    try {
      const page = await api("/api/agents/" + agentId + "/text?path=" + encodeURIComponent(path));
      const next = undoHunkInText(page.text, g);
      if (!next.ok) {
        setMark((m) => ({ ...m, [i]: "err" }));
        return;
      }
      await api("/api/agents/" + agentId + "/text", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: page.path || path, text: next.text, mtime: page.mtime }),
      });
      setMark((m) => ({ ...m, [i]: "undone" }));
    } catch {
      setMark((m) => ({ ...m, [i]: "err" }));
    } finally {
      setBusy(-1);
    }
  }
  if (!groups.length) {
    return (
      <div className="diff">
        {hunks.map((h, i) => <DiffLine key={i} h={h} />)}
      </div>
    );
  }
  return (
    <div className="diff">
      {groups.map((g, i) => {
        const st = mark[i];
        const lines = [];
        g.ctxBefore.forEach((text) => lines.push({ kind: "ctx", text }));
        g.dels.forEach((text) => lines.push({ kind: "del", text }));
        g.adds.forEach((text) => lines.push({ kind: "add", text }));
        g.ctxAfter.forEach((text) => lines.push({ kind: "ctx", text }));
        return (
          <div key={i} className="diff-hunk">
            <div className="diff-hunk-bar">
              {st === "kept" ? <span className="diff-hunk-note">Kept</span> : null}
              {st === "undone" ? <span className="diff-hunk-note">Undone</span> : null}
              {st === "err" || st === "nowrite" ? (
                <>
                  <span className="diff-hunk-note">{st === "nowrite" ? "Can't undo this write." : "File changed."}</span>
                  {onOpenFile ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => onOpenFile(path)}>Open</button> : null}
                </>
              ) : null}
              {!st ? (
                <>
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => setMark((m) => ({ ...m, [i]: "kept" }))} disabled={busy >= 0}>Keep</button>
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => undo(i, g)} disabled={busy >= 0}>{busy === i ? "Undo…" : "Undo"}</button>
                </>
              ) : null}
            </div>
            {lines.map((h, j) => <DiffLine key={j} h={h} />)}
          </div>
        );
      })}
    </div>
  );
}

export function summarizeArgs(args) {
  if (!args) return "";
  if (typeof args.query === "string") return args.query;
  if (typeof args.command === "string") return args.command;
  if (typeof args.path === "string") return args.path;
  const s = JSON.stringify(args);
  return s.length > 2 ? s : "";
}

export default memo(Conversation);
export { statLabel };
