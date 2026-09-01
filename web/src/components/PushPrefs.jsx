import { useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import { pushBlockedReason, currentSubscription, subscribePush, unsubscribePush, setPushPrefs, sendTestPush } from "../lib/push.js";
import { toast, toastError } from "../lib/toast.js";

// Push notifications for THIS device (ADR-0047). Two switches — needs me,
// finished — an Enable/Disable, a Send test. The state is the browser's
// own subscription, read on mount; the server only holds a copy.
export default function PushPrefs() {
  const [reason, setReason] = useState(() => pushBlockedReason());
  const [sub, setSub] = useState(undefined); // undefined = loading, null = none
  const [prefs, setPrefs] = useState({ actions: true, finished: true });
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    setReason(pushBlockedReason());
    currentSubscription().then((s) => setSub(s || null)).catch(() => setSub(null));
  }, []);

  async function enable() {
    setBusy(true);
    try {
      const rec = await subscribePush(prefs);
      if (rec && rec.prefs) setPrefs(rec.prefs);
      setSub(await currentSubscription());
      toast.ok("Push enabled on this device.");
    } catch (e) {
      toastError(e);
      setReason(pushBlockedReason());
    } finally { setBusy(false); }
  }
  async function disable() {
    setBusy(true);
    try { await unsubscribePush(); setSub(null); toast.info("Push disabled on this device."); } catch (e) { toastError(e); } finally { setBusy(false); }
  }
  async function toggle(key, value) {
    const next = { ...prefs, [key]: value };
    setPrefs(next);
    if (!sub) return;
    try { await setPushPrefs(next); } catch (e) { toastError(e); }
  }
  async function test() {
    setBusy(true);
    try { await sendTestPush(); toast.ok("Sent — it arrives in a moment."); } catch (e) { toastError(e); } finally { setBusy(false); }
  }

  return (
    <div className="push-prefs">
      <p className="push-prefs-lead">Wake this device when nobody is at the computer: PiCode stays quiet while a browser on the host machine is open.</p>
      {reason ? (
        <p className="push-prefs-reason">{reason}</p>
      ) : sub === undefined ? null : sub ? (
        <>
          <div className="set-rows">
            <div className="set-row">
              <label htmlFor="push-actions">When an agent needs me</label>
              <Switch.Root id="push-actions" className="rx-switch" checked={prefs.actions} onCheckedChange={(v) => toggle("actions", v)}>
                <Switch.Thumb className="rx-switch-thumb" />
              </Switch.Root>
            </div>
            <div className="set-row">
              <label htmlFor="push-finished">When a run finishes</label>
              <Switch.Root id="push-finished" className="rx-switch" checked={prefs.finished} onCheckedChange={(v) => toggle("finished", v)}>
                <Switch.Thumb className="rx-switch-thumb" />
              </Switch.Root>
            </div>
          </div>
          <div className="push-prefs-actions">
            <button type="button" className="btn btn-sm" disabled={busy} onClick={test}>Send test</button>
            <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={disable}>Disable on this device</button>
          </div>
        </>
      ) : (
        <div className="push-prefs-actions">
          <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={enable}>Enable push on this device</button>
        </div>
      )}
    </div>
  );
}
