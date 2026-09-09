#!/usr/bin/env node
// Browser regression matrix. Communication requests are synthetic; no peer is contacted.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve, join } from 'node:path';
const base = new URL(process.argv[2]);
assert.ok(['http:','https:'].includes(base.protocol) && ['localhost', '127.0.0.1'].includes(base.hostname));
const state = await (await fetch(new URL('/api/communication', base))).json();
assert.ok(state.owners.some(o => o.sessionKey.includes('/var/qa/peer-communication/project/') || o.sessionKey.includes('/var/qa/peer-launch/')), 'Refuse non-owned scratch fixture');
const out = resolve(process.argv[3] || 'var/screenshots/peer-browser'); mkdirSync(out, {recursive:true});
const results=[];
for (const app of ['desktop','mobile']) {
 const session=`peer-browser-${app}-${process.pid}`;
 const ab=(...args)=>execFileSync('agent-browser',['--session',session,...(base.protocol==='https:'?['--ignore-https-errors']:[]),...args],{encoding:'utf8',maxBuffer:4*1024*1024}).trim();
 const ev=code=>JSON.parse(ab('eval',code));
 const wait=code=>ab('wait','--fn',code);
 const button=name=>ab('find','role','button','click','--name',name,'--exact');
 const pause=ms=>new Promise(r=>setTimeout(r,ms));
 async function capture(name){await pause(550);const audit=ev('window.__picodeOverlayAudit()');assert.equal(audit.ok,true,`${app} ${name}`);assert.ok(ev('document.documentElement.scrollWidth <= innerWidth+1'));ab('screenshot',join(out,`${app}-${name}.png`));writeFileSync(join(out,`${app}-${name}.json`),JSON.stringify(audit));}
 try {
  ab('set','viewport',...(app==='desktop'?['1440','1000']:['390','844']));
  ab('open',new URL(`/${app}/${app==='mobile'?'?mobile=1':''}#/clis/messages`,base).href);
  wait('!!document.querySelector(".peer-picker select")');
  ev(`window.qaOriginalFetch=window.fetch;window.qa={fail:false,historyFail:false,owners:[],connections:[],messages:[],writes:[],delay:false};
    window.fetch=async(input,options={})=>{
      const path=new URL(String(input),location.origin).pathname, method=options.method||'GET',q=window.qa;
      const reply=(body,status=200)=>new Response(JSON.stringify(body),{status,headers:{'Content-Type':'application/json'}});
      if(!path.startsWith('/api/communication'))return window.qaOriginalFetch(input,options);
      if(method==='GET'&&path==='/api/communication')return q.fail?reply({error:'Unavailable'},503):reply({owners:q.owners,connections:q.connections,endpoint:'/mcp/communication',launchCLIs:q.auto?['pi','codex']:[],launches:q.auto?Object.fromEntries(q.connections.map(p=>[p.id,p.active])):{}});
      if(method==='GET'){const before=Number(new URL(String(input),location.origin).searchParams.get('before'))||Infinity;return q.historyFail?reply({error:'Unavailable'},503):reply({messages:q.messages.filter(m=>(path.includes(m.senderId)||path.includes(m.recipientId))&&m.seq<before).sort((a,b)=>b.seq-a.seq).slice(0,100)})}
      q.writes.push({path,method});
      if(method==='POST'&&q.missing)return reply({error:'Install the Pi MCP adapter before connecting.',code:'adapter_missing'},400);
      if(method==='DELETE'){q.connections=q.connections.map(p=>path.endsWith(p.id)?{...p,active:false,revokedAt:'2026-09-09T12:00:00Z'}:p);return reply({ok:true})}
      const body=JSON.parse(options.body),owner=q.owners.find(o=>o.ownerId===body.ownerId),id='fixture-connection-'+q.writes.length;
      const connection={...owner,id,active:true,createdAt:'2026-09-09T12:00:00Z',revokedAt:null};
      q.connections=[connection,...q.connections.map(p=>p.ownerId===owner.ownerId?{...p,active:false}:p)];
      if(q.delay)await new Promise(r=>{q.resolve=r});
      return reply(q.auto?{connection,automatic:true}:{connection,token:'pcc_browser_fixture_only',endpoint:'/mcp/communication'},201);
    };`);
  button('Refresh');wait('document.querySelector(".peer-body").textContent.includes("No agents or terminals yet.")');await capture('empty');
  ev(`qa.owners=[{kind:'agent',ownerId:'qa-agent',workspaceId:'qa-workspace',label:'Atlas',cli:'pi',sessionKey:''}]`);
  button('Refresh');wait('!!document.querySelector(".peer-picker select")');await capture('blocked');
  ev(`qa.owners[0].sessionKey='fixture-session';qa.owners.push({kind:'terminal',ownerId:'qa-terminal',workspaceId:'qa-workspace',label:'Review',cli:'codex',sessionKey:'fixture-codex'})`);
  button('Refresh');wait('document.querySelector(".peer-body").textContent.includes("Enable messages")');button('Enable messages');wait('!!document.querySelector(".peer-setup")');await capture('setup');
  ev(`qa.messages=[{seq:1,id:'fixture-message',senderId:qa.connections[0].id,recipientId:'fixture-other',body:'A durable message survives refresh.',createdAt:'2026-09-09T12:01:00Z',ackedAt:null}]`);
  button('Dismiss');button('Refresh');wait('document.querySelectorAll(".peer-history li").length===1');
  ev('qa.fail=true');button('Refresh');wait('!!document.querySelector(".peer-body [role=alert]")');assert.equal(ev('document.querySelectorAll(".peer-history li").length'),1);assert.ok(ev('[...document.querySelectorAll(".peer-connection button")].every(b=>b.disabled)'));await capture('error-retains-history');
  ev('qa.fail=false');button('Try again');wait('!document.querySelector(".peer-body [role=alert]")');
  ev('qa.historyFail=true');button('Refresh');wait('!!document.querySelector(".peer-body [role=alert]")');assert.equal(ev('document.querySelectorAll(".peer-history li").length'),1);
  ev('qa.historyFail=false');button('Try again');wait('!document.querySelector(".peer-body [role=alert]")');
  ev(`qa.messages=Array.from({length:101},(_,i)=>({...qa.messages[0],seq:i+1,id:'page-'+i}))`);button('Refresh');wait('document.querySelectorAll(".peer-history li").length===100');button('Older messages');wait('document.querySelectorAll(".peer-history li").length===101');assert.ok(ev('![...document.querySelectorAll(".peer-body button")].some(b=>b.textContent==="Older messages")'));ev('qa.messages=qa.messages.slice(0,1)');button('Refresh');wait('document.querySelectorAll(".peer-history li").length===1');
  button('Disable');wait('!!document.querySelector(":is([role=dialog],[role=alertdialog])")');await capture('confirm');button('Cancel');wait('!document.querySelector(":is([role=dialog],[role=alertdialog])")');
  // A delayed credential response from one owner must never appear in another.
  ev('qa.delay=true');button('Replace connection');wait('!!document.querySelector(":is([role=dialog],[role=alertdialog])")');await pause(550);button('Replace');wait('typeof qa.resolve==="function"');
  ev(`location.hash='#/clis/messages/terminal%3Aqa-terminal'`);wait('document.querySelector(".peer-picker select")?.value==="terminal:qa-terminal"');
  ev('qa.resolve();qa.delay=false');await pause(600);assert.equal(ev('!!document.querySelector(".peer-setup")'),false,'stale credential appeared');assert.equal(ev('document.querySelector(".peer-picker select").value'),'terminal:qa-terminal');
  button('Enable messages');wait('!!document.querySelector(".peer-setup")');button('Dismiss');button('Disable');wait('!!document.querySelector(":is([role=dialog],[role=alertdialog])")');await pause(550);ab('click',':is([role=dialog],[role=alertdialog]) button:last-child');
  wait('!document.querySelector(":is([role=dialog],[role=alertdialog])")');wait('document.querySelector(".peer-body").textContent.includes("Messages disabled")');
  assert.ok(ev('qa.writes.some(w=>w.method==="DELETE")'));
  ev('qa.auto=true');button('Refresh');wait('!document.querySelector(".peer-body button").disabled');button('Enable messages');wait('document.querySelector(".peer-body").textContent.includes("Configured · resume to connect")');assert.equal(ev('!!document.querySelector(".peer-setup")'),false);await capture('automatic-ready');ev("document.documentElement.dataset.theme='dark';document.documentElement.style.colorScheme='dark'");await capture('automatic-ready-dark');
  ev("qa.connections=[];qa.missing=true;qa.owners.find(o=>o.ownerId==='qa-terminal').cli='pi'");button('Refresh');wait('document.querySelector(".peer-body").textContent.includes("Messages disabled")');button('Enable messages');wait('document.querySelector(".peer-body").textContent.includes("Open Packages")');await capture('adapter-missing');
  results.push({app,pass:true});
 } finally {ab('close');}
}
writeFileSync(join(out,'result.json'),JSON.stringify(results,null,2));console.log(JSON.stringify(results));
