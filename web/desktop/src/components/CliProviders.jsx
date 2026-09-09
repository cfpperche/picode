import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { CLI_PROVIDERS, cliProvidersHash, cliProvidersLocation, supportsCliProviders } from "@picode/shared/domain/cliProviders.js";
import AgentClisFrame from "./AgentClisFrame.jsx";
import CliTabs from "./CliTabs.jsx";
import CliCombo from "./CliCombo.jsx";
import Providers from "./Providers.jsx";

export default function CliProviders({ hidden, hash, onCatalogChange }) {
  const route = cliProvidersLocation(hash);
  useEffect(() => {
    if (!hidden && route.redirect) location.replace(route.redirect);
  }, [hidden, route.redirect]);
  const supported = supportsCliProviders(route.id);
  const blocked = route.invalid || route.scoped || !supported;
  return <AgentClisFrame id="cli-providers-view" hidden={hidden}>
    <CliTabs view="providers" />
    <div className="cli-settings-body">
      <div className="cli-settings-heading">
        <h3>Providers</h3>
        {supported && !route.invalid ? <div className="cli-settings-picker">CLI
          <CliCombo ariaLabel="Providers CLI" value={route.id} options={CLI_PROVIDERS} align="end" onChange={id => { location.hash = cliProvidersHash(id); }} />
        </div> : null}
      </div>
      {blocked ? <div className="cli-notice" role="status"><span>{route.invalid ? "This provider link is invalid." : !supported ? "Providers are not available for this CLI." : "Provider accounts are managed for this machine."}</span><a className="btn btn-ghost btn-sm" href={route.scoped && supported ? cliProvidersHash() : "#/clis"}>{route.scoped && supported ? "Machine providers" : "Back to CLIs"}</a></div>
        : !hidden && !route.redirect ? <PiProviders wantAdd={route.add} onCatalogChange={onCatalogChange} /> : null}
    </div>
  </AgentClisFrame>;
}

function PiProviders({ wantAdd, onCatalogChange }) {
  const [catalog, setCatalog] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const live = useRef(false);
  const sequence = useRef(0);
  const notify = useRef(onCatalogChange);
  notify.current = onCatalogChange;
  const refresh = useCallback(async () => {
    const request = ++sequence.current;
    setLoading(true);
    try {
      const next = await api("/api/catalog");
      if (!Array.isArray(next?.providers)) throw new Error("Invalid provider catalog.");
      if (!live.current || request !== sequence.current) return;
      setCatalog(next); setError(""); notify.current?.(next);
    } catch {
      if (live.current && request === sequence.current) setError("Couldn’t load providers.");
    } finally {
      if (live.current && request === sequence.current) setLoading(false);
    }
  }, []);
  useEffect(() => {
    live.current = true;
    refresh();
    window.addEventListener("focus", refresh);
    return () => { live.current = false; sequence.current++; window.removeEventListener("focus", refresh); };
  }, [refresh]);
  return <>
    <p className="cli-providers-scope">This machine</p>
    {error ? <div className="cli-notice is-error" role="alert"><span>{error}</span><button type="button" className="btn btn-ghost btn-sm" disabled={loading} onClick={refresh}>{loading ? "Retrying…" : "Try again"}</button></div> : null}
    {!catalog && !error ? <div className="cli-loading" aria-label="Loading providers"><div /><div /><div /></div> : null}
    {catalog ? <Providers embedded hidden={false} catalog={catalog} onRefresh={refresh} wantAdd={wantAdd} /> : null}
  </>;
}
