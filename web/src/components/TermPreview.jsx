import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { applyXtermOptions, readTermPrefs, xtermOptions, xtermTheme } from "../lib/termTheme.js";
import "@xterm/xterm/css/xterm.css";

const SAMPLE = [
  "\x1b[32mgoat@picode\x1b[0m:\x1b[34m~\x1b[0m$ echo hello",
  "hello",
  "\x1b[32mgoat@picode\x1b[0m:\x1b[34m~\x1b[0m$ ls",
  "src  docs  Makefile",
  "\x1b[32mgoat@picode\x1b[0m:\x1b[34m~\x1b[0m$ ",
].join("\r\n");

function paint(term) {
  term.reset();
  term.write(SAMPLE);
}

function skin(el) {
  if (!el) return;
  const p = readTermPrefs();
  el.style.padding = p.padding + "px";
  el.style.background = xtermTheme(p.theme).background;
}

export default function TermPreview() {
  const hostRef = useRef(null);

  useEffect(() => {
    const el = hostRef.current;
    if (!el) return undefined;
    const term = new Terminal({ ...xtermOptions(), disableStdin: true, scrollback: 80 });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(el);
    function apply() {
      applyXtermOptions(term);
      term.options.scrollback = 80;
      skin(el);
      paint(term);
      requestAnimationFrame(() => fit.fit());
    }
    apply();
    const ro = new ResizeObserver(() => { if (el.isConnected) fit.fit(); });
    ro.observe(el);
    window.addEventListener("picode-term-theme", apply);
    return () => {
      window.removeEventListener("picode-term-theme", apply);
      ro.disconnect();
      term.dispose();
    };
  }, []);

  return (
    <div
      className="term-preview"
      ref={hostRef}
      role="img"
      aria-label="Terminal preview"
    />
  );
}
