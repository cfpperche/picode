### Added
- **Custom endpoints: load the model list instead of copying it.** The Add/Edit
  endpoint dialog has a **Load models** button under the ids: it asks the
  endpoint what it serves and adds the ids you are missing (your typed ones
  stay, nothing is reordered). It also fills context window and max output when
  every listed model reports the same number, and says so when they differ. It
  works for OpenAI-compatible gateways, Anthropic and Google endpoints.
- **The failure says what to fix.** A refused key, a URL that needs `/v1`, a
  host that does not resolve, or an account that cannot list: each is one line
  naming the next step, in both apps and in the same words.

### Fixed
- **A gateway's error message can no longer leak your key.** If an endpoint
  echoes the API key back in an error, PiCode redacts it before the message
  reaches the browser.
