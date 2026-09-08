import { useState } from "react";
import * as Popover from "@radix-ui/react-popover";
import { IconClock, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { REMINDER_PRESETS, browserZone, buildReminder, readReminderPrefs, reminderLine, toLocalInput } from "@picode/shared/domain/pinReminder.js";
import { toast, toastError } from "../lib/toast.js";

// The "Remind me" chip on a pin (ADR-0100): presets first, custom last,
// one reminder per pin. Picking writes PUT /api/pins/{id}/reminder at
// once — a reminder is not part of the draft, it is a promise the server
// keeps from the moment it is made — and the chip reads the rule back in
// words with its next fire.
export default function PinReminderPicker({ pinId, reminder, onChange }) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [hours, setHours] = useState(24);
  const [fromClose, setFromClose] = useState(false);
  const [at, setAt] = useState(() => toLocalInput(new Date(Date.now() + 3600_000)));
  const [cron, setCron] = useState("0 9 * * *");
  const prefs = readReminderPrefs();

  async function set(presetId, extra) {
    const body = buildReminder(presetId, { now: new Date(), tz: browserZone(), morning: prefs.morning, hours, fromClose, at, cron, ...extra });
    if (!body) { toast("Pick a valid date and time.", "info"); return; }
    setBusy(true);
    try {
      const r = await api("/api/pins/" + encodeURIComponent(pinId) + "/reminder", {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body),
      });
      onChange(r);
      setOpen(false);
      toast.ok("Reminder set: " + r.label + ".");
    } catch (e) { toastError(e); }
    finally { setBusy(false); }
  }

  async function remove() {
    setBusy(true);
    try {
      await api("/api/pins/" + encodeURIComponent(pinId) + "/reminder", { method: "DELETE" });
      onChange(null);
      setOpen(false);
      toast.info("Reminder removed.");
    } catch (e) { toastError(e); }
    finally { setBusy(false); }
  }

  const label = reminder ? reminderLine(reminder) : "Remind me";
  const groups = ["once", "repeat", "custom"];
  const titles = { once: "Once", repeat: "Repeat", custom: "Custom" };

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button type="button" className={"btn btn-ghost btn-sm pin-remind-chip" + (reminder ? " has-reminder" : "")} disabled={busy} title="Reminder">
          <IconClock /> <span className="pin-remind-label">{label}</span>
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content className="pin-remind-pop" side="bottom" align="start" sideOffset={6} collisionPadding={8}>
          {groups.map((g) => (
            <div key={g} className="pin-remind-group">
              <div className="pin-remind-group-title">{titles[g]}</div>
              {REMINDER_PRESETS.filter((p) => p.group === g).map((p) => {
                if (p.id === "everyNh") {
                  return (
                    <div key={p.id} className="pin-remind-row">
                      <span>Every</span>
                      <input type="number" min="1" max="8784" className="pin-remind-num" value={hours} aria-label="Hours" onChange={(e) => setHours(e.target.value)} />
                      <span>hours</span>
                      <button type="button" className="btn btn-sm" disabled={busy} onClick={() => set("everyNh")}>Set</button>
                      <label className="pin-remind-check">
                        <input type="checkbox" checked={fromClose} onChange={(e) => setFromClose(e.target.checked)} /> count from when I close it
                      </label>
                    </div>
                  );
                }
                if (p.id === "pick") {
                  return (
                    <div key={p.id} className="pin-remind-row">
                      <input type="datetime-local" className="pin-remind-when" value={at} aria-label="Date and time" onChange={(e) => setAt(e.target.value)} />
                      <button type="button" className="btn btn-sm" disabled={busy} onClick={() => set("pick")}>Set</button>
                    </div>
                  );
                }
                if (p.id === "cron") {
                  return (
                    <div key={p.id} className="pin-remind-row">
                      <input type="text" className="pin-remind-cron" value={cron} aria-label="Cron expression" spellCheck={false} onChange={(e) => setCron(e.target.value)} placeholder="minute hour day month weekday" />
                      <button type="button" className="btn btn-sm" disabled={busy} onClick={() => set("cron")}>Set</button>
                    </div>
                  );
                }
                return (
                  <button key={p.id} type="button" className="pin-remind-preset" disabled={busy} onClick={() => set(p.id)}>
                    {p.label}{(p.id === "tomorrow" || p.id === "nextMonday" || p.id === "daily" || p.id === "weekdays") ? <span className="pin-remind-hint">{prefs.morning}</span> : null}
                  </button>
                );
              })}
            </div>
          ))}
          {reminder ? (
            <div className="pin-remind-foot">
              <span className="pin-remind-current">{reminderLine(reminder)}</span>
              <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={remove}><IconX size={12} /> Remove</button>
            </div>
          ) : null}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
