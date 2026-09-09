import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";

export const peerOwnerKey = owner => `${owner.kind}:${owner.ownerId}`;
export function usePeerCommunication(hidden, ownerKey) {
  const [data, setData] = useState(null), [loadError, setLoadError] = useState("");
  const [errorCode, setErrorCode] = useState("");
  const [error, setError] = useState(""), [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true), [secret, setSecret] = useState(null);
  const [historyId, setHistoryId] = useState(""), [history, setHistory] = useState(null);
  const [historyError, setHistoryError] = useState("");
  const [reading, setReading] = useState(false), [hasOlder, setHasOlder] = useState(false);
  const live = useRef(false), generation = useRef(0), historyGeneration = useRef(0), mutating = useRef(false);
  const owner = data?.owners.find(o => peerOwnerKey(o) === ownerKey) || (!ownerKey ? data?.owners[0] : null);
  const connections = data?.connections.filter(p => p.kind === owner?.kind && p.ownerId === owner?.ownerId) || [];
  const active = connections.find(p => p.active);
  const selected = connections.find(p => p.id === historyId) || active || connections[0];
  const refresh = useCallback(async () => {
    const gen = ++generation.current;
    setLoading(true);
    try {
      const next = await api("/api/communication");
      if (!Array.isArray(next?.owners) || !Array.isArray(next?.connections)) throw new Error("Invalid communication response.");
      if (live.current && gen === generation.current) { setData(next); setLoadError(""); }
    } catch { if (live.current && gen === generation.current) setLoadError("Couldn’t refresh connections. Last results are still shown."); }
    finally { if (live.current && gen === generation.current) setLoading(false); }
  }, []);
  const readHistory = useCallback(async (older = false) => {
    if (!selected?.id) { setHistory(null); setHasOlder(false); return; }
    const gen = ++historyGeneration.current;
    setReading(true);
    try {
      const before = older && history?.length ? `?before=${history.at(-1).seq}` : "";
      const next = await api(`/api/communication/${encodeURIComponent(selected.id)}/messages${before}`);
      if (!Array.isArray(next?.messages)) throw new Error("Invalid message history.");
      if (live.current && gen === historyGeneration.current) {
        setHistory(prev => older ? [...(prev || []), ...next.messages] : next.messages); setHistoryError(""); setHasOlder(next.messages.length === 100);
      }
    } catch { if (live.current && gen === historyGeneration.current) setHistoryError("Couldn’t refresh messages. Last results are still shown."); }
    finally { if (live.current && gen === historyGeneration.current) setReading(false); }
  }, [selected?.id, history]);
  const readRef = useRef(readHistory); readRef.current = readHistory;
  useEffect(() => {
    if (hidden) { setSecret(null); return; }
    live.current = true; refresh();
    const off = subscribeFeed(ev => {
      if (/^(peer\.|agent\.|terminal\.|workspace\.|feed\.(open|reset))/.test(ev.type)) { refresh(); readRef.current(); }
    });
    return () => { live.current = false; generation.current++; historyGeneration.current++; off(); };
  }, [hidden, refresh]);
  useEffect(() => {
    historyGeneration.current++; setHistory(null); setHistoryError(""); setHasOlder(false); setReading(false);
    if (!hidden && selected?.id) readRef.current();
  }, [hidden, selected?.id]);
  // A connection revoked in another view must not leave its setup secret visible.
  useEffect(() => { if (secret && data && !data.connections.some(p => p.id === secret.connection.id && p.active)) setSecret(null); }, [data, secret]);
  async function mutate(action) {
    if (mutating.current || !owner || loadError || historyError) return;
    mutating.current = true; setBusy(true); setError(""); setErrorCode(""); setSecret(null);
    try {
      if (action === "enable") {
        const result = await api("/api/communication", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ kind: owner.kind, ownerId: owner.ownerId, sessionKey: owner.sessionKey, automatic: !!data?.launchCLIs?.includes(owner.cli) }) });
        if (live.current) {
          // Apply the receipt before exposing the secret; a feed refresh can race it.
          generation.current++;
          setData(prev => ({ ...prev, launches: { ...prev.launches, [result.connection.id]: !!result.automatic }, connections: [result.connection, ...prev.connections.filter(p => p.id !== result.connection.id).map(p => p.kind === owner.kind && p.ownerId === owner.ownerId ? { ...p, active: false } : p)] }));
          setHistoryId(result.connection.id); setSecret(result.automatic ? null : result);
        }
      } else if (active) {
        setData(prev => ({ ...prev, connections: prev.connections.map(p => p.id === active.id ? { ...p, active: false } : p) }));
        await api(`/api/communication/${encodeURIComponent(active.id)}`, { method: "DELETE" });
      }
    } catch (err) { if (live.current) { setError(err.message || "Couldn’t update this connection."); setErrorCode(err.body?.code || ""); } }
    finally { mutating.current = false; if (live.current) { setBusy(false); refresh(); } }
  }
  return { data, owner, connections, active, selected, history, reading, hasOlder, secret, setSecret, loading, busy, error, errorCode, loadError, historyError, refresh, readHistory, setHistoryId, mutate };
}
