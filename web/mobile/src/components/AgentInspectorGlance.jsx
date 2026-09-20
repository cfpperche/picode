import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { gitURL } from "../lib/git/model.js";
import { useGitRead } from "../lib/git/useGitRead.js";
import { IconChevronRight } from "./Icons.jsx";

// One honest line on the agent screen: what this agent's folder looks like
// right now and whether the branch has a pull request. Rendered only while
// there is something to review — no changes and no PR means no chrome. It
// loads with the screen, reloads on the shared feed and on focus, and never
// polls on its own (ADR-0048).
export default function AgentInspectorGlance({ agentId, onOpen }) {
  const [root, setRoot] = useState("");
  const [status, setStatus] = useState(null);
  const rootRef = useRef("");
  const loadRef = useRef(null);
  const request = useRef(null);

  const load = useCallback(async () => {
    if (!agentId) return;
    request.current?.abort();
    const controller = new AbortController();
    request.current = controller;
    try {
      const page = await api(gitURL({ kind: "agent", id: agentId }, "browse", { root: rootRef.current }), { signal: controller.signal });
      const next = await api(gitURL({ kind: "agent", id: agentId }, "gitstatus", { root: page.root }), { signal: controller.signal });
      if (controller.signal.aborted) return;
      rootRef.current = page.root; setRoot(page.root); setStatus(next);
    } catch {
      if (!controller.signal.aborted) setStatus(null);
    }
  }, [agentId]);
  loadRef.current = load;
  useEffect(() => { setRoot(""); rootRef.current = ""; setStatus(null); load(); return () => request.current?.abort(); }, [load]);

  useEffect(() => {
    let timer;
    const refresh = () => {
      if (!rootRef.current || document.hidden) return;
      clearTimeout(timer);
      timer = setTimeout(() => loadRef.current?.(), 150);
    };
    const unsub = subscribeFeed(event => {
      if (event.type === "feed.open" || event.type === "feed.reset" || (event.type === "git.updated" && event.data?.path === rootRef.current)) refresh();
    });
    window.addEventListener("focus", refresh);
    document.addEventListener("visibilitychange", refresh);
    return () => { clearTimeout(timer); unsub(); window.removeEventListener("focus", refresh); document.removeEventListener("visibilitychange", refresh); };
  }, []);

  const prRead = useGitRead(root && status?.git ? gitURL({ kind: "agent", id: agentId }, "pr", { root }) : "", 0, null, !root || !status?.git);
  if (!status?.git) return null;
  const files = (status.changes || []).length + (status.worktrees || []).reduce((n, wt) => n + ((wt.changes || []).length), 0);
  const pr = prRead.data?.status === "ok" ? prRead.data.pr : null;
  if (!files && !pr) return null;
  const add = (status.totals?.add || 0) + (status.worktrees || []).reduce((n, wt) => n + (wt.totals?.add || 0), 0);
  const del = (status.totals?.del || 0) + (status.worktrees || []).reduce((n, wt) => n + (wt.totals?.del || 0), 0);
  return <button type="button" className="m-insp-glance" onClick={() => onOpen(root)}>
    <span className="m-insp-glance-main">
      {files ? <><strong>{files}</strong>&nbsp;changed {files === 1 ? "file" : "files"}{add || del ? <> · <span className="m-git-add">+{add}</span> <span className="m-git-del">−{del}</span></> : null}</> : null}
      {files && pr ? " · " : null}
      {pr ? <><strong>PR #{pr.number}</strong>{pr.checks?.failed ? <> · {pr.checks.failed} check{pr.checks.failed === 1 ? "" : "s"} failing</> : pr.draft ? " · draft" : null}</> : null}
    </span>
    <span className="m-insp-glance-go">Review<IconChevronRight size={13} /></span>
  </button>;
}
