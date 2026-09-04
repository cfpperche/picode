// Event bursts must not supersede every response while a resource sample is
// loading. Serialize event-driven reads and retain one trailing refresh.
export function createRefreshQueue(load) {
  let active = null;
  let pending = false;
  let stopped = false;
  return {
    request() {
      if (stopped) return Promise.resolve();
      pending = true;
      if (!active) active = (async () => {
        while (pending && !stopped) {
          pending = false;
          await load();
        }
      })().finally(() => { active = null; });
      return active;
    },
    stop() { stopped = true; pending = false; },
  };
}
