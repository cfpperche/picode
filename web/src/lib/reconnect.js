// Health + boot watch. Two failure modes:
//  1. server down for a while → miss → Reconnecting overlay → reload on up
//  2. server restarted FAST (between polls) → WS dies, health never failed →
//     compare bootId: a change means a new binary → reload immediately.
// Any unexpected WS close should call window.__picodeKickHealth() so case 2
// is caught within ~1s instead of the next poll.

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
  onState,
  downAfter = 1,
  okMs = 2500,
  downMs = 800,
} = {}) {
  let fails = 0;
  let down = false;
  let timer = 0;
  let stopped = false;
  let boot = null;

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
        reload();
        return;
      }
      if (down) {
        if (onState) onState("up");
        reload();
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
    timer = setTimeout(tick, down ? downMs : okMs);
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
    window.addEventListener("offline", onOffline);
    window.addEventListener("online", kick);
    if (typeof document !== "undefined") document.addEventListener("visibilitychange", kick);
  }
  return () => {
    stopped = true;
    clearTimeout(timer);
    if (typeof window !== "undefined") {
      delete window.__picodeKickHealth;
      window.removeEventListener("offline", onOffline);
      window.removeEventListener("online", kick);
      if (typeof document !== "undefined") document.removeEventListener("visibilitychange", kick);
    }
  };
}

function defaultReload() {
  location.reload();
}
