### Added

- **Link two Matrix panels, and the two sessions can message each other
  (ADR-0116).** On a matrix canvas, hover a panel and drag the connector in
  its header onto another panel. The line grants exactly one thing: those
  two sessions gain each other as a contact in PiCode's existing messaging
  (ADR-0104). One can send the other a message and read the replies — that
  is all it does.
- **A link never grants a transcript.** It does not let one session read the
  other's history, scrollback, session file or anything typed into it. The
  supported way to get context out of another session is to ask it and let
  it answer in its own words. The guide says so in plain words.
- **Two sessions in different project folders can now be paired**, per pair,
  by hand. The workspace rule is unchanged and there is no owner-wide
  switch: the blast radius of a link is two named sessions you drew a line
  between.
- **Nothing is connected quietly.** Drawing a line to a session that is not
  connected offers the existing connection in one line, with what it grants;
  drawing across two project folders asks again, separately, naming both
  folders. Cancel at either point writes nothing at all — no link, no
  connection.
- **Removing the line removes the permission**, with nothing left over:
  PiCode derives who may message whom from the links that exist right now,
  so deleting the line, either panel, or the matrix revokes it immediately.
- **A link that grants nothing reads broken**, in amber and marked
  *Broken*, with the reason when you point at it — the connection was
  revoked, the session moved, the target is gone, or it was never connected.
  It is never a faded version of a working link.
- **Matrix links, in the Messages view.** Every link you have drawn,
  anywhere: both ends, the matrix it lives on, whether it works right now,
  and a Remove that revokes it exactly as the canvas does. It is not
  filtered by the folder picker, because links across folders are the ones
  worth seeing. Grid mode, which has no plane to draw on, shows a per-panel
  link count in the header and sends you here.
