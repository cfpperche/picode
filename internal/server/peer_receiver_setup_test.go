package server

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativePiSetupSharesRegistrationAndPreservesDraft(t *testing.T) {
	if _, e := exec.LookPath("node"); e != nil {
		t.Skip("node unavailable")
	}
	dir := t.TempDir()
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("trusted")) }))
	defer tlsServer.Close()
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tlsServer.Certificate().Raw})
	os.WriteFile(filepath.Join(dir, "trust.pem"), cert, 0600)
	if e := os.WriteFile(filepath.Join(dir, "receiver.mjs"), []byte(piInboxReplyExtensionTS), 0600); e != nil {
		t.Fatal(e)
	}
	js := `import assert from 'node:assert/strict';
import {createServer} from 'node:http';
import {writeFileSync,mkdirSync,readFileSync} from 'node:fs';
import {join} from 'node:path';
import {pathToFileURL} from 'node:url';
const dir=process.argv[2],tlsURL=process.argv[3],hooks={},listeners={};let registrations=0,disposals=0,sends=0,session='a',releaseDispose;
process.env.PICODE_DATA=dir;process.env.PICODE_AGENT_ID='fixture';delete process.env.PICODE_TERM_ID;
const hellos=[];const server=createServer((req,res)=>{let body='';req.on('data',v=>body+=v);req.on('end',()=>{if(req.url.endsWith('tui-hello'))hellos.push(JSON.parse(body));res.writeHead(204);res.end()})});
await new Promise(r=>server.listen(0,'127.0.0.1',r));
writeFileSync(join(dir,'server.json'),JSON.stringify({url:'http://127.0.0.1:'+server.address().port}));
const {default:register}=await import(pathToFileURL(join(dir,'receiver.mjs')));
let live=false;
const pi={on:(n,fn)=>hooks[n]=fn,events:{on:(n,fn)=>listeners[n]=fn,emit:(n,r)=>{if(n==='pi-mcp-adapter:runtime-register:v1'){assert.equal(live,false,'duplicate registration');live=true;registrations++;r.result={ok:true,registration:{dispose:async()=>{disposals++;live=false;if(releaseDispose)await releaseDispose}}}}}},sendUserMessage:()=>{sends++}};
register(pi);
const ctx={sessionManager:{getSessionFile:()=>session},isIdle:()=>true,hasPendingMessages:()=>false,hasUI:false};
await hooks.session_start({},ctx);
await assert.rejects(fetch(tlsURL));
const config=id=>({caBundle:readFileSync(join(dir,'trust.pem'),'utf8'),connection:{id,ownerId:'fixture',sessionKey:session},url:'http://localhost/mcp/communication',token:'fixture'});
const configure=async c=>{const r={config:c,ctx};listeners['picode:communication-configure'](r);await r.promise};
await configure(config('peer_one'));assert.equal(await (await fetch(tlsURL)).text(),'trusted');await configure(config('peer_one'));assert.equal(registrations,1);
await configure(config('peer_two'));assert.equal(registrations,2);assert.equal(disposals,1);
assert.equal(hellos.at(-1).connection,'peer_two');assert.equal(hellos.at(-1).pid,process.pid);
// A session switch while old registration disposal awaits must refuse the new one.
let release;releaseDispose=new Promise(r=>release=r);const pending=configure(config('peer_three'));session='b';release();
await assert.rejects(pending,/Conversation changed/);assert.equal(registrations,2);
releaseDispose=undefined;await hooks.session_before_switch();
await configure(config('peer_four'));assert.equal(registrations,3);
assert.equal(sends,0,'setup started a model turn');
await hooks.session_shutdown();assert.equal(live,false);
console.log('native setup replacement, session fence, ordered hello and no model prompt passed');process.exit(0);
`
	path := filepath.Join(dir, "test.mjs")
	if e := os.WriteFile(path, []byte(js), 0600); e != nil {
		t.Fatal(e)
	}
	if out, e := exec.Command("node", path, dir, tlsServer.URL).CombinedOutput(); e != nil {
		t.Fatalf("%v: %s", e, out)
	}
}
