import { useState } from "react";
import * as Switch from "@radix-ui/react-switch";

// The settings-page row controls (the ChatGPT Work Browser shape the owner
// asked for, 2026-09-14), shared by Settings ▸ Browser and Settings ▸
// Computer. The CSS (`.set-*` in styles/app.css) was always shared; the JSX
// moved here when the second page arrived (ADR-0148).

// One row of a section card: title + description left, control right.
export function Item({ title, desc, children }) {
  return (
    <div className="set-item">
      <div className="set-item-body">
        <span className="set-item-t">{title}</span>
        {desc ? <span className="set-item-d">{desc}</span> : null}
      </div>
      <div className="set-item-ctl">{children}</div>
    </div>
  );
}

export function SwitchCtl({ checked, onChange, label }) {
  return (
    <Switch.Root className="rx-switch" checked={!!checked} onCheckedChange={onChange} aria-label={label}>
      <Switch.Thumb className="rx-switch-thumb" />
    </Switch.Root>
  );
}

// Workspace provenance (VS Code marker language): which workspace a grant
// row belongs to. `name` resolves via GET /api/workspaces; the raw id stays
// in `title` so a truncated chip never hides the truth.
export function WsTag({ id, name }) {
  if (!id) return <span className="grant-ws grant-ws-none" title="No workspace">No workspace</span>;
  return <span className="grant-ws" title={id}>{name || id}</span>;
}

// Expandable audit list (Stripe log language). `rows` carry a stable `key`;
// the accessors keep browser Raw calls and computer Recent steps on one
// renderer. `detail` returns the expanded node, or null for no expansion.
export function AuditList({ rows, method, outcome, actor, when, detail, empty }) {
  const [open, setOpen] = useState(() => new Set());
  if (!rows || rows.length === 0) {
    return (
      <div className="set-item"><div className="set-item-body"><span className="set-item-d">{empty || "Nothing yet."}</span></div></div>
    );
  }
  const toggle = (key) => setOpen((cur) => {
    const next = new Set(cur);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    return next;
  });
  return (
    <ul className="set-audit">
      {rows.map((row) => {
        const key = row.key;
        const isOpen = open.has(key);
        const body = detail ? detail(row) : null;
        const out = outcome(row);
        const bad = String(out).toLowerCase() !== "allowed";
        return (
          <li key={key} className={"set-audit-row" + (isOpen ? " is-open" : "")}>
            <button
              type="button"
              className="set-audit-head"
              aria-expanded={body ? isOpen : undefined}
              onClick={body ? () => toggle(key) : undefined}
              style={body ? undefined : { cursor: "default" }}
            >
              <code className="set-audit-method">{method(row)}</code>
              <span className={"set-audit-outcome" + (bad ? " is-bad" : " is-ok")}>{out}</span>
              <span className="set-audit-actor">{actor(row)}</span>
              <span className="set-audit-when">{when(row)}</span>
              <span className="set-audit-toggle" aria-hidden="true">{body ? "▶" : ""}</span>
            </button>
            {isOpen && body ? <div className="set-audit-detail">{body}</div> : null}
          </li>
        );
      })}
    </ul>
  );
}
