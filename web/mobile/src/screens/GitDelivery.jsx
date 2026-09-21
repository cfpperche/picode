import { useEffect, useRef, useState } from "react";
import { deliveryURL, observeDelivery } from "@picode/shared/client/delivery.js";
import { deliveryRows, deliveryReason, integrationLabel, validationLabel, observationLabel } from "@picode/shared/domain/delivery.js";
import "../styles/mobile-delivery.css";

export default function GitDelivery({ owner, hidden, onHistory, selection = "", onDetail }) {
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
  const data=state.data, rows=deliveryRows(data?.changes,filter), detail=data?.changes.find(c=>c.id===selected);
  return <section className="m-delivery" aria-label="Delivery" hidden={!!hidden}>
    <div className="m-delivery-toolbar">
      <label data-align-row>Target <select aria-label="Delivery target" value={target||data?.target||""} onChange={e=>{setState({data:null,busy:true});setSelected("");setTarget(e.target.value);}}><option value="">Select target</option>{data?.targets.map(t=><option key={t} value={t}>{t}</option>)}</select></label>
      <div className="m-delivery-actions" data-align-row><select aria-label="Delivery filter" value={filter} onChange={e=>setFilter(e.target.value)}><option value="all">All changes</option><option value="attention">Needs attention</option></select>
      <button className="btn" disabled={state.busy||state.moved} onClick={()=>ref.current?.refresh()}>Refresh</button></div>
    </div>
    {!state.error ? <p className="m-delivery-status" role="status">{observationLabel(state)}</p> : null}
    {state.moved?<div className="m-delivery-warning"><p>This project's folder changed.</p><button className="btn" onClick={()=>{root.current="";setState({data:null,busy:true});setSelected("");setTarget("");setFollow(n=>n+1);}}>Follow folder</button></div>:null}
    {state.error&&!state.moved?<div className="m-delivery-warning"><p>{data?"Could not update; showing the last check.":"Could not read delivery status."}</p><button className="btn" onClick={()=>ref.current?.refresh()}>Retry</button></div>:null}
    {!data&&state.busy?<div className="m-delivery-skeleton" aria-label="Loading deliveries">{[1,2,3].map(n=><div key={n}/>)}</div>:null}
    {data&&!data.targetOid?<div className="m-delivery-empty"><p>Choose the branch changes will join.</p><button className="btn" onClick={()=>document.querySelector('[aria-label="Delivery target"]')?.focus()}>Select target</button></div>:null}
    {data?.targetOid&&!data.complete?<div className="m-delivery-warning"><p>Some changes could not be checked.</p><ul>{data.issues.map(x=><li key={x}>{x}</li>)}</ul><button className="btn" onClick={()=>ref.current?.refresh()}>Retry</button></div>:null}
    {data?.targetOid&&!rows.length?<div className="m-delivery-empty"><p>{filter==="attention"?"No changes match this filter.":"No observed changes for this branch."}</p><button className="btn" onClick={()=>filter==="attention"?setFilter("all"):onHistory()}>{filter==="attention"?"Clear filter":"View history"}</button></div>:null}
    {detail?<article className="m-delivery-detail"><button className="btn" onClick={()=>setSelected("")}>Back to deliveries</button><h3>{detail.title}</h3><p>{deliveryReason(detail)}</p><dl><dt>Revision</dt><dd>{detail.revision}</dd><dt>Source</dt><dd>{detail.registered?"Registered by an agent":"Observed in Git"} · {detail.branch}</dd><dt>Associated agents</dt><dd>{detail.agents.join(", ")||"Not recorded"}</dd><dt>Session</dt><dd>Not recorded</dd></dl><h4>Evidence</h4>{detail.evidence.length?detail.evidence.map(e=><p key={e.id}>{{scoped:"Change checks", "full-ci":"Project checks",land:"Integration attempt"}[e.kind]}: {e.outcome}<br/><time>{e.at||"Finish not recorded"}</time><br/><code>{e.source}</code></p>):<p>Checks have not been recorded.</p>}<button className="btn" onClick={()=>onHistory(detail.revision)}>Review changes</button></article>:null}
    {!detail?<ul className="m-delivery-list">{rows.map(c=><li key={c.id}><div><h3>{c.title}</h3><p className="m-delivery-meta">{c.branch} · {c.registered?"Registered delivery":"Observed branch"}</p><p>{deliveryReason(c)}</p><p className="m-delivery-meta">{c.agents.length?"Associated: "+c.agents.join(", "):"Agent association not recorded"}</p></div><div className="m-delivery-facts"><span>{integrationLabel[c.integration]}</span><span data-failed={c.validation==="failed"}>{validationLabel[c.validation]}</span><button className="btn" onClick={()=>setSelected(c.id)}>View details</button></div></li>)}</ul>:null}
  </section>;
}
