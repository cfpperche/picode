### Added
- **Custom endpoints: the base-URL field now explains itself.** A hint and an
  example follow the API type you pick — an OpenAI-style root usually ends in
  `/v1`, Google's in `/v1beta`, and Anthropic-compatible endpoints differ — so
  the field stops being guesswork.
- **More thinking formats, including the two that need an object.** The format
  picker now offers what pi actually supports: `openai` (the old
  `reasoning_effort` name was written back as `openai`), `deepseek`, `qwen`,
  `qwen-chat-template`, `openrouter`, `together`, `zai`, `ant-ling`,
  `string-thinking`, plus **chat-template** and **baseten**, which reveal a
  JSON editor for `chat_template_kwargs` / `chat_template_args` with pi's
  `$var` references. A typo is explained inline instead of being saved.
- **Verify a custom endpoint for real.** The row's **Verify with the endpoint
  (1 request)** sends one minimal completion (one word in, the smallest output
  ceiling the API accepts) and reports what the endpoint said — the model, how
  long it took and the tokens it counted. The dialog can do the same before
  saving. Built-in providers keep pi's free answer.

### Fixed
- **A wrong key on a custom endpoint no longer reads as verified.** Verify used
  to answer from credential presence, which stayed green for a bogus key; a
  gateway that cannot list, a refused key, an exhausted account and an unknown
  model each now say which one happened.
