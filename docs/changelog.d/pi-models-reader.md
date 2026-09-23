### Changed
- Model pickers, Providers and the agent status bar no longer wait about two seconds on every catalog read: the llama.cpp server's answer is kept and refreshed in the background (an unreachable server no longer stalls anything), and Pi's model list is kept until a sign-in, a custom provider, a package or Pi itself changes. **Try again** on Providers asks Pi afresh.
