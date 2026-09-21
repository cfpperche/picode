import { DndContext, PointerSensor, closestCenter, useSensor, useSensors } from "@dnd-kit/core";
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
  return (
    <DndContext
      sensors={sensorList}
      collisionDetection={closestCenter}
      modifiers={[restrictToVerticalAxis]}
      // Vertical autoscroll only. threshold.x 0 divides by zero and scrolls
      // sideways; layout-shift compensation on x does the same when a row
      // sticks out (dnd-kit 6.1+).
      autoScroll={{ threshold: { x: 0.2, y: 0.2 }, layoutShiftCompensation: { x: false, y: true } }}
      onDragEnd={(event) => {
        swallowFollowingClick(event);
        const activeId = String(event.active.id);
        const next = reorderIds(ids, activeId, event.over ? String(event.over.id) : "");
        if (next !== ids && onReorder) onReorder(next, activeId);
      }}
    >
      <SortableContext items={ids} strategy={verticalListSortingStrategy}>
        {children}
      </SortableContext>
    </DndContext>
  );
}

export function SortableRow({ id, children }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });
  const style = {
    // Translate, not Transform: scaleX on a full-width row also overflows.
    // x stays 0 even if a modifier is skipped.
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
