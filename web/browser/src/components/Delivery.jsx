import { useEffect, useRef, useState } from "react";
import { deliveryURL, observeDelivery, setDeliveryObserver } from "@picode/shared/client/delivery.js";
import { deliveryRows, deliveryReason, integrationLabel, validationLabel, observationLabel, integrationExplanation, validationExplanation, environmentLane, publicationLabel, publicationReason } from "@picode/shared/domain/delivery.js";
import { IconChevronLeft } from "./Icons.jsx";
import { toast } from "../lib/toast.js";
import "../styles/delivery.css";

// A receipt's kind is stored, its label is presentation.
const EVIDENCE_KIND = { scoped: "Relevant checks", "full-ci": "Full project checks", land: "Integration attempt" };

// The public guide explains connecting the local instance (ADR-0170 D2); the
// lane's "View setup" opens it rather than a dead link.
const SETUP_URL = "https://cfpperche.github.io/picode/guide/delivery#deployment";

// Delivery, the Git tab's second view, draws the same page a system route
// draws: .settings-wrap + .settings-head + .settings-card (the Agent CLIs
// standard, ADR-0103; owner 2026-09-21). The Git tab's own strip — the
// workspace picker and the History/Delivery toggle — stays where it is and
// outside the frame, because History is still a canvas and must not move.
//
// Inside the card the view has two lenses (ADR-0170): Integration reads Git
// and check evidence, Deployment reads the connected instance's own revision
// and its recorded deployment attempts. Neither lens deploys anything.
export default function Delivery({ owner, lane = "integration", onLane, hidden, onHistory }) {
  const root=useRef("");
  const [follow,setFollow]=useState(0);
  const [target,setTarget]=useState("");
  const [filter,setFilter]=useState("all");
  const [selected,setSelected]=useState("");
  const [state,setState]=useState({data:null,busy:true});
  const ref=useRef(null);
  useEffect(()=>{
    if(hidden)return;

    const sub=observeDelivery({url:deliveryURL(owner,root.current,target),onChange:s=>{setState(s);if(s.data&&!root.current)root.current=s.data.root;}});
    ref.current=sub;return()=>sub.stop();
  },[owner.kind,owner.id,target,hidden,follow]);
  const data=state.data, rows=deliveryRows(data?.changes,filter), detail=data?.changes.find(c=>c.id===selected);
  const env=data?.environments?.[0]||null;
  const refresh=()=>ref.current?.refresh();
  const followFolder=()=>{root.current="";setState({data:null,busy:true});setSelected("");setTarget("");setFollow(n=>n+1);};
  const deployment=lane==="deployment";
  return <section className="delivery-page" aria-label="Delivery" hidden={!!hidden}>
    <div className="settings-wrap">
      <header className="settings-head"><h2>Delivery</h2></header>
      <div className="settings-card delivery-card">
        <nav className="delivery-lanes" aria-label="Delivery lenses">
          <button className="delivery-lane" aria-current={deployment?undefined:"page"} onClick={()=>onLane?.("integration")}>Integration</button>
          <button className="delivery-lane" aria-current={deployment?"page":undefined} onClick={()=>onLane?.("deployment")}>Deployment</button>
        </nav>
        <div className="delivery-body">
          {deployment
            ? <DeploymentLane env={env} data={data} owner={owner} state={state} onRefresh={refresh} onFollow={followFolder} onLane={onLane}/>
            : <>
              <div className="delivery-toolbar" data-align-row>
                <label data-align-row>Target <select aria-label="Delivery target" value={target||data?.target||""} onChange={e=>{setState({data:null,busy:true});setSelected("");setTarget(e.target.value);}}><option value="">Select target</option>{data?.targets.map(t=><option key={t} value={t}>{t}</option>)}</select></label>
                <select aria-label="Delivery filter" value={filter} onChange={e=>setFilter(e.target.value)}><option value="all">All changes</option><option value="attention">Needs attention</option></select>
                <button className="btn delivery-refresh" disabled={state.busy||state.moved} onClick={refresh}>Refresh</button>
              </div>
              <p className="delivery-guide" role="note"><strong>How to read this:</strong> Integrated means included in the target branch. Checks describe recorded test results. Neither status confirms publication.</p>
              {!state.error ? <p className="delivery-status" role="status">{observationLabel(state)}</p> : null}
              {state.moved?<div className="delivery-warning"><p>This project's folder changed.</p><button className="btn" onClick={followFolder}>Follow folder</button></div>:null}
              {state.error&&!state.moved?<div className="delivery-warning"><p>{data?"Could not update; showing the last check.":"Could not read delivery status."}</p><button className="btn" onClick={refresh}>Retry</button></div>:null}
              {!data&&state.busy?<div className="delivery-skeleton" aria-label="Loading deliveries">{[1,2,3].map(n=><div key={n}/>)}</div>:null}
              {data&&!data.targetOid?<div className="mcp-empty"><p>Choose the branch changes will join.</p><button className="btn" onClick={()=>document.querySelector('[aria-label="Delivery target"]')?.focus()}>Select target</button></div>:null}
              {data?.targetOid&&!data.complete?<div className="delivery-warning"><p>Some changes could not be checked.</p><ul>{data.issues.map(x=><li key={x}>{x}</li>)}</ul><button className="btn" onClick={refresh}>Retry</button></div>:null}
              {data?.targetOid&&!rows.length?<div className="mcp-empty"><p>{filter==="attention"?"No changes match this filter.":"No observed changes for this branch."}</p><button className="btn" onClick={()=>filter==="attention"?setFilter("all"):onHistory()}>{filter==="attention"?"Clear filter":"View history"}</button></div>:null}
              {detail?<article className="delivery-detail"><button className="delivery-back" onClick={()=>setSelected("")}><IconChevronLeft size={13} />Back to deliveries</button><h3>{detail.title}</h3><p>{deliveryReason(detail)}</p><div className="delivery-summary"><div><strong>{integrationLabel[detail.integration]}</strong><span>{integrationExplanation(detail.integration)}</span></div><div><strong>{validationLabel[detail.validation]}</strong><span>{validationExplanation(detail.validation)}</span></div>{detail.integration==="integrated"?<div><strong>{publicationLabel[detail.publication]||publicationLabel.unknown}</strong><span>{publicationReason(detail, env)}</span></div>:null}</div><dl className="delivery-pairs"><dt>Revision</dt><dd><code>{detail.revision}</code></dd><dt>Source</dt><dd>{detail.registered?"Registered by an agent":"Observed in Git"} · {detail.branch}</dd><dt>Associated agents</dt><dd>{detail.agents.join(", ")||"Association not recorded"}</dd></dl><h4 className="delivery-heading">Recorded evidence</h4>{detail.evidence.length?<ul className="delivery-evidence">{detail.evidence.map(e=><li key={e.id}><strong>{EVIDENCE_KIND[e.kind]||e.kind}: {e.outcome}</strong><span className="delivery-meta"><time>{e.at||"Finish not recorded"}</time> · <code>{e.source}</code></span></li>)}</ul>:<p>{validationExplanation(detail.validation)}</p>}<button className="btn" onClick={()=>onHistory(detail.revision)}>Open history</button></article>:null}
              {!detail?<ul className="delivery-list">{rows.map(c=><li key={c.id}><div><h3>{c.title}</h3><p className="delivery-meta">{c.branch} · {c.registered?"Registered delivery":"Observed branch"}</p><p>{deliveryReason(c)}</p><p className="delivery-meta">{c.agents.length?"Associated: "+c.agents.join(", "):"Agent association not recorded"}</p></div><div className="delivery-facts"><span>{integrationLabel[c.integration]}</span><span data-failed={c.validation==="failed"}>{validationLabel[c.validation]}</span>{c.integration==="integrated"&&c.publication==="not-published"?<span>Not published</span>:null}<button className="btn" onClick={()=>setSelected(c.id)}>View details</button></div></li>)}</ul>:null}
            </>}
        </div>
      </div>
    </div>
  </section>;
}

// The Deployment lens (ADR-0170 D2): one connected environment, the instance's
// own identity, the integrated changes its artifact does not contain, and the
// newest recorded attempt. Read-only: the single write is connecting or
// disconnecting the environment itself.
function DeploymentLane({ env, data, owner, state, onRefresh, onFollow, onLane }) {
  const model = environmentLane(env, data?.targetOid || "");
  // A read that failed is not a read that is still running: the lane says so
  // instead of showing a skeleton forever.
  const spinning = !env && state.busy && !state.moved;
  const failure = !env && !spinning ? (state.moved ? "This project's folder changed." : state.error ? "Could not read delivery status." : "No deployment status yet.") : "";
  const busy = state.busy || state.moved;
  const connected = !!env?.repositoryKey;
  const connect = async (next) => {
    try {
      await setDeliveryObserver(owner, next);
      onRefresh();
    } catch (e) {
      toast(e.message || "Could not change the deployment environment.");
    }
  };
  return <>
    <div className="delivery-toolbar" data-align-row>
      <label data-align-row>Environment <select aria-label="Deployment environment" value={connected?"picode-self":"none"} disabled={busy} onChange={e=>connect(e.target.value)}><option value="none">Not connected</option><option value="picode-self">This PiCode instance</option></select></label>
      <button className="btn delivery-refresh" disabled={busy} onClick={onRefresh}>Refresh</button>
    </div>
    {spinning?<div className="delivery-skeleton" aria-label="Loading deployments">{[1,2].map(n=><div key={n}/>)}</div>:!env?<div className="delivery-warning"><p>{failure}</p><button className="btn" onClick={state.moved?onFollow:onRefresh}>{state.moved?"Follow folder":"Retry"}</button></div>:<>
      <p className="delivery-guide" role="status">{model.facts[0]}</p>
      {model.headline?<p className="delivery-headline">{model.headline}</p>:null}
      {model.action?.kind==="setup"?<div className="delivery-env-row is-action" data-align-row><a className="btn" href={SETUP_URL} target="_blank" rel="noreferrer">{model.action.label}</a></div>:null}
      {model.unpublished?<div className="delivery-env-row" data-align-row><span className="delivery-env-fact">{model.unpublished.text}</span><button className="btn" onClick={()=>onLane?.("integration")}>View changes</button></div>:null}
      {model.attempt?<div className="delivery-env-row" data-align-row data-state={model.attempt.outcome}><span className="delivery-env-fact">{model.attempt.text}{model.attempt.error?" · "+model.attempt.error:""}</span></div>:connected?<p className="delivery-guide">No deployment recorded for this project yet.</p>:null}
      {model.busy?<div className="delivery-env-row" data-align-row><span className="delivery-env-fact">{model.busy.text}</span><a className="btn" href="#/">View activity</a></div>:null}
      {connected?<details className="delivery-evidence-fold"><summary>View evidence</summary><dl className="delivery-pairs"><dt>Running revision</dt><dd><code>{env.revision||"Not recorded"}</code></dd><dt>Instance boot</dt><dd><code>{env.boot||"Not recorded"}</code></dd><dt>Artifact</dt><dd>{env.artifact}</dd>{env.lastAttempt?<><dt>Attempt</dt><dd><code>{env.lastAttempt.id}</code> · {env.lastAttempt.outcome}</dd><dt>Attempt source</dt><dd><code>{env.lastAttempt.builtRevision||"Not recorded"}</code>{env.lastAttempt.clean?" · clean checkout":""}</dd></>:null}</dl>{env.issues?.length?<ul className="delivery-evidence">{env.issues.map((x,i)=><li key={i}><span className="delivery-meta">{x}</span></li>)}</ul>:null}</details>:null}
    </>}
  </>;
}
