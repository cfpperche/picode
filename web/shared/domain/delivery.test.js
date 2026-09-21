import { test } from "node:test";
import assert from "node:assert/strict";
import { deliveryRows, deliveryReason, observationLabel, integrationExplanation, validationExplanation } from "./delivery.js";
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
