import { useEffect, useState } from "react";
import AttachComposer from "./AttachComposer.jsx";
import { api } from "@picode/shared/client/api.js";
import { focusPane } from "../lib/termActions.js";
import { toast } from "../lib/toast.js";
import { createUseDeliveryModes } from "@picode/shared/client/useDeliveryModes.js";
import { deliveryNotice, deliveryPlaceholder } from "@picode/shared/domain/deliveryModes.js";

const useDeliveryModes = createUseDeliveryModes({ useEffect, useState });

const json = (body) => ({ method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

// The prompt door of ADR-0089, now opened on demand from the terminal's
// context menu instead of standing between the pane and the window edge.
// `seed` carries what the menu started it with — a selected line as the
// message, a larger selection as a staged text file (lib/termMenu.js).
// The composer itself (AttachComposer) is shared with Fork agent…; Send
// here stages each file in the terminal's folder and pastes the message.
export default function TermAttachBar({ term, seed, ownerKind, onClose }) {
  const agentId = ownerKind === "agent" ? term.id : "";
  const termId = ownerKind === "agent" ? "" : term.id;
  const dropBase = agentId
    ? "/api/agents/" + encodeURIComponent(agentId)
    : "/api/terminals/" + encodeURIComponent(termId);
  const [items, setItems] = useState([]);
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [sendError, setSendError] = useState("");
  // Steer / Follow-up while the CLI works (ADR-0206); hidden when idle.
  const { options, delivery, setDelivery } = useDeliveryModes(dropBase);

  async function send() {
    if (busy || (!text.trim() && !items.length)) return;
    setBusy(true);
    try {
      const paths = [];
      for (const it of items) {
        if (it.path) { paths.push(it.path); continue; }
        const d = await api(dropBase + "/drop", json({ name: it.name, mime: it.mime, data: it.data }));
        paths.push(d.path);
      }
      // No success toast: the user is looking at the terminal and sees the
      // message land. Toasts stay reserved for failures and for the one
      // receipt the pane cannot show: a prompt PiCode could not confirm.
      const res = await api(dropBase + "/prompt", json({ message: text, paths, delivery }));
      const notice = deliveryNotice(res, delivery);
      if (notice) toast.warn(notice);
      setItems([]);
      setText("");
      setSendError("");
      // The send landed: close the bar and hand the pane its keyboard back.
      // Failures return above through the catch, which keeps the bar open
      // with the text staged and the reason inside it.
      if (onClose) onClose();
      focusPane(term.id);
    } catch (e) {
      // Door refusals (working, occupied, busy, closed) name their fix in
      // the message — show them here, inside the card, where the Send
      // happened. A toast can lose the layering to the pane around it.
      setSendError(e && e.message ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <AttachComposer
      text={text}
      setText={setText}
      items={items}
      setItems={setItems}
      onSubmit={send}
      busy={busy}
      error={sendError}
      onClose={onClose}
      seed={seed}
      agentId={agentId}
      termId={termId}
      deliveryOptions={options}
      delivery={delivery}
      onDelivery={setDelivery}
      placeholder={deliveryPlaceholder(delivery, "Message the terminal")}
      sendLabel={delivery === "follow_up" ? "Queue" : "Send"}
    />
  );
}
