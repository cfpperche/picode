import { createRoot } from "react-dom/client";
import { overlayAudit } from "@picode/shared/domain/overlayAudit.js";
import Git from "../../src/screens/Git.jsx";
import "../../src/index.css";
import "../../src/mobile.css";

window.__picodeOverlayAudit = overlayAudit;
async function boot() {
const fleet = await fetch('/api/workspaces').then(r=>r.json());
const workspace = fleet.find(w=>w.name==='picode');
window.qaGit = { deliveries: [], fail: false, delay: false };
let mount = 0;
window.renderGit = function ({ workspaces = fleet, owner = {kind:'workspace',id:workspace.id}, root = '', initialCommit = '' } = {}) {
 const props = {owner,root,initialCommit,title:workspace.name,workspaces,freeAgents:[],onBack:()=>{window.qaGit.back=true},onOpenFile:file=>{window.qaGit.file=file},onOpenTerminal:async (...args)=>{qaGit.deliveries.push({channel:'terminal',args});if(qaGit.delay)await new Promise(resolve=>qaGit.release=resolve);if(qaGit.fail)throw Error('QA channel unavailable');return{mode:'prepared'}},onAskAgent:async(...args)=>{qaGit.deliveries.push({channel:'agent',args});if(qaGit.delay)await new Promise(resolve=>qaGit.release=resolve);if(qaGit.fail)throw Error('QA channel unavailable');return{busy:true}}};
 app.render(<div id="m-app"><Git key={++mount} {...props}/></div>);
};
const app = createRoot(document.getElementById('root'));
renderGit();

}
boot();
