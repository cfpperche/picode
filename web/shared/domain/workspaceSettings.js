// The Settings dialog's integration section, pure: where the value in force
// comes from, and what Save has to do. The page is GET
// /api/delivery/integration?workspace=<id> (ADR-0182): `declared` says the
// workspace has a layer of its own, `effective.fromScope` names the layer in
// force — the workspace id, "" for the machine, "default" for the built-in.
//
// The dialog offers one choice — follow the machine, or give this workspace
// its own — so the layers never have to be explained as layers.
//
// | mode    | declared | draft vs stored | Save does                      |
// | ------- | -------- | --------------- | ------------------------------ |
// | machine | no       | —               | nothing                        |
// | machine | yes      | —               | DELETE (inherit again)         |
// | own     | no       | —               | PUT (creates the layer)        |
// | own     | yes      | same            | nothing                        |
// | own     | yes      | changed         | PUT with expectedVersion       |

export function integrationSource(page) {
  if (page && page.declared) return "own";
  // The machine layer's scope is "", which the server's omitempty drops:
  // anything that is not the built-in default is the machine's.
  const from = page && page.effective ? page.effective.fromScope : "default";
  return from === "default" ? "default" : "machine";
}

// inheritedLine: the one sentence that tells the owner what the workspace
// follows when it has no settings of its own.
// The runner (ADR-0182) lands only a fast-forward, and only under rules
// someone declared: with none — the built-in default — or with fast-forward
// off, an authorized branch stays blocked, so the line says that instead of
// describing rules that would never run.
export function inheritedLine(page) {
  const eff = (page && page.effective) || {};
  if (integrationSource(page) === "default") return "No rules yet: an authorized branch will not land until this workspace or this machine has some.";
  // No closing period: a check such as `go test ./...` already ends in dots.
  return "This machine's rules: " + rulesSummary(eff);
}

// blocksLanding: rules the runner refuses to act on (fast-forward off).
export function blocksLanding(rules) {
  if (!rules || !rules.mode) return true; // nothing runs until a mode is declared (ADR-0186)
  if (rules.mode === "provider") return false; // the project's own queue integrates it
  return rules.ffOnly === false;
}

// rulesSummary: the rules in words — "fast-forward only, after make ci".
export function rulesSummary(rules) {
  if (rules && rules.mode === "provider") return "the project's own merge queue integrates it";
  if (!rules || rules.mode !== "local") return "no integration mode declared";
  const checks = cleanChecks(rules && rules.checks);
  if (blocksLanding(rules)) return "fast-forward off, so authorized branches stay blocked";
  return "fast-forward only" + (checks.length ? ", after " + checks.join(" · ") : ", no checks");
}

// integrationIntro: the one sentence that says who does the integrating, true
// for every mode — PiCode's local runner, the project's own queue, or nothing
// declared yet. It sits above the editor, so it follows the draft.
export function integrationIntro(rules) {
  const mode = (rules && rules.mode) || "";
  if (mode === "provider") return "Your project's own merge queue integrates it: PiCode enqueues and observes, and that provider's checks decide.";
  if (mode === "local") return "When you authorize a branch an agent delivered, PiCode runs these checks and then merges it.";
  return "Choose who integrates before anything runs — PiCode here, or your project's own merge queue.";
}

// cleanChecks drops the form's blank rows; the declaration never holds one.
export function cleanChecks(rows) {
  return (rows || []).map((c) => String(c || "").trim()).filter(Boolean);
}

function same(a, b) {
  return (a.mode || "") === (b.mode || "") && !!a.ffOnly === !!b.ffOnly && JSON.stringify(cleanChecks(a.checks)) === JSON.stringify(cleanChecks(b.checks));
}

export function integrationSave(page, mode, draft) {
  const declared = !!(page && page.declared);
  if (mode === "machine") return declared ? { action: "delete" } : { action: "none" };
  const body = { mode: draft.mode || "", ffOnly: !!draft.ffOnly, checks: draft.mode === "provider" ? [] : cleanChecks(draft.checks) };
  if (!declared) return { action: "put", body };
  const stored = page.settings || {};
  if (same(stored, body)) return { action: "none" };
  return { action: "put", body: { ...body, expectedVersion: stored.version || 0 } };
}

// draftFrom seeds the editable fields from what is in force, so switching to
// "own" starts from the current rules instead of a blank form.
export function draftFrom(page) {
  const eff = (page && (page.declared ? page.settings : page.effective)) || {};
  return { mode: eff.mode || "", ffOnly: eff.ffOnly !== false, checks: [...(eff.checks || [])] };
}

// followsRows: the Preferences list of who follows the machine's rules. One
// row per workspace, in sidebar order, from GET /api/delivery/integrations
// (`machine`, and `workspaces` keyed by id for the ones with their own).
//
// | workspace layer | machine layer | row                                        |
// | --------------- | ------------- | ------------------------------------------ |
// | own, ff on      | —             | own · "Own rules · fast-forward only, …"    |
// | own, ff off     | —             | blocked · "Own rules · fast-forward off, …" |
// | none            | ff on         | machine · "These rules"                    |
// | none            | ff off        | blocked · "These rules — blocked"          |
// | none            | none          | blocked · "Blocked — no rules"             |
export function followsRows(workspaces, layers) {
  const machine = (layers && layers.machine) || null;
  const own = (layers && layers.workspaces) || {};
  return (workspaces || []).map((w) => {
    const mine = own[w.id];
    if (mine) return { id: w.id, name: w.name, state: blocksLanding(mine) ? "blocked" : "own", text: "Own rules · " + rulesSummary(mine) };
    if (!machine) return { id: w.id, name: w.name, state: "blocked", text: "Blocked — no rules" };
    if (blocksLanding(machine)) return { id: w.id, name: w.name, state: "blocked", text: "These rules — blocked" };
    return { id: w.id, name: w.name, state: "machine", text: "These rules" };
  });
}

// machinePage shapes the machine layer like a workspace's GET page, so Save
// reuses integrationSave: the machine has no "inherit", only its own rules.
export function machinePage(layers) {
  const m = (layers && layers.machine) || null;
  return m ? { declared: true, settings: m, effective: m } : { declared: false, effective: { fromScope: "default", mode: "", ffOnly: true, checks: [] } };
}
