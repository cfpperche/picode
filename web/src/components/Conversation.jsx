import { memo, useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import "katex/dist/katex.min.css";
import { basename, statLabel, groupHunks, undoHunkInText } from "../lib/diff.js";
import { groupTurns, fmtWorked, fmtElapsed, stepLabel, turnDurationMs, firstTs, dayKey, fmtDayMark, workingIndex, pathsFromTurn } from "../lib/turns.js";
import { IconCopy } from "./Icons.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { isSearchTool, hitsFromTool, searchQuery } from "../lib/searchCards.js";
import { mdComponents } from "./SourceBlock.jsx";
import { api } from "../lib/api.js";
import ImageLightbox from "./ImageLightbox.jsx";
import FileCard from "./FileCard.jsx";

const WINDOW_STEP = 60;

function Conversation({ items, onToggleTool, onToggleFiles, convRef, onScroll, hidden, streaming, agentId, onAbortBash, onReplyAsk, onQueueRemove, onQueueEdit, onQueueSave, onQueueCancelEdit, onOpenTab, after, earlierRemaining, onFetchEarlier }) {
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
            acc.nodes.push(<Loose key={"l" + i} it={t.item} items={items} onToggleFiles={onToggleFiles} onAbortBash={onAbortBash} onReplyAsk={onReplyAsk} agentId={agentId} onOpenTab={onOpenTab} />);

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
        {after}
      </div>
      <ImageLightbox src={preview} onClose={() => setPreview("")} />
    </div>
  );
}

function Loose({ it, items, onToggleFiles, onAbortBash, onReplyAsk, agentId, onOpenTab }) {
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
    return <FilesChanged it={it} items={items} onToggleFiles={onToggleFiles} agentId={agentId} onOpenTab={onOpenTab} />;
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

function DiffLine({ h }) {
  return (
    <div className={"diff-line " + h.kind}>
      <span className="diff-gutter">{h.kind === "add" ? "+" : h.kind === "del" ? "−" : h.kind === "gap" ? "·" : " "}</span>
      <span className="diff-text">{h.text}</span>
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
