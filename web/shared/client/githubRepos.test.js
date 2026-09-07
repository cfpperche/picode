import { describe, it, mock } from "node:test";
import assert from "node:assert/strict";
import { fetchGithubRepos } from "./githubRepos.js";

function withFetch(fn) {
  const real = globalThis.fetch;
  globalThis.fetch = fn;
  return () => { globalThis.fetch = real; };
}

function jsonRes(body, ok = true, status = 200) {
  return { ok, status, statusText: status === 200 ? "OK" : "Err", json: async () => body };
}

describe("fetchGithubRepos", () => {
  it("maps an available payload onto ready with repos", async () => {
    const undo = withFetch(async () => jsonRes({ available: true, repos: [{ nameWithOwner: "o/r", url: "https://x" }] }));
    try {
      const res = await fetchGithubRepos();
      assert.deepEqual(res, { phase: "ready", repos: [{ nameWithOwner: "o/r", url: "https://x" }] });
    } finally { undo(); }
  });

  it("passes refresh=1 through to the endpoint", async () => {
    let asked = "";
    const undo = withFetch(async (path) => { asked = path; return jsonRes({ available: true, repos: [] }); });
    try {
      await fetchGithubRepos(true);
      assert.equal(asked, "/api/github/repos?refresh=1");
    } finally { undo(); }
  });

  it("classifies the visible reasons into blocked kinds", async () => {
    for (const [reason, kind] of [
      ["GitHub CLI (gh) is not installed on this machine", "install"],
      ["GitHub CLI is not logged in — run gh auth login", "login"],
    ]) {
      const undo = withFetch(async () => jsonRes({ available: false, reason }));
      try {
        const res = await fetchGithubRepos();
        assert.deepEqual(res, { phase: "blocked", kind, reason });
      } finally { undo(); }
    }
  });

  it("degrades a server list failure to an error with Retry", async () => {
    const undo = withFetch(async () => jsonRes({ available: false, reason: "could not list repositories — check your connection and gh login" }));
    try {
      const res = await fetchGithubRepos();
      assert.equal(res.phase, "error");
      assert.match(res.reason, /could not list/);
    } finally { undo(); }
  });

  it("degrades a network failure to an error, not a crash", async () => {
    const undo = withFetch(async () => { throw new TypeError("fetch failed"); });
    try {
      const res = await fetchGithubRepos();
      assert.equal(res.phase, "error");
      assert.ok(res.reason.length > 0);
    } finally { undo(); }
  });
});
