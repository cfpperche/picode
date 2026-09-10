// Design mode (ADR-0114 phases): pure helpers turning a canvas pick into an
// agent prompt. The surface captures the click; the AGENT owns the browser
// semantics — it inspects the element with its own tools (ADR-0003).

// The prompt the surface sends for one picked element. `intent` is the
// user's description of the change; (x, y) are viewport CSS pixels.
export function buildPickPrompt(x, y, intent) {
	const want = (intent || "").trim();
	return [
		`Design pick: in the browser you have open, the element at viewport (${x}, ${y}).`,
		want ? `What I want changed: ${want}` : "Tell me what this element is, then wait for my change request.",
		"Use agent_browser to inspect it at those coordinates (elementFromPoint in the live page), read its HTML and computed styles, then apply the change to the source code. A screenshot of the page as picked is attached.",
	].join(" ");
}

// Whether a click is worth sending: needs a live frame, mapped point and
// (eventually) intent — an empty intent still asks "what is this?".
export function pickIsValid(point, hasFrame) {
	return Boolean(point && hasFrame);
}
