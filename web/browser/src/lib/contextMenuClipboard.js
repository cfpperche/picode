const NON_TEXT_INPUT_TYPES = new Set(["checkbox", "radio", "range", "color", "file", "button", "submit", "reset", "image", "hidden"]);
const RANGE_TEXT_INPUT_TYPES = new Set(["text", "search", "url", "tel", "password"]);

export function isEditableTarget(el) {
  if (!el) return false;
  if (el.tagName === "TEXTAREA") return !el.disabled && !el.readOnly;
  if (el.tagName === "INPUT") return !el.disabled && !el.readOnly && !NON_TEXT_INPUT_TYPES.has(el.type || "text");
  return false;
}

function canSetRangeText(el) {
  return el.tagName === "TEXTAREA" || RANGE_TEXT_INPUT_TYPES.has(el.type || "text");
}

// Splices at the caret when the element supports it (correct paste
// semantics); falls back to the native value setter for input types
// (number, email, date…) that throw on setRangeText/selectionStart.
export function insertAtCaret(el, text) {
  if (!el || !text) return;
  el.focus();
  if (canSetRangeText(el)) {
    const start = el.selectionStart ?? el.value.length;
    const end = el.selectionEnd ?? el.value.length;
    el.setRangeText(text, start, end, "end");
  } else {
    const proto = el.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(proto, "value").set.call(el, text);
  }
  el.dispatchEvent(new Event("input", { bubbles: true }));
}
