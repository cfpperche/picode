// ADR-0078: only the existing terminal doors execute Git. The server checks
// live cwd, foreground jobs and repository activity; viewer state is a hint.
export async function deliverGitCommand({ owner, root, command, run = false, terminals = [], workspaceId = "", api, wait = ms => new Promise(resolve => setTimeout(resolve, ms)) }) {
  if (!owner?.id || !root || !command) throw new Error("Open a project folder before preparing a command.");
  const post = (id, resource, body) => api("/api/terminals/" + encodeURIComponent(id) + "/" + resource, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  const targetRefused = error => error?.status === 409 && (["cli", "closed"].includes(error.body?.reason) || /^(This terminal (moved to|is running)|Open the terminal first|Start the terminal first)/i.test(error?.message || ""));
  async function deliver(id) {
    let reason = "";
    if (run) {
      try { await post(id, "run", { text: command, root }); return { id, ran: true, reason }; }
      catch (error) {
        if (targetRefused(error)) throw error;
        // Only an explicit conflict can fall back to preparation. Network
        // failure has an unknown outcome and must not duplicate a command.
        if (error?.status !== 409) throw error;
        reason = error.message;
      }
    }
    await post(id, "type", { text: command, root });
    return { id, ran: false, reason };
  }
  async function create() {
    const page = await api("/api/terminals", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: command.trim().split(/\s+/)[0] || "git", cwd: root, workspaceId }) });
    await wait(900);
    return page.id;
  }
  const candidate = owner.kind === "term" ? owner.id : terminals.find(term => term?.cwd === root && term.running !== false && !term.tui && !term.cli && !term.launchCli && !term.state)?.id;
  if (!candidate) return deliver(await create());
  try { return await deliver(candidate); }
  catch (error) {
    if (!targetRefused(error)) throw error;
    return deliver(await create());
  }
}
