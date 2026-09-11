package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The receiver leaves a file that is addressed to another process in the
// same terminal (pid mismatch) or that it cannot answer for (no session of
// its own), so the addressee — the only process holding the item's session —
// still finds it. A file addressed here follows the ordinary rules.
func TestNativePiReceiverSkipsFilesNotAddressedToIt(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node unavailable")
	}
	dir := t.TempDir()
	module := filepath.Join(dir, "receiver.mjs")
	os.WriteFile(module, []byte(piInboxReplyExtensionTS), 0600)
	harness := `import { createServer } from 'node:http';
import { writeFileSync,mkdirSync,existsSync,unlinkSync } from 'node:fs';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
const [dir]=process.argv.slice(2), hooks={};let acks=[];let sends=0;
process.env.PICODE_DATA=dir;process.env.PICODE_AGENT_ID='fixture';delete process.env.PICODE_TERM_ID;
const server=createServer((req,res)=>{let data='';req.on('data',v=>data+=v);req.on('end',()=>{res.writeHead(204);res.end();if(req.url.endsWith('tui-ack'))acks.push(JSON.parse(data))})});
await new Promise(r=>server.listen(0,'127.0.0.1',r));
writeFileSync(join(dir,'server.json'),JSON.stringify({url:'http://127.0.0.1:'+server.address().port}));
const {default:register}=await import(pathToFileURL(join(dir,'receiver.mjs')));
register({on:(name,fn)=>hooks[name]=fn,sendUserMessage:async()=>{sends++}});
const inbox=join(dir,'tui-inbox','fixture');mkdirSync(inbox,{recursive:true});
const write=(name,doc)=>writeFileSync(join(inbox,name+'.json'),JSON.stringify({createdAt:new Date().toISOString(),...doc}));
const sleep=(ms)=>new Promise(r=>setTimeout(r,ms));
let session='original';
const ctx=()=>({sessionManager:{getSessionFile:()=>session},isIdle:()=>true,hasPendingMessages:()=>false,hasUI:false});
await hooks.session_start({},ctx());
const settle=async(name,ms)=>{await sleep(ms);const a=acks.splice(0);return {a,sends:(()=>{const v=sends;sends=0;return v})(),left:existsSync(join(inbox,name+'.json'))}};
// A file addressed to another process stays for its addressee: no ack, no send.
write('foreign',{nonce:'f',pid:process.pid+12345,sessionPath:'original',payload:'p'});
{
	const r=await settle('foreign',1600);
	if(r.a.length||r.sends||!r.left){console.error({case:'foreign',...r});process.exit(1)}
}
// A receiver with no session of its own cannot answer for any file: it also
// leaves it (the daemon's ack wait cleans up and reopens the item).
session=undefined;
await hooks.session_start({},ctx());
write('nosession',{nonce:'n',sessionPath:'original',payload:'p'});
{
	const r=await settle('nosession',1600);
	if(r.a.filter(x=>x.nonce==='n').length||r.sends||!r.left){console.error({case:'nosession',...r});process.exit(1)}
}
// The daemon sweeps what nobody took (its ack deadline); cases below start clean.
unlinkSync(join(inbox,'foreign.json'));unlinkSync(join(inbox,'nosession.json'));
// Addressed here with the matching session: submitted and acked ok.
session='original';
await hooks.session_start({},ctx());
write('mine',{nonce:'m',pid:process.pid,sessionPath:'original',payload:'p'});
{
	const r=await settle('mine',2500);
	if(!r.a.find(x=>x.nonce==='m'&&x.ok===true)||r.sends!==1||r.left){console.error({case:'mine',...r});process.exit(1)}
}
// Addressed here but showing another session: refused with the session reason.
write('other',{nonce:'o',pid:process.pid,sessionPath:'elsewhere',payload:'p'});
{
	const r=await settle('other',2500);
	if(!r.a.find(x=>x.nonce==='o'&&x.ok===false&&x.reason==='the terminal is showing a different session')||r.sends!==0||r.left){console.error({case:'other',...r});process.exit(1)}
}
console.log('receiver pid/session addressing passed');process.exit(0);
`
	path := filepath.Join(dir, "harness.mjs")
	os.WriteFile(path, []byte(harness), 0600)
	if out, err := exec.Command("node", path, dir).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
}

func TestNativePiAttentionReceiverDecisionTable(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node unavailable")
	}
	for _, tc := range []struct{ name, mode string }{
		{"idle managed", "managed"}, {"idle terminal", "terminal"}, {"became busy", "busy"}, {"queued prompt", "queued"}, {"retained draft", "draft"}, {"session switched", "switched"}, {"owner reply still queues", "owner"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			module := filepath.Join(dir, "receiver.mjs")
			os.WriteFile(module, []byte(piInboxReplyExtensionTS), 0600)
			harness := `import { createServer } from 'node:http';
import { writeFileSync,mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
const [dir,mode]=process.argv.slice(2), hooks={};let sends=0;
process.env.PICODE_DATA=dir;process.env.PICODE_AGENT_ID='fixture';delete process.env.PICODE_TERM_ID;
const server=createServer((req,res)=>{let data='';req.on('data',v=>data+=v);req.on('end',()=>{
 res.writeHead(204);res.end();if(!req.url.endsWith('tui-ack'))return;
 const ack=JSON.parse(data),want=['managed','terminal','owner'].includes(mode);
 if(ack.ok!==want || sends!==(want?1:0)){console.error({mode,ack,sends});process.exit(1)}process.exit(0);
})});
await new Promise(r=>server.listen(0,'127.0.0.1',r));
writeFileSync(join(dir,'server.json'),JSON.stringify({url:'http://127.0.0.1:'+server.address().port}));
const {default:register}=await import(pathToFileURL(join(dir,'receiver.mjs')));
register({on:(name,fn)=>hooks[name]=fn,sendUserMessage:async()=>{sends++}});
const inbox=join(dir,'tui-inbox','fixture');mkdirSync(inbox,{recursive:true});
writeFileSync(join(inbox,'one.json'),JSON.stringify({nonce:'one',sessionPath:'original',payload:'pointer',attentionOnly:mode!=='owner',createdAt:new Date().toISOString()}));
await hooks.session_start({}, {sessionManager:{getSessionFile:()=>mode==='switched'?'new':'original'},isIdle:()=>!['busy','owner'].includes(mode),hasPendingMessages:()=>mode==='queued',hasUI:mode!=='managed',ui:{getEditorText:()=>mode==='draft'?'keep draft':''}});
setTimeout(()=>{console.error('receiver timeout');process.exit(1)},3000);
`
			path := filepath.Join(dir, "harness.mjs")
			os.WriteFile(path, []byte(harness), 0600)
			if out, err := exec.Command("node", path, dir, tc.mode).CombinedOutput(); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
		})
	}
}
