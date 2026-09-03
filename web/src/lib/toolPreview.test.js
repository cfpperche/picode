import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { previewFromDetails } from "./toolPreview.js";

const frame = { image: "data:image/jpeg;base64,AAA", url: "https://example.com", title: "Example" };

describe("previewFromDetails", () => {
  it("accepts a valid preview and normalizes optional fields", () => {
    assert.deepEqual(previewFromDetails({ preview: frame }), {
      image: frame.image, url: frame.url, title: frame.title,
    });
    assert.deepEqual(previewFromDetails({ preview: { image: "data:image/png;base64,BBB" } }), {
      image: "data:image/png;base64,BBB", url: "", title: "",
    });
  });
  it("rejects missing, empty, or malformed previews", () => {
    assert.equal(previewFromDetails(null), null);
    assert.equal(previewFromDetails(undefined), null);
    assert.equal(previewFromDetails({}), null);
    assert.equal(previewFromDetails({ preview: null }), null);
    assert.equal(previewFromDetails({ preview: {} }), null);
    assert.equal(previewFromDetails({ preview: { image: "" } }), null);
    assert.equal(previewFromDetails({ preview: { image: 42 } }), null);
    assert.equal(previewFromDetails("nope"), null);
  });
  it("ignores non-string url/title instead of passing them through", () => {
    assert.deepEqual(previewFromDetails({ preview: { image: "x", url: 7, title: {} } }), {
      image: "x", url: "", title: "",
    });
  });
});
