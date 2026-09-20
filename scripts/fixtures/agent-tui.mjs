#!/usr/bin/env node
// Disposable terminal acceptance fixture. No model, network or credentials.
if (process.argv.includes("--version")) { console.log("agent-tui-fixture 1.0.0"); process.exit(0); }
let wheels = 0;
let input = "";
process.stdout.write("\x1b[?1049h\x1b[?1000h\x1b[?1006h\x1b[?2004h\x1b[2J\x1b[HAgent TUI fixture\r\nScroll events: 0\r\nReady for input");
if (process.stdin.isTTY) process.stdin.setRawMode(true);
process.stdin.resume();
process.stdin.on("data", data => {
  const text = data.toString();
  input = (input + text).slice(-8192);
  wheels += [...text.matchAll(/\x1b\[<(?:64|65);\d+;\d+[Mm]/g)].length;
  process.stdout.write("\x1b[2;1H\x1b[2KScroll events: " + wheels);
  if (input.includes("QA-MESSAGE")) process.stdout.write("\x1b[4;1HQA-MESSAGE received");
});
process.on("SIGTERM", () => process.exit(0));
