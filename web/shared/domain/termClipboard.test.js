import assert from "node:assert/strict";
import { test } from "node:test";
import { decodeOsc52, wireTermClipboard, MAX_OSC52_BASE64 } from "./termClipboard.js";

const b64 = (s) => Buffer.from(s, "utf8").toString("base64");

test("a plain copy decodes", () => {
  assert.equal(decodeOsc52("c;" + b64("hello")), "hello");
  // An empty selection field means the default clipboard.
  assert.equal(decodeOsc52(";" + b64("hello")), "hello");
});

test("UTF-8 survives the round trip", () => {
  const text = "acentuação — ✓ 日本語";
  assert.equal(decodeOsc52("c;" + b64(text)), text);
});

test("the read form is refused", () => {
  // OSC 52 ; c ; ? asks the terminal to *report* the clipboard. Answering it
  // would let anything in the pane read what the user copied.
  assert.equal(decodeOsc52("c;?"), null);
  assert.equal(decodeOsc52(";?"), null);
});

test("an empty payload does not clear the clipboard", () => {
  // The spec says empty clears the selection. Wiping what the user copied,
  // because a stray sequence went by, is not worth honouring.
  assert.equal(decodeOsc52("c;"), null);
});

test("selections a browser does not have are ignored, not aliased", () => {
  assert.equal(decodeOsc52("p;" + b64("primary")), null);
  assert.equal(decodeOsc52("s;" + b64("cut buffer")), null);
  assert.equal(decodeOsc52("0;" + b64("buffer 0")), null);
});

test("malformed payloads are dropped, never pasted raw", () => {
  assert.equal(decodeOsc52("no-semicolon"), null);
  assert.equal(decodeOsc52("c;not!valid!base64!"), null);
  assert.equal(decodeOsc52(""), null);
  assert.equal(decodeOsc52(null), null);
  assert.equal(decodeOsc52(undefined), null);
});

test("whitespace inside the payload is tolerated", () => {
  const wrapped = b64("hello").slice(0, 3) + "\n " + b64("hello").slice(3);
  assert.equal(decodeOsc52("c;" + wrapped), "hello");
});

test("an oversized payload is refused whole, not truncated", () => {
  const big = "A".repeat(MAX_OSC52_BASE64 + 4);
  assert.equal(decodeOsc52("c;" + big), null);
});

// A fake terminal exposing just the parser surface the handler uses.
function fakeTerm() {
  const t = { handler: null, disposed: false };
  return {
    calls: t,
    parser: {
      registerOscHandler(code, fn) {
        assert.equal(code, 52);
        t.handler = fn;
        return { dispose: () => { t.disposed = true; } };
      },
    },
  };
}

test("the handler copies and always claims the sequence", () => {
  const term = fakeTerm();
  const written = [];
  wireTermClipboard(term, { write: (text) => { written.push(text); } });

  assert.equal(term.calls.handler("c;" + b64("copied")), true);
  assert.deepEqual(written, ["copied"]);

  // Refused input is still consumed: returning false would let the escape
  // sequence fall through and print as garbage.
  assert.equal(term.calls.handler("c;?"), true);
  assert.equal(term.calls.handler("garbage"), true);
  assert.deepEqual(written, ["copied"], "nothing else reached the clipboard");
});

test("a clipboard refusal is reported, not thrown", () => {
  const term = fakeTerm();
  const errors = [];
  wireTermClipboard(term, {
    write: () => { throw new Error("no user gesture"); },
    onError: (e) => errors.push(e.message),
  });
  // The browser can refuse a write with no user gesture. Losing the copy is
  // acceptable; taking the terminal down with it is not.
  assert.equal(term.calls.handler("c;" + b64("x")), true);
  assert.deepEqual(errors, ["no user gesture"]);
});

test("a rejected promise is caught too", async () => {
  const term = fakeTerm();
  const errors = [];
  wireTermClipboard(term, {
    write: () => Promise.reject(new Error("denied")),
    onError: (e) => errors.push(e.message),
  });
  term.calls.handler("c;" + b64("x"));
  await new Promise((r) => setTimeout(r, 0));
  assert.deepEqual(errors, ["denied"]);
});

test("wiring returns the disposable xterm handed back", () => {
  const term = fakeTerm();
  const d = wireTermClipboard(term, { write: () => {} });
  d.dispose();
  assert.equal(term.calls.disposed, true);
});
