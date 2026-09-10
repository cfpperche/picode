import { test } from "node:test";
import assert from "node:assert/strict";
import { buildPickPrompt, pickIsValid } from "./browserDesign.js";

test("buildPickPrompt carries coordinates, intent and the inspection instruction", () => {
	const prompt = buildPickPrompt(640, 400, "make the button the brand color");
	assert.match(prompt, /viewport \(640, 400\)/);
	assert.match(prompt, /What I want changed: make the button the brand color/);
	assert.match(prompt, /elementFromPoint/);
	assert.match(prompt, /screenshot/);
});

test("buildPickPrompt without intent asks what the element is", () => {
	const prompt = buildPickPrompt(10, 20, "   ");
	assert.match(prompt, /Tell me what this element is/);
	assert.doesNotMatch(prompt, /What I want changed/);
});

test("pickIsValid needs a point and a frame", () => {
	assert.equal(pickIsValid(null, true), false);
	assert.equal(pickIsValid({ x: 1, y: 2 }, false), false);
	assert.equal(pickIsValid({ x: 1, y: 2 }, true), true);
});
