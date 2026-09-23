import { useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { llamaServiceSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { askConfirm } from "../lib/confirm.js";

const defaults = { port: 18080, context: 4096, threads: 2, jinja: true };
const labels = { install: "Install", update: "Update", start: "Start", stop: "Stop", restart: "Restart", rollback: "Restore previous version", cleanup: "Clean selected items" };
export default function LlamaService({ onConnect }) {
 const [state,setState]=useState(null),[error,setError]=useState(""),[config,setConfig]=useState(defaults);
 const [version,setVersion]=useState("b10809"),[pending,setPending]=useState(""),[files,setFiles]=useState([]),[selected,setSelected]=useState([]);
 const initialized=useRef(false), locked=useRef(false);
 // readError is the service read's own (a successful read clears it); error
 // stays an action's. One read at a time, with one rerun queued: an install
 // saves its progress about four times a second, and each save is an event.
 const [readError,setReadError]=useState(""), reading=useRef(false), readAgain=useRef(false);
 const actionLabel=action=>action==="update"&&state?.current?.version===version?"Reinstall":labels[action];
 async function refresh(){
  if(reading.current){readAgain.current=true;return;}
  reading.current=true;
  try {
   const [next,cache]=await Promise.all([api("/api/llama/service"),api("/api/llama/service/cache")]);
   setState(current=>!current || next.revision>=current.revision?next:current);setFiles(cache.files||[]);
   setSelected(current=>current.filter(name=>(cache.files||[]).some(f=>f.name===name&&f.eligible)));
   if(!initialized.current){setConfig(next.created?next.config:defaults);initialized.current=true;}
   setReadError("");
  } catch(e){setReadError(e.message||"Could not check the local service.");}
  finally{reading.current=false;if(readAgain.current){readAgain.current=false;refresh();}}
 }
 useEffect(()=>{
  refresh();
  return subscribeFeed(event=>{if(["llama.service","feed.open","feed.reset"].includes(event.type))refresh();if(event.type==="feed.down")setReadError("Live updates paused. Refresh to check the service.");});
 },[]);
 async function save(e){
  e.preventDefault();if(locked.current)return;
  const parsed=parseForm(llamaServiceSchema,config);if(!parsed.ok){setError(parsed.error);return;}
  locked.current=true;setPending("Saving settings…");setError("");
  try{const next=await api("/api/llama/service",{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({config:parsed.value,revision:state.revision})});setState(next);}
  catch(e){setError(e.message);}
  finally{locked.current=false;setPending("");}
 }
 async function act(action){
  if(locked.current)return;locked.current=true;setPending("Preparing review…");setError("");
  try{
   const review=await api("/api/llama/service/preview",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({action,version,files:selected,revision:state.revision})});
   const consumers=review.consumers||[];
   const result=await askConfirm({
    title:action==="cleanup"?"Clean selected items":actionLabel(action)+" local service",
    message: action==="cleanup" ? "Remove "+selected.map(name=>{const f=files.find(f=>f.name===name);return f?.label?f.label+" ("+name+")":name;}).join(", ")+", freeing "+(review.bytes/1048576).toFixed(1)+" MiB. Other files are retained." :
     (action==="install"||action==="update" ? "Use verified CPU release "+version+". " : "")+
     (consumers.length ? "This interrupts: "+consumers.join(", ")+". " : "")+
     "Applies only to the service created here.",
    confirmLabel:actionLabel(action),
    danger:action==="cleanup",
    ...(consumers.length?{choices:[{id:"interrupt",label:"I accept interrupting connected applications",checked:false}]}:{}),
   });
   if(!result)return;
   if(consumers.length&&!result.interrupt){setError("Confirm interruption before applying this action.");return;}
   setPending("Submitting action…");
   const response=await api("/api/llama/service/execute",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({token:review.token,interrupt:!!result.interrupt})});
   setState(current=>({...current,busy:true,jobs:[response.job,...(current.jobs||[])]}));setSelected([]);await refresh();
  }catch(e){setError(e.message);}
  finally{locked.current=false;setPending("");}
 }
 if(!state&&!error&&!readError)return <div className="llama-skeleton" aria-label="Loading local service"><div/><div/><div/></div>;
 return <section className="llama-owned" aria-label="Local service">
  <div className="llama-toolbar"><h3>Local service</h3><button className="btn btn-ghost btn-sm" onClick={()=>{setError("");refresh();}}>Refresh service</button></div>
  {error?<p role="alert">{error}</p>:null}
  {readError&&readError!==error?<p role="status">{readError}</p>:null}
  {!state?null:!state.supported?<div className="llama-empty"><p>Local services require Linux or WSL on x64 or ARM64.</p><a className="btn btn-primary btn-sm" href="#/llama/server">Configure external server</a></div>:<>
   <p className="settings-desc">On {state.host} · {state.created?(state.running?"Running":"Stopped"):"Not configured"}</p>
   {!state.created?<p>Create a separate CPU service managed by PiCode.</p>:<p className="settings-desc">Address: {state.url} · {state.current?.version||"Not installed"}</p>}
   <form className="llama-form" noValidate onSubmit={save}>
    <label>Performance preset<select value={config.threads===2&&config.context===4096?"light":config.threads===4&&config.context===8192?"balanced":"custom"} onChange={e=>{if(e.target.value==="light")setConfig(c=>({...c,threads:2,context:4096}));if(e.target.value==="balanced")setConfig(c=>({...c,threads:4,context:8192}));}}>
     <option value="light">Light · 2 threads · 4K</option><option value="balanced">Balanced · 4 threads · 8K</option><option value="custom">Custom</option>
    </select></label>
    <details><summary>Advanced settings</summary><div className="llama-owned-fields">
     {[["port","Port"],["context","Context size"],["threads","CPU threads"]].map(([name,label])=><label key={name}>{label}<input type="number" value={config[name]} onChange={e=>setConfig(c=>({...c,[name]:Number(e.target.value)}))}/></label>)}
     <label className="llama-owned-check"><input type="checkbox" checked={config.jinja} onChange={e=>setConfig(c=>({...c,jinja:e.target.checked}))}/>Enable chat templates</label>
    </div></details>
    <p className="settings-desc">CPU only. Models load on request. Saving settings does not restart the server.</p>
    {state.restartRequired?<p role="status">Saved settings need a restart to take effect.</p>:null}
    <button className="btn btn-primary btn-sm" disabled={!!pending||state.busy}>{state.created?"Save settings":"Create local service"}</button>
   </form>
   {state.created?<>
    {state.running?<button type="button" className="btn btn-primary btn-sm" onClick={()=>onConnect?.(state.url)}>Use this connection</button>:null}
    <div className="llama-owned-controls">
     <label>Verified version<select value={version} onChange={e=>setVersion(e.target.value)}>{state.versions.map(v=><option key={v}>{v}</option>)}</select></label>
     <div className="llama-actions">
      {(!state.current?["install"]:state.running?["stop","restart","update"]:["start","update"]).concat(state.previous?["rollback"]:[]).map(action=><button key={action} type="button" className="btn btn-ghost btn-sm" disabled={!!pending||state.busy} onClick={()=>act(action)}>{actionLabel(action)}</button>)}
     </div>
    </div>
    <details><summary>Effective command</summary>{state.command?.length?<pre className="llama-owned-command">{JSON.stringify(state.command,null,2)}</pre>:<p>Install a verified version to preview its command.</p>}</details>
    <details><summary>Cache and installations</summary>
     {!files.length?<p>No cache files or installations to clean.</p>:<ul className="prov-list">{files.map(f=><li key={f.name} className="llama-cache-row">
      <label className="llama-owned-check">{f.eligible?<input type="checkbox" checked={selected.includes(f.name)} onChange={e=>setSelected(s=>e.target.checked?[...s,f.name]:s.filter(n=>n!==f.name))}/>:null}<span>{f.label||f.name}</span></label>
      {f.label?<small>{f.name}</small>:null}
      <small>{f.label&&!f.eligible?"Retained":(f.bytes/1048576).toFixed(1)+" MiB"} · {f.reason}</small>
     </li>)}</ul>}
     {files.some(f=>f.eligible)?<button className="btn btn-ghost btn-sm" disabled={!selected.length||!!pending||state.busy} onClick={()=>act("cleanup")}>Clean selected items</button>:null}
    </details>
    <div className="llama-toolbar"><h4>Service activity</h4><a className="btn btn-ghost btn-sm" href="/api/llama/service/diagnostics" download>Export diagnostics</a></div>
    {pending?<p className="llama-working" role="status">{pending}</p>:null}
    {!state.jobs?.length?<p className="settings-desc">No service actions yet. Install the verified version to begin.</p>:<ol className="llama-service-jobs">{state.jobs.map(j=><li key={j.id} className={["queued","running"].includes(j.state)?"llama-working":""}><strong>{labels[j.action]||j.action} · {j.state}</strong><span>{j.message}</span>{j.done>0?<small>{(j.done/1048576).toFixed(1)} MiB{j.total>0?" / "+(j.total/1048576).toFixed(1)+" MiB":""}</small>:null}</li>)}</ol>}
   </>:null}
  </>}
 </section>;
}
