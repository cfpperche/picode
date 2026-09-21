import { test } from "node:test";
import assert from "node:assert/strict";
import { deliveryRows, deliveryReason, observationLabel, integrationExplanation, validationExplanation, environmentLane, attemptLine, publicationLabel, publicationReason } from "./delivery.js";
test("failed integrated change remains integrated and leads attention",()=>{
 const a={id:"a",title:"A",integration:"integrated",validation:"failed"},b={id:"b",title:"B",integration:"not-integrated",validation:"unknown"};
 assert.deepEqual(deliveryRows([b,a],"attention"),[a,b]);assert.match(deliveryReason(a),/after integration/);
});
test("stale source outranks a historical review request",()=>assert.match(deliveryReason({sourceStatus:"changed",review:"requested"}),/branch changed/));
test("a one-second observation does not read as a plural",()=>{
 assert.match(observationLabel({data:{observedAt:new Date(Date.now()-1000).toISOString()}},Date.now()),/Checked 1 second ago/);
 assert.match(observationLabel({data:{observedAt:new Date(Date.now()-4000).toISOString()}},Date.now()),/Checked 4 seconds ago/);
});
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
// D2 (ADR-0170): the Deployment lane's copy is the surface contract's own,
// and an attempt that never finished is unknown — never running, never failed.
test("an unfinished deployment attempt is unknown, not running and not failed",()=>{
 assert.match(attemptLine({outcome:"unknown"}).text,/Unknown outcome/);
 assert.doesNotMatch(attemptLine({outcome:"unknown"}).text,/running|Running/);
 assert.match(attemptLine({outcome:"failed",revisionBefore:"abc"}).text,/previous version is still responding/);
 assert.match(attemptLine({outcome:"passed",builtRevision:"0123456789"}).text,/Passed · 0123456/);
 assert.equal(attemptLine(null),null);
 // The outcome rides along: the surface tints a failure from it.
 assert.equal(attemptLine({outcome:"failed"}).outcome,"failed");
 assert.equal(attemptLine({outcome:"unknown"}).outcome,"unknown");
 assert.equal(attemptLine({outcome:"passed"}).outcome,"passed");
});
test("the lane names what it could not establish instead of guessing",()=>{
 const off=environmentLane(null);
 assert.equal(off.headline,"Deployment is not connected for this project.");
 assert.equal(off.action.kind,"setup");
 const unmapped=environmentLane({status:"unknown",reasonCode:"revision-unmapped",observedAt:new Date().toISOString()},"main");
 assert.match(unmapped.headline,/included changes are unconfirmed/);
 const conflict=environmentLane({status:"conflict",reasonCode:"binding-conflict",observedAt:new Date().toISOString()},"main");
 assert.match(conflict.headline,/claim this environment/);
 assert.equal(environmentLane({status:"known",observedAt:new Date().toISOString(),unpublished:{count:0,changes:[]}},"main").unpublished.count,0);
});
test("the lane reports the published set and the live environment separately",()=>{
 const now=new Date().toISOString();
 const env={status:"known",reasonCode:"",displayVersion:"0.4.0+abc1234",revision:"a".repeat(40),boot:"b",responding:true,artifact:"clean",observedAt:now,unpublished:{count:1,changes:["c"]},busy:{status:"known",owners:2},issues:[]};
 const lane=environmentLane(env,"a".repeat(40),Date.parse(now));
 assert.match(lane.facts[0],/Running 0.4.0\+abc1234 · Responding · checked 0 seconds ago/);
 assert.equal(lane.unpublished.text,"Integrated, not published: 1 change");
 assert.equal(lane.busy.text,"Agents are still working.");
 // No guard read and no safety claim.
 assert.equal(environmentLane({...env,busy:{status:"unknown",owners:0}},"x",Date.parse(now)).busy,null);
});
test("an unknown guard read never becomes an all-clear",()=>{
 const env={status:"known",observedAt:new Date().toISOString(),unpublished:{count:0,changes:[]},busy:{status:"unknown",owners:0}};
 assert.equal(environmentLane(env,"main").busy,null);
 assert.doesNotMatch(JSON.stringify(environmentLane(env,"main")),/safe|ready/i);
});
test("publication means membership of the running artifact",()=>{
 assert.equal(publicationLabel.published,"Published");
 assert.match(publicationReason({publication:"not-published"},{status:"known"}),/does not contain this revision/);
 assert.match(publicationReason({publication:"unknown"},{status:"unknown",reasonCode:"artifact-dirty"}),/unfinished changes/);
 assert.match(publicationReason({publication:"unknown"},null),/No environment is connected/);
});
