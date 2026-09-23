import { missionRequestReceipt } from "../domain/missions.js";
import { api } from "./api.js";
import { feedConnected, subscribeFeed } from "./feed.js";

// Shared data and mutation lifecycle; each shell owns its presentation.
export function createUseMissions({ useCallback, useEffect, useRef, useState }) {
  return function useMissions(id = "", workspace = "") {
    const [data, setData] = useState(null);
    const [workspaces, setWorkspaces] = useState([]);
    const [hasMore, setHasMore] = useState(false);
    const [moreBusy, setMoreBusy] = useState(false);
    const paging = useRef(false);
    const [error, setError] = useState("");
    const [busy, setBusy] = useState("");
    const [connected, setConnected] = useState(feedConnected);
    const pending = useRef(false), retry = useRef(null), epoch = useRef(0);
    const current = useRef(null);
    const receiptKey = "picode-mission-request:" + (id || "new");
    const keepReceipt = value => { retry.current = value; try { if (value) sessionStorage.setItem(receiptKey, JSON.stringify(value)); else sessionStorage.removeItem(receiptKey); } catch { /* optional local recovery */ } };
    const reload = useCallback(async () => {
      const ticket = ++epoch.current;
      try {
        const [result, ws] = await Promise.all([
          api(id ? "/api/missions/" + encodeURIComponent(id) : "/api/missions?workspace=" + encodeURIComponent(workspace)),
          api("/api/workspaces"),
        ]);
        if (ticket !== epoch.current) return;
        current.current = result.mission || null;
        const key = id ? "history" : "missions", rows = result[key] || [];
        setHasMore(rows.length > 100);
        setData({ ...result, [key]: rows.slice(0, 100) }); setWorkspaces(ws); setError("");
      } catch (e) { if (ticket === epoch.current) setError(e.message); }
    }, [id, workspace]);
    useEffect(() => { setData(null); current.current = null; try { retry.current = JSON.parse(sessionStorage.getItem(receiptKey)) || null; } catch { retry.current = null; } reload(); return () => { epoch.current++; }; }, [reload]);
    useEffect(() => subscribeFeed(ev => {
      if (ev.type === "feed.down") setConnected(false);
      if (ev.type === "feed.open") setConnected(true);
      if (ev.type.startsWith("mission.") || ev.type === "feed.open" || ev.type === "feed.reset" || ev.type === "workspace.updated" || ev.type === "agent.updated" || ev.type === "git.updated" || ev.type === "terminal.runtime" || ev.type === "agent.status") reload();
    }), [reload]);
    const mutate = useCallback(async (action, fields = {}) => {
      if (pending.current) return null;
      pending.current = true; setBusy(action); setError("");
      keepReceipt(missionRequestReceipt(id, action, fields, current.current?.version || 0, retry.current, () => crypto.randomUUID()));
      const payload = retry.current.payload;
      try {
        const result = await api(id ? "/api/missions/" + encodeURIComponent(id) + "/actions" : "/api/missions", {
          method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ ...payload, requestId: retry.current.requestId }),
        });
        keepReceipt(null);
        await reload();
        return result;
      } catch (e) {
        if (e.status && e.status < 500) keepReceipt(null);
        setError(e.message); return null;
      } finally { pending.current = false; setBusy(""); }
    }, [id, reload]);
    const loadMore = useCallback(async () => {
      const rows = id ? data?.history : data?.missions;
      if (paging.current || !hasMore || !rows?.length) return;
      paging.current = true; setMoreBusy(true);
      const ticket = epoch.current;
      const before = rows.at(-1).sequence;
      try {
        const result = await api(id ? `/api/missions/${encodeURIComponent(id)}?before=${before}` : `/api/missions?workspace=${encodeURIComponent(workspace)}&before=${before}`);
        if (ticket !== epoch.current) return;
        const key = id ? "history" : "missions", next = result[key] || [];
        setHasMore(next.length > 100);
        setData(old => ({ ...old, [key]: [...old[key], ...next.slice(0, 100)] }));
      } catch (e) { if (ticket === epoch.current) setError(e.message); }
      finally { paging.current = false; setMoreBusy(false); }
    }, [data, hasMore, id, workspace]);
    return { data, mission: data?.mission, workspaces, error, setError, busy, connected, hasMore, moreBusy, reload, mutate, loadMore };
  }

}
