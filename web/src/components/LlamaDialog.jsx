import * as Dialog from "@radix-ui/react-dialog";
import LlamaPanel from "./LlamaPanel.jsx";

export default function LlamaDialog({ open, onClose, onRefresh }) {
  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create dlg-llama" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">llama.cpp</Dialog.Title>
          <Dialog.Description className="dlg-body">Load and download GGUF on the local router.</Dialog.Description>
          {open ? <LlamaPanel hideTitle onRefresh={onRefresh} /> : null}
          <div className="dlg-actions">
            <button type="button" className="btn btn-primary btn-sm" onClick={onClose}>Close</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
