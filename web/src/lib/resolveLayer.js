export function resolveLayer(layer, parent) {
  const has = (layer && layer.has) || {};
  const l = layer || {};
  const p = parent || {};
  return {
    compactionEnabled: has.compactionEnabled ? l.compactionEnabled : p.compactionEnabled,
    steeringMode: has.steeringMode ? l.steeringMode : (p.steeringMode || "one-at-a-time"),
    followUpMode: has.followUpMode ? l.followUpMode : (p.followUpMode || "one-at-a-time"),
    defaultProvider: has.defaultProvider ? l.defaultProvider : (p.defaultProvider || ""),
    defaultModel: has.defaultModel ? l.defaultModel : (p.defaultModel || ""),
    defaultThinkingLevel: has.defaultThinkingLevel ? l.defaultThinkingLevel : (p.defaultThinkingLevel || ""),
  };
}

export function catalogBase(catalog) {
  const p = catalog && catalog.providers && catalog.providers[0];
  const m = p && p.models && p.models[0];
  return {
    compactionEnabled: true,
    steeringMode: "one-at-a-time",
    followUpMode: "one-at-a-time",
    defaultProvider: p ? p.id : "",
    defaultModel: m ? m.id : "",
    defaultThinkingLevel: "medium",
  };
}
