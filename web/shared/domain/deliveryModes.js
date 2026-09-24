// Attach delivery modes (ADR-0206). The server's GET /prompt names the
// modes a CLI has and its state; this decides what the composer offers.
//
// Idle: nothing to choose — every CLI sends a message at once, so the
// selector hides and Send is a prompt. Working: the modes that reach a
// busy CLI (steer, follow-up); Prompt is not among them because the door
// refuses it mid-turn. A CLI with none of them keeps the plain composer and
// the server's refusal names why.

const LABELS = Object.freeze({ prompt: "Prompt", steer: "Steer", follow_up: "Follow-up", interrupt: "Stop and send" });

// One line per choice, for people who never met the words.
const HINTS = Object.freeze({
  steer: "Reaches the agent in this turn",
  follow_up: "Sent when this turn ends",
  interrupt: "Stops the agent, then sends",
});

// Placeholder per mode: what Send will do, in the words a person uses.
const PLACEHOLDERS = Object.freeze({
  steer: "Steer the running turn",
  follow_up: "Queue for after this turn",
  interrupt: "Stop the agent and send this",
});

export function deliveryOptions(modes, state) {
  if (state !== "working") return [];
  return (Array.isArray(modes) ? modes : [])
    .filter((m) => m === "steer" || m === "follow_up" || m === "interrupt")
    .map((id) => ({ id, label: LABELS[id], hint: HINTS[id] }));
}

// The mode Send uses: the person's pick while it is still offered, else the
// first offered one, else a plain prompt. Stop and send is never the
// default: it discards the step in flight, so it is always a deliberate pick.
export function pickDelivery(options, current) {
  if (!options.length) return "prompt";
  if (options.some((o) => o.id === current)) return current;
  const safe = options.find((o) => o.id !== "interrupt");
  return (safe || options[0]).id;
}

// The Send button's words for a mode.
export function deliverySendLabel(delivery) {
  if (delivery === "follow_up") return "Queue";
  if (delivery === "interrupt") return "Stop and send";
  return "Send";
}

export function deliveryPlaceholder(delivery, fallback) {
  return PLACEHOLDERS[delivery] || fallback;
}

// The one receipt worth a toast: a send PiCode could not confirm. A prompt
// is confirmed by the input row emptying; a mid-turn send by the CLI's own
// queue render.
export function deliveryNotice(res, delivery) {
  if (!res || res.delivery !== "unconfirmed") return "";
  return delivery === "steer" || delivery === "follow_up" || delivery === "interrupt"
    ? "Sent, but PiCode could not confirm the CLI took it. Check the terminal."
    : "Sent, but PiCode could not confirm it left the composer. Check the terminal.";
}

// What the composer says while a send is in flight: Stop and send waits for
// the CLI to stop (up to a few seconds), so it names that wait.
export function deliveryBusyText(delivery) {
  return delivery === "interrupt" ? "Stopping the agent…" : "Sending…";
}
