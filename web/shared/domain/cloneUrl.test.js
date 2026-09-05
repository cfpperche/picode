import test from "node:test";
import assert from "node:assert/strict";
import { deriveRepo, looksLikeRepoUrl, cloneDest, parentDir } from "./cloneUrl.js";

test("deriveRepo accepts the spellings the server accepts", () => {
  const ok = [
    ["https://github.com/octo/hello", "hello", ""],
    ["https://github.com/octo/hello.git", "hello", ""],
    ["https://github.com/octo/hello/", "hello", ""],
    ["https://github.com/octo/hello/tree/main", "hello", "main"],
    ["https://github.com/octo/hello/tree/feat/x/", "hello", "feat/x"],
    ["git@github.com:octo/hello.git", "hello", ""],
    ["ssh://git@host.tld/org/repo.git", "repo", ""],
    ["git://host.tld/org/repo", "repo", ""],
    ["  https://gitlab.com/a/b/c.git  ", "c", ""],
  ];
  for (const [input, name, branch] of ok) {
    assert.deepEqual(deriveRepo(input), { name, branch }, input);
  }
});

test("deriveRepo rejects what the server rejects", () => {
  const bad = [
    "",
    "   ",
    "--upload-pack=/bin/sh",
    "-ogit@github.com:o/r.git",
    "https://github.com/o/r; rm -rf /",
    "url with space",
    "/tmp/some/local/path",
    "~/code/repo",
    "plainword",
    "https://github.com/",
  ];
  for (const input of bad) {
    assert.equal(deriveRepo(input).name, "", input);
    assert.equal(looksLikeRepoUrl(input), false, input);
  }
  // file:// is a server-side rejection (ParseRemote); the form's tolerant
  // mirror also refuses it so no name is derived from a local path.
  assert.equal(deriveRepo("file:///tmp/repo").name, "", "file://");
});

test("cloneDest joins parent and name", () => {
  assert.equal(cloneDest("~/code", "repo"), "~/code/repo");
  assert.equal(cloneDest("~/code/", "repo"), "~/code/repo");
  assert.equal(cloneDest("", "repo"), "~/repo");
});

test("parentDir returns the folder above", () => {
  assert.equal(parentDir("~/code/repo"), "~/code");
  assert.equal(parentDir("/one"), "");
  assert.equal(parentDir(""), "");
});
