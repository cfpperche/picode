import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

// A fill that carries text (primary buttons, send, a pressed tool) must pass
// WCAG AA with the text on it, in both themes. --accent is text and borders
// on the dark ground and is light on purpose; white on it was 3.02:1, which
// is why fills read --accent-fill instead.
const root = new URL("..", import.meta.url).pathname;
const theme = readFileSync(join(root, "shared/tokens/theme.css"), "utf8");

function block(selector) {
  const i = theme.indexOf(selector + " {");
  assert.ok(i >= 0, "missing " + selector);
  return theme.slice(i, theme.indexOf("\n}", i));
}

function token(css, name, fallback) {
  const m = new RegExp("--" + name + ":\\s*([^;]+);").exec(css);
  const v = m ? m[1].trim() : fallback;
  const ref = /^var\(--([a-z-]+)\)$/.exec(v || "");
  return ref ? token(css, ref[1], fallback) : v;
}

function luminance(hex) {
  const c = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255)
    .map((x) => (x <= 0.03928 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4));
  return 0.2126 * c[0] + 0.7152 * c[1] + 0.0722 * c[2];
}

function contrast(a, b) {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

test("text on the accent fill passes AA in both themes", () => {
  const dark = block(":root");
  const light = block(':root[data-theme="light"]');
  for (const [name, css] of [["dark", dark], ["light", light]]) {
    const fill = token(css, "accent-fill", token(dark, "accent-fill"));
    const on = token(css, "on-accent", token(dark, "on-accent"));
    assert.match(fill, /^#[0-9a-f]{6}$/i, name + " fill is a hex colour");
    const ratio = contrast(fill, on === "#fff" ? "#ffffff" : on);
    assert.ok(ratio >= 4.5, `${name}: ${on} on ${fill} is ${ratio.toFixed(2)}:1`);
  }
});

test("no stylesheet puts white text on the bare accent", () => {
  const dirs = ["browser/src/styles", "mobile/src/styles"];
  const offenders = [];
  for (const dir of dirs) {
    for (const f of readdirSync(join(root, dir)).filter((n) => n.endsWith(".css"))) {
      const css = readFileSync(join(root, dir, f), "utf8");
      for (const rule of css.split("}")) {
        if (/background[a-z-]*:\s*var\(--accent\)\s*[;}]/.test(rule + ";") && /(^|[^-])color:\s*(#fff\b|#ffffff\b|white\b)/i.test(rule)) {
          offenders.push(f + ": " + rule.trim().split("{")[0].trim());
        }
      }
    }
  }
  assert.deepEqual(offenders, [], "use --accent-fill with --on-accent");
});
