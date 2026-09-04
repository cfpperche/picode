import assert from "node:assert/strict";
import {
  appendFileSync,
  copyFileSync,
  mkdirSync,
  mkdtempSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

import {
  DOC_SCREENSHOT_SURFACES,
  FINGERPRINT_VERSION,
  SURFACE_PROFILES,
  TUTORIAL_VIDEOS,
  VIDEO_STILL_SURFACES,
  screenshotInputFailures,
  surfaceFingerprint,
  surfaceInputFiles,
} from "./lib/docs-surfaces.mjs";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");

function copySurfaceTree() {
  const target = mkdtempSync(join(tmpdir(), "picode-docs-surfaces-"));
  const files = new Set();
  for (const profile of Object.keys(SURFACE_PROFILES)) {
    for (const pipeline of ["screenshots", "video"]) {
      for (const rel of surfaceInputFiles(root, profile, { pipeline })) files.add(rel);
    }
  }
  for (const rel of files) {
    const out = join(target, rel);
    mkdirSync(dirname(out), { recursive: true });
    copyFileSync(join(root, rel), out);
  }
  return target;
}

function capturedManifest(tree) {
  const surfaces = {};
  for (const [name, profile] of Object.entries(DOC_SCREENSHOT_SURFACES)) {
    surfaces[name] = {
      profile,
      inputHash: surfaceFingerprint(tree, profile, { pipeline: "screenshots" }),
    };
  }
  return { fingerprintVersion: FINGERPRINT_VERSION, surfaces };
}

function changedSurfaces(tree, manifest) {
  return screenshotInputFailures(tree, manifest).map((failure) => failure.split(":", 1)[0]);
}

function withSurfaceTree(run) {
  const tree = copySurfaceTree();
  try {
    run(tree, capturedManifest(tree));
  } finally {
    rmSync(tree, { recursive: true, force: true });
  }
}

test("screenshot freshness decision table", async (t) => {
  await t.test("unchanged inputs keep every surface current", () => {
    withSurfaceTree((tree, manifest) => {
      assert.deepEqual(screenshotInputFailures(tree, manifest), []);
    });
  });

  await t.test("test files and unrelated server handlers change no surface", () => {
    withSurfaceTree((tree, manifest) => {
      for (const rel of ["web/src/lib/appPrimitives.test.js", "internal/server/agent_bash.go"]) {
        const path = join(tree, rel);
        mkdirSync(dirname(path), { recursive: true });
        writeFileSync(path, "unrelated change\n");
      }
      assert.deepEqual(screenshotInputFailures(tree, manifest), []);
    });
  });

  await t.test("a shared style invalidates all public screenshots", () => {
    withSurfaceTree((tree, manifest) => {
      appendFileSync(join(tree, "web/src/styles/app.css"), "\n:root { --docs-test: 1; }\n");
      assert.deepEqual(changedSurfaces(tree, manifest).sort(), Object.keys(DOC_SCREENSHOT_SURFACES).sort());
    });
  });

  await t.test("a shared shell dependency invalidates all public screenshots", () => {
    withSurfaceTree((tree, manifest) => {
      appendFileSync(join(tree, "web/src/lib/feed.js"), "\n// docs fingerprint test\n");
      assert.deepEqual(changedSurfaces(tree, manifest).sort(), Object.keys(DOC_SCREENSHOT_SURFACES).sort());
    });
  });

  await t.test("a dashboard component invalidates only the desktop fleet", () => {
    withSurfaceTree((tree, manifest) => {
      appendFileSync(join(tree, "web/src/components/DashboardView.jsx"), "\n// docs fingerprint test\n");
      assert.deepEqual(changedSurfaces(tree, manifest), ["app-fleet"]);
    });
  });

  await t.test("an Inbox screen component invalidates only the Inbox screenshot", () => {
    withSurfaceTree((tree, manifest) => {
      appendFileSync(join(tree, "web/src/mobile/screens/Inbox.jsx"), "\n// docs fingerprint test\n");
      assert.deepEqual(changedSurfaces(tree, manifest), ["app-mobile-inbox"]);
    });
  });

  await t.test("screenshot capture changes do not stale video surfaces", () => {
    withSurfaceTree((tree) => {
      const before = surfaceFingerprint(tree, "desktop-dashboard", { pipeline: "video" });
      appendFileSync(join(tree, "scripts/docs-shots.mjs"), "\n// docs fingerprint test\n");
      const after = surfaceFingerprint(tree, "desktop-dashboard", { pipeline: "video" });
      assert.equal(after, before);
    });
  });
});

test("tutorials declare only the UI surfaces their stills use", () => {
  const surfaces = Object.fromEntries(
    TUTORIAL_VIDEOS.map((video) => [
      video.id,
      [...new Set(video.stills.map((still) => VIDEO_STILL_SURFACES[still]))],
    ]),
  );
  assert.deepEqual(surfaces, {
    "create-agent": ["desktop-agents", "desktop-create-agent", "desktop-agent"],
    "automate-it": ["desktop-automations", "desktop-inbox"],
    "take-it-anywhere": ["mobile-work", "mobile-agent", "mobile-inbox"],
  });

  for (const [still, profile] of Object.entries(VIDEO_STILL_SURFACES)) {
    assert.ok(SURFACE_PROFILES[profile], `${still} references unknown profile ${profile}`);
  }
});

test("surface input sets contain no test files", () => {
  for (const profile of Object.keys(SURFACE_PROFILES)) {
    const files = surfaceInputFiles(root, profile);
    assert.equal(files.some((file) => /(?:\.test\.[cm]?[jt]sx?|_test\.go)$/.test(file)), false, profile);
  }
});
