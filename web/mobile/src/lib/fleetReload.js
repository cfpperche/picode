// Polls share an active read. A refresh after a mutation must start after
// that read, and its caller must await the new result. Concurrent forced
// refreshes share that follow-up; a later mutation can request another.
export function createFleetReload(read) {
  let flight = null;
  let queued = null;
  function reload({ force = false } = {}) {
    if (flight) {
      if (!force) return flight;
      if (!queued) {
        queued = flight.catch(() => {}).then(() => {
          queued = null;
          return reload();
        });
      }
      return queued;
    }
    flight = Promise.resolve().then(read).finally(() => { flight = null; });
    return flight;
  }
  return { reload, get pending() { return flight !== null; } };
}
