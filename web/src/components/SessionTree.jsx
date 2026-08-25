import * as Dialog from "@radix-ui/react-dialog";
import { cardsFrom } from "../lib/sessionCards.js";

function Card({ card, leaf, onFork }) {
  const replies = card.info.filter((i) => i.kind === "reply");
  const tools = card.info.filter((i) => i.kind === "tool").length;
  const meta = card.info.filter((i) => i.kind === "meta").map((i) => i.text);
  const reply = replies.length ? replies[replies.length - 1].text : "";
  return (
    <li className="tree-chain-item">
      <button
        type="button"
        className={"tree-card" + (card.id === leaf ? " leaf" : "")}
        onClick={() => onFork(card.id)}
      >
        <span className="tree-card-prompt">{card.text}</span>
        {reply ? <span className="tree-card-reply">{reply}</span> : null}
        {tools || meta.length ? (
          <span className="tree-card-meta">
            {tools ? tools + (tools === 1 ? " tool" : " tools") : ""}
            {tools && meta.length ? " · " : ""}
            {meta.join(" · ")}
          </span>
        ) : null}
      </button>
      {card.children && card.children.length ? (
        <ol className="tree-chain">
          {card.children.map((c) => (
            <Card key={c.id} card={c} leaf={leaf} onFork={onFork} />
          ))}
        </ol>
      ) : null}
    </li>
  );
}

export default function SessionTree({ open, onClose, mode, tree, onFork, onClone }) {
  const cards = cardsFrom((tree && tree.tree) || []);
  const leaf = tree && tree.leafId;
  const forkOnly = mode === "fork";
  const title = forkOnly ? "Fork from a prompt" : "Session tree";

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-tree" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">
            {forkOnly ? "New session from that prompt." : "Each card is a prompt. Fork from a card. Clone copies this branch."}
          </Dialog.Description>
          <div className="tree-list">
            {cards.length === 0 ? (
              <p className="side-empty">No messages yet</p>
            ) : (
              <ol className="tree-chain">
                {cards.map((c) => (
                  <Card key={c.id} card={c} leaf={leaf} onFork={onFork} />
                ))}
              </ol>
            )}
          </div>
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Close</button>
            {!forkOnly ? (
              <button type="button" className="btn btn-primary btn-sm" onClick={onClone} disabled={cards.length === 0}>Clone</button>
            ) : null}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
