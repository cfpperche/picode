import { useEffect, useState } from "react";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { api } from "@picode/shared/client/api.js";
import { CLI_SETTINGS, cliSettingsHash, cliSettingsLocation, supportsCliSettings, loadPiSettingsContext } from "@picode/shared/domain/cliSettings.js";
import PageFrame from "./PageFrame.jsx";
import CliTabs from "./CliTabs.jsx";
import PiSettings from "./PiSettings.jsx";

const EDITORS = { pi: { Editor: PiSettings, loadContext: loadPiSettingsContext } };

export default function CliSettings({ hidden, hash, legacyAgentId, catalog, onAgentConfig }) {
  const route = cliSettingsLocation(hash, legacyAgentId);
  useEffect(() => {
    if (!hidden && route.redirect) location.replace(route.redirect);
  }, [hidden, route.redirect]);
  const supported = supportsCliSettings(route.id);
  return <PageFrame id="agent-clis-view" title="Agent CLIs" hidden={hidden} wide>
    <CliTabs view="settings" />
    <div className="cli-settings-body">
    <div className="cli-settings-heading">
      <h3>Settings</h3>
      {supported ? <label className="cli-settings-picker">CLI
        <select aria-label="Settings CLI" value={route.id} onChange={event => { location.hash = cliSettingsHash(event.target.value); }}>
          {CLI_SETTINGS.map(cli => <option key={cli.id} value={cli.id}>{cli.name}</option>)}
        </select>
      </label> : null}
    </div>
    {!supported ? <div className="cli-notice" role="status"><span>Settings are not available for this CLI.</span><a className="btn btn-ghost btn-sm" href="#/clis">Back to CLIs</a></div>
      : !hidden && !route.redirect ? <SettingsEditor key={route.id + ":" + route.agentId} route={route} catalog={catalog} onAgentConfig={onAgentConfig} /> : null}
    </div>
  </PageFrame>;
}

function SettingsEditor({ route, catalog, onAgentConfig }) {
  const [context, setContext] = useState(null);
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    if (!route.agentId) return;
    let active = true;
    setError("");
    EDITORS[route.id].loadContext(route.agentId, api).then(context => {
      if (active) setContext(context);
    }).catch(error => { if (active) setError(error.message); });
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
  if (error) return <div className="cli-notice is-error" role="alert"><span>{error}</span><button type="button" className="btn btn-ghost btn-sm" onClick={() => setRetry(value => value + 1)}>Try again</button><a className="btn btn-ghost btn-sm" href={cliSettingsHash(route.id)}>Global settings</a></div>;
  if (route.agentId && !context) return <div className="cli-loading" aria-label="Loading agent settings"><div /><div /><div /></div>;
  const saveAgent = async patch => {
    const current = await EDITORS[route.id].loadContext(route.agentId, api);
    await onAgentConfig(current.agent, patch);
    setContext({ ...current, agent: { ...current.agent, ...patch } });
    setRetry(value => value + 1);
  };
  return <>
    {context ? <p className="cli-settings-context"><a href={"#/agent/" + encodeURIComponent(context.agent.id)}>Back to agent</a><a href={cliSettingsHash(route.id)}>Global settings</a></p> : null}
    <Editor embedded hidden={false} agent={context?.agent} workspace={context?.workspace} catalog={catalog} focus={route.focus} onAgentConfig={saveAgent} />
  </>;
}
