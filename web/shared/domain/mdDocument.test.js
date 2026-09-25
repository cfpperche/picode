import assert from "node:assert/strict";
import { test } from "node:test";
import {
  anchorTargets, createSlugger, rehypeDocHeadings, rehypeGithubAlerts,
  resolveDocImage, resolveDocLink, splitFrontmatter,
} from "./mdDocument.js";

test("splitFrontmatter", () => {
  const cases = [
    { name: "none", in: "# Title\n", front: null, rows: null, body: "# Title\n" },
    { name: "simple rows", in: "---\ntitle: \"Hi\"\ndraft: false\n---\n# Body", front: "title: \"Hi\"\ndraft: false", rows: [["title", "Hi"], ["draft", "false"]], body: "# Body" },
    { name: "nested stays raw", in: "---\ntags:\n  - a\n---\nx", front: "tags:\n  - a", rows: null, body: "x" },
    { name: "crlf and dots", in: "---\r\na: 1\r\n...\r\nx", front: "a: 1", rows: [["a", "1"]], body: "x" },
    { name: "empty block", in: "---\n---\nx", front: "", rows: [], body: "x" },
    { name: "unclosed is body", in: "---\na: 1\n", front: null, rows: null, body: "---\na: 1\n" },
  ];
  for (const c of cases) {
    const got = splitFrontmatter(c.in);
    assert.equal(got.front, c.front, c.name);
    assert.deepEqual(got.rows, c.rows, c.name);
    assert.equal(got.body, c.body, c.name);
  }
});

test("createSlugger follows github-slugger", () => {
  const slug = createSlugger();
  assert.equal(slug("Closing a session (the rite, in one command)"), "closing-a-session-the-rite-in-one-command");
  assert.equal(slug("What's new?"), "whats-new");
  assert.equal(slug("Ação rápida"), "ação-rápida");
  assert.equal(slug("snake_case & more"), "snake_case--more");
  assert.equal(slug("Intro"), "intro");
  assert.equal(slug("Intro"), "intro-1");
  assert.equal(slug("Intro"), "intro-2");
  assert.equal(slug("Intro-1"), "intro-1-1");
});

const h = (tagName, children, properties = {}) => ({ type: "element", tagName, properties, children });
const t = (value) => ({ type: "text", value });

test("rehypeDocHeadings prefixes and dedupes ids", () => {
  const tree = { type: "root", children: [h("h1", [t("Intro")]), h("h2", [t("Intro")]), h("p", [t("x")])] };
  rehypeDocHeadings()(tree);
  assert.equal(tree.children[0].properties.id, "user-content-intro");
  assert.equal(tree.children[1].properties.id, "user-content-intro-1");
  assert.equal(tree.children[2].properties.id, undefined);
});

test("rehypeGithubAlerts", () => {
  const cases = [
    {
      name: "marker then text",
      in: h("blockquote", [t("\n"), h("p", [t("[!NOTE]\nRead this.")]), t("\n")]),
      tag: "div", cls: ["md-alert", "md-alert-note"], title: "Note", firstBody: "Read this.",
    },
    {
      name: "marker alone drops its paragraph",
      in: h("blockquote", [h("p", [t("[!warning]")]), h("p", [t("Careful.")])]),
      tag: "div", cls: ["md-alert", "md-alert-warning"], title: "Warning", firstBody: "Careful.",
    },
    {
      name: "unknown kind stays a quote",
      in: h("blockquote", [h("p", [t("[!FOO]\nx")])]),
      tag: "blockquote",
    },
    {
      name: "marker mid-line stays a quote",
      in: h("blockquote", [h("p", [t("[!NOTE] inline")])]),
      tag: "blockquote",
    },
  ];
  for (const c of cases) {
    const tree = { type: "root", children: [c.in] };
    rehypeGithubAlerts()(tree);
    const node = tree.children[0];
    assert.equal(node.tagName, c.tag, c.name);
    if (c.tag !== "div") continue;
    assert.deepEqual(node.properties.className, c.cls, c.name);
    const [title, ...rest] = node.children.filter((n) => n.type === "element");
    assert.equal(title.children[0].value, c.title, c.name);
    assert.equal(rest[0].children.map((n) => n.value).join(""), c.firstBody, c.name);
  }
});

test("rehypeGithubAlerts ignores nested quotes", () => {
  const inner = h("blockquote", [h("p", [t("[!TIP]\nx")])]);
  const tree = { type: "root", children: [h("blockquote", [inner])] };
  rehypeGithubAlerts()(tree);
  assert.equal(inner.tagName, "blockquote");
});

test("resolveDocLink", () => {
  const from = "docs/guide/intro.md";
  const cases = [
    ["#setup", { kind: "anchor", id: "setup" }],
    ["https://x.dev/a", { kind: "external", href: "https://x.dev/a" }],
    ["mailto:a@b.c", { kind: "external", href: "mailto:a@b.c" }],
    ["//cdn.x/a.png", { kind: "external", href: "https://cdn.x/a.png" }],
    ["javascript:alert(1)", { kind: "none" }],
    ["file:///etc/passwd", { kind: "none" }],
    ["next.md", { kind: "file", path: "docs/guide/next.md", id: "" }],
    ["./next.md#part", { kind: "file", path: "docs/guide/next.md", id: "part" }],
    ["../../README.md", { kind: "file", path: "README.md", id: "" }],
    ["../../../etc/passwd", { kind: "none" }],
    ["/LICENSE", { kind: "file", path: "LICENSE", id: "" }],
    ["my%20file.md?plain=1", { kind: "file", path: "docs/guide/my file.md", id: "" }],
    ["sub/", { kind: "none" }],
    ["", { kind: "none" }],
  ];
  for (const [href, want] of cases) assert.deepEqual(resolveDocLink(from, href), want, href);
  assert.deepEqual(resolveDocLink("README.md", "docs/a.svg"), { kind: "file", path: "docs/a.svg", id: "" });
});

test("resolveDocImage", () => {
  assert.equal(resolveDocImage("a.md", "data:image/png;base64,x").kind, "external");
  assert.equal(resolveDocImage("a.md", "#x").kind, "none");
  assert.equal(resolveDocImage("a.md", "mailto:a@b.c").kind, "none");
  assert.deepEqual(resolveDocImage("docs/a.md", "img/b.png"), { kind: "file", path: "docs/img/b.png", id: "" });
});

test("anchorTargets", () => {
  assert.deepEqual(anchorTargets("#fn-1"), ["user-content-fn-1", "fn-1"]);
  assert.deepEqual(anchorTargets("user-content-a"), ["user-content-a"]);
  assert.deepEqual(anchorTargets("a%C3%A7%C3%A3o"), ["user-content-ação", "ação"]);
  assert.deepEqual(anchorTargets(""), []);
});
