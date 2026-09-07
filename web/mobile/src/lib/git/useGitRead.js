import { useEffect, useRef, useState } from "react";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { isMoved } from "./model.js";

// A refresh keeps its last good answer. A new owner/root/query has a new
// identity, so an old response can never fill a different Git screen.
export function useGitRead(url, nonce = 0, onMoved, paused = false) {
  const [state, setState] = useState({ data: null, error: "", loading: !!url });
  const currentURL = useRef("");
  const moved = useRef(onMoved);
  moved.current = onMoved;
  useEffect(() => {
    if (paused) return;
    if (!url) { currentURL.current = ""; setState({ data: null, error: "", loading: false }); return; }
    const request = new AbortController();
    const changed = currentURL.current !== url;
    currentURL.current = url;
    setState(s => ({ data: changed ? null : s.data, error: "", loading: true }));
    api(url, { signal: request.signal }).then(data => {
      if (!request.signal.aborted) setState({ data, error: "", loading: false });
    }).catch(error => {
      if (request.signal.aborted) return;
      if (isMoved(error)) { moved.current?.(error); setState(s => ({ ...s, error: "", loading: false })); return; }
      setState(s => ({ ...s, error: humanizeError(error.message || "Could not read Git data."), loading: false }));
    });
    return () => request.abort();
  }, [url, nonce, paused]);
  return state;
}
