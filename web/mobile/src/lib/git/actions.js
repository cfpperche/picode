// Mobile-owned adaptation of ADR-0078's established terminal and agent channels.
import { agentsOf, displayAgentName } from "@picode/shared/domain/tree.js";

export function shellQuote(text) {
  return "'" + String(text).replace(/'/g, "'\\''") + "'";
}

// A ref that needs quoting is one with a character git allows but a shell
// might read (git already forbids spaces, ~ ^ : ? * [ and backslash).
function refArg(name) {
  const s = String(name || "");
  return /^[A-Za-z0-9._\/@+-]+$/.test(s) ? s : shellQuote(s);
}

// gitActionCommand: the exact command the rail types into the terminal for
// the human to submit. Push publishes a branch that has no upstream yet.
export function gitActionCommand(action, { branch = "", upstream = "", message = "" } = {}) {
  const push = upstream ? "git push" : branch ? `git push -u origin ${refArg(branch)}` : "git push";
  switch (action) {
    case "fetch": return "git fetch --prune";
    case "pull": return "git pull --ff-only";
    case "push": return push;
    case "commit": return `git add -A && git commit -m ${shellQuote(message)}`;
    case "commit-push": return `git add -A && git commit -m ${shellQuote(message)} && ${push}`;
    case "pr": return "gh pr create --fill";
    default: return "";
  }
}

// gitActions: which actions a repository offers. A detached HEAD has nothing
// to push or pull; a folder without a branch at all offers none.
export function gitActions(status) {
  if (!status || !status.git || !status.branch) return [];
  const out = ["fetch"];
  if (!status.detached) out.push("pull", "push");
  out.push("commit");
  if (!status.detached) out.push("commit-push");
  return out;
}

// branchChip: what the tabs row says about the checkout — name, arrows for
// ahead/behind, and one word for the two states that need it.
export function branchChip(status) {
  if (!status || !status.git || !status.branch) return null;
  const ahead = Number(status.ahead) || 0;
  const behind = Number(status.behind) || 0;
  const detached = !!status.detached;
  const unpublished = !detached && !status.upstream;
  const parts = [];
  if (ahead) parts.push(`${ahead} ahead`);
  if (behind) parts.push(`${behind} behind`);
  const title = detached ? `Detached at ${status.branch}`
    : unpublished ? `Branch ${status.branch} has no upstream yet`
      : `Branch ${status.branch}` + (parts.length ? `, ${parts.join(" and ")} ${status.upstream}` : ` is level with ${status.upstream}`);
  return { name: status.branch, ahead, behind, detached, unpublished, upstream: status.upstream || "", worktree: status.worktree || "", title };
}

// askableAgents: the agents the Git menu can ask instead of the terminal
// (ADR-0078 stage 3) — running, managed or in a TUI, and working inside the
// rail's repository. The anchored agent comes first, then by name; at most
// three, so the menu stays a menu. Stopped agents are not offered: the
// terminal is their door.
export function askableAgents(ctx, { root = "", repoRoot = "", anchor = null } = {}) {
  const top = normDir(repoRoot || root);
  if (!top) return [];
  const out = [];
  const consider = (agent, ws) => {
    if (!agent || !agent.id || !agent.running) return;
    if (agent.mode !== "managed" && agent.mode !== "interactive") return;
    const path = normDir(agent.workPath || (ws && ws.path) || "");
    if (!path || (path !== top && !path.startsWith(top + "/"))) return;
    out.push({ id: agent.id, name: displayAgentName(agent, ws), mode: agent.mode, streaming: !!agent.streaming, path });
  };
  for (const ws of (ctx && ctx.workspaces) || []) for (const a of agentsOf(ws)) consider(a, ws);
  for (const a of (ctx && ctx.freeAgents) || []) consider(a, null);
  const anchored = anchor && anchor.kind === "agent" ? anchor.id : "";
  out.sort((a, b) => (a.id === anchored ? -1 : b.id === anchored ? 1 : a.name.localeCompare(b.name)));
  return out.slice(0, 3);
}

function normDir(p) {
  return String(p || "").replace(/\\/g, "/").replace(/\/+$/, "");
}

// askChannelHint: the short note beside an "Ask <name>" entry — where the
// prompt lands and when the agent will act on it.
export function askChannelHint(who) {
  if (!who) return "";
  if (who.mode === "interactive") return "in its terminal";
  return who.streaming ? "after its turn" : "";
}

// askGitPrompt: what the agent is asked, in the plain words the human reads
// in the form's preview — the folder, the branch, the action and its rules.
// Where a command would be blind the prompt asks for judgment instead.
export function askGitPrompt(action, { root = "", branch = "", upstream = "", message = "" } = {}) {
  const where = root ? `In ${root}` : "In this folder";
  const br = branch ? `the branch ${branch}` : "the current branch";
  const push = upstream
    ? `push ${br} to its upstream (${upstream})`
    : `push ${br} and set its upstream (origin/${branch || "<branch>"})`;
  const msg = String(message || "").trim();
  const commit = msg
    ? `commit the current changes on ${br} with this message: "${msg}".`
    : `review the uncommitted changes on ${br} and commit them with a fitting message.`;
  switch (action) {
    case "fetch": return `${where}, run git fetch --prune and tell me how ${br} stands against its upstream.`;
    case "pull": return `${where}, bring ${br} up to date with its upstream by fast-forward only (git pull --ff-only). If it cannot fast-forward, stop and tell me why instead of merging or rebasing.`;
    case "push": return `${where}, ${push}. Never force-push; if the push is rejected, tell me why.`;
    case "commit": return `${where}, ${commit} Do not push.`;
    case "commit-push": return `${where}, ${commit} Then ${push}, never with force.`;
    case "pr": return `${where}, create a pull request for ${br} with gh pr create, writing the title and body from the commits, and give me its URL.`;
    default: return "";
  }
}

const ASK_VERBS = { fetch: "fetch", pull: "pull", push: "push", commit: "commit", "commit-push": "commit and push", pr: "open a pull request" };

// askedNote: the toast after an ask, honest about when the agent acts. The
// server says which door the prompt took: a managed agent's queue (now, or
// after the turn it is in) or a TUI's terminal.
export function askedNote(name, action, result) {
  const verb = ASK_VERBS[action] || "do it";
  const who = name || "the agent";
  const r = result || {};
  if (r.mode === "interactive") return `Asked ${who} to ${verb} in its terminal.`;
  if (r.busy) return `Queued for ${who}: ${verb} after its current turn.`;
  return `Asked ${who} to ${verb}.`;
}
