### Changed

- **The Development flow guide's "When the flow bends" cards are checked against the repo scripts that print them.** `make docs-check` fails when a card's quoted wording and the message in its script no longer match, and names the card and its line in the guide. Example values, and the Vale and dead-link cards, are checked only as far as this repository's own files go; the guide says which parts are not checked.
