import { Alert as Dialog } from "./MobileSheet.jsx";

export default function FileLeaveDialog({ open, path, onPick }) {
  return <Dialog.Root open={open} onOpenChange={value => { if (!value) onPick("cancel"); }}>
    <Dialog.Portal>
      <Dialog.Overlay className="dlg-overlay" />
      <Dialog.Content className="dlg m-file-leave" onCloseAutoFocus={event => event.preventDefault()}>
        <Dialog.Title className="dlg-title">Save changes?</Dialog.Title>
        <Dialog.Description className="dlg-body">{path} has unsaved changes.</Dialog.Description>
        <div className="dlg-actions" data-align-row>
          <button type="button" className="btn btn-sm btn-ghost" onClick={() => onPick("cancel")}>Cancel</button>
          <button type="button" className="btn btn-sm btn-danger" onClick={() => onPick("discard")}>Discard</button>
          <button type="button" className="btn btn-sm btn-primary" onClick={() => onPick("save")}>Save</button>
        </div>
      </Dialog.Content>
    </Dialog.Portal>
  </Dialog.Root>;
}
