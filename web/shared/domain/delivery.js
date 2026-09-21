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
  if (c.integration === "integrated") {
    // D2 (ADR-0170): an integrated change now has a publication fact, so the
    // old "publication is not shown here" would contradict the row it explains.
    if (c.publication === "published") return "Included in the target branch and in the running version.";
    if (c.publication === "not-published") return "Included in the target branch, not in the running version.";
    return "Included in the target branch. What is running is unconfirmed.";
  }
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
  return age > 30 ? "Last check is out of date" : "Checked " + seconds(age);
}

// D2 publication observation (ADR-0170, docs/plans/delivery-publication.md).
// The server derives every state; the words below are the surface contract's
// own, so desktop and phone cannot drift. Nothing here says safe, ready,
// approved or deployed-by-PiCode: the lane reports what an artifact contains
// and what a producer recorded, and names what it could not establish.

export const publicationLabel = { published: "Published", "not-published": "Not published", unknown: "Publication unconfirmed" };

// publicationReason explains one change's publication state in the detail.
export function publicationReason(c, env) {
  if (!env || env.status === "unconfigured") return "No environment is connected to this project.";
  if (c.publication === "published") return "The running version contains this revision.";
  if (c.publication === "not-published") return "Integrated in the target, but the running version does not contain this revision yet.";
  if (env.reasonCode === "artifact-dirty") return "The running version was built from unfinished changes; its contents are unconfirmed.";
  if (env.reasonCode === "revision-unmapped" || env.reasonCode === "revision-unknown") return "The running version does not map to a revision of this repository.";
  if (env.reasonCode === "divergent-history") return "The running version is not on this branch's line, so inclusion cannot be confirmed.";
  return "This change's presence in the running version could not be confirmed.";
}

// environmentLane is the Deployment lane's whole model: one headline, its
// action, the environment's facts, and the newest recorded attempt.
export function environmentLane(env, targetOid = "", now = Date.now()) {
  if (!env || env.status === "unconfigured") {
    return { status: "unconfigured", headline: "Deployment is not connected for this project.", action: { kind: "setup", label: "View setup" }, facts: [], unpublished: null, attempt: null, busy: null };
  }
  // The instance's identity is a fact about this machine, so every configured
  // project carries it — including the states where the connection is unusable.
  const version = env.displayVersion ? "Running " + env.displayVersion : "Running version not recorded";
  const health = env.responding === true ? "Responding" : env.responding === false ? "Not responding" : "Health unknown";
  const age = Math.max(0, Math.floor((now - Date.parse(env.observedAt || 0)) / 1000));
  const checked = env.observedAt ? (age > 30 ? "check is out of date" : "checked " + seconds(age)) : "check time unknown";
  if (env.status === "conflict") {
    return { status: "conflict", headline: "Two projects claim this environment; disconnect one.", action: { kind: "setup", label: "View setup" }, facts: [`${version} · ${health} · ${checked}`], unpublished: null, attempt: null, busy: null };
  }
  const out = {
    status: env.status,
    headline: "",
    action: null,
    facts: [`${version} · ${health} · ${checked}`],
    unpublished: null,
    attempt: attemptLine(env.lastAttempt),
    busy: env.busy && env.busy.status === "known" && env.busy.owners > 0
      ? { text: "Agents are still working.", action: { kind: "activity", label: "View activity" } }
      : null,
  };
  if (env.reasonCode === "target-required" || !targetOid) {
    out.headline = "Choose the branch changes will join.";
  } else if (env.status === "unknown") {
    if (env.reasonCode === "repository-mismatch") {
      out.headline = "The connected environment points at another repository.";
      out.action = { kind: "setup", label: "View setup" };
    } else {
      out.headline = "Running version found; included changes are unconfirmed.";
    }
  } else {
    out.headline = "";
  }
  if (env.status === "known" && env.unpublished) {
    const n = env.unpublished.count;
    out.unpublished = { count: n, text: "Integrated, not published: " + n + (n === 1 ? " change" : " changes") };
  }
  if (env.issues && env.issues.length) out.facts = out.facts.concat(env.issues);
  return out;
}

// attemptLine keeps the design's distinction: an attempt that never finished
// is unknown, never running and never a failure (O21).
export function attemptLine(a) {
  if (!a) return null;
  const evidence = { kind: "evidence", label: "View evidence" };
  // `outcome` rides along: the surface tints a failure from it, and dropping it
  // here silently removed the attribute (visual review, 2026-09-21).
  if (a.outcome === "unknown") return { outcome: "unknown", text: "Last attempt: Unknown outcome", detail: "The deploy started and never recorded a result.", action: evidence, error: a.error || "" };
  if (a.outcome === "failed") return { outcome: "failed", text: "Last attempt: Failed" + (a.revisionBefore ? " · previous version is still responding" : ""), detail: "", action: evidence, error: a.error || "" };
  return { outcome: "passed", text: "Last attempt: Passed" + (a.builtRevision ? " · " + short(a.builtRevision) : ""), detail: "", action: evidence, error: "" };
}

// seconds keeps the one-second case from reading "1 seconds ago".
function seconds(age) { return age + (age === 1 ? " second ago" : " seconds ago"); }

function short(rev) { return typeof rev === "string" && rev.length > 7 ? rev.slice(0, 7) : rev || ""; }
