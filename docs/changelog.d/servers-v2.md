### Added
- **The Servers panel names what a port is, and can stop it.** Each row now
  carries what the port actually answered (`page`, `api`, or nothing yet), the
  scheme that answered, the process behind it and how long it has been up —
  and a menu with **Copy address**, **Show terminal**, **Stop server…** and
  **Hide from the list**. *Open* only appears where the URL leads somewhere a
  browser can show: an agent CLI's internal HTTPS control channel is no longer
  offered as a page it could never be (measured: plain HTTP to that port answers
  `400 Client sent an HTTP request to an HTTPS server`). A port that only
  answers an API, started outside PiCode, waits behind **Show N more**.
  Stop signals only a process PiCode can still re-derive inside one of its own
  panes, with the pid and the process's start token checked at that moment
  (ADR-0151); a port PiCode cannot attribute is never stoppable. Hiding is keyed
  by the listener — port, pid, start token — so a new server on the same port is
  born visible.

### Fixed
- **A stopped server no longer keeps its row.** The panel's cached probe answer
  was trusted for two minutes regardless of whether anything still listened, so
  a server that had just stopped stayed on screen (found in the first QA pass of
  the rework, minutes after Stop said it was gone). A row now needs the port to
  be listening at this read, unless a PiCode pane still owns the socket.
- **"unnamed page" beside "not a page".** A port that is not a page and has no
  title shows its port alone instead of borrowing the page vocabulary.
