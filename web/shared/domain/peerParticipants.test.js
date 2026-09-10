import test from "node:test";
import assert from "node:assert/strict";
import { participantState, selectedWorkspace } from "./peerParticipants.js";
const owner = { kind: "terminal", ownerId: "a", cli: "codex", workspaceId: "w", sessionKey: "native" };
const pref = { ...owner, enabled: true, phase: "connected", appliedConnection: "peer_a" };
const peer = { id: "peer_a", active: true };
test("participation states distinguish setup from native proof", () => {
 for (const [o,p,c,live,checks,label] of [
  [owner,null,null,null,[],"Off"],
  [owner,{...pref,workspaceId:"other"},null,"open",[],"Off"],
  [{...owner,sessionKey:""},pref,peer,"open",[],"Needs a conversation"],
  [owner,pref,peer,null,[],"Stopped"],
  [owner,pref,peer,"needs-you",[],"Needs your input"],
  [owner,{...pref,phase:"preparing"},peer,"working",[],"Waiting to connect"],
  [owner,pref,peer,"open",[],"Connected · not tested"],
  [owner,pref,peer,"open",[{phase:"passed",senderId:"peer_a",recipientId:"peer_b"}],"Verified"],
  [owner,pref,{...peer,id:"new"},"open",[{phase:"passed",senderId:"peer_a"}],"Preparing…"],
 ]) assert.equal(participantState(o,p,c,live,checks).label,label);
});
test("workspace context never defaults to an unrelated first owner",()=>{
 const data={owners:[owner],workspaces:[{id:"w"},{id:"other"}]};
 assert.equal(selectedWorkspace(data,""),"");
 assert.equal(selectedWorkspace(data,"terminal:a"),"w");
 assert.equal(selectedWorkspace(data,"workspace:other"),"other");
 assert.equal(selectedWorkspace(data,"agent:missing"),"");
});
