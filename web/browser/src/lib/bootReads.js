// The desktop boot's reads, all started at once. The boot used to be a
// queue — system, then the Pi model catalog (`pi --list-models`, 2–4 s),
// then the CLI catalog, and only then the workspaces — so the sidebar
// waited on Pi for every cold load, and Pi is one CLI among nine, never a
// dependency (ADR-0179). Now the fleet is the only thing the boot awaits;
// everything else is applied whenever it lands.
//
// fleet: what tab restoration needs, with today's failure rules —
//   workspaces failing rejects (the boot's catch handles it); free agents
//   and terminals failing read as empty; apps failing read as not-loaded
//   (appsOk: false), so saved app tabs survive a failed /api/apps.
// side: system+version, catalog and CLIs, each its own promise.
export function bootReads(api) {
  const workspaces = api("/api/workspaces");
  const free = api("/api/agents?free=1");
  const terminals = api("/api/terminals");
  const apps = api("/api/apps");
  const settled = Promise.allSettled([workspaces, free, terminals, apps]);
  const side = {
    system: Promise.all([api("/api/system"), api("/api/version")]).then(([system, version]) => ({ system, version })),
    // Asked once the fleet has answered: it holds a connection for seconds,
    // and on plain HTTP/1.1 (six per host) it queued the fleet's own reads
    // behind it after a quick reload. The fleet answers in tens of ms.
    catalog: settled.then(() => api("/api/catalog")),
    clis: api("/api/clis"),
  };
  // Settled, so a rejection that nobody awaits yet is never "unhandled".
  const fleet = settled.then(([w, f, t, a]) => {
    if (w.status === "rejected") throw w.reason;
    return {
      workspaces: w.value,
      freeAgents: f.status === "fulfilled" ? f.value : [],
      terminals: t.status === "fulfilled" ? ((t.value && t.value.terminals) || []) : [],
      apps: a.status === "fulfilled" ? a.value : null,
      appsOk: a.status === "fulfilled",
    };
  });
  return { fleet, side };
}
