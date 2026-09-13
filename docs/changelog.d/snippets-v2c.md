### Added

- **Snippets are easier to write.** The studio editor now validates the
  body as you type — a broken `{{placeholder}}` shows a line naming the
  problem and **Save** says why it is off — and everything the grammar
  knows is editable from a table under the body: **Default**,
  **Optional** and **Enum** per placeholder, written back into the text.
  A **Try it** pane gives each placeholder a sample value (enums become a
  picker) and shows the expanded prompt, including what will be asked for
  at run time.
- **Save a prompt straight from the composer.** Select text and a **Save
  as snippet** button appears in the message bar; the same action is in
  the right-click menu as **Save selection as snippet**, so any text you
  select — in the composer or anywhere in PiCode — can become a snippet
  without retyping it in the studio. On the phone it is **Message
  options → Save as snippet**.
- **Import a prompt from another tool.** The snippets list gained
  **Import**: paste a prompt and PiCode finds its slots, offering
  `[BRACKETS]` and `UPPER_CASE` as `{{placeholders}}` — each suggestion
  can be switched off — then opens the editor with the result. Available
  on the phone too.

### Fixed

- Right-clicking a selection **inside a text field** now offers Copy and
  **Save selection as snippet** enabled: the menu reads the field's own
  selection, which the page's selection never saw.
