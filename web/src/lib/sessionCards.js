export function infoLine(n) {
  if (n.role === "assistant") return { kind: "reply", text: n.text || "reply" };
  if (n.role === "toolResult") {
    const t = (n.text || "").trim();
    return { kind: "tool", text: !t || t.startsWith("<") ? "tool" : t };
  }
  const k = (n.kind || "").replace(/_/g, " ");
  return { kind: "meta", text: k || "note" };
}

export function leafUserId(nodes, leafId) {
  if (!leafId) return "";
  const byId = new Map();
  function walk(list) {
    for (const n of list || []) {
      byId.set(n.id, n);
      walk(n.children);
    }
  }
  walk(nodes);
  let n = byId.get(leafId);
  while (n) {
    if (n.role === "user") return n.id;
    n = n.parentId ? byId.get(n.parentId) : null;
  }
  return "";
}

export function cardsFrom(nodes) {
  const out = [];
  for (const n of nodes || []) {
    if (n.role === "user") {
      const info = [];
      const nested = [];
      for (const c of n.children || []) {
        if (c.role === "user") nested.push(c);
        else {
          info.push(infoLine(c));
          collect(c, info, nested);
        }
      }
      out.push({ id: n.id, text: n.text || "Message", info, children: cardsFrom(nested) });
    } else {
      out.push(...cardsFrom(n.children));
    }
  }
  return out;
}

function collect(n, info, nested) {
  for (const c of n.children || []) {
    if (c.role === "user") nested.push(c);
    else {
      info.push(infoLine(c));
      collect(c, info, nested);
    }
  }
}
