import assert from "node:assert/strict";
import { test } from "node:test";
import { clearPreviewOverlay, mintPreview, previewForms, previewReachable, putPreviewOverlay } from "./preview.js";

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

test("putPreviewOverlay PUTs the editor buffer to the ticket", async () => {
  const f = fakeFetch({ ok: true, status: 204 });
  const original = globalThis.fetch;
  globalThis.fetch = f.impl;
  try {
    await putPreviewOverlay("/preview/tok/index.html", "<h1>não salvo</h1>");
    assert.equal(f.calls.length, 1);
    assert.equal(f.calls[0].url, "/preview/tok/index.html");
    assert.equal(f.calls[0].opts.method, "PUT");
    assert.equal(f.calls[0].opts.body, "<h1>não salvo</h1>");
  } finally {
    globalThis.fetch = original;
  }
});

test("putPreviewOverlay throws when the ticket refuses", async () => {
  const f = fakeFetch({ ok: false, status: 413 });
  const original = globalThis.fetch;
  globalThis.fetch = f.impl;
  try {
    await assert.rejects(() => putPreviewOverlay("/preview/tok/index.html", "x"), (err) => err.status === 413);
  } finally {
    globalThis.fetch = original;
  }
});

test("previewForms prefers the ticket's own origin over the sandbox", () => {
  const mint = {
    path: "site/index.html",
    sandbox: { url: "/preview/tok/site/index.html", events: "/preview/tok/__events" },
    origin: { url: "http://abc.localhost:8473/site/index.html", events: "http://abc.localhost:8473/__events" },
  };
  assert.deepEqual(previewForms(mint), [
    { mode: "origin", url: "http://abc.localhost:8473/site/index.html", events: "http://abc.localhost:8473/__events" },
    { mode: "sandbox", url: "/preview/tok/site/index.html", events: "/preview/tok/__events" },
  ]);
  // A remote mint carries no origin: the path form is all there is.
  assert.deepEqual(previewForms({ sandbox: mint.sandbox }), [
    { mode: "sandbox", url: "/preview/tok/site/index.html", events: "/preview/tok/__events" },
  ]);
  assert.deepEqual(previewForms({}), []);
  assert.deepEqual(previewForms(null), []);
});

test("clearPreviewOverlay DELETEs the overlay and surfaces refusals", async () => {
  const f = fakeFetch({ ok: true, status: 204 });
  const original = globalThis.fetch;
  globalThis.fetch = f.impl;
  try {
    await clearPreviewOverlay("http://abc.localhost:8473/site/index.html");
    assert.equal(f.calls[0].url, "http://abc.localhost:8473/site/index.html");
    assert.equal(f.calls[0].opts.method, "DELETE");
  } finally {
    globalThis.fetch = original;
  }
  const bad = fakeFetch({ ok: false, status: 404 });
  globalThis.fetch = bad.impl;
  try {
    await assert.rejects(
      () => clearPreviewOverlay("http://gone.localhost:8473/site/index.html"),
      (err) => err.status === 404,
    );
  } finally {
    globalThis.fetch = original;
  }
});
