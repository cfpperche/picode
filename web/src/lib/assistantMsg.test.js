import assert from "node:assert/strict";
import { test } from "node:test";
import { mergeAssistant, blocksFromMessage } from "./assistantMsg.js";

test("blocksFromMessage", () => {
  const b = blocksFromMessage({
    content: [
      { type: "thinking", thinking: "plan" },
      { type: "text", text: "Teste recebido." },
    ],
  });
  assert.equal(b.length, 2);
  assert.equal(b[1].text, "Teste recebido.");
});

test("mergeAssistant fills empty chat", () => {
  const cur = [{ kind: "block", cls: "user", actor: "You", text: "teste" }];
  const next = mergeAssistant(cur, { content: [{ type: "text", text: "Teste recebido." }] });
  assert.equal(next.length, 2);
  assert.equal(next[1].text, "Teste recebido.");
});

test("mergeAssistant extends streamed prefix", () => {
  const cur = [
    { kind: "block", cls: "user", actor: "You", text: "teste" },
    { kind: "block", cls: "", actor: "agent", text: "Tes" },
  ];
  const next = mergeAssistant(cur, { content: [{ type: "text", text: "Teste recebido." }] });
  assert.equal(next.length, 2);
  assert.equal(next[1].text, "Teste recebido.");
});
