// Opt-in real Pi/model/browser smoke for the browser-capture SIDECAR.
// The native pi-agent-browser-native checkout must be CLEAN (no capture patch):
// frames must reach the capture directory through the sidecar extension alone.
// Usage: node scripts/verify-browser-capture-sidecar.mjs <native-checkout> <receipt-dir> [provider/model]
import { spawn } from 'node:child_process';
import { createServer } from 'node:http';
import { mkdir, mkdtemp, copyFile, readFile, writeFile, rm } from 'node:fs/promises';
import { createWriteStream } from 'node:fs';
import { tmpdir, homedir } from 'node:os';
import { join, resolve } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';
import assert from 'node:assert/strict';

const [source, out, model = 'zai/glm-5.3-flash'] = process.argv.slice(2);
if (!source || !out) throw Error('Provide the native package checkout and private receipt directory.');
const worktree = resolve(new URL('..', import.meta.url).pathname);
const sidecar = join(worktree, 'packages/pi-browser-capture/extensions/capture.ts');
const output = resolve(out); await mkdir(output, { recursive: true, mode: 0o700 });
const home = await mkdtemp(join(tmpdir(), 'capture-sidecar-'));
const agentDir = join(home, 'agent'); await mkdir(agentDir, { mode: 0o700 });
// Use existing credentials only in this disposable Pi home; never print them.
for (const name of ['auth.json', 'models.json']) {
  try { await copyFile(join(homedir(), '.pi/agent', name), join(agentDir, name)); }
  catch (e) { if (e.code !== 'ENOENT') throw e; }
}
await writeFile(join(agentDir, 'settings.json'), JSON.stringify({ packages: [], compaction: { enabled: false }, retry: { enabled: false } }), { mode: 0o600 });
const server = createServer((_request, response) => {
  response.setHeader('Content-Type', 'text/html');
  response.end('<!doctype html><title>Browser capture sidecar proof</title><body style="background:#174b70;color:white;font:48px sans-serif;padding:50px">Sidecar capture proof <span id="tick">0</span><script>let n=0;setInterval(()=>document.querySelector("#tick").textContent=++n,250)</script>');
});
await new Promise(r => server.listen(0, '127.0.0.1', r));
const url = `http://127.0.0.1:${server.address().port}/`;
const child = spawn('pi', ['--mode', 'rpc', '--approve', '--no-extensions', '--no-skills', '--no-prompt-templates', '--no-themes', '-e', resolve(source), '-e', sidecar, '--model', model, '--thinking', 'minimal', '--session-dir', output], {
  cwd: home, env: { ...process.env, PI_CODING_AGENT_DIR: agentDir, PI_OFFLINE: '1' }, stdio: ['pipe', 'pipe', 'pipe'],
});
const log = createWriteStream(join(output, 'rpc.jsonl'), { mode: 0o600 });
const errors = createWriteStream(join(output, 'stderr.log'), { mode: 0o600 }); child.stderr.pipe(errors);
const events = []; let buffer = '', next = 0;
child.stdout.setEncoding('utf8');
child.stdout.on('data', chunk => {
  buffer += chunk;
  for (let i; (i = buffer.indexOf('\n')) >= 0;) {
    const line = buffer.slice(0, i); buffer = buffer.slice(i + 1);
    if (!line.trim()) continue;
    log.write(line + '\n');
    try { events.push({ at: Date.now(), event: JSON.parse(line) }); } catch {}
  }
});
const send = command => child.stdin.write(JSON.stringify(command) + '\n');
async function waitFor(test, timeout = 180000, since = 0) {
  const until = Date.now() + timeout;
  while (Date.now() < until) { const value = events.find(r => r.at >= since && test(r)); if (value) return value; if (child.exitCode !== null) throw Error(`Pi exited: ${child.exitCode}`); await delay(50); }
  throw Error('Timed out waiting for Pi; inspect private RPC receipts.');
}
async function request(command) { const id = `sidecar-${++next}`; send({ ...command, id }); const row = await waitFor(r => r.event.type === 'response' && r.event.id === id, 30000); assert.equal(row.event.success, true, row.event.error); return row.event.data; }

// Capture-directory watcher: the proof channel. Reads latest-wins files the
// sidecar writes beside the session, exactly what the PiCode daemon will watch.
function watchCaptureDir(captureDir) {
  const state = { frames: [], maxSeq: 0, sawState: false, finals: [] };
  const timer = setInterval(async () => {
    try {
      const s = JSON.parse(await readFile(join(captureDir, 'state.json'), 'utf8'));
      if (s?.capturing) state.sawState = true;
    } catch {}
    try {
      const meta = JSON.parse(await readFile(join(captureDir, 'current.json'), 'utf8'));
      if (!meta?.final && meta.seq > state.maxSeq) {
        const image = await readFile(join(captureDir, 'current.jpg'));
        state.maxSeq = meta.seq;
        state.frames.push({ seq: meta.seq, at: Date.now(), toolCallId: meta.toolCallId, bytes: image.length, url: meta.url ?? null, jpegOk: image[0] === 0xff && image[1] === 0xd8 && image.at(-1) === 0xd9 && image.length <= 200 * 1024 });
        state.lastImage = image;
      }
      if (meta?.final) state.finals.push({ ...meta, at: Date.now() });
    } catch {}
  }, 150);
  return { state, stop: () => clearInterval(timer) };
}

try {
  await request({ type: 'get_state' });
  const commands = await request({ type: 'get_commands' });
  assert.ok(commands.commands.some(c => c.name === 'browser-captures'), 'sidecar command missing');
  const state0 = await request({ type: 'get_state' });
  const sessionFile = state0.sessionFile;
  assert.ok(sessionFile, 'session file path required for the capture directory');
  const captureDir = `${sessionFile}.capture`;

  // A prompt response only returns once the whole turn has settled. A pure
  // slash-command turn emits no agent_settled at all, so don't wait for one.
  await request({ type: 'prompt', message: '/browser-captures on' });

  // The watcher must run DURING the turn: a prompt response only returns
  // once the whole agent turn has settled.
  const watcher = watchCaptureDir(captureDir);
  const jobAt = Date.now();
  await request({ type: 'prompt', message: `This is a bounded integration test using a local page with dummy data. Use the native agent_browser tool, not bash. Make one job call with steps open ${url} and wait 15000 milliseconds, sessionMode fresh. Then close only your browser session with agent_browser and finish with one sentence. Do not create files or change settings.` });
  await waitFor(r => r.event.type === 'agent_settled', 180000, jobAt);
  await delay(1500);
  const { state } = watcher;

  const starts = events.filter(r => r.at >= jobAt && r.event.type === 'tool_execution_start' && r.event.toolName === 'agent_browser');
  const endFor = id => events.find(r => r.event.type === 'tool_execution_end' && r.event.toolCallId === id);
  const jobCall = starts.map(r => ({ id: r.event.toolCallId, start: r.at, end: endFor(r.event.toolCallId)?.at ?? Infinity }))
    .sort((a, b) => (b.end - b.start) - (a.end - a.start))[0];
  assert.ok(jobCall, 'no agent_browser tool call observed');
  const frames = state.frames.filter(f => f.toolCallId === jobCall.id);
  assert.ok(frames.length >= 2, `expected at least 2 intra-call frames, got ${frames.length}`);
  assert.ok(frames.every(f => f.jpegOk), 'every frame must be a bounded JPEG');
  const endAt = endFor(jobCall.id).at;
  assert.ok(endAt - frames[0].at > 1000, 'first frame must land at least 1s before the tool ends');
  const jobFinal = state.finals.find(f => f.toolCallId === jobCall.id);
  assert.ok(jobFinal, 'a final marker must exist for the job call');
  assert.ok(endFor(jobCall.id).at - jobFinal.at < 10000, 'final marker should land right after the tool ends');

  // The clean native package cannot emit capture previews over RPC.
  const leaked = events.filter(r => r.event.type === 'tool_execution_update' && r.event.partialResult?.details && (r.event.partialResult.details.capture || r.event.partialResult.details.preview));
  assert.equal(leaked.length, 0, 'no in-tool capture updates may exist without the patch');

  if (state.lastImage) await writeFile(join(output, 'last-frame.jpg'), state.lastImage, { mode: 0o600 });

  // Phase 2: turning captures off must stop frames for further browser calls.
  await request({ type: 'prompt', message: '/browser-captures off' });
  let offAt = Date.now();
  await request({ type: 'prompt', message: `Use the native agent_browser tool once, not bash: open ${url} with sessionMode fresh, wait 3000 milliseconds, then close only your browser session and finish with one sentence.` });
  await waitFor(r => r.event.type === 'agent_settled', 120000, offAt);
  await delay(1500);
  const after = await readdirSafe(captureDir);
  const afterMeta = await readJsonSafe(join(captureDir, 'current.json'));
  assert.ok(!after?.includes('state.json'), 'no capture may be armed while disabled');
  assert.ok(!afterMeta || afterMeta.final || afterMeta.seq <= state.maxSeq, 'no new frames may be written while disabled');

  // Session entries: consent trail and the persisted final capture.
  const sessionLines = (await readFile(sessionFile, 'utf8')).trim().split('\n').map(l => { try { return JSON.parse(l); } catch { return {}; } });
  const consents = sessionLines.filter(e => e.type === 'custom' && e.customType === 'browser-capture-consent').map(e => e.data);
  assert.deepEqual(consents, [{ enabled: true }, { enabled: false }], 'consent must be branch-persisted');
  const finals = sessionLines.filter(e => e.type === 'custom' && e.customType === 'browser-capture-final');
  assert.ok(finals.some(e => e.data?.toolCallId === jobCall.id && typeof e.data?.image === 'string' && e.data.image.startsWith('data:image/jpeg;')), 'final frame must be persisted for replay');

  const summary = {
    model, url, sessionFile,
    jobCall: { toolCallId: jobCall.id, durationMs: endFor(jobCall.id).at - jobCall.start },
    frames: frames.map(f => ({ seq: f.seq, beforeEndMs: endAt - f.at, bytes: f.bytes, url: f.url })),
    finalMarker: jobFinal,
    persistedFinals: finals.length,
    killSwitch: { maxSeqAfterOff: afterMeta?.seq ?? 0, framesStillValid: after ?? [] },
  };
  await writeFile(join(output, 'summary.json'), JSON.stringify(summary, null, 2));
  console.log(JSON.stringify(summary, null, 2));
  log.end(); server.closeAllConnections(); await new Promise(r => server.close(r));
  if (child.exitCode === null) child.kill('SIGKILL');
  await rm(home, { recursive: true, force: true });
  process.exit(0);
} finally {
  if (child.exitCode === null) { send({ type: 'abort' }); await delay(500); child.kill('SIGTERM'); }
  await Promise.race([new Promise(r => child.once('close', r)), delay(10000)]);
  if (child.exitCode === null) child.kill('SIGKILL');
  log.end(); server.closeAllConnections(); await new Promise(r => server.close(r));
  await rm(home, { recursive: true, force: true });
}

async function readdirSafe(dir) { try { return await (await import('node:fs/promises')).readdir(dir); } catch { return []; } }
async function readJsonSafe(file) { try { return JSON.parse(await readFile(file, 'utf8')); } catch { return undefined; } }
