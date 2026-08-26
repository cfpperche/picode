import { useEffect, useRef } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import Link from "@tiptap/extension-link";
import Placeholder from "@tiptap/extension-placeholder";
import { Markdown } from "tiptap-markdown";
import { pinFileFromDrop, pinFileURL } from "../lib/pinFileDrop.js";
import { IconBold, IconItalic, IconHeading, IconList, IconListOl, IconCode, IconQuote } from "./Icons.jsx";

export default function PinEditor({ pinId, markdown, onMarkdown, onFiles, onReady }) {
  const filesFn = useRef(onFiles);
  const mdFn = useRef(onMarkdown);
  const edRef = useRef(null);
  filesFn.current = onFiles;
  mdFn.current = onMarkdown;
  const editor = useEditor({
    immediatelyRender: false,
    extensions: [
      StarterKit.configure({ heading: { levels: [2, 3] } }),
      Image.configure({ inline: false, allowBase64: false }),
      Link.configure({ openOnClick: false, autolink: true }),
      Placeholder.configure({ placeholder: "Write…" }),
      Markdown.configure({ html: false, tightLists: true, bulletListMarker: "-" }),
    ],
    content: markdown || "",
    editorProps: {
      attributes: { class: "pin-tiptap", "aria-label": "Pin body" },
      handlePaste(_view, event) {
        const files = [...(event.clipboardData && event.clipboardData.files ? event.clipboardData.files : [])];
        if (!files.length) return false;
        if (filesFn.current) filesFn.current(files);
        return true;
      },
      handleDrop(_view, event) {
        const existing = pinFileFromDrop(event);
        if (existing) {
          event.preventDefault();
          event.stopPropagation();
          const url = pinFileURL(existing);
          if (url && edRef.current) edRef.current.chain().focus().setImage({ src: url, alt: existing.name || "" }).run();
          return true;
        }
        const files = [...(event.dataTransfer && event.dataTransfer.files ? event.dataTransfer.files : [])];
        if (!files.length) return false;
        event.preventDefault();
        event.stopPropagation();
        if (filesFn.current) filesFn.current(files);
        return true;
      },
    },
    onUpdate: ({ editor: ed }) => {
      if (mdFn.current) mdFn.current(ed.storage.markdown.getMarkdown());
    },
  });

  useEffect(() => {
    if (!editor) return;
    const cur = editor.storage.markdown.getMarkdown();
    if ((markdown || "") !== cur) editor.commands.setContent(markdown || "");
  }, [editor, pinId]);

  useEffect(() => {
    edRef.current = editor;
    if (onReady) onReady(editor);
    return () => { edRef.current = null; if (onReady) onReady(null); };
  }, [editor, onReady]);

  if (!editor) return <div className="pin-tiptap pin-tiptap-empty" />;

  return (
    <div className="pin-editor">
      <div className="pin-editor-bar" role="toolbar" aria-label="Format">
        <MarkBtn editor={editor} active={editor.isActive("bold")} onClick={() => editor.chain().focus().toggleBold().run()} title="Bold"><IconBold /></MarkBtn>
        <MarkBtn editor={editor} active={editor.isActive("italic")} onClick={() => editor.chain().focus().toggleItalic().run()} title="Italic"><IconItalic /></MarkBtn>
        <MarkBtn editor={editor} active={editor.isActive("heading", { level: 2 })} onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()} title="Heading"><IconHeading /></MarkBtn>
        <MarkBtn editor={editor} active={editor.isActive("bulletList")} onClick={() => editor.chain().focus().toggleBulletList().run()} title="List"><IconList /></MarkBtn>
        <MarkBtn editor={editor} active={editor.isActive("orderedList")} onClick={() => editor.chain().focus().toggleOrderedList().run()} title="Numbered list"><IconListOl /></MarkBtn>
        <MarkBtn editor={editor} active={editor.isActive("codeBlock")} onClick={() => editor.chain().focus().toggleCodeBlock().run()} title="Code"><IconCode /></MarkBtn>
        <MarkBtn editor={editor} active={editor.isActive("blockquote")} onClick={() => editor.chain().focus().toggleBlockquote().run()} title="Quote"><IconQuote /></MarkBtn>
      </div>
      <EditorContent editor={editor} />
    </div>
  );
}

function MarkBtn({ active, onClick, title, children }) {
  return (
    <button type="button" className={"pin-mark" + (active ? " on" : "")} title={title} onClick={onClick}>
      {children}
    </button>
  );
}
