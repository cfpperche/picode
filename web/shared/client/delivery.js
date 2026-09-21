import { api } from "./api.js";
import { subscribeFeed } from "./feed.js";
import { ownerBase } from "../domain/gitOwner.js";
export function deliveryURL(owner, root = "", target = "") {
  const q = new URLSearchParams(); if(root) q.set("root",root); if(target) q.set("target",target);
  return ownerBase(owner)+encodeURIComponent(owner.id)+"/delivery?"+q;
}
// ADR-0170: Git commits and producer files do not reliably emit feed events.
// Only a visible Delivery pane runs this 15s reconciliation loop.
export function observeDelivery({ url, onChange, read = api, subscribe = subscribeFeed, doc = document, win = window }) {
  let live=true, inFlight=false, blocked=false, currentURL=url;
  let state={data:null,busy:true,error:false,moved:false,offline:false};
  const controller=new AbortController();
  const emit=patch=>{state={...state,...patch};if(live)onChange(state);};
  const refresh=async(fresh=false)=>{
    if(!live||inFlight||blocked||doc.hidden)return;
    inFlight=true;emit({busy:true});
    try {
      const data=await read(currentURL+(fresh?"&fresh=1":""),{signal:controller.signal});
      if(live) {
        const [path,search] = currentURL.split("?");
        const q = new URLSearchParams(search);
        if (!q.has("root") && data.root) q.set("root",data.root);
        if (!q.has("target") && data.target) q.set("target",data.target);
        currentURL=path+"?"+q;
        emit({data,busy:false,error:false,offline:false});
      }
    } catch(e) { if(live) {blocked=e.status===409;emit({busy:false,error:true,moved:blocked});} }
    finally{inFlight=false;}
  };
  const visible=()=>{if(!doc.hidden)refresh();};
  const off=subscribe(ev=>{
    if(ev.type==="feed.down"){emit({offline:true});return;}
    if(["feed.open","feed.reset","git.updated","delivery.changed","agent.updated","workspace.updated"].includes(ev.type))refresh();
  });
  const timer=win.setInterval(()=>{emit({});refresh();},15000);
  doc.addEventListener("visibilitychange",visible);win.addEventListener("focus",visible);
  refresh();
  return {refresh:()=>refresh(true),stop(){live=false;controller.abort();off();win.clearInterval(timer);doc.removeEventListener("visibilitychange",visible);win.removeEventListener("focus",visible);}};
}
