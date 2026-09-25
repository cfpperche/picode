import assert from "node:assert/strict";
import { test } from "node:test";
import { createBrowserChannel, EVENTS_VERB } from "./browserChannel.js";

// A stand-in for the EventSource: the test decides when a frame arrives.
function fakeSource() {
  const listeners = new Map();
  return {
    addEventListener(name, fn) {
      listeners.set(name, fn);
    },
    removeEventListener(name) {
      listeners.delete(name);
    },
    close() {
      listeners.clear();
    },
    // frame delivers one SSE command frame to the channel.
    frame(data) {
      listeners.get("command")?.({ data });
    },
    get listening() {
      return listeners.has("command");
    },
  };
}

function harness({ tab = "w:7", invoke, post, ensureSession = null, hiddenPixelWait = 4000, revealSettle = 400 } = {}) {
  const source = fakeSource();
  const calls = { invoke: [], post: [], errors: [] };
  const channel = createBrowserChannel({
    activeTabId: () => tab,
    ensureSession,
    invoke:
      invoke ??
      (async (cmd, args) => {
        calls.invoke.push([cmd, args]);
        return { ok: cmd };
      }),
    source,
    post:
      post ??
      (async (result) => {
        calls.post.push(result);
      }),
    onError: (message, err) => calls.errors.push(String(err?.message || err || message)),
    hiddenPixelWait,
    revealSettle,
  });
  return { source, calls, channel };
}

// settle lets the async command path finish.
const settle = () => new Promise((resolve) => setTimeout(resolve, 0));

test("a CDP command reaches the bridge with its tier, params and domains", async () => {
  const { source, calls } = harness();
  source.frame(JSON.stringify({ id: "c1", method: "Page.captureScreenshot", params: { format: "png" }, tier: "act", domains: ["example.com"] }));
  await settle();
  assert.deepEqual(calls.invoke, [
    ["btab_cdp_call", { id: "7", method: "Page.captureScreenshot", paramsJson: '{"format":"png"}', tier: "act", domains: ["example.com"] }],
  ]);
  assert.deepEqual(calls.post, [{ id: "c1", output: { ok: "btab_cdp_call" } }]);
});

test("a command without a tier is treated as read", async () => {
  const { source, calls } = harness();
  source.frame(JSON.stringify({ id: "c2", method: "DOM.getDocument" }));
  await settle();
  assert.equal(calls.invoke[0][1].tier, "read");
  assert.equal(calls.invoke[0][1].paramsJson, "{}");
});

test("the events verb goes to the shell ring, not to the page", async () => {
  const { source, calls } = harness();
  source.frame(JSON.stringify({ id: "c3", method: EVENTS_VERB, params: { since: 12 } }));
  await settle();
  assert.deepEqual(calls.invoke, [["btab_cdp_events", { id: "7", since: 12 }]]);
  assert.deepEqual(calls.post, [{ id: "c3", output: { ok: "btab_cdp_events" } }]);
});

test("no work-browser tab on screen is an answer, not a hang", async () => {
  const { source, calls } = harness({ tab: "t:5" });
  source.frame(JSON.stringify({ id: "c4", method: "DOM.getDocument" }));
  await settle();
  assert.deepEqual(calls.invoke, []);
  assert.equal(calls.post.length, 1);
  assert.equal(calls.post[0].id, "c4");
  assert.match(calls.post[0].error, /no work-browser tab/);
});

test("a session drive opens that principal's split, not the selected tab", async () => {
  const seen = [];
  const { source, calls } = harness({
    tab: "w:9",
    ensureSession: async (cmd) => {
      seen.push(cmd.agent);
      return { id: "3" };
    },
  });
  source.frame(JSON.stringify({ id: "s1", session: true, agent: "ag-1", method: "shell.open", params: { url: "https://example.com/" } }));
  await settle();
  assert.deepEqual(seen, ["ag-1"]);
  assert.deepEqual(calls.invoke, [["btab_navigate", { id: "3", url: "https://example.com/" }]]);
  assert.equal(calls.post[0].output.opened, true);
  assert.equal(calls.post[0].output.url, "https://example.com/");
});


test("a bridge refusal travels back as the command's error", async () => {
  const { source, calls } = harness({
    invoke: async () => {
      throw new Error("Runtime.evaluate needs the act tier; this agent has read");
    },
  });
  source.frame(JSON.stringify({ id: "c5", method: "Runtime.evaluate", tier: "read" }));
  await settle();
  assert.equal(calls.post[0].id, "c5");
  assert.match(calls.post[0].error, /needs the act tier/);
});

test("frames that are not commands are ignored", async () => {
  const { source, calls } = harness();
  source.frame("not json");
  source.frame(JSON.stringify({ method: "DOM.getDocument" }));
  await settle();
  assert.deepEqual(calls.invoke, []);
  assert.deepEqual(calls.post, []);
});

test("a failed result post is reported and does not throw", async () => {
  const { source, calls } = harness({
    post: async () => {
      throw new Error("result 500");
    },
  });
  source.frame(JSON.stringify({ id: "c6", method: "DOM.getDocument" }));
  await settle();
  assert.deepEqual(calls.post, []);
  assert.equal(calls.errors.length, 1);
  assert.match(calls.errors[0], /result 500/);
});

test("close stops the stream from answering", async () => {
  const { source, calls, channel } = harness();
  channel.close();
  assert.equal(source.listening, false);
});

test("a computer frame goes to the computer runner, never to the tab bridge", async () => {
  const source = fakeSource();
  const calls = { invoke: [], post: [], computer: [] };
  createBrowserChannel({
    activeTabId: () => "w:7",
    invoke: async (cmd, args) => {
      calls.invoke.push([cmd, args]);
      return {};
    },
    source,
    post: async (r) => calls.post.push(r),
    runComputer: async (cmd) => {
      calls.computer.push(cmd.method);
      return { output: { ok: true } };
    },
  });
  source.frame(JSON.stringify({ id: "k1", kind: "computer", method: "screenshot", principal: "agent-1" }));
  await settle();
  assert.deepEqual(calls.invoke, []);
  assert.deepEqual(calls.computer, ["screenshot"]);
  assert.deepEqual(calls.post, [{ id: "k1", output: { ok: true } }]);
});

test("a computer frame on a page without a runner is answered, not hung", async () => {
  const { source, calls } = harness();
  source.frame(JSON.stringify({ id: "k2", kind: "computer", method: "screenshot", principal: "agent-1" }));
  await settle();
  assert.deepEqual(calls.invoke, []);
  assert.equal(calls.post.length, 1);
  assert.match(calls.post[0].error, /^not_connected/);
});

// The session split off screen (ADR-0172): a verb runs where the split is and
// never asks the page to reveal it, except a screenshot that does not paint.
const until = (cond) => new Promise((resolve) => {
  const tick = () => (cond() ? resolve() : setTimeout(tick, 1));
  tick();
});

test("a hidden session split answers a screenshot without being revealed", async () => {
  let reveals = 0;
  const { source, calls } = harness({
    ensureSession: async () => ({ id: "3", onScreen: false, reveal: () => { reveals += 1; } }),
    hiddenPixelWait: 20,
  });
  source.frame(JSON.stringify({ id: "h1", session: true, agent: "ag-1", method: "Page.captureScreenshot" }));
  await until(() => calls.post.length === 1);
  assert.equal(reveals, 0);
  assert.equal(calls.invoke.length, 1);
  assert.deepEqual(calls.post[0].output, { ok: "btab_cdp_call" });
});

test("a hidden split that does not paint is revealed once and asked again", async () => {
  let reveals = 0;
  let n = 0;
  const { source, calls } = harness({
    ensureSession: async () => ({ id: "3", onScreen: false, reveal: () => { reveals += 1; } }),
    hiddenPixelWait: 10,
    revealSettle: 1,
    invoke: async () => {
      n += 1;
      if (n === 1) return new Promise(() => {}); // a hidden WebView2 that never paints
      return { data: "png" };
    },
  });
  source.frame(JSON.stringify({ id: "h2", session: true, agent: "ag-1", method: "Page.captureScreenshot" }));
  await until(() => calls.post.length === 1);
  assert.equal(reveals, 1);
  assert.equal(n, 2);
  assert.deepEqual(calls.post[0].output, { data: "png" });
  assert.match(calls.errors[0], /did not paint/);
});

test("other verbs on a hidden split never reveal it", async () => {
  let reveals = 0;
  const { source, calls } = harness({
    ensureSession: async () => ({ id: "3", onScreen: false, reveal: () => { reveals += 1; } }),
    hiddenPixelWait: 1,
  });
  for (const method of ["Runtime.evaluate", "Input.dispatchMouseEvent", "Accessibility.getFullAXTree"]) {
    source.frame(JSON.stringify({ id: method, session: true, agent: "ag-1", method }));
  }
  await until(() => calls.post.length === 3);
  assert.equal(reveals, 0);
});

test("the events verb on a session drive carries the binding decision", async () => {
  const { source, calls } = harness({
    ensureSession: async () => ({ id: "3", onScreen: false, decision: { host: "ag-1", reveal: "none" } }),
    invoke: async () => ({ events: [], last: 0 }),
  });
  source.frame(JSON.stringify({ id: "e1", session: true, agent: "ag-1", method: "shell.events" }));
  await settle();
  assert.deepEqual(calls.post[0].output, { events: [], last: 0, session: { host: "ag-1", reveal: "none" } });
});
