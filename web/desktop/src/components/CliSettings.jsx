import { useEffect, useState } from "react";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { api } from "@picode/shared/client/api.js";
import { cliSettingsHash, supportsCliSettings, loadPiSettingsContext } from "@picode/shared/domain/cliSettings.js";
import PiSettings from "./PiSettings.jsx";

const EDITORS = { pi: { Editor: PiSettings, loadContext: loadPiSettingsContext } };

export default function CliSettings({ hidden, route, catalog, onAgentConfig }) {
  const supported = supportsCliSettings(route.id);
  return <section id="cli-settings-view" hidden={hidden}>
    {!supported ? <div className="cli-notice" role="status"><span>Settings are not available for this CLI.</span><a className="btn btn-ghost btn-sm" href={cliSettingsHash("pi")}>Open Pi settings</a></div>
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
  const saveAgent = async patch => {
    const current = await EDITORS[route.id].loadContext(route.agentId, api);
    try {
      await onAgentConfig(current.agent, patch);
      setContext({ ...current, agent: { ...current.agent, ...patch } });
    } finally { setRetry(value => value + 1); }
  };
  return <>
    {notice}
    {context ? <p className="cli-settings-context"><a href={"#/agent/" + encodeURIComponent(context.agent.id)}>Back to agent</a><a href={cliSettingsHash(route.id)}>Global settings</a></p> : null}
    <Editor embedded disabled={!!error} hidden={false} agent={context?.agent} workspace={context?.workspace} catalog={catalog} focus={route.focus} onAgentConfig={saveAgent} />
  </>;
}
