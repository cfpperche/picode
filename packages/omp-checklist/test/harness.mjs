// Fires one event in a fresh process for the mirror's silence table: a
// pending client request would keep this process alive, so a clean exit is
// the verdict that nothing was published. The recorder URL travels in
// PICODE_TERM_URL; the event, its payload and the branch fixture in argv.
import ompChecklist from "../extensions/checklist.ts";

const [event, payload, mode, branch] = process.argv.slice(2);
const handlers = new Map();
ompChecklist({ on(name, handler) { handlers.set(name, handler); } });
const handler = handlers.get(event);
if (!handler) throw new Error("missing handler: " + event);
const ctx = {
	mode,
	sessionManager: {
		getSessionId: () => "native-omp-1",
		getBranch: () => JSON.parse(branch || "[]"),
	},
};
await handler(payload ? JSON.parse(payload) : {}, ctx);
