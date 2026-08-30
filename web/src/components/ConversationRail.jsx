import { memo, useEffect, useState } from "react";
import * as Tooltip from "@radix-ui/react-tooltip";
import { railAnchors } from "../lib/rail.js";

function ConversationRail({ items, convRef }) {
  const anchors = railAnchors(items);
  const [on, setOn] = useState(false);
  const [active, setActive] = useState("");

  useEffect(() => {
    const root = convRef && convRef.current;
    if (!root) return;
    function measure() {
      const show = anchors.length >= 2 && root.scrollHeight > root.clientHeight + 24;
      setOn(show);
      root.classList.toggle("with-rail", show);
      let best = "";
      let bestDist = Infinity;
      const mid = root.getBoundingClientRect().top + Math.min(120, root.clientHeight * 0.25);
      for (const n of root.querySelectorAll("[data-rail]")) {
        const d = Math.abs(n.getBoundingClientRect().top - mid);
        if (d < bestDist) { bestDist = d; best = n.getAttribute("data-rail") || ""; }
      }
      setActive(best);
    }
    measure();
    root.addEventListener("scroll", measure, { passive: true });
    const ro = typeof ResizeObserver !== "undefined" ? new ResizeObserver(measure) : null;
    if (ro) ro.observe(root);
    return () => {
      root.removeEventListener("scroll", measure);
      if (ro) ro.disconnect();
    };
  }, [anchors.length, items, convRef]);

  if (!on) return null;

  function jump(id) {
    const root = convRef.current;
    const el = root && root.querySelector('[data-rail="' + id + '"]');
    if (el) el.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function page(dir) {
    const root = convRef.current;
    if (!root) return;
    root.scrollBy({ top: dir * root.clientHeight * 0.85, behavior: "smooth" });
  }

  return (
    <Tooltip.Provider delayDuration={200}>
      <div className="conv-rail">
        <div className="conv-rail-stack">
          <button type="button" className="rail-chev" aria-label="Scroll up" onClick={() => page(-1)}>
            <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true"><path d="M2 8l4-4 4 4" fill="none" stroke="currentColor" strokeWidth="1.5" /></svg>
          </button>
          <div className="rail-ticks" role="navigation" aria-label="Conversation">
            {anchors.map((a) => (
              <Tooltip.Root key={a.id}>
                <Tooltip.Trigger asChild>
                  <button
                    type="button"
                    className={"rail-tick" + (a.id === active ? " active" : "") + (a.cls === "user" ? " user" : "")}
                    aria-label={a.actor}
                    onClick={() => jump(a.id)}
                  />
                </Tooltip.Trigger>
                <Tooltip.Portal>
                  <Tooltip.Content className="rail-pop" side="left" sideOffset={10} collisionPadding={8}>
                    <div className="rail-pop-actor">{a.actor}</div>
                    <div className="rail-pop-text">{a.preview}</div>
                  </Tooltip.Content>
                </Tooltip.Portal>
              </Tooltip.Root>
            ))}
          </div>
          <button type="button" className="rail-chev" aria-label="Scroll down" onClick={() => page(1)}>
            <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true"><path d="M2 4l4 4 4-4" fill="none" stroke="currentColor" strokeWidth="1.5" /></svg>
          </button>
        </div>
      </div>
    </Tooltip.Provider>
  );
}

export default memo(ConversationRail);
