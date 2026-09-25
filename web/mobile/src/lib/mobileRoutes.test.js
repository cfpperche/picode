import { prefSection } from "./routes.js";
import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { mobileRoute, mobileHash, toolHash, tabOf, parentHash } from "./mobileRoutes.js";

describe("mobileRoute", () => {
  it("opens independent integrations deep links from More", () => {
    const webhooks = mobileRoute("#/integrations/webhooks");
    assert.equal(webhooks.section, "integrations");
    assert.equal(tabOf(webhooks), "more");
    // `#/integrations*` stopped meaning Pi's connectors on 2026-09-25.
    for (const hash of ["#/more/integrations", "#/integrations", "#/integrations/connectors"]) {
      assert.equal(mobileRoute(hash).section, "integrations", hash);
    }
  });
  it("opens Apps on the phone and keeps the Inbox route", () => {
    const route = mobileRoute("#/app/docker");
    assert.deepEqual(route, { screen: "app", id: "docker", section: "" });
    assert.equal(tabOf(route), "more");
    assert.equal(parentHash(route), "#/more/apps");
    assert.equal(mobileHash("app", "docker"), "#/app/docker");
    assert.equal(mobileRoute("#/more/apps").section, "apps");
    // #/app/inbox was the Inbox's old address, retired 2026-09-25.
    assert.notEqual(mobileRoute("#/app/inbox").screen, "inbox");
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
    assert.deepEqual(mobileRoute("#/agent/ag%3A1?view=terminal"), { screen: "agent", id: "ag:1", section: "", view: "terminal" });
    assert.deepEqual(mobileRoute("#/term/t%201"), { screen: "term", id: "t 1", section: "" });
    // The pre-Inspector #/changes/* address was retired 2026-09-25.
    assert.notEqual(mobileRoute("#/changes/a/ag1").screen, "inspector");
    assert.deepEqual(mobileRoute("#/inspector/a/ag1"), { screen: "inspector", id: "ag1", section: "agent" });
    assert.deepEqual(mobileRoute("#/inspector/t/t%201?root=%2Fw%2Fapp&view=pr"), { screen: "inspector", id: "t 1", section: "term", root: "/w/app", view: "pr" });
    assert.equal(mobileHash("inspector", "t1", "term"), "#/inspector/t/t1");
    assert.deepEqual(mobileRoute("#/more"), { screen: "more", id: "", section: "" });
    assert.deepEqual(mobileRoute("#/more/providers"), { screen: "more", id: "", section: "" });
    assert.deepEqual(mobileRoute("#/more/nope"), { screen: "more", id: "", section: "" });
  });
  it("maps desktop hashes to the closest mobile section instead of a dead end", () => {
    assert.equal(mobileRoute("#/preferences/notifications").section, "preferences");
    assert.equal(mobileRoute("#/termset/t1").section, "preferences");
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
    assert.equal(mobileHash("agent", "a1", "", "terminal"), "#/agent/a1?view=terminal");
    assert.equal(mobileHash("agent", "a1", "", "chat"), "#/agent/a1?view=chat");
    assert.equal(mobileRoute("#/agent/a1?view=chat").view, "chat");
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
    // Back lands where the resource lives: a workspace's agent or terminal
    // into the Workspaces view focused on that group, a free one into its
    // own flat list, an unknown owner into the legacy parent.
    assert.equal(parentHash(mobileRoute("#/agent/a1"), "ws1"), "#/work/workspaces/ws1");
    assert.equal(parentHash(mobileRoute("#/agent/a1"), "ws /?#"), "#/work/workspaces/ws%20%2F%3F%23");
    assert.equal(parentHash(mobileRoute("#/agent/a1"), null), "#/work/agents");
    assert.equal(parentHash(mobileRoute("#/term/t1"), "ws1"), "#/work/workspaces/ws1");
    assert.equal(parentHash(mobileRoute("#/term/t1"), null), "#/work/terminals");
    assert.deepEqual(mobileRoute("#/work/workspaces/ws%201"), { screen: "work", id: "ws 1", section: "workspaces" });
    assert.deepEqual(mobileRoute("#/work/workspaces"), { screen: "work", id: "", section: "workspaces" });
    assert.deepEqual(mobileRoute("#/work/nope/x"), { screen: "work", id: "", section: "" });
    assert.equal(parentHash(mobileRoute("#/inspector/a/ag1")), "#/agent/ag1");
    assert.equal(parentHash(mobileRoute("#/inspector/t/t1")), "#/term/t1");
    assert.equal(parentHash(mobileRoute("#/inspector/w/w1")), "#/work");
    assert.equal(tabOf(mobileRoute("#/inspector/a/ag1")), "work");
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
  for (const hash of ["#/clis/codex/sessions", "#/clis/pi/sessions/w1"]) {
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

it("snippets on the phone mirror the pins map (ADR-0130)", () => {
  assert.deepEqual(mobileRoute("#/snippets/deploy-abc123"), { screen: "snip", id: "deploy-abc123", section: "" });
  assert.deepEqual(mobileRoute("#/snippets/new"), { screen: "snipEdit", id: "", section: "" });
  assert.deepEqual(mobileRoute("#/snippets/deploy-abc123/edit"), { screen: "snipEdit", id: "deploy-abc123", section: "" });
  assert.deepEqual(mobileRoute("#/snippets"), { screen: "more", id: "", section: "snippets" });
  assert.deepEqual(mobileRoute("#/more/snippets"), { screen: "more", id: "", section: "snippets" });
  assert.deepEqual(mobileRoute("#/more/outcomes"), { screen: "more", id: "", section: "outcomes" });
  assert.deepEqual(mobileRoute("#/outcomes"), { screen: "more", id: "", section: "outcomes" });
  assert.deepEqual(mobileRoute("#/more/history"), { screen: "more", id: "", section: "history" });
  assert.deepEqual(mobileRoute("#/history"), { screen: "more", id: "", section: "history" });
  assert.equal(mobileHash("snip", "deploy-abc123"), "#/snippets/deploy-abc123");
  assert.equal(mobileHash("snipEdit", "deploy-abc123"), "#/snippets/deploy-abc123/edit");
  assert.equal(mobileHash("snipEdit", ""), "#/snippets/new");
  assert.equal(tabOf({ screen: "snip", id: "x" }), "more");
  assert.equal(tabOf({ screen: "snipEdit", id: "" }), "more");
  assert.equal(parentHash({ screen: "snip", id: "x" }), "#/more/snippets");
  assert.equal(parentHash({ screen: "snipEdit", id: "x" }), "#/snippets/x");
  assert.equal(parentHash({ screen: "snipEdit", id: "" }), "#/more/snippets");
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

it("native packages links use Agent CLIs; the Pi-era addresses are retired", () => {
  for (const hash of ["#/clis/pi/packages", "#/clis/pi/packages/config/pi-roles?workspaceId=w", "#/clis/codex/connectors"]) {
    assert.deepEqual(mobileRoute(hash), { screen: "more", id: "", section: "clis" });
  }
  for (const hash of ["#/packages", "#/more/packages", "#/packages/config/pi-roles", "#/mcps", "#/more/mcps"]) {
    assert.notEqual(mobileRoute(hash).section, "clis", hash);
  }
});

it("native provider navigation; the Pi-era aliases are retired", () => {
  for (const hash of ["#/clis/pi/providers", "#/clis/pi/providers/new", "#/clis/codex/providers"]) assert.deepEqual(mobileRoute(hash), { screen: "more", id: "", section: "clis" });
  for (const hash of ["#/providers", "#/providers/new", "#/more/providers", "#/more/providers/new"]) assert.notEqual(mobileRoute(hash).section, "clis", hash);
  assert.deepEqual(mobileRoute("#/more/providers/llama?tab=models"), { screen: "more", id: "", section: "llama" });
});


it("Preferences knows the Landing work tab", () => {
  assert.equal(prefSection("#/preferences/landing"), "landing");
  assert.equal(prefSection("#/preferences/nope"), "appearance");
});
