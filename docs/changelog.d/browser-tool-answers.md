### Fixed
- **The browser tool now shows what the act verbs answer.** `evaluate` returns
  the value (or `the page threw: …`), `navigate` says where it went, `cdp`
  returns the method's JSON. All three used to be rendered as an accessibility
  tree, so a call that worked arrived as "the page has no accessible content".
