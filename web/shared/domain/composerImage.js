export const MAX_IMAGES = 4;
export const MAX_IMAGE_BYTES = 4 * 1024 * 1024;

const OK = {
  "image/png": "image/png",
  "image/jpeg": "image/jpeg",
  "image/jpg": "image/jpeg",
  "image/gif": "image/gif",
  "image/webp": "image/webp",
};

export function sceneHasInk(elements) {
  return (elements || []).some((el) => el && el.isDeleted !== true);
}

export function sniffImage(file) {
  if (!file) return null;
  const mime = OK[(file.type || "").toLowerCase()];
  if (!mime) return null;
  return { mime, name: file.name || "image", size: file.size || 0 };
}

// Partition a device file list (Photos / picker) before readImage.
// already = chips already on the composer.
export function planDeviceImages(files, already) {
  const list = [...(files || [])].filter(Boolean);
  const out = [];
  let notImage = 0;
  let tooLarge = 0;
  let tooMany = false;
  const have = Math.max(0, already | 0);
  for (const f of list) {
    const info = sniffImage(f);
    if (!info) {
      notImage += 1;
      continue;
    }
    if (info.size > MAX_IMAGE_BYTES) {
      tooLarge += 1;
      continue;
    }
    if (have + out.length >= MAX_IMAGES) {
      tooMany = true;
      continue;
    }
    out.push(f);
  }
  return { files: out, notImage, tooLarge, tooMany };
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
