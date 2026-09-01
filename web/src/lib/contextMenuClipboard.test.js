import { test } from "node:test";
import assert from "node:assert/strict";
import { isEditableTarget, insertAtCaret } from "./contextMenuClipboard.js";

test("isEditableTarget accepts textareas and text-bearing inputs, rejects the rest", () => {
  assert.equal(isEditableTarget(null), false);
  assert.equal(isEditableTarget({ tagName: "DIV" }), false);
  assert.equal(isEditableTarget({ tagName: "TEXTAREA" }), true);
  assert.equal(isEditableTarget({ tagName: "TEXTAREA", disabled: true }), false);
  assert.equal(isEditableTarget({ tagName: "TEXTAREA", readOnly: true }), false);
  assert.equal(isEditableTarget({ tagName: "INPUT" }), true); // no type => text
  assert.equal(isEditableTarget({ tagName: "INPUT", type: "text" }), true);
  // number/email/date etc. can't setRangeText but are still real text-bearing fields
  assert.equal(isEditableTarget({ tagName: "INPUT", type: "number" }), true);
  assert.equal(isEditableTarget({ tagName: "INPUT", type: "checkbox" }), false);
  assert.equal(isEditableTarget({ tagName: "INPUT", type: "file" }), false);
  assert.equal(isEditableTarget({ tagName: "INPUT", type: "text", disabled: true }), false);
});

test("insertAtCaret splices at the caret when setRangeText is supported", () => {
  let value = "hello world";
  let dispatched = null;
  const el = {
    tagName: "INPUT",
    type: "text",
    get value() { return value; },
    selectionStart: 5,
    selectionEnd: 5,
    focus() {},
    setRangeText(text, start, end) { value = value.slice(0, start) + text + value.slice(end); },
    dispatchEvent(ev) { dispatched = ev; },
  };
  insertAtCaret(el, "!!!");
  assert.equal(value, "hello!!! world");
  assert.equal(dispatched.type, "input");
});

test("insertAtCaret replaces the selection, not the whole value", () => {
  let value = "hello world";
  const el = {
    tagName: "TEXTAREA",
    get value() { return value; },
    selectionStart: 0,
    selectionEnd: 5, // "hello" selected
    focus() {},
    setRangeText(text, start, end) { value = value.slice(0, start) + text + value.slice(end); },
    dispatchEvent() {},
  };
  insertAtCaret(el, "goodbye");
  assert.equal(value, "goodbye world");
});

test("insertAtCaret does nothing for empty input", () => {
  let called = false;
  const el = { tagName: "INPUT", type: "text", setRangeText() { called = true; } };
  insertAtCaret(el, "");
  insertAtCaret(null, "text");
  assert.equal(called, false);
});
