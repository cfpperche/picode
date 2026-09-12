import { memo, useCallback, useEffect, useLayoutEffect, useRef } from "react";
import Conversation from "../Conversation.jsx";
import { useAgentSocket } from "../../hooks/useAgentSocket.js";
import { stuckToBottom, pinToBottom, PANEL_PAD } from "@picode/shared/domain/stickScroll.js";

// AgentChatPanel — the loaded body of a managed agent's panel: the agent's
// live conversation, read-only (docs/plans/matrix-app.md §2.4, §4.4's
// `agent, managed, running` row). It is the tab's own `Conversation`, not a
// second conversation surface: the same turn list, the same tool cards, the
// same markdown, given the panel's height and nothing else. What it drops is
// everything that writes — no composer, no queue controls, no ask form, no
// snippet Run — because the one way to answer is **Open**, which takes the
// reader to the agent's tab.
//
// One scroller. `.conversation` is `position: absolute; inset: 0` over its
// own relative parent, so the panel body scrolls in exactly one place; it
// sticks to the bottom the way the tab's does (`stickScroll.js`: "at the
// bottom" is inside the last 320 px), and a reader who has scrolled up stays
// where they are while the turn streams. Another panel's socket cannot move
// this one — each mount has its own socket, its own scroller, and the
// wrapper is `React.memo`, so an event in one panel re-renders one panel.
//
// The **header** is not ours and never depends on this socket (plan §4.4):
// the chip is the fleet's (`agentRowStatus`), which is why a panel that is
// unloaded, quiet or a name-plate still says **Needs you**. What the socket
// adds down here is the question itself, and the bar that names the way out.
const AgentChatPanel = memo(function AgentChatPanel({ agentId, workspaceId, ask, onOpen, onOpenTab }) {
  const { state, scrollTick, ready } = useAgentSocket(agentId, workspaceId);
  const convRef = useRef(null);
  const nearBottom = useRef(true);
  const items = state.items;

  const onScroll = useCallback(() => {
    const el = convRef.current;
    // The tab's 320 px pad is the composer floating over it; a panel has no
    // composer and is a few hundred pixels tall, so it uses the panel pad —
    // otherwise every position reads as "the bottom" and a reader who
    // scrolled up is yanked back on the next delta.
    if (el) nearBottom.current = stuckToBottom(el, PANEL_PAD);
  }, []);

  // Before the paint, so a delta never shows the list jumping — and again on
  // the next frame, because a block that is still laying out (markdown, a
  // diff, a tool card) grows after the commit. The tab does the same.
  useLayoutEffect(() => {
    if (!nearBottom.current) return undefined;
    pinToBottom(convRef.current);
    const f = requestAnimationFrame(() => { if (nearBottom.current) pinToBottom(convRef.current); });
    return () => cancelAnimationFrame(f);
  }, [scrollTick, items]);
  // A panel mounts already at the end of the conversation — a reader
  // arriving at a live agent wants the last thing it said, not the first.
  useEffect(() => {
    nearBottom.current = true;
    pinToBottom(convRef.current);
  }, [agentId]);

  // The column keeps growing after the commit that pinned it — a code block
  // measuring itself, a diff, KaTeX arriving, the needs-you bar taking 45 px
  // out of the scroller — and content growth fires no scroll event, so
  // nothing would re-pin. The observer is the honest answer: while the
  // reader is at the bottom, stay there; the moment they scroll up,
  // `nearBottom` is false and this does nothing.
  const hasConv = items.length > 0;
  useEffect(() => {
    const el = convRef.current;
    const col = el && el.querySelector(".conv-col");
    if (!col || typeof ResizeObserver === "undefined") return undefined;
    const ro = new ResizeObserver(() => { if (nearBottom.current) pinToBottom(convRef.current); });
    ro.observe(col);
    return () => ro.disconnect();
  }, [hasConv, !!ask]);

  let body;
  if (!ready && items.length === 0) {
    body = (
      <div className="cv-skel" aria-busy="true">
        <span className="skel-line" />
        <span className="skel-line" />
        <span className="skel-line" />
      </div>
    );
  } else if (items.length === 0) {
    body = (
      <div className="cv-placeholder cv-state" role="status">
        <span>{state.status === "disconnected" ? "Disconnected — reconnecting." : "Nothing said in this session yet."}</span>
        <button type="button" className="btn btn-sm" onClick={onOpen}>Open</button>
      </div>
    );
  } else {
    body = (
      <Conversation
        // The tab owns the id `conversation`; a canvas may hold many of
        // these at once, and two elements may not share one id.
        id={null}
        readOnly
        items={items}
        streaming={state.streaming}
        agentId={agentId}
        convRef={convRef}
        onScroll={onScroll}
        onOpenTab={onOpenTab}
      />
    );
  }

  return (
    <div className="cv-chat">
      {body}
      {ask ? (
        // The reason this phase exists: an agent blocked on a human says so
        // where the reader is looking, without scrolling — and the one
        // action is the tab, which is where the question can be answered.
        <div className="cv-chat-ask" data-align-row>
          <span className="cv-chat-ask-t" title={ask}><span>{ask}</span></span>
          <button type="button" className="btn btn-sm btn-primary" onClick={onOpen}>Open</button>
        </div>
      ) : null}
    </div>
  );
});

export default AgentChatPanel;
