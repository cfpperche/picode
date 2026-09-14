# 2026-09-13 — snippets-docs: the v2 guide, and the braces the page was showing

Shipped `docs-site/guide/snippets.md` for v2: the slug line while you type
(free / in use with Save off, own address never a clash), the placeholder
table (Default, Optional, Enum → dropdown at send time, reserved names from
the target), drafts surviving a closed tab, and a "Start from something" table
(starters, save-a-selection, import, duplicate, phone "Save as snippet").

Fixed a user-facing defect on the **published** page: every placeholder was
written `&#123;&#123;name&#125;&#125;` inside a code span, which markdown escapes
again — the live site showed the entity text, not `{{name}}`. Replacing them
with raw braces failed the build (VitePress compiles markdown as a Vue
template, so `{{…}}` is an interpolation); the fix is
`<code v-pre>{{name}}</code>`, which renders literal braces and builds.
`docs-site` had 92 such entities on this page and none elsewhere.

Verified: `make docs` green; the built HTML has zero `&#123;` left, and the
rendered page was read at 1280×900 — the v1 tables and the new "The
placeholder table" / "Create one" sections show real `{{name}}`, `{{cwd}}`,
`{{{{` and `/snip:review-pr`. Two screenshots read.
Publishing: the Pages workflow deploys on push to `main` touching
`docs-site/**`; nothing else to run.
Merge: fast-forward ready.
