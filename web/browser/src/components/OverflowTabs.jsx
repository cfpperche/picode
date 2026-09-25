import { useLayoutEffect, useRef } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { revealLeft } from "../lib/tabStrip.js";
import { useTabStrip } from "../lib/useTabStrip.js";
import { IconChevronLeft, IconChevronRight, IconList, IconCheck } from "./Icons.jsx";

// The editor tab strip's overflow chrome (AgentTabs, docs/benchmarks/
// 2026-09-06-tab-strip-overflow.md) for the in-page tab bars: no
// scrollbar, and while the tabs overflow an edge fade on the side that
// still has tabs, arrows that step most of a viewport, and an "All tabs"
// list with the out-of-view tabs first. The selected tab is brought into
// view by code. Unlike the editor strip, the vertical wheel stays the
// page's, and there is no position indicator (inside a card it read as a
// stray scrollbar thumb).
//
// Children are the tabs themselves; each carries `data-tab={id}` and sits
// directly in the strip. `items` ({ id, label, href? }) feed the list: a
// pick follows `href` when the tab is a link, else calls `onPick(id)`.
export default function OverflowTabs({ as: Strip = "nav", className = "", frameClassName = "", label, role, items, selectedId, onPick, listLabel = "All tabs", children }) {
  const ref = useRef(null);
  const revealed = useRef(false);
  const key = items.map((i) => i.id + "\u0000" + i.label).join("\u0001");
  const strip = useTabStrip(ref, [key], { wheel: false });

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el || selectedId == null) return;
    const tab = [...el.children].find((c) => c.dataset && c.dataset.tab === String(selectedId));
    if (!tab) return;
    const left = revealLeft(el, { left: tab.offsetLeft, width: tab.offsetWidth });
    // The first reveal jumps: a glide from 0 on page load is motion nobody asked for.
    if (left != null) el.scrollTo(revealed.current ? { left } : { left, behavior: "instant" });
    revealed.current = true;
  }, [selectedId, key, strip.overflow]);

  const pick = (item) => {
    if (item.href) window.location.assign(item.href);
    else if (onPick) onPick(item.id);
  };
  const hidden = new Set([...strip.hidden.left, ...strip.hidden.right]);
  const outOfView = items.filter((i) => hidden.has(String(i.id)));
  const inView = items.filter((i) => !hidden.has(String(i.id)));
  const listItem = (item) => (
    <DropdownMenu.Item key={item.id} className="um-item tab-list-item" onSelect={() => pick(item)} aria-current={item.id === selectedId ? "true" : undefined}>
      <span className="um-item-name"><span className="um-name">{item.label}</span></span>
      {item.id === selectedId ? <IconCheck size={13} /> : null}
    </DropdownMenu.Item>
  );

  return (
    <div className={"ovt" + (frameClassName ? " " + frameClassName : "")} data-overflow={strip.overflow ? "" : undefined}>
      {strip.overflow ? (
        <button type="button" className="ovt-arrow" disabled={strip.atStart} title="Scroll tabs left" aria-label="Scroll tabs left" onClick={() => strip.step(-1)}>
          <IconChevronLeft />
        </button>
      ) : null}
      <div className="ovt-scroller">
        <Strip ref={ref} className={"ovt-strip" + (className ? " " + className : "")} role={role} aria-label={label} data-at-start={strip.atStart} data-at-end={strip.atEnd}>
          {children}
        </Strip>
      </div>
      {strip.overflow ? (
        <button type="button" className="ovt-arrow" disabled={strip.atEnd} title="Scroll tabs right" aria-label="Scroll tabs right" onClick={() => strip.step(1)}>
          <IconChevronRight />
        </button>
      ) : null}
      {strip.overflow ? (
        <DropdownMenu.Root>
          <DropdownMenu.Trigger asChild>
            <button type="button" className="ovt-arrow" title={listLabel} aria-label={listLabel}>
              <IconList />
            </button>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content className="um-popover tab-list" side="bottom" align="end" sideOffset={4} collisionPadding={8}>
              {outOfView.length ? <div className="um-label">Out of view</div> : null}
              {outOfView.map(listItem)}
              {outOfView.length && inView.length ? <DropdownMenu.Separator className="um-divider" /> : null}
              {inView.map(listItem)}
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      ) : null}
    </div>
  );
}
