import { useEffect } from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import {
  Bold, Italic, Strikethrough, List, ListOrdered,
  Heading2, Heading3, Minus, Undo2, Redo2,
} from 'lucide-react'
import { cn } from '@/lib'

interface RichTextEditorProps {
  content: string
  onChange: (html: string) => void
  placeholder?: string
  className?: string
  editable?: boolean
}

export function RichTextEditor({
  content,
  onChange,
  placeholder = 'Write something...',
  className,
  editable = true,
}: RichTextEditorProps) {
  const editor = useEditor({
    extensions: [
      StarterKit,
      Placeholder.configure({ placeholder }),
    ],
    content,
    editable,
    onUpdate: ({ editor: e }) => onChange(e.getHTML()),
  })

  useEffect(() => {
    if (editor && content !== editor.getHTML()) {
      editor.commands.setContent(content)
    }
  }, [content, editor])

  useEffect(() => {
    editor?.setEditable(editable)
  }, [editable, editor])

  if (!editor) return null

  const btn = (
    active: boolean,
    action: () => void,
    icon: React.ReactNode,
    title: string,
  ) => (
    <button
      type="button"
      title={title}
      onClick={action}
      className={cn(
        'p-1.5 rounded hover:bg-secondary/80 transition-colors',
        active && 'bg-secondary text-primary',
      )}
    >
      {icon}
    </button>
  )

  const s = 16

  return (
    <div className={cn('rounded-lg border border-border bg-card overflow-hidden', className)}>
      {editable && (
        <div className="flex flex-wrap items-center gap-0.5 px-2 py-1.5">
          {btn(editor.isActive('bold'), () => editor.chain().focus().toggleBold().run(), <Bold size={s} />, 'Bold')}
          {btn(editor.isActive('italic'), () => editor.chain().focus().toggleItalic().run(), <Italic size={s} />, 'Italic')}
          {btn(editor.isActive('strike'), () => editor.chain().focus().toggleStrike().run(), <Strikethrough size={s} />, 'Strikethrough')}
          <span className="mx-1 h-5 w-px bg-border" />
          {btn(editor.isActive('heading', { level: 2 }), () => editor.chain().focus().toggleHeading({ level: 2 }).run(), <Heading2 size={s} />, 'Heading 2')}
          {btn(editor.isActive('heading', { level: 3 }), () => editor.chain().focus().toggleHeading({ level: 3 }).run(), <Heading3 size={s} />, 'Heading 3')}
          <span className="mx-1 h-5 w-px bg-border" />
          {btn(editor.isActive('bulletList'), () => editor.chain().focus().toggleBulletList().run(), <List size={s} />, 'Bullet List')}
          {btn(editor.isActive('orderedList'), () => editor.chain().focus().toggleOrderedList().run(), <ListOrdered size={s} />, 'Ordered List')}
          {btn(false, () => editor.chain().focus().setHorizontalRule().run(), <Minus size={s} />, 'Horizontal Rule')}
          <span className="mx-1 h-5 w-px bg-border" />
          {btn(false, () => editor.chain().focus().undo().run(), <Undo2 size={s} />, 'Undo')}
          {btn(false, () => editor.chain().focus().redo().run(), <Redo2 size={s} />, 'Redo')}
        </div>
      )}
      <EditorContent
        editor={editor}
        className={cn(
          'prose prose-sm max-w-none px-3 py-2 text-foreground [&_.tiptap]:min-h-[120px] [&_.tiptap]:outline-none',
          '[&_.tiptap_p.is-editor-empty:first-child::before]:text-muted-foreground [&_.tiptap_p.is-editor-empty:first-child::before]:content-[attr(data-placeholder)] [&_.tiptap_p.is-editor-empty:first-child::before]:float-left [&_.tiptap_p.is-editor-empty:first-child::before]:h-0 [&_.tiptap_p.is-editor-empty:first-child::before]:pointer-events-none',
          editable && 'border-t border-border',
        )}
      />
    </div>
  )
}
