# llama.cpp

PiCode talks to a **running** [llama.cpp](https://github.com/ggml-org/llama.cpp) router. It does not install the binary or delete GGUF files.

Canonical: [pi llama.cpp](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/llama-cpp.md).

## 1. Install `llama-server`

Download a build for your OS from [llama.cpp releases](https://github.com/ggml-org/llama.cpp/releases). Put `llama-server` (Windows: `llama-server.exe`) on `PATH`.

WSL: use a **Linux** build and run the server **inside WSL**. A Windows `.exe` is not `127.0.0.1` from PiCode in WSL.

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
