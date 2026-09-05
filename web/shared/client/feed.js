// Change feed client (ADR-0048): one EventSource per shell. Durable events
// carry an id the browser echoes as Last-Event-ID on reconnect, so a
// backgrounded phone replays what it missed. The cursor also survives a
// reload through sessionStorage. Consumers subscribe to entity types and
// either patch their state (lib/feedReducers.js) or refetch.
//
// Meta events reach subscribers too:
//   feed.open   connected (first time or after a drop): refresh lists once
//   feed.down   stream lost: polls resume until feed.open
//   feed.reset  the server could not replay the cursor: refresh everything

const CURSOR_KEY = "picode-feed-seq";
const BOOT_KEY = "picode-feed-boot";

let source = null;
let connected = false;
let cursor = 0;
const subs = new Set();
let wasOpen = false;

export function feedConnected() {
  return connected;
}

export function feedCursor() {
  return cursor;
}

// subscribeFeed(fn) -> unsubscribe. fn({type, id, data, agentId, workspaceId}).
export function subscribeFeed(fn) {
  subs.add(fn);
  return () => subs.delete(fn);
}

function emit(ev) {
  for (const fn of [...subs]) {
    try { fn(ev); } catch (e) { console.error("feed subscriber:", e); }
  }
}

function readCursor() {
  try { return parseInt(sessionStorage.getItem(CURSOR_KEY) || "0", 10) || 0; } catch { return 0; }
}

function writeCursor(id) {
  cursor = id;
  try { sessionStorage.setItem(CURSOR_KEY, String(id)); } catch { /* private mode */ }
}

function kick() {
  if (typeof window !== "undefined" && typeof window.__picodeKickHealth === "function") window.__picodeKickHealth();
}

// While the stream is up the health watch idles at HEALTH_IDLE_MS; any
// stream error kicks it back to its fast cadence.
const HEALTH_IDLE_MS = 20000;
function pace(connectedNow) {
  if (typeof window !== "undefined" && typeof window.__picodeHealthPace === "function") window.__picodeHealthPace(connectedNow ? HEALTH_IDLE_MS : 0);
}

// parseChange(data, lastEventId) -> event object or null. Exported for tests.
export function parseChange(data, lastEventId) {
  let ev = null;
  try { ev = JSON.parse(data); } catch { return null; }
  if (!ev || typeof ev.type !== "string") return null;
  const id = parseInt(lastEventId || "", 10);
  return { ...ev, id: Number.isFinite(id) && id > 0 ? id : ev.id || 0 };
}

// startFeed({EventSourceImpl, onState}) -> stop(). Idempotent per page.
export function startFeed({ EventSourceImpl, onState } = {}) {
  const ES = EventSourceImpl || (typeof EventSource !== "undefined" ? EventSource : null);
  if (!ES) return () => {};
  if (source) return stopFeed;
  cursor = readCursor();
  const url = "/api/events" + (cursor > 0 ? "?after=" + cursor : "");
  source = new ES(url);

  source.onopen = () => {
    connected = true;
    pace(true);
    const first = !wasOpen;
    wasOpen = true;
    if (onState) onState("open");
    emit({ type: "feed.open", data: { first } });
  };
  source.onerror = () => {
    pace(false);
    if (connected) {
      connected = false;
      if (onState) onState("down");
      emit({ type: "feed.down", data: {} });
    }
    kick();
  };
  source.addEventListener("hello", (e) => {
    let hello = null;
    try { hello = JSON.parse(e.data); } catch { hello = null; }
    if (!hello) return;
    let seen = "";
    try { seen = sessionStorage.getItem(BOOT_KEY) || ""; } catch { /* ignore */ }
    if (seen && hello.bootId && seen !== hello.bootId) {
      // A new binary: the health watch reloads the page. Don't keep a
      // cursor from the old process's log semantics either.
      try { sessionStorage.removeItem(CURSOR_KEY); } catch { /* ignore */ }
      kick();
    }
    try { if (hello.bootId) sessionStorage.setItem(BOOT_KEY, hello.bootId); } catch { /* ignore */ }
  });
  source.addEventListener("reset", () => {
    writeCursor(0);
    emit({ type: "feed.reset", data: {} });
  });
  source.addEventListener("change", (e) => {
    const ev = parseChange(e.data, e.lastEventId);
    if (!ev) return;
    if (ev.id > 0) writeCursor(ev.id);
    emit(ev);
  });
  return stopFeed;
}

export function stopFeed() {
  if (source) {
    try { source.close(); } catch { /* ignore */ }
  }
  source = null;
  connected = false;
  wasOpen = false;
}

// Test hook: reset module state between tests.
export function _resetFeedForTests() {
  stopFeed();
  subs.clear();
  cursor = 0;
}
