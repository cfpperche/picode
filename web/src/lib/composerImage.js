export const MAX_IMAGES = 4;
export const MAX_IMAGE_BYTES = 4 * 1024 * 1024;

const OK = {
  "image/png": "image/png",
  "image/jpeg": "image/jpeg",
  "image/jpg": "image/jpeg",
  "image/gif": "image/gif",
  "image/webp": "image/webp",
};

export function sniffImage(file) {
  if (!file) return null;
  const mime = OK[(file.type || "").toLowerCase()];
  if (!mime) return null;
  return { mime, name: file.name || "image", size: file.size || 0 };
}

export function readImage(file) {
  return new Promise((resolve, reject) => {
    const info = sniffImage(file);
    if (!info) {
      reject(new Error("not-image"));
      return;
    }
    if (info.size > MAX_IMAGE_BYTES) {
      reject(new Error("too-large"));
      return;
    }
    const r = new FileReader();
    r.onload = () => {
      const s = String(r.result || "");
      const i = s.indexOf(",");
      resolve({ ...info, data: i >= 0 ? s.slice(i + 1) : s, url: s });
    };
    r.onerror = () => reject(new Error("read"));
    r.readAsDataURL(file);
  });
}
