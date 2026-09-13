### Added

- Providers: **Custom endpoint** in Add provider registers any OpenAI-compatible (or Anthropic-/Google-compatible) gateway for pi from the GUI — name, base URL, API key and model ids; no hand-editing `~/.pi/agent/models.json`. Definitions merge into pi's own models file, the key is stored with every other sign-in, and the roster gains a `custom` badge with Edit endpoint / Sign out / Remove endpoint actions.
