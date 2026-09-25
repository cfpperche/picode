import { useDeferredValue, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import Markdown from "react-markdown";
import { Info, Lightbulb, Link as LinkIcon, MessageSquareWarning, OctagonAlert, TriangleAlert } from "lucide-react";
import "katex/dist/katex.min.css";
import { api } from "@picode/shared/client/api.js";
import { svgDataUrl } from "@picode/shared/domain/filePreview.js";
import { docPipeline, withSourceLines } from "@picode/shared/domain/mdPipeline.js";
import { DOC_ID_PREFIX, anchorTargets, resolveDocImage, resolveDocLink, splitFrontmatter } from "@picode/shared/domain/mdDocument.js";
import { safeImgSrc } from "@picode/shared/domain/mdSafe.js";
import { CopyBtn, mdComponents } from "./SourceBlock.jsx";

const ALERT_ICON = { note: Info, tip: Lightbulb, important: MessageSquareWarning, warning: TriangleAlert, caution: OctagonAlert };
const HEADINGS = ["h1", "h2", "h3", "h4", "h5", "h6"];
// An outline earns its column only when the document has sections to jump
// between; a short note keeps the whole width.
const OUTLINE_MIN = 3;
// Room kept above a heading scrolled to the top of the pane.
const JUMP_GAP = 16;

// MarkdownDoc renders a markdown file the way GitHub does (see
// docs/architecture/file-preview.md): GitHub's HTML allow-list, alerts, math,
// diagrams, heading anchors, an outline, and links and images that resolve
// against the file's own folder. `assetUrl(path, resource)` turns a repository
// path into a file-API URL; `onOpenPath(path)` opens another file of the tree.
// `sourceLines` stamps every block with its file line (`data-line`) for the
// Split view's scroll sync.
export default function MarkdownDoc({ text, path = "", assetUrl, onOpenPath, sourceLines = false }) {
  const rootRef = useRef(null);
  // Typing in the Split view re-renders on every key; the preview may lag a
  // frame behind the editor rather than make the editor wait for it.
  const shown = useDeferredValue(text);
  const { front, rows, body, bodyLine } = useMemo(() => splitFrontmatter(shown), [shown]);
  const pipeline = useMemo(() => (sourceLines ? withSourceLines(bodyLine - 1) : docPipeline), [sourceLines, bodyLine]);
  const [outline, setOutline] = useState([]);
  const [active, setActive] = useState("");
  // The outline entry last clicked: a section near the end can never reach
  // the top of the pane, so at the bottom of the scroll the click decides.
  const pickedRef = useRef("");
  // Callers pass fresh closures on every render; the renderers read them
  // through refs so a parent render never remounts the headings (the outline
  // and its scroll tracking hold on to those nodes).
  const assetRef = useRef(assetUrl);
  const openRef = useRef(onOpenPath);
  assetRef.current = assetUrl;
  openRef.current = onOpenPath;
  const canOpen = !!onOpenPath;

  const components = useMemo(() => {
    function jump(e, frag) {
      e.preventDefault();
      const root = rootRef.current;
      if (!root) return;
      for (const id of anchorTargets(frag)) {
        const el = root.querySelector(`[id="${CSS.escape(id)}"]`);
        if (el) { scrollToHeading(root, el); return; }
      }
    }
    const base = mdComponents({ CopyBtn, onRun: null });
    const heading = (Tag) => function Heading({ node: _node, id, children, ...rest }) {
      const frag = id ? id.slice(DOC_ID_PREFIX.length) : "";
      return (
        <Tag id={id} {...rest}>
          {children}
          {frag ? (
            <a className="md-anchor" href={"#" + frag} aria-label="Link to this section" onClick={(e) => jump(e, frag)}>
              <LinkIcon size={14} aria-hidden="true" />
            </a>
          ) : null}
        </Tag>
      );
    };
    return {
      ...base,
      // The chat's renderer drops <pre> (SourceBlock is the block); a
      // document keeps its source line on a wrapper instead.
      pre({ node: _node, children, ...rest }) {
        return rest["data-line"] ? <div className="md-src-block" data-line={rest["data-line"]}>{children}</div> : <>{children}</>;
      },
      ...Object.fromEntries(HEADINGS.map((h) => [h, heading(h)])),
      p({ node: _node, className, children, ...rest }) {
        if (className === "md-alert-title") {
          const Icon = ALERT_ICON[rest["data-alert"]] || Info;
          return <p className={className} {...rest}><Icon size={15} aria-hidden="true" />{children}</p>;
        }
        return <p className={className} {...rest}>{children}</p>;
      },
      img({ node: _node, src, alt, width, height, title, align }) {
        const at = resolveDocImage(path, src);
        const props = { className: "md-doc-img", alt: alt || "", width, height, title, align, loading: "lazy" };
        if (at.kind === "file" && /\.svg$/i.test(at.path)) return <RepoSvg path={at.path} assetRef={assetRef} {...props} />;
        const url = at.kind === "external" ? safeImgSrc(at.href) : at.kind === "file" && assetRef.current ? assetRef.current(at.path, "blob") : "";
        if (!url) return alt ? <span className="md-img-missing">{alt}</span> : null;
        return <img src={url} {...props} />;
      },
      a({ node: _node, href, children, className, ...rest }) {
        const to = resolveDocLink(path, href);
        const props = { className, ...rest };
        if (to.kind === "anchor") return <a {...props} href={"#" + to.id} onClick={(e) => jump(e, to.id)}>{children}</a>;
        if (to.kind === "external") return <a {...props} href={to.href} target="_blank" rel="noreferrer noopener">{children}</a>;
        if (to.kind === "file" && canOpen) {
          return <a {...props} href={"#" + to.path} title={to.path} onClick={(e) => { e.preventDefault(); openRef.current?.(to.path); }}>{children}</a>;
        }
        return <span className="md-link-inert" title={to.kind === "file" ? to.path : undefined}>{children}</span>;
      },
    };
  }, [path, canOpen]);

  // The outline is read back from the rendered headings, so it can never
  // disagree with the ids the anchors point at.
  useLayoutEffect(() => {
    const root = rootRef.current;
    if (!root) return;
    const next = [...root.querySelectorAll(".md-doc-body :is(h1, h2, h3)[id]")].map((el) => ({
      id: el.id,
      depth: Number(el.tagName[1]),
      text: el.textContent.trim(),
    })).filter((h) => h.text);
    setOutline((prev) => (JSON.stringify(prev) === JSON.stringify(next) ? prev : next));
  }, [body, components]);

  // The current section is the last heading that has reached the top of the
  // pane; looked up by id on every scroll, so a re-rendered heading can never
  // leave the highlight stuck.
  useEffect(() => {
    const root = rootRef.current;
    const scroller = root && scrollerOf(root);
    if (!scroller || outline.length < OUTLINE_MIN) return undefined;
    let frame = 0;
    const update = () => {
      frame = 0;
      const box = scroller.getBoundingClientRect();
      const top = box.top + JUMP_GAP * 2;
      const atEnd = scroller.scrollTop + scroller.clientHeight >= scroller.scrollHeight - 2;
      let current = outline[0].id;
      let lastSeen = "";
      let pickedSeen = false;
      for (const h of outline) {
        const el = root.querySelector(`[id="${CSS.escape(h.id)}"]`);
        if (!el) continue;
        const y = el.getBoundingClientRect().top;
        if (y <= top) current = h.id;
        if (y < box.bottom) lastSeen = h.id;
        if (h.id === pickedRef.current && y >= box.top - 1 && y < box.bottom) pickedSeen = true;
      }
      // At the bottom nothing else can reach the top: the clicked entry wins
      // while it is on screen, otherwise the last heading in view.
      if (atEnd) current = pickedSeen ? pickedRef.current : lastSeen || current;
      setActive(current);
    };
    const onScroll = () => { if (!frame) frame = requestAnimationFrame(update); };
    update();
    scroller.addEventListener("scroll", onScroll, { passive: true });
    return () => { scroller.removeEventListener("scroll", onScroll); cancelAnimationFrame(frame); };
  }, [outline]);

  const showOutline = outline.length >= OUTLINE_MIN;
  const top = Math.min(...outline.map((h) => h.depth));
  return (
    <div className="md-doc-wrap">
      <div ref={rootRef} className={"md-doc" + (showOutline ? " md-doc-has-outline" : "")}>
        <article className="md md-doc-body">
          {front != null ? <Frontmatter front={front} rows={rows} line={sourceLines ? 1 : undefined} /> : null}
          <Markdown
            remarkPlugins={pipeline.remarkPlugins}
            rehypePlugins={pipeline.rehypePlugins}
            remarkRehypeOptions={pipeline.remarkRehypeOptions}
            components={components}
          >
            {body}
          </Markdown>
        </article>
        {showOutline ? (
          <nav className="md-outline" aria-label="Outline">
            <p className="md-outline-title">On this page</p>
            <ul>
              {outline.map((h) => (
                <li key={h.id} style={{ "--depth": h.depth - top }}>
                  <a
                    href={"#" + h.id.slice(DOC_ID_PREFIX.length)}
                    aria-current={active === h.id ? "location" : undefined}
                    onClick={(e) => {
                      e.preventDefault();
                      const el = rootRef.current?.querySelector(`[id="${CSS.escape(h.id)}"]`);
                      pickedRef.current = h.id;
                      setActive(h.id);
                      if (el) scrollToHeading(rootRef.current, el);
                    }}
                  >
                    {h.text}
                  </a>
                </li>
              ))}
            </ul>
          </nav>
        ) : null}
      </div>
    </div>
  );
}

// scrollerOf finds the pane's own scroll box: the nearest ancestor that
// scrolls *and* overflows (on the phone the preview box is overflow:auto but
// grows with its content; its parent is the one that scrolls).
// scrollIntoView would also move every clipped ancestor — the pane header
// with them — so jumps scroll this one element and nothing else.
function scrollerOf(el) {
  let first = null;
  for (let n = el.parentElement; n; n = n.parentElement) {
    const { overflowY } = getComputedStyle(n);
    if (overflowY !== "auto" && overflowY !== "scroll") continue;
    if (n.scrollHeight > n.clientHeight + 1) return n;
    first = first || n;
  }
  return first;
}

function scrollToHeading(root, el) {
  const scroller = scrollerOf(root);
  if (!scroller) return;
  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const top = scroller.scrollTop + el.getBoundingClientRect().top - scroller.getBoundingClientRect().top - JUMP_GAP;
  scroller.scrollTo({ top: Math.max(0, top), behavior: reduce ? "auto" : "smooth" });
}

// An SVG from the repository is read as text and shown as a data: image —
// an <img> never runs an SVG's scripts, and the file API never has to serve
// SVG as a document on the app's origin.
function RepoSvg({ path, assetRef, alt, ...props }) {
  const [url, setUrl] = useState("");
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    const ctl = new AbortController();
    setUrl("");
    setFailed(false);
    const src = assetRef.current ? assetRef.current(path, "text") : "";
    if (!src) { setFailed(true); return undefined; }
    api(src, { signal: ctl.signal })
      .then((page) => { const u = svgDataUrl(page.text || ""); if (u) setUrl(u); else setFailed(true); })
      .catch(() => { if (!ctl.signal.aborted) setFailed(true); });
    return () => ctl.abort();
  }, [path, assetRef]);
  if (failed) return alt ? <span className="md-img-missing">{alt}</span> : null;
  if (!url) return <span className="md-img-pending" aria-hidden="true" />;
  return <img src={url} alt={alt} {...props} />;
}

// GitHub shows frontmatter as a table; a block that is not flat key/value
// stays readable as the YAML it is.
function Frontmatter({ front, rows, line }) {
  if (rows && !rows.length) return null;
  if (!rows) return <pre className="md-frontmatter-raw" data-line={line}><code>{front}</code></pre>;
  return (
    <table className="md-frontmatter" data-line={line}>
      <tbody>
        {rows.map(([k, v], i) => <tr key={i}><th scope="row">{k}</th><td>{v}</td></tr>)}
      </tbody>
    </table>
  );
}
