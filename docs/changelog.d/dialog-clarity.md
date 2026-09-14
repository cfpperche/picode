### Changed
- **The custom endpoint dialog reads field by field.** Name, Base URL, API key
  and Model ids carry a label above the control and one line of help under it,
  with a clear gap between one field and the next — the placeholders were the
  only label before, so a filled field could not be told from an empty one.
  The helper prose that read as a lecture (a file path, a protocol note) is
  gone; what stayed says what the control does.
- **Limits belong to the model, not to the list.** Context window and max
  output are one row per model id (Advanced → Model limits), so Load models
  fills each model's own numbers — a gateway whose models disagree is no
  longer refused with a blank field and an explanation. A number typed by hand
  survives a load; a row follows its id into and out of the list.
- **Advanced is a section, not a wall.** The disclosure holds grouped sections
  — Request compatibility, Thinking, Model limits — each with a legend and a
  hairline, so it reads as structure once open and stays one line while
  closed.
- **Load models and Verify key report inside Model ids**, the field that owns
  them, so a refusal line ("The endpoint refused the key (401): invalid api
  key provided. Check the API key and try again.") sits next to the action that
  caused it and never pushes the alternative route below the fold.