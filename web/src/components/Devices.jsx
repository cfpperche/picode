import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { openPairDrawer } from "./ShareDrawer.jsx";
import { feedConnected, subscribeFeed } from "../lib/feed.js";
import { touches } from "../lib/feedReducers.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { relTime, absTime } from "../lib/relTime.js";
import PageFrame from "./PageFrame.jsx";

const EXT_GUIDE = "https://cfpperche.github.io/picode/guide/browser-extension";

// Devices (ADR-0043 presence + ADR-0049 identity, one surface): who may
// enter, and whether they are here right now. Identity is the paired
// session; liveness is the presence ping that session sends. Access
// rules and the install token live in Preferences → Server.
export default function Devices({ hidden }) {
  const [rows, setRows] = useState(null);
  const [live, setLive] = useState([]);

  async function load() {
    try {
      const [s, d] = await Promise.all([api("/api/auth/sessions"), api("/api/devices").catch(() => [])]);
      setRows(s.items || []);
      setLive(Array.isArray(d) ? d : []);
    } catch (e) { toastError(e); }
  }
  useEffect(() => {
    if (hidden) return;
    load();
    const t = setInterval(() => { if (!feedConnected()) load(); }, 4000);
    const unsub = subscribeFeed((ev) => { if (ev.type === "feed.open" || touches(ev, ["device", "session", "pairing"])) load(); });
    return () => { clearInterval(t); unsub(); };
  }, [hidden]);
  // The QR lives in one place: the pairing drawer (phone icon), which
  // picks the address the phone can reach. A computer with no camera
  // gets a link on the clipboard instead.
  function pair() { openPairDrawer(); }
  async function copyLink() {
    try {
      const p = await api("/api/auth/pairings", { method: "POST" });
      await navigator.clipboard.writeText(p.url);
      toast.ok("Pairing link copied — it works once, for ten minutes.");
    } catch (e) { toastError(e); }
  }
  async function forget(row) {
    const ok = await askConfirm({
      title: "Forget " + (row.label || row.id) + "?",
      message: row.current ? "This is the device you are using — you will need to pair it again." : "That device will need a new pairing link to get back in.",
      confirmLabel: "Forget", danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/auth/sessions/" + encodeURIComponent(row.id), { method: "DELETE" });
      if (row.current) location.reload(); else load();
    } catch (e) { toastError(e); }
  }

  // One click for the pile of stale rows (offline, not this one): QA
  // browsers and old phones that no longer connect. Nothing online,
  // nothing current and no token session is ever touched (the install
  // token keeps its own explicit Forget), and the confirm names them first.
  const offline = (rows || []).filter((r) => !r.online && !r.current && r.kind !== "token");
  async function forgetOffline() {
    if (!offline.length) return;
    const names = offline.map((r) => r.label || r.id);
    const listed = names.slice(0, 6).join(", ") + (names.length > 6 ? " + " + (names.length - 6) + " more" : "");
    const ok = await askConfirm({
      title: "Forget " + names.length + " offline devices?",
      message: listed + ". Each will need a new pairing link to get back in.",
      confirmLabel: "Forget all", danger: true,
    });
    if (!ok) return;
    const results = await Promise.allSettled(
      offline.map((r) => api("/api/auth/sessions/" + encodeURIComponent(r.id), { method: "DELETE" })),
    );
    const failed = results.filter((r) => r.status === "rejected").length;
    if (failed) toastError(new Error(failed + " of " + names.length + " could not be forgotten"));
    else toast.ok(names.length + (names.length === 1 ? " device forgotten." : " devices forgotten."));
    load();
  }

  const unpaired = live.filter((d) => !d.session && d.online);
  const extOnline = live.some((d) => d.kind === "extension" && d.online);

  return (
    <PageFrame id="devices-view" title="Devices" hidden={hidden}>
      {rows === null ? (
        <div className="mcp-skel" aria-hidden="true"><span className="skel-line w-70" /><span className="skel-line w-40" /></div>
      ) : rows.length === 0 ? (
        <div className="mcp-empty">
          <p>No paired devices yet.</p>
          <button type="button" className="btn btn-primary" onClick={pair}>Pair a device</button>
        </div>
      ) : (
        <>
          <ul className="dev-list">
            {rows.map((r) => (
              <li key={r.id} className={"dev-row" + (r.online ? "" : " off")} data-align-row>
                <span className="dev-dot-cell"><span className={"share-dot" + (r.online ? "" : " off")} /></span>
                <span className="dev-name">
                  {r.label || r.id}
                  {r.current ? <span className="devs-tag">this device</span> : null}
                  {r.kind === "token" ? <span className="devs-tag">token</span> : null}
                  {r.pingKind === "extension" ? <span className="devs-tag">extension</span> : null}
                </span>
                <span className="dev-ip">{r.ip}</span>
                <span className="dev-seen" title={absTime(r.pingSeen || r.lastSeenAt)}>{r.online ? "online" : "seen " + relTime(r.pingSeen || r.lastSeenAt)}</span>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => forget(r)}>Forget</button>
              </li>
            ))}
          </ul>
          <div className="devs-foot">
            <div className="devs-actions" data-align-row>
              <button type="button" className="btn btn-primary" onClick={pair}>Pair a device</button>
              <button type="button" className="btn btn-ghost" onClick={copyLink}>Copy a pairing link</button>
              {offline.length ? (
                <button type="button" className="btn btn-ghost" onClick={forgetOffline}>Forget offline ({offline.length})</button>
              ) : null}
            </div>
            {extOnline ? null : (
              <p className="settings-desc dev-ext-note">Chrome extension: not connected. <a href={EXT_GUIDE} target="_blank" rel="noreferrer">Open guide</a></p>
            )}
          </div>
        </>
      )}
      {unpaired.length ? (
        <section className="settings-section">
          <h3>Online without pairing</h3>
          <p className="settings-desc">Only possible while pairing is off. Turn it on in Preferences → Server.</p>
          <ul className="dev-list">
            {unpaired.map((d) => (
              <li key={d.id} className="dev-row">
                <span className="share-dot" />
                <span className="dev-name">{d.name}</span>
                <span className="dev-ip">{d.ip}</span>
                <span className="dev-seen">online</span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </PageFrame>
  );
}
