import { useCallback, useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import { api } from "@picode/shared/client/api.js";
import { toast, toastError } from "../lib/toast.js";

// Surface wrapper policies (ADR-0138 tmux guard, ADR-0180 browser
// hand-off): PATH wrappers for every terminal PiCode opens. They are not
// tmux options and not settings of the CLI selected below, so they do not
// live on Terminal defaults — that page is font, color and options a
// terminal inherits. Both default on; the switches are the opt-out.
const ROWS = [
  {
    id: "tmux-guard",
    label: "tmux guard",
    on: "On — refuses kill-server, pattern kills and other terminals’ sessions inside every PiCode terminal. Your own sessions stay killable by exact name.",
    off: "Off — every tmux command reaches the server, including kill-server.",
    toastOn: "Terminal guard is on for terminals opened from now on.",
    toastOff: "Terminal guard is off for terminals opened from now on.",
    docs: "https://cfpperche.github.io/picode/guide/agent-clis#tmux-guard",
  },
  {
    id: "open-url",
    label: "browser hand-off",
    on: "On — a CLI’s “open in browser” (logins) opens in the app’s browser tab or your default browser instead of a browser inside WSL.",
    off: "Off — CLI logins open a browser inside WSL again.",
    toastOn: "Browser hand-off is on for terminals opened from now on.",
    toastOff: "Browser hand-off is off for terminals opened from now on.",
    docs: "https://cfpperche.github.io/picode/guide/agent-clis#browser-hand-off",
  },
];

export default function SurfaceWrappers({ hidden }) {
  const [rows, setRows] = useState({});
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState("");

  const load = useCallback(async () => {
    try {
      const d = await api("/api/terminals/wiring");
      const next = {};
      for (const spec of ROWS) {
        const row = (d.clis || []).find((r) => r.id === spec.id);
        if (row) next[spec.id] = row;
      }
      setRows(next);
      setErr("");
    } catch (e) {
      setErr(e?.message || "Could not read the wrapper states.");
    }
  }, []);

  useEffect(() => {
    if (hidden) return;
    load();
  }, [hidden, load]);

  async function setOn(spec, on) {
    const row = rows[spec.id];
    if (!row || busy) return;
    setRows({ ...rows, [spec.id]: { ...row, wired: on } });
    setBusy(spec.id);
    try {
      const d = await api(`/api/terminals/wiring/${spec.id}/${on ? "enable" : "disable"}`, { method: "POST" });
      const fresh = (d.clis || []).find((r) => r.id === spec.id);
      if (fresh) setRows((cur) => ({ ...cur, [spec.id]: fresh }));
      toast.ok(on ? spec.toastOn : spec.toastOff);
    } catch (e) {
      setRows((cur) => ({ ...cur, [spec.id]: row }));
      toastError(e);
    } finally {
      setBusy("");
    }
  }

  return (
    <>
      {ROWS.map((spec) => {
        const row = rows[spec.id];
        if (!row && !err) return null; // the list loaded but lacks the row: hidden, not disabled
        return (
          <section key={spec.id} className="cli-guard" aria-label={spec.label}>
            <div>
              <strong>{spec.label}</strong>
              <p>
                {row?.wired ? spec.on : row ? spec.off : "Checking whether it is on."}
              </p>
              <p>
                Every PiCode terminal, not the CLI selected below. Applies to terminals opened from now on.{" "}
                <a href={spec.docs} target="_blank" rel="noreferrer">How it works ↗</a>
              </p>
              {err ? (
                <div className="cli-notice is-error" role="alert">
                  <span>{row ? "Couldn’t refresh the state — this switch may be out of date." : err}</span>
                  <button type="button" className="btn btn-ghost btn-sm" onClick={load}>Try again</button>
                </div>
              ) : null}
            </div>
            <Switch.Root
              className="rx-switch"
              checked={!!row?.wired}
              disabled={!row || busy === spec.id}
              aria-label={spec.label}
              onCheckedChange={(on) => setOn(spec, on)}
            >
              <Switch.Thumb className="rx-switch-thumb" />
            </Switch.Root>
          </section>
        );
      })}
    </>
  );
}
