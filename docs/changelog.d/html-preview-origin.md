### Added

- HTML previews run on their own address (`<ticket>.localhost`), so a page can keep settings, register a worker and use its own storage — and neither **Save** nor **Reload** resets it.
- When the browser cannot open that address, or when PiCode is reached from another machine, the preview falls back to the sandboxed frame with one line above it saying so.
