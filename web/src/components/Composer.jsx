import { useEffect, useRef, useState } from "react";
import ProviderChip from "./ProviderChip.jsx";
import ModelChip from "./ModelChip.jsx";
import ThinkingChip from "./ThinkingChip.jsx";
import ModeChip from "./ModeChip.jsx";
import KindChip from "./KindChip.jsx";
import { IconSend, IconStop, IconExpand, IconCollapse, IconMic, IconWave, IconSpeaker, IconSpeakerOff, IconX, IconCheck, IconDocs } from "./Icons.jsx";
import VoiceMeter from "./VoiceMeter.jsx";
import ComposerStatus from "./ComposerStatus.jsx";
import { Command } from "cmdk";
import { api } from "../lib/api.js";
import { filterSlash } from "../lib/slash.js";
import { atQuery, insertAtPath } from "../lib/atMention.js";
import { commandDocUrl } from "../lib/commandDocs.js";
import { newHist, histPush, histUp, histDown, histTyped, caretFirstLine, caretLastLine } from "../lib/composerHist.js";
import {
  speechSupported, createRecognizer, mergeTranscript, humanizeSpeechError, discloseSttOnce,
  unlockMic, speakText, stopSpeak,
} from "../lib/speech.js";
import { toast } from "../lib/toast.js";

export default function Composer({
  kind, onKind, value, onChange, onSend, status, streaming,
  stopped, onToggleDock, onStop, onAbort, catalog, cfg, onConfig, onSlash, statusBar, onCompact, sessionBar, lastReply,
  slashExtra, agentId,
}) {
  const ta = useRef(null);
  const hist = useRef(newHist());
  const rec = useRef(null);
  const wantListen = useRef(false);
  const gen = useRef(0);
  const modeRef = useRef("off");
  const finals = useRef("");
  const dictateBase = useRef("");
  const valueRef = useRef(value);
  const streamingRef = useRef(!!streaming);
  const prevStream = useRef(!!streaming);
  const mutedRef = useRef(false);
  const streamRef = useRef(null);
  const [slashIdx, setSlashIdx] = useState(0);
  const [expanded, setExpanded] = useState(false);
  const [voice, setVoice] = useState(false);
  const [listening, setListening] = useState(false);
  const [caption, setCaption] = useState("");
  const [muted, setMuted] = useState(false);
  const [dictate, setDictate] = useState(false);
  const [micStream, setMicStream] = useState(null);
  const [caret, setCaret] = useState(0);
  const [atHits, setAtHits] = useState(null);
  const [atOk, setAtOk] = useState(false);
  const [atIdx, setAtIdx] = useState(0);
  const [atHide, setAtHide] = useState("");
  const hits = filterSlash(value, slashExtra);
  const at = hits.length ? null : atQuery(value, caret);
  const atKey = at ? "@" + at.query : "";
  const showAt = !!(at && atOk && atHits && atHide !== atKey);

  useEffect(() => { valueRef.current = value; }, [value]);
  useEffect(() => { streamingRef.current = !!streaming; }, [streaming]);
  useEffect(() => { mutedRef.current = muted; }, [muted]);

  useEffect(() => {
    const el = ta.current;
    if (!el) return;
    if (expanded) { el.style.height = "100%"; return; }
    el.style.height = "auto";
    el.style.height = Math.max(52, Math.min(el.scrollHeight, 160)) + "px";
  }, [value, expanded, voice]);

  useEffect(() => { setSlashIdx(0); }, [value]);
  useEffect(() => { setAtIdx(0); }, [atKey]);

  useEffect(() => {
    if (!agentId || !atKey || hits.length) {
      setAtHits(null);
      setAtOk(false);
      return;
    }
    const q = atKey.slice(1);
    const t = setTimeout(async () => {
      try {
        const d = await api("/api/agents/" + encodeURIComponent(agentId) + "/files?q=" + encodeURIComponent(q));
        if (!d.cwdOk) { setAtHits(null); setAtOk(false); return; }
        setAtOk(true);
        setAtHits(d.hits || []);
      } catch {
        setAtHits(null);
        setAtOk(false);
      }
    }, 120);
    return () => clearTimeout(t);
  }, [agentId, atKey, hits.length]);

  useEffect(() => {
    return () => stopRec();
  }, []);

  useEffect(() => {
    if (!voice) return;
    if (streaming) {
      stopRec();
      stopSpeak();
      return;
    }
    startListen("voice");
  }, [streaming, voice]);

  useEffect(() => {
    const was = prevStream.current;
    prevStream.current = !!streaming;
    if (voice && was && !streaming && !mutedRef.current && lastReply) {
      speakText(lastReply);
    }
  }, [streaming, voice, lastReply]);

  useEffect(() => {
    const onKey = (e) => {
      const mod = e.ctrlKey || e.metaKey;
      if (mod && e.shiftKey && e.key.toLowerCase() === "o") {
        e.preventDefault();
        toggleVoice();
        return;
      }
      if (mod && !e.shiftKey && e.key.toLowerCase() === "d") {
        e.preventDefault();
        if (voice) return;
        if (dictate) confirmDictate();
        else startListen("dictate");
        return;
      }
      if (e.key === "Escape") {
        if (dictate) { e.preventDefault(); cancelDictate(); return; }
        if (voice) { e.preventDefault(); leaveVoice(); return; }
        if (expanded) { e.preventDefault(); setExpanded(false); }
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [voice, expanded, listening, caption, streaming, dictate]);

  function markCaret(el) {
    if (el && typeof el.selectionStart === "number") setCaret(el.selectionStart);
  }

  function pickAt(hit) {
    if (!hit) return;
    const next = insertAtPath(value, caret, hit.path);
    onChange(next.text);
    setAtHits(null);
    setAtHide(atKey);
    requestAnimationFrame(() => {
      const el = ta.current;
      if (!el) return;
      el.focus();
      el.setSelectionRange(next.caret, next.caret);
      setCaret(next.caret);
    });
  }

  function pickSlash(cmd) {
    if (!cmd) return;
    if (cmd.run === "insert") {
      onChange(cmd.insert || cmd.label + " ");
      requestAnimationFrame(() => ta.current?.focus());
      return;
    }
    onChange("");
    if (cmd.run === "copy") {
      const t = lastReply || "";
      if (!t) { toast.info("No assistant reply yet."); return; }
      navigator.clipboard.writeText(t).then(() => toast.ok("Copied last reply.")).catch(() => toast.error("Clipboard blocked."));
      return;
    }
    if (onSlash) onSlash(cmd);
  }

  function stopRec() {
    wantListen.current = false;
    gen.current += 1;
    const r = rec.current;
    rec.current = null;
    setListening(false);
    if (r) try { r.abort(); } catch { /* already stopped */ }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
    }
    setMicStream(null);
  }

  async function startListen(mode) {
    const my = ++gen.current;
    wantListen.current = false;
    const prev = rec.current;
    rec.current = null;
    setListening(false);
    if (prev) try { prev.abort(); } catch { /* already stopped */ }
    if (mode === "dictate") setDictate(true);

    if (!speechSupported()) {
      toast.error(humanizeSpeechError("not-supported"));
      setDictate(false);
      return;
    }
    try {
      const stream = await unlockMic(undefined, true);
      if (my !== gen.current) {
        stream.getTracks().forEach((t) => t.stop());
        return;
      }
      streamRef.current = stream;
      setMicStream(stream);
    } catch {
      if (my !== gen.current) return;
      toast.error("Microphone permission denied.");
      setCaption("Microphone blocked — click the mic to retry");
      setDictate(false);
      return;
    }
    if (my !== gen.current) return;
    await new Promise((ok) => setTimeout(ok, 80));
    if (my !== gen.current) return;

    discloseSttOnce((m) => toast.info(m));
    wantListen.current = true;
    modeRef.current = mode;
    finals.current = "";
    if (mode === "dictate") dictateBase.current = valueRef.current || "";
    try {
      rec.current = createRecognizer({
        onInterim: (t) => {
          const shown = mergeTranscript(finals.current, t);
          setCaption(shown);
          if (modeRef.current === "dictate") onChange(mergeTranscript(dictateBase.current, shown));
        },
        onFinal: (t) => {
          finals.current = mergeTranscript(finals.current, t);
          setCaption(finals.current);
          if (modeRef.current === "dictate") onChange(mergeTranscript(dictateBase.current, finals.current));
        },
        onError: (code) => {
          const msg = humanizeSpeechError(code);
          if (msg) toast.error(msg);
          if (code === "not-allowed" || code === "audio-capture") {
            wantListen.current = false;
            setListening(false);
            setCaption("Microphone blocked — click the mic to retry");
            setDictate(false);
          }
        },
        onEnd: () => {
          setListening(false);
          if (modeRef.current === "voice" && !streamingRef.current && finals.current.trim()) {
            const text = finals.current.trim();
            finals.current = "";
            setCaption("");
            histPush(hist.current, text);
            onSend(text);
          }
          if (wantListen.current && rec.current && my === gen.current) {
            try { rec.current.start(); setListening(true); } catch { /* restart race */ }
          }
        },
      });
      rec.current.start();
      setListening(true);
    } catch (e) {
      toast.error(humanizeSpeechError((e && e.message) || "failed"));
    }
  }

  function confirmDictate() {
    stopRec();
    modeRef.current = "off";
    setDictate(false);
    setCaption("");
  }

  function cancelDictate() {
    const base = dictateBase.current || "";
    stopRec();
    modeRef.current = "off";
    setDictate(false);
    setCaption("");
    onChange(base);
  }

  function toggleDictate() {
    if (dictate) confirmDictate();
    else startListen("dictate");
  }

  function toggleVoice() {
    if (voice) leaveVoice();
    else enterVoice();
  }

  function enterVoice() {
    setVoice(true);
  }

  function leaveVoice() {
    const leftover = (caption || finals.current || "").trim();
    stopRec();
    stopSpeak();
    modeRef.current = "off";
    setVoice(false);
    setCaption("");
    finals.current = "";
    if (leftover && !streamingRef.current) onChange(mergeTranscript(valueRef.current, leftover));
  }

  function interrupt() {
    stopSpeak();
    if (streaming) {
      if (onAbort) onAbort();
      return;
    }
    stopRec();
  }

  function toggleMute() {
    setMuted((m) => {
      const next = !m;
      if (next) stopSpeak();
      else if (lastReply) {
        const ok = speakText(lastReply);
        if (!ok) toast.error("This browser cannot speak replies.");
      } else {
        toast.info("I'll speak the agent's replies.");
      }
      return next;
    });
  }


  return (
    <div className={"composer-wrap" + (expanded ? " expanded" : "") + (voice ? " voice" : "")}>
      <div className="composer" onClick={(e) => { if (e.target === e.currentTarget) ta.current?.focus(); }}>
        <button
          type="button"
          className="composer-expand"
          title={expanded ? "Collapse" : "Expand"}
          aria-label={expanded ? "Collapse composer" : "Expand composer"}
          onClick={() => setExpanded((v) => !v)}
        >
          {expanded ? <IconCollapse /> : <IconExpand />}
        </button>
        {showAt && !voice && (
          <Command
            className="slash-menu"
            shouldFilter={false}
            loop
            label="Files"
            value={atHits[atIdx] ? atHits[atIdx].path : ""}
            onValueChange={(id) => {
              const i = atHits.findIndex((h) => h.path === id);
              if (i >= 0) setAtIdx(i);
            }}
          >
            <Command.List>
              {atHits.length === 0 ? (
                <div className="slash-empty">No files</div>
              ) : atHits.map((h) => (
                <Command.Item
                  key={h.path}
                  value={h.path}
                  className={"slash-item" + (h.path === (atHits[atIdx] && atHits[atIdx].path) ? " active" : "")}
                  onSelect={() => pickAt(h)}
                >
                  <span className="slash-label">@{h.path}</span>
                </Command.Item>
              ))}
            </Command.List>
          </Command>
        )}
        {hits.length > 0 && !voice && (
          <Command
            className="slash-menu"
            shouldFilter={false}
            loop
            value={hits[slashIdx] ? hits[slashIdx].id : ""}
            onValueChange={(id) => {
              const i = hits.findIndex((c) => c.id === id);
              if (i >= 0) setSlashIdx(i);
            }}
          >
            <Command.List>
              {hits.map((c) => (
                <Command.Item
                  key={c.id}
                  value={c.id}
                  className={"slash-item" + (c.id === (hits[slashIdx] && hits[slashIdx].id) ? " active" : "")}
                  onSelect={() => pickSlash(c)}
                >
                  <span className="slash-label">{c.label}</span>
                  {c.docs === false ? (
                    <span className="slash-hint">{c.hint}</span>
                  ) : (
                  <a
                    className="slash-hint"
                    href={commandDocUrl(c.id)}
                    target="_blank"
                    rel="noreferrer"
                    onPointerDown={(e) => e.stopPropagation()}
                    onClick={(e) => e.stopPropagation()}
                  >
                    <IconDocs />
                    {c.hint}
                  </a>
                  )}
                </Command.Item>
              ))}
            </Command.List>
          </Command>
        )}
        {sessionBar ? <div className="composer-tools">{sessionBar}</div> : null}
        {voice ? (
          <div className="composer-voice-body">
            <p className={"composer-voice-caption" + (caption ? "" : " placeholder")}>
              {caption || (listening ? "Listening…" : streaming ? "Working…" : "Speak to the agent")}
            </p>
          </div>
        ) : (
          <textarea
            id="task-input"
            ref={ta}
            rows={2}
            placeholder="Message the agent, / for commands, or @ a file"
            value={value}
            onChange={(e) => { histTyped(hist.current); onChange(e.target.value); markCaret(e.target); }}
            onSelect={(e) => markCaret(e.target)}
            onClick={(e) => markCaret(e.target)}
            onKeyUp={(e) => markCaret(e.target)}
            onKeyDown={(e) => {
              if (hits.length) {
                if (e.key === "ArrowDown") { e.preventDefault(); setSlashIdx((i) => Math.min(hits.length - 1, i + 1)); return; }
                if (e.key === "ArrowUp") { e.preventDefault(); setSlashIdx((i) => Math.max(0, i - 1)); return; }
                if (e.key === "Tab" || (e.key === "Enter" && !e.shiftKey)) { e.preventDefault(); pickSlash(hits[slashIdx]); return; }
                if (e.key === "Escape") { e.preventDefault(); onChange(""); return; }
              }
              if (showAt) {
                if (e.key === "ArrowDown") { e.preventDefault(); setAtIdx((i) => Math.min(atHits.length - 1, i + 1)); return; }
                if (e.key === "ArrowUp") { e.preventDefault(); setAtIdx((i) => Math.max(0, i - 1)); return; }
                if (e.key === "Tab" || (e.key === "Enter" && !e.shiftKey)) {
                  e.preventDefault();
                  if (atHits[atIdx]) pickAt(atHits[atIdx]);
                  return;
                }
                if (e.key === "Escape") { e.preventDefault(); setAtHide(atKey); return; }
              }
              if (e.key === "ArrowUp" && caretFirstLine(ta.current)) {
                e.preventDefault();
                onChange(histUp(hist.current, value || ""));
                requestAnimationFrame(() => { const el = ta.current; if (el) el.setSelectionRange(el.value.length, el.value.length); });
                return;
              }
              if (e.key === "ArrowDown" && caretLastLine(ta.current)) {
                e.preventDefault();
                onChange(histDown(hist.current, value || ""));
                requestAnimationFrame(() => { const el = ta.current; if (el) el.setSelectionRange(el.value.length, el.value.length); });
                return;
              }
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                histPush(hist.current, value);
                onSend();
              }
            }}
          />
        )}
        <div className="composer-controls">
          {voice ? (
            <div className="composer-left" data-align-row>
              <div className="voice-cluster">
                <button
                  type="button"
                  className="icon-btn"
                  title="Back to text (Ctrl+Shift+O)"
                  aria-label="Back to text"
                  onClick={leaveVoice}
                >
                  <IconWave />
                </button>
                <button
                  type="button"
                  className={"icon-btn" + (listening ? " listening" : "")}
                  title={listening ? "Stop listening" : "Listen"}
                  aria-pressed={listening}
                  onClick={() => listening ? stopRec() : startListen("voice")}
                >
                  <IconMic />
                </button>
                <button
                  type="button"
                  className={"icon-btn" + (muted ? " muted" : "")}
                  title={muted ? "Unmute replies" : "Mute replies"}
                  aria-pressed={!muted}
                  onClick={toggleMute}
                >
                  {muted ? <IconSpeakerOff /> : <IconSpeaker />}
                </button>
              </div>
            </div>
          ) : (
            <div className="composer-left">
              <ProviderChip catalog={catalog} cfg={cfg} onChange={onConfig || (() => {})} />
              <ModelChip catalog={catalog} cfg={cfg} onChange={onConfig || (() => {})} />
              <ThinkingChip catalog={catalog} cfg={cfg} onChange={onConfig || (() => {})} />
              <ModeChip cfg={cfg} onChange={onConfig || (() => {})} />
              <KindChip value={kind} onChange={onKind} />
            </div>
          )}
          <div className="composer-right" data-align-row>
            <span id="chat-status-text" className="sr-only">{status}{streaming ? " streaming" : ""}</span>
            {voice ? (
              <button type="button" className="btn-voice-interrupt" id="btn-voice-interrupt" onClick={interrupt}>
                Interrupt
              </button>
            ) : dictate ? (
              <div className="dictate-bar" data-align-row>
                <VoiceMeter stream={micStream} />
                <button type="button" className="icon-btn" title="Cancel dictation" aria-label="Cancel dictation" onClick={cancelDictate}>
                  <IconX />
                </button>
                <button type="button" className="icon-btn icon-btn-ok" title="Done" aria-label="Confirm dictation" onClick={confirmDictate}>
                  <IconCheck />
                </button>
              </div>
            ) : (
              <>
                <button
                  type="button"
                  className="icon-btn icon-btn-mic"
                  title="Dictation (Ctrl+D)"
                  aria-label="Dictation"
                  onClick={() => startListen("dictate")}
                >
                  <IconMic />
                </button>
                <button
                  type="button"
                  className="icon-btn icon-btn-wave"
                  title="Voice mode (Ctrl+Shift+O)"
                  aria-label="Enter voice mode"
                  onClick={enterVoice}
                >
                  <IconWave />
                </button>
                {streaming ? (
                  <button id="task-abort" type="button" className="icon-btn icon-btn-stop" title="Stop" onClick={onAbort}>
                    <IconStop size={16} />
                  </button>
                ) : (
                  <button id="task-send" type="button" className="icon-btn icon-btn-send" title="Send" disabled={!value || !value.trim()} onClick={() => { histPush(hist.current, value); onSend(); }}>
                    <IconSend size={16} />
                  </button>
                )}
              </>
            )}
          </div>
        </div>
        <ComposerStatus bar={statusBar} onCompact={onCompact} />
      </div>
    </div>
  );
}
