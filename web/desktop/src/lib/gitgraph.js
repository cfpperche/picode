// Column layout for a commit DAG, and the SVG paths that draw it.
//
// The allocator is ported from Git Graph for Visual Studio Code
// (https://github.com/mhutchie/vscode-git-graph), file web/graph.ts:
//   Copyright (c) 2019-present, Michael Hutchison. MIT licensed.
// Method names are kept (determinePath, getAvailableColour,
// registerUnavailablePoint) so this stays diffable against the original.
//
// What is dropped: the original's isCommitted/numUncommitted plumbing. Its
// "Uncommitted Changes" row came back in ADR-0038, but as a wrapper instead:
// layoutUncommitted prepends a pseudo-commit whose parent is HEAD, runs the
// ordinary layout, and splits the pseudo's trail out for dashed drawing.
//
// The layout is greedy and single-pass: walk the commits top to bottom, and on
// each row a branch claims the leftmost column still free. There is no global
// optimiser and no crossing minimisation.

const NULL_VERTEX_ID = -1;

class Branch {
  constructor(colour) {
    this.colour = colour;
    this.end = 0;
    this.lines = [];
  }

  addLine(p1, p2, lockedFirst) {
    this.lines.push({ p1, p2, lockedFirst });
  }
}

class Vertex {
  constructor(id) {
    this.id = id;
    this.x = 0;
    this.nextX = 0;
    this.parents = [];
    this.children = [];
    this.nextParent = 0;
    this.onBranch = null;
    this.connections = [];
  }

  addParent(vertex) {
    this.parents.push(vertex);
  }

  addChild(vertex) {
    this.children.push(vertex);
  }

  getNextParent() {
    return this.nextParent < this.parents.length ? this.parents[this.nextParent] : null;
  }

  registerParentProcessed() {
    this.nextParent++;
  }

  isMerge() {
    return this.parents.length > 1;
  }

  addToBranch(branch, x) {
    if (this.onBranch === null) {
      this.onBranch = branch;
      this.x = x;
    }
  }

  isNotOnBranch() {
    return this.onBranch === null;
  }

  getBranch() {
    return this.onBranch;
  }

  getPoint() {
    return { x: this.x, y: this.id };
  }

  getNextPoint() {
    return { x: this.nextX, y: this.id };
  }

  // The column already reserved on this row for a line heading to `vertex` on
  // `onBranch`, so a merge rejoins the column its parent is already using
  // instead of opening a new one.
  getPointConnectingTo(vertex, onBranch) {
    for (let i = 0; i < this.connections.length; i++) {
      const c = this.connections[i];
      if (c && c.connectsTo === vertex && c.onBranch === onBranch) return { x: i, y: this.id };
    }
    return null;
  }

  // Claiming a column is the whole allocator: a column is taken only if it is
  // the next free one, and taking it moves the frontier right by one.
  registerUnavailablePoint(x, connectsToVertex, onBranch) {
    if (x === this.nextX) {
      this.nextX = x + 1;
      this.connections[x] = { connectsTo: connectsToVertex, onBranch };
    }
  }

  getColour() {
    return this.onBranch !== null ? this.onBranch.colour : 0;
  }
}

class Graph {
  constructor() {
    this.vertices = [];
    this.branches = [];
    this.availableColours = [];
  }

  load(commits, onlyFollowFirstParent) {
    const lookup = Object.create(null);
    commits.forEach((c, i) => {
      if (c && typeof c.hash === "string") lookup[c.hash] = i;
    });

    this.vertices = commits.map((_, i) => new Vertex(i));
    const nullVertex = new Vertex(NULL_VERTEX_ID);

    for (let i = 0; i < commits.length; i++) {
      const parents = commits[i].parents || [];
      for (let j = 0; j < parents.length; j++) {
        const at = lookup[parents[j]];
        if (typeof at === "number") {
          this.vertices[i].addParent(this.vertices[at]);
          this.vertices[at].addChild(this.vertices[i]);
        } else if (!onlyFollowFirstParent || j === 0) {
          // The parent is outside the loaded window: the line runs off the
          // bottom rather than stopping short.
          this.vertices[i].addParent(nullVertex);
        }
      }
    }

    let i = 0;
    while (i < this.vertices.length) {
      if (this.vertices[i].getNextParent() !== null || this.vertices[i].isNotOnBranch()) {
        this.determinePath(i);
      } else {
        i++;
      }
    }
  }

  determinePath(startAt) {
    let i = startAt;
    let vertex = this.vertices[i];
    let parentVertex = this.vertices[i].getNextParent();
    let lastPoint = vertex.isNotOnBranch() ? vertex.getNextPoint() : vertex.getPoint();
    let curVertex;
    let curPoint;

    if (
      parentVertex !== null &&
      parentVertex.id !== NULL_VERTEX_ID &&
      vertex.isMerge() &&
      !vertex.isNotOnBranch() &&
      !parentVertex.isNotOnBranch()
    ) {
      // A merge between two commits that already sit on branches: follow the
      // parent's branch down until the point that connects to it is found.
      let foundPointToParent = false;
      const parentBranch = parentVertex.getBranch();
      for (i = startAt + 1; i < this.vertices.length; i++) {
        curVertex = this.vertices[i];
        curPoint = curVertex.getPointConnectingTo(parentVertex, parentBranch);
        if (curPoint !== null) {
          foundPointToParent = true;
        } else {
          curPoint = curVertex.getNextPoint();
        }
        parentBranch.addLine(
          lastPoint,
          curPoint,
          !foundPointToParent && curVertex !== parentVertex ? lastPoint.x < curPoint.x : true,
        );
        curVertex.registerUnavailablePoint(curPoint.x, parentVertex, parentBranch);
        lastPoint = curPoint;

        if (foundPointToParent) {
          vertex.registerParentProcessed();
          break;
        }
      }
    } else {
      const branch = new Branch(this.getAvailableColour(startAt));
      vertex.addToBranch(branch, lastPoint.x);
      vertex.registerUnavailablePoint(lastPoint.x, vertex, branch);
      for (i = startAt + 1; i < this.vertices.length; i++) {
        curVertex = this.vertices[i];
        curPoint =
          parentVertex === curVertex && !parentVertex.isNotOnBranch()
            ? curVertex.getPoint()
            : curVertex.getNextPoint();
        branch.addLine(lastPoint, curPoint, lastPoint.x < curPoint.x);
        curVertex.registerUnavailablePoint(curPoint.x, parentVertex, branch);
        lastPoint = curPoint;

        if (parentVertex === curVertex) {
          vertex.registerParentProcessed();
          const parentVertexOnBranch = !parentVertex.isNotOnBranch();
          parentVertex.addToBranch(branch, curPoint.x);
          vertex = parentVertex;
          parentVertex = vertex.getNextParent();
          if (parentVertex === null || parentVertexOnBranch) break;
        }
      }
      if (i === this.vertices.length && parentVertex !== null && parentVertex.id === NULL_VERTEX_ID) {
        vertex.registerParentProcessed();
      }
      branch.end = i;
      this.branches.push(branch);
      this.availableColours[branch.colour] = i;
    }
  }

  // A colour is reusable as soon as the branch that held it ended above this
  // row. availableColours[i] is that end row, which is why long histories do
  // not exhaust the palette.
  getAvailableColour(startAt) {
    for (let i = 0; i < this.availableColours.length; i++) {
      if (startAt > this.availableColours[i]) return i;
    }
    this.availableColours.push(0);
    return this.availableColours.length - 1;
  }
}

// layout places every commit in a column and returns the branch lines that
// join them. Coordinates are grid cells: x is a column, y is a row index.
export function layout(commits, { onlyFollowFirstParent = false } = {}) {
  const list = Array.isArray(commits) ? commits : [];
  if (list.length === 0) return { columns: 0, vertices: [], branches: [] };

  const graph = new Graph();
  graph.load(list, onlyFollowFirstParent);

  let columns = 0;
  const vertices = graph.vertices.map((v) => {
    const point = v.getPoint();
    if (point.x + 1 > columns) columns = point.x + 1;
    return { x: point.x, colour: v.getColour() };
  });
  for (const branch of graph.branches) {
    for (const line of branch.lines) {
      columns = Math.max(columns, line.p1.x + 1, line.p2.x + 1);
    }
  }

  return {
    columns,
    vertices,
    branches: graph.branches.map((b) => ({ colour: b.colour, lines: b.lines })),
  };
}

export const GRID = { x: 14, y: 26, offsetX: 12, offsetY: 13 };

// The hash of the pseudo-commit standing in for the dirty working tree. "*" can
// never collide with an object name, and reads as "changes" on its own.
export const UNCOMMITTED = "*";

// layoutUncommitted lays out the history with an "Uncommitted Changes" row on
// top: a pseudo-commit whose only parent is HEAD goes through the ordinary
// allocator — exactly how the original handles it — and its trail down to the
// HEAD row comes back separately so the caller can draw it dashed. When HEAD
// is missing or outside the loaded window there is no row to anchor the trail,
// so the plain layout comes back and rows === commits.
export function layoutUncommitted(commits, head) {
  const list = commits || [];
  const headAt = head ? list.findIndex((c) => c.hash === head) : -1;
  if (headAt < 0) {
    return { placed: layout(list), rows: list, dashed: null };
  }
  const rows = [{ hash: UNCOMMITTED, parents: [head] }, ...list];
  const placed = layout(rows);
  const headRow = headAt + 1;
  // The pseudo is vertex 0, so the first branch created starts at it; its
  // lines down to the HEAD row are the trail. Whatever the branch draws below
  // HEAD is real history and stays solid.
  let dashed = null;
  const first = placed.branches[0];
  if (first) {
    dashed = { colour: first.colour, lines: first.lines.filter((l) => l.p2.y <= headRow) };
    first.lines = first.lines.filter((l) => l.p2.y > headRow);
  }
  return { placed, rows, dashed };
}

// simplifyLines merges consecutive collinear vertical segments. A branch that
// runs straight down 200 rows becomes one segment, not 200 — this is what
// keeps the emitted path short on a long history.
export function simplifyLines(lines) {
  const out = lines.map((l) => ({
    p1: { x: l.p1.x, y: l.p1.y },
    p2: { x: l.p2.x, y: l.p2.y },
    lockedFirst: l.lockedFirst,
  }));
  let i = 0;
  while (i < out.length - 1) {
    const line = out[i];
    const next = out[i + 1];
    if (
      line.p1.x === line.p2.x &&
      line.p2.x === next.p1.x &&
      next.p1.x === next.p2.x &&
      line.p2.y === next.p1.y
    ) {
      line.p2.y = next.p2.y;
      out.splice(i + 1, 1);
    } else {
      i++;
    }
  }
  return out;
}

// branchPath turns grid lines into one SVG path string. A sideways move is a
// cubic when rounded, two straight segments when angular — the same two
// shapes Git Graph offers, and the only difference between its styles.
export function branchPath(lines, { grid = GRID, style = "rounded", expandAt = -1, expandY = 0 } = {}) {
  const simplified = simplifyLines(lines);
  const d = grid.y * (style === "angular" ? 0.38 : 0.8);
  let path = "";
  let prev = null;

  for (const line of simplified) {
    const x1 = line.p1.x * grid.x + grid.offsetX;
    const x2 = line.p2.x * grid.x + grid.offsetX;
    let y1 = line.p1.y * grid.y + grid.offsetY;
    let y2 = line.p2.y * grid.y + grid.offsetY;
    if (expandAt > -1) {
      if (line.p1.y > expandAt) {
        y1 += expandY;
        y2 += expandY;
      } else if (line.p2.y > expandAt) {
        y2 += expandY;
      }
    }

    if (prev === null || x1 !== prev.x || y1 !== prev.y) {
      path += "M" + x1.toFixed(0) + "," + y1.toFixed(1);
    }
    if (x1 === x2) {
      path += "L" + x2.toFixed(0) + "," + y2.toFixed(1);
    } else if (style === "angular") {
      path += line.lockedFirst
        ? "L" + x2.toFixed(0) + "," + (y2 - d).toFixed(1)
        : "L" + x1.toFixed(0) + "," + (y1 + d).toFixed(1);
      path += "L" + x2.toFixed(0) + "," + y2.toFixed(1);
    } else {
      path +=
        "C" + x1.toFixed(0) + "," + (y1 + d).toFixed(1) +
        " " + x2.toFixed(0) + "," + (y2 - d).toFixed(1) +
        " " + x2.toFixed(0) + "," + y2.toFixed(1);
    }
    prev = { x: x2, y: y2 };
  }
  return path;
}

// Branch colours. Indices come from the allocator, which reuses them, so the
// list only needs to be long enough that adjacent branches differ.
export const COLOURS = [
  "#7aa2f7", "#9ece6a", "#e0af68", "#bb9af7",
  "#7dcfff", "#f7768e", "#73daca", "#ff9e64",
];

export function colourAt(i) {
  return COLOURS[((i % COLOURS.length) + COLOURS.length) % COLOURS.length];
}

// repoNameFromKey mirrors the server's repoName: the key is a common dir, so
// the repository is the directory holding it (or the bare repo itself).
export function repoNameFromKey(key) {
  const clean = String(key || "").replace(/\/+$/, "");
  if (!clean) return "";
  const parts = clean.split("/");
  const base = parts.pop() || "";
  if (base === ".git") return parts.pop() || "";
  return base.replace(/\.git$/, "");
}
