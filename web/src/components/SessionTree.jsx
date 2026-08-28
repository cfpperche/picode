import { useEffect, useRef } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { cardsFrom, leafUserId } from "../lib/sessionCards.js";

function flattenCards(cards, depth, out) {
  for (const c of cards || []) {
    out.push({ ...c, depth });
    flattenCards(c.children, depth + 1, out);
  }
  return out;
}

function CardBody({ card, current }) {
  const replies = card.info.filter((i) => i.kind === "reply");
  const tools = card.info.filter((i) => i.kind === "tool").length;
  const meta = card.info.filter((i) => i.kind === "meta").map((i) => i.text);
  const reply = replies.length ? replies[replies.length - 1].text : "";
  return (
    <>
      <span className="tree-card-prompt">{card.text}</span>
      {reply ? <span className="tree-card-reply">{reply}</span> : null}
      {current ? <span className="tree-card-now">Now</span> : null}
      {!current && (tools || meta.length) ? (
        <span className="tree-card-meta">
          {tools ? tools + (tools === 1 ? " tool" : " tools") : ""}
          {tools && meta.length ? " · " : ""}
          {meta.join(" · ")}
        </span>
      ) : null}
    </>
  );
}

function Card({ card, leaf, onFork }) {
  const current = card.id === leaf;
  return (
    <li className="flex gap-2.5">
      <div className="relative flex w-4 shrink-0 items-center justify-center self-stretch" aria-hidden="true">
        <span className="absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-[var(--border)]" />
        <span className={"relative z-[1] size-2 shrink-0 rounded-full " + (current ? "bg-accent" : "bg-[var(--border-strong)]")} />
      </div>
      <div className="min-w-0 flex-1 bg-[var(--bg-base)] py-2">
        {current ? (
          <div className="tree-card w-full leaf now">
            <CardBody card={card} current />
          </div>
        ) : (
          <button type="button" className="tree-card w-full" onClick={() => onFork(card.id)}>
            <CardBody card={card} current={false} />
          </button>
        )}
      </div>
    </li>
  );
}

export default function SessionTree({ open, onClose, mode, tree, onFork, onClone }) {
  const raw = (tree && tree.tree) || [];
  const cards = flattenCards(cardsFrom(raw), 0, []);
  const leaf = leafUserId(raw, tree && tree.leafId);
  const forkOnly = mode === "fork";
  const title = forkOnly ? "Continue from a prompt" : "Prompts";
  const listRef = useRef(null);
  useEffect(() => {
    if (!open) return;
    const el = listRef.current && listRef.current.querySelector(".tree-card.now");
    if (el) el.scrollIntoView({ block: "nearest" });
  }, [open, leaf, cards.length]);

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-tree" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">
            This session stays. A new one starts from the prompt you pick.
          </Dialog.Description>
          <div className="tree-list" ref={listRef}>
            {cards.length === 0 ? (
              <p className="side-empty">No messages yet.</p>
            ) : (
              <ol className="m-0 flex list-none flex-col p-0">
                {cards.map((c) => (
                  <Card key={c.id} card={c} leaf={leaf} onFork={onFork} />
                ))}
              </ol>
            )}
          </div>
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Close</button>
            {!forkOnly ? (
              <button type="button" className="btn btn-primary btn-sm" onClick={onClone} disabled={cards.length === 0}>Duplicate</button>
            ) : null}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
