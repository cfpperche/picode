# llama.cpp

PiCode connects to an existing [llama.cpp](https://github.com/ggml-org/llama.cpp)
router, or creates a separate local CPU service on Linux/WSL.

## Let PiCode manage a local service

Open **llama.cpp → Local service** (`#/llama/service`) on desktop or mobile.
Choose **Create local service**, then **Install** and review the pinned CPU
version. Installation verifies the official archive and installed files before
executing the server. Choose **Start**, then **Use this connection** and
**Save connection** to use it for model operations.

The Light preset uses two CPU threads and 4096 context; Balanced uses four
threads and 8192 context. Both disable GPU use and automatic model loading.
Advanced settings expose port, context, thread count and chat templates.
Saving settings never restarts a running server. A restart notice appears
when saved settings differ from those in use. **Effective command** shows the
saved arguments for the verified executable.

Start, stop, restart, update and rollback have a review step and durable
activity. Updates retain the previous installation; a candidate that fails to
start restores the previous version when possible. Reviews expire after five
minutes and reject changed state. Stopping a running server requires explicit
confirmation of interruption, including other applications using it.

This service stops when PiCode stops, including after a crash. It does not
start automatically after a PiCode restart. Interrupted actions remain in
history for review; they are never replayed automatically.

**Cache** lists exact file sizes and eligibility. Stop the service first.
Only verified installer archives and model files from tracked successful
downloads can be selected. Models referenced by configured agents, changed
files and files without ownership evidence are retained. A final check runs
before deletion. **Export diagnostics** includes settings and action states,
excluding keys, host names, paths, model names and raw logs.

The initial catalog contains CPU releases b10809 and b10826 for Linux x64
and ARM64. Real acceptance currently covers Linux x64; ARM64 requires host
acceptance. Existing external servers retain their connection/model controls;
PiCode never adopts their processes or cache. GPU builds and secret-bearing
environment configuration are not part of this local CPU service.

## Connect an existing router

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

Open **Providers → llama.cpp → Manage**, or use `/llama` on desktop.
For a new connection, choose **Set up llama.cpp** in Providers.
The manager has **Models**, **Server**, **Activity** and **Local service** pages on desktop and mobile:

1. **Server** → enter the URL and optional API key → **Save connection**.
   Leaving the key blank keeps the saved key. **Test connection** checks the
   saved connection; the result distinguishes authentication, timeout and
   unsupported model management.
2. **Models → Download model** to download a GGUF, or drop files in `--models-dir` and restart the router.
3. **Load**, then pick the model on the agent chip.

Quantization choices show the source file size and a conservative runtime
memory estimate. Guidance is a starting point; actual memory also depends on
context length, runtime settings and other loaded models.

| | pi TUI | PiCode |
|---|---|---|
| Login | URL + optional key | same |
| Manage | `/llama` | Models, Server and Activity pages |

## Follow an operation

Load, unload and download continue when you leave the page. Open **Activity**
to see their status and file progress. Returning or reconnecting refreshes the
history. An unknown file size shows progress without an invented percentage.

**Result unknown** means PiCode could not confirm the outcome. Use **Check
result** to consult the original server; this does not repeat the operation.
If you changed the connection, restore the original server and key first.
An unresolved operation blocks another operation on the same model. Choosing
**Unload other models first** also waits for other PiCode model jobs to finish.

**Cancel download** appears only for a running download on the verified
b10809 server build family. PiCode waits for confirmation that downloading
stopped. A download that finishes first is shown as completed. Other builds
show a cancellation limitation; load and unload cannot be canceled here.

After a PiCode restart, queued work is interrupted and dispatched work is
checked without repeating commands. History retains the last 50 operations
and every unresolved operation. On modest hardware, load one model at a time
and unload it when finished; these controls do not tune GPU or context settings.
