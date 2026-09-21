import { test } from "node:test";
import assert from "node:assert/strict";
import { observeDelivery } from "./delivery.js";
function env(){let tick,feed;const events={};return {doc:{hidden:false,addEventListener(n,f){events[n]=f;},removeEventListener(){}},win:{setInterval(f,ms){assert.equal(ms,15000);tick=f;return 1;},clearInterval(){},addEventListener(){},removeEventListener(){}},subscribe(f){feed=f;return()=>{};},tick(){tick();},feed(e){feed(e);}};}
const settle=()=>new Promise(r=>setImmediate(r));
test("hidden panes pause, refresh retains evidence, disposed reads cannot replace context",async()=>{
 const e=env();let states=[],calls=0,resolve;
 const read=async()=>{calls++;if(calls===1)return {observedAt:"now",changes:[1]};return new Promise(r=>resolve=r);};
 const sub=observeDelivery({...e,url:"/delivery?",read,onChange:s=>states.push(s)});await settle();
 e.doc.hidden=true;e.tick();assert.equal(calls,1);e.doc.hidden=false;sub.refresh();assert.deepEqual(states.at(-1).data.changes,[1]);sub.refresh();assert.equal(calls,2);
 sub.stop();const n=states.length;resolve({changes:[2]});await settle();assert.equal(states.length,n);
});
test("root mismatch stops background reads until explicit follow recreates observer",async()=>{
 const e=env();let state,calls=0;const sub=observeDelivery({...e,url:"/delivery?",read:async()=>{calls++;throw Object.assign(new Error(),{status:409});},onChange:s=>state=s});await settle();assert.equal(state.moved,true);e.tick();sub.refresh();assert.equal(calls,1);sub.stop();
});
test("first response pins root and target for subsequent reads",async()=>{
 const e=env();const urls=[];const sub=observeDelivery({...e,url:"/delivery?",read:async url=>{urls.push(url);return {root:"/original",target:"main"};},onChange(){}});await settle();sub.refresh();await settle();assert.match(urls[1],/root=%2Foriginal/);assert.match(urls[1],/target=main/);sub.stop();
});
