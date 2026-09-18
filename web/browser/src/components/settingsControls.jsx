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
