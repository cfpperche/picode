import { useEffect, useRef, useState } from "react";
import { deliveryURL, observeDelivery } from "@picode/shared/client/delivery.js";
import { deliveryRows, deliveryReason, integrationLabel, validationLabel, observationLabel, integrationExplanation, validationExplanation, missionsFor, missionLinks } from "@picode/shared/domain/delivery.js";
import { IconChevronLeft } from "./Icons.jsx";
import "../styles/delivery.css";

// A receipt's kind is stored, its label is presentation.
const EVIDENCE_KIND = { scoped: "Relevant checks", "full-ci": "Full project checks", land: "Integration attempt" };

// Delivery, the Git tab's second view, draws the same page a system route
// draws: .settings-wrap + .settings-head + .settings-card (the Agent CLIs
// standard, ADR-0103; owner 2026-09-21). The Git tab's own strip — the
// workspace picker and the History/Delivery toggle — stays where it is and
// outside the frame, because History is still a canvas and must not move.
// The objective a change serves: the mission cites the delivery as evidence, so
// this is the reverse of the link Missions stores (ADR-0199). Nothing here
// implies integration authority — the mission's state is the mission's own.
function objectiveLine(data, id, className) {
  const links = missionLinks(missionsFor(data, id));
  if (!links.length) return null;
  return <p className={className}>Serves {links.map((l, i) => <span key={l.id}>{i ? " · " : ""}<a href={l.href}>{l.title}</a></span>)}</p>;
}

function objectivePairs(data, id) {
  const links = missionLinks(missionsFor(data, id));
  return <><dt>Objective</dt><dd>{links.length ? links.map((l, i) => <span key={l.id}>{i ? " · " : ""}<a href={l.href}>{l.title}</a>{l.label ? " (" + l.label + ")" : ""}</span>) : "No mission cites this change"}</dd></>;
}

export default function Delivery({ owner, hidden, onHistory }) {
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
  return <section className="delivery-page" aria-label="Delivery" hidden={!!hidden}>
    <div className="settings-wrap">
      <header className="settings-head"><h2>Delivery</h2></header>
      <div className="settings-card">
        <div className="delivery-toolbar" data-align-row>
          <label data-align-row>Target <select aria-label="Delivery target" value={target||data?.target||""} onChange={e=>{setState({data:null,busy:true});setSelected("");setTarget(e.target.value);}}><option value="">Select target</option>{data?.targets.map(t=><option key={t} value={t}>{t}</option>)}</select></label>
          <select aria-label="Delivery filter" value={filter} onChange={e=>setFilter(e.target.value)}><option value="all">All changes</option><option value="attention">Needs attention</option></select>
          <button className="btn delivery-refresh" disabled={state.busy||state.moved} onClick={()=>ref.current?.refresh()}>Refresh</button>
        </div>
        <p className="delivery-guide" role="note"><strong>How to read this:</strong> Integrated means included in the target branch. Checks describe recorded test results. Neither status confirms publication.</p>
        {!state.error ? <p className="delivery-status" role="status">{observationLabel(state)}</p> : null}
        {state.moved?<div className="delivery-warning"><p>This project's folder changed.</p><button className="btn" onClick={()=>{root.current="";setState({data:null,busy:true});setSelected("");setTarget("");setFollow(n=>n+1);}}>Follow folder</button></div>:null}
        {state.error&&!state.moved?<div className="delivery-warning"><p>{data?"Could not update; showing the last check.":"Could not read delivery status."}</p><button className="btn" onClick={()=>ref.current?.refresh()}>Retry</button></div>:null}
        {!data&&state.busy?<div className="delivery-skeleton" aria-label="Loading deliveries">{[1,2,3].map(n=><div key={n}/>)}</div>:null}
        {data&&!data.targetOid?<div className="mcp-empty"><p>Choose the branch changes will join.</p><button className="btn" onClick={()=>document.querySelector('[aria-label="Delivery target"]')?.focus()}>Select target</button></div>:null}
        {data?.targetOid&&!data.complete?<div className="delivery-warning"><p>Some changes could not be checked.</p><ul>{data.issues.map(x=><li key={x}>{x}</li>)}</ul><button className="btn" onClick={()=>ref.current?.refresh()}>Retry</button></div>:null}
        {data?.targetOid&&!rows.length?<div className="mcp-empty"><p>{filter==="attention"?"No changes match this filter.":"No observed changes for this branch."}</p><button className="btn" onClick={()=>filter==="attention"?setFilter("all"):onHistory()}>{filter==="attention"?"Clear filter":"View history"}</button></div>:null}
        {detail?<article className="delivery-detail"><button className="delivery-back" onClick={()=>setSelected("")}><IconChevronLeft size={13} />Back to deliveries</button><h3>{detail.title}</h3><p>{deliveryReason(detail)}</p><div className="delivery-summary"><div><strong>{integrationLabel[detail.integration]}</strong><span>{integrationExplanation(detail.integration)}</span></div><div><strong>{validationLabel[detail.validation]}</strong><span>{validationExplanation(detail.validation)}</span></div></div><dl className="delivery-pairs">{objectivePairs(data,detail.id)}<dt>Revision</dt><dd><code>{detail.revision}</code></dd><dt>Source</dt><dd>{detail.registered?"Registered by an agent":"Observed in Git"} · {detail.branch}</dd><dt>Associated agents</dt><dd>{detail.agents.join(", ")||"Association not recorded"}</dd></dl><h4 className="delivery-heading">Recorded evidence</h4>{detail.evidence.length?<ul className="delivery-evidence">{detail.evidence.map(e=><li key={e.id}><strong>{EVIDENCE_KIND[e.kind]||e.kind}: {e.outcome}</strong><span className="delivery-meta"><time>{e.at||"Finish not recorded"}</time> · <code>{e.source}</code></span></li>)}</ul>:<p>{validationExplanation(detail.validation)}</p>}<button className="btn" onClick={()=>onHistory(detail.revision)}>Open history</button></article>:null}
        {!detail?<ul className="delivery-list">{rows.map(c=><li key={c.id}><div><h3>{c.title}</h3><p className="delivery-meta">{c.branch} · {c.registered?"Registered delivery":"Observed branch"}</p><p>{deliveryReason(c)}</p><p className="delivery-meta">{c.agents.length?"Associated: "+c.agents.join(", "):"Agent association not recorded"}</p>{objectiveLine(data,c.id,"delivery-meta")}</div><div className="delivery-facts"><span>{integrationLabel[c.integration]}</span><span data-failed={c.validation==="failed"}>{validationLabel[c.validation]}</span><button className="btn" onClick={()=>setSelected(c.id)}>View details</button></div></li>)}</ul>:null}
      </div>
    </div>
  </section>;
}
