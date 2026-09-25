import { useRef } from "react";
import { IconAgent } from "./Icons.jsx";
import "../styles/mobile-settings.css";

export default function PageFrame({ id, title, context, children, hidden, wide, embedded, className = "" }) {
  // Tab bars inside a page (pref-tabs, cli-tabs, cli-pane-tabs, llama-nav)
  // reveal their own selected tab: OverflowTabs.
  const rootRef = useRef(null);
  if (embedded) return <section ref={rootRef} id={id} hidden={hidden} aria-label={title}>{context && <p className="settings-ctx">{context}</p>}{children}</section>;
  return (
    <section ref={rootRef} id={id} className={"pane-view" + (className ? " " + className : "")} hidden={hidden} aria-label={title}>
      <div className={"settings-wrap" + (wide ? " wide" : "")}>
        <div className="settings-card">
          {context ? (
            <p className="settings-ctx" title={context}>
              <IconAgent />
              <span>{context}</span>
            </p>
          ) : null}
          {children}
        </div>
      </div>
    </section>
  );
}
