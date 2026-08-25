import { useState } from "react";
import Markdown from "react-markdown";
import { basename, statLabel } from "../lib/diff.js";
import { groupTurns, fmtWorked, stepLabel, turnDurationMs } from "../lib/turns.js";
import { IconCopy } from "./Icons.jsx";

export default function Conversation({ items, onToggleTool, onToggleFiles, convRef, onScroll, hidden }) {
  const turns = groupTurns(items);
  return (
    <div id="conversation" className="conversation" ref={convRef} onScroll={onScroll} style={{ visibility: hidden ? "hidden" : "visible" }}>
      <div className="conv-col">
        {turns.reduce((acc, t, i) => {
          if (t.kind === "loose") {
            acc.nodes.push(<Loose key={"l" + i} it={t.item} items={items} onToggleFiles={onToggleFiles} />);
          } else {
            const n = acc.n++;
            acc.nodes.push(<Turn key={"t" + n} turn={t} i={n} onToggleTool={onToggleTool} />);
          }
          return acc;
        }, { n: 0, nodes: [] }).nodes}
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

function Turn({ turn, i, onToggleTool }) {
  const live = turn.work.length > 0 && turn.replies.length === 0;
  const [open, setOpen] = useState(live);
  const shown = live || open;
  return (
    <div className="turn" id={"turn-" + i} data-rail={"turn-" + i}>
      {turn.user ? <Block it={turn.user} /> : null}
      {turn.work.length > 0 ? (
        <div className={"work" + (shown ? " open" : "")}>
          <button type="button" className="work-head" onClick={() => setOpen((v) => !v)}>
            <span>{live ? "Working…" : fmtWorked(turnDurationMs(turn))}</span>
            <span className="tp-chevron">›</span>
          </button>
          {shown ? (
            <ol className="work-steps">
              {turn.work.map((it, j) => (
                <li key={it.id || j} className="work-step">
                  <span className="work-step-lab">{stepLabel(it)}</span>
                  {it.kind === "tool" ? <Tool it={it} onToggle={onToggleTool} /> : null}
                </li>
              ))}
            </ol>
          ) : null}
        </div>
      ) : null}
      {turn.replies.map((it, j) => <Block key={j} it={it} />)}
    </div>
  );
}

function Block({ it }) {
  return (
    <div className={"block " + (it.cls || "")}>
      <div className="actor">
        {it.actor}
        {it.chip ? <span className="chip">{it.chip}</span> : null}
        {it.cls !== "user" && it.cls !== "thinking" && it.text ? <CopyBtn text={it.text} /> : null}
      </div>
      <div className={"block-content" + (it.cls !== "user" && it.cls !== "thinking" ? " md" : "")}>
        {it.cls !== "user" && it.cls !== "thinking" ? <Markdown>{it.text || ""}</Markdown> : it.text}
      </div>
    </div>
  );
}

function Tool({ it, onToggle }) {
  const ch = it.change;
  return (
    <div className={"tool-pill" + (it.status === "ok" ? " ok" : "") + (it.status === "error" ? " err" : "") + (it.expanded ? " expanded" : "")}>
      <div className="tool-pill-head" onClick={() => onToggle(it.id)}>
        <span className="tp-chevron">›</span>
        <span className="tp-name">{it.name}</span>
        <span className="tp-args">{ch ? basename(ch.path) || it.args : it.args}</span>
        {ch ? <span className="tp-stat"><span className="add">{ch.add ? "+" + ch.add : ""}</span>{ch.add && ch.del ? " " : ""}<span className="del">{ch.del ? "−" + ch.del : ""}</span></span> : null}
        <span className="tp-status">{it.status || "···"}</span>
      </div>
      <div className={"tp-detail" + (ch ? " tp-diff" : "")}>
        {ch ? <DiffHunks hunks={ch.hunks} /> : it.detail}
      </div>
    </div>
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
  if (typeof args.command === "string") return args.command;
  if (typeof args.path === "string") return args.path;
  const s = JSON.stringify(args);
  return s.length > 2 ? s : "";
}

export { statLabel };
