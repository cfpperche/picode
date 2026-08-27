import { useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import "katex/dist/katex.min.css";
import { basename, statLabel } from "../lib/diff.js";
import { groupTurns, fmtWorked, fmtElapsed, stepLabel, turnDurationMs, firstTs, dayKey, fmtDayMark, workingIndex } from "../lib/turns.js";
import { IconCopy } from "./Icons.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { isSearchTool, hitsFromTool, searchQuery } from "../lib/searchCards.js";
import { mdComponents } from "./SourceBlock.jsx";
import { api } from "../lib/api.js";
import ImageLightbox from "./ImageLightbox.jsx";

export default function Conversation({ items, onToggleTool, onToggleFiles, convRef, onScroll, hidden, streaming, agentId, onAbortBash, onReplyAsk, onQueueRemove, onQueueEdit, onQueueSave, onQueueCancelEdit }) {
  const [preview, setPreview] = useState("");
  const turns = groupTurns((items || []).filter((it) => it.kind !== "sys" || it.err));
  const busy = workingIndex(turns, !!streaming);
  return (
    <div id="conversation" className="conversation" ref={convRef} onScroll={onScroll} style={{ visibility: hidden ? "hidden" : "visible" }}>
      <div className="conv-col">
        {turns.reduce((acc, t, i) => {
          if (t.kind === "loose") {
            acc.nodes.push(<Loose key={"l" + i} it={t.item} items={items} onToggleFiles={onToggleFiles} onAbortBash={onAbortBash} onReplyAsk={onReplyAsk} />);
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
            acc.nodes.push(<Turn key={"t" + n} turn={t} i={n} live={live} queued={queued} onToggleTool={onToggleTool} agentId={agentId} onPreview={setPreview} onQueueRemove={onQueueRemove} onQueueEdit={onQueueEdit} onQueueSave={onQueueSave} onQueueCancelEdit={onQueueCancelEdit} />);          }
          return acc;
        }, { n: 0, day: "", nodes: [] }).nodes}
      </div>
      <ImageLightbox src={preview} onClose={() => setPreview("")} />
    </div>
  );
}

function Loose({ it, items, onToggleFiles, onAbortBash, onReplyAsk }) {
  if (it.kind === "sys") {
    return <div className={"sys-line" + (it.err ? " err" : "")}>{it.text}</div>;
  }
  if (it.kind === "ask") {
    return <AskCard it={it} onReply={onReplyAsk} />;
  }
  if (it.kind === "bash") {
    return <BashBlock it={it} onAbort={onAbortBash} />;
  }
  if (it.kind === "files") {
    const i = items.indexOf(it);
    return (
      <div className={"files-changed" + (it.expanded ? " expanded" : "")}>
        <button type="button" className="files-changed-head" onClick={() => onToggleFiles(i)}>
          <span className="tp-chevron">›</span>
          {it.paths.length} {it.paths.length === 1 ? "file" : "files"} changed
        </button>
        {it.expanded ? (
          <ul className="files-changed-list">
            {it.paths.map((p) => <li key={p}>{p}</li>)}
          </ul>
        ) : null}
      </div>
    );
  }
  return null;
}

function AskCard({ it, onReply }) {
  const [text, setText] = useState(it.prefill || "");
  const open = it.status === "open";
  const lines = String(it.title || "The agent is asking something").split("\n");
  const title = lines[0] || "The agent is asking something";
  const extra = it.message || lines.slice(1).join("\n");
  const done = it.status === "cancelled" ? "Cancelled"
    : it.status === "timeout" ? "Timed out"
    : it.status === "answered" ? (it.answer || "Answered")
    : "";

  function send(body) {
    if (!open || !onReply) return;
    onReply(it.id, body);
  }

  function onKey(e) {
    if (!open) return;
    if (e.key === "Escape") {
      e.preventDefault();
      send({ cancelled: true });
    }
    if (e.key === "Enter" && (it.method === "input" || it.method === "editor") && !e.shiftKey) {
      e.preventDefault();
      send({ value: text });
    }
  }

  return (
    <div className={"ask-card" + (open ? " open" : " done")} onKeyDown={onKey}>
      <p className="ask-kicker">The agent is asking something</p>
      <p className="ask-title">{title}</p>
      {extra ? <p className="ask-msg">{extra}</p> : null}
      {open && it.method === "select" ? (
        <div className="ask-actions">
          {(it.options || []).map((opt) => (
            <button key={opt} type="button" className="btn btn-sm" onClick={() => send({ value: opt })}>{opt}</button>
          ))}
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => send({ cancelled: true })}>Cancel</button>
        </div>
      ) : null}
      {open && it.method === "confirm" ? (
        <div className="ask-actions" data-align-row>
          <button type="button" className="btn btn-primary btn-sm" onClick={() => send({ confirmed: true })}>Yes</button>
          <button type="button" className="btn btn-sm" onClick={() => send({ confirmed: false })}>No</button>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => send({ cancelled: true })}>Cancel</button>
        </div>
      ) : null}
      {open && it.method === "input" ? (
        <div className="ask-input-row" data-align-row>
          <input className="ask-input" value={text} placeholder={it.placeholder || ""} autoFocus onChange={(e) => setText(e.target.value)} />
          <button type="button" className="btn btn-primary btn-sm" onClick={() => send({ value: text })}>Send</button>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => send({ cancelled: true })}>Cancel</button>
        </div>
      ) : null}
      {open && it.method === "editor" ? (
        <>
          <textarea className="ask-input ask-area" rows={4} value={text} placeholder={it.placeholder || ""} autoFocus onChange={(e) => setText(e.target.value)} />
          <div className="ask-actions" data-align-row>
            <button type="button" className="btn btn-primary btn-sm" onClick={() => send({ value: text })}>Send</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => send({ cancelled: true })}>Cancel</button>
          </div>
        </>
      ) : null}
      {!open && done ? <p className="ask-done">{done}</p> : null}
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

function Turn({ turn, i, live, queued, onToggleTool, agentId, onPreview, onQueueRemove, onQueueEdit, onQueueSave, onQueueCancelEdit }) {
  const [userOpen, setUserOpen] = useState(false);
  const [now, setNow] = useState(Date.now());
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
                  {it.kind === "tool" ? <Tool it={it} onToggle={onToggleTool} /> : null}
                </li>
              ))}
            </ol>
          ) : null}
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

function Tool({ it, onToggle }) {
  const ch = it.change;
  const search = isSearchTool(it.name);
  const hits = search ? hitsFromTool(it) : [];
  const q = search ? (searchQuery(it.toolArgs) || it.args) : "";
  return (
    <div className={"tool-pill" + (it.status === "ok" ? " ok" : "") + (it.status === "error" ? " err" : "") + (it.expanded ? " expanded" : "") + (search ? " search" : "")}>
      <div className="tool-pill-head" onClick={() => onToggle(it.id)}>
        <span className="tp-chevron">›</span>
        <span className="tp-name">{search ? "search" : it.name}</span>
        <span className="tp-args">{ch ? basename(ch.path) || it.args : (q || it.args)}</span>
        {ch ? <span className="tp-stat"><span className="add">{ch.add ? "+" + ch.add : ""}</span>{ch.add && ch.del ? " " : ""}<span className="del">{ch.del ? "−" + ch.del : ""}</span></span> : null}
        <span className="tp-status">{hits.length ? hits.length : (it.status || "···")}</span>
      </div>
      <div className={"tp-detail" + (ch ? " tp-diff" : "") + (hits.length ? " tp-search" : "")}>
        {ch ? <DiffHunks hunks={ch.hunks} /> : hits.length ? <SearchHits hits={hits} /> : it.detail}
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

function DiffHunks({ hunks }) {
  if (!hunks || !hunks.length) return <div className="diff-empty">No diff</div>;
  return (
    <div className="diff">
      {hunks.map((h, i) => (
        <div key={i} className={"diff-line " + h.kind}>
          <span className="diff-gutter">{h.kind === "add" ? "+" : h.kind === "del" ? "−" : h.kind === "gap" ? "·" : " "}</span>
          <span className="diff-text">{h.text}</span>
        </div>
      ))}
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

export { statLabel };
