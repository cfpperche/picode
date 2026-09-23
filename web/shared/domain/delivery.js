import { MISSION_STATES, missionHash } from "./missions.js";
export const integrationLabel = { integrated: "Integrated", "not-integrated": "Not integrated", "update-needed": "Update needed", unknown: "Integration unconfirmed" };
export const validationLabel = { passed: "Checks passed", "scoped-passed": "Relevant checks passed", "scope-reusable": "Prior checks may be reused", "needs-recheck": "Needs recheck", failed: "Checks failed", unknown: "Checks not recorded" };
export function deliveryReason(c) {
  if (c.sourceStatus === "changed") return "The branch changed after this delivery was recorded.";
  if (c.checkout === "dirty" && c.integration !== "integrated") return "There are unfinished changes in this checkout.";
  if (c.validation === "failed") return c.integration === "integrated" ? "The project checks failed after integration." : "The recorded checks failed.";
  if (c.integration === "update-needed") return "This change and the target have moved apart.";
  if (c.validation === "needs-recheck") return "These checks do not cover the current change and target.";
  if (c.validation === "scope-reusable") return "Prior scoped evidence is not confirmation of the current project's checks.";
  if (c.review === "other-target") return "The review request names another target branch.";
  if (c.review === "requested") return "An agent requested review; human approval is not recorded.";
  if (c.sourceStatus === "removed") return "The branch is gone; this record retains its revision.";
  if (c.integration === "integrated") return "Included in the target branch. Publication is not shown here.";
  return "No review request was recorded.";
}
export function integrationExplanation(state) {
  if (state === "integrated") return "Included in the selected target branch.";
  if (state === "not-integrated") return "The selected target does not include this change yet.";
  if (state === "update-needed") return "The change and target moved independently; inspect before integrating.";
  return "PiCode could not confirm whether the target includes this change.";
}
export function validationExplanation(state) {
  if (state === "passed") return "A clean full-project check covers the selected revision.";
  if (state === "scoped-passed") return "Relevant checks passed; this is not a full-project result.";
  if (state === "scope-reusable") return "Earlier relevant checks may apply, but current checks are not confirmed.";
  if (state === "needs-recheck") return "The recorded checks no longer cover the current change or target.";
  if (state === "failed") return "A recorded check failed for this revision.";
  return "No check evidence was found for this revision.";
}
export function attention(c) { return c.integration !== "integrated" || c.validation === "failed" || c.validation === "unknown"; }
// missionsFor: the objectives a change serves, from the read's `missions` map
// (delivery id → links). The link is one-way — the mission cites the delivery as
// evidence (ADR-0199) — so a change nobody cites says nothing rather than "none".
export function missionsFor(data, id) {
  const links = data && data.missions ? data.missions[id] : null;
  return Array.isArray(links) ? links : [];
}

// missionLinks: what a Delivery row or detail shows for those objectives — the
// title, the mission's own state label, and the route back to the mission.
export function missionLinks(links = []) {
  return (links || []).map((l) => ({
    id: l.id,
    title: l.title || l.id,
    label: MISSION_STATES[l.state] || l.state || "",
    href: missionHash(l.id),
  }));
}

export function deliveryRows(rows = [], filter = "all") {
  const rank = c => c.validation === "failed" ? 0 : c.integration === "update-needed" || c.checkout === "dirty" ? 1 : c.integration === "integrated" ? 3 : 2;
  return rows.filter(c => filter !== "attention" || attention(c)).slice().sort((a,b) => rank(a)-rank(b) || a.title.localeCompare(b.title) || a.id.localeCompare(b.id));
}
export function observationLabel(state, now = Date.now()) {
  if (state.busy) return state.data ? "Updating…" : "Checking deliveries…";
  if (state.error) return state.data ? "Could not update; showing the last check." : "Could not read delivery status.";
  if (state.offline) return "Offline; showing the last check.";
  if (!state.data) return "Not checked";
  const age = Math.max(0, Math.floor((now-Date.parse(state.data.observedAt))/1000));
  return age > 30 ? "Last check is out of date" : "Checked " + age + " seconds ago";
}
