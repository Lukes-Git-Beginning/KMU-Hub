/**
 * Shared TipTap Rich Text Editor component.
 *
 * Used across Wiki, Kommunikation, Helpdesk, E-Mail, and Formulare modules.
 * Supports all standard formatting, tables, task lists, code blocks, and images.
 */
import { useEditor, EditorContent as TipTapContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Underline from '@tiptap/extension-underline'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import { Table, TableRow, TableCell, TableHeader } from '@tiptap/extension-table'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import TextAlign from '@tiptap/extension-text-align'
import { EditorToolbar } from './EditorToolbar'
import { EditorBubbleMenu } from './EditorBubbleMenu'
import { EditorFooter } from './EditorFooter'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

export interface RichTextEditorProps {
  content?: string
  onChange?: (html: string) => void
  placeholder?: string
  readOnly?: boolean
  className?: string
  showToolbar?: boolean
  showFooter?: boolean
  minHeight?: string
  maxHeight?: string
  autofocus?: boolean
  /** Simplified toolbar for compact contexts (e.g. chat reply, email) */
  compact?: boolean
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RichTextEditor({
  content = '',
  onChange,
  placeholder = 'Schreiben Sie hier...',
  readOnly = false,
  className,
  showToolbar = true,
  showFooter = true,
  minHeight = '150px',
  maxHeight = '500px',
  autofocus = false,
  compact = false,
}: RichTextEditorProps) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [1, 2, 3] },
        codeBlock: false, // replaced by lowlight version if needed
      }),
      Placeholder.configure({ placeholder }),
      Underline,
      Link.configure({
        openOnClick: false,
        HTMLAttributes: { class: 'text-primary underline cursor-pointer' },
      }),
      Image.configure({
        inline: false,
        HTMLAttributes: { class: 'rounded-md max-w-full' },
      }),
      Table.configure({ resizable: true }),
      TableRow,
      TableCell,
      TableHeader,
      TaskList,
      TaskItem.configure({ nested: true }),
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
    ],
    content,
    editable: !readOnly,
    autofocus,
    onUpdate: ({ editor: e }) => {
      onChange?.(e.getHTML())
    },
    editorProps: {
      attributes: {
        class: cn(
          'tiptap-content prose prose-sm dark:prose-invert max-w-none',
          'focus:outline-none px-4 py-3',
        ),
        style: `min-height: ${minHeight}; max-height: ${maxHeight}; overflow-y: auto;`,
      },
    },
  })

  if (!editor) return null

  return (
    <div
      className={cn(
        'rounded-lg border border-border bg-card text-card-foreground',
        'focus-within:ring-1 focus-within:ring-primary/30',
        readOnly && 'border-transparent bg-transparent',
        className,
      )}
    >
      {showToolbar && !readOnly && (
        <EditorToolbar editor={editor} compact={compact} />
      )}

      <EditorBubbleMenu editor={editor} />

      <TipTapContent editor={editor} />

      {showFooter && !readOnly && <EditorFooter editor={editor} />}
    </div>
  )
}

export default RichTextEditor
