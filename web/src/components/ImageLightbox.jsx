import { useEffect } from "react";
import { IconX } from "./Icons.jsx";

export default function ImageLightbox({ src, onClose }) {
  useEffect(() => {
    if (!src) return;
    function onKey(e) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [src, onClose]);
  if (!src) return null;
  return (
    <div className="img-lite" role="dialog" aria-modal="true" aria-label="Image preview" onClick={onClose}>
      <div className="img-lite-card" onClick={(e) => e.stopPropagation()}>
        <button type="button" className="img-lite-x" onClick={onClose} aria-label="Close">
          <IconX size={16} />
        </button>
        <img src={src} alt="" />
      </div>
    </div>
  );
}
