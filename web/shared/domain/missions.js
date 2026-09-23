export const MISSION_STATES = { ready: "Ready", "in-progress": "In progress", blocked: "Needs you", "in-review": "Ready for review", paused: "Paused", completed: "Completed", cancelled: "Cancelled" };
export function missionLocation(hash = "") {
  const [path, query = ""] = String(hash).replace(/^#/, "").split("?");
  const match = /^\/mission\/([^/]+)$/.exec(path);
  let id = "";
  try { id = match ? decodeURIComponent(match[1]) : ""; } catch { return null; }
  return match || path === "/missions" || path === "/missions/new" ? { id, create: path === "/missions/new", workspace: new URLSearchParams(query).get("workspace") || "" } : null;
}
export function missionHash(id = "") { return id ? "#/mission/" + encodeURIComponent(id) : "#/missions"; }
export function missionDraft(mission, workspace = "") {
  return { workspaceId: mission?.workspaceId || workspace, title: mission?.title || "", objective: mission?.objective || "", context: mission?.context || "", nextAction: mission?.nextAction || "", criteriaText: (mission?.criteria || []).map(c => c.text).join("\n") };
}
export function missionDraftPayload(draft, base) {
  const { criteriaText, ...rest } = draft;
  return { ...rest, criteria: criteriaText.split("\n").map(x => x.trim()).filter(Boolean).map((text, i) => ({ id: base?.criteria?.[i]?.id || "", text })) };
}
export function missionActions(v) {
  if (!v) return [];
  const terminal = ["completed", "cancelled"].includes(v.state);
  const reserved = !!v.assignment?.reserved;
  if (terminal) return reserved ? ["release"] : ["reopen", "archive"];
  const actions = ["report", "evidence"];
  if (v.state === "paused") actions.push("resume"); else actions.push("pause");
  if (v.state === "blocked") actions.push("decision");
  if (v.state === "in-review") actions.push("accept", "changes");
  if (["in-progress", "blocked"].includes(v.state)) actions.push("request-review");
  if (v.state === "ready" && reserved) {
    if (v.assignment.delivery === "prepared") actions.push("dispatch");
    actions.push("acknowledge");
  }
  actions.push("assign", "cancel");
  if (reserved) actions.push("release"); else actions.push("edit", "relink");
  const primary = v.state === "blocked" ? "decision" : v.state === "in-review" ? "accept" : v.state === "paused" ? "resume" : !reserved ? "assign" : v.state === "ready" ? (v.assignment.delivery === "prepared" ? "dispatch" : "acknowledge") : "report";
  return [primary, ...actions.filter(x => x !== primary)];
}
export function latestMissionEvidence(v, criterionId, revision) {
  return (v?.evidence || []).filter(e => e.criterionId === criterionId && e.scopeVersion === v.scopeVersion && (!v.repository || e.revision === revision)).at(-1) || null;
}
export function missionDraftKey(id) { return "picode-mission-draft:" + (id || "new"); }

// Version belongs to the saved request, not to a reconstructed form. A lost
// response followed by a reload must replay the original operation verbatim.
export function missionRequestReceipt(id, action, fields, version, prior, newID) {
  const { expectedVersion: _expectedVersion, ...intent } = fields;
  const fingerprint = JSON.stringify({ id, action, fields: intent });
  if (prior?.fingerprint === fingerprint) return prior;
  return { fingerprint, requestId: newID(), payload: { action, expectedVersion: version, ...fields } };
}
