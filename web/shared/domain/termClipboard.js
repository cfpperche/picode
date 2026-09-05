// OSC 52 — "put this text on the clipboard" — for the browser terminal.
//
// xterm.js does nothing with OSC 52 on its own (xtermjs/xterm.js#3260), so a
// copy made inside a PiCode terminal never reaches the system clipboard. This
// registers the missing handler.
//
// Write only, on purpose. The obvious dependency, @xterm/addon-clipboard, also
// implements the *read* form (`OSC 52 ; c ; ?`), which answers with the user's
// clipboard contents. These terminals exist to run third-party coding agents;
// handing any process in the pane a way to read the clipboard is not a trade
// we make. The query form is consumed and left unanswered.
//
// See docs/benchmarks/2026-08-30-web-terminal-clipboard.md.

// A clipboard write larger than this is refused outright. Truncating would put
// a silently incomplete payload in the user's clipboard, which is worse than
// nothing — they would paste it somewhere believing it whole.
export const MAX_OSC52_BASE64 = 1 << 20; // 1 MiB of base64 ≈ 768 KiB of text

// Selections we honour. A browser has one clipboard: `c` is it, and an empty
// field means the default. X11's primary (`p`) and the cut buffers have no
// counterpart here, so they are ignored rather than aliased onto the clipboard
// — pasting something the user never copied is a surprise, not a feature.
const HONOURED = new Set(["", "c"]);

// decodeOsc52 turns an OSC 52 payload (everything after `52;`) into the text to
// copy, or null when there is nothing to do.
export function decodeOsc52(payload) {
  const raw = String(payload ?? "");
  const semi = raw.indexOf(";");
  if (semi < 0) return null;

  const selection = raw.slice(0, semi);
  if (!HONOURED.has(selection)) return null;

  const data = raw.slice(semi + 1).replace(/\s+/g, "");
  if (data === "" || data === "?") return null; // clear, or the read form

  if (data.length > MAX_OSC52_BASE64) return null;

  try {
    return utf8FromBase64(data);
  } catch {
    return null; // not valid base64 — drop it rather than paste garbage
  }
}

function utf8FromBase64(b64) {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return new TextDecoder("utf-8", { fatal: false }).decode(bytes);
}

// wireTermClipboard registers the handler on a terminal. `write` and `onError`
// are injectable so the behaviour can be tested without a browser clipboard.
// Returns xterm's disposable.
export function wireTermClipboard(term, opts = {}) {
  const write =
    opts.write ||
    ((text) =>
      typeof navigator !== "undefined" && navigator.clipboard
        ? navigator.clipboard.writeText(text)
        : Promise.reject(new Error("no clipboard API")));
  const onError = opts.onError || (() => {});

  return term.parser.registerOscHandler(52, (payload) => {
    const text = decodeOsc52(payload);
    // Always claim the sequence, even when refusing it. Returning false would
    // let it fall through as text, and for the read form silence *is* the
    // answer — no reply means nothing leaks.
    if (text === null) return true;
    try {
      const result = write(text);
      if (result && typeof result.catch === "function") result.catch(onError);
    } catch (err) {
      // A clipboard write can be refused outright (no user gesture, no
      // permission). Losing a copy is acceptable; throwing into the parser and
      // breaking the terminal is not.
      onError(err);
    }
    return true;
  });
}
