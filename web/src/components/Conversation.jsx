export default function Conversation({ items, onToggleTool, convRef, onScroll, hidden }) {
  return (
    <div id="conversation" className="conversation" ref={convRef} onScroll={onScroll} style={{ visibility: hidden ? "hidden" : "visible" }}>
      <div className="conv-col">
        {items.map((it, i) => {
          if (it.kind === "sys") {
            return <div key={i} className={"sys-line" + (it.err ? " err" : "")}>{it.text}</div>;
          }
          if (it.kind === "tool") {
            return (
              <div key={it.id} className={"tool-pill" + (it.status === "ok" ? " ok" : "") + (it.status === "error" ? " err" : "") + (it.expanded ? " expanded" : "")}>
                <div className="tool-pill-head" onClick={() => onToggleTool(it.id)}>
                  <span className="tp-chevron">›</span>
                  <span className="tp-name">{it.name}</span>
                  <span className="tp-args">{it.args}</span>
                  <span className="tp-status">{it.status || "···"}</span>
                </div>
                <div className="tp-detail">{it.detail}</div>
              </div>
            );
          }
          return (
            <div key={i} className={"block " + (it.cls || "")}>
              <div className="actor">
                {it.actor}
                {it.chip ? <span className="chip">{it.chip}</span> : null}
              </div>
              <div className="block-content">{it.text}</div>
            </div>
          );
        })}
      </div>
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
