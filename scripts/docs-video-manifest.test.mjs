import assert from "node:assert/strict";
import test from "node:test";

import { manifestFailures } from "./docs-video-manifest.mjs";

function manifest({ input = "input-a", composition = "comp-a", still = "still-a", mp4 = "mp4-a" } = {}) {
  return {
    fingerprintVersion: 1,
    videos: {
      tutorial: {
        file: "tutorial.mp4",
        composition: "index.html",
        compositionHash: composition,
        stills: {
          screen: {
            sha256: still,
            surface: "desktop-agent",
            inputHash: input,
          },
        },
        mp4,
      },
    },
  };
}

const shipped = (hash = "mp4-a") => () => hash;

test("video delivery decision table", async (t) => {
  await t.test("matching inputs pass both the CI floor and strict freshness", () => {
    const committed = manifest();
    const fresh = manifest();
    assert.deepEqual(manifestFailures(committed, fresh, { publishedHash: shipped() }), []);
    assert.deepEqual(manifestFailures(committed, fresh, { strict: true, publishedHash: shipped() }), []);
  });

  await t.test("surface drift passes CI but fails the explicit freshness audit", () => {
    const committed = manifest({ input: "input-before" });
    const fresh = manifest({ input: "input-after" });
    assert.deepEqual(manifestFailures(committed, fresh, { publishedHash: shipped() }), []);
    assert.match(
      manifestFailures(committed, fresh, { strict: true, publishedHash: shipped() }).join("\n"),
      /desktop-agent inputs changed/,
    );
  });

  await t.test("a changed composition or referenced still fails the CI floor", () => {
    const committed = manifest();
    const fresh = manifest({ composition: "comp-b", still: "still-b" });
    const failures = manifestFailures(committed, fresh, { publishedHash: shipped() }).join("\n");
    assert.match(failures, /composition changed/);
    assert.match(failures, /still screen changed/);
  });

  await t.test("a missing source still fails even if a bad manifest recorded it missing", () => {
    const committed = manifest({ still: "MISSING" });
    const fresh = manifest({ still: "MISSING" });
    assert.match(
      manifestFailures(committed, fresh, { publishedHash: shipped() }).join("\n"),
      /still screen missing/,
    );
  });

  await t.test("a missing or altered shipped MP4 fails the CI floor", () => {
    const committed = manifest();
    const fresh = manifest();
    assert.match(
      manifestFailures(committed, fresh, { publishedHash: shipped(null) }).join("\n"),
      /tutorial\.mp4 missing/,
    );
    assert.match(
      manifestFailures(committed, fresh, { publishedHash: shipped("tampered") }).join("\n"),
      /does not match manifest/,
    );
  });
});
