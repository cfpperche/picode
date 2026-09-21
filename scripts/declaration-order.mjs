// Use-before-declaration (TDZ) checker for the web frontend.
//
// The bug class: a block-scoped binding (const/let/class) read earlier in a
// function body than its declaration is a ReferenceError the moment the
// function runs — in a React component that means the render throws and the
// whole app renders blank. It happened on main (4f1a68a7): WebTab's meta-poll
// effect listed `showFullUrl` in its deps array while the `const` sat further
// down the same render body. No JS gate caught it, so this checker exists.
//
// Rule, per function body: flag a READ of a block-scoped binding declared
// LATER in that same body, where the read sits outside any nested function
// (callbacks run after initialization — exempt). Shadowing is honored: a
// nested scope that re-declares the name hides the outer binding for its own
// subtree. Hoisted names (function declarations, var, imports) are never
// flagged. Conservative by design: when unsure, do not flag — false positives
// break CI and get checkers deleted.
//
// Parser: @babel/parser, already in web/node_modules (via @vitejs/plugin-react);
// it parses JSX natively, unlike acorn which would need acorn-jsx on top.
//
// CLI: node scripts/declaration-order.mjs [paths...] — with no arguments it
// scans the frontend trees this repo ships (web/browser/src, web/shared).

import { readdirSync, readFileSync, statSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "..");

// Anchored at web/package.json so the dependency is the one `make web` (npm ci)
// installs, not whatever happens to sit on scripts/' lookup path.
const webRequire = createRequire(join(repoRoot, "web", "package.json"));
let parse;
try {
  parse = webRequire("@babel/parser").parse;
} catch (err) {
  throw new Error(
    "@babel/parser is not resolvable from web/node_modules — run `make web` (or `cd web && npm ci`) first",
    { cause: err },
  );
}

const PARSE_OPTIONS = { sourceType: "unambiguous", plugins: ["jsx"] };

// Node types whose subtree runs after the body completes (or at class
// definition time for exotic slots we deliberately do not chase). References
// inside them are exempt.
const FUNCTION_LIKE = new Set([
  "FunctionDeclaration",
  "FunctionExpression",
  "ArrowFunctionExpression",
  "ObjectMethod",
  "ClassMethod",
  "ClassPrivateMethod",
  "StaticBlock",
]);

// Non-node metadata keys skipped by the generic walker.
const META_KEYS = new Set([
  "loc",
  "start",
  "end",
  "range",
  "leadingComments",
  "trailingComments",
  "innerComments",
  "extra",
]);

function isNode(value) {
  return Boolean(value) && typeof value === "object" && typeof value.type === "string";
}

// Names bound by a destructuring pattern (`const { a, b: [c] } = x` → a, c).
function patternNames(node, out = []) {
  if (!isNode(node)) return out;
  switch (node.type) {
    case "Identifier":
      out.push(node.name);
      break;
    case "ObjectPattern":
      for (const prop of node.properties) {
        patternNames(prop.type === "RestElement" ? prop.argument : prop.value, out);
      }
      break;
    case "ArrayPattern":
      for (const el of node.elements) if (el) patternNames(el, out);
      break;
    case "AssignmentPattern":
      patternNames(node.left, out);
      break;
    case "RestElement":
      patternNames(node.argument, out);
      break;
  }
  return out;
}

function patternHas(pattern, name) {
  return patternNames(pattern).includes(name);
}

// Does this block-like scope bind `name` at its top level? Any declaration
// kind counts: let/const/class shadow from the block top, and var/function
// hoist — in all four cases reads inside the subtree do not see the outer
// binding.
function scopeDeclares(scope, name) {
  const statements =
    scope.type === "SwitchStatement"
      ? scope.cases.flatMap((c) => c.consequent)
      : scope.body;
  for (const stmt of statements) {
    if (stmt.type === "VariableDeclaration") {
      if (stmt.declarations.some((d) => patternHas(d.id, name))) return true;
    } else if (
      (stmt.type === "FunctionDeclaration" || stmt.type === "ClassDeclaration") &&
      stmt.id?.name === name
    ) {
      return true;
    }
  }
  return false;
}

function scopeStatements(scope) {
  return scope.type === "SwitchStatement"
    ? scope.cases.flatMap((c) => c.consequent)
    : scope.body;
}

// Walk a subtree in search of reads of one binding. `binding` marks pattern
// position (the identifier is a target, not a read) so that pattern defaults
// (`const { a = later } = o`) still count as reads.
function walk(node, binding, decl, refs) {
  if (!isNode(node)) return;
  // A loop-head declaration scopes to the entire loop: reads in the head
  // and body resolve to it, never to a later outer binding of the same name.
  const forHead =
    node.type === "ForStatement"
      ? node.init
      : node.type === "ForInStatement" || node.type === "ForOfStatement"
        ? node.left
        : null;
  if (
    forHead?.type === "VariableDeclaration" &&
    forHead.declarations.some((d) => patternHas(d.id, decl.name))
  ) {
    return;
  }
  switch (node.type) {
    case "Identifier":
      if (!binding && node.name === decl.name && node.start < decl.pos) refs.push(node);
      return;

    // Runs later — exempt wholesale.
    case "FunctionDeclaration":
    case "FunctionExpression":
    case "ArrowFunctionExpression":
    case "StaticBlock":
      return;

    // Method boundaries, but a computed key is evaluated right now.
    case "ObjectMethod":
    case "ClassMethod":
    case "ClassPrivateMethod":
      if (node.computed) walk(node.key, false, decl, refs);
      return;

    // Class bodies run later; `extends` is evaluated immediately.
    case "ClassDeclaration":
    case "ClassExpression":
      if (node.superClass) walk(node.superClass, false, decl, refs);
      return;

    case "ClassBody":
      return;

    case "BlockStatement":
    case "SwitchStatement":
      if (scopeDeclares(node, decl.name)) return; // shadowed inside
      for (const stmt of scopeStatements(node)) walk(stmt, false, decl, refs);
      return;

    case "CatchClause":
      if (node.param && patternHas(node.param, decl.name)) return;
      walk(node.body, false, decl, refs);
      return;

    case "VariableDeclaration":
      for (const d of node.declarations) {
        walkPattern(d.id, decl, refs);
        if (d.init) walk(d.init, false, decl, refs);
      }
      return;

    case "AssignmentExpression":
      walkPattern(node.left, decl, refs);
      walk(node.right, false, decl, refs);
      return;

    case "ForInStatement":
    case "ForOfStatement":
      if (node.left.type === "VariableDeclaration") walk(node.left, false, decl, refs);
      else walkPattern(node.left, decl, refs); // `for (x of xs)` — write target
      walk(node.right, false, decl, refs);
      walk(node.body, false, decl, refs);
      return;

    case "MemberExpression":
    case "OptionalMemberExpression":
      walk(node.object, false, decl, refs);
      if (node.computed) walk(node.property, false, decl, refs);
      return;

    // Object keys are not reads; values (including shorthand) are.
    case "ObjectProperty":
      if (node.computed) walk(node.key, false, decl, refs);
      walk(node.value, false, decl, refs);
      return;

    case "WithStatement":
      // Dynamic scope: a read inside the body may resolve to the with
      // object instead of the binding. Do not flag.
      walk(node.object, false, decl, refs);
      return;

    case "LabeledStatement":
      walk(node.body, false, decl, refs);
      return;

    case "BreakStatement":
    case "ContinueStatement":
      return;

    case "UpdateExpression":
      // `x++` reads x — a TDZ read like any other.
      walk(node.argument, false, decl, refs);
      return;

    default: {
      for (const [key, value] of Object.entries(node)) {
        if (META_KEYS.has(key)) continue;
        if (Array.isArray(value)) {
          for (const item of value) walk(item, false, decl, refs);
        } else if (isNode(value)) {
          walk(value, false, decl, refs);
        }
      }
    }
  }
}

function walkPattern(node, decl, refs) {
  if (!isNode(node)) return;
  switch (node.type) {
    case "AssignmentPattern":
      walkPattern(node.left, decl, refs);
      walk(node.right, false, decl, refs); // the default is a read
      return;
    case "ObjectPattern":
      for (const prop of node.properties) {
        if (prop.type === "RestElement") walkPattern(prop.argument, decl, refs);
        else {
          if (prop.computed) walk(prop.key, false, decl, refs);
          walkPattern(prop.value, decl, refs);
        }
      }
      return;
    case "ArrayPattern":
      for (const el of node.elements) if (el) walkPattern(el, decl, refs);
      return;
    case "RestElement":
      walkPattern(node.argument, decl, refs);
      return;
    default:
      walk(node, true, decl, refs); // Identifier targets land here: no read
  }
}

// Block-scoped bindings declared anywhere in a function body, at any block
// depth: {name, pos, block}. Loop-head declarations are skipped — they scope
// to the loop, so reads before the loop name something else.
function collectDecls(body, out) {
  const visit = (node, forHead) => {
    if (!isNode(node)) return;
    switch (node.type) {
      case "FunctionDeclaration":
      case "FunctionExpression":
      case "ArrowFunctionExpression":
      case "StaticBlock":
        return; // its own scan unit
      case "ClassDeclaration":
        if (node.id) out.push({ name: node.id.name, pos: node.start, block: current() });
        return;
      case "ClassExpression":
        return; // its id binds inside the class body only
      case "VariableDeclaration":
        if (!forHead && (node.kind === "let" || node.kind === "const")) {
          for (const d of node.declarations) {
            for (const name of patternNames(d.id)) {
              out.push({ name, pos: d.start, block: current() });
            }
          }
        }
        return;
      case "BlockStatement":
      case "SwitchStatement": {
        push(node);
        for (const stmt of scopeStatements(node)) visit(stmt, false);
        pop();
        return;
      }
      case "ForStatement":
        visit(node.init, true);
        visit(node.test, false);
        visit(node.update, false);
        visit(node.body, false);
        return;
      case "ForInStatement":
      case "ForOfStatement":
        visit(node.left, true);
        visit(node.right, false);
        visit(node.body, false);
        return;
      case "LabeledStatement":
        visit(node.body, false);
        return;
      default: {
        for (const [key, value] of Object.entries(node)) {
          if (META_KEYS.has(key)) continue;
          if (Array.isArray(value)) {
            for (const item of value) visit(item, false);
          } else if (isNode(value)) {
            visit(value, false);
          }
        }
      }
    }
  };

  const stack = [body];
  const current = () => stack[stack.length - 1];
  const push = (scope) => stack.push(scope);
  const pop = () => stack.pop();

  for (const stmt of body.body) visit(stmt, false);
  return out;
}

function* functionBodies(node) {
  if (!isNode(node)) return;
  if (FUNCTION_LIKE.has(node.type)) {
    yield node;
    return; // a nested function belongs to its own unit
  }
  for (const [key, value] of Object.entries(node)) {
    if (META_KEYS.has(key)) continue;
    if (Array.isArray(value)) {
      for (const item of value) yield* functionBodies(item);
    } else if (isNode(value)) {
      yield* functionBodies(value);
    }
  }
}

function functionName(fn) {
  if (fn.id?.name) return fn.id.name;
  if (fn.type === "ClassMethod" || fn.type === "ObjectMethod" || fn.type === "ClassPrivateMethod") {
    const k = fn.key;
    if (k?.type === "Identifier") return k.name;
    if (k?.type === "PrivateName") return "#" + k.name;
  }
  return "(anonymous)";
}

// Scan one parsed file. Returns violations:
// [{name, line, column, pos, function}] sorted by position.
export function scanAst(ast) {
  const violations = [];
  for (const fn of functionBodies(ast.program)) {
    const body = fn.body;
    if (body?.type !== "BlockStatement") continue;
    for (const decl of collectDecls(body, [])) {
      const refs = [];
      for (const stmt of scopeStatements(decl.block)) {
        if (stmt.start >= decl.pos) break; // everything after is initialized
        walk(stmt, false, decl, refs);
      }
      for (const ref of refs) {
        violations.push({
          name: decl.name,
          pos: ref.start,
          line: ref.loc.start.line,
          column: ref.loc.start.column,
          function: functionName(fn),
        });
      }
    }
  }
  return violations.sort((a, b) => a.pos - b.pos);
}

export function scanSource(code, { file = "<input>" } = {}) {
  const ast = parse(code, PARSE_OPTIONS);
  return scanAst(ast).map((v) => ({ ...v, file }));
}

// Returns {violations, skipped} for a list of files or directories.
// A file that does not parse is reported as skipped, never a violation —
// the checker stays conservative even on sources it cannot read.
export function scanPaths(paths) {
  const violations = [];
  const skipped = [];
  for (const file of expandPaths(paths)) {
    let found;
    try {
      found = scanSource(readFileSync(file, "utf8"), { file });
    } catch (err) {
      skipped.push({ file, reason: String(err?.message || err).split("\n")[0] });
      continue;
    }
    violations.push(...found);
  }
  return { violations, skipped };
}

export const DEFAULT_SCAN_ROOTS = ["web/browser/src", "web/shared"]
  .map((p) => join(repoRoot, p));

function expandPaths(paths) {
  const files = [];
  for (const p of paths) {
    const abs = resolve(repoRoot, p);
    const st = statSync(abs, { throwIfNoEntry: false });
    if (!st) continue;
    if (st.isDirectory()) files.push(...expandDir(abs));
    else if (/\.(js|jsx)$/.test(abs)) files.push(abs);
  }
  return files.sort();
}

function expandDir(dir) {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === "node_modules" || entry.name.startsWith(".")) continue;
    const abs = join(dir, entry.name);
    if (entry.isDirectory()) files.push(...expandDir(abs));
    else if (/\.(js|jsx)$/.test(entry.name)) files.push(abs);
  }
  return files;
}

function main(argv) {
  const targets = argv.length > 0 ? argv : DEFAULT_SCAN_ROOTS.map((r) => relative(repoRoot, r));
  const { violations, skipped } = scanPaths(targets);
  for (const s of skipped) console.error(`skipped (unparseable): ${rel(s.file)}: ${s.reason}`);
  for (const v of violations) {
    console.log(
      `${rel(v.file)}:${v.line}:${v.column}: '${v.name}' read before its declaration in ${v.function}() (TDZ — blank-window class)`,
    );
  }
  if (violations.length > 0) {
    console.error(`\n${violations.length} use-before-declaration violation(s)`);
    process.exitCode = 1;
  } else {
    console.log(`no use-before-declaration violations (${targets.length} target(s))`);
  }
}

function rel(file) {
  return relative(repoRoot, file).split(sep).join("/");
}

if (import.meta.url === pathToFileURL(process.argv[1] || "").href) {
  main(process.argv.slice(2));
}
