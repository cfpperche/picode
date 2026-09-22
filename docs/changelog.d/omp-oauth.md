### Added

- Providers: **Guided sign-in** in the Add provider dialog now covers omp's
  OAuth providers (Anthropic, OpenAI/Codex, GitHub Copilot, Kimi, xAI) —
  one click opens the provider's authorize page in a browser tab, the
  callback returns to PiCode, and the subscription is stored in the vault
  and injected into Omp's terminal through its declared env names
  (ADR-0178). No terminal detour for these providers; the strip remains for
  the rest.
