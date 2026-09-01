import { useEffect, useState } from "react";
import { mobileRoute, mobileHash, parentHash } from "../../lib/mobileRoutes.js";

// Hash routing for the phone (ADR-0044). Tabs replace the current entry
// (siblings, not history); pushed screens add one, so the hardware Back
// closes the agent screen instead of leaving the PWA. The on-screen Back
// is deterministic: it goes to the parent, never "wherever history was".
export function useHashRoute() {
  const [route, setRoute] = useState(() => mobileRoute(location.hash));
  useEffect(() => {
    const on = () => setRoute(mobileRoute(location.hash));
    window.addEventListener("hashchange", on);
    return () => window.removeEventListener("hashchange", on);
  }, []);
  return route;
}

export function goTab(screen) {
  const h = mobileHash(screen);
  if (location.hash === h) return;
  location.replace(h);
}

export function push(hash) {
  if (location.hash === hash) return;
  location.hash = hash;
}

export function goBack(route) {
  location.replace(parentHash(route));
}
