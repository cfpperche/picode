import { useEffect } from "react";
import { createPortal } from "react-dom";
import { IconX } from "./Icons.jsx";

export default function ImageLightbox({ src, onClose }) {
  useEffect(() => {
    if (!src) return;
    function onKey(e) {
      if (e.key !== "Escape") return;
      e.preventDefault();
      e.stopPropagation();
      onClose();
    }
    document.addEventListener("keydown", onKey, true);
    return () => document.removeEventListener("keydown", onKey, true);
  }, [src, onClose]);
  if (!src) return null;
  return createPortal(
    <div className="img-lite" role="dialog" aria-modal="true" aria-label="Image preview" onClick={onClose}>
      <button type="button" className="img-lite-x" onClick={onClose} aria-label="Close">
        <IconX size={16} />
      </button>
      <img src={src} alt="" />
    </div>,
    document.body,
  );
}
