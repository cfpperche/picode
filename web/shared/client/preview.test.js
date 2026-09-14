import assert from "node:assert/strict";
import { test } from "node:test";
import { mintPreview, previewReachable } from "./preview.js";

function fakeFetch(result) {
  const calls = [];
  const impl = async (url, opts) => {
    calls.push({ url, opts });
    return result;
  };
  return { impl, calls };
}

function okResponse(body) {
  return { ok: true, status: 200, statusText: "OK", json: async () => body };
}

test("mintPreview posts the owner, path and root", async () => {
  const f = fakeFetch(okResponse({ url: "/preview/tok/site/index.html", expiresAt: "2026-09-14T13:00:00Z" }));
  const original = globalThis.fetch;
  globalThis.fetch = f.impl;
  try {
    const out = await mintPreview({ kind: "term", id: "t1" }, "site/index.html", "/proj");
    assert.equal(out.url, "/preview/tok/site/index.html");
    assert.equal(f.calls.length, 1);
    assert.equal(f.calls[0].url, "/api/previews");
    assert.equal(f.calls[0].opts.method, "POST");
    assert.deepEqual(JSON.parse(f.calls[0].opts.body), {
      kind: "term", id: "t1", path: "site/index.html", root: "/proj",
    });
  } finally {
    globalThis.fetch = original;
  }
});

test("mintPreview surfaces the server message", async () => {
  const f = fakeFetch({
    ok: false, status: 400, statusText: "Bad Request",
    json: async () => ({ error: "only .html and .htm files preview" }),
  });
  const original = globalThis.fetch;
  globalThis.fetch = f.impl;
  try {
    await assert.rejects(
      () => mintPreview({ kind: "workspace", id: "w" }, "notes.md", ""),
      (err) => err.message === "only .html and .htm files preview" && err.status === 400,
    );
  } finally {
    globalThis.fetch = original;
  }
});

test("previewReachable treats only a real refusal as failure", async () => {
  const original = globalThis.fetch;
  const cases = [
    { status: 200, ok: true, want: true },
    { status: 404, ok: false, want: false },
    { status: 403, ok: false, want: false },
    { status: 405, ok: false, want: true }, // HEAD-hostile proxy
  ];
  try {
    for (const c of cases) {
      globalThis.fetch = async () => ({ ok: c.ok, status: c.status });
      assert.equal(await previewReachable("/preview/tok/index.html"), c.want, `status ${c.status}`);
    }
  } finally {
    globalThis.fetch = original;
  }
});
