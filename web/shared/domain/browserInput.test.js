import { test } from "node:test";
import assert from "node:assert/strict";
import { drawRect, keyboardMessages, mapPoint, modifiersMask, pointerMessage } from "./browserInput.js";

test("drawRect letterboxes the frame into the canvas", () => {
	assert.deepEqual(drawRect(1000, 500, 1280, 800), { x: 100, y: 0, w: 800, h: 500, scale: 0.625 });
	assert.equal(drawRect(0, 0, 10, 10), null);
});

test("mapPoint maps canvas coordinates into viewport pixels", () => {
	const canvas = { left: 0, top: 0 };
	const rect = drawRect(1000, 500, 1280, 800);
	const p = mapPoint(500, 250, canvas, rect, 1280, 800);
	// Center of the canvas maps to the center of the viewport.
	assert.equal(p.x, 640);
	assert.equal(p.y, 400);
	// Outside the drawn rect is not on the page.
	assert.equal(mapPoint(-10, 250, canvas, rect, 1280, 800), null);
	assert.equal(mapPoint(500, 600, canvas, rect, 1280, 800), null);
});

test("modifiersMask follows the engine's bitmask", () => {
	assert.equal(modifiersMask({}), 0);
	assert.equal(modifiersMask({ shiftKey: true }), 8);
	assert.equal(modifiersMask({ ctrlKey: true }), 2);
	assert.equal(modifiersMask({ altKey: true, metaKey: true }), 5);
});

test("keyboardMessages: printable keys are char, the rest are key bookends", () => {
	assert.deepEqual(keyboardMessages("keydown", { key: "a" }), [{ type: "input_keyboard", eventType: "char", text: "a" }]);
	assert.deepEqual(keyboardMessages("keydown", { key: "Enter", code: "Enter" }), [
		{ type: "input_keyboard", eventType: "keyDown", key: "Enter", code: "Enter", modifiers: 0 },
	]);
	assert.deepEqual(keyboardMessages("keyup", { key: "Enter", code: "Enter" }), [
		{ type: "input_keyboard", eventType: "keyUp", key: "Enter", code: "Enter", modifiers: 0 },
	]);
	// Ctrl+A is a key event with modifiers, not a char.
	assert.deepEqual(keyboardMessages("keydown", { key: "a", code: "KeyA", ctrlKey: true }), [
		{ type: "input_keyboard", eventType: "keyDown", key: "a", code: "KeyA", modifiers: 2 },
	]);
});

test("pointerMessage builds mouse and touch shapes", () => {
	assert.deepEqual(pointerMessage("mouse", "down", { x: 10, y: 20 }, 1, 1), {
		type: "input_mouse", eventType: "mousePressed", x: 10, y: 20, button: "left", clickCount: 1,
	});
	assert.deepEqual(pointerMessage("mouse", "move", { x: 1, y: 2 }), {
		type: "input_mouse", eventType: "mouseMoved", x: 1, y: 2,
	});
	assert.deepEqual(pointerMessage("touch", "down", { x: 5, y: 6 }, 3), {
		type: "input_touch", eventType: "touchStart", touchPoints: [{ x: 5, y: 6, id: 3 }],
	});
	assert.deepEqual(pointerMessage("touch", "up", { x: 5, y: 6 }, 3), {
		type: "input_touch", eventType: "touchEnd", touchPoints: [],
	});
	assert.equal(pointerMessage("mouse", "down", null), null);
});
