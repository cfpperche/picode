import { groupTurns } from "./turns.js";

function clip(text) {
  const t = String(text || "").replace(/\s+/g, " ").trim();
  if (!t) return "";
  return t.length > 140 ? t.slice(0, 137) + "…" : t;
}

export function railAnchors(items) {
  const out = [];
  let i = 0;
  groupTurns(items).forEach((t) => {
    if (t.kind !== "turn") return;
    const id = "turn-" + (i++);
    const userText = clip(t.user && t.user.text);
    const agentText = clip(t.replies[0] && t.replies[0].text);
    if (userText) out.push({ id: id + "-user", actor: "You", cls: "user", preview: userText });
    if (agentText) out.push({ id: id + "-agent", actor: "Agent", cls: "", preview: agentText });
    else if (t.work.length && !userText) out.push({ id: id + "-agent", actor: "Agent", cls: "", preview: "Work" });
  });
  return out;
}
