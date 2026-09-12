import { useEffect, useState } from "react";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { api } from "@picode/shared/client/api.js";
import { cliSettingsHash, supportsCliSettings, loadPiSettingsContext } from "@picode/shared/domain/cliSettings.js";
import PiSettings from "./PiSettings.jsx";

const EDITORS = { pi: { Editor: PiSettings, loadContext: loadPiSettingsContext } };

export default function CliSettings({ hidden, route, catalog, onAgentConfig }) {
  const supported = supportsCliSettings(route.id);
  return <section id="cli-settings-view" hidden={hidden}>
    {route.invalid || !supported ? <div className="cli-notice" role="status"><span>{route.invalid ? "This settings link is invalid." : "Settings are not available for this CLI."}</span><a className="btn btn-ghost btn-sm" href={cliSettingsHash("pi")}>Open Pi settings</a></div>
      : !hidden ? <SettingsEditor key={route.id + ":" + route.agentId} route={route} catalog={catalog} onAgentConfig={onAgentConfig} /> : null}
  </section>;
}

function SettingsEditor({ route, catalog, onAgentConfig }) {
  const [context, setContext] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(!!route.agentId);
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    if (!route.agentId) return;
    let active = true;
    setLoading(true);
    EDITORS[route.id].loadContext(route.agentId, api).then(context => {
      if (active) { setContext(context); setError(null); }
    }).catch(error => {
      if (!active) return;
      setError(error);
      if (error.status === 404) setContext(null);
    }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [route.id, route.agentId, retry]);
  useEffect(() => {
    if (!route.agentId) return;
    let timer;
    const unsubscribe = subscribeFeed(event => {
      if (/^agent\.(updated|deleted|status)$/.test(event.type) && (event.agentId === route.agentId || event.data?.id === route.agentId)
        || /^(workspace\.(updated|deleted)|feed\.(open|reset))$/.test(event.type)) {
        clearTimeout(timer);
        timer = setTimeout(() => setRetry(value => value + 1), 80);
      }
    });
    return () => { clearTimeout(timer); unsubscribe(); };
  }, [route.agentId]);
  const { Editor } = EDITORS[route.id];
  const notice = error ? <div className="cli-notice is-error" role="alert"><span>{error.message}</span><button type="button" className="btn btn-ghost btn-sm" disabled={loading} onClick={() => setRetry(value => value + 1)}>{loading ? "Retrying…" : "Try again"}</button><a className="btn btn-ghost btn-sm" href={cliSettingsHash(route.id)}>Global settings</a></div> : null;
  if (route.agentId && !context) return notice || <div className="cli-loading" aria-label="Loading agent settings"><div /><div /><div /></div>;
  // The layer and the sub-tab live on the route (docs/plans/cli-settings-ux.md):
  // a reload, a bookmark or another pane's link lands on the same view. Moving
  // off a deep link drops its focus — the row has been shown.
  const writeRoute = (next) => {
    // Read the hash, not the route prop: it lags one render behind a pill
    // click, and a tab click in that window must not drop the layer.
    const q = new URLSearchParams(location.hash.split("?")[1] || "");
    location.hash = cliSettingsHash(route.id, {
      agentId: next.agentId ?? q.get("agentId") ?? route.agentId,
      layer: next.layer ?? q.get("layer") ?? route.layer,
      tab: next.tab ?? q.get("tab") ?? route.tab,
    });
  };
  const saveAgent = async patch => {
    const current = await EDITORS[route.id].loadContext(route.agentId, api);
    try {
      await onAgentConfig(current.agent, patch);
      setContext({ ...current, agent: { ...current.agent, ...patch } });
    } finally { setRetry(value => value + 1); }
  };
  return <>
    {notice}
    <Editor embedded disabled={!!error} hidden={false} agent={context?.agent} workspace={context?.workspace} catalog={catalog} focus={route.focus} layer={route.layer} tab={route.tab} onLayerChange={(layer) => writeRoute({ layer })} onTabChange={(tab) => writeRoute({ tab })} onAgentConfig={saveAgent} />
  </>;
}
