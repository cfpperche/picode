import { groupTurns } from "./turns.js";

export function railAnchors(items) {
  const out = [];
  let i = 0;
  groupTurns(items).forEach((t) => {
    if (t.kind !== "turn") return;
    const id = "turn-" + (i++);
    const text = String((t.user && t.user.text) || (t.replies[0] && t.replies[0].text) || "").replace(/\s+/g, " ").trim();
    if (!text && t.work.length === 0) return;
    const preview = text.length > 140 ? text.slice(0, 137) + "…" : (text || "Work");
    out.push({
      id,
      actor: t.user ? "You" : "Agent",
      cls: t.user ? "user" : "",
      preview,
    });
  });
  return out;
}
