### Fixed
- **Lifecycle buttons on mobile (and Muse Code on desktop).** The Install
  and Update buttons were gated on the integration capability — inverted
  on mobile (`!cap.integration`) — so update-capable rows without it
  never showed Update. Both buttons now follow only the lifecycle flags,
  like the `···` menu already did.
