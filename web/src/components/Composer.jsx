import { useEffect, useRef, useState } from "react";
import ProviderChip from "./ProviderChip.jsx";
import ModelChip from "./ModelChip.jsx";
import ThinkingChip from "./ThinkingChip.jsx";
import ModeChip from "./ModeChip.jsx";
import KindChip from "./KindChip.jsx";
import { IconSend, IconStop, IconExpand, IconCollapse, IconMic, IconWave, IconSpeaker } from "./Icons.jsx";
import ComposerStatus from "./ComposerStatus.jsx";
import { filterSlash } from "../lib/slash.js";
import { newHist, histPush, histUp, histDown, histTyped, caretFirstLine, caretLastLine } from "../lib/composerHist.js";
import {
  speechSupported, createRecognizer, mergeTranscript, humanizeSpeechError, discloseSttOnce,
} from "../lib/speech.js";
import { toast } from "../lib/toast.js";

export default function Composer({
  kind, onKind, value, onChange, onSend, status, streaming,
  stopped, onToggleDock, onStop, onAbort, catalog, cfg, onConfig, onSlash, statusBar, onCompact, sessionBar,
}) {
  const ta = useRef(null);
  const hist = useRef(newHist());
  const rec = useRef(null);
  const wantListen = useRef(false);
  const modeRef = useRef("off");
  const finals = useRef("");
  const dictateBase = useRef("");
  const valueRef = useRef(value);
  const streamingRef = useRef(!!streaming);
  const [slashIdx, setSlashIdx] = useState(0);
  const [expanded, setExpanded] = useState(false);
  const [voice, setVoice] = useState(false);
  const [listening, setListening] = useState(false);
  const [caption, setCaption] = useState("");
  const hits = filterSlash(value);
  const canTalk = speechSupported();

  useEffect(() => { valueRef.current = value; }, [value]);
  useEffect(() => { streamingRef.current = !!streaming; }, [streaming]);

  useEffect(() => {
    const el = ta.current;
    if (!el) return;
    if (expanded) { el.style.height = "100%"; return; }
    el.style.height = "auto";
    el.style.height = Math.max(52, Math.min(el.scrollHeight, 160)) + "px";
  }, [value, expanded, voice]);

  useEffect(() => { setSlashIdx(0); }, [value]);

  useEffect(() => {
    return () => stopRec(true);
  }, []);

  useEffect(() => {
    if (!voice) return;
    if (streaming) {
      wantListen.current = false;
      rec.current && rec.current.stop();
      setListening(false);
      return;
    }
    startListen("voice");
  }, [streaming, voice]);

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
        if (!voice) toggleDictate();
        return;
      }
      if (e.key === "Escape") {
        if (voice) { e.preventDefault(); leaveVoice(); return; }
        if (expanded) { e.preventDefault(); setExpanded(false); }
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [voice, expanded, listening, caption, streaming]);

  function pickSlash(cmd) {
    onChange("");
    if (!cmd) return;
    if (onSlash) onSlash(cmd);
  }

  function stopRec(abort) {
    wantListen.current = false;
    if (rec.current) {
      if (abort) rec.current.abort();
      else rec.current.stop();
      rec.current = null;
    }
    setListening(false);
  }

  function startListen(mode) {
    if (!canTalk) {
      toast.error(humanizeSpeechError("not-supported"));
      return;
    }
    stopRec(true);
    modeRef.current = mode;
    finals.current = "";
    if (mode === "dictate") dictateBase.current = valueRef.current || "";
    discloseSttOnce((m) => toast.info(m));
    wantListen.current = true;
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
          if (wantListen.current && rec.current) {
            try { rec.current.start(); setListening(true); } catch { /* restart race */ }
          }
        },
      });
      rec.current.start();
      setListening(true);
    } catch (e) {
      toast.error(humanizeSpeechError(e && e.message));
    }
  }

  function toggleDictate() {
    if (listening && modeRef.current === "dictate") {
      stopRec(false);
      modeRef.current = "off";
      return;
    }
    startListen("dictate");
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
    stopRec(true);
    modeRef.current = "off";
    setVoice(false);
    setCaption("");
    finals.current = "";
    if (leftover && !streamingRef.current) onChange(mergeTranscript(valueRef.current, leftover));
  }

  function interrupt() {
    if (streaming) {
      if (onAbort) onAbort();
      return;
    }
    stopRec(false);
  }

  const dictating = listening && !voice;

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
        {hits.length > 0 && !voice && (
          <ul className="slash-menu" role="listbox">
            {hits.map((c, i) => (
              <li key={c.id}>
                <button
                  type="button"
                  className={"slash-item" + (i === slashIdx ? " active" : "")}
                  onMouseEnter={() => setSlashIdx(i)}
                  onClick={() => pickSlash(c)}
                >
                  <span className="slash-label">{c.label}</span>
                  <span className="slash-hint">{c.hint}</span>
                </button>
              </li>
            ))}
          </ul>
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
            placeholder="Message the agent, or / for commands"
            value={value}
            onChange={(e) => { histTyped(hist.current); onChange(e.target.value); }}
            onKeyDown={(e) => {
              if (hits.length) {
                if (e.key === "ArrowDown") { e.preventDefault(); setSlashIdx((i) => Math.min(hits.length - 1, i + 1)); return; }
                if (e.key === "ArrowUp") { e.preventDefault(); setSlashIdx((i) => Math.max(0, i - 1)); return; }
                if (e.key === "Tab" || (e.key === "Enter" && !e.shiftKey)) { e.preventDefault(); pickSlash(hits[slashIdx]); return; }
                if (e.key === "Escape") { e.preventDefault(); onChange(""); return; }
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
                  onClick={() => listening ? stopRec(false) : startListen("voice")}
                >
                  <IconMic />
                </button>
                <button
                  type="button"
                  className="icon-btn"
                  title="Spoken replies come in V2"
                  disabled
                >
                  <IconSpeaker />
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
            ) : (
              <>
                <button
                  type="button"
                  className={"icon-btn" + (dictating ? " listening" : "")}
                  title="Dictation (Ctrl+D)"
                  aria-label="Dictation"
                  aria-pressed={dictating}
                  onClick={toggleDictate}
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
