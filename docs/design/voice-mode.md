# Voice in PiCode

- **Date:** 2026-08-25 (updated same day after V1 ship)
- **Status:** V1 shipped. Owner dogfooding. Later phases below.
- **Brief:** ChatGPT idle (mic + waveform) + **Grok x.ai voice composer**.
  Owner screenshots `122712` / `122725` (ChatGPT) and `123411` / `123433`
  (Grok: voice replaces the composer, chat stays).
- **Audience:** coding ADE, not a chatbot. The agent stays `pi`.

## 1. What the screenshots ask for

ChatGPT ships **two** controls, not one "voice button":

| Control | Label (PT) | Shortcut | Job |
|---|---|---|---|
| Mic | Ditadura | Ctrl+D | Speech → text **in the composer**. User edits, then Send. |
| Waveform | Entrar no modo de voz | Ctrl+Shift+O | Swap the composer into **voice composer**. Chat stays. |

Grok on x.ai (123411): same pill, textarea gone, caption + mic/speaker/
expand cluster + **Interrupt**. That is the voice-mode chrome. ChatGPT
full-bleed orb is **not** the pattern.

## 2. Benchmarks (what we steal, what we refuse)

### ChatGPT (the visual brief)

- Two affordances, right of the input, next to send.
- Dictation is a *modality of the composer*. Voice mode is a *place*.
- Voice mode is full-bleed: orb/waveform, live captions, barge-in.

### Cursor

Composer-class dictation (talk into the prompt, still an editor). No
ChatGPT-style spoken coding agent. **Steal dictation placement. Do not
become an editor.**

### t3code

Composer depth (drafts, queue, waiting). No voice receipt. **Steal
nothing audio-specific.** Voice still has to respect `steer`/`follow_up`
when the agent is busy.

### paseo (AgentDeck study 2026-08-23)

Native apps add **voice input**; they also advertise a Voice mode with
zero telemetry. Our 2026-08-24 study deferred "Voice and pairing-relay"
because we are not a fleet client. **That deferral was about paseo's
relay, not about a mic in our own HTTPS app.** Owner brief reopens voice
as a PiCode surface.

### OpenAI platform (2026 docs)

They themselves split the architecture:

| Goal | API |
|---|---|
| Live spoken agent | Realtime / `gpt-realtime-*` (speech-to-speech) |
| Live captions only | Realtime transcription |
| File / bounded clip | `/v1/audio/transcriptions` (`gpt-transcribe`) |
| Speak text | TTS |

Quote of the decision we copy: *speech-to-speech for natural talk;
chained STT → existing text agent → TTS when you need control,
transcripts, and tools.*

A coding ADE is the second case.

### xAI / Grok 4.6 (current catalog)

`grok-4.6` input = TEXT+IMAGE, output = TEXT. There is no audio
modality on the model the owner is using. Voice cannot mean "ask Grok
in audio." It means STT in, optional TTS out, **same text agent**.

### Pi ecosystem (npm `keywords:pi-package` + voice)

TUI already has dictation packages: `pi-voice-stt`, `@artale/pi-voice`,
`@juicesharp/rpiv-voice` (local sherpa-onnx), `@codexstar/pi-listen`
(STT+TTS), `@maddeye/pi-voice`, others. **ADR-0010 still holds:** we
do not ship a parallel marketplace or a PiCode-owned STT engine.
Those packages remain the **terminal escape hatch**. The browser cannot
reuse their OS-mic capture; it needs `getUserMedia`.

## 3. Decisions

### D1 — Two products, never one button

Dictation and voice mode stay separate. V1 shipped both chrome and a
first TTS. Later phases polish engines, not the split.

Why: ChatGPT idle split; coding needs an edit gate on dictation; Grok
shows voice as a *composer state*, not a new route.

### D2 — The coding brain stays `pi` (chained pipeline)

```
mic → STT → composer or prompt RPC → pi (tools, JSONL) → text
                                              ↘ optional TTS
```

**Refuse** wiring OpenAI Realtime / Grok Voice as the agent. That would
bypass ADR-0003 (user-installed pi), ADR-0005 (history is JSONL), and
the Worked/error/rail UI. Philosophy §2: GUI convenience, TUI escape
hatch. Speech-to-speech is a different product.

### D3 — Dictation fills the composer; it does not send

Toggle (Ctrl+D). Recording replaces the send cluster with live
waveform + X (revert) + check (keep text). User then hits Send.
Hold-to-talk is later. Auto-send stays off for dictation.

Why: speech-to-code is lossy (`rm -rf`, paths, package names). Cursor
and ChatGPT dictation both land in an editable field.

### D4 — Voice mode replaces the composer, not the chat

Grok x.ai: activating voice **swaps the composer innards**. Conversation
stays. Session toolbar stays. Model chips hide; Interrupt takes Send's
place.

```
idle:     [textarea]  [chips…] [mic] [wave] [send]
voice:    [caption ]  [wave mic speaker]     [Interrupt]
```

Not an exclusive view (that was wrong — ChatGPT orb, not Grok).
Chat/Terminal exclusivity is unchanged. Voice is a composer *mode*.

Turns still go through `sendTask` → pi JSONL.

### D5 — STT v1 = Web Speech API; cloud is opt-in

Chrome/Edge on the owner's HTTPS origin (`ADR-0007`) already can
`webkitSpeechRecognition`. Zero keys, Portuguese works, interim
results exist.

Disclose: Chromium often ships audio to Google. Settings line:
"Browser speech recognition may send audio to the browser vendor."

Fallback (V1.1, not V1): `MediaRecorder` → existing provider key
(OpenAI `gpt-transcribe`, Groq Whisper) if the user turns it on.
No PiCode-billed STT. No default package install.

Why: matches "nothing installed by default" (ADR-0010) and "one
binary, working."

### D6 — TTS = `speechSynthesis`, voice-mode only (shipped in V1)

Dictation does not speak. Speaker toggles mute. Unmute speaks the last
assistant **text** (not tool dumps, not thinking). Interrupt cancels TTS.
Neural / cloud TTS is V3.

Why: the TUI packages already cover local neural TTS for people who
want it in the terminal. Browser synthesis is the door, not the cage.

### D7 — Placement: composer-right, left of Send

Session toolbar (above the textarea) is **session actions**.
Provider/model chips are **model config**. Mic + waveform are **input
modality**, ChatGPT-right. Expand stays top-right of the card.

Shortcuts (from the brief, unused today except Ctrl+K palette):

- `Ctrl+D` dictation toggle
- `Ctrl+Shift+O` enter/leave voice mode

### D8 — Permissions and WSL

Mic requires a user gesture + browser permission prompt (we cannot
replace that with Radix). Feature-detect; if denied, toast with the
real error, not a fake settings page.

WSL: Chrome **on Windows** hitting `https://localhost:8445` sees the
Windows mic. A Linux browser inside WSL often has **no** capture
device — empty-state copy must say so. Do not fake a waveform.

### D9 — What never ships

- Audio blobs in session JSONL (history is text; ADR-0005).
- Always-on listening.
- Invented visualizer without analyser data.
- Replacing Stop/Send with a voice orb on the chat surface.
- A PiCode STT/TTS marketplace.
- Auto-send from **dictation** (voice composer may send on silence).

## 4. Phases

### V1 — shipped 2026-08-25

Idle mic (filled chip) + waveform. Dictation bar: live `AnalyserNode`
levels, X cancels, check keeps text. Voice composer (Grok): caption,
mic/speaker/Interrupt. Silence sends through `sendTask`. Speaker =
browser `speechSynthesis` of last assistant text. Mic uses
`getUserMedia` then Web Speech API. First-use vendor-STT toast.

Code: `web/src/lib/speech.js`, `VoiceMeter.jsx`, `Composer.jsx`.

### V1.1 — after dogfood (only if needed)

If Chromium PT-BR is bad: opt-in cloud STT (`MediaRecorder` → OpenAI
`gpt-transcribe` or Groq Whisper with keys already in Providers).
Empty copy when WSL Linux has no capture device.
Hold-to-talk as an alternate dictate gesture.

Do **not** start V1.1 until the owner reports a real quality gap.

### V2 — barge-in and turn-taking

- Interrupt while TTS plays: `speechSynthesis.cancel` + Abort RPC
  (partially there; make it reliable when the agent is mid-tool).
- Pause listen for the full turn, not only `streaming` boolean.
- Speak **after** `agent_settled`, skip thinking/tool dumps (already
  last assistant block; tighten to final reply only).
- Voice composer caption shows the in-flight transcript more clearly.

Still the Grok composer, not a ChatGPT orb.

### V3 — cloud STT/TTS (opt-in, no default)

Settings: STT = Browser | OpenAI | Groq. TTS = Browser | OpenAI.
Packages page may *link* TUI voice packages (`pi-voice-stt`, …) as
the terminal escape hatch. Nothing installed by default (ADR-0010).

### Out of scope until pi itself speaks audio

Speech-to-speech coding agent (OpenAI Realtime / Grok Voice as the
brain). MCP audio tools. Mobile-first voice chrome (after V2 desktop).

## 5. Why this and not the alternatives

| Alternative | Why not |
|---|---|
| One mic that auto-sends | Coding transcripts need an edit gate |
| OpenAI Realtime as the agent | Forks pi, drops tools/JSONL, billed to us or forces OpenAI |
| Wrap `pi-voice-stt` in the browser | TUI packages capture OS mic; browser needs getUserMedia |
| Ship Whisper in the Go binary | Fat binary, not a pi primitive, contradicts "standard library first" |
| Exclusive full-pane voice (ChatGPT orb) | Grok keeps the chat; we copy that |
| Voice overlay on Chat | Repeats the managed-vs-TUI confusion we just killed |
| Wait for Grok audio | grok-4.6 has no audio I/O; we'd stall the brief |

## 6. Open questions (dogfood)

1. Hold-to-talk vs toggle — **toggle shipped**. Hold only if asked.
2. Auto-send on silence in voice mode — **yes, shipped**. Dictation stays manual.
3. Cloud STT — **V1.1, only if PT-BR on Chrome is not good enough**.
4. Neural TTS — V3, not before cloud STT is decided.
