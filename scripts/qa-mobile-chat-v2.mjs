#!/usr/bin/env node
// Browser regressions against a disposable docs fixture. Every agent mutation
// and agent WebSocket is intercepted: this script never starts a model turn.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const base = new URL(process.argv[2] || "http://localhost:18768");
assert.ok(["localhost", "127.0.0.1"].includes(base.hostname) && base.protocol === "http:", "Use a loopback HTTP fixture");
const out = resolve(process.argv[3] || "var/screenshots/mobile-v2-chat");
mkdirSync(out, { recursive: true });
const session = "mobile-chat-v2-" + process.pid;
const browser = (...args) => execFileSync("agent-browser", ["--session", session, ...args], { encoding: "utf8", maxBuffer: 4 * 1024 * 1024 }).trim();
const evaluate = code => JSON.parse(browser("eval", code));
const settle = () => new Promise(done => setTimeout(done, 650));
const click = name => browser("find", "role", "button", "click", "--name", name, "--exact");
const fleet = await (await fetch(new URL("/api/workspaces", base))).json();
const workspace = fleet.find(w => w.name === "picode" && /[\\/]picode-docs-fixture-[^\\/]+[\\/]work[\\/]picode$/.test(w.path));
assert.ok(workspace, "Refuse to use a non-synthetic workspace");
const [agent, other] = ["Atlas", "Borealis"].map(name => workspace.agents.find(a => a.name === name));
assert.ok(agent && other, "Need both synthetic agents");
assert.equal(agent.mode, "stopped");
assert.equal(other.mode, "stopped");
const shot = async name => {
  await settle();
  assert.equal(evaluate("window.__picodeOverlayAudit().ok"), true, name + " overlay geometry");
  assert.equal(evaluate("document.documentElement.scrollWidth <= innerWidth"), true, name + " page overflow");
  browser("screenshot", resolve(out, name + ".png"));
};
const draftText = () => evaluate('document.querySelector("#task-input").value');
const pictures = () => evaluate('document.querySelectorAll(".composer-pics img").length');
const navigate = async id => {
  evaluate(`(()=>{location.hash=${JSON.stringify("#/agent/" + id)};return true})()`);
  await settle();
  browser("wait", "#task-input");
};
const pasteImage = () => evaluate(`(()=>{const d=new DataTransfer();const bytes=Uint8Array.from(atob("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jp1sAAAAASUVORK5CYII="),c=>c.charCodeAt(0));d.items.add(new File([bytes],"qa.png",{type:"image/png"}));document.querySelector("#task-input").dispatchEvent(new ClipboardEvent("paste",{clipboardData:d,bubbles:true,cancelable:true}));return true})()`);
const emit = event => evaluate(`(()=>{window.qaSocket.onmessage({data:JSON.stringify({event:${JSON.stringify(event)}})});return true})()`);

try {
  browser("set", "viewport", "390", "844");
  browser("open", new URL("/mobile/?theme=dark#/agent/" + agent.id, base).href);
  browser("wait", "#task-input");
  const measurements = evaluate('({composer:document.querySelector(".composer-wrap").getBoundingClientRect().height,header:document.querySelector(".m-head").getBoundingClientRect().height})');
  assert.ok(measurements.composer <= 120, "Empty composer <=120px");
  assert.ok(measurements.header <= 56, "Header <=56px");
  await shot("empty-dark");
  browser("fill", "#task-input", "Keep this mobile draft");
  browser("press", "Enter");
  assert.equal(draftText(), "Keep this mobile draft\n", "Mobile Enter keeps a newline");
  pasteImage();
  browser("wait", ".composer-pics");
  click("Message options");
  browser("wait", "#task-kind");
  browser("select", "#task-kind", "steer");
  await shot("options-dark");
  click("Done");
  await settle();
  evaluate('(()=>{location.hash="#/work";return true})()');
  await settle();
  await navigate(other.id);
  assert.equal(draftText(), "");
  assert.equal(pictures(), 0);
  browser("fill", "#task-input", "Borealis has its own draft");
  await navigate(agent.id);
  assert.equal(draftText(), "Keep this mobile draft\n");
  assert.equal(pictures(), 1);
  click("Message options");
  browser("wait", "#task-kind");
  assert.equal(evaluate('document.querySelector("#task-kind").value'), "steer");
  click("Done");
  await settle();
  click("Settings");
  browser("wait", "#ag-set-thinking");
  await shot("settings-dark");
  click("Done");
  await settle();
  assert.equal(draftText(), "Keep this mobile draft\n");
  assert.equal(pictures(), 1);
  await shot("retained-draft");

  evaluate(`(()=>{
    window.qaRequests=[]; window.qaMode="fail";
    const realFetch=window.fetch, RealSocket=window.WebSocket;
    window.fetch=(url,options={})=>{
      const path=new URL(url,location.href).pathname;
      if(options.method && options.method!=="GET" && path.startsWith("/api/agents/")) {
        const row={path,body:options.body?JSON.parse(options.body):null};window.qaRequests.push(row);
        if(["/tasks","/prompt","/bash"].some(suffix=>path.endsWith(suffix))) {
          if(window.qaMode==="pending") return new Promise(done=>{window.qaFinish=()=>done(new Response("{}",{status:200}));});
          return Promise.resolve(window.qaMode==="fail"?new Response(JSON.stringify({error:"QA: connection unavailable"}),{status:503}):new Response("{}",{status:200}));
        }
        return Promise.resolve(new Response("{}",{status:200}));
      }
      return realFetch(url,options);
    };
    window.WebSocket=function(url,...args){
      if(!String(url).includes("/ws/agent?")) return new RealSocket(url,...args);
      const socket={readyState:1,close(){this.readyState=3;this.onclose?.();}};
      window.qaSocket=socket;return socket;
    };
    window.WebSocket.OPEN=1; return true;
  })()`);
  click("Send");
  browser("wait", ".m-composer-error");
  assert.equal(draftText(), "Keep this mobile draft\n");
  assert.equal(pictures(), 1);
  assert.equal(evaluate('Array.from(document.querySelectorAll(".work-head")).some(node=>node.textContent.includes("Queued"))'), false, "Failed delivery is never shown as queued");
  await shot("send-error");
  evaluate('(()=>{window.qaMode="accepted";return true})()');
  click("Retry");
  browser("wait", "--fn", '!document.querySelector("#task-input").value');
  assert.equal(pictures(), 0);
  const request = evaluate('window.qaRequests.filter(x=>x.path.endsWith("/tasks")||x.path.endsWith("/prompt")).at(-1)');
  assert.equal(request.body.kind, "steer");
  assert.equal(request.body.images.length, 1);

  emit({ type: "snapshot", streaming: false, waiting: false });
  browser("fill", "#task-input", "Pending message");
  pasteImage();
  browser("wait", ".composer-pics");
  evaluate('(()=>{window.qaMode="pending";return true})()');
  click("Send");
  browser("wait", "--fn", '!!window.qaFinish');
  assert.equal(evaluate('document.querySelector("#task-send").disabled'), true);
  browser("fill", "#task-input", "New edits during the request");
  evaluate('(()=>{window.qaOldSocket=window.qaSocket;return true})()');
  await navigate(other.id);
  assert.equal(draftText(), "Borealis has its own draft");
  evaluate('(()=>{window.qaOldSocket.onmessage({data:JSON.stringify({event:{type:"message_update",assistantMessageEvent:{type:"text_delta",delta:"STALE_AGENT_REPLY"}}})});return true})()');
  assert.equal(evaluate('document.querySelector("#conversation").textContent.includes("STALE_AGENT_REPLY")'), false, "A closed agent socket cannot contaminate another conversation");
  await navigate(agent.id);
  assert.equal(draftText(), "New edits during the request");
  assert.equal(pictures(), 1);
  evaluate('(()=>{window.qaFinish();window.qaMode="accepted";return true})()');
  browser("wait", "--fn", '!document.querySelector("#task-send").disabled');
  assert.equal(draftText(), "New edits during the request");
  assert.equal(pictures(), 1);
  await shot("pending-edit-retained");
  click("Send");
  browser("wait", "--fn", '!document.querySelector("#task-input").value');

  emit({ type: "snapshot", streaming: false, waiting: true, dialog: { id: "qa-question", method: "confirm", title: "Continue this change?", message: "Review the pending change." } });
  browser("wait", "#task-abort");
  await shot("waiting-with-stop");
  click("Message options");
  browser("wait", "#task-kind");
  browser("select", "#task-kind", "prompt");
  click("Done");
  await settle();
  browser("fill", "#task-input", "Follow up after the question");
  evaluate('(()=>{window.qaMode="fail";return true})()');
  click("Send");
  browser("wait", ".m-composer-error");
  assert.equal(evaluate('!!document.querySelector("#task-abort")'), true, "Stop remains available after a failed follow-up");
  assert.equal(draftText(), "Follow up after the question");
  await shot("waiting-send-error");
  evaluate('(()=>{window.qaMode="accepted";return true})()');
  click("Retry");
  browser("wait", "--fn", '!document.querySelector("#task-input").value');
  assert.equal(evaluate('window.qaRequests.filter(x=>x.path.endsWith("/prompt")).at(-1).body.kind'), "follow_up");
  click("Stop response");
  assert.ok(evaluate('window.qaRequests.some(x=>x.path.endsWith("/abort"))'));

  for (const [width,height] of [[320,844],[360,844],[430,844],[844,390]]) {
    browser("set", "viewport", String(width), String(height));
    click("Message options");
    browser("wait", ".m-composer-options");
    await shot(`options-${width}x${height}`);
    click("Done");
    await settle();
  }
  browser("set", "viewport", "320", "844");
  evaluate('(()=>{window.SpeechRecognition=class{start(){}abort(){}};navigator.mediaDevices.getUserMedia=async()=>new MediaStream();localStorage.setItem("picode-stt-hint","1");return true})()');
  click("Message options");
  browser("wait", ".m-composer-options");
  await settle();
  click("Dictation");
  browser("wait", ".dictate-bar");
  await shot("dictation-320");
  click("Cancel dictation");
  click("Message options");
  browser("wait", ".m-composer-options");
  await settle();
  click("Voice mode");
  browser("wait", ".composer-voice-body");
  await shot("voice-320");
  click("Back to text");
  browser("set", "viewport", "390", "844");
  writeFileSync(resolve(out,"result.json"),JSON.stringify({ok:true,measurements,retainedDraft:true,agentIsolation:true,images:true,newline:true,failedSend:true,acceptedSend:true,pendingEdits:true,waitingStop:true,noModelTurns:true},null,2)+"\n");
  console.log("Mobile v2 conversation browser regressions: PASS", JSON.stringify(measurements));
} finally {
  browser("close");
}
