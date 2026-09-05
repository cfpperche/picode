// Match desktop configuration semantics: only a changed tool mode restarts
// a running agent, preserving its managed/interactive runtime.
export async function saveAgentConfig(agent, patch, request, closeTerminal = () => {}) {
  if (!agent?.id) throw new Error("Select an agent first.");
  const base = "/api/agents/" + encodeURIComponent(agent.id);
  await request(base, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) });
  const changed = Object.hasOwn(patch, "opMode") && (patch.opMode || "full") !== (agent.opMode || "full");
  if (!changed || !["managed", "interactive"].includes(agent.mode)) return;
  const interactive = agent.mode === "interactive";
  try {
    await request(base + (interactive ? "/close" : "/managed/stop"), { method: "POST" });
    if (interactive) closeTerminal(agent.id);
    await request(base + (interactive ? "/open" : "/managed/start"), { method: "POST" });
  } catch (error) {
    throw new Error("Settings saved, but the agent could not restart. Check its state and use Start. " + error.message);
  }
}
