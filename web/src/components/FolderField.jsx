import { useEffect, useState } from "react";
import FolderPicker from "./FolderPicker.jsx";
import { IconFolder } from "./Icons.jsx";

export default function FolderField({ name, placeholder, resetKey, value, onChange }) {
  const [inner, setInner] = useState("");
  const [open, setOpen] = useState(false);
  const controlled = value !== undefined;
  const val = controlled ? value : inner;

  useEffect(() => { if (!controlled) setInner(""); }, [resetKey, controlled]);

  function set(next) {
    if (controlled) onChange && onChange(next);
    else setInner(next);
  }

  return (
    <div className="folder-field">
      <input name={name} type="text" value={val} onChange={(e) => set(e.target.value)} placeholder={placeholder} autoComplete="off" />
      <button type="button" className="btn btn-ghost btn-sm" title="Browse" onClick={() => setOpen(true)}>
        <IconFolder size={14} /> Browse
      </button>
      <FolderPicker
        open={open}
        start={val}
        onClose={() => setOpen(false)}
        onPick={(p) => { set(p); setOpen(false); }}
      />
    </div>
  );
}
