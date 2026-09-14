### Added
- **Custom endpoints: pick how thinking travels on the wire.** The Advanced
  section of the Add/Edit endpoint dialog now names the thinking format —
  `reasoning_effort` (the OpenAI standard), DeepSeek, Qwen, OpenRouter,
  Together or Z.AI — instead of assuming every gateway speaks
  `reasoning_effort`. The choice is written as `compat.thinkingFormat` in
  `~/.pi/agent/models.json`; leaving it at the default writes nothing, so pi
  keeps choosing for that API type.

### Fixed
- **A hand-edited `compat` key no longer hides the provider from the GUI.** A
  string value in `compat` (such as `"thinkingFormat": "deepseek"`) made the
  whole entry fail to decode, so the provider silently vanished from the
  roster, Edit and the catalog while pi kept using it. The file is now read
  key by key and unknown keys are left untouched.