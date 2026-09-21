import { blockNote, noteIsExternal, pickupLine } from "@picode/shared/domain/cliKeys.js";

// The Keyboard pane of a CLI whose map PiCode cannot edit yet (ADR-0174). One
// state line and one action: the vendor's key list for a CLI that refuses
// remapping, where its few editable keys live for one that keeps no map file at
// all, the docs for one whose adapter has not shipped. It never says "coming
// soon" — that is a promise about us dressed as a fact about the CLI.
//
// The pickup sentence only appears when there is a map to pick up: a refused
// CLI has nothing to reload, and a partial one is edited in Settings, where the
// pickup of that file is already stated per row.
export default function CliKeyboard({ row }) {
  const note = blockNote(row.id);
  const external = noteIsExternal(note);
  return (
    <section className="settings-section key-pane" data-cli={row.id}>
      <h3>Keyboard</h3>
      {row.state === "planned" && row.keymap !== "partial" ? <p className="settings-desc">{pickupLine(row.id)}</p> : null}
      <div className="cli-notice" role="status">
        <span>{note.line}</span>
        <a className="btn btn-ghost btn-sm" href={note.href} {...(external ? { target: "_blank", rel: "noreferrer" } : {})}>{note.action}</a>
      </div>
    </section>
  );
}
