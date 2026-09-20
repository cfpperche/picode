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
    enabledModels: has.enabledModels ? (l.enabledModels || []) : (p.enabledModels || []),
    defaultTools: has.defaultTools ? (l.defaultTools || []) : (p.defaultTools || PI_TOOLS),
    // The keys pi keeps in its machine file. They land here for the same
    // reason as the rest: a resolver that names a fixed list drops whatever
    // it was not told about, and the Theme row rendered empty while the file
    // said `dark` (2026-09-20).
    theme: has.theme ? l.theme : (p.theme || ""),
    hideThinkingBlock: has.hideThinkingBlock ? !!l.hideThinkingBlock : !!p.hideThinkingBlock,
    quietStartup: has.quietStartup ? !!l.quietStartup : !!p.quietStartup,
    defaultProjectTrust: has.defaultProjectTrust ? l.defaultProjectTrust : (p.defaultProjectTrust || "ask"),
    shellPath: has.shellPath ? l.shellPath : (p.shellPath || ""),
  };
}

export const PI_TOOLS = ["read", "bash", "powershell", "edit", "write", "grep", "find", "ls"];

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
    enabledModels: [],
    defaultTools: PI_TOOLS,
    theme: "",
    hideThinkingBlock: false,
    quietStartup: false,
    // pi resolves anything that is not "always" or "never" to "ask".
    defaultProjectTrust: "ask",
    shellPath: "",
  };
}
