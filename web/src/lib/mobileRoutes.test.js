import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { mobileRoute, mobileHash, tabOf, parentHash } from "./mobileRoutes.js";

describe("mobileRoute", () => {
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
    assert.deepEqual(mobileRoute("#/more"), { screen: "more", id: "", section: "" });
    assert.deepEqual(mobileRoute("#/more/providers"), { screen: "more", id: "", section: "providers" });
    assert.deepEqual(mobileRoute("#/more/nope"), { screen: "more", id: "", section: "" });
  });
  it("maps desktop hashes to the closest mobile section instead of a dead end", () => {
    assert.equal(mobileRoute("#/preferences/notifications").section, "preferences");
    assert.equal(mobileRoute("#/providers/new").section, "providers");
    assert.equal(mobileRoute("#/termset/t1").section, "preferences");
    assert.equal(mobileRoute("#/app/inbox").screen, "inbox");
    assert.equal(mobileRoute("#/sessions/w1").screen, "work");
    assert.equal(mobileRoute("#/file/a/x/y").screen, "work");
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
    assert.equal(parentHash(mobileRoute("#/inbox/x")), "#/inbox");
    assert.equal(parentHash(mobileRoute("#/more/system")), "#/more");
    assert.equal(parentHash(mobileRoute("#/more")), "#/");
  });
});
