import assert from "node:assert/strict";
import { test } from "node:test";
import { pinFileFromDrop, pinFileURL } from "./pinFileDrop.js";

function ev(map) {
  return { dataTransfer: { getData: (k) => map[k] || "" } };
}

test("pinFileFromDrop", () => {
  assert.deepEqual(
    pinFileFromDrop(ev({ "application/x-picode-pin-file": JSON.stringify({ pinId: "p1", fileId: "f1" }) })),
    { pinId: "p1", fileId: "f1" },
  );
  assert.deepEqual(
    pinFileFromDrop(ev({ "text/uri-list": "https://localhost:8445/api/pins/hello-abc/files/sketch-def" })),
    { pinId: "hello-abc", fileId: "sketch-def" },
  );
  assert.equal(pinFileFromDrop(ev({ "text/plain": "hello" })), null);
  assert.equal(pinFileURL({ pinId: "p", fileId: "f" }), "/api/pins/p/files/f");
});
