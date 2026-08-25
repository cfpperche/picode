import * as Dialog from "@radix-ui/react-dialog";

const KEYS = [
  ["Ctrl+K", "Command palette"],
  ["/", "Commands, skills, and templates"],
  ["Enter", "Send"],
  ["Shift+Enter", "New line"],
  ["Esc", "Close overlay"],
  ["↑ / ↓", "Prompt history"],
];

export default function Hotkeys({ open, onClose }) {
  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Shortcuts</Dialog.Title>
          <Dialog.Description className="dlg-body">PiCode keys on this machine.</Dialog.Description>
          <ul className="hotkey-list">
            {KEYS.map(([k, d]) => (
              <li key={k} className="hotkey-row">
                <kbd>{k}</kbd>
                <span>{d}</span>
              </li>
            ))}
          </ul>
          <div className="dlg-actions">
            <button type="button" className="btn btn-primary btn-sm" onClick={onClose}>Close</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
