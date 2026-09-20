### Changed

- The prompt door now verifies deliveries for CLIs with a measured input
  reader: the composer must read empty before and after, and refusals
  name themselves (working, needs-you, occupied, busy). The response
  carries a delivery receipt; CLIs without a reader answer "unverified".
- Automations can target CLI agents: the prompt is delivered through the
  agent's launch terminal via the door, and the receipt becomes the run's
  outcome (done / skipped / failed with the door's named reason).
- The graph and pane ask on a non-pi CLI terminal is delivered through
  the door (fire and forget, provenance recorded).
