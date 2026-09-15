import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { cliProvidersHash, supportsCliProviders } from "@picode/shared/domain/cliProviders.js";
import Providers from "./Providers.jsx";
import CustomEndpointPage from "./CustomEndpointPage.jsx";

export default function CliProviders({ hidden, cli = "pi", add = false, invalid = false, scoped = false, custom = "", customId = "", onCatalogChange }) {
  const supported = supportsCliProviders(cli);
  const blocked = invalid || scoped || !supported;
  return <section id="cli-providers-view" className="cli-providers-pane" hidden={hidden}>
    {blocked ? <div className="cli-notice" role="status"><span>{invalid ? "This provider link is invalid." : !supported ? "Providers are not available for this CLI." : "Accounts stay on this machine."}</span><a className="btn btn-ghost btn-sm" href={cliProvidersHash("pi")}>Open Pi</a></div>
      : !hidden ? <PiProviders wantAdd={add} custom={custom} customId={customId} onCatalogChange={onCatalogChange} /> : null}
  </section>;
}

function PiProviders({ wantAdd, custom, customId, onCatalogChange }) {
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
    {error ? <div className="cli-notice is-error" role="alert"><span>{error}</span><button type="button" className="btn btn-ghost btn-sm" disabled={loading} onClick={refresh}>{loading ? "Retrying…" : "Try again"}</button></div> : null}
    {!catalog && !error ? <div className="cli-loading" aria-label="Loading providers"><div /><div /><div /></div> : null}
    {catalog ? (custom
      ? <CustomEndpointPage catalog={catalog} onRefresh={refresh} editId={custom === "edit" ? customId : ""} />
      : <Providers embedded hidden={false} catalog={catalog} onRefresh={refresh} wantAdd={wantAdd} />) : null}
  </>;
}
