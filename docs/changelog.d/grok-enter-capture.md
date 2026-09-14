### Fixed

- **Communication delivery survives slow TUI redraws.** A native CLI
  (Grok and the others) that renders the received prompt late under load no
  longer ends a delivery as uncertain: the post-paste composer check is
  sampled inside a bounded window, and a refusal log names which guard
  refused without exposing screen content.
