import { useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { cliPackagesHash, supportsCliPackages, loadPiPackagesContext, packageContextKey } from "@picode/shared/domain/cliPackages.js";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import Packages from "./Packages.jsx";
import { askConfirm } from "../lib/confirm.js";
import PackagesConfig from "./PackagesConfig.jsx";
import PackageConfigGeneric from "./PackageConfigGeneric.jsx";
import PackageDescribe from "./PackageDescribe.jsx";

export default function CliPackages({ hidden, route, catalog, onPackageUpdates, describe = false }) {
  const supported = supportsCliPackages(route.id);
  return <section id="cli-packages-view" hidden={hidden}>
    {route.invalid || !supported ?
      <div className="cli-notice" role="status"><span>{route.invalid ? "This package link is invalid." : "Packages are not available for this CLI."}</span><a className="btn btn-ghost btn-sm" href={cliPackagesHash("pi")}>{route.invalid ? "All packages" : "Open Pi packages"}</a></div>
      : !hidden ? <PackagesTarget key={route.id + ":" + route.workspaceId + ":" + route.agentId} route={route} catalog={catalog} describe={describe} onUpdates={(updates, workspaceId) => onPackageUpdates?.(updates, workspaceId)} /> : null}
  </section>;
}

function PackagesTarget({ route, catalog, onUpdates, describe = false }) {
  const [context, setContext] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [retry, setRetry] = useState(0);
  const live = useRef(true);
  const targetKey = useRef(null);
  function validateTarget(next) {
    const key = packageContextKey(next);
    if (targetKey.current !== null && targetKey.current !== key) throw Object.assign(new Error("The package location changed. Reload before editing."), { status: 409 });
    targetKey.current = key;
  }
  const blocked = useRef(false);
  blocked.current = !!error;
  useEffect(() => { live.current = true; return () => { live.current = false; }; }, []);
  useEffect(() => {
    let active = true;
    setLoading(true);
    loadPiPackagesContext(route, api).then(next => {
      if (active) { validateTarget(next); setContext(next); setError(null); }
    }).catch(err => {
      if (active) { setError(err); if (err.status === 404) setContext(null); }
    }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [route.workspaceId, route.agentId, route.scope, retry]);
  useEffect(() => {
    if (!route.workspaceId && !route.agentId) return;
    let timer;
    const unsubscribe = subscribeFeed(event => {
      if (/^(agent|workspace)\.(updated|deleted)$|^feed\.(open|reset)$/.test(event.type)) {
        clearTimeout(timer); timer = setTimeout(() => setRetry(value => value + 1), 80);
      }
    });
    return () => { clearTimeout(timer); unsubscribe(); };
  }, [route.workspaceId, route.agentId]);
  async function beforeMutation() {
    if (!live.current || blocked.current) throw new Error("Reload the package context before making changes.");
    try {
      const next = await loadPiPackagesContext(route, api);
      validateTarget(next);
      if (!live.current || blocked.current) throw new Error("The package context changed. Try again.");
    } catch (err) {
      if (live.current) { setError(err); if (err.status === 404) setContext(null); }
      throw err;
    }
  }
  async function recover() {
    if (error?.status !== 409) { setRetry(value => value + 1); return; }
    if (await askConfirm({ title: "Reload package context", message: "Discard local changes and reload the package location?", confirmLabel: "Reload" })) location.reload();
  }
  const notice = error ? <div className="cli-notice is-error" role="alert"><span>{error.message}</span><button type="button" className="btn btn-ghost btn-sm" disabled={loading} onClick={recover}>{loading ? "Retrying…" : error.status === 409 ? "Reload page" : "Try again"}</button><a className="btn btn-ghost btn-sm" href={cliPackagesHash(route.id)}>Machine packages</a></div> : null;
  if (!context) return notice || <div className="cli-loading" aria-label="Loading package context"><div /><div /><div /></div>;
  const workspace = context.workspace, agent = context.agent;
  const listHash = cliPackagesHash(route.id, { ...route, pkg: "" });
  const props = { embedded: true, hidden: false, workspaceId: workspace?.id || "", workspaceName: workspace?.name || "", workspacePath: workspace?.path || "", agentId: agent?.id || "", agentName: agent ? displayAgentName(agent, workspace) : "", beforeMutation, onUpdates: updates => { if (live.current) onUpdates(updates, workspace?.id || ""); } };
  return <>
    {notice}
    <fieldset className="cli-packages-fields" disabled={!!error}>
      {route.pkg && describe ? <PackageDescribe {...props} pkg={route.pkg} backHash={listHash} configHashFor={p => cliPackagesHash(route.id, { ...route, pkg: p })} /> :
        route.pkg ? (route.pkg === "pi-roles" ?
        <PackagesConfig {...props} pkg={route.pkg} catalog={catalog} backHash={listHash} initialScope={route.scope === "agent" ? "agent" : "workspace"} onScopeChange={scope => { location.hash = cliPackagesHash(route.id, { ...route, scope: scope === "agent" ? "agent" : "project" }); }} />
        : <PackageConfigGeneric {...props} pkg={route.pkg} backHash={listHash} describeHash={p => cliPackagesHash(route.id, { ...route, pkg: p }) + "&describe=1"} listHash={listHash} />)
        : <Packages {...props} scope={route.scope} onScopeChange={scope => { location.hash = cliPackagesHash(route.id, { ...route, scope }); }} configHash={pkg => cliPackagesHash(route.id, { ...route, pkg })} describeHash={pkg => { const base = cliPackagesHash(route.id, { ...route, pkg }); return base + (base.includes("?") ? "&" : "?") + "describe=1"; }} />}
    </fieldset>
  </>;
}
