// PiCode Inbox reply receiver (ADR-0060). Injected with -e into interactive
// agents spawned by PiCode and, since ADR-0089's 2026-09-08 amendment, into
// pi launched as an Agent CLI terminal. It consumes the daemon's one-shot
// reply files and submits each reply through the TUI's own message path, so
// it renders in this terminal exactly like a typed message — queued natively
// while a turn is streaming. It also says hello every few minutes so the
// daemon knows a live receiver exists before choosing this channel over tmux
// paste; a terminal's hello carries its session file, which is how the
// daemon learns which conversation an Ask would land in.

import { existsSync, mkdirSync, readFileSync, readdirSync, unlinkSync, watch as fsWatch } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";

const agentID = (process.env.PICODE_AGENT_ID || "").trim();
const termID = (process.env.PICODE_TERM_ID || "").trim();
// An agent's identity wins: a pi spawned by PiCode as an agent also has a
// terminal id, and its replies must keep landing where they always did.
const owner = agentID ? { kind: "agents", id: agentID, dir: agentID } : termID ? { kind: "terminals", id: termID, dir: "term-" + termID } : null;
const dataDir = process.env.PICODE_DATA || join(homedir(), ".picode");

/**
 * Where the daemon answers. A terminal is told outright through
 * PICODE_TERM_URL (ADR-0056 — the same variable the activity hook uses), which
 * also keeps a pi launched by a scratch instance from reporting to the
 * production one; an agent re-reads server.json every post, since the port
 * can rebind at runtime.
 */
function serverBase() {
	const told = (process.env.PICODE_TERM_URL || "").trim();
	if (owner && owner.kind === "terminals" && told) return told.replace("//127.0.0.1:", "//localhost:");
	try {
		const raw = JSON.parse(readFileSync(join(dataDir, "server.json"), "utf8"));
		const url = String(raw.url || "").trim();
		if (!url) return null;
		// The mkcert CA is not in Node's store; loopback posts skip
		// verification (the in-repo pi-inbox precedent) and prefer the
		// hostname the cert was issued for.
		return url.replace("//127.0.0.1:", "//localhost:");
	} catch {
		return null;
	}
}

function post(path, body) {
	return new Promise((resolve) => {
		const base = serverBase();
		if (!base) return resolve();
		let payload;
		try {
			payload = JSON.stringify(body ?? {});
		} catch {
			return resolve();
		}
		let req;
		try {
			const fn = base.startsWith("https:") ? httpsRequest : httpRequest;
			req = fn(base + path, {
				method: "POST",
				headers: { "content-type": "application/json", "content-length": Buffer.byteLength(payload) },
				rejectUnauthorized: false,
				timeout: 5000,
			}, (res) => {
				res.resume();
				res.on("end", () => resolve());
				res.on("error", () => resolve());
			});
		} catch {
			return resolve();
		}
		req.on("timeout", () => req.destroy(new Error("timeout")));
		req.on("error", () => resolve());
		req.end(payload);
	});
}

export default function (pi) {
	if (!owner) return;

	const replyDir = join(dataDir, "tui-inbox", owner.dir);
	let latestCtx = null;
	let draining = false;

	const sessionFile = () => {
		try {
			return latestCtx?.sessionManager?.getSessionFile?.() || "";
		} catch {
			return "";
		}
	};

	const hello = () => {
		void post(`/api/${owner.kind}/${owner.id}/tui-hello`, { session: sessionFile() });
	};

	async function drain() {
		if (draining) return;
		draining = true;
		try {
			if (!existsSync(replyDir)) return;
			for (const name of readdirSync(replyDir)) {
				if (!name.endsWith(".json")) continue;
				const file = join(replyDir, name);
				let doc = null;
				try {
					doc = JSON.parse(readFileSync(file, "utf8"));
				} catch {
					unlinkSync(file);
					continue;
				}
				const ack = async (ok, reason) => {
					try {
						unlinkSync(file);
					} catch {}
					await post(`/api/${owner.kind}/${owner.id}/tui-ack`, { nonce: doc.nonce, ok, reason });
				};
				const age = Date.now() - (doc.createdAt ? Date.parse(doc.createdAt) : 0);
				if (!doc.nonce || !doc.payload || (Number.isFinite(age) && age > 24 * 60 * 60 * 1000)) {
					await ack(false, "the reply file was stale");
					continue;
				}
				const session = sessionFile();
				if (!doc.sessionPath || !session || session !== doc.sessionPath) {
					// The operator switched this TUI to another session: the
					// exact-session rule wins, and the item reopens.
					await ack(false, "the terminal is showing a different session");
					continue;
				}
                if (doc.attentionOnly && (!latestCtx?.isIdle?.() || latestCtx?.hasPendingMessages?.() ||
                    (latestCtx?.hasUI && latestCtx?.ui?.getEditorText?.() !== ""))) {
                    await ack(false, "the conversation is busy or has a draft");
                    continue;
                }
				try {
					await pi.sendUserMessage(doc.payload, { deliverAs: "followUp", triggerTurn: true });
					await ack(true, "");
				} catch (err) {
					await ack(false, "the terminal could not submit the reply: " + String(err?.message || err));
				}
			}
		} catch {
			// The directory vanished or a transient read failed: the daemon's
			// ack wait and boot reconciliation own the truth.
		} finally {
			draining = false;
		}
	}

	try {
		mkdirSync(replyDir, { recursive: true });
	} catch {}

	setInterval(hello, 5 * 60 * 1000).unref?.();
	setInterval(drain, 1000).unref?.();
	try {
		fsWatch(replyDir, () => void drain());
	} catch {}

	pi.on("session_start", async (_event, ctx) => {
		latestCtx = ctx;
		hello();
		void drain();
	});
	pi.on("session_shutdown", async () => {
		hello();
	});
}
