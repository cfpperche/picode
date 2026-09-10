// Browser surface input mapping (ADR-0115): pure helpers that turn DOM
// pointer/keyboard events into the engine's stream input messages. Kept
// framework-free and unit-tested; BrowserSurface only wires listeners.

// The frame is the viewport, letterboxed into the canvas. Map a canvas-local
// coordinate into viewport CSS pixels using the latest frame metadata.
export function mapPoint(clientX, clientY, canvasRect, drawRect, deviceWidth, deviceHeight) {
	if (!drawRect || !deviceWidth || !deviceHeight || !canvasRect) return null;
	const x = (clientX - canvasRect.left - drawRect.x) / drawRect.w;
	const y = (clientY - canvasRect.top - drawRect.y) / drawRect.h;
	if (x < 0 || x > 1 || y < 0 || y > 1) return null;
	return { x: Math.round(x * deviceWidth), y: Math.round(y * deviceHeight) };
}

// The letterboxed drawn rect for the current canvas size and frame aspect,
// in canvas-local CSS pixels (mirrors BrowserSurface's drawFrame math).
export function drawRect(canvasW, canvasH, frameW, frameH) {
	if (!canvasW || !canvasH || !frameW || !frameH) return null;
	const scale = Math.min(canvasW / frameW, canvasH / frameH);
	const w = frameW * scale;
	const h = frameH * scale;
	return { x: (canvasW - w) / 2, y: (canvasH - h) / 2, w, h, scale };
}

// Modifier bitmask the engine expects: 1=Alt, 2=Ctrl, 4=Meta, 8=Shift.
export function modifiersMask({ altKey, ctrlKey, metaKey, shiftKey } = {}) {
	return (altKey ? 1 : 0) | (ctrlKey ? 2 : 0) | (metaKey ? 4 : 0) | (shiftKey ? 8 : 0);
}

// Keyboard events for one DOM keydown/keyup: printable keys become a single
// `char` message; everything else becomes keyDown/keyUp bookends.
export function keyboardMessages(type, event) {
	const key = event.key || "";
	if (type === "keydown" && key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
		return [{ type: "input_keyboard", eventType: "char", text: key }];
	}
	const base = { type: "input_keyboard", eventType: type === "keydown" ? "keyDown" : "keyUp", key, code: event.code || "", modifiers: modifiersMask(event) };
	if (type === "keydown") return [base];
	return [base];
}

// Pointer events → mouse or touch messages. `down`/`move`/`up` carry the
// mapped viewport point; touch keeps a stable touch id per pointer.
export function pointerMessage(pointerType, phase, point, pointerId = 0, clickCount = 0) {
	if (!point) return null;
	if (pointerType === "touch") {
		const eventType = phase === "down" ? "touchStart" : phase === "move" ? "touchMove" : "touchEnd";
		const touchPoints = phase === "up" ? [] : [{ x: point.x, y: point.y, id: pointerId }];
		return { type: "input_touch", eventType, touchPoints };
	}
	const eventType = phase === "down" ? "mousePressed" : phase === "move" ? "mouseMoved" : "mouseReleased";
	const msg = { type: "input_mouse", eventType, x: point.x, y: point.y };
	if (phase !== "move") {
		msg.button = "left";
		msg.clickCount = clickCount;
	}
	return msg;
}
