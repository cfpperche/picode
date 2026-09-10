import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";

export function useWorkspaceCommunication(hidden, workspace) {
  const [data, setData] = useState(null), [error, setError] = useState("");
  const [loadError, setLoadError] = useState("");
  const [loading, setLoading] = useState(true), [busy, setBusy] = useState("");
  const [history, setHistory] = useState([]), [hasOlder, setHasOlder] = useState(false);
  const active = useRef(false), version = useRef(0), historyVersion = useRef(0), writing = useRef(false);
  const refresh = useCallback(async (clearActionError = false) => {
    const seq = ++version.current;
    try {
      const next = await api("/api/communication/workspaces");
      if (!Array.isArray(next?.owners) || !Array.isArray(next?.participants) || !Array.isArray(next?.workspaces)) throw new Error("Could not load communication.");
      if (active.current && seq === version.current) { setData(next); setLoadError(""); if (clearActionError) setError(""); }
    } catch (e) { if (active.current && seq === version.current) setLoadError(e.message); }
    finally { if (active.current && seq === version.current) setLoading(false); }
  }, []);
  const readHistory = useCallback(async (before = "") => {
    if (!workspace) return;
    const seq = ++historyVersion.current;
    try {
      const next = await api(`/api/communication/workspaces/${encodeURIComponent(workspace)}/history${before ? `?before=${before}` : ""}`);
      if (!Array.isArray(next?.messages)) throw new Error("Could not load activity.");
      if (active.current && seq === historyVersion.current) { setHistory(old => before ? [...old, ...next.messages] : next.messages); setHasOlder(next.messages.length === 100); }
    } catch (e) { if (active.current && seq === historyVersion.current) setError(e.message); }
  }, [workspace]);
  useEffect(() => {
    if (hidden) return;
    active.current = true; refresh();
    const off = subscribeFeed(e => { if (/^(peer\.|terminal\.|agent\.|workspace\.|feed\.(open|reset))/.test(e.type)) { refresh(); readHistory(); } });
    return () => { active.current = false; version.current++; historyVersion.current++; off(); };
  }, [hidden, refresh, readHistory]);
  useEffect(() => { setHistory([]); setHasOlder(false); if (!hidden) readHistory(); }, [hidden, readHistory]);
  async function mutate(action, body) {
    if (writing.current || !workspace) return false;
    writing.current = true; setBusy(action); setError("");
    const generation = version.current;
    try {
      await api(`/api/communication/workspaces/${encodeURIComponent(workspace)}/${action}`, { method: action === "participants" ? "PUT" : "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
      if (active.current) { await refresh(); await readHistory(); }
      return true;
    } catch (e) { if (active.current && version.current >= generation) setError(e.message || "Could not complete this action."); return false; }
    finally { writing.current = false; if (active.current) setBusy(""); }
  }
  return { data, error: error || loadError, loading, busy, history, hasOlder, refresh, readHistory, mutate, setError };
}
