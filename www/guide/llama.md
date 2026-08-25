# llama.cpp

PiCode talks to a [llama.cpp](https://github.com/ggml-org/llama.cpp) router. `/llama` can **Install llama-server** (CPU build from GitHub releases into `~/.picode/llama/`) and **Start router**. It does not delete GGUF files.

Canonical: [pi llama.cpp](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/llama-cpp.md).

## 1. Install `llama-server`

In PiCode: `/llama` → **Install llama-server** (CPU build for this OS, `~/.picode/llama/`).

Or install yourself from [llama.cpp releases](https://github.com/ggml-org/llama.cpp/releases) and put it on `PATH`.

WSL: PiCode is Linux — the installer gets `ubuntu-x64`. A Windows `.exe` is the wrong machine.

## 2. Start the router

No `--model` / `-m` (that is single-model mode).

```bash
mkdir -p ~/.picode/llama-models
llama-server \
  --models-dir ~/.picode/llama-models \
  --no-models-autoload \
  --jinja \
  --host 127.0.0.1 \
  --port 8080
```

Optional API key: pass the same value to `llama-server --api-key` and to PiCode.

Gated Hugging Face repos: export `HF_TOKEN` **in the llama-server process**.

## 3. In PiCode

`/llama` (or Providers → llama.cpp):

1. URL `http://127.0.0.1:8080` → Save → Retry.
2. **Download** a GGUF, or drop files in `--models-dir` and restart the router.
3. **Load**, then pick the model on the agent chip.

| | pi TUI | PiCode |
|---|---|---|
| Login | URL + optional key | same |
| Manage | `/llama` | dialog + Providers panel |
