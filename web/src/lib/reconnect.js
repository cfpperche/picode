export async function pingHealth(fetchImpl = fetch) {
  try {
    const c = new AbortController();
    const t = setTimeout(() => c.abort(), 2500);
    const res = await fetchImpl("/api/health", { cache: "no-store", signal: c.signal });
    clearTimeout(t);
    return !!res && res.ok;
  } catch {
    return false;
  }
}

// Poll /api/health. After `downAfter` misses, onState("down"). When it
// comes back, onState("up") then reload() so hashed assets match the new binary.
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

  async function tick() {
    if (stopped) return;
    const ok = await ping();
    if (stopped) return;
    if (ok) {
      fails = 0;
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
    window.addEventListener("offline", onOffline);
    window.addEventListener("online", kick);
    document.addEventListener("visibilitychange", kick);
  }
  return () => {
    stopped = true;
    clearTimeout(timer);
    if (typeof window !== "undefined") {
      window.removeEventListener("offline", onOffline);
      window.removeEventListener("online", kick);
      document.removeEventListener("visibilitychange", kick);
    }
  };
}

function defaultReload() {
  location.reload();
}
