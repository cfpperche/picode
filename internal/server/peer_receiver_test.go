package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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
