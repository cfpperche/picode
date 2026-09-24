const priority = { blocked: 0, "in-review": 1, "in-progress": 2, ready: 3, paused: 4, completed: 5, cancelled: 6 };

export function activeMissions(rows = []) {
  return rows.filter((m) => !m.archived && m.state !== "completed" && m.state !== "cancelled" && m.state !== "accepted")
    .sort((a, b) => (priority[a.state] ?? 7) - (priority[b.state] ?? 7) || String(b.updatedAt || "").localeCompare(String(a.updatedAt || "")));
}

export function attentionItems({ workspaceId, inbox = [], missions = [], agents = [], statusOf = () => "ready", pr = null }) {
  const items = [];
  const questions = inbox.filter((it) => it.workspaceId === workspaceId && it.state !== "done");
  for (const it of questions) items.push({ key: `inbox:${it.id}`, label: it.title, detail: "Question", href: `#/app/inbox/item/${encodeURIComponent(it.id)}` });
  for (const m of activeMissions(missions)) {
    if (m.state === "blocked" || m.state === "in-review") items.push({ key: `mission:${m.id}`, label: m.title, detail: m.state === "blocked" ? "Blocked mission" : "Ready for review", href: `#/mission/${encodeURIComponent(m.id)}` });
  }
  const asking = new Set(questions.map((it) => it.sourceId));
  for (const a of agents) {
    if (statusOf(a) === "needs-you" && !asking.has(a.id)) items.push({ key: `agent:${a.id}`, label: a.name || "Agent", detail: "Waiting for you", agentId: a.id });
  }
  if (pr?.checks?.failed > 0) items.push({ key: "checks", label: `${pr.checks.failed} failed ${pr.checks.failed === 1 ? "check" : "checks"}`, detail: "Pull request", href: pr.url });
  if (pr?.reviewDecision === "CHANGES_REQUESTED") items.push({ key: "review", label: "Changes requested", detail: "Pull request", href: pr.url });
  return items;
}

const missionActions = {
  create: "created", block: "blocked", "request-review": "requested review", accept: "accepted", pause: "paused", resume: "resumed", cancel: "cancelled", reopen: "reopened", "request-changes": "requested changes",
};

export function activityLabel(item) {
  if (item.kind === "commit") return "Commit";
  if (item.kind === "agent") return "Agent created";
  if (item.kind === "inbox") return item.action === "resolved" ? "Question resolved" : "Question asked";
  if (item.kind === "mission") return `Mission ${missionActions[item.action] || "updated"}`;
  return "Update";
}

export function recentActivity(events = [], commits = [], since = null, now = Date.now()) {
  const floor = Math.max(now - 7 * 86400_000, since || 0);
  const rows = [...events.map((e) => ({ ...e, at: Date.parse(e.createdAt) })),
    ...commits.map((c) => ({ kind: "commit", id: c.hash, entityId: c.hash, title: c.subject, at: c.at * 1000 }))];
  return rows.filter((e) => Number.isFinite(e.at) && e.at > floor)
    .sort((a, b) => b.at - a.at || String(b.id).localeCompare(String(a.id))).slice(0, 6);
}
