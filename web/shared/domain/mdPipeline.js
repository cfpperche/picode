// The one markdown-as-document pipeline both apps hand to react-markdown.
// Order matters: raw HTML is parsed, then everything the author wrote is
// sanitized with GitHub's own allow-list (hast-util-sanitize's default
// schema), and only then do the trusted steps add KaTeX markup, alert classes
// and heading ids — sanitizing after them would strip their classes.
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeRaw from "rehype-raw";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";
import rehypeKatex from "rehype-katex";
import { rehypeDocHeadings, rehypeGithubAlerts, rehypeSourceLines } from "./mdDocument.js";

// `math-inline` / `math-display` ride on the code element remark-math emits;
// the default schema keeps only `language-*` there. Without them a `$$` block
// still renders (its <pre> says display), but keeping them costs nothing.
const schema = {
  ...defaultSchema,
  attributes: {
    ...defaultSchema.attributes,
    code: [["className", /^language-./, "math-inline", "math-display"]],
  },
};

export const docPipeline = Object.freeze({
  remarkPlugins: [remarkGfm, remarkMath],
  rehypePlugins: [rehypeRaw, [rehypeSanitize, schema], rehypeKatex, rehypeGithubAlerts, rehypeDocHeadings],
  // Sanitize prefixes every id once (user-content-…); a second prefix from
  // remark-rehype would leave footnote ids that no link points at.
  remarkRehypeOptions: { clobberPrefix: "" },
});

// withSourceLines is the same pipeline plus `data-line` on every block, for
// the Split view's scroll sync. `offset` is how many lines the frontmatter
// took off the top, so the numbers are the file's own.
export function withSourceLines(offset = 0) {
  return { ...docPipeline, rehypePlugins: [...docPipeline.rehypePlugins, [rehypeSourceLines, { offset }]] };
}
