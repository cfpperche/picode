import { useCallback, useLayoutEffect, useRef } from "react";

// A tab that is only hidden keeps its component mounted — but display:none
// drops the scroll box, so the offset has to be captured as it happens and put
// back before the reveal paints. Reading it once the tab is already hidden
// reads 0, and restoring it after the paint flashes the top of the list.
//
// Returns a ref for the surface root. `scroll` does not bubble, so the listener
// is a capturing one: every descendant matching a selector keeps its own
// offset, without threading a ref through the components in between.
export function useKeptScroll(hidden, selectors) {
  const tops = useRef(new Map());
  const nodeRef = useRef(null);
  const keys = selectors.join(",");

  const onScroll = useCallback(
    (e) => {
      const el = e.target;
      if (!el || typeof el.matches !== "function") return;
      for (const sel of keys.split(",")) {
        if (el.matches(sel)) {
          tops.current.set(sel, el.scrollTop);
          return;
        }
      }
    },
    [keys],
  );

  // A callback ref, not useRef: a surface swaps its root element between the
  // skeleton, error and loaded branches, and the listener has to follow.
  const rootRef = useCallback(
    (node) => {
      if (nodeRef.current) nodeRef.current.removeEventListener("scroll", onScroll, true);
      nodeRef.current = node;
      if (node) node.addEventListener("scroll", onScroll, true);
    },
    [onScroll],
  );

  useLayoutEffect(() => {
    if (hidden || !nodeRef.current) return;
    for (const [sel, top] of tops.current) {
      const el = nodeRef.current.querySelector(sel);
      if (el) el.scrollTop = top;
    }
  }, [hidden]);

  return rootRef;
}
