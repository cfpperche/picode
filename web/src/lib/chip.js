export function shortModel(id) {
  if (!id) return "";
  if (id.length <= 22) return id;
  return id.replace(/^claude-/, "").replace(/^openai\//, "");
}

export function providerChipLabel(cfg) {
  return (cfg && cfg.provider) || "Provider";
}

export function modelChipLabel(cfg) {
  const model = (cfg && cfg.model) || "";
  const thinking = (cfg && cfg.thinking) || "";
  if (!model) return "Model";
  return thinking ? shortModel(model) + " · " + thinking : shortModel(model);
}

export function providerChoices(catalog, currentId) {
  const all = (catalog && catalog.providers) || [];
  const signed = all.filter((p) => p.signedIn);
  const ids = new Set(signed.map((p) => p.id));
  const opts = [{ id: "", label: "Default" }];
  for (const p of signed) opts.push({ id: p.id, label: p.id });
  if (currentId && !ids.has(currentId)) {
    const p = all.find((x) => x.id === currentId);
    if (p) opts.push({ id: p.id, label: p.id, hint: "not signed in" });
  }
  return opts;
}

export function modelChoices(catalog, providerId) {
  if (!providerId) return [];
  const p = ((catalog && catalog.providers) || []).find((x) => x.id === providerId);
  return ((p && p.models) || []).map((m) => ({ id: m.id, label: m.id, thinking: !!m.thinking }));
}

export function filterChoices(options, query) {
  const q = (query || "").trim().toLowerCase();
  if (!q) return options || [];
  return (options || []).filter((o) => (o.label + " " + (o.hint || "")).toLowerCase().includes(q));
}
