import test from "node:test";
import assert from "node:assert/strict";
import { deliveryOptions, pickDelivery, deliveryPlaceholder, deliveryNotice, deliveryInfo, deliverySendLabel, deliveryBusyText } from "./deliveryModes.js";

test("idle hides the selector; working offers the busy modes the CLI has", () => {
  assert.deepEqual(deliveryOptions(["prompt", "steer", "follow_up"], "idle"), []);
  assert.deepEqual(deliveryOptions(["prompt", "steer", "follow_up"], ""), []);
  assert.deepEqual(deliveryOptions(["prompt", "steer", "follow_up"], "needs-you"), []);
  assert.deepEqual(deliveryOptions(["prompt", "steer", "follow_up"], "working").map((o) => o.id), ["steer", "follow_up"]);
  assert.deepEqual(deliveryOptions(["prompt", "follow_up"], "working"), [{ id: "follow_up", label: "Follow-up", hint: "Sent when this turn ends" }]);
  assert.deepEqual(deliveryOptions(["prompt"], "working"), []);
  assert.deepEqual(deliveryOptions(undefined, "working"), []);
});

test("pickDelivery keeps the person's pick while it is offered", () => {
  const both = deliveryOptions(["prompt", "steer", "follow_up"], "working");
  assert.equal(pickDelivery(both, "follow_up"), "follow_up");
  assert.equal(pickDelivery(both, "prompt"), "steer");
  assert.equal(pickDelivery([], "steer"), "prompt");
});

test("placeholder and receipt copy", () => {
  assert.equal(deliveryPlaceholder("steer", "Message the terminal"), "Steer the running turn");
  assert.equal(deliveryPlaceholder("prompt", "Message the terminal"), "Message the terminal");
  assert.match(deliveryNotice({ delivery: "unconfirmed" }, "steer"), /CLI took it/);
  assert.match(deliveryNotice({ delivery: "unconfirmed" }, "prompt"), /left the composer/);
  assert.equal(deliveryNotice({ delivery: "queued" }, "steer"), "");
  assert.equal(deliveryNotice({ delivery: "verified" }, "prompt"), "");
  assert.equal(deliveryNotice({ delivery: "accepted" }, "follow_up"), "");
  assert.match(deliveryInfo({ delivery: "accepted", reason: "shows-at-turn-end" }), /when the current turn ends/);
  assert.equal(deliveryInfo({ delivery: "queued" }), "");
});

test("stop and send is offered while working but never the default", () => {
  const all = deliveryOptions(["prompt", "steer", "follow_up", "interrupt"], "working");
  assert.deepEqual(all.map((o) => o.id), ["steer", "follow_up", "interrupt"]);
  assert.equal(pickDelivery(all, ""), "steer");
  assert.equal(pickDelivery(all, "interrupt"), "interrupt");
  const agy = deliveryOptions(["prompt", "follow_up", "interrupt"], "working");
  assert.equal(pickDelivery(agy, ""), "follow_up");
  assert.equal(deliverySendLabel("interrupt"), "Stop and send");
  assert.equal(deliverySendLabel("follow_up"), "Queue");
  assert.equal(deliveryPlaceholder("interrupt", "x"), "Stop the agent and send this");
});

test("busy copy names the stop wait", () => {
  assert.equal(deliveryBusyText("interrupt"), "Stopping the agent…");
  assert.equal(deliveryBusyText("steer"), "Sending…");
});
