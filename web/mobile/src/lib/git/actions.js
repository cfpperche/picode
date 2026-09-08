// Mobile reaches ADR-0078's terminal and agent channels through the same
// composer as desktop and the git graph (ADR-0096). This file was a verbatim
// copy of the desktop block until then; it is now the mobile-side name for
// the shared module, so mobile keeps its own presentation and shares the
// logic (ADR-0072, ADR-0095).
export {
  shellQuote,
  gitActionCommand,
  gitActions,
  branchChip,
  askableAgents,
  askChannelHint,
  askGitPrompt,
  askedNote,
} from "@picode/shared/domain/gitCommands.js";
