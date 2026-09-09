import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { mobileRoute, mobileHash, toolHash, tabOf, parentHash } from "./mobileRoutes.js";

describe("mobileRoute", () => {
  it("opens independent integrations deep links from More", () => {
    for (const hash of ["#/more/integrations", "#/integrations", "#/integrations/connectors", "#/integrations/webhooks"]) {
      const route = mobileRoute(hash);
      assert.equal(route.section, "integrations");
      assert.equal(tabOf(route), "more");
      assert.equal(parentHash(route), "#/more");
    }
  });
  it("opens Apps on the phone and keeps the Inbox route", () => {
    const route = mobileRoute("#/app/docker");
    assert.deepEqual(route, { screen: "app", id: "docker", section: "" });
    assert.equal(tabOf(route), "more");
    assert.equal(parentHash(route), "#/more/apps");
    assert.equal(mobileHash("app", "docker"), "#/app/docker");
    assert.equal(mobileRoute("#/more/apps").section, "apps");
    assert.equal(mobileRoute("#/app/inbox").screen, "inbox");
    assert.deepEqual(mobileRoute("#/app/docker/plan/qa%20review"), { screen: "app", id: "docker", section: "", path: "plan/qa review" });
  });
  it("parses the four tabs and the two pushed screens", () => {
    assert.deepEqual(mobileRoute("#/"), { screen: "now", id: "", section: "" });
    assert.deepEqual(mobileRoute(""), { screen: "now", id: "", section: "" });
    assert.deepEqual(mobileRoute("#/inbox"), { screen: "inbox", id: "", section: "" });
    assert.deepEqual(mobileRoute("#/inbox/ib_1%2F2"), { screen: "inbox", id: "ib_1/2", section: "" });
    assert.deepEqual(mobileRoute("#/work"), { screen: "work", id: "", section: "" });
    assert.deepEqual(mobileRoute("#/work/terminals"), { screen: "work", id: "", section: "terminals" });
    assert.deepEqual(mobileRoute("#/work/nope"), { screen: "work", id: "", section: "" });
    assert.deepEqual(mobileRoute("#/agents"), { screen: "work", id: "", section: "agents" });
    assert.deepEqual(mobileRoute("#/agent/ag%3A1"), { screen: "agent", id: "ag:1", section: "" });
    assert.deepEqual(mobileRoute("#/term/t%201"), { screen: "term", id: "t 1", section: "" });
    assert.deepEqual(mobileRoute("#/changes/a/ag1"), { screen: "changes", id: "ag1", section: "agent" });
    assert.deepEqual(mobileRoute("#/changes/w/ws%201"), { screen: "changes", id: "ws 1", section: "workspace" });
    assert.equal(mobileRoute("#/changes/x/ag1").screen, "now");
    assert.equal(mobileHash("changes", "t1", "term"), "#/changes/t/t1");
    assert.deepEqual(mobileRoute("#/more"), { screen: "more", id: "", section: "" });
    assert.deepEqual(mobileRoute("#/more/providers"), { screen: "more", id: "", section: "providers" });
    assert.deepEqual(mobileRoute("#/more/nope"), { screen: "more", id: "", section: "" });
  });
  it("maps desktop hashes to the closest mobile section instead of a dead end", () => {
    assert.equal(mobileRoute("#/preferences/notifications").section, "preferences");
    assert.equal(mobileRoute("#/providers/new").section, "providers");
    assert.equal(mobileRoute("#/termset/t1").section, "preferences");
    assert.equal(mobileRoute("#/app/inbox").screen, "inbox");
    assert.equal(mobileRoute("#/sessions/w1").section, "clis");
    assert.equal(mobileRoute("#/file/a/x/y").screen, "files");
    assert.equal(mobileRoute("#/whatever").screen, "now");
  });
  it("builds hashes that parse back, sharing the desktop agent link", () => {
    for (const [screen, id] of [["now", ""], ["inbox", ""], ["inbox", "i 1"], ["work", ""], ["work", "agents"], ["agent", "ag/1"], ["term", "t1"], ["more", ""], ["more", "system"]]) {
      const r = mobileRoute(mobileHash(screen, id));
      assert.equal(r.screen, screen, screen);
      if (screen === "more" || screen === "work") assert.equal(r.section, id);
      else assert.equal(r.id, id);
    }
    assert.equal(mobileHash("agent", "a1"), "#/agent/a1");
    assert.equal(mobileHash("term", "t1"), "#/term/t1");
  });
  it("lights the parent tab for pushed screens and knows where Back lands", () => {
    assert.equal(tabOf(mobileRoute("#/agent/a1")), "work");
    assert.equal(tabOf(mobileRoute("#/term/t1")), "work");
    assert.equal(tabOf(mobileRoute("#/work/agents")), "work");
    assert.equal(tabOf(mobileRoute("#/inbox/x")), "inbox");
    assert.equal(tabOf(mobileRoute("#/more/system")), "more");
    assert.equal(tabOf(mobileRoute("#/")), "now");
    assert.equal(parentHash(mobileRoute("#/agent/a1")), "#/work");
    assert.equal(parentHash(mobileRoute("#/term/t1")), "#/work/terminals");
    assert.equal(parentHash(mobileRoute("#/changes/a/ag1")), "#/agent/ag1");
    assert.equal(parentHash(mobileRoute("#/changes/t/t1")), "#/term/t1");
    assert.equal(parentHash(mobileRoute("#/changes/w/w1")), "#/work");
    assert.equal(tabOf(mobileRoute("#/changes/a/ag1")), "work");
    assert.equal(parentHash(mobileRoute("#/inbox/x")), "#/inbox");
    assert.equal(parentHash(mobileRoute("#/more/system")), "#/more");
    assert.equal(parentHash(mobileRoute("#/more")), "#/");
  });
});

it("opens complete automation and session workflows without dropping nested links", () => {
  for (const hash of ["#/automations", "#/automations/new", "#/automations/saved", "#/more/automations"]) {
    assert.equal(mobileRoute(hash).section, "automations");
    assert.equal(tabOf(mobileRoute(hash)), "more");
  }
  for (const hash of ["#/clis/sessions", "#/clis/sessions/w1?cli=claude", "#/sessions/w1"]) {
    assert.equal(mobileRoute(hash).section, "clis");
  }
});

it("llama deep links open the dedicated manager", () => {
 for (const hash of ["#/llama", "#/llama/models", "#/llama/server", "#/llama/activity", "#/providers/llama"]) assert.deepEqual(mobileRoute(hash), { screen: "more", id: "", section: "llama" });
});

it("opens editor/tree/Git links with owner identity and folder preconditions", () => {
  for (const kind of ["agent", "term", "workspace"]) {
    const owner = { kind, id: "id /?#" };
    for (const screen of ["files", "git"]) {
      const opts = { root: "/tmp/project ?#", ...(screen === "files" ? { path: "src/app ?#.js" } : { commit: "abc123" }) };
      const route = mobileRoute(toolHash(screen, owner, opts));
      assert.deepEqual(route, { screen, id: owner.id, section: kind, ...opts });
      assert.equal(tabOf(route), "work");
      assert.equal(parentHash(route), kind === "workspace" ? "#/work" : mobileHash(kind, owner.id));
      assert.equal(mobileRoute(toolHash(screen, owner)).screen, screen);
    }
  }
  assert.equal(mobileRoute("#/tree/w/project").screen, "files");
  assert.equal(mobileRoute("#/git/a/agent").screen, "git");
  assert.equal(mobileRoute("#/git/x/nope").screen, "work");
});

it("pins on the phone: list under More, read-only screen, new and edit forms", () => {
  assert.deepEqual(mobileRoute("#/pins/deploy-abc123"), { screen: "pin", id: "deploy-abc123", section: "" });
  assert.deepEqual(mobileRoute("#/pins/new"), { screen: "pinEdit", id: "", section: "" });
  assert.deepEqual(mobileRoute("#/pins/deploy-abc123/edit"), { screen: "pinEdit", id: "deploy-abc123", section: "" });
  assert.deepEqual(mobileRoute("#/pins"), { screen: "more", id: "", section: "pins" });
  assert.equal(mobileHash("pin", "deploy-abc123"), "#/pins/deploy-abc123");
  assert.equal(mobileHash("pinEdit", "deploy-abc123"), "#/pins/deploy-abc123/edit");
  assert.equal(mobileHash("pinEdit", ""), "#/pins/new");
  assert.equal(tabOf({ screen: "pin", id: "x" }), "more");
  assert.equal(tabOf({ screen: "pinEdit", id: "" }), "more");
  assert.equal(parentHash({ screen: "pin", id: "x" }), "#/more/pins");
  assert.equal(parentHash({ screen: "pinEdit", id: "x" }), "#/pins/x");
  assert.equal(parentHash({ screen: "pinEdit", id: "" }), "#/more/pins");
});

it("native packages and legacy configuration links use Agent CLIs", () => {
  for (const hash of ["#/packages", "#/more/packages", "#/packages/config/pi-roles", "#/clis/packages/pi?agentId=a", "#/clis/packages/pi/config/pi-roles?workspaceId=w"]) {
    assert.deepEqual(mobileRoute(hash), { screen: "more", id: "", section: "clis" });
  }
});
