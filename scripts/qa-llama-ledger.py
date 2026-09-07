#!/usr/bin/env python3
"""Opt-in real download/ownership/cleanup acceptance on a fresh docs fixture.

Requires a separately started picode-docs-fixture and the verified b10809 CPU
archive. Never loads a model. Only the synthetic fixture's private files change.
"""
import argparse
import hashlib
import json
import pathlib
import platform
import shutil
import socket
import sqlite3
import time
import urllib.error
import urllib.request

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--base', required=True)
parser.add_argument('--archive', required=True)
parser.add_argument('--report', required=True)
args = parser.parse_args()
model = 'ggml-org/Qwen3-0.6B-GGUF:Q4_0'
report = {'model': model, 'version': 'b10809', 'checks': [], 'pass': False}


def api(path, body=None, method=None, expected=200):
    request = urllib.request.Request(args.base + path,
        data=None if body is None else json.dumps(body).encode(),
        method=method or ('GET' if body is None else 'POST'),
        headers={'Content-Type': 'application/json'})
    try:
        response = urllib.request.urlopen(request, timeout=60)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        data = json.load(response)
        assert response.status == expected, (path, response.status, data)
        return data


def wait(read, predicate, seconds=120):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        value = read()
        if predicate(value):
            return value
        time.sleep(1)
    raise AssertionError(('Timed out', value))


def snapshot():
    return api('/api/llama/service')


def review(action, files=None, expected=200):
    return api('/api/llama/service/preview', {
        'action': action, 'version': 'b10809', 'files': files or [],
        'revision': snapshot()['revision']}, expected=expected)


def execute(preview, expected=202):
    return api('/api/llama/service/execute', {
        'token': preview['token'], 'interrupt': True}, expected=expected)


def action(name, files=None):
    job = execute(review(name, files))['job']
    state = wait(snapshot, lambda s: not s['busy'] and s['jobs'][0]['id'] == job['id'])
    assert state['jobs'][0]['state'] == 'succeeded', state['jobs'][0]
    print(name + ' succeeded', flush=True)
    return state


fleet = api('/api/workspaces')
assert fleet and all(w['path'].startswith('/tmp/picode-docs-fixture-') for w in fleet)
root = pathlib.Path(fleet[0]['path']).parents[1]
assert root.parent == pathlib.Path('/tmp') and root.name.startswith('picode-docs-fixture-')
assert not snapshot()['created'], 'A fresh, unconfigured fixture is required'
with socket.socket() as listener:
    listener.bind(('127.0.0.1', 0))
    port = listener.getsockname()[1]

try:
    state = api('/api/llama/service', {'revision': 0, 'config': {
        'port': port, 'threads': 2, 'context': 4096, 'jinja': True}}, method='PUT')
    owned = pathlib.Path(state['modelsDir']).parent
    assert owned == root / 'llama-owned'
    arch = 'arm64' if platform.machine() == 'aarch64' else 'x64'
    shutil.copyfile(args.archive, owned / 'cache' / ('llama-b10809-bin-ubuntu-' + arch + '.tar.gz'))
    sentinel = owned / 'models' / 'unknown.txt'
    sentinel.write_text('unowned sentinel')
    action('install')
    state = action('start')
    api('/api/providers/llama.cpp', {'url': state['url'], 'key': ''}, method='PUT')
    job = api('/api/llama/download', {'id': model, 'requestKey': 'real-ledger-download'}, expected=202)['job']
    report['jobID'] = job['id']
    print('Downloading ' + model + ' without loading inference', flush=True)
    def read_job():
        return next(j for j in api('/api/llama/jobs')['jobs'] if j['id'] == job['id'])
    done = wait(read_job, lambda j: j['state'] in ('succeeded', 'failed', 'interrupted'), 600)
    report['download'] = done
    assert done['state'] == 'succeeded', done
    state = wait(snapshot, lambda s: any(m['job'] == job['id'] for m in (s.get('models') or {}).values()), 30)
    name, entry = next((n, m) for n, m in state['models'].items() if m['job'] == job['id'])
    path = owned / name
    assert path.is_relative_to(owned / 'models') and not path.is_symlink()
    with path.open('rb') as file:
        digest = hashlib.file_digest(file, 'sha256').hexdigest()
    assert digest == entry['sha256'] and path.stat().st_size == entry['size']
    report['ownedFile'] = {'name': name, 'size': entry['size'], 'sha256': digest}
    with urllib.request.urlopen(state['url'] + '/models') as response:
        catalog = json.load(response)['data']
    assert all(m['status']['value'] == 'unloaded' for m in catalog), catalog
    report['checks'].append('real download succeeds; exact path, size and SHA-256 recorded; model remains unloaded')
    with sqlite3.connect('file:' + str(root / 'picode.db') + '?mode=ro', uri=True) as database:
        persisted = json.loads(database.execute('select payload from llama_service where id=1').fetchone()[0])
    assert persisted['models'][name] == entry
    report['checks'].append('ownership persisted in SQLite')
    review('cleanup', [name], expected=409)
    action('stop')
    refusal = review('cleanup', [name], expected=409)
    assert 'Referenced by' in refusal['error'], refusal
    report['checks'].append('running service and configured references both refuse cleanup')
    for workspace in fleet:
        for agent in workspace.get('agents', []):
            api('/api/agents/' + agent['id'], {'provider': 'openai', 'model': 'unrelated-fixture-model'}, method='PATCH')
    for agent in api('/api/agents?free=1'):
        api('/api/agents/' + agent['id'], {'provider': 'openai', 'model': 'unrelated-fixture-model'}, method='PATCH')
    review('cleanup', ['models/unknown.txt'], expected=409)
    report['checks'].append('preexisting unowned file cannot be selected')
    preview = review('cleanup', [name])
    assert preview['bytes'] == entry['size']
    # Alter and restore one byte: same-sized changes must fail hash revalidation.
    with path.open('r+b') as file:
        original = file.read(1)
        file.seek(0)
        file.write(bytes([original[0] ^ 1]))
    execute(preview, expected=409)
    assert path.exists()
    with path.open('r+b') as file:
        file.write(original)
    report['checks'].append('same-size modification after review refuses deletion')
    action('cleanup', [name])
    assert not path.exists() and name not in (snapshot().get('models') or {})
    if entry.get('snapshot'):
        assert not (owned / entry['snapshot']).is_symlink()
    assert sentinel.read_text() == 'unowned sentinel'
    report['checks'].append('reviewed exact model removed; ownership removed; unowned sentinel retained')
    report['pass'] = True
finally:
    report_path = pathlib.Path(args.report)
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report), flush=True)
