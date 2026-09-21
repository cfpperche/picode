import { useEffect, useRef, useState } from "react";
import { deliveryURL, observeDelivery, setDeliveryObserver } from "@picode/shared/client/delivery.js";
import { deliveryRows, deliveryReason, integrationLabel, validationLabel, observationLabel, integrationExplanation, validationExplanation, environmentLane, publicationLabel, publicationReason } from "@picode/shared/domain/delivery.js";
import { toastError } from "../lib/toast.js";
import "../styles/mobile-delivery.css";

// The public guide explains connecting this instance (ADR-0170 D2); the lane's
// "View setup" opens it rather than a dead link.
const SETUP_URL = "https://cfpperche.github.io/picode/guide/delivery#deployment";

// The phone's Delivery section holds both lenses (ADR-0170). Integration reads
// Git and recorded check evidence; Deployment reads the connected instance's own
// revision and its recorded attempts. Neither lens deploys anything: the only
// write in either is connecting or disconnecting the environment itself. The
// lens itself is the screen's state (Git.jsx), so Back and a re-entry keep it.
export default function GitDelivery({ owner, hidden, lane = "integration", onLane, onHistory, selection = "", onDetail }) {
  const root=useRef("");
  const [follow,setFollow]=useState(0);
  const [target,setTarget]=useState("");
  const [filter,setFilter]=useState("all");
  const selected=selection, setSelected=id=>{if(id || selection)onDetail(id);};
  const [state,setState]=useState({data:null,busy:true});
  const ref=useRef(null);
  useEffect(()=>{
    if(hidden)return;

    const sub=observeDelivery({url:deliveryURL(owner,root.current,target),onChange:s=>{setState(s);if(s.data&&!root.current)root.current=s.data.root;}});
    ref.current=sub;return()=>sub.stop();
  },[owner.kind,owner.id,target,hidden,follow]);
  const data=state.data, rows=deliveryRows(data?.changes,filter), detail=data?.changes.find(c=>c.id===selected), env=data?.environments?.[0]||null;
  const refresh=()=>ref.current?.refresh();
  // A read that fails keeps the last observation and says so above whichever
  // lens is on screen, so both branches carry these three lines.
  const notices = <>
    {state.moved?<div className="m-delivery-warning"><p>This project's folder changed.</p><button className="btn" onClick={()=>{root.current="";setState({data:null,busy:true});setSelected("");setTarget("");setFollow(n=>n+1);}}>Follow folder</button></div>:null}
    {state.error&&!state.moved?<div className="m-delivery-warning"><p>{data?"Could not update; showing the last check.":"Could not read delivery status."}</p><button className="btn" onClick={refresh}>Retry</button></div>:null}
    {!data&&state.busy?<div className="m-delivery-skeleton" aria-label="Loading deliveries">{[1,2,3].map(n=><div key={n}/>)}</div>:null}
  </>;
  // A selected change is a change, whichever lens was open when it was picked.
  const deployment=lane==="deployment"&&!detail;
  return <section className="m-delivery" aria-label="Delivery" hidden={!!hidden}>
    {deployment ? <>{notices}<DeploymentLane env={env} data={data} owner={owner} busy={state.busy||state.moved} onRefresh={refresh} onLane={onLane}/></> : <>
    <div className="m-delivery-toolbar">
      <label data-align-row>Target <select aria-label="Delivery target" value={target||data?.target||""} onChange={e=>{setState({data:null,busy:true});setSelected("");setTarget(e.target.value);}}><option value="">Select target</option>{data?.targets.map(t=><option key={t} value={t}>{t}</option>)}</select></label>
      <div className="m-delivery-actions" data-align-row><select aria-label="Delivery filter" value={filter} onChange={e=>setFilter(e.target.value)}><option value="all">All changes</option><option value="attention">Needs attention</option></select>
      <button className="btn" disabled={state.busy||state.moved} onClick={refresh}>Refresh</button></div>
    </div>
    <p className="m-delivery-guide" role="note"><strong>How to read this:</strong> Integrated means included in the target branch. Checks describe recorded test results. Neither status confirms publication.</p>
    {!state.error ? <p className="m-delivery-status" role="status">{observationLabel(state)}</p> : null}
    {notices}
    {data&&!data.targetOid?<div className="m-delivery-empty"><p>Choose the branch changes will join.</p><button className="btn" onClick={()=>document.querySelector('[aria-label="Delivery target"]')?.focus()}>Select target</button></div>:null}
    {data?.targetOid&&!data.complete?<div className="m-delivery-warning"><p>Some changes could not be checked.</p><ul>{data.issues.map(x=><li key={x}>{x}</li>)}</ul><button className="btn" onClick={refresh}>Retry</button></div>:null}
    {data?.targetOid&&!rows.length?<div className="m-delivery-empty"><p>{filter==="attention"?"No changes match this filter.":"No observed changes for this branch."}</p><button className="btn" onClick={()=>filter==="attention"?setFilter("all"):onHistory()}>{filter==="attention"?"Clear filter":"View history"}</button></div>:null}
    {detail?<article className="m-delivery-detail"><button className="btn" onClick={()=>setSelected("")}>Back to deliveries</button><h3>{detail.title}</h3><p>{deliveryReason(detail)}</p><div className="m-delivery-summary"><div><strong>{integrationLabel[detail.integration]}</strong><span>{integrationExplanation(detail.integration)}</span></div><div><strong>{validationLabel[detail.validation]}</strong><span>{validationExplanation(detail.validation)}</span></div></div>{detail.integration==="integrated"?<div className="m-delivery-summary"><div><strong>{publicationLabel[detail.publication]||publicationLabel.unknown}</strong><span>{publicationReason(detail, env)}</span></div></div>:null}<dl><dt>Revision</dt><dd>{detail.revision}</dd><dt>Source</dt><dd>{detail.registered?"Registered by an agent":"Observed in Git"} · {detail.branch}</dd><dt>Associated agents</dt><dd>{detail.agents.join(", ")||"Association not recorded"}</dd></dl><h4>Recorded evidence</h4>{detail.evidence.length?detail.evidence.map(e=><p key={e.id}>{{scoped:"Relevant checks", "full-ci":"Full project checks",land:"Integration attempt"}[e.kind]}: {e.outcome}<br/><time>{e.at||"Finish not recorded"}</time><br/><code>{e.source}</code></p>):<p>{validationExplanation(detail.validation)}</p>}<button className="btn" onClick={()=>onHistory(detail.revision)}>Open history</button></article>:null}
    {!detail?<ul className="m-delivery-list">{rows.map(c=><li key={c.id}><div><h3>{c.title}</h3><p className="m-delivery-meta">{c.branch} · {c.registered?"Registered delivery":"Observed branch"}</p><p>{deliveryReason(c)}</p><p className="m-delivery-meta">{c.agents.length?"Associated: "+c.agents.join(", "):"Agent association not recorded"}</p></div><div className="m-delivery-facts"><span>{integrationLabel[c.integration]}</span><span data-failed={c.validation==="failed"}>{validationLabel[c.validation]}</span><button className="btn" onClick={()=>setSelected(c.id)}>View details</button></div></li>)}</ul>:null}
    </>}
  </section>;
}

// The Deployment lens (ADR-0170 D2): one environment, the instance's own
// identity, the integrated changes its artifact does not contain, and the newest
// recorded attempt — one fact per line, its action at the end of the line.
// Read-only apart from the environment binding, and an unfinished attempt is
// "Unknown outcome", never running and never failed.
function DeploymentLane({ env, data, owner, busy, onRefresh, onLane }) {
  const model=environmentLane(env, data?.targetOid||"");
  const connected=!!env?.repositoryKey;
  // The write is a round trip on a phone's network: one at a time, so a second
  // tap cannot race the refresh that follows the first.
  const [saving,setSaving]=useState(false);
  const connect=async next=>{
    if(saving)return;
    setSaving(true);
    try{await setDeliveryObserver(owner,next);onRefresh();}
    catch(e){toastError(e);}
    finally{setSaving(false);}
  };
  const locked=busy||saving;
  // The read always answers with this environment (connected, unconfigured or
  // in conflict); the screen's own skeleton covers the window before it does.
  const running=model.facts[0]||"";
  const issues=model.facts.slice(1);
  return <>
    <div className="m-delivery-toolbar">
      <label data-align-row>Environment <select aria-label="Deployment environment" value={connected?"picode-self":"none"} disabled={locked} onChange={e=>connect(e.target.value)}><option value="none">Not connected</option><option value="picode-self">This PiCode instance</option></select></label>
      <div className="m-delivery-actions" data-align-row><button className="btn" disabled={locked} onClick={onRefresh}>Refresh</button></div>
    </div>
    {!env?null:<>
    {model.headline?<p className="m-delivery-headline">{model.headline}</p>:null}
    {running?<p className="m-delivery-meta">{running}</p>:null}
    {model.action?.kind==="setup"?<div className="m-delivery-env-row" data-align-row data-align-wrap><a className="btn" href={SETUP_URL} target="_blank" rel="noreferrer">{model.action.label}</a></div>:null}
    {model.unpublished?<div className="m-delivery-env-row" data-align-row data-align-wrap><span className="m-delivery-env-fact">{model.unpublished.text}</span><button className="btn" onClick={()=>onLane?.("integration")}>View changes</button></div>:null}
    {model.attempt?<div className="m-delivery-env-row" data-align-row data-align-wrap><span className="m-delivery-env-fact">{model.attempt.text}{model.attempt.error?" · "+model.attempt.error:""}</span></div>:connected?<p className="m-delivery-meta">No deployment recorded for this project yet.</p>:null}
    {model.busy?<div className="m-delivery-env-row" data-align-row data-align-wrap><span className="m-delivery-env-fact">{model.busy.text}</span><a className="btn" href="#/">View activity</a></div>:null}
    {connected?<details className="m-delivery-evidence"><summary>View evidence</summary><dl><dt>Running revision</dt><dd><code>{env.revision||"Not recorded"}</code></dd><dt>Instance boot</dt><dd><code>{env.boot||"Not recorded"}</code></dd><dt>Artifact</dt><dd>{env.artifact}</dd>{env.lastAttempt?<><dt>Attempt</dt><dd><code>{env.lastAttempt.id}</code> · {env.lastAttempt.outcome}</dd><dt>Attempt source</dt><dd><code>{env.lastAttempt.builtRevision||"Not recorded"}</code>{env.lastAttempt.clean?" · clean checkout":""}</dd></>:null}</dl>{issues.length?<ul className="m-delivery-issues">{issues.map(x=><li key={x}>{x}</li>)}</ul>:null}{data?.issues?.length?<><p className="m-delivery-meta">Some changes could not be checked.</p><ul className="m-delivery-issues">{data.issues.map(x=><li key={x}>{x}</li>)}</ul></>:null}</details>:null}
    </>}
  </>;
}
