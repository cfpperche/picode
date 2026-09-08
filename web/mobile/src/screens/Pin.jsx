import { useEffect, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import ScreenHeader from "../components/ScreenHeader.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { pinFileSrc } from "@picode/shared/domain/pinDraft.js";
import { reminderLine } from "@picode/shared/domain/pinReminder.js";
import { safeImgSrc } from "@picode/shared/domain/mdSafe.js";

// A pin on the phone, read-only (ADR-0100 slice 3; the editor stays on
// the desk): title, tags, the reminder line, the note and its pictures.
// It is where a reminder's "Open" lands, so it follows the feed for the
// pin's own updates.
export default function Pin({ pinId, onBack }) {
  const [pin, setPin] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let stop = false;
    function load() {
      api("/api/pins/" + encodeURIComponent(pinId)).then((p) => { if (!stop) { setPin(p); setError(""); } })
        .catch((e) => { if (!stop) setError(e && e.message ? e.message : "Could not open this pin."); });
    }
    load();
    const unsub = subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset") load();
      else if (ev.type === "pin.updated" && ev.data && ev.data.id === pinId) load();
      else if (ev.type === "pin.deleted" && ev.data && ev.data.id === pinId) setError("This pin was deleted.");
    });
    return () => { stop = true; unsub(); };
  }, [pinId]);

  const images = pin ? (pin.files || []).filter((f) => f.kind === "image" || f.kind === "sketch") : [];
  const others = pin ? (pin.files || []).filter((f) => f.kind !== "image" && f.kind !== "sketch") : [];
  const line = pin ? reminderLine(pin.reminder) : "";

  return (
    <div className="m-screen m-pin">
      <ScreenHeader title={pin ? pin.title : "Pin"} onBack={onBack} />
      {error ? <p className="m-pin-msg">{error}</p> : null}
      {!pin && !error ? <p className="m-pin-msg">Loading…</p> : null}
      {pin ? (
        <div className="m-pin-body">
          {pin.tags && pin.tags.length || pin.starred || pin.archivedAt ? (
            <div className="m-pin-tags">
              {pin.starred ? <span className="m-pin-tag m-pin-flag">on top</span> : null}
              {pin.archivedAt ? <span className="m-pin-tag m-pin-flag">archived</span> : null}
              {(pin.tags || []).map((t) => <span key={t} className="m-pin-tag">#{t}</span>)}
            </div>
          ) : null}
          {line && !pin.archivedAt ? <div className="m-pin-remind">{line}</div> : null}
          {images.length ? (
            <div className="m-pin-gallery">
              {images.map((f) => <img key={f.id} src={pinFileSrc(pin.id, f)} alt={f.name} loading="lazy" />)}
            </div>
          ) : null}
          {pin.body ? (
            <div className="m-pin-md">
              <Markdown remarkPlugins={[remarkGfm]} urlTransform={(url) => safeImgSrc(url) || (url.startsWith("http") ? url : "")}>{pin.body}</Markdown>
            </div>
          ) : null}
          {others.length ? (
            <ul className="m-pin-files">
              {others.map((f) => <li key={f.id}><a href={pinFileSrc(pin.id, f)} download={f.name}>{f.name}</a></li>)}
            </ul>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
