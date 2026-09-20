// Health + boot watch. Two failure modes:
//  1. server down for a while → miss → Reconnecting overlay → reload on up
//  2. server restarted FAST (between polls) → WS dies, health never failed →
//     compare bootId: a change means a new binary → reload immediately.
// Any unexpected WS close should call window.__picodeKickHealth() so case 2
// is caught within ~1s instead of the next poll.
//
// Both reloads are gated: during a restart window the page itself can
// answer 404, and reloading into that body leaves the shell parked on it
// forever (2026-09-18, back-to-back deploys bricked the desktop app). The
// reload waits — bounded — for this page's own path to answer 200 again.

export async function pageServing(fetchImpl = fetch, path) {
  try {
    const target = path ?? (typeof location === "undefined" ? "/" : location.pathname);
    const res = await fetchImpl(target, { cache: "no-store" });
    return !!res && res.ok;
  } catch {
    return false;
  }
}

async function reloadWhenServing(reload, probe, waitMs) {
  for (let i = 0; i < 20; i++) {
    if (await probe()) break;
    await new Promise((r) => setTimeout(r, waitMs));
  }
  reload();
}

export async function pingHealth(fetchImpl = fetch) {
  try {
    const c = new AbortController();
    const t = setTimeout(() => c.abort(), 2500);
    const res = await fetchImpl("/api/health", { cache: "no-store", signal: c.signal });
    clearTimeout(t);
    if (!res || !res.ok) return null;
    const body = await res.json().catch(() => null);
    return (body && body.bootId) || "ok";
  } catch {
    return null;
  }
}

export function startReconnectWatch({
  ping = pingHealth,
  reload = defaultReload,
  probe,
  waitMs = 750,
  onState,
  downAfter = 1,
  okMs = 2500,
  downMs = 800,
} = {}) {
  const reloadServing = () => reloadWhenServing(reload, probe, waitMs);
  let fails = 0;
  let down = false;
  let timer = 0;
  let stopped = false;
  let boot = null;
  let pace = okMs;

  async function tick() {
    if (stopped) return;
    const res = await ping();
    if (stopped) return;
    if (res !== null) {
      fails = 0;
      if (boot === null) {
        boot = res;
      } else if (res !== boot) {
        // Fast restart: never saw downtime, but this is a new process.
        if (onState) onState("up");
        reloadServing();
        return;
      }
      if (down) {
        if (onState) onState("up");
        reloadServing();
        return;
      }
      if (onState) onState("ok");
    } else {
      fails += 1;
      if (!down && fails >= downAfter) {
        down = true;
        if (onState) onState("down");
      }
    }
    timer = setTimeout(tick, down ? downMs : pace);
  }

  // setPace: the change feed (ADR-0048) already detects a new binary and
  // a lost server, so while it is connected the health poll can idle;
  // any feed error kicks an immediate tick regardless of pace.
  function setPace(ms) {
    pace = Math.max(okMs, ms || okMs);
  }

  function kick() {
    clearTimeout(timer);
    tick();
  }

  function onOffline() {
    fails = downAfter;
    if (!down) {
      down = true;
      if (onState) onState("down");
    }
    kick();
  }

  tick();
  if (typeof window !== "undefined") {
    window.__picodeKickHealth = kick;
    window.__picodeHealthPace = setPace;
    window.addEventListener("offline", onOffline);
    window.addEventListener("online", kick);
    if (typeof document !== "undefined") document.addEventListener("visibilitychange", kick);
  }
  return () => {
    stopped = true;
    clearTimeout(timer);
    if (typeof window !== "undefined") {
      delete window.__picodeKickHealth;
      delete window.__picodeHealthPace;
      window.removeEventListener("offline", onOffline);
      window.removeEventListener("online", kick);
      if (typeof document !== "undefined") document.removeEventListener("visibilitychange", kick);
    }
  };
}

function defaultReload() {
  location.reload();
}
