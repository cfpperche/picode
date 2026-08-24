import { useEffect, useState } from "react";

const TTL = { err: 7000, ok: 3500, info: 4000 };

export default function Toasts() {
  const [items, setItems] = useState([]);

  useEffect(() => {
    const timers = new Map();
    function dismiss(id) {
      setItems((cur) => cur.filter((t) => t.id !== id));
      const t = timers.get(id);
      if (t) { clearTimeout(t); timers.delete(id); }
    }
    function onToast(e) {
      const t = e.detail;
      if (!t || !t.message) return;
      setItems((cur) => [...cur.slice(-4), t]);
      const ms = TTL[t.kind] || TTL.info;
      timers.set(t.id, setTimeout(() => dismiss(t.id), ms));
    }
    window.addEventListener("picode-toast", onToast);
    return () => {
      window.removeEventListener("picode-toast", onToast);
      timers.forEach((id) => clearTimeout(id));
    };
  }, []);

  if (!items.length) return null;
  return (
    <div className="toasts" aria-live="polite">
      {items.map((t) => (
        <div key={t.id} className={"toast toast-" + t.kind} role={t.kind === "err" ? "alert" : "status"}>
          <span className="toast-msg">{t.message}</span>
          <button
            type="button"
            className="toast-x"
            aria-label="Dismiss"
            onClick={() => setItems((cur) => cur.filter((x) => x.id !== t.id))}
          >×</button>
        </div>
      ))}
    </div>
  );
}
