# Live desktop overlay acceptance

Plan: `docs/plans/live-desktop-overlays.md`; boundary: ADR-0161.

The Windows product shell was exercised on an isolated scratch instance at 150 percent scaling, including a native resize to an 852×658 viewport and the Ctrl+K palette in light and dark themes. Native screenshots prove composition; targeted CDP input proves document handling, not physical Windows focus or hit testing. Guarded physical typing passed only in the separate composition prototype.

## Debts

- [ ] Verify physical pointer routing and keyboard focus in the integrated shell: address typing/selection/Escape, outside-click dismissal with the intended page action, menu/submenu navigation, modal Tab/Shift+Tab containment and focus restoration, and tab close while an overlay is open. The guarded product input attempt sent nothing because its window was not foreground.
- [ ] Exercise maximize and moving between monitors at Windows 100/125/150/200 percent scaling, including IME composition. Geometry unit tests cover all four scales; native product resize and screenshot evidence cover only 150 percent, not a full product/DPI verdict.
- [ ] Complete the visual/input matrix for remaining overlay families, including share dialogs, tooltips, toasts and focus-edge controls, and theme combinations beyond the palette. The palette passed native screenshot review in both themes; acceptance does not cover every surface matched by the shared selector vocabulary.
