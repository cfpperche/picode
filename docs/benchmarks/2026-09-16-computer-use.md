# Study: Computer use for agents — benchmarks, vendor contracts, OS primitives, and the policy around them

- **Date:** 2026-09-16
- **Sources:** fetched live on 2026-09-16 by a research sweep (eight
  angles, three of which finished their write-up; the other five were
  harvested from their fetch logs) and read here. Primary pages: the
  official OSWorld-Verified results workbook
  (`osworld-v1.xlang.ai/static/data/osworld_verified_results.xlsx`, parsed
  directly) and the OSWorld 2.0 `official-results.json` (updated
  2026-09-03); xlang.ai's OSWorld-Verified post (2025-07-28); arXiv pages
  for OSWorld-Human, OSWorld-MCP, WindowsAgentArena (+V2), WindowsWorld,
  MyPCBench, OpenComputer, macOSWorld, MacAgentBench, TheAgentCompany,
  ScreenSpot-Pro, OSWorld-G/Jedi, UI-Vision, OS-Harm, RedTeamCUA, VPI-Bench,
  the pop-up attack paper, WASP, EIA, SafeArena, ST-WebAgentBench, CaMeL,
  the reliability paper 2604.17849; vendor docs — Anthropic computer-use and
  browser-use tool pages and release notes, OpenAI computer-use guide and
  integration guide, Azure's mirror of the safety checks, Google's Gemini
  computer-use page (updated 2026-09-04), the Nova Act README, the UI-TARS
  prompt and coordinate READMEs, Copilot Studio computer-use docs, Windows
  365 for Agents MCP docs, the Windows security book chapter on agentic
  security, the Experimental agentic features support page, the WindowsAI
  CSP page, MCP-on-Windows docs, the `winapp ui` docs, Windows Sandbox CLI
  docs, the MXC repo, Foundry Local GA post; GitHub READMEs and `gh api`
  metadata for Agent S, UI-TARS-desktop, Cua, OpenAdapt, Windows-Use,
  Windows-MCP, Terminator, computer-control-mcp, Magentic-UI, UFO2,
  OmniParser; crates.io metadata for xcap, windows-capture, enigo, rdev,
  uiautomation, arboard, windows; Win32 docs for SendInput,
  SetForegroundWindow, high-DPI, desktops, UI Automation security; local
  reads of `@earendil-works/pi-coding-agent` 0.85.0 and
  `packages/pi-browser`. Aggregator leaderboards (steel.dev, benchlm.ai)
  are cited only where the official page failed to render, and are marked.
  **Verification caveat:** the planned second-pass fact check did not run
  (session limit); every number below is as read on the cited page on
  2026-09-16, two independent passes agree on the vendor tool contracts,
  and anything taken from a search snippet is marked *snippet*.
- **Scope:** the owner's question — now that PiCode has a desktop app,
  which tools should agents get to use the operating system (windows,
  apps, files), vision (screenshots to the model) and input
  (mouse/keyboard); what the field measures, what the vendors' contracts
  look like, what a Rust/Tauri shell on Windows can actually do, and what
  the per-agent policy must encode before any of it ships.

## The one-paragraph answer

The field converged in 2025–2026 on one loop — *screenshot in, a small
fixed action vocabulary out, coordinates in the screenshot's own pixel
space, a human confirmation hook on consequential actions* — and on one
finding: agents now beat the OSWorld human baseline (72.36 %) on
short tasks, still fail most hour-long ones (OSWorld 2.0 binary
completion tops out near 30–40 %), take 2.7–4.3× more steps than a person,
and follow instructions found on screen unless a policy layer stops them.
PiCode already owns the three hard parts for the browser — a per-principal
policy (tiers × allowlist, ADR-0128/0134/0143), a shell↔daemon command
channel (ADR-0132) and human co-visibility (ADR-0135). Computer use is
those same three parts pointed at windows instead of tabs, plus two things
the browser never needed: an *actuator in Rust on the Windows side*
(capture, input, UI Automation) and a *decision about where the agent's
desktop is* — the user's own, a secondary desktop, or a sandbox VM.
Microsoft's own agent workspace is not available to third parties yet, so
that decision is ours.

## 1. What the field measures

### 1.1 Desktop benchmarks

| Benchmark | Environment / tasks | Evaluator | Best result (date) | Human | Why it matters to PiCode |
|---|---|---|---|---|---|
| **OSWorld → OSWorld-Verified** (xlang, 2024-04 → 2025-07-28) | Ubuntu VM, 369 tasks (361 without Google Drive), 10 apps incl. LibreOffice, GIMP, VLC, Chrome, VS Code, Thunderbird; AWS parallel (50×) | 134 execution-based evaluator functions; 300+ task/evaluator fixes in the Verified pass | Official verified sheet, 100 steps: Intelligence-Indeed Agent 90.19 % (Jul 2026, agentic framework with coding actions); `claude-fable-5` 85.96 % and `claude-opus-5` 83.39 % (Aug 2026, general models); Holo3-35B-A3B 82.56 % (Apr 2026, open weights); Sonnet 4.6 72.11 % (Mar 2026). Aggregators list Qwen3.8 Max 86.1 % *self-reported* | 72.36 % | The reference vocabulary and harness; "beats human" is now true on 30-step tasks |
| **OSWorld 2.0** (xlang, 2026-06-26; releases v2026.06.24, v2026.08.08, v2.1) | 108 long-horizon workflows, 31 self-hosted sites, 7 domains, ~318 tool calls/task, 69.6 % of tasks > 1 h, 1920×1080, 500-step default | Checkpoints: **binary** (all pass) and **partial** (fraction), 27.25 checkpoints/task | Official JSON (2026-09-03): Claude Opus 5 max effort 31.43 % binary / 68.31 % partial; GPT-5.6 Sol max 27.34 / 62.72. Self-reported: Fable 5.1 41.7 / 77.9 (Aug release), Simular Sai 28.25 / 73.0 at $15.70 per task, GPT-6 Astra 72.6 partial (offline subset) | median 1.6 h | Shows the real gap: long tasks fail on cross-source reasoning (42.6 %), visual-spatial precision (41.7 %), implicit state (39.8 %) |
| **OSWorld-Human** (2506.16042, MLSys 2026) | Human trajectories for OSWorld; 16 agents | Steps ratio + Weighted Efficiency Score | Best agents use **2.7–4.3× more steps** than needed; a double-spacing edit takes an agent 12 min vs < 30 s for a human; best baseline 41.4 % success but 15.6 % WES | — | Efficiency is a first-class metric; step budgets and latency belong on our dashboard |
| **OSWorld-MCP** (2510.24563) | OSWorld + 158 validated MCP tools over 7 apps | Success + tool-invocation rate | Tools lift o3 8.3 → 20.4 % (15 steps); invocation rate only 36.3 % | — | Evidence for "API when available, GUI when not" — the hybrid PiCode already lives (browser verbs + CDP) |
| **Windows Agent Arena** (Microsoft, 2409.08264) → **WAA-V2** (2505.13909) | Windows 11 VM in Docker (needs nested virtualization), 154 tasks → 141 tasks / 11 apps, 30-step default, snapshot restore per task | Deterministic scripts | Navi 19.5 % (2024); PC Agent-E 36.0 % vs Claude 3.7 32.6 % (2025) | 74.5 % | The only Windows suite with programmatic checkers; not runnable inside WSL2 without KVM on the host |
| **WindowsWorld** (2604.27776, 2026-04) | Real Windows VMs, 17 apps (Word, Excel, PowerPoint, Acrobat, Thunderbird, Chrome, Explorer, Calculator, Task Manager, Snipping Tool, GIMP, Paint, Photos, VLC, VS Code, PowerShell, Terminal); 15–40 steps by level | Process-centric checkpoints judged by Qwen3-VL-Plus (Pearson 0.91 with humans) | Gemini-3-flash-preview: 50.32 % intermediate, **20.44 % final** | none | Closest task mix to what PiCode users would ask on Windows; shows Windows lags Ubuntu numbers badly |
| **MyPCBench** (2606.16748) | QEMU Ubuntu 24.04 in Docker, 184 rubric tasks, 17 logged-in web apps + LibreOffice, 100 turns | Rubric + Gemini judge | Opus 4.6 55.4 % perfect; single-app 66 % → 7+ apps 36 % | none | Multi-app drop is the pattern to expect |
| **OpenComputer** (2605.19769) | 33 desktop apps, Docker sandboxes | **Hard-coded verifiers** (94.2 % human agreement) vs LLM judge (79.2 %) | GPT-5.4 68.3 %; open 8B models fall from ~50 % on OSWorld to 6–11 % here | — | Verifier design lesson: code beats judges by 15 points of agreement |
| **macOSWorld** (2506.04135) / **MacAgentBench** (2606.22557) | AWS Mac / Docker-QEMU macOS Tahoe; 202 / 676 tasks, 15 / 50 steps | Rule-based, 156 getters (MacAgentBench) | Claude CUA 37.1 % (2025); OpenClaw + Opus 4.6 73.7 % pass@1, 58.6 % pass^4 (2026) — "primarily driven by its skill library" | — | Skills/tools beat raw GUI skill; deception buttons fooled Claude/OpenAI CUAs 72 / 69 % of the time |
| **TheAgentCompany** (2412.14161) | Simulated company: GitLab, OwnCloud, Plane, RocketChat; 175 tasks | Checkpoints, 29 % LLM-judged | Gemini 2.5 Pro + OpenHands 30.3 % full, 39.3 % partial, $4.2/task | — | Work-shaped tasks and a cost column |

### 1.2 Web benchmarks (what the existing browser tool would be measured on)

| Benchmark | Shape | Evaluator | Best result (date) | Human |
|---|---|---|---|---|
| **WebArena** (812 tasks, self-hosted Shopping/Admin/Reddit/GitLab/Wiki/Map) → **WebArena-Verified** (ServiceNow, NeurIPS 2025 SEA: 812 reviewed + 258 hard; LLM judging replaced by type-aware comparison; **offline replay from network traces**; `uvx webarena-verified`) | Sandboxed, reproducible | Deterministic | WebTactix 74.3 % (Feb 2026, *aggregator*); Operator 58.1 % (2025) | 78.24 % |
| **Online-Mind2Web** (OSU, 300 tasks / 136 live sites, CC BY 4.0) | Live web | WebJudge (o4-mini 85.7 % human agreement; WebJudge-7B 87.0 %) or human | GPT-5.4 native computer use 93 % (Mar 2026, vendor); UI-TARS-2 88.2 %; Operator 61.3 % — judges differ per row (*aggregator*) | — |
| **Mind2Web 2** (130 long-horizon research tasks) | Live web, agentic search | Agent-as-a-Judge over rubric trees; partial completion + pass@3 | Deep-research systems at 50–70 % of human | yes |
| **BrowseComp / -Plus** (1,266 hard lookups) | Info seeking | Exact answer | GPT-5.6 Sol Ultra 92.2 %, Opus 5 90.8 % (Jul 2026, *aggregator*) | trainers |
| **WebVoyager** (586 tasks) | Live web | LLM judge + manual review | browser-use 89.1 % on 531 kept tasks (2024-12; 55 tasks dropped; "the eval model is not good", corrected by hand) | — |
| **BearCubs** (111 questions needing real interaction) | Live web, multimodal | Human-verified | ChatGPT agent 65.8 %; Operator 23.4 %; Anthropic CU 14.4 % (2025-07) | 84.7 % |
| **WebBench** (Halluminate, 2,454 open tasks / 452 sites) | Live web; READ 64 % vs CREATE/UPDATE/DELETE | Human review of recordings | READ > 75 %; **write-heavy 46.6 %** (2025-05) | — |
| **ClawBench** (153 write-heavy live tasks, 144 platforms; submissions intercepted) | Live web, side-effect safe | Human truth + agentic judge | Sonnet 4.6 33.3 % (2026-04) | — |
| **CAP** (420 tasks / 108 sites, COLM 2026), **WebForge** (934 generated sites), **StressWeb** (perturbations) | 2026 wave | mixed | "perception-heavy interactions are the bottleneck"; robustness gaps hidden by clean benchmarks | — |

Read across: write actions on live sites are where every agent still fails
(46.6 % → 33.3 %), which is exactly the class PiCode gates behind `act`.

### 1.3 Grounding (can the model point at the right pixel)

| Benchmark | Size | Best closed (date) | Best open (licence) |
|---|---|---|---|
| **ScreenSpot-Pro** (1,581 instructions, 23 pro apps, 3 OSes, high-res) | micro-average accuracy | GPT-6 Astra 92.7 %, Claude Opus 4.8 87.9 %, GPT-5.4 85.4 % (Sep 2026, *aggregator* of the official board) | Holo2-235B-A22B 78.5 % agentic (licence not read); Holo2-30B-A3B 66.1 % (non-commercial); Holo2-8B 58.9 % (Apache-2.0); OpenCUA-72B 60.8 % (CC BY-SA); UI-TARS-1.5-7B 61.6 %; OmniParser V2 39.5 % |
| **OSWorld-G** (564 samples: text, element, layout, fine manipulation, refusals) | accuracy | Holo2-235B 79.0 % (*snippet*) | Jedi-7B 54.1 % (Apache-2.0, 4 M sample dataset); UI-TARS-72B 57.1 % |
| **UI-Vision** (83 desktop apps, CC BY 4.0), **MMBench-GUI** (L1–L4, 8 k tasks, 6 platforms) | — | "critical limitations … drag-and-drop, professional software" | — |
| **AndroidWorld** (116 tasks / 20 apps), **AndroidLab** (138 tasks) | mobile | UI-TARS-2 73.3 % | — |

Two facts that shape our design: (1) grounding is now a frontier-model
capability (Opus 4.8 87.9 % without a helper model), so PiCode does not
need to ship a grounder; (2) every study still finds the **accessibility
tree + screenshot** combination beats pixels alone (OSWorld ablation; WAA's
Navi = UIA + OCR + Set-of-Marks; UFO2 = UIA + vision), and UIA-first
tools without vision at all (Windows-Use, Windows-MCP with 2 M+ Claude
Desktop users, Terminator) work on the large class of apps that expose an
accessibility provider. Chromium enables native UIA on Windows since
Chrome 138 (2025-08-14), so WebView2 tabs and Electron apps are visible to
the same tree.

### 1.4 Safety and reliability benchmarks

| Benchmark | Measures | Headline |
|---|---|---|
| **OS-Harm** (150 OSWorld tasks) | unsafe execution: misuse / injection / misbehavior | GPT-4.1 21 %, Claude 3.7 29 % average unsafe; desktop notifications and mails inject "in half the cases" |
| **RedTeamCUA** (864 hybrid web+OS cases) | indirect prompt injection ASR | Claude 3.7 CUA 42.9 %, Operator 7.6 %; attempt rate 92.5 % |
| **VPI-Bench** (306 visual injections, ICLR 2026) | injection rendered in UI | browser agents up to 100 %, computer-use agents up to 51 %; system-prompt defences "limited" |
| **Pop-up attacks** (2411.02391) | adversarial pop-ups on OSWorld/VWA | 86 % ASR, −47 % task success; "ignore pop-ups" instructions ineffective |
| **WASP**, **EIA**, **SafeArena**, **ST-WebAgentBench** | web-agent injection, PII leak, misuse, policy compliance | partial attacker success up to 86 %; 70 % PII leak; 34.7 % of harmful tasks completed; completion-under-policy < 2/3 of nominal |
| **Vendor cards** | with vs without safeguards | Operator 62 → 23 % (2025); Claude Opus 5 browser use 3.7 → 0 % with "auto mode" probes + classifier (2026-07); GUI computer use 0.54 → 0.25 % |
| **Reliability** (2604.17849) | repeated runs | pass@10 ≈ 78 % but **pass^10 ≈ 36 %** of tasks; clarifying the instruction first lifts pass^3 (GPT-5 0.454 → 0.576) |

Read across: screen content is an attack surface at double-digit rates
without a policy layer, and single-run success overstates reliability by
about 2×.

## 2. The action contract the vendors converged on

| Vendor contract (state 2026-09-16) | Vocabulary | Screenshot transport | Coordinates | Safety hook |
|---|---|---|---|---|
| **Anthropic `computer_toolset_20260801`** (GA 2026-08-19, no beta header; earlier `computer_20251124`/`computer_20250124` stay in beta) | 17 members: `screenshot`, `zoom(region)`, `left/right/middle/double/triple_click`, `left_click_drag`, `mouse_move`, `left_mouse_down/up`, `cursor_position`, `scroll(direction, amount)`, `type`, `key(repeat)`, `hold_key`, `wait`; several per turn = batch, stop at first failure | `tool_result` with an `image` block (base64 PNG); ~4,500 tokens of definitions per request | **Screenshot pixel space**; the API rejects oversize images (≤ 2,576 px long edge / 4,784 visual tokens on Opus 4.7+; 1,568 px earlier); recommended 1280×720 or 1024×768, pre-downscale and rescale coordinates yourself | Server-side prompt-injection classifiers on screenshots (opt-out); human confirmation "is your job"; sibling `browser_toolset_20260801` (31 members, `ref_N` targets from `read_page`) |
| **OpenAI `tools:[{type:"computer"}]`** (Responses API, GA with GPT-5.4 2026-03-05) | `click(button,x,y)`, `double_click`, `drag(path)`, `move`, `scroll(x,y,dx,dy)`, `keypress(keys[])`, `type`, `wait(ms)`, `screenshot`, batched `actions[]` | `computer_call_output` → `computer_screenshot` data URL, `detail:"original"` | Screenshot pixels; 1440×900 or 1600×900 advised; 30,000-patch cap | `pending_safety_checks` (`malicious_instructions`, `irrelevant_domain`, `sensitive_domain`) echoed back as `acknowledged_safety_checks`; watch mode; GPT-6 Astra prefers driving UIs through code execution |
| **Google `computer_use`** (Gemini 3.x, "Preview" label; built in since 3.5 Flash 2026-06-24) | browser/mobile/desktop sets: `click…right_click`, `mouse_down/up`, `move`, `type(press_enter)`, `drag_and_drop`, `wait`, `press_key`, `hotkey`, `take_screenshot`, `scroll(magnitude_in_pixels)`, `navigate/go_back/forward`; mobile `open_app`, `long_press`; every action carries an `intent` | `function_result` parts, image/png | **Normalized 0–999** (1000×1000); client denormalizes | `safety_decision` = allowed / `require_confirmation` / blocked; 7 policy categories (`FINANCIAL_TRANSACTIONS`, `COMMUNICATION_TOOL`, `DATA_MODIFICATION`, `USER_CONSENT_MANAGEMENT`, …); opt-in `enable_prompt_injection_detection` |
| **ByteDance UI-TARS** (prompt contract; UI-TARS-1.5-7B open, UI-TARS-desktop Apache-2.0) | `click(point)`, `left_double`, `right_single`, `drag`, `hotkey`, `type`, `scroll`, `wait`, `finished`; mobile adds `long_press`, `open_app`, `press_home/back` | chat image | absolute pixels in the `smart_resize`d image (factor 28) | none in the model |
| **Microsoft Windows 365 for Agents MCP** (GA 2026-06; cloud PC) | 65 tools: `take_screenshot`, `zoom_region`, `analyze_screen` (OCR boxes), `click`, `drag_mouse`, `type_text`, `press_keys`, `list_windows`, `activate_window`, **`get_accessibility_tree`**, **`find_ui_element`**, `clipboard_read/write`, allow-listed `execute_shell_command`, `launch_application`, Edge `browser_*` with `browser_snapshot` refs | MCP | screen pixels | allow-listed commands; corner-cursor failsafe; WebRTC screenshare with take/release control |
| **Copilot Studio computer use** (GA 2026-05) | natural-language steps on a virtual mouse/keyboard | activity map | hosted machine | URL/app allow-list, Enforce HTTPS, human-supervision e-mail; 5 credits/step |
| **Devin** (1024×768 desktop) / **Manus "My Computer"** (2026-03) | click/type/scroll/drag/screenshot/shortcuts / terminal + files with Allow once / Always | — | — | org admin toggle / per-command approval |
| **Amazon Nova Act** (GA 2025-12, browser-only, $4.75 per agent hour) | `act("natural language")` | internal | internal | `human_input_callbacks`, `max_steps` |

Read across: the vocabularies are the same fifteen verbs; the two
differences that matter are the coordinate frame (pixels of the returned
screenshot, or 0–999 normalized) and where the safety decision lives (in the
API response for OpenAI/Google, in the client for Anthropic). Microsoft's
cloud contract is the only one that ships **an accessibility tree and OCR
next to the pixels** — the hybrid the benchmarks say wins.

**Local fact (measured):** pi can already carry an image to the model.
`@earendil-works/pi-coding-agent` 0.85.0 (the copy under `packages/pi-compact`; the runtime the daemon launches was not pinned down) types `AgentToolResult.content`
as `(TextContent | ImageContent)[]`, the `read` tool emits image blocks
for PNG/JPEG, `agent-session` resizes tool-result images after the
`tool_result` hook, and pi-ai maps them to Anthropic `image/base64`,
OpenAI `input_image` and Gemini `inlineData`. Our `browser` tool's
`screenshot` verb does **not** use this: it writes the PNG to `tmpdir()`
and returns a text line with the path, so the model only sees pixels if it
then calls `read`. That is a one-line change and the first win.

## 3. What a Rust shell on Windows can actually do

WSL2 cannot capture or drive the Windows desktop — every framework that
splits brain (agent) from actuator does it over a socket, and in PiCode
that socket exists: ADR-0132's `browser.command` SSE frame down, result
POST up. The shell is the actuator; the question is only which primitives.

| Need | Windows primitive | Rust crate (crates.io, 2026-09-16) | Notes |
|---|---|---|---|
| Screenshot of monitor / window | Windows.Graphics.Capture (WGC), DXGI Desktop Duplication, GDI `BitBlt`/`PrintWindow` | **xcap 0.9.8** (Apache-2.0, 1.87 M downloads, `Monitor::all()`, `Window::all()`, `capture_region()`; uses WGC + GDI + DXGI on Windows, min Windows 8.1); **windows-capture 2.0.1** (MIT, WGC/D3D11, cursor/border/dirty-region settings) | The shell already captures a WebView2 through COM (`btab_preview`); OS capture is the same idea one level up. `WDA_EXCLUDEFROMCAPTURE` windows come out black by design |
| Mouse / keyboard | `SendInput` | **enigo 0.6.1** (MIT, 2.85 M downloads; "on Windows, mouse coordinates and display size use physical pixels; Enigo temporarily switches the calling thread to per-monitor DPI awareness") | UIPI: input into a higher-integrity window is silently dropped; `SetForegroundWindow` only works from the foreground process or with the user's last input — the `winapp` CLI reports `no_interactive_desktop` / `foreground_not_target` / `target_moved` for exactly these cases |
| Accessibility tree of native apps | IUIAutomation (COM) | **uiautomation 0.25.1** (Apache-2.0, released 2026-09-04; `UITreeWalker`, matchers, patterns behind the `pattern` feature, `send_keys`, `screenshot` feature) | UIA-pattern actions (`Invoke`, `SetValue`, `Scroll`) work without the foreground and on a locked session; Microsoft's own agent-facing CLI **`winapp ui`** (public preview 2026-08-19, Tauri listed as a supported framework) exposes `inspect/search/invoke/get-value/set-value/wait-for/scroll/screenshot/record --frames/list-windows/click/drag/send-keys` with JSON output and AutomationId or slug selectors — a ready shape to copy or shell out to |
| Windows and apps | `EnumWindows`, `ShellExecute`, `CreateDesktop` | `windows` 0.62.2 (the shell pins 0.61 via webview2-com) | UFO2's "Picture-in-Picture" virtual desktop runs the agent on a secondary desktop so the user keeps theirs |
| Clipboard | Win32 clipboard | **arboard 3.6.1** (MIT/Apache-2.0, 46 M downloads) | Clipboard write is a `full`-tier act (data leaves the window) |
| OCR | `Windows.Media.Ocr.OcrEngine` (any PC, since Windows 10) / `Microsoft.Windows.AI.Imaging.TextRecognizer` (NPU-only, Copilot+ PCs) | via `windows` WinRT bindings | Gives word boxes for Set-of-Marks without a model |
| Local vision model | **Foundry Local** (GA 2026-04-09, OpenAI-compatible HTTP, Windows/Linux/macOS) | — | A free VLM endpoint the WSL daemon could call; not needed for v1 |
| Isolation | see §4 | — | — |

DPI is the trap: screenshots are physical pixels, per-monitor scaling
differs, and the model's coordinates are in the screenshot it was given.
The shell must own the mapping (capture size → screen coordinates) exactly
as it owns the WebView2 bounds today, and the same lesson from
`docs/handoff/open/work-browser-tabs.md` applies: a windowed WebView2
cannot be painted over, so a "live view" of the agent's window is a still
plus a native hide, not an overlay.

### Who ships this open-source (for patterns, not code — licences stay put)

| Project | Licence · activity | Perception | Actuation | Isolation | Take-away |
|---|---|---|---|---|---|
| **Agent S3 / Sai** (simular) | Apache-2.0 · 12.3 k★ · pushed 2026-09-05 | screenshots, UI-TARS-1.5-7B grounder | `pyautogui` code, `exec()` "with the same permissions as the user" | none | Best-of-N over behaviour narratives → 72.6 % OSWorld-Verified; Sai 73 % partial on 2.0 at $15.70/task |
| **UI-TARS-desktop / Agent TARS** (ByteDance) | Apache-2.0 · 39 k★ · pushed 2026-09-11 | screenshots | local operator (Windows/macOS/browser) | none | Remote operators discontinued 2025-08-20; model contract in §2 |
| **Cua** (trycua) | MIT · 22.7 k★ · pushed 2026-09-16 | **cua-driver**: "observe and control an OS without stealing focus"; `get_window_state` returns an accessibility tree, actions ground on `element_index`; 56 MCP tools *(deepwiki summary, not the README)* | native drivers (Rust port `cua-driver-rs`, *same source*) | Windows Sandbox, Docker, Lume VMs, cloud fleets; **cua-bench** creates/verifies tasks without a VM | The closest architecture to what PiCode needs: background operation + a11y tree + sandboxes |
| **Windows-MCP** (CursorTouch) | MIT · 7 k★ · pushed 2026-09-16 · "2 M+ users in Claude Desktop Extensions" | UIA "without computer vision", optional `use_vision` | 20 tools: Click, Type, Scroll, Move, Shortcut, Wait, WaitFor, Screenshot, Snapshot, App, PowerShell, FileSystem, Scrape, Clipboard, Process, Notification, Registry | none | Proof that UIA-first works for mainstream Windows apps; also proof of what a `full` tier looks like (PowerShell, Registry) |
| **Windows-Use** (same author) | MIT · v0.8.1 2026-04 | UIA + optional screenshots + annotations | UIAutomation + PyAutoGUI | "NO sandbox or isolation layer" — run it in a VM | Tool split `click/type/scroll/move/shortcut/app/shell/scrape/desktop/wait/done` |
| **Terminator** (mediar-ai) | MIT · Rust | Windows accessibility tree with selectors `name:`/`role:`/`window:` | `SendInput` in `platforms/windows/input.rs` | none | "Playwright for Windows"; `terminator-mcp-agent`; Windows-only |
| **UFO2** (Microsoft Research) | MIT | hybrid UIA + vision, GUI–API action layer | HostAgent + AppAgents | **Picture-in-Picture virtual desktop** | The non-disruptive-execution pattern |
| **Magentic-UI → MagenticLite** (Microsoft) | MIT | browser via Playwright | VM ("Quicksand") | check-ins before critical actions, take-over | The HITL vocabulary |
| **OmniParser V2** (Microsoft) | detector v3 MIT | screenshot → boxes + labels (ScreenSpot-Pro 39.5) | OmniTool Windows 11 VM | VM | Superseded by frontier grounding; useful only as a SoM fallback |
| **OpenAdapt** (MIT, v1.16 2026-08), **Open Interpreter** (Apache-2.0, 68 k★), **self-operating-computer** (MIT, last push 2025-09), **Bytebot** (archived 2025-09), **OS-Copilot** (last push 2024) | — | — | — | — | Record-and-replay (OpenAdapt "VERIFIED only if an independent check agrees") is a pattern for our eval suite; the rest is stale |
| **Anthropic quickstart** | Docker: Ubuntu + Xvfb + Mutter + xdotool + noVNC, 1024×768 | screenshots | xdotool | container | The reference implementation of the §2 contract |

## 4. Where the agent's desktop lives (the decision Microsoft has not made for us)

| Option | Isolation boundary | Agent sees / drives | Human sees | Availability (2026-09-16) |
|---|---|---|---|---|
| **The user's own desktop, one bound window** | none (the browser model: co-visibility + tiers + Ask) | the window(s) it is bound to | everything, live | today — ADR-0135's split-pane grammar, one level up |
| **Secondary desktop (`CreateDesktop`, UFO2 PiP)** | same session, separate desktop object; input does not collide with the user's | a whole desktop of its own | a thumbnail/still we render | today, needs a spike: apps opened there are invisible to the user's UIA clients and some apps misbehave |
| **Windows Sandbox** | Hyper-V container VM, disposable | a full Windows 11 | the `wsb connect` window | GA app on 24H2; **`wsb` CLI preview**: `start --config`, `exec` (no process I/O), `share --allow-write`, `connect`, `stop`, `--raw` JSON |
| **Windows Agent Workspace + agent account** (Copilot Actions) | separate standard account and session, six known folders, Agent ID for audit | its own desktop | "authorize, monitor, take over" | **private preview, no public API**; Insider builds 26100.7344+; the support page's change log ends 2025-12-05; CSP knobs Enterprise/Education only |
| **MXC session isolation** (Build 2026) | agent account + session via `Windows.AI.IsolationSession.Preview` | **no display server** — "GUI applications unsupported" | stdout/stderr | Insider preview; WinMD not redistributable; process-isolation backend (`processcontainer`, 24H2+) is public preview and has a `ui.injection` policy flag |
| **Windows 365 for Agents** | Cloud PC per session | 65 MCP tools | WebRTC screenshare | GA 2026-06, enterprise tenants; the reference design, not a free local option |
| **WSLg Linux GUI apps** | WSL VM | screenshots work | RAIL windows on the host | UIA cannot see inside RAIL windows (wslg #580, *snippet*); Recall skips them |

Read across: nobody outside Microsoft can spawn an *interactive* agent
desktop through a Windows API today. Every shipping third-party product
either runs on the user's desktop with a policy (Windows-MCP, Terminator,
Manus) or in a VM (Cua, Magentic, Copilot Studio). PiCode's honest
sequence is the first option now, the sandbox as the `full`-tier arena when
a spike proves `wsb` is usable, and MXC watched.

## 5. What the policy must encode (the vendors agree on five controls)

1. **Isolate and minimise** — "dedicated virtual machine or container with
   minimal privileges" (Anthropic), "isolated browser or VM and an allow
   list of sites and actions" (OpenAI), agent account with no admin rights
   and six known folders (Microsoft).
2. **Screen content is untrusted** — "Text in a page, document, or tool
   result cannot grant permission or override the user's instructions"
   (OpenAI); Anthropic concedes "Claude will follow commands found in
   content" and runs classifiers on screenshots; measured ASR without a
   policy layer is 7–43 % (RedTeamCUA), 86 % for pop-ups.
3. **Confirm consequential actions at action time** — the same list
   everywhere: deleting data, changing permissions or sharing, sending or
   posting as the user, payments, credentials, CAPTCHAs, ToS/cookie
   consent; "typing sensitive information counts as transmission"; "do not
   ask early" (OpenAI). Gemini makes it a per-category policy with
   overrides; Windows makes it per-agent per-folder Allow always / Ask /
   Never.
4. **Co-visibility and take-over** — watch mode that pauses when the user
   looks away (Operator), take-over that stops screenshots while the human
   types credentials, "authorize, monitor, and take over" (Windows).
5. **Bound the run and audit it** — step/time/cost limits with
   cancellation; "log prompts, screenshots, model-suggested actions, safety
   responses, and all actions ultimately executed" (Google); a
   "tamper-evident audit log" and an Agent ID distinct from the user
   (Microsoft principles).

PiCode already implements 2 (read has no execute path), 3 (the Ask bar with
`allow/deny/ask` per site and kind, 60 s watchdog, "Always allow" writes a
standing), 4 (ADR-0135 split, closing is revoking) and half of 5 (the
events feed) for the browser. What is new for the OS is the *target*
dimension — a window/app allowlist instead of a domain allowlist — and the
*action classes* that need Ask regardless of tier.

## 6. How to measure our own tool

- **Task format to copy:** OSWorld's task JSON — `id`, `instruction`,
  `config` (setup steps: `download`, `launch`, `execute`, `open`,
  `command`), `related_apps`, `evaluator` with a getter (`vm_file`,
  `vm_command_line`, clipboard, window title…) and a metric
  (`compare_table`, `check_include_exclude`, …). Harbor's layout is the
  simpler container-native sibling: `instruction.md`, `task.toml`,
  `environment/`, `solution/solve.sh`, `tests/test.sh` writing
  `/logs/verifier/reward.txt`; trajectories in ATIF (`trajectory.json`
  with `steps[]`, `tool_calls`, `observation`, `metrics`).
- **Checkers beat judges:** OpenComputer measured 94.2 % human agreement
  for hard-coded verifiers against 79.2 % for an LLM judge; WindowsWorld
  had to fall back to a VLM judge and reports it. A 12–20 task Windows
  suite for PiCode is checkable in code: file exists with content, window
  title present (UIA), clipboard content, a Settings toggle read back
  (UIA), a registry value, a document saved by Notepad/WordPad/LibreOffice.
- **Protocol:** 100-step budget and 1920×1080 are the OSWorld-Verified
  conventions; report **pass^3** next to pass@1 (2604.17849), steps and
  wall-clock per task (OSWorld-Human), and cost per task (OSWorld 2.0's
  `estimatedCostUsd`, Sai's $15.70). Public harnesses do **not** run on a
  WSL2 developer box: WAA needs nested virtualization, OSWorld needs a VM
  or AWS; **cua-bench** "create and verify a simulated task without a VM,
  Docker, or model API key" is the one that does.
- **Where to log:** per-step screenshot hash + action + policy decision on
  the events feed (ADR-0048) — the same refusal as the browser study:
  frames are ephemeral bulk, the ledger keeps the audit.

## 7. What PiCode adapts (proposal shape, pre-ADR)

| Benchmark pattern | PiCode adaptation |
|---|---|
| One fixed vocabulary, coordinates in screenshot space (Anthropic/OpenAI) | A **`computer` tool as a pi package beside `browser`**, verbs mirroring the 17 Anthropic members plus `windows` (list), `focus`, `snapshot` (UIA tree of the bound window, the way `browser snapshot` returns the AX tree), `clipboard`, `open` (the existing `btab_open_path`/`open_external`). Results carry **image blocks**, not paths — and the `browser` `screenshot` verb gets the same fix |
| Brain in one process, actuator in another (Cua, W365) | Reuse ADR-0132: `computer.command` frames down the same SSE line, result POST up; the shell executes with `xcap`/`windows-capture`, `enigo` (`SendInput`), `uiautomation`, `arboard`. No new port, no new credential, no Python sidecar |
| Accessibility tree + screenshot beats pixels (OSWorld, WAA, UFO2, Windows-MCP) | `snapshot` is UIA-first; `screenshot` (+ optional `Windows.Media.Ocr` boxes) for what UIA cannot see; `zoom` for dense UIs; the model does the grounding |
| Tiers × allowlist (ADR-0128) | `{tier, domains}` → `{tier, domains, apps}`: **read** = screenshot/snapshot of the **bound window** only; **act** = input into bound windows, `focus` among them; **full** = any window, clipboard write, launch, file dialogs, PowerShell-class verbs. Unmanaged callers stay read-only by construction (ADR-0143 unchanged) |
| Confirmation classes (OpenAI/Google/Windows) | Ask regardless of tier for: sending/posting as the user, payments, deleting files, credentials, installs/elevation — through the existing Ask bar (`permissions.rs` decision table, 60 s watchdog, "Always allow" as a standing) |
| Co-visibility, watch mode, take-over (ADR-0135, Operator, Windows) | "Bind window" from the agent's pane: the bound window is shown as a live still (the `btab_preview` trick, since a native window cannot be painted over); input pauses when the human moves the mouse; closing the binding revokes |
| Audit + budgets (Google, Microsoft, OSWorld-Human) | Every step to the events feed with screenshot hash; per-run step/time budget with cancel; steps and wall-clock on the dashboard next to tokens |
| Isolation ladder (§4) | v1 the user's desktop with binding; v2 spike Windows Sandbox via `wsb` as the `full` arena and/or a secondary desktop (UFO2); watch MXC and the agent workspace for an API |
| Verified tasks with code checkers (OSWorld, Harbor, OpenComputer) | A 12–20 task in-house Windows suite in OSWorld's JSON shape, pass^3 + steps + cost, run against a scratch shell |

**Boundary:** this crosses security model (agents act on the OS), protocol
(a second command family on the ADR-0132 line) and process (the shell gains
an actuator role) — an ADR before code, in the ADR-0128 lineage.

## Refusals

| Temptation | Why not |
|---|---|
| A Python/pyautogui sidecar on Windows (what Agent S, Windows-Use, the quickstart do) | The shell is the only Windows resident (ADR-0142) and it is Rust; a second runtime is a second thing to install, update and sign |
| Raw `SendInput`/screenshot from WSL | Impossible by construction; the actuator is the shell |
| Ship a grounding model (OmniParser, Holo2, Jedi) | Frontier models ground at 85–93 % on ScreenSpot-Pro; the field's helpers are 40–78 % and most are non-commercial |
| Frames on the events feed | Durable audit vs ephemeral bulk (2026-09-02 and 2026-09-10 refusals stand); hash + path |
| A "computer" default of `read` on *the whole desktop* | The browser default reads one tab the human opened (ADR-0134); a desktop is every window including the password manager — read must be scoped to a bound window |
| Wait for Windows Agent Workspace | Private preview, no API, no 2026 change-log entry; design for our own isolation and adopt theirs if it opens |
| Copy code from Windows-MCP/Terminator/Cua | Patterns only — licences and languages differ; the shapes (tool split, selectors, background driving) are what we take |

## Decisions for the owner

1. **Arena for v1:** the user's own desktop with a bound window (fast,
   matches the browser grammar) — or a Windows Sandbox spike first?
2. **Default policy:** `read` on the bound window only (mirrors ADR-0134),
   or nothing until granted, because the OS is wider than a tab?
3. **Vocabulary:** mirror Anthropic's 17 members one-to-one (a Claude-backed
   pi agent maps directly and the docs are theirs) or a smaller house set
   with `snapshot`/`windows`/`focus` added?
4. **Ask classes:** the list in §7, and whether "Always allow" may be
   granted per app (Windows does per folder).
5. **Image blocks now:** change the `browser` `screenshot` verb to return
   an image block — independent of everything else and the cheapest win.
6. **Scope:** Windows-only in v1 (the shell is the actuator); native-Linux
   PiCode gets no `computer` tool until a Linux shell exists.
