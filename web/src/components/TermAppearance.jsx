import { useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import ThemeCard from "./ThemeCard.jsx";
import TermPreview from "./TermPreview.jsx";
import { IconSun, IconTerminal } from "./Icons.jsx";
import {
  readTermPrefs, persistTermPrefs,
  TERM_FONTS, TERM_CURSORS, TERM_NEWLINES,
  TERM_SIZE_MIN, TERM_SIZE_MAX,
  TERM_LINE_MIN, TERM_LINE_MAX,
  TERM_TRACK_MIN, TERM_TRACK_MAX,
  TERM_PAD_MIN, TERM_PAD_MAX,
  TERM_SCROLL_MIN, TERM_SCROLL_MAX,
} from "../lib/termTheme.js";

// How terminals LOOK in this browser: font, colors, cursor, padding. Kept in
// localStorage on purpose — a laptop and a phone should not have to agree on
// a font size — while everything behavioural on the same page travels with
// the terminal (ADR-0024). `active` gates the live preview so a hidden page
// does not keep an xterm instance around.
export default function TermAppearance({ active }) {
  const [term, setTerm] = useState(readTermPrefs);

  useEffect(() => {
    function sync() { setTerm(readTermPrefs()); }
    window.addEventListener("picode-term-theme", sync);
    return () => window.removeEventListener("picode-term-theme", sync);
  }, []);

  function saveTerm(patch) {
    setTerm(persistTermPrefs(patch));
  }

  return (
    <div className="term-pref">
      <div className="term-pref-form">
        <h3 className="settings-h">Colors</h3>
        <div className="theme-cards two" role="radiogroup" aria-label="Terminal colors">
          <ThemeCard option="light" label="Light" desc="Bright glass" active={term.theme === "light"} onPick={(m) => saveTerm({ theme: m })} icon={<IconSun size={15} />} />
          <ThemeCard option="dark" label="Dark" desc="Classic terminal" active={term.theme === "dark"} onPick={(m) => saveTerm({ theme: m })} icon={<IconTerminal size={15} />} />
        </div>
        <div className="set-rows" style={{ marginTop: 14 }}>
          <div className="set-row">
            <label htmlFor="term-font">Font</label>
            <select id="term-font" className="set-wide" value={term.font} onChange={(e) => saveTerm({ font: e.target.value })}>
              {TERM_FONTS.map((f) => <option key={f.id} value={f.id}>{f.label}</option>)}
            </select>
          </div>
          <div className="set-row">
            <label htmlFor="term-size">Size</label>
            <input id="term-size" type="number" min={TERM_SIZE_MIN} max={TERM_SIZE_MAX} value={term.fontSize} onChange={(e) => saveTerm({ fontSize: e.target.value })} />
          </div>
          <div className="set-row">
            <label htmlFor="term-line">Line height</label>
            <input id="term-line" type="number" min={TERM_LINE_MIN} max={TERM_LINE_MAX} step={0.05} value={term.lineHeight} onChange={(e) => saveTerm({ lineHeight: e.target.value })} />
          </div>
          <div className="set-row">
            <label htmlFor="term-track">Letter spacing</label>
            <input id="term-track" type="number" min={TERM_TRACK_MIN} max={TERM_TRACK_MAX} step={0.5} value={term.letterSpacing} onChange={(e) => saveTerm({ letterSpacing: e.target.value })} />
          </div>
          <div className="set-row">
            <label htmlFor="term-cursor">Cursor</label>
            <select id="term-cursor" value={term.cursorStyle} onChange={(e) => saveTerm({ cursorStyle: e.target.value })}>
              {TERM_CURSORS.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
            </select>
          </div>
          <div className="set-row">
            <label htmlFor="term-blink">Blink</label>
            <Switch.Root id="term-blink" className="rx-switch" checked={term.cursorBlink} onCheckedChange={(v) => saveTerm({ cursorBlink: v })}>
              <Switch.Thumb className="rx-switch-thumb" />
            </Switch.Root>
          </div>
          <div className="set-row">
            <label htmlFor="term-scroll">Scrollback</label>
            <input id="term-scroll" type="number" min={TERM_SCROLL_MIN} max={TERM_SCROLL_MAX} step={1000} value={term.scrollback} onChange={(e) => saveTerm({ scrollback: e.target.value })} />
          </div>
          <div className="set-row">
            <label htmlFor="term-pad">Padding</label>
            <input id="term-pad" type="number" min={TERM_PAD_MIN} max={TERM_PAD_MAX} value={term.padding} onChange={(e) => saveTerm({ padding: e.target.value })} />
          </div>
        </div>
        <h3 className="settings-h">Keys</h3>
        <div className="set-rows">
          <div className="set-row">
            <label htmlFor="term-newline">New line</label>
            <select id="term-newline" className="set-wide" value={term.newlineKey} onChange={(e) => saveTerm({ newlineKey: e.target.value })}>
              {TERM_NEWLINES.map((k) => <option key={k.id} value={k.id}>{k.label}</option>)}
            </select>
          </div>
        </div>
        <ul className="hotkey-list term-key-facts">
          <li className="hotkey-row"><kbd>Shift+drag</kbd><span>Select</span></li>
          <li className="hotkey-row"><kbd>Ctrl+C</kbd><span>Copy if selected; else interrupt</span></li>
          <li className="hotkey-row"><kbd>Ctrl+V</kbd><span>Paste</span></li>
          <li className="hotkey-row"><kbd>Ctrl+`</kbd><span>New terminal</span></li>
          <li className="hotkey-row"><kbd>Ctrl+Shift+C</kbd><span>Copy</span></li>
          <li className="hotkey-row"><kbd>Ctrl+Shift+V</kbd><span>Paste</span></li>
          <li className="hotkey-row"><kbd>Ctrl++ / − / 0</kbd><span>Font size</span></li>
        </ul>
      </div>
      {active ? (
        <div className="term-pref-preview-slot">
          <TermPreview />
        </div>
      ) : null}
    </div>
  );
}
