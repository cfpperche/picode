# Voice in PiCode

- **Date:** 2026-08-25
- **Status:** plan (not implemented)
- **Brief:** ChatGPT composer — dictation (mic, Ctrl+D) vs voice mode
  (waveform, Ctrl+Shift+O). Owner screenshots
  `Screenshot 2026-08-25 122712.png` / `122725.png`.
- **Audience:** coding ADE, not a chatbot. The agent stays `pi`.

## 1. What the screenshots ask for

ChatGPT ships **two** controls, not one "voice button":

| Control | Label (PT) | Shortcut | Job |
|---|---|---|---|
| Mic | Ditadura | Ctrl+D | Speech → text **in the composer**. User edits, then Send. |
| Waveform | Entrar no modo de voz | Ctrl+Shift+O | Leave the composer. Live spoken conversation. |

That split is the product. Collapsing them into one mic that auto-sends
is how coding agents eat bad transcripts.

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

### D1 — Two products, two phases

**V1 Dictation. V2 Voice mode.** Never one button that does both.

Why: ChatGPT brief; coding needs a chance to edit; voice mode is an
exclusive view (same rule as Chat/Terminal).

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

Hold-to-talk or toggle. Interim text in the textarea. Stop → user hits
Send (or Enter). Auto-send is an opt-in later, default off.

Why: speech-to-code is lossy (`rm -rf`, paths, package names). Cursor
and ChatGPT dictation both land in an editable field.

### D4 — Voice mode is an exclusive view

Same contract as Terminal: never shown with Chat. Full pane under the
tabs: waveform from real `AnalyserNode` data (no invented viz), live
caption, last agent reply, Stop / Back to chat.

Turns still appear in the session JSONL so Chat replay is truthful.

Why: ChatGPT voice is a place. Our Chat/Terminal exclusivity already
killed phantom "Working"; a third overlay would revive it.

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

### D6 — TTS v1 = `speechSynthesis`, voice-mode only

Dictation does not speak. Voice mode may speak assistant **text**
after `message_end` (not tool dumps, not thinking). Neural TTS is V3.

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
- Auto-send as default.

## 4. Build plan

### V1 — Dictation (composer)

1. `lib/speech.js` — feature detect, start/stop, interim + final
   callbacks, no UI.
2. Mic button on `composer-right` (hidden if `SpeechRecognition`
   missing). Recording state = filled mic + pulse from real volume.
3. Insert into textarea at caret. Do not send.
4. `Ctrl+D`. Palette command "Dictate".
5. Disclose vendor STT in a one-line hint the first time.
6. Visual gate: idle / recording / denied. Overlay audit on the
   composer-right row (`data-align-row`).

Done when: owner can dictate Portuguese into the composer, edit, send
to Grok 4.6, and the turn is normal JSONL.

### V2 — Voice mode (exclusive view)

1. Waveform button → `termWanted`-style `voiceWanted` set.
2. `VoiceView.jsx` fills the pane (Chat hidden, Terminal hidden).
3. Loop: listen (VAD or push-to-talk) → STT → `sendTask` → wait
   `agent_settled` → speak **assistant text only** → listen again.
4. Barge-in V2.1: abort TTS + `Abort` RPC on voice.
5. Back to Chat keeps the transcript.

Done when: a spoken "what files did we change?" round-trips through
pi tools and is audible, then visible in Chat.

### V3 — Cloud STT/TTS (opt-in)

Settings: STT engine = Browser | OpenAI | Groq (keys already in
Providers). TTS = Browser | OpenAI. Still no default. Packages page
can *link* TUI voice packages for the escape hatch, not install them.

### Out of scope until pi itself speaks audio

Speech-to-speech coding agent. MCP audio tools. Mobile-first voice
chrome (mobile parity comes after V2 desktop).

## 5. Why this and not the alternatives

| Alternative | Why not |
|---|---|
| One mic that auto-sends | Coding transcripts need an edit gate |
| OpenAI Realtime as the agent | Forks pi, drops tools/JSONL, billed to us or forces OpenAI |
| Wrap `pi-voice-stt` in the browser | TUI packages capture OS mic; browser needs getUserMedia |
| Ship Whisper in the Go binary | Fat binary, not a pi primitive, contradicts "standard library first" |
| Voice overlay on Chat | Repeats the managed-vs-TUI confusion we just killed |
| Wait for Grok audio | grok-4.6 has no audio I/O; we'd stall the brief |

## 6. Open questions for the owner

1. V1 dictation: toggle (ChatGPT) or hold-to-talk (many TUI packages)?
   Recommendation: **toggle**, hold as later.
2. Auto-send after silence in voice mode (V2) — yes, that's the point
   of the place. Dictation stays manual send.
3. Cloud STT in V1.1 if Chromium quality on PT-BR is bad?
