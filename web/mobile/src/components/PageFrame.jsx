import { useEffect, useRef } from "react";
import { IconAgent } from "./Icons.jsx";
import "../styles/mobile-settings.css";

export default function PageFrame({ id, title, context, children, hidden, wide, embedded, className = "" }) {
  const rootRef = useRef(null);
  useEffect(() => {
    if (hidden) return undefined;
    let frame;
    const navs = () => rootRef.current?.querySelectorAll(".pref-tabs, .cli-tabs, .cli-pane-tabs, .llama-nav") || [];
    // mask hides a tab that is only partly in view (a clipped label reads as
    // a different word); a fully visible one is shown again. It runs on every
    // scroll of the strip: masking only on mount and hash changes left a tab
    // hidden after the owner swiped it fully into view — invisible, so it
    // could not be tapped (Preferences → Landing work, 2026-09-25).
    const mask = (nav) => {
      const rail = nav.getBoundingClientRect();
      for (const el of nav.querySelectorAll('[role="tab"]')) {
        const box = el.getBoundingClientRect();
        const inView = box.right > rail.left + 1 && box.left < rail.right - 1;
        const fully = box.left >= rail.left - 1 && box.right <= rail.right + 1;
        el.style.visibility = inView && !fully ? "hidden" : "";
      }
    };
    // revealTab also brings the selected tab into view — on mount, on a hash
    // change and when the selection changes, never on the owner's own scroll
    // (that would pull the strip back under their finger).
    const revealTab = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => {
        for (const nav of navs()) {
          const selected = nav.querySelector('[aria-selected="true"], [aria-current="page"], .active');
          if (selected) {
            const item = selected.getBoundingClientRect();
            const rail = nav.getBoundingClientRect();
            if (item.left < rail.left) nav.scrollLeft += item.left - rail.left;
            else if (item.right > rail.right) nav.scrollLeft += item.right - rail.right;
          }
          mask(nav);
        }
      });
    };
    const onScroll = (e) => {
      const nav = e.target;
      if (!(nav instanceof Element) || !nav.matches(".pref-tabs, .cli-tabs, .cli-pane-tabs, .llama-nav")) return;
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => mask(nav));
    };
    // The selection lives in component state on the phone (no hash), so a
    // tab change is seen as the selected attribute moving.
    const seen = new MutationObserver(revealTab);
    for (const nav of navs()) seen.observe(nav, { subtree: true, attributes: true, attributeFilter: ["aria-selected", "aria-current", "class"] });
    const root = rootRef.current;
    revealTab();
    window.addEventListener("hashchange", revealTab);
    window.addEventListener("resize", revealTab);
    root?.addEventListener("scroll", onScroll, true);
    return () => {
      cancelAnimationFrame(frame);
      seen.disconnect();
      window.removeEventListener("hashchange", revealTab);
      window.removeEventListener("resize", revealTab);
      root?.removeEventListener("scroll", onScroll, true);
    };
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
