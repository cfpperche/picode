#!/usr/bin/env python3
"""Exercise an explicitly selected disposable llama router model (stdlib only).

Loads and unloads the selected model. Refuses an already active model.
No downloads, service changes, credentials, or production Pi configuration.
"""

import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile
import threading
import time
import urllib.request


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--url", required=True)
    parser.add_argument("--model", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--pi", help="Optional real Pi executable for a read-tool round trip")
    args = parser.parse_args()
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    report = {"model": args.model, "checks": {}, "limitations": [
        "One small model and selected build; not GPU performance or coding readiness",
        "Does not validate PiCode browser jobs, downloads, cancellation, or restart recovery",
    ]}
    output = Path(args.out)
    output.parent.mkdir(parents=True, exist_ok=True)

    def request(path, body=None):
        data = None if body is None else json.dumps(body).encode()
        req = urllib.request.Request(args.url.rstrip("/") + path, data=data,
                                     headers={"Content-Type": "application/json"})
        return opener.open(req, timeout=120)

    def api(path, body=None):
        with request(path, body) as response:
            return json.load(response)

    def state():
        for model in api("/models")["data"]:
            if model["id"] == args.model:
                return model["status"]["value"]
        return "missing"

    def wait(wanted):
        deadline = time.monotonic() + 120
        while time.monotonic() < deadline:
            current = state()
            if current == wanted:
                return
            if current == "failed":
                raise RuntimeError("model entered failed state")
            time.sleep(0.5)
        raise RuntimeError("model transition timed out")

    selected = False
    try:
        props = api("/props")
        report["build"] = props.get("build_info")
        report["checks"]["router"] = props.get("role") == "router"
        if not report["checks"]["router"] or state() != "unloaded":
            raise RuntimeError("requires a router and an explicitly selected unloaded model")
        subscribed = threading.Event()
        observed = []

        def observe_load():
            try:
                with request("/models/sse") as response:
                    if response.headers.get_content_type() != "text/event-stream":
                        return
                    subscribed.set()
                    for raw in response:
                        line = raw.decode().strip()
                        if not line.startswith("data: "):
                            continue
                        event = json.loads(line[6:])
                        if event.get("model") == args.model:
                            observed.append(event.get("event"))
                            if event.get("data", {}).get("status") == "loaded":
                                return
            except Exception:
                pass

        observer = threading.Thread(target=observe_load, daemon=True)
        observer.start()
        report["checks"]["sseSubscription"] = subscribed.wait(5)
        selected = True
        api("/models/load", {"model": args.model})
        wait("loaded")
        report["checks"]["load"] = True
        observer.join(timeout=5)
        report["checks"]["sseLoadEvent"] = "model_status" in observed
        base = {"model": args.model, "temperature": 0, "max_tokens": 256,
                "chat_template_kwargs": {"enable_thinking": False}}
        chunks = []
        done = False
        with request("/v1/chat/completions", {**base, "stream": True,
                     "messages": [{"role": "user", "content": "Reply with exactly: LLAMA_STREAM_OK"}]}) as response:
            for raw in response:
                line = raw.decode().strip()
                if line == "data: [DONE]":
                    done = True
                elif line.startswith("data: "):
                    event = json.loads(line[6:])
                    for choice in event.get("choices", []):
                        chunk = choice.get("delta", {}).get("content")
                        if chunk:
                            chunks.append(chunk)
        report["checks"]["stream"] = done and len(chunks) > 1 and "LLAMA_STREAM_OK" in "".join(chunks)
        report["streamChunks"] = len(chunks)
        response = api("/v1/chat/completions", {**base,
            "messages": [{"role": "user", "content": "Call qa_echo with value LLAMA_TOOL_OK."}],
            "tools": [{"type": "function", "function": {"name": "qa_echo",
                       "description": "Echo a test value", "parameters": {"type": "object",
                       "properties": {"value": {"type": "string"}}, "required": ["value"]}}}],
            "tool_choice": {"type": "function", "function": {"name": "qa_echo"}}})
        calls = response["choices"][0]["message"].get("tool_calls", [])
        report["checks"]["toolProtocol"] = bool(calls) and calls[0]["function"]["name"] == "qa_echo" and json.loads(calls[0]["function"]["arguments"]) == {"value": "LLAMA_TOOL_OK"}
        if args.pi:
            with tempfile.TemporaryDirectory(prefix="llama-pi-qa-") as tmp:
                root = Path(tmp)
                config = root / "agent"
                config.mkdir()
                (root / "marker.txt").write_text("LLAMA_PI_READ_OK_7842\n")
                (config / "models.json").write_text(json.dumps({"providers": {"llama-qa": {
                    "baseUrl": args.url.rstrip("/") + "/v1", "api": "openai-completions",
                    "apiKey": "local-test", "compat": {"supportsDeveloperRole": False,
                    "supportsReasoningEffort": False}, "models": [{"id": args.model,
                    "reasoning": False, "contextWindow": 8192, "maxTokens": 1024}]}}}))
                command = [args.pi, "--mode", "json", "--print", "--no-session", "--offline",
                           "--no-extensions", "--no-skills", "--no-prompt-templates", "--no-context-files",
                           "--provider", "llama-qa", "--model", args.model, "--thinking", "off",
                           "--tools", "read", "--system-prompt",
                           "Read only marker.txt in the current directory when asked. Report its exact contents.",
                           "Use the read tool to read marker.txt, then report its contents."]
                run = subprocess.run(command, cwd=tmp, env={**os.environ,
                    "PI_CODING_AGENT_DIR": str(config)}, capture_output=True, text=True, timeout=180)
                events = []
                for line in run.stdout.splitlines():
                    try:
                        events.append(json.loads(line))
                    except json.JSONDecodeError:
                        pass
                tool_done = any(e.get("type") == "tool_execution_end" and e.get("toolName") == "read" and not e.get("isError") for e in events)
                answers = [e.get("message", {}) for e in events if e.get("type") == "message_end" and e.get("message", {}).get("role") == "assistant"]
                report["checks"]["piReadRoundTrip"] = run.returncode == 0 and tool_done and any("LLAMA_PI_READ_OK_7842" in json.dumps(a) for a in answers)
                report["piExit"] = run.returncode
                report["piEventTypes"] = sorted({e.get("type", "unknown") for e in events})
                if not report["checks"]["piReadRoundTrip"]:
                    report["piDiagnostic"] = json.dumps(answers)[-4000:]
    except Exception as error:
        report["error"] = type(error).__name__ + ": " + str(error)
    finally:
        if selected:
            try:
                api("/models/unload", {"model": args.model})
                wait("unloaded")
                report["checks"]["unload"] = True
            except Exception:
                report["checks"]["unload"] = False
        report["pass"] = bool(report["checks"]) and all(report["checks"].values()) and "error" not in report
        output.write_text(json.dumps(report, indent=2) + "\n")
        print(json.dumps(report, indent=2))
    return 0 if report["pass"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
