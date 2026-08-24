export function shortModel(id) {
  if (!id) return "";
  if (id.length <= 22) return id;
  return id.replace(/^claude-/, "").replace(/^openai\//, "");
}

export function chipLabel(cfg) {
  const model = (cfg && cfg.model) || "";
  const thinking = (cfg && cfg.thinking) || "";
  const provider = (cfg && cfg.provider) || "";
  if (model) return thinking ? shortModel(model) + " · " + thinking : shortModel(model);
  if (provider) return provider;
  return "Default model";
}
