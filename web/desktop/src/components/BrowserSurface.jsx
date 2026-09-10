import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { drawRect, keyboardMessages, mapPoint, pointerMessage } from "@picode/shared/domain/browserInput.js";
import { buildPickPrompt, pickIsValid } from "@picode/shared/domain/browserDesign.js";
import { IconMonitor } from "./Icons.jsx";

// Browser surface (ADR-0114/0115 + design phase): a live view of the
// agent's browser. Frames render on a canvas; with consent the view can
// drive the browser (Control); Design mode picks an element and hands the
// point to the AGENT, which inspects it with its own tools (ADR-0003) —
// the surface never parses the page.

const EMPTY = "no-browser";
const UNREACHABLE = "engine-unreachable";

export default function BrowserSurface({ agentId, onBack }) {
	const canvasRef = useRef(null);
	const stageRef = useRef(null);
	const frameRef = useRef(null); // last decoded frame (Image)
	const metaRef = useRef(null); // its { deviceWidth, deviceHeight }
	const socketRef = useRef(null);
	const [phase, setPhase] = useState("connecting"); // connecting | live | idle | unreachable | ended
	const [pageUrl, setPageUrl] = useState("");
	const [control, setControl] = useState("watch"); // watch | pending | on
	const [design, setDesign] = useState(false);
	const [pick, setPick] = useState(null); // { vx, vy, sx, sy } viewport + stage coords
	const [pickText, setPickText] = useState("");
	const [sending, setSending] = useState(false);
	const [notice, setNotice] = useState("");

	const drawFrame = useCallback(() => {
		const canvas = canvasRef.current;
		const frame = frameRef.current;
		if (!canvas || !frame) return;
		const stage = stageRef.current;
		const dpr = window.devicePixelRatio || 1;
		const w = Math.max(1, Math.floor((stage ? stage.clientWidth : canvas.clientWidth) * dpr));
		const h = Math.max(1, Math.floor((stage ? stage.clientHeight : canvas.clientHeight) * dpr));
		if (canvas.width !== w || canvas.height !== h) {
			canvas.width = w;
			canvas.height = h;
		}
		const ctx = canvas.getContext("2d");
		if (!ctx) return;
		ctx.fillStyle = "#000";
		ctx.fillRect(0, 0, w, h);
		const scale = Math.min(w / frame.width, h / frame.height);
		const dw = Math.max(1, Math.floor(frame.width * scale));
		const dh = Math.max(1, Math.floor(frame.height * scale));
		ctx.imageSmoothingEnabled = scale < 1;
		ctx.drawImage(frame, Math.floor((w - dw) / 2), Math.floor((h - dh) / 2), dw, dh);
	}, []);

	// Refit the last frame when the pane resizes.
	useEffect(() => {
		const stage = stageRef.current;
		if (!stage || typeof ResizeObserver === "undefined") return;
		const ro = new ResizeObserver(() => drawFrame());
		ro.observe(stage);
		return () => ro.disconnect();
	}, [drawFrame]);

	useEffect(() => {
		if (!agentId) return;
		let disposed = false;
		let bitmapUrl = null;
		setPhase("connecting");
		setPageUrl("");
		setDesign(false);
		setPick(null);
		frameRef.current = null;
		metaRef.current = null;

		const proto = location.protocol === "https:" ? "wss" : "ws";
		const ws = new WebSocket(`${proto}://${location.host}/ws/browser?agent=${encodeURIComponent(agentId)}`);
		socketRef.current = ws;

		ws.onopen = () => {
			// Bound the bandwidth: 12 fps is plenty for supervising an agent.
			try { ws.send(JSON.stringify({ type: "config", maxFps: 12 })); } catch { /* closed */ }
		};

		ws.onmessage = (event) => {
			if (disposed) return;
			let msg;
			try { msg = JSON.parse(event.data); } catch { return; }
			if (msg.type === "status") {
				if (msg.connected) setPhase("live");
				else if (msg.reason === UNREACHABLE) setPhase("unreachable");
				else setPhase("idle");
				return;
			}
			if (msg.type === "url" && msg.url) {
				setPageUrl(msg.url);
				return;
			}
			if (msg.type === "error") {
				// The daemon's consent mirror is the truth: if control was
				// flipped off under us, say so and drop back to watch.
				if (msg.code === "watch-only" || msg.code === "read-only") {
					setControl("watch");
					setNotice("Control is off for this session (/browser-input).");
				} else {
					setNotice(msg.message || "The browser rejected a control message.");
				}
				return;
			}
			if (msg.type === "frame" && msg.data) {
				try {
					const blob = base64ToBlob(msg.data);
					if (bitmapUrl) URL.revokeObjectURL(bitmapUrl);
					bitmapUrl = URL.createObjectURL(blob);
					const img = new Image();
					img.onload = () => {
						if (disposed) return;
						frameRef.current = img;
						if (msg.metadata && msg.metadata.deviceWidth) {
							metaRef.current = { deviceWidth: msg.metadata.deviceWidth, deviceHeight: msg.metadata.deviceHeight };
						}
						drawFrame();
					};
					img.src = bitmapUrl;
				} catch { /* malformed frame: skip */ }
			}
		};

		ws.onclose = () => {
			if (disposed) return;
			setPhase((p) => (p === "live" ? "ended" : p === "connecting" ? "idle" : p));
		};
		ws.onerror = () => { /* onclose follows; it owns the state */ };

		return () => {
			disposed = true;
			if (bitmapUrl) URL.revokeObjectURL(bitmapUrl);
			try { ws.close(); } catch { /* already closed */ }
			if (socketRef.current === ws) socketRef.current = null;
		};
	}, [agentId, drawFrame]);

	const sendInput = useCallback((msg) => {
		const ws = socketRef.current;
		if (!ws || ws.readyState !== 1) return;
		try { ws.send(JSON.stringify(msg)); } catch { /* closed mid-send */ }
	}, []);

	// Control toggle: consent lives in the session's capture dir (ADR-0115);
	// the chip flips it through the daemon route and mirrors the honest state.
	const toggleControl = useCallback(async () => {
		if (control === "pending") return;
		const turningOn = control !== "on";
		setControl("pending");
		try {
			await api(`/api/agents/${encodeURIComponent(agentId)}/browser-input`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ on: turningOn }) });
			setControl(turningOn ? "on" : "watch");
			if (turningOn) {
				setNotice("Control on. Click the page, type with the keyboard — every input goes to the agent's browser.");
				stageRef.current?.focus();
			} else {
				setNotice("");
			}
		} catch {
			setControl(turningOn ? "watch" : "on");
			setNotice("Could not change control: the agent is not reachable.");
		}
	}, [agentId, control]);

	// The letterboxed drawn rect of the current frame in canvas CSS pixels.
	const currentRect = useCallback(() => {
		const canvas = canvasRef.current;
		const frame = frameRef.current;
		if (!canvas || !frame) return null;
		const dpr = window.devicePixelRatio || 1;
		return drawRect(canvas.width / dpr, canvas.height / dpr, frame.width, frame.height);
	}, []);

	// Crop the page area out of the rendered canvas as base64 JPEG.
	const cropFrame = useCallback(() => {
		const canvas = canvasRef.current;
		const frame = frameRef.current;
		if (!canvas || !frame) return null;
		const rect = currentRect();
		if (!rect) return null;
		const dpr = window.devicePixelRatio || 1;
		const out = document.createElement("canvas");
		out.width = Math.max(1, Math.round(rect.w * dpr));
		out.height = Math.max(1, Math.round(rect.h * dpr));
		const ctx = out.getContext("2d");
		if (!ctx) return null;
		ctx.drawImage(canvas, rect.x * dpr, rect.y * dpr, rect.w * dpr, rect.h * dpr, 0, 0, out.width, out.height);
		return out.toDataURL("image/jpeg", 0.85).split(",")[1] || null;
	}, [currentRect]);

	// Pointer → design pick, else input_mouse / input_touch (Control).
	const onPointer = useCallback((phaseName) => (e) => {
		const live = phase === "live";
		if (!live) return;
		const canvas = canvasRef.current;
		const frame = frameRef.current;
		const meta = metaRef.current;
		if (!canvas || !frame || !meta) return;
		const stage = stageRef.current;
		const dpr = window.devicePixelRatio || 1;
		const box = canvas.getBoundingClientRect();
		const rect = drawRect(canvas.width / dpr, canvas.height / dpr, frame.width, frame.height);
		const point = mapPoint(e.clientX, e.clientY, { left: box.left, top: box.top }, rect, meta.deviceWidth, meta.deviceHeight);

		// Design mode intercepts the press: capture the pick, send nothing.
		if (design) {
			if (phaseName !== "down") return;
			if (!pickIsValid(point, true)) return;
			setPick({
				vx: point.x,
				vy: point.y,
				sx: e.clientX - box.left,
				sy: e.clientY - box.top,
			});
			setPickText("");
			return;
		}

		if (control !== "on") return;
		if (phaseName === "move") {
			const msg = pointerMessage(e.pointerType, "move", point);
			if (msg) sendInput(msg);
			return;
		}
		if (!point) return;
		if (phaseName === "down") e.currentTarget.setPointerCapture?.(e.pointerId);
		const msg = pointerMessage(e.pointerType, phaseName === "down" ? "down" : "up", point, e.pointerId, phaseName === "down" ? 1 : 0);
		if (msg) sendInput(msg);
	}, [control, design, phase, sendInput]);

	const onKeyDown = useCallback((e) => {
		if (design || control !== "on") return;
		for (const msg of keyboardMessages("keydown", e)) sendInput(msg);
		e.preventDefault();
	}, [control, design, sendInput]);

	const onKeyUp = useCallback((e) => {
		if (design || control !== "on") return;
		for (const msg of keyboardMessages("keyup", e)) sendInput(msg);
		e.preventDefault();
	}, [control, design, sendInput]);

	// Send the pick: cropped frame + mapped coordinates + the user's intent,
	// through the same prompt path the composer uses.
	const sendPick = useCallback(async () => {
		if (sending || !pick) return;
		setSending(true);
		try {
			const data = cropFrame();
			if (!data) throw new Error("no frame");
			await api(`/api/agents/${encodeURIComponent(agentId)}/prompt`, {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					kind: "prompt",
					message: buildPickPrompt(pick.vx, pick.vy, pickText),
					images: [{ mimeType: "image/jpeg", data }],
				}),
			});
			setNotice("Pick sent to the agent — follow the turn in the chat.");
			setPick(null);
			setPickText("");
		} catch (err) {
			setNotice("Could not send the pick: " + (err && err.message ? err.message : "unknown error"));
		} finally {
			setSending(false);
		}
	}, [agentId, cropFrame, pick, pickText, sending]);

	const retry = () => {
		if (socketRef.current) { try { socketRef.current.close(); } catch { /* noop */ } }
		setPhase("connecting");
	};

	const live = phase === "live";
	const controlling = control === "on" && !design;

	return (
		<section className="browser-surface" aria-label="Agent browser">
			<div className="browser-bar">
				<span className={"browser-dot" + (live ? " live" : "")} title={live ? "Live" : "Not streaming"} />
				<span className="browser-url" title={pageUrl || ""}>{live || phase === "ended" ? pageUrl || "about:blank" : ""}</span>
				{live ? (
					<>
						<button
							type="button"
							className={"browser-chip browser-chip-btn" + (design ? " on" : "")}
							title={design
								? "Design is on: click an element to describe a change. Click to turn off."
								: "Design: click an element on the page to ask the agent for a change."}
							onClick={() => { setDesign((d) => !d); setPick(null); setNotice(""); }}
						>
							Design
						</button>
						<button
							type="button"
							className={"browser-chip browser-chip-btn" + (controlling ? " on" : "")}
							title={controlling
								? "Control is on: click and type to drive the agent's browser. Click to hand back watch-only."
								: "Watch-only. Click to enable control of the agent's browser."}
							onClick={toggleControl}
						>
							{control === "pending" ? "…" : controlling ? "Control on" : "Watch-only"}
						</button>
					</>
				) : (
					<span className="browser-chip">Watch-only</span>
				)}
				{phase === "unreachable" || phase === "ended" ? (
					<button type="button" className="btn btn-sm" onClick={retry}>Retry</button>
				) : null}
				{phase !== "idle" ? (
					<button type="button" className="btn btn-sm" onClick={onBack}>Back to chat</button>
				) : null}
			</div>
			{notice ? <div className="browser-notice" role="status">{notice}</div> : null}
			{pick ? (
				<form
					className="browser-pick"
					onSubmit={(e) => { e.preventDefault(); void sendPick(); }}
				>
					<span className="browser-pick-coords">({pick.vx}, {pick.vy})</span>
					<input
						type="text"
						className="browser-pick-input"
						placeholder="What should change here?"
						value={pickText}
						autoFocus
						onChange={(e) => setPickText(e.target.value)}
						onKeyDown={(e) => { if (e.key === "Escape") setPick(null); }}
					/>
					<button type="submit" className="btn btn-primary btn-sm" disabled={sending}>{sending ? "Sending…" : "Send"}</button>
					<button type="button" className="btn btn-sm" onClick={() => setPick(null)}>Cancel</button>
				</form>
			) : null}
			<div
				className={"browser-stage" + (controlling ? " controlling" : "") + (design ? " designing" : "")}
				ref={stageRef}
				data-phase={phase}
				tabIndex={controlling ? 0 : -1}
				onPointerDown={onPointer("down")}
				onPointerMove={onPointer("move")}
				onPointerUp={onPointer("up")}
				onKeyDown={onKeyDown}
				onKeyUp={onKeyUp}
				onContextMenu={controlling ? (e) => e.preventDefault() : undefined}
			>
				<canvas ref={canvasRef} className="browser-canvas" hidden={phase !== "live" && phase !== "ended"} />
				{pick ? <span className="browser-pick-dot" style={{ left: pick.sx, top: pick.sy }} aria-hidden="true" /> : null}
				{phase === "connecting" ? (
					<div className="browser-empty" role="status">
						<IconMonitor size={20} />
						<p className="browser-empty-title">Attaching to the browser…</p>
					</div>
				) : null}
				{phase === "idle" ? (
					<div className="browser-empty">
						<IconMonitor size={20} />
						<p className="browser-empty-title">No browser is running in this session.</p>
						<p className="browser-empty-hint">The view attaches when the agent opens one.</p>
						<button type="button" className="btn btn-primary btn-sm" onClick={onBack}>Back to chat</button>
					</div>
				) : null}
				{phase === "unreachable" ? (
					<div className="browser-empty">
						<IconMonitor size={20} />
						<p className="browser-empty-title">The browser stopped responding.</p>
						<button type="button" className="btn btn-primary btn-sm" onClick={retry}>Retry</button>
					</div>
				) : null}
			</div>
		</section>
	);
}

function base64ToBlob(b64) {
	const bin = atob(b64);
	const bytes = new Uint8Array(bin.length);
	for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
	return new Blob([bytes], { type: "image/jpeg" });
}
