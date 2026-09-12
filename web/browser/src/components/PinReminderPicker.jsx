import { useEffect, useState } from "react";
import * as Popover from "@radix-ui/react-popover";
import { IconClock, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { browserZone, buildReminder, formFromReminder, reminderLine } from "@picode/shared/domain/pinReminder.js";
import { toast, toastError } from "../lib/toast.js";

// The "Remind me" chip on a pin (ADR-0100): one small form, nothing
// preset. Once — a date and a time. Repeat — every N hours, or every N
// days at a time of day, optionally counted from when the card is closed.
// One Set at the bottom; a reminder is not part of the draft, it is a
// promise the server keeps from the moment it is made, so Set writes at
// once and the chip reads the rule back in words with its next fire.
export default function PinReminderPicker({ pinId, reminder, onChange }) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState(() => formFromReminder(reminder));
  const [error, setError] = useState("");

  // Opening starts from the rule that is set, not from blanks.
  useEffect(() => {
    if (open) { setForm(formFromReminder(reminder)); setError(""); }
  }, [open, reminder]);

  const patch = (p) => { setForm((f) => ({ ...f, ...p })); setError(""); };

  async function set() {
    const built = buildReminder({ ...form, tz: browserZone() });
    if (built.error) { setError(built.error); return; }
    setBusy(true);
    try {
      const r = await api("/api/pins/" + encodeURIComponent(pinId) + "/reminder", {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(built.body),
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
  const days = form.unit === "days";

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button type="button" className={"btn btn-ghost btn-sm pin-remind-chip" + (reminder ? " has-reminder" : "")} disabled={busy} title="Reminder">
          <IconClock /> <span className="pin-remind-label">{label}</span>
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content className="pin-remind-pop" side="bottom" align="end" sideOffset={6} collisionPadding={8}>
          <div className="pin-remind-modes" role="radiogroup" aria-label="Reminder">
            <button type="button" role="radio" aria-checked={form.mode === "once"} className={"pin-remind-mode" + (form.mode === "once" ? " on" : "")} onClick={() => patch({ mode: "once" })}>Once</button>
            <button type="button" role="radio" aria-checked={form.mode === "repeat"} className={"pin-remind-mode" + (form.mode === "repeat" ? " on" : "")} onClick={() => patch({ mode: "repeat" })}>Repeat</button>
          </div>

          {form.mode === "once" ? (
            <label className="pin-remind-field">
              <span>Date and time</span>
              <input type="datetime-local" className="pin-remind-when" value={form.at} onChange={(e) => patch({ at: e.target.value })} />
            </label>
          ) : (
            <>
              <div className="pin-remind-field pin-remind-every">
                <span>Every</span>
                <input type="number" min="1" max="8784" className="pin-remind-num" value={form.every} aria-label="Every how many" onChange={(e) => patch({ every: e.target.value })} />
                <select className="pin-remind-unit" value={form.unit} aria-label="Hours or days" onChange={(e) => patch({ unit: e.target.value })}>
                  <option value="hours">hours</option>
                  <option value="days">days</option>
                </select>
                {days ? (
                  <>
                    <span>at</span>
                    <input type="time" className="pin-remind-time" value={form.time} aria-label="Time of day" onChange={(e) => patch({ time: e.target.value })} />
                  </>
                ) : null}
              </div>
              <label className="pin-remind-check">
                <input type="checkbox" checked={!!form.fromClose} onChange={(e) => patch({ fromClose: e.target.checked })} /> count from when I close it
              </label>
              {days && Number(form.every) > 1 ? <p className="pin-remind-note">Counted as a duration: across a daylight-saving change the hour shifts by one.</p> : null}
            </>
          )}

          {error ? <p className="pin-remind-error" role="alert">{error}</p> : null}

          <div className="pin-remind-foot">
            <span className={"pin-remind-current" + (reminder ? "" : " none")}>{reminder ? reminderLine(reminder) : "No reminder"}</span>
            <span className="pin-remind-btns">
              {reminder ? <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={remove}><IconX size={12} /> Remove</button> : null}
              <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={set}>Set</button>
            </span>
          </div>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
