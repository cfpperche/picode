// windowsSettings.js — the one OS screen PiCode opens on purpose.
//
// Settings ▸ Browser ▸ Password manager says plainly that PiCode does not
// store passwords or passkeys (it can turn saving on or off and delete what
// the profile kept). What Windows keeps — your face, finger or PIN, and the
// passkeys bound to them — belongs to Windows, so the row offers that screen
// instead of imitating it. The target is the documented URI and the shell
// command is the same `btab_open_external` the rest of the app uses, which
// hands over http(s) and `ms-settings:` only (desktop-shell/src/external.rs).

export const SIGNIN_TARGET = "ms-settings:signinoptions";
export const OPEN_EXTERNAL_COMMAND = "btab_open_external";

// shellInvoke is the capability check the rest of the app uses: no Tauri
// invoke means no shell, and the row is not offered at all.
export function shellInvoke(win) {
  const invoke = win?.__TAURI__?.core?.invoke;
  return typeof invoke === "function" ? invoke : null;
}

// openWindowsSignIn asks the shell to open Windows' own sign-in options.
// Returns "no-shell" | "asked" | "failed" — the caller says which, in one
// line, so a refusal is never swallowed.
export async function openWindowsSignIn(win) {
  const invoke = shellInvoke(win);
  if (!invoke) return "no-shell";
  try {
    await invoke(OPEN_EXTERNAL_COMMAND, { url: SIGNIN_TARGET });
    return "asked";
  } catch {
    return "failed";
  }
}
