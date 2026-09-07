// Terminal and Changes browser acceptance on a disposable loopback fixture.
// Creates and removes one dedicated shell; never starts a model CLI.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
const base = process.argv[2] || 'http://127.0.0.1:18831';
const url = new URL(base);
assert.ok(url.protocol === 'http:' && ['localhost', '127.0.0.1'].includes(url.hostname));
const out = resolve(process.argv[3] || 'var/screenshots/mobile-v2/tools-integrated');
mkdirSync(out, { recursive: true });
const browserSession = 'mobile-v2-tools-' + process.pid;
const ab = (...args) => execFileSync('agent-browser', ['--session',browserSession, ...args], { encoding:'utf8', maxBuffer:4000000 }).trim();
const ev = code => JSON.parse(ab('eval',code));
const wait = code => ab('wait','--fn',code);
const click = name => ab('find','role','button','click','--name',name,'--exact');
const pause = ms => new Promise(r => setTimeout(r,ms));
const shot = async name => { await pause(ev('Boolean(document.querySelector("[role=dialog]"))') ? 850 : 180); ab('screenshot',resolve(out,name+'.png')); };
const nav = hash => ev(`location.hash=${JSON.stringify(hash)}`);
const fleet = await (await fetch(base+'/api/workspaces')).json();
const ws = fleet.find(w=>w.name==='picode');
const fresh = fleet.find(w=>w.name==='fresh');
assert.match(ws.path,/^\/tmp\/picode-docs-fixture-.*\/work\/picode$/);
const created = await fetch(base+'/api/terminals', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:'Mobile v2 QA shell',cwd:ws.path,workspaceId:ws.id})});
assert.ok(created.ok);
const term = await created.json();
assert.ok(term.id);
const route = '#/changes/w/'+ws.id;
const result = { fixture:base, decisions:[], screenshots:[], geometry:[] };
const capture = async name => { await shot(name); result.screenshots.push(name+'.png'); const g=ev('({viewport:innerWidth,width:document.documentElement.scrollWidth,audit:window.__picodeOverlayAudit()})'); assert.equal(g.width,g.viewport,name+' page overflow'); assert.equal(g.audit.ok,true,name+' overlays'); result.geometry.push({name,...g}); };
try {
ab('set','viewport','320','740');
ab('open',base+'/mobile/?qa='+Date.now()+'#/work');
ab('wait','.m-screen');
ev(`(()=>{const real=window.fetch.bind(window);window.qa={status:'',patch:'',attach:'',mockCli:false,calls:{status:0,patch:0,attach:0}};window.fetch=async function(url,opt){const s=String(url);let lane=s.includes('/gitstatus')?'status':s.includes('/gitdiff?')?'patch':s.endsWith('/open')?'attach':'';if(lane){qa.calls[lane]++;if(qa[lane]==='error')return new Response(JSON.stringify({error:'Connection unavailable. Try again.'}),{status:503});if(qa[lane]==='delay')await new Promise(r=>setTimeout(r,2200));if(lane==='status'&&qa.status==='empty')return new Response(JSON.stringify({git:true,changes:[]}));if(lane==='status'&&qa.status==='long')return new Response(JSON.stringify({git:true,changes:[{path:'web/mobile/src/components/deeply-nested-workspace-tools/WorkspaceTerminalLongName.jsx',kind:'modified'}]}));if(lane==='patch'&&qa.status==='long')return new Response(JSON.stringify({patch:'diff --git a/WorkspaceTerminalLongName.jsx b/WorkspaceTerminalLongName.jsx\\n@@ -1,1 +1,1 @@\\n-const oldName = true;\\n+const workspaceName = \\"A long line that must remain horizontally scrollable inside the patch instead of widening the page\\";'}));}const res=await real(url,opt);if(s==='/api/terminals'&&qa.mockCli){const data=await res.json();data.terminals=data.terminals.map(t=>t.id===${JSON.stringify(term.id)}?{...t,name:'Workspace release terminal',launchCli:'pi',running:true}:t);return new Response(JSON.stringify(data),{status:res.status});}return res;};return true})()`);

ev("qa.status='error'");nav(route);ab('wait','[role="alert"]');await capture('changes-status-error-320');
ev("qa.status='delay'");click('Retry');ab('wait','[aria-busy="true"]');await capture('changes-loading-320');ab('wait','.gg-file-head');
assert.equal(ev('document.querySelectorAll(".gg-file").length'),4);result.decisions.push('Initial status error -> Retry -> loading skeleton -> real file list');
ev("qa.status='';qa.patch='error'");ab('click','.gg-file-head','--first');ab('wait','[role="alert"]');await capture('changes-patch-error-320');
const patchCount=ev('qa.calls.patch');ev("qa.patch='delay'");click('Retry');ab('wait','[aria-label="Loading patch"]');await capture('changes-patch-loading-320');ab('wait','.diff');assert.equal(ev('qa.calls.patch'),patchCount+1);result.decisions.push('Failed patch -> explicit Retry -> loading skeleton -> patch');
ev("qa.patch='';qa.status='error'");click('Refresh');ab('wait','[role="alert"]');assert.equal(ev('document.querySelectorAll(".gg-file").length'),4);assert.equal(ev('Boolean(document.querySelector(".diff"))'),true);await capture('changes-retained-error-320');result.decisions.push('Refresh error retains loaded list and expanded patch');
ev("qa.status=''");click('Retry');wait('!document.querySelector("[role=alert]") && document.querySelector(".gg-detail").getAttribute("aria-busy")==="false" && document.querySelector(".gg-file-head").getAttribute("aria-expanded")==="false"');
ev("qa.patch='error'");ab('click','.gg-file-head','--first');ab('wait','[role="alert"]');ab('click','.gg-file-head','--first');ev("qa.patch=''");ab('click','.gg-file-head','--first');ab('wait','.diff');result.decisions.push('Failed patch reopens with a fresh request');
nav('#/changes/w/'+fresh.id);wait('document.querySelector(".gg-detail")?.innerText.includes("This folder is not a Git repository.")');await capture('changes-nongit-320');assert.equal(ev('document.querySelectorAll(".gg-file").length'),0);result.decisions.push('Owner switch clears prior files; non-Git folder has Back action');
ev("qa.status='empty'");nav(route);wait('document.querySelector(".gg-detail")?.innerText.includes("No uncommitted changes.")');await capture('changes-empty-320');result.decisions.push('Clean working tree shows empty state and Back action');
ev("qa.status='long'");click('Refresh');ab('wait','.gg-file-head');await capture('changes-long-path-320');ab('click','.gg-file-head','--first');ab('wait','.diff');await capture('changes-long-patch-320');assert.ok(ev('document.querySelector(".diff").scrollWidth>document.querySelector(".diff").clientWidth'));result.decisions.push('Long filename/directory wraps; code scroll belongs to patch only');
ev("qa.status='';qa.attach='error'");nav('#/term/'+term.id);ab('wait','[role="alert"]');await capture('terminal-attach-error-320');
ev("qa.attach='delay'");click('Retry');ab('wait','[aria-label="Attaching terminal"]');await capture('terminal-loading-320');ab('wait','.xterm-screen');ev("qa.attach=''");assert.equal(ev('Boolean(document.querySelector(".m-keybar"))'),false);await capture('terminal-open-320');result.decisions.push('Terminal attach error -> Retry -> skeleton -> real xterm; no unsolicited key bar');
click('Terminal actions');ab('wait','[role="dialog"]');await capture('terminal-actions-320');ab('click','.m-term-actions .m-tool-action:nth-of-type(2)');ab('wait','.m-git-file-row');result.decisions.push('Terminal actions sheet opens Git in same terminal context');
nav('#/term/'+term.id);ab('wait','.xterm-screen');
ev(`(()=>{const match=window.matchMedia.bind(window);window.matchMedia=q=>q==='(hover: hover) and (pointer: fine)'?{matches:false}:match(q);return true})()`);
click('Show keyboard');ab('wait','.m-keybar');await capture('terminal-keys-320');assert.equal(ev('document.activeElement.classList.contains("xterm-helper-textarea")'),true);ab('click','.m-head [aria-label="Hide keyboard"]');wait('!document.querySelector(".m-keybar")');result.decisions.push('User keyboard tap focuses xterm and reveals keys; Hide removes accessory');
ev('qa.mockCli=true;window.dispatchEvent(new Event("focus"));true');ab('wait','[aria-label="Attach"]');await capture('terminal-cli-header-320');assert.ok(ev('document.querySelector(".m-head-title").getBoundingClientRect().width')>=120);click('Attach');ab('wait','[role="dialog"]');await capture('terminal-attach-sheet-320');ab('press','Escape');wait('!document.querySelector(".dlg-overlay")');
click('Terminal actions');ab('wait','[role="dialog"]');await pause(850);click('Remove terminal');wait('document.body.innerText.includes("Remove terminal?")');await capture('terminal-remove-confirm-320');click('Cancel');wait('!document.querySelector(".dlg-overlay")');result.decisions.push('CLI attachment remains accessible; Remove still opens existing confirmation and Cancel keeps terminal');
for(const width of [360,390,430,844]){ab('set','viewport',String(width),width===844?'390':'844');nav(route);ab('wait','.gg-file-head');await capture('changes-'+width);nav('#/term/'+term.id);ab('wait','.xterm-screen');await capture('terminal-'+width);}
ab('set','viewport','390','844');ab('open',base+'/mobile/?theme=dark#/changes/w/'+ws.id);ab('wait','.gg-file-head');ab('click','.gg-file-head','--first');ab('wait','.diff');await capture('changes-dark-390');
writeFileSync(resolve(out,'result.json'),JSON.stringify({ok:true,...result},null,2)+'\n');
console.log('PASS: '+result.decisions.length+' decision rows; '+result.screenshots.length+' screenshots; overlay audit and page width checks');

} finally {
  try { ab('close'); } finally {
    const removed = await fetch(base+'/api/terminals/'+encodeURIComponent(term.id), {method:'DELETE'});
    assert.ok(removed.ok, 'Remove only the terminal created by this QA run');
  }
}
