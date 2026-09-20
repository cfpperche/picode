// The computer tool's half of the shell line (ADR-0148). The daemon pushes a
// kind:"computer" frame down the same stream the work-browser commands ride;
// this module runs it through the Tauri bridge (computer_call) and hands the
// result back to the channel that posts it. The grant was checked by the
// daemon; the shell checks its mirrored copy again. Nothing here decides.

export function createComputerRunner({ invoke }) {
  return async (cmd) => {
    if (!invoke) return { error: "not_connected: the desktop app is not driving the computer" };
    const principal = typeof cmd.principal === "string" ? cmd.principal.trim() : "";
    if (!principal) return { error: "disabled: the command names no principal" };
    try {
      const output = await invoke("computer_call", {
        principal,
        action: String(cmd.method || ""),
        paramsJson: JSON.stringify(cmd.params ?? {}),
      });
      return { output };
    } catch (e) {
      return { error: String(e?.message || e) };
    }
  };
}

// enabledKeys is what the page mirrors into the shell (computer_set_grants):
// the grant keys of every principal whose switch is on, in the daemon's own
// namespace (an agent id, or term:<terminal id>).
export function enabledKeys(policies) {
  if (!Array.isArray(policies)) return [];
  const keys = [];
  for (const p of policies) {
    if (!p || !p.enabled) continue;
    const key = p.termId ? "term:" + p.termId : p.agentId;
    if (typeof key === "string" && key.trim() && !keys.includes(key)) keys.push(key);
  }
  return keys;
}
