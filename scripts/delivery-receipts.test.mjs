import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, readFileSync, rmSync, readdirSync, symlinkSync, statSync, writeFileSync, copyFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { startReceipt, finishReceipt, recordSafely } from "./delivery-receipts.mjs";
function repo(t){const d=mkdtempSync(join(tmpdir(),"delivery-"));t.after(()=>rmSync(d,{recursive:true,force:true}));const git=(...a)=>execFileSync("git",a,{cwd:d,encoding:"utf8",stdio:["ignore","pipe","ignore"]}).trim();git("init","-b","main");git("-c","user.name=Test","-c","user.email=test@example.invalid","-c","commit.gpgsign=false","commit","--allow-empty","-m","fixture");return {d,git};}
test("receipts preserve failed result, exact source and one atomic record per operation",t=>{
 const {d,git}=repo(t), id=startReceipt(d,"full-ci"), path=join(d,".git/picode-delivery",id+".json");
 assert.equal(JSON.parse(readFileSync(path)).outcome,"started");finishReceipt(d,id,7);
 const r=JSON.parse(readFileSync(path));assert.equal(r.outcome,"failed");assert.equal(r.source,git("rev-parse","HEAD"));assert.equal(r.clean,true);
 assert.equal(readdirSync(join(d,".git/picode-delivery")).length,1);if(process.platform!=="win32") assert.equal(statSync(path).mode&0o777,0o600);
 assert.throws(()=>finishReceipt(d,id,0));
});
test("receipt directory symlink is refused and recording errors remain advisory",t=>{
 const {d}=repo(t), outside=join(d,"outside");mkdirSync(outside);symlinkSync(outside,join(d,".git/picode-delivery"),process.platform==="win32"?"junction":"dir");
 assert.throws(()=>startReceipt(d,"scoped"));assert.equal(recordSafely(()=>startReceipt(d,"scoped")),"");assert.deepEqual(readdirSync(outside),[]);
});
test("land receipt retains deleted branch source",t=>{
 const {d,git}=repo(t);git("branch","feature");const id=startReceipt(d,"land","feature");git("branch","-d","feature");finishReceipt(d,id,0);
 assert.equal(JSON.parse(readFileSync(join(d,".git/picode-delivery",id+".json"))).sourceRef,"feature");
});

test("CI exit code survives receipt write failure", {skip:process.platform==="win32"&&"POSIX shell fixture"}, t => {
 const {d}=repo(t);mkdirSync(join(d,"scripts"));mkdirSync(join(d,"bin"));mkdirSync(join(d,"outside"));
 for(const file of ["ci.sh","delivery-receipts.mjs"]) copyFileSync(new URL(file,import.meta.url),join(d,"scripts",file));
 writeFileSync(join(d,"bin/make"),"#!/bin/sh\nexit 7\n",{mode:0o700});
 symlinkSync(join(d,"outside"),join(d,".git/picode-delivery"));
 const result=spawnSync("bash",[join(d,"scripts/ci.sh")],{cwd:d,env:{...process.env,PATH:join(d,"bin")+":"+process.env.PATH},encoding:"utf8"});
 assert.equal(result.status,7);assert.deepEqual(readdirSync(join(d,"outside")),[]);
});

// Both implementations exercise ADR-0124 with the same cases as TestReuseParity.
import { decideReuse } from "./ci-scope-reuse.mjs";
test("observation reuse cases match the producer's gate reuse decision",()=>{
 for(const [stamp,head,covered,dirty,want] of [[null,"a","b",false,false],[{tree:"a",covered:"b"},"a","c",false,true],[{tree:"a",covered:"b"},"c","b",false,true],[{tree:"a",covered:"b"},"c","c",false,false],[{tree:"a",covered:"b"},"a","b",true,false]]) assert.equal(decideReuse({stamp,head,covered,dirty}).reuse,want);
});
