import { useCallback, useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import { api } from "@picode/shared/client/api.js";
import { toast, toastError } from "../lib/toast.js";

// The tmux guard (ADR-0138) is a PATH wrapper for every terminal PiCode
// opens. It is not a tmux option and not a setting of the CLI selected
// below, so it does not live on Terminal defaults — that page is font,
// color and options a terminal inherits. The accident it stops is an
// agent typing kill-server, which is why the switch sits on Agent CLIs.
export default function TmuxGuardRow({ hidden }) {
  const [guard, setGuard] = useState(null);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const d = await api("/api/terminals/wiring");
      const row = (d.clis || []).find((r) => r.id === "tmux-guard") || null;
      setGuard(row);
      setErr(row ? "" : "The guard row is missing from the wiring list.");
    } catch (e) {
      setErr(e?.message || "Could not read the guard state.");
    }
  }, []);

  useEffect(() => {
    if (hidden) return;
    load();
  }, [hidden, load]);

  async function setOn(on) {
    if (!guard || busy) return;
    const prev = guard;
    setGuard({ ...guard, wired: on });
    setBusy(true);
    try {
      const d = await api(`/api/terminals/wiring/tmux-guard/${on ? "enable" : "disable"}`, { method: "POST" });
      const fresh = (d.clis || []).find((r) => r.id === "tmux-guard");
      if (fresh) setGuard(fresh);
      toast.ok(on ? "Terminal guard is on for terminals opened from now on." : "Terminal guard is off for terminals opened from now on.");
    } catch (e) {
      setGuard(prev);
      toastError(e);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="cli-guard" aria-label="tmux guard">
      <div>
        <strong>tmux guard</strong>
        <p>
          {guard?.wired
            ? "On — refuses kill-server, pattern kills and other terminals’ sessions inside every PiCode terminal. Your own sessions stay killable by exact name."
            : guard
              ? "Off — every tmux command reaches the server, including kill-server."
              : "Checking whether the guard is on."}
        </p>
        <p>
          Every PiCode terminal, not the CLI selected below. Applies to terminals opened from now on.{" "}
          <a href="https://cfpperche.github.io/picode/guide/agent-clis#tmux-guard" target="_blank" rel="noreferrer">How it works ↗</a>
        </p>
        {err ? (
          <div className="cli-notice is-error" role="alert">
            <span>{guard ? "Couldn’t refresh the guard state — this switch may be out of date." : err}</span>
            <button type="button" className="btn btn-ghost btn-sm" onClick={load}>Try again</button>
          </div>
        ) : null}
      </div>
      <Switch.Root
        className="rx-switch"
        checked={!!guard?.wired}
        disabled={!guard || busy}
        aria-label="tmux guard"
        onCheckedChange={setOn}
      >
        <Switch.Thumb className="rx-switch-thumb" />
      </Switch.Root>
    </section>
  );
}
