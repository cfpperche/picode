import test from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const script = join(root, "scripts/release-notes.mjs");
const runOptions = { cwd: root, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] };

test("release-notes emits the tagged changelog section", () => {
  const notes = execFileSync(process.execPath, [script, "v0.1.0"], runOptions);
  assert.match(notes, /^# PiCode v0\.1\.0/);
  assert.match(notes, /### Added/);
  assert.doesNotMatch(notes, /See CHANGELOG\.md for what changed/);
});

test("release-notes rejects a tag without a changelog section", () => {
  assert.throws(
    () => execFileSync(process.execPath, [script, "9.9.9", "--check"], runOptions),
    (error) => error && error.status === 1,
  );
});
