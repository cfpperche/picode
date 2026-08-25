import { useEffect, useState } from "react";
import FolderPicker from "./FolderPicker.jsx";
import { IconFolder } from "./Icons.jsx";

export default function FolderField({ name, placeholder, resetKey }) {
  const [val, setVal] = useState("");
  const [open, setOpen] = useState(false);

  useEffect(() => { setVal(""); }, [resetKey]);

  return (
    <div className="folder-field">
      <input name={name} type="text" value={val} onChange={(e) => setVal(e.target.value)} placeholder={placeholder} autoComplete="off" />
      <button type="button" className="btn btn-ghost btn-sm" title="Browse" onClick={() => setOpen(true)}>
        <IconFolder size={14} /> Browse
      </button>
      <FolderPicker
        open={open}
        start={val}
        onClose={() => setOpen(false)}
        onPick={(p) => { setVal(p); setOpen(false); }}
      />
    </div>
  );
}
