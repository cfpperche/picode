import { useState } from "react";
import { basename, statLabel } from "../lib/diff.js";
import { IconCopy } from "./Icons.jsx";

export default function Conversation({ items, onToggleTool, onToggleFiles, convRef, onScroll, hidden }) {
  return (
    <div id="conversation" className="conversation" ref={convRef} onScroll={onScroll} style={{ visibility: hidden ? "hidden" : "visible" }}>
      <div className="conv-col">
        {items.map((it, i) => {
          if (it.kind === "sys") {
            return <div key={i} className={"sys-line" + (it.err ? " err" : "")}>{it.text}</div>;
          }
          if (it.kind === "files") {
            return (
              <div key={i} className={"files-changed" + (it.expanded ? " expanded" : "")}>
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
          if (it.kind === "tool") {
            const ch = it.change;
            return (
              <div key={it.id} className={"tool-pill" + (it.status === "ok" ? " ok" : "") + (it.status === "error" ? " err" : "") + (it.expanded ? " expanded" : "")}>
                <div className="tool-pill-head" onClick={() => onToggleTool(it.id)}>
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
          return (
            <div key={i} className={"block " + (it.cls || "")}>
              <div className="actor">
                {it.actor}
                {it.chip ? <span className="chip">{it.chip}</span> : null}
                {it.cls !== "user" && it.cls !== "thinking" && it.text ? <CopyBtn text={it.text} /> : null}
              </div>
              <div className="block-content">{it.text}</div>
            </div>
          );
        })}
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
