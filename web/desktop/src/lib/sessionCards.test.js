import assert from "node:assert/strict";
import { test } from "node:test";
import { cardsFrom, leafUserId } from "./sessionCards.js";

test("user cards swallow tools and replies", () => {
  const cards = cardsFrom([
    {
      id: "u1", role: "user", text: "hi",
      children: [
        { id: "a1", role: "assistant", text: "hello", children: [
          { id: "t1", role: "toolResult", text: "<p>x</p>", children: [] },
        ] },
        { id: "u2", role: "user", text: "again", children: [] },
      ],
    },
  ]);
  assert.equal(cards.length, 1);
  assert.equal(cards[0].text, "hi");
  assert.equal(cards[0].info.some((i) => i.kind === "reply"), true);
  assert.equal(cards[0].info.some((i) => i.kind === "tool" && i.text === "tool"), true);
  assert.equal(cards[0].children.length, 1);
  assert.equal(cards[0].children[0].text, "again");
});

test("leafUserId walks up to the user prompt", () => {
  const tree = [{
    id: "u1", role: "user", text: "hi",
    children: [{ id: "a1", parentId: "u1", role: "assistant", text: "yo", children: [] }],
  }];
  assert.equal(leafUserId(tree, "a1"), "u1");
  assert.equal(leafUserId(tree, "u1"), "u1");
  assert.equal(leafUserId(tree, "gone"), "");
});
