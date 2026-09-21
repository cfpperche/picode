import { useLayoutEffect, useRef, useState } from "react";
import { DndContext, DragOverlay, PointerSensor, closestCenter, useSensor, useSensors } from "@dnd-kit/core";
import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { reorderIds } from "../lib/sidebarOrder.js";

// Official vertical-list lock from @dnd-kit/modifiers (restrictToVerticalAxis).
// The pointer delta's x is discarded, so a sideways nudge cannot translate a
// full-width row. A translateX widens the scrollport: overflow-y: auto
// computes overflow-x to auto, and the bar in the screenshot is that width.
const restrictToVerticalAxis = ({ transform }) => ({ ...transform, x: 0 });

// One list, one context, so a row cannot drop into a neighbouring list
// (agents stay agents, terminals stay terminals, workspace groups stay
// groups). Pointer only: the row already uses Space and Enter to open,
// and Move up / Move down in its menu is the keyboard path (WCAG 2.5.7).

function reducedMotion() {
  return typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

// Neighbours ease into the gap. 160ms, no bounce — inside the 100–200ms bar.
const ROW_EASE = { duration: 160, easing: "cubic-bezier(0.2, 0, 0, 1)" };

function dragIdOf(id) {
  return String(id).replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

// The overlay is the lifted card. The in-list copy stays put as the slot,
// translated by the sortable strategy to the index where the row will land.
function LiftedRow({ id }) {
  const host = useRef(null);
  useLayoutEffect(() => {
    const src = document.querySelector('[data-drag-id="' + dragIdOf(id) + '"]');
    const el = host.current;
    if (!src || !el) return;
    const clone = src.cloneNode(true);
    clone.removeAttribute("data-drag-id");
    clone.classList.remove("is-placeholder");
    clone.classList.add("is-lifted");
    clone.style.transform = "none";
    clone.style.transition = "none";
    clone.style.width = "100%";
    clone.style.margin = "0";
    el.replaceChildren(clone);
  }, [id]);
  return <div ref={host} className="dnd-lift-host" />;
}

// A drag that ends still emits the click that started it. Swallow that one
// click so a workspace header does not collapse and a row does not open.
function swallowFollowingClick(event) {
  let target = event.activatorEvent && event.activatorEvent.target;
  if (target && target.nodeType !== 1) target = target.parentElement;
  if (!target || !target.addEventListener) return;
  const stop = (ev) => {
    ev.preventDefault();
    ev.stopPropagation();
    target.removeEventListener("click", stop, true);
  };
  target.addEventListener("click", stop, true);
}

export function SortableList({ ids, onReorder, children }) {
  const sensorList = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));
  const [activeId, setActiveId] = useState(null);
  const quiet = reducedMotion();
  function endDrag(event) {
    swallowFollowingClick(event);
    document.body.classList.remove("is-row-dragging");
    const active = String(event.active.id);
    setActiveId(null);
    const next = reorderIds(ids, active, event.over ? String(event.over.id) : "");
    if (next !== ids && onReorder) onReorder(next, active);
  }
  return (
    <DndContext
      sensors={sensorList}
      collisionDetection={closestCenter}
      modifiers={[restrictToVerticalAxis]}
      // Vertical autoscroll only. threshold.x 0 divides by zero and scrolls
      // sideways; layout-shift compensation on x does the same when a row
      // sticks out (dnd-kit 6.1+).
      autoScroll={{ threshold: { x: 0.2, y: 0.2 }, layoutShiftCompensation: { x: false, y: true } }}
      onDragStart={(event) => {
        setActiveId(String(event.active.id));
        document.body.classList.add("is-row-dragging");
      }}
      onDragCancel={(event) => {
        document.body.classList.remove("is-row-dragging");
        setActiveId(null);
        swallowFollowingClick(event);
      }}
      onDragEnd={endDrag}
    >
      <SortableContext items={ids} strategy={verticalListSortingStrategy}>
        {children}
      </SortableContext>
      <DragOverlay
        modifiers={[restrictToVerticalAxis]}
        zIndex={40}
        dropAnimation={quiet ? null : { duration: 160, easing: ROW_EASE.easing }}
      >
        {activeId ? <LiftedRow id={activeId} /> : null}
      </DragOverlay>
    </DndContext>
  );
}

export function SortableRow({ id, children }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    transition: reducedMotion() ? null : ROW_EASE,
  });
  const style = {
    // Translate, not Transform: scaleX on a full-width row also overflows.
    // x stays 0 even if a modifier is skipped. With an overlay mounted, this
    // transform slides the placeholder into the slot, not the lifted card.
    transform: CSS.Translate.toString(transform ? { ...transform, x: 0 } : null),
    transition: reducedMotion() ? "none" : transition,
  };
  return children({
    setNodeRef,
    style,
    isDragging,
    onPointerDown: listeners ? listeners.onPointerDown : undefined,
    describedBy: attributes["aria-describedby"],
  });
}
