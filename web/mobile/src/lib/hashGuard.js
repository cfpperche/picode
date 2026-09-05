let activeGuard = null;

// Install before React mounts its route observers. Window-targeted events run
// listeners in registration order, even when a later listener uses capture.
export function installHashGuard(target = window) {
  const onHash = (event) => activeGuard?.(event);
  target.addEventListener("hashchange", onHash);
  return () => target.removeEventListener("hashchange", onHash);
}

export function registerHashGuard(guard) {
  activeGuard = guard;
  return () => { if (activeGuard === guard) activeGuard = null; };
}
