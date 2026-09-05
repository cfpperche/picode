import { test } from "node:test";
import assert from "node:assert/strict";
import { captureState, previewFromDetails, updateCapture, toolResultDetail, MAX_CAPTURE_BYTES } from "./toolPreview.js";

const image = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aX1kAAAAASUVORK5CYII=";
const frame = { image, url: "https://example.com/", title: "Example" };

test("legacy bounded capture works without invented time or source", () => {
  assert.deepEqual(previewFromDetails({ preview: frame }), frame);
  assert.deepEqual(previewFromDetails({ preview: { image } }), { image, url: "", title: "" });
});

test("capture policy decision table rejects remote, malformed and excessive input", () => {
  for (const value of [null, undefined, {}, { preview: null }]) assert.equal(previewFromDetails(value), null);
  for (const image of [undefined, "", 42, "x", "https://example.com/a.png", "http://localhost/a.png", "/api/file", "//example.com", "blob:x", "file:///tmp/a.png", "data:image/svg+xml;base64,PHN2Zy8+", "data:image/png;base64,AAA", "data:image/png;base64,!!!!", "data:image/png;base64," + "A".repeat(MAX_CAPTURE_BYTES * 2)]) {
    assert.equal(previewFromDetails({ preview: { image } }), null, String(image).slice(0, 60));
    assert.equal(captureState({ preview: { image } }).previewError, "Capture unavailable");
  }
  assert.equal(captureState({}).previewError, "");
});

test("header dimensions bound decompression before browser loading", () => {
  for (const [w, h] of [[0, 1], [1601, 1], [1600, 1600], [0xffffffff, 1]]) {
    const bytes = Buffer.from(image.split(",")[1], "base64");
    bytes.writeUInt32BE(w, 16); bytes.writeUInt32BE(h, 20);
    assert.equal(previewFromDetails({ preview: { image: "data:image/png;base64," + bytes.toString("base64") } }), null);
  }
});

test("JPEG SOF dimensions validated; truncated markers are rejected", () => {
  const jpeg = Buffer.from([0xff,0xd8,0xff,0xc0,0,11,8,0,10,0,20,1,1,0x11,0,0xff,0xd9]);
  const p = previewFromDetails({ preview: { image: "data:image/jpeg;base64," + jpeg.toString("base64") } });
  assert.ok(p);
  for (let n = 1; n < 15; n++) assert.equal(previewFromDetails({ preview: { image: "data:image/jpeg;base64," + jpeg.subarray(0,n).toString("base64") } }), null);
});

test("caption strips credentials/query/fragment and bounds labels", () => {
  const p = previewFromDetails({ preview: { image, title: "a".repeat(1000), url: "https://u:secret@example.com/page?token=secret#private", ts: 12345, source: "session/tab" } });
  assert.equal(p.url, "https://example.com/page");
  assert.equal(p.title.length, 160);
  assert.equal(p.ts, 12345);
  assert.equal(p.source, "session/tab");
  for (const ts of [Infinity, -1, 1.2, "123", 8640000000000001]) assert.equal(previewFromDetails({ preview: { image, ts } }).ts, undefined);
  assert.deepEqual(previewFromDetails({ preview: { image, title: {}, url: 7 } }), { image, url: "", title: "" });
});

test("update decisions: no metadata, older frame, terminal tool, invalid frame", () => {
  const item = { status: "···", preview: { ...frame, ts: 200 } };
  assert.equal(updateCapture(item, {}), item);
  assert.equal(updateCapture(item, { preview: { ...frame, ts: 100 } }), item);
  const done = { ...item, status: "ok" };
  assert.equal(updateCapture(done, { preview: { ...frame, ts: 300 } }), done);
  assert.equal(updateCapture(item, { preview: { image: "https://example.com" } }).previewError, "Capture unavailable");
  assert.equal(updateCapture(item, { preview: { ...frame, ts: 300 } }).preview.ts, 300);
});

test("text detail never includes preview pixels or rejected URL", () => {
  for (const pixels of [image, "https://example.com/tracker"]) {
    const result = { details: { preview: { image: pixels }, verified: false }, content: [{ type: "text", text: "OK" }] };
    const detail = toolResultDetail(result);
    assert.ok(!detail.includes(pixels));
    assert.match(detail, /capture omitted/);
    assert.match(detail, /verified/);
    assert.equal(result.details.preview.image, pixels);
  }
});
