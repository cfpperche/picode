// watchReminders keeps the sticky reminder cards true to the Inbox
// (ADR-0100) for one shell. Both apps call it with their own routes and
// toast door; the rules live in @picode/shared/domain/notice.js.
//
//   - on start and on feed.open / feed.reset: list the open reminder
//     items and raise a card each (or one collapsed card above the cap)
//   - pin.reminded: re-list (the payload names the item; the list is the
//     truth for snooze and collapse)
//   - inbox.updated for a reminder item: re-list, so a close or a snooze
//     on another device withdraws the card here
//   - X on a card → the item goes done; Snooze → snoozed for the
//     viewer's snooze minutes; both through the Inbox state route
import { api } from "./api.js";
import { subscribeFeed } from "./feed.js";
import { reminderPlan, reminderNotice, remindersCollapsedNotice } from "../domain/notice.js";
import { readReminderPrefs, snoozeUntil } from "../domain/pinReminder.js";

export function watchReminders({ notify, dismiss, pinHash, inboxHash, prefsStore } = {}) {
  let shown = new Set();
  let stopped = false;
  let inflight = null;

  async function setState(id, body) {
    await api("/api/inbox/" + encodeURIComponent(id) + "/state", {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body),
    });
  }

  function cardFor(fire) {
    return reminderNotice(fire, {
      pinHash: pinHash ? pinHash(fire.pinId) : "",
      snooze: async () => {
        const mins = readReminderPrefs(prefsStore).snoozeMin;
        try { await setState(fire.inboxId, { snoozedUntil: snoozeUntil(mins) }); } catch { /* the card stays; the row did not move */ }
      },
      close: async () => {
        try { await setState(fire.inboxId, { state: "done" }); } catch { /* same */ }
      },
    });
  }

  async function reconcile() {
    if (stopped) return;
    if (inflight) return inflight;
    inflight = (async () => {
      let items = [];
      try {
        const d = await api("/api/inbox?kind=reminder");
        items = (d && d.items) || [];
      } catch { return; } finally { inflight = null; }
      if (stopped) return;
      const plan = reminderPlan(items, shown, {
        notice: cardFor,
        collapsed: (n) => remindersCollapsedNotice(n, inboxHash || ""),
      });
      for (const key of plan.hide) dismiss(key);
      for (const n of plan.show) notify(n);
      shown = plan.keys;
    })();
    return inflight;
  }

  const unsub = subscribeFeed((ev) => {
    if (!ev || !ev.type) return;
    if (ev.type === "feed.open" || ev.type === "feed.reset" || ev.type === "pin.reminded") { reconcile(); return; }
    if (ev.type === "inbox.updated" || ev.type === "inbox.created" || ev.type === "inbox.deleted") {
      const it = ev.data || {};
      if (it.kind === "reminder" || !it.kind) reconcile();
    }
  });
  reconcile();

  return () => {
    stopped = true;
    unsub();
    for (const key of shown) dismiss(key);
    shown = new Set();
  };
}
