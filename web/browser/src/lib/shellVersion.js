// The shell's side of the client handshake (ADR-0216). /desktop/ is served
// by the daemon, so it is always as new as the daemon; the Windows shell
// that hosts it may be older. The shell announces itself through an
// initialization script (desktop-shell/src/main.rs, SHELL_PROTOCOL):
// `window.__PICODE_SHELL__ = { version, protocol }`.
//
// A feature that calls a shell command added after protocol 1 asks
// shellSupports(n) first and hides when it is false — instead of failing
// with "not allowed by ACL" on an older shell.
//
// What each protocol brought (bump desktop-shell SHELL_PROTOCOL with it):
//   0 — a shell from before the handshake (no announcement): every command
//       that existed on 2026-09-24
//   1 — the announcement itself

export function shellInfo(win = globalThis.window) {
  if (!win || !win.__TAURI__) return null; // not inside the shell
  const s = win.__PICODE_SHELL__;
  if (!s || typeof s !== "object") return { version: "", protocol: 0 };
  const protocol = Number.isInteger(s.protocol) && s.protocol >= 0 ? s.protocol : 0;
  return { version: typeof s.version === "string" ? s.version : "", protocol };
}

// True inside a shell that speaks at least protocol n. Outside the shell
// there is nothing to call, so the answer is false.
export function shellSupports(n, win = globalThis.window) {
  const info = shellInfo(win);
  return !!info && info.protocol >= n;
}
