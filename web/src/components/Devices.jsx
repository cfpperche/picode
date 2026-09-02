import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { feedConnected, subscribeFeed } from "../lib/feed.js";
import { touches } from "../lib/feedReducers.js";
import PageFrame from "./PageFrame.jsx";

export default function Devices({ hidden }) {
  const [list, setList] = useState([]);

  useEffect(() => {
    if (hidden) return;
    let on = true;
    const load = () => api("/api/devices").then((d) => { if (on) setList(d || []); }).catch(() => {});
    load();
    // The server announces device.online / device.offline (its expiry
    // ticker turns silence into an event); the 4 s poll is the fallback
    // for when the feed is down.
    const t = setInterval(() => { if (!feedConnected()) load(); }, 4000);
    const unsub = subscribeFeed((ev) => { if (touches(ev, ["device"]) || ev.type === "feed.open") load(); });
    return () => { on = false; clearInterval(t); unsub(); };
  }, [hidden]);

  const remote = list.filter((d) => !d.host);
  const host = list.filter((d) => d.host);

  return (
    <PageFrame id="devices-view" title="Devices" hidden={hidden}>
      <section className="settings-section">
        <h3>This machine</h3>
        <DeviceList items={host} empty="This browser is not pinging yet." />
      </section>
      <section className="settings-section">
        <h3>Other devices</h3>
        <DeviceList items={remote} empty="No phones or other machines connected." />
      </section>
    </PageFrame>
  );
}

function DeviceList({ items, empty }) {
  if (!items.length) return <p className="settings-desc">{empty}</p>;
  return (
    <ul className="dev-list">
      {items.map((d) => (
        <li key={d.id} className={"dev-row" + (d.online ? "" : " off")}>
          <span className={"share-dot" + (d.online ? "" : " off")} />
          <span className="dev-name">{d.name}</span>
          <span className="dev-ip">{d.ip}</span>
          <span className="dev-seen">{d.online ? "online" : "last seen " + d.lastSeen.slice(11, 19) + "Z"}</span>
        </li>
      ))}
    </ul>
  );
}
