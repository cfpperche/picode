### Changed

- **Docker, Inbox and tmux now read like every other page.** Their surfaces keep the app's tab strip, head actions and filter, but inside the same page frame a system route uses: a 1240px card, the view's tabs as an underline nav with count chips, the filter in the card toolbar, and the list and detail panes inside the card (side by side, stacking on narrow windows). Empty, blocked and error states are one line plus one action instead of a full-page poster.

### Fixed

- Opening a link to an inbox item that is gone (answered or removed elsewhere) says so — "This item is no longer in the list" with **Back to the list** — instead of `not found` with a *Try again* that could never work.
- The tmux app's Sessions tab shows its session count instead of a `21 · 21 unclaimed` sentence, which no tab badge in the product had room for.
- The split view's empty detail pane says "Pick an item from the list" — on a narrow window the list is above, not to the left.
