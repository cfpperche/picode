// Browser-only QA fixture. Load with --init-script against the disposable docs
// fixture on :18785. Does NOT certify a real Pi/browser-package emitter.
(() => {
  if (location.origin !== "http://127.0.0.1:18785") return;
  const image = globalThis.__captureFixtureImage;
  if (!image) throw new Error("Supply __captureFixtureImage before this fixture");
  const realFetch = window.fetch.bind(window);
  const NativeWebSocket = window.WebSocket;
  let active = null;
  let scenario = new URL(location.href).searchParams.get("capture") || "valid";
  const startedAt = Date.now();
  let done = false;
  const requested = [];
  const pngHeaderOnly = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwC";
  const details = () => scenario === "none" ? {} : { preview: scenario === "blocked"
    ? { image: "https://example.invalid/must-not-load.png" }
    : { image: scenario === "broken" ? pngHeaderOnly : image,
      title: "PiCode fleet", url: "https://user:secret@example.com/fleet?token=secret#private",
      ts: Date.now(), source: "QA browser / tab 1" } };
  const tool = () => ({ kind: "tool", id: "capture-qa", name: "capture_fixture", args: "Observe the fleet", toolArgs: {},
    ts: startedAt, status: done ? "ok" : "···", ...(done ? { result: details() } : {}) });
  const send = (event) => active?.dispatchEvent(new MessageEvent("message", { data: JSON.stringify({ agentId: active.agent, event }) }));
  window.fetch = async (input, options) => {
    const url = new URL(typeof input === "string" ? input : input.url, location.href);
    if (url.pathname.endsWith("/sessions/transcript")) {
      const agent = url.searchParams.get("agent");
      const payload = { path: "/tmp/" + agent + ".jsonl", events: [
        { kind: "user", text: "Show me the latest capture", ts: startedAt }, tool(),
      ], remaining: 0, total: 2 };
      requested.push({ agent, at: Date.now() });
      // A successful but deliberately stale read; never abort this route.
      await new Promise((r) => setTimeout(r, 1600));
      return Response.json(payload);
    }
    if (url.pathname.endsWith("/sessions") && url.pathname.includes("/workspaces/")) {
      const agent = url.searchParams.get("agent");
      const path = "/tmp/" + agent + ".jsonl";
      return Response.json({ current: path, sessions: [{ path, name: "Capture QA", timestamp: 1700000000000 }] });
    }
    const response = await realFetch(input, options);
    if (url.pathname === "/api/workspaces") {
      const fleet = await response.json();
      for (const ws of fleet) for (const a of ws.agents || []) if (a.name === "Atlas" || a.name === "Borealis") {
        a.mode = "managed"; a.running = true; a.streaming = !done;
      }
      return Response.json(fleet);
    }
    return response;
  };
  class CaptureSocket extends EventTarget {
    static CONNECTING = 0; static OPEN = 1; static CLOSING = 2; static CLOSED = 3;
    constructor(url, protocols) {
      super();
      if (!String(url).includes("/ws/agent")) return new NativeWebSocket(url, protocols);
      this.agent = new URL(url).searchParams.get("agent");
      this.readyState = 1;
      active = this;
      done = false;
      this.addEventListener("message", (e) => this.onmessage?.(e));
      this.addEventListener("close", (e) => this.onclose?.(e));
      this.timers = [
        setTimeout(() => { if (active === this) send({ type: "snapshot", streaming: true, waiting: false }); }, 30),
        setTimeout(() => { if (active === this) send({ type: "agent_start" }); }, 100),
        setTimeout(() => { if (active === this) send({ type: "tool_execution_start", toolCallId: "capture-qa", toolName: "capture_fixture", args: {} }); }, 200),
        setTimeout(() => { if (active === this) send({ type: "tool_execution_update", toolCallId: "capture-qa", partialResult: { details: details() } }); }, 400),
      ];
    }
    send() {}
    close() { this.timers.forEach(clearTimeout); this.readyState = 3; this.dispatchEvent(new Event("close")); }
  }
  window.WebSocket = CaptureSocket;
  window.captureQA = {
    requested,
    mode(value) { scenario = value; send({ type: "tool_execution_update", toolCallId: "capture-qa", partialResult: { details: details() } }); },
    finish() { done = true; send({ type: "tool_execution_end", toolCallId: "capture-qa", toolName: "capture_fixture", result: { content: [{ type: "text", text: "Capture complete" }], details: details() } }); send({ type: "agent_settled" }); },
    disconnect() { active?.close(); },
    late() { send({ type: "tool_execution_update", toolCallId: "capture-qa", partialResult: { details: { preview: { image: "https://example.invalid/late" } } } }); },
  };
})();
