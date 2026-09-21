// The desktop shell's clipboard-file door. Browsers never expose Explorer
// file copies as bytes (VSCode #301603 went native for the same reason),
// but the Windows-native shell reads CF_HDROP and hands the bytes back, so
// the existing drop flow stages them unchanged. Real browsers pay nothing:
// every entry self-gates on the Tauri invoke.
export function shellInvoke() {
  return (typeof window !== "undefined" &&
    window.__TAURI__ && window.__TAURI__.core &&
    typeof window.__TAURI__.core.invoke === "function")
    ? window.__TAURI__.core.invoke
    : null;
}

// shellClipboardFiles(): File[] staged from the OS clipboard, [] when it
// holds no files, null outside the shell or when the shell refuses. Caps
// live on both sides: the shell reads at most 4 files of 4 MB, and the
// attach bar re-plans whatever arrives.
export async function shellClipboardFiles(invoke) {
  const call = invoke === undefined ? shellInvoke() : invoke;
  if (!call) return null;
  try {
    const rows = await call("clipboard_files");
    const files = [];
    for (const row of rows || []) {
      try {
        files.push(clipboardFileFromRow(row));
      } catch { /* one bad row never sinks the paste */ }
    }
    return files;
  } catch {
    return null;
  }
}
// clipboardFileFromRow: a base64 shell row → File. Pure, so the conversion
// is unit-tested without a shell. A row without data is not a file and
// throws, so the caller skips it; a present-but-empty payload stays a
// legitimate 0-byte file.
export function clipboardFileFromRow(row) {
  if (!row || typeof row !== "object" || row.data == null) throw new Error("empty row");
  const name = String(row.name || "file");
  const mime = String(row.mime || "application/octet-stream");
  const bin = atob(String(row.data));
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return new File([bytes], name, { type: mime });
}
