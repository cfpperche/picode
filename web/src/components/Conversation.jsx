import { useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import { basename, statLabel } from "../lib/diff.js";
import { groupTurns, fmtWorked, fmtElapsed, stepLabel, turnDurationMs, firstTs, dayKey, fmtDayMark, workingIndex } from "../lib/turns.js";
import { IconCopy } from "./Icons.jsx";
import { isSearchTool, hitsFromTool, searchQuery } from "../lib/searchCards.js";
import { mdComponents } from "./SourceBlock.jsx";
import { api } from "../lib/api.js";

export default function Conversation({ items, onToggleTool, onToggleFiles, convRef, onScroll, hidden, streaming, agentId }) {
  const turns = groupTurns((items || []).filter((it) => it.kind !== "sys" || it.err));
  const busy = workingIndex(turns, !!streaming);
  return (
    <div id="conversation" className="conversation" ref={convRef} onScroll={onScroll} style={{ visibility: hidden ? "hidden" : "visible" }}>
      <div className="conv-col">
        {turns.reduce((acc, t, i) => {
          if (t.kind === "loose") {
            acc.nodes.push(<Loose key={"l" + i} it={t.item} items={items} onToggleFiles={onToggleFiles} />);
          } else {
            const n = acc.n++;
            const live = i === busy;
            const queued = !!streaming && !live && t.replies.length === 0 && t.work.length === 0 && !!t.user;
            const ts = firstTs(t);
            const day = dayKey(ts);
            if (day && day !== acc.day) {
              acc.nodes.push(<div key={"d" + n} className="day-mark">{fmtDayMark(ts)}</div>);
              acc.day = day;
            }
            acc.nodes.push(<Turn key={"t" + n} turn={t} i={n} live={live} queued={queued} onToggleTool={onToggleTool} agentId={agentId} />);
          }
          return acc;
        }, { n: 0, day: "", nodes: [] }).nodes}
      </div>
    </div>
  );
}

function Loose({ it, items, onToggleFiles }) {
  if (it.kind === "sys") {
    return <div className={"sys-line" + (it.err ? " err" : "")}>{it.text}</div>;
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

function Turn({ turn, i, live, queued, onToggleTool, agentId }) {
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
      {turn.user ? <Block it={turn.user} railId={"turn-" + i + "-user"} /> : null}
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

function Block({ it, railId, agentId }) {
  const user = it.cls === "user";
  const md = !user && it.cls !== "thinking";
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
        {md ? <Markdown components={mdComponents({ CopyBtn, onRun: agentId ? onRun : null })}>{it.text || ""}</Markdown> : it.text}
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
