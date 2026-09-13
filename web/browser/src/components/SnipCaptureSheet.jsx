import * as Dialog from "./ResponsiveDialog.jsx";
import { Editor } from "./Snippets.jsx";
import { titleFromText } from "@picode/shared/domain/snipDraft.js";

// Capture (snippets v2, F3): text you already wrote becomes a snippet
// without a trip to the studio and without retyping. The sheet is the real
// editor — same validation, same placeholder table, same Save — opened over
// whatever you were doing. keepDraft is off: this text belongs to the sheet,
// not to the "new snippet" draft the studio may be holding. Content unmounts
// when the sheet closes, so the next capture starts from the new selection.
export default function SnipCaptureSheet({ open, text, onClose, onSaved }) {
  const body = String(text == null ? "" : text);
  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg snip-capture" aria-describedby={undefined}>
          <Dialog.Title className="sr-only">Save as snippet</Dialog.Title>
          <Dialog.Description className="sr-only">Review the snippet captured from your selection, then save it.</Dialog.Description>
          {open ? (
            <Editor
              prefill={{ title: titleFromText(body), body }}
              keepDraft={false}
              cancelLabel="Cancel"
              onCancel={onClose}
              onSaved={(id) => { if (onSaved) onSaved(id); }}
            />
          ) : null}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
