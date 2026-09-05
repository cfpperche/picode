import { useEffect, useState } from "react";

export default function VoiceMeter({ stream, bars = 16 }) {
  const [lv, setLv] = useState(() => Array(bars).fill(4));

  useEffect(() => {
    if (!stream) return;
    let ac;
    let raf = 0;
    let dead = false;
    const AudioCtx = window.AudioContext || window.webkitAudioContext;
    if (!AudioCtx) return;
    try {
      ac = new AudioCtx();
      const src = ac.createMediaStreamSource(stream);
      const an = ac.createAnalyser();
      an.fftSize = 64;
      src.connect(an);
      const data = new Uint8Array(an.frequencyBinCount);
      const tick = () => {
        if (dead) return;
        an.getByteFrequencyData(data);
        const out = [];
        const step = Math.max(1, Math.floor(data.length / bars));
        for (let i = 0; i < bars; i++) {
          let s = 0;
          for (let j = 0; j < step; j++) s += data[i * step + j] || 0;
          out.push(4 + (s / step / 255) * 16);
        }
        setLv(out);
        raf = requestAnimationFrame(tick);
      };
      tick();
    } catch { /* no analyser in this browser */ }
    return () => {
      dead = true;
      if (raf) cancelAnimationFrame(raf);
      if (ac) ac.close();
    };
  }, [stream, bars]);

  return (
    <span className="dictate-wave" aria-hidden="true">
      {lv.map((h, i) => <i key={i} style={{ height: Math.round(h) + "px" }} />)}
    </span>
  );
}
