import { useEffect, useRef } from "react";
import { IconAgent } from "./Icons.jsx";
import "../styles/mobile-settings.css";

export default function PageFrame({ id, title, context, children, hidden, wide, embedded, className = "" }) {
  const rootRef = useRef(null);
  useEffect(() => {
    if (hidden) return undefined;
    let frame;
    const revealTab = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => {
        for (const nav of rootRef.current?.querySelectorAll(".pref-tabs, .cli-tabs, .cli-pane-tabs, .llama-nav") || []) {
          const selected = nav.querySelector('[aria-selected="true"], [aria-current="page"], .active');
          if (!selected) continue;
          const item = selected.getBoundingClientRect();
          const rail = nav.getBoundingClientRect();
          if (item.left < rail.left) nav.scrollLeft += item.left - rail.left;
          else if (item.right > rail.right) nav.scrollLeft += item.right - rail.right;
        }
      });
    };
    revealTab();
    window.addEventListener("hashchange", revealTab);
    return () => { cancelAnimationFrame(frame); window.removeEventListener("hashchange", revealTab); };
  }, [hidden]);
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
