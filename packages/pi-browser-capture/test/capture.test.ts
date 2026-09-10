import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdir, mkdtemp, readFile, symlink, unlink, writeFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { boundedJpeg, caption, captionUrl, createImplicitSessionName, findCapturePort, writeInputMirror } from "../extensions/capture.ts";

test("implicit session name matches the upstream shape and is stable", () => {
	const name = createImplicitSessionName("0123abcd-4567-89ef-a000-000000000001", "/home/me/My_Project");
	assert.match(name, /^piab-my-project-[0-9a-f]{12}-[0-9a-f]{8}$/);
	assert.equal(name, createImplicitSessionName("0123abcd-4567-89ef-a000-000000000001", "/home/me/My_Project"));
	assert.notEqual(name, createImplicitSessionName("0123abcd-4567-89ef-a000-000000000002", "/home/me/My_Project"));
	// The session id is normalized: dashes removed, lowercased, then hashed.
	assert.equal(
		createImplicitSessionName("0123ABCD-4567-89EF-A000-000000000001", "/home/me/My_Project"),
		name,
	);
});

test("boundedJpeg enforces SOI/EOI, markers and viewport limits", () => {
	const jpeg = (width: number, height: number) => {
		const bytes = [0xff, 0xd8, 0xff, 0xc0, 0x00, 0x0b, 0x08, height >> 8, height & 255, width >> 8, width & 255, 0x03, 0, 0, 0, 0xff, 0xd9];
		return Buffer.from(bytes).toString("base64");
	};
	assert.match(boundedJpeg(jpeg(32, 16)) ?? "", /^data:image\/jpeg;base64,/);
	assert.equal(boundedJpeg(jpeg(2000, 16)), undefined, "width above 1600 is refused");
	assert.equal(boundedJpeg(jpeg(32, 1500) + "!!!"), undefined, "invalid base64 is refused");
	assert.equal(boundedJpeg(Buffer.from("not a jpeg").toString("base64")), undefined);
});

test("captions strip control characters and non-web URLs", () => {
	assert.equal(caption("a\u0000b\tc"), "a b c");
	assert.equal(caption(42), "");
	assert.equal(captionUrl("https://example.com/path?secret=1#frag"), "https://example.com/path");
	assert.equal(captionUrl("file:///etc/passwd"), undefined);
	assert.equal(captionUrl("javascript:alert(1)"), undefined);
});

test("findCapturePort reads the rendezvous only under strict conditions", async t => {
	const root = await mkdtemp(join(tmpdir(), "piab-capture-test-"));
	t.after(() => rm(root, { recursive: true, force: true }));
	const env = { PI_AGENT_BROWSER_SOCKET_DIR: root } as NodeJS.ProcessEnv;
	await chmod70(root);
	await writeFile(join(root, "sess-a.pid"), `${process.pid}\n`);
	await writeFile(join(root, "sess-a.stream"), "45678\n");
	assert.equal(await findCapturePort("sess-a", "", env), 45678);
	// Oversized or non-numeric files are refused.
	await writeFile(join(root, "sess-b.pid"), `${process.pid}x\n`);
	await writeFile(join(root, "sess-b.stream"), "45679\n");
	assert.equal(await findCapturePort("sess-b", "", env), undefined);
	// Missing pid file.
	await writeFile(join(root, "sess-c.stream"), "45680\n");
	assert.equal(await findCapturePort("sess-c", "", env), undefined);
	// Prefix match covers -fresh- rotations.
	await writeFile(join(root, "sess-a-fresh-abc123.pid"), `${process.pid}\n`);
	await writeFile(join(root, "sess-a-fresh-abc123.stream"), "45681\n");
	const port = await findCapturePort("sess-a", "", env);
	assert.ok([45678, 45681].includes(port!), "base and fresh rotation must both be candidates");
	// Symlinked root is refused.
	const outside = await mkdtemp(join(tmpdir(), "piab-outside-"));
	t.after(() => rm(outside, { recursive: true, force: true }));
	await rm(root, { recursive: true, force: true });
	await symlink(outside, root);
	assert.equal(await findCapturePort("sess-a", "", env), undefined);
	await unlink(root);
});

async function chmod70(path: string) {
	await chmod(path, 0o700);
}

import { chmod } from "node:fs/promises";

test("writeInputMirror writes atomic consent state beside the session file", async () => {
	const sessionFile = join(await mkdtemp(join(tmpdir(), "pi-input-")), "session.jsonl");
	await writeInputMirror(sessionFile, true);
	const dir = `${sessionFile}.capture`;
	const raw = JSON.parse(await readFile(join(dir, "input.json"), "utf8"));
	assert.equal(raw.on, true);
	assert.equal(typeof raw.ts, "number");
	await writeInputMirror(sessionFile, false);
	const off = JSON.parse(await readFile(join(dir, "input.json"), "utf8"));
	assert.equal(off.on, false);
	await rm(dir, { recursive: true, force: true });
});
