### Fixed
- Cleaning caches from the desktop Management window works when the tool is installed only for your account. Before, the Go build cache, the uv cache and the pnpm store failed with "go is not installed" (or uv, or pnpm), even though your terminal runs these tools fine.
