import test from "node:test";
import assert from "node:assert/strict";
import { wireTerminalRuntime } from "./terminalRuntime.js";
import { dropTermSocket } from "./termSocket.js";

test("one runtime wires input, resize, output, touch, clipboard and reconnect callbacks", t => {
  const sent = [], lines = [], written = [], events = new Map();
  class Socket {
    static OPEN = 1;
    readyState = 1;
    constructor(url) { this.url = url; }
    send(data) { sent.push(data); }
    close() {}
  }
  const previous = {};
  for (const [key, value] of Object.entries({
    WebSocket: Socket,
    ResizeObserver: class { observe() {} disconnect() {} },
    requestAnimationFrame: () => 1,
    cancelAnimationFrame: () => {},
  })) { previous[key] = globalThis[key]; globalThis[key] = value; }
  t.after(() => {
    for (const key of Object.keys(previous)) {
      if (previous[key] === undefined) delete globalThis[key]; else globalThis[key] = previous[key];
    }
  });
  let input, resize, wheel, keyHandler, osc, opened = 0;
  const term = {
    cols: 80, rows: 24, modes: { mouseTrackingMode: "none" },
    buffer: { active: { type: "alternate" } },
    attachCustomWheelEventHandler(fn) { wheel = fn; },
    attachCustomKeyEventHandler(fn) { keyHandler = fn; },
    parser: { registerOscHandler(code, fn) { assert.equal(code, 52); osc = fn; } },
    onData(fn) { input = fn; }, onResize(fn) { resize = fn; },
    write(data) { written.push(data); }, writeln(line) { lines.push(line); },
  };
  const entry = { term, fit: {}, paneEl: {
    addEventListener: (name, fn) => events.set(name, fn),
    removeEventListener: name => events.delete(name),
  } };
  wireTerminalRuntime(entry, { url: "ws://scratch/ws/term?session=fixture", reservedKey: e => e.key === "Escape", onOpen: () => opened++ });
  assert.equal(entry.sock.url, "ws://scratch/ws/term?session=fixture");
  input("hello");
  assert.equal(new TextDecoder().decode(sent.pop()), "hello");
  resize();
  assert.deepEqual(JSON.parse(sent.pop()), { type: "resize", cols: 80, rows: 24 });
  term.cols = 1; resize(); assert.equal(sent.length, 0);
  wheel({ deltaY: 40, preventDefault() {} });
  assert.match(new TextDecoder().decode(sent.pop()), /\x1b\[<65;/);
  assert.equal(keyHandler({ type: "keydown", key: "Escape" }), false);
  assert.equal(osc("c;?"), true, "clipboard reads stay refused");
  entry.sock.onmessage({ data: new Uint8Array([65]).buffer });
  assert.equal(written[0][0], 65);
  entry.sock.onmessage({ data: '{"type":"error","message":"fixture failure"}' });
  assert.match(lines[0], /fixture failure/);
  entry.sock.onopen(); assert.equal(opened, 1);
  assert.equal(events.size, 4);
  entry.unwireTouch(); entry.unwireFit(); dropTermSocket(entry);
  assert.equal(events.size, 0);
});
