import test from "node:test";
import assert from "node:assert/strict";
import { participantState, participantOptions, selectedWorkspace } from "./peerParticipants.js";
const owner = { kind: "terminal", ownerId: "a", cli: "codex", workspaceId: "w", sessionKey: "native" };
const pref = { ...owner, enabled: true, phase: "connected", appliedConnection: "peer_a" };
const peer = { id: "peer_a", active: true };
test("activity, connection and historical proof stay independent", () => {
 for (const [o,p,c,live,identity,label,connection,kind,ready] of [
  [owner,null,null,null,undefined,"Off","","off",false],
  [owner,{...pref,workspaceId:"other"},null,"idle",undefined,"Off","","off",false],
  [{...owner,workspaceId:"other"},pref,{...peer,active:false},"idle","confirmed","Off","","off",false],
  [{...owner,sessionKey:""},pref,peer,"open","unobserved","No conversation","Not connected","action",false],
  [owner,pref,peer,null,undefined,"Stopped","Not connected","action",false],
  [owner,pref,peer,"needs-you","confirmed","Needs your input","Connected","connected",true],
  [owner,{...pref,phase:"waiting",problem:"Waiting for the current turn or approval."},peer,"working","confirmed","Working","Waiting for turn","waiting",false],
  [owner,pref,peer,"idle","confirmed","Idle","Connected","connected",true],
  [owner,pref,peer,"open","unobserved","First message needed","Waiting for identity","waiting",false],
  [owner,{...pref,phase:"waiting-conversation",problem:"Waiting for this conversation to identify itself."},peer,"open","unobserved","First message needed","Waiting for identity","waiting",false],
  [owner,{...pref,phase:"error",problem:"Install the adapter."},peer,"idle","confirmed","Idle","Connection failed","error",false],
  [owner,pref,{...peer,id:"new"},"idle","confirmed","Idle","Connecting","preparing",false],
 ]) {
   const got=participantState(o,p,c,live,[{phase:"passed",senderId:"peer_a"}],identity);
   assert.deepEqual([got.label,got.connection,got.kind,got.ready],[label,connection,kind,ready]);
 }
 const staleProof=participantState(owner,pref,peer,"open",[{phase:"passed",senderId:"peer_a"}],"unobserved");
 assert.equal(staleProof.verified,false);
});
test("enabled but unavailable owners stay selected across connection replacement",()=>{
 const other={...owner,ownerId:"b"};
 const options=participantOptions([owner,other],"terminal:b","terminal:a");
 assert.equal(options.sender,other);
 assert.equal(options.recipient,owner);
 assert.deepEqual(options.receivers,[owner]);
});
test("workspace context never defaults to an unrelated first owner",()=>{
 const data={owners:[owner],workspaces:[{id:"w"},{id:"other"}]};
 assert.equal(selectedWorkspace(data,""),"");
 assert.equal(selectedWorkspace(data,"terminal:a"),"w");
 assert.equal(selectedWorkspace(data,"workspace:other"),"other");
 assert.equal(selectedWorkspace(data,"agent:missing"),"");
});

test("missing Pi adapter always offers Packages, without parsing error prose",()=>{
 for (const kind of ["terminal","agent"]) {
  const state=participantState({...owner,kind,cli:"pi"},{...pref,kind,cli:"pi",phase:"adapter-missing",problem:"Localized error text"},peer,"idle",[],"confirmed");
  assert.equal(state.action,"packages");assert.equal(state.kind,"error");
 }
});

test("persistent recording failure has a repair action, never reconnecting",()=>{
 for (const phase of ["connected","waiting-conversation"]) {
  const got=participantState(owner,{...pref,phase},peer,"open",[{phase:"passed",senderId:peer.id}],"unobserved","restart-required");
  assert.deepEqual([got.label,got.connection,got.ready,got.verified,got.action],["State unavailable","Connection failed",false,false,"terminal-controls"]);
 }
 const recovered=participantState(owner,pref,peer,"idle",[],"confirmed");
 assert.equal(recovered.ready,true);
 const off=participantState(owner,{...pref,enabled:false},peer,"open",[],"unobserved","restart-required");
 assert.equal(off.kind,"off");
});
test("activation is offered only for an unidentified open terminal",()=>{
 const waiting={...pref,phase:"waiting-conversation"};
 assert.equal(participantState(owner,waiting,peer,"open",[],"unobserved").activation,true);
 assert.equal(participantState(owner,waiting,peer,"working",[],"unobserved").activation,false);
 assert.equal(participantState({...owner,kind:"agent"},waiting,peer,"open",[],"unobserved").activation,false);
 assert.equal(participantState(owner,pref,peer,"open",[],"confirmed").activation,false);
});
