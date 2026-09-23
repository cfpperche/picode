import { test } from "node:test";
import assert from "node:assert/strict";
import { deliveryRows, deliveryReason, observationLabel, integrationExplanation, validationExplanation, missionsFor, missionLinks } from "./delivery.js";
test("failed integrated change remains integrated and leads attention",()=>{
 const a={id:"a",title:"A",integration:"integrated",validation:"failed"},b={id:"b",title:"B",integration:"not-integrated",validation:"unknown"};
 assert.deepEqual(deliveryRows([b,a],"attention"),[a,b]);assert.match(deliveryReason(a),/after integration/);
});
test("stale source outranks a historical review request",()=>assert.match(deliveryReason({sourceStatus:"changed",review:"requested"}),/branch changed/));
test("refresh errors and stale observations never look current",()=>{
 assert.match(observationLabel({error:true,data:{}}),/last check/);
 assert.match(observationLabel({data:{observedAt:"2000-01-01"}},Date.now()),/out of date/);
});
test("summary and detail use the same plain-language state meanings",()=>{
 assert.match(deliveryReason({integration:"integrated"}),/Included in the target/);
 assert.match(integrationExplanation("integrated"),/Included/);
 assert.match(validationExplanation("passed"),/full-project/);
 assert.match(validationExplanation("unknown"),/No check evidence/);
});

// The Delivery surface says which objective a change serves, from the mission
// that cites it as evidence — the only direction the link exists (ADR-0199).
test("the mission link a delivery row shows", () => {
  const read = { missions: { delivery_abc: [{ id: "mission_1", title: "Ship the queue", state: "in-review" }] } };
  assert.deepEqual(missionsFor(read, "delivery_abc"), [{ id: "mission_1", title: "Ship the queue", state: "in-review" }]);
  assert.deepEqual(missionsFor(read, "delivery_other"), []);
  assert.deepEqual(missionsFor(null, "delivery_abc"), []);
  assert.deepEqual(missionLinks(missionsFor(read, "delivery_abc")), [
    { id: "mission_1", title: "Ship the queue", label: "Ready for review", href: "#/mission/mission_1" },
  ]);
  assert.deepEqual(missionLinks([{ id: "mission_2" }]), [{ id: "mission_2", title: "mission_2", label: "", href: "#/mission/mission_2" }]);
  assert.deepEqual(missionLinks(), []);
});
