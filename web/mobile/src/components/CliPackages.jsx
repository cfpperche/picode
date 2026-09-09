import { useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { CLI_PACKAGES, cliPackagesHash, cliPackagesLocation, supportsCliPackages, loadPiPackagesContext, packageContextKey } from "@picode/shared/domain/cliPackages.js";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import PageFrame from "./PageFrame.jsx";
import CliTabs from "./CliTabs.jsx";
import CliCombo from "./CliCombo.jsx";
import Packages from "./Packages.jsx";
import { askConfirm } from "../lib/confirm.js";
import { setShell } from "@picode/shared/client/shell.js";

export default function CliPackages({ hidden, hash, legacyContext = {}, legacyContextReady = true, catalog, onPackageUpdates }) {
  const route = cliPackagesLocation(hash, legacyContext);
  const [hasUpdates, setHasUpdates] = useState(false);
  useEffect(() => { setHasUpdates(false); }, [route.id, route.workspaceId, route.agentId]);
  useEffect(() => {
    if (!hidden && legacyContextReady && route.redirect) location.replace(route.redirect);
  }, [hidden, route.redirect, legacyContextReady]);
  const supported = supportsCliPackages(route.id);
  return <PageFrame id="cli-packages-view" title="Agent CLIs" hidden={hidden} wide>
    <CliTabs hasPackageUpdates={hasUpdates} view="packages" packagesHref={cliPackagesHash(route.id, { ...route, pkg: "" })} />
    <div className="cli-settings-body">
      <div className="cli-settings-heading">
        <h3>Packages{route.pkg ? " · " + route.pkg : ""}</h3>
        {supported && !route.invalid ? <div className="cli-settings-picker">CLI
          <CliCombo ariaLabel="Packages CLI" value={route.id} options={CLI_PACKAGES} align="end" onChange={id => { location.hash = cliPackagesHash(id); }} />
        </div> : null}
      </div>
      {route.invalid || !supported || (route.pkg && route.pkg !== "pi-roles") ?
        <div className="cli-notice" role="status"><span>{route.invalid ? "This package link is invalid." : !supported ? "Packages are not available for this CLI." : "Configuration is not available for this package."}</span><a className="btn btn-ghost btn-sm" href={route.invalid || !supported ? "#/clis" : cliPackagesHash(route.id, { ...route, pkg: "" })}>{route.invalid || !supported ? "Back to CLIs" : "All packages"}</a></div>
        : route.legacy && !legacyContextReady ? <div className="cli-loading" aria-label="Loading package context"><div /><div /><div /></div> : !hidden && !route.redirect ? <PackagesTarget key={route.id + ":" + route.workspaceId + ":" + route.agentId} route={route} catalog={catalog} onUpdates={(updates, workspaceId) => { setHasUpdates(updates.length > 0); onPackageUpdates?.(updates, workspaceId); }} /> : null}
    </div>
  </PageFrame>;
}

function PackagesTarget({ route, catalog, onUpdates }) {
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
    {agent || workspace ? <p className="cli-settings-context">{agent ? <a href={"#/agent/" + encodeURIComponent(agent.id)}>Back to agent</a> : null}<a href={cliPackagesHash(route.id)}>Machine packages</a></p> : null}
    <fieldset className="cli-packages-fields" disabled={!!error}>
      {route.pkg ? <div className="cli-notice" role="status"><span>Package configuration is available in the desktop layout.</span><button type="button" className="btn btn-ghost btn-sm" onClick={() => setShell("desktop")}>Open desktop layout</button></div> : <Packages {...props} scope={route.scope} onScopeChange={scope => { location.hash = cliPackagesHash(route.id, { ...route, scope }); }} configHash={pkg => cliPackagesHash(route.id, { ...route, pkg })} />}
    </fieldset>
  </>;
}
