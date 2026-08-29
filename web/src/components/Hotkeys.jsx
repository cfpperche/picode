import * as Dialog from "@radix-ui/react-dialog";
import { termShortcutRows } from "../lib/termKeys.js";

const APP = [
  ["Ctrl+K", "Command palette"],
  ["/", "Commands, skills, and templates"],
  ["Enter", "Send"],
  ["Shift+Enter", "New line in the composer"],
  ["Esc", "Close overlay"],
  ["↑ / ↓", "Prompt history"],
];

export default function Hotkeys({ open, onClose }) {
  const term = open ? termShortcutRows() : [];
  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create dlg-hotkeys" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Shortcuts</Dialog.Title>
          <Dialog.Description className="dlg-body">PiCode keys on this machine.</Dialog.Description>
          <div className="hotkey-body">
            <h3 className="hotkey-group">App</h3>
            <ul className="hotkey-list">
              {APP.map(([k, d]) => (
                <li key={k} className="hotkey-row">
                  <kbd>{k}</kbd>
                  <span>{d}</span>
                </li>
              ))}
            </ul>
            <h3 className="hotkey-group">Terminal</h3>
            <ul className="hotkey-list">
              {term.map((r) => (
                <li key={r.key} className="hotkey-row">
                  <kbd>{r.key}</kbd>
                  <span>{r.label}</span>
                </li>
              ))}
            </ul>
          </div>
          <div className="dlg-actions">
            <a className="btn btn-ghost btn-sm" href="#/preferences/terminal" onClick={onClose}>Terminal prefs</a>
            <button type="button" className="btn btn-primary btn-sm" onClick={onClose}>Close</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
