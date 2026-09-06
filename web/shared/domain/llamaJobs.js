// Feed events and HTTP snapshots can arrive out of order.
export function mergeLlamaJob(current, incoming) {
  const old = current.find(j => j.id === incoming.id);
  if (old && (old.revision || 0) > (incoming.revision || 0)) return current;
  return [incoming, ...current.filter(j => j.id !== incoming.id)].sort((a, b) => b.createdAt.localeCompare(a.createdAt));
}

export function mergeLlamaSnapshot(current, snapshot, startedAt) {
  let result = snapshot;
  for (const job of current) {
    const server = snapshot.find(j => j.id === job.id);
    if (server && (job.revision || 0) > (server.revision || 0) || !server && Date.parse(job.createdAt) >= startedAt) result = mergeLlamaJob(result, job);
  }
  return result;
}
