// Pi's Settings rows, as data (ADR-0163 shape, applied to Pi 2026-09-20).
//
// The eight guest CLIs declare their rows in a table and the pane renders
// whatever the table says; Pi's six original rows were hand-written JSX, so a
// setting pi gained was invisible here until someone edited a component. That
// asymmetry is what the owner saw: forty keys persisted, eight shown.
//
// Three of Pi's rows are not scalars — the model defaults are three coupled
// selects fed by a live catalog, scoped models is a free list, tools is a grid
// over a fixed set. They are declared here with their own `kind` rather than
// pretended into scalars, so the table still answers "what rows exist, in what
// order, in which group" and the renderer owns only how each kind is drawn.
//
// `keys` is what the row reads and resets; `key` is shorthand for one of them.
// `defaultOn` / `fallback` are what pi itself does when nobody sets the key,
// read out of the installed bundle on 2026-09-20:
//
//	compaction.enabled  `?? true`                 → on
//	steeringMode        `|| "one-at-a-time"`
//	followUpMode        `|| "one-at-a-time"`
//	defaultProjectTrust `"always"|"never"` else "ask"
//	quietStartup        `?? false`                → off
//	hideThinkingBlock   no default; absent is off

export const PI_MODES = ["one-at-a-time", "all"];

// pi's built-in tool set, in the order the grid draws it.
export const PI_TOOLS = ["read", "bash", "powershell", "edit", "write", "grep", "find", "ls"];

export const PI_GROUP_SESSION = "Session";
export const PI_GROUP_MODEL = "Model";
export const PI_GROUP_APPROVALS = "Approvals";
export const PI_GROUP_INTERFACE = "Interface";

// `fields` is what a row reads: the key, its type, and `unset` — the value in
// force when no layer sets it. `unset` is the whole reason resolveLayer can be
// derived from this table instead of repeating it, which is how the Theme row
// shipped rendering empty (2026-09-20).
const str = (key, unset = "") => ({ key, type: "string", unset });
const bool = (key, unset) => ({ key, type: "bool", unset });
const list = (key, unset) => ({ key, type: "list", unset });

export const PI_ROWS = [
  { fields: [bool("compactionEnabled", true)], label: "Auto-compact", kind: "bool", group: PI_GROUP_SESSION, defaultOn: true, fallback: "On" },
  { fields: [str("steeringMode", "one-at-a-time")], label: "Steering", kind: "select", group: PI_GROUP_SESSION, options: PI_MODES, fallback: "one-at-a-time" },
  { fields: [str("followUpMode", "one-at-a-time")], label: "Follow-up", kind: "select", group: PI_GROUP_SESSION, options: PI_MODES, fallback: "one-at-a-time" },

  { fields: [str("defaultProvider"), str("defaultModel"), str("defaultThinkingLevel")], label: "Defaults", kind: "model", group: PI_GROUP_MODEL, stack: true },
  { fields: [list("enabledModels", [])], label: "Scoped models", kind: "patterns", group: PI_GROUP_MODEL, stack: true, anchor: "scoped-models" },

  { fields: [list("defaultTools", PI_TOOLS)], label: "Tools", kind: "tools", group: PI_GROUP_APPROVALS, stack: true },
  // pi keeps the rest in its machine file and nowhere else, so they are only
  // offered on that layer: writing them to a workspace file would write a key
  // pi never reads there.
  { fields: [str("defaultProjectTrust", "ask")], label: "New folders", kind: "select", group: PI_GROUP_APPROVALS, machine: true, fallback: "ask", options: ["ask", "always", "never"], optionLabels: { ask: "Ask each time", always: "Trust automatically", never: "Never trust" } },
  { fields: [str("shellPath")], label: "Shell", kind: "text", group: PI_GROUP_APPROVALS, machine: true, fallback: "Pi default", help: "The shell pi runs commands in." },

  { fields: [str("theme")], label: "Theme", kind: "text", group: PI_GROUP_INTERFACE, machine: true, fallback: "Pi default", help: "A built-in name (dark, light) or one of your own themes." },
  { fields: [bool("hideThinkingBlock", false)], label: "Hide thinking", kind: "bool", group: PI_GROUP_INTERFACE, machine: true, defaultOn: false, fallback: "Off" },
  { fields: [bool("quietStartup", false)], label: "Quiet startup", kind: "bool", group: PI_GROUP_INTERFACE, machine: true, defaultOn: false, fallback: "Off" },
];

// piFields flattens the table to the keys the pane reads and writes.
export function piFields() {
  return PI_ROWS.flatMap((row) => row.fields);
}

// rowKeys is what a row reads, marks as set and hands back.
export function rowKeys(row) {
  return row.fields.map((f) => f.key);
}

// rowKey is the single key a scalar row writes.
export function rowKey(row) {
  return row.fields[0].key;
}

// piRowsFor returns the rows a layer may edit, grouped in declaration order.
// The machine layer sees everything; a workspace or agent layer sees only what
// pi reads from its own file.
export function piRowsFor(layer) {
  const machine = layer === "global";
  const rows = PI_ROWS.filter((r) => machine || !r.machine);
  const out = [];
  for (const row of rows) {
    let group = out.find((g) => g.name === row.group);
    if (!group) {
      group = { name: row.group, rows: [] };
      out.push(group);
    }
    group.rows.push(row);
  }
  return out;
}

// piRowState answers the three things every row's label has to say: does this
// layer set it, where does the value come from otherwise, and what is the
// value in force.
export function piRowState(row, values, own, parentLabel) {
  const has = (own && own.has) || {};
  const keys = rowKeys(row);
  const setHere = keys.some((k) => !!has[k]);
  return {
    setHere,
    source: setHere ? "Set here" : parentLabel,
    // Only the keys this layer actually set are handed back, so a row that
    // overrides one of its three fields resets only that one.
    resetKeys: keys.filter((k) => !!has[k]),
    value: values ? values[rowKey(row)] : undefined,
  };
}
