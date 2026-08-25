export function speechCtor(win = globalThis) {
  if (!win) return null;
  return win.SpeechRecognition || win.webkitSpeechRecognition || null;
}

export function speechSupported(win = globalThis) {
  return !!speechCtor(win);
}

export function mergeTranscript(base, addition) {
  const a = (base || "").replace(/\s+$/, "");
  const b = (addition || "").replace(/^\s+|\s+$/g, "");
  if (!b) return a;
  if (!a) return b;
  return a + " " + b;
}

export function collectResults(results, fromIndex = 0) {
  let finalText = "";
  let interim = "";
  const list = results || [];
  for (let i = fromIndex; i < list.length; i++) {
    const row = list[i];
    const t = (row && row[0] && row[0].transcript) || "";
    if (row && row.isFinal) finalText += t;
    else interim += t;
  }
  return { final: finalText, interim };
}

export function humanizeSpeechError(code) {
  switch (code) {
    case "not-allowed":
      return "Microphone permission denied.";
    case "audio-capture":
      return "No microphone found.";
    case "network":
      return "Speech recognition needs a network connection.";
    case "no-speech":
    case "aborted":
      return "";
    case "not-supported":
      return "Voice needs Chrome or Edge on HTTPS.";
    default:
      return code ? "Speech recognition failed (" + code + ")." : "Speech recognition failed.";
  }
}

export function discloseSttOnce(notify, store) {
  try {
    const s = store || (typeof localStorage !== "undefined" ? localStorage : null);
    if (s && s.getItem("picode-stt-hint")) return false;
    if (s) s.setItem("picode-stt-hint", "1");
  } catch { /* private mode */ }
  if (notify) notify("Browser speech recognition may send audio to the browser vendor.");
  return true;
}

export async function unlockMic(nav, keep) {
  const n = nav || (typeof navigator !== "undefined" ? navigator : null);
  if (!n || !n.mediaDevices || !n.mediaDevices.getUserMedia) {
    throw new Error("not-supported");
  }
  const stream = await n.mediaDevices.getUserMedia({ audio: true });
  if (!keep) stream.getTracks().forEach((t) => t.stop());
  return stream;
}

export function speakable(text) {
  return String(text || "")
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/[#*_>]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

export function speakText(text, win = globalThis) {
  const synth = win && win.speechSynthesis;
  if (!synth) return false;
  const t = speakable(text);
  if (!t) return false;
  synth.cancel();
  const u = new win.SpeechSynthesisUtterance(t);
  u.lang = (win.navigator && win.navigator.language) || "pt-BR";
  synth.speak(u);
  return true;
}

export function stopSpeak(win = globalThis) {
  if (win && win.speechSynthesis) win.speechSynthesis.cancel();
}

export function createRecognizer(opts = {}, win = globalThis) {
  const Ctor = speechCtor(win);
  if (!Ctor) throw new Error("not-supported");
  const rec = new Ctor();
  rec.continuous = true;
  rec.interimResults = true;
  rec.lang = opts.lang || (win.navigator && win.navigator.language) || "pt-BR";
  rec.onresult = (e) => {
    const { final: fin, interim } = collectResults(e.results, e.resultIndex);
    if (fin && opts.onFinal) opts.onFinal(fin);
    if (interim && opts.onInterim) opts.onInterim(interim);
  };
  rec.onerror = (e) => {
    if (opts.onError) opts.onError((e && e.error) || "");
  };
  rec.onend = () => {
    if (opts.onEnd) opts.onEnd();
  };
  return {
    start() {
      rec.start();
    },
    stop() {
      try { rec.stop(); } catch { /* already stopped */ }
    },
    abort() {
      try { rec.abort(); } catch { /* already stopped */ }
    },
  };
}
