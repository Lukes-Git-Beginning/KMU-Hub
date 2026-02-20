import type { Editor } from '@tiptap/react'
import {
  Bold,
  Italic,
  Underline,
  Strikethrough,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  ListChecks,
  AlignLeft,
  AlignCenter,
  AlignRight,
  Link,
  Image,
  Table,
  Code,
  Minus,
  Undo,
  Redo,
  Quote,
} from 'lucide-react'
import { useCallback } from 'react'
import { ToolbarButton } from './ToolbarButton'
import { Separator } from '@/components/ui/separator'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface EditorToolbarProps {
  editor: Editor
  compact?: boolean
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function EditorToolbar({ editor, compact = false }: EditorToolbarProps) {
  const addLink = useCallback(() => {
    const url = window.prompt('URL eingeben:')
    if (url) {
      editor.chain().focus().setLink({ href: url }).run()
    }
  }, [editor])

  const addImage = useCallback(() => {
    const url = window.prompt('Bild-URL eingeben:')
    if (url) {
      editor.chain().focus().setImage({ src: url }).run()
    }
  }, [editor])

  const addTable = useCallback(() => {
    editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
  }, [editor])

  return (
    <div className="flex flex-wrap items-center gap-0.5 border-b border-border px-2 py-1.5">
      {/* Undo / Redo */}
      <ToolbarButton
        icon={Undo}
        onClick={() => editor.chain().focus().undo().run()}
        disabled={!editor.can().undo()}
        tooltip="Rückgängig"
      />
      <ToolbarButton
        icon={Redo}
        onClick={() => editor.chain().focus().redo().run()}
        disabled={!editor.can().redo()}
        tooltip="Wiederholen"
      />

      <Separator orientation="vertical" className="mx-1 h-5" />

      {/* Format */}
      <ToolbarButton
        icon={Bold}
        onClick={() => editor.chain().focus().toggleBold().run()}
        active={editor.isActive('bold')}
        tooltip="Fett"
      />
      <ToolbarButton
        icon={Italic}
        onClick={() => editor.chain().focus().toggleItalic().run()}
        active={editor.isActive('italic')}
        tooltip="Kursiv"
      />
      <ToolbarButton
        icon={Underline}
        onClick={() => editor.chain().focus().toggleUnderline().run()}
        active={editor.isActive('underline')}
        tooltip="Unterstrichen"
      />
      {!compact && (
        <ToolbarButton
          icon={Strikethrough}
          onClick={() => editor.chain().focus().toggleStrike().run()}
          active={editor.isActive('strike')}
          tooltip="Durchgestrichen"
        />
      )}

      <Separator orientation="vertical" className="mx-1 h-5" />

      {/* Headings */}
      {!compact && (
        <>
          <ToolbarButton
            icon={Heading1}
            onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
            active={editor.isActive('heading', { level: 1 })}
            tooltip="Überschrift 1"
          />
          <ToolbarButton
            icon={Heading2}
            onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
            active={editor.isActive('heading', { level: 2 })}
            tooltip="Überschrift 2"
          />
          <ToolbarButton
            icon={Heading3}
            onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
            active={editor.isActive('heading', { level: 3 })}
            tooltip="Überschrift 3"
          />
          <Separator orientation="vertical" className="mx-1 h-5" />
        </>
      )}

      {/* Lists */}
      <ToolbarButton
        icon={List}
        onClick={() => editor.chain().focus().toggleBulletList().run()}
        active={editor.isActive('bulletList')}
        tooltip="Aufzählung"
      />
      <ToolbarButton
        icon={ListOrdered}
        onClick={() => editor.chain().focus().toggleOrderedList().run()}
        active={editor.isActive('orderedList')}
        tooltip="Nummerierung"
      />
      {!compact && (
        <ToolbarButton
          icon={ListChecks}
          onClick={() => editor.chain().focus().toggleTaskList().run()}
          active={editor.isActive('taskList')}
          tooltip="Aufgabenliste"
        />
      )}

      {!compact && (
        <>
          <Separator orientation="vertical" className="mx-1 h-5" />

          {/* Align */}
          <ToolbarButton
            icon={AlignLeft}
            onClick={() => editor.chain().focus().setTextAlign('left').run()}
            active={editor.isActive({ textAlign: 'left' })}
            tooltip="Linksbündig"
          />
          <ToolbarButton
            icon={AlignCenter}
            onClick={() => editor.chain().focus().setTextAlign('center').run()}
            active={editor.isActive({ textAlign: 'center' })}
            tooltip="Zentriert"
          />
          <ToolbarButton
            icon={AlignRight}
            onClick={() => editor.chain().focus().setTextAlign('right').run()}
            active={editor.isActive({ textAlign: 'right' })}
            tooltip="Rechtsbündig"
          />
        </>
      )}

      <Separator orientation="vertical" className="mx-1 h-5" />

      {/* Insert */}
      <ToolbarButton
        icon={Link}
        onClick={addLink}
        active={editor.isActive('link')}
        tooltip="Link einfügen"
      />
      {!compact && (
        <>
          <ToolbarButton icon={Image} onClick={addImage} tooltip="Bild einfügen" />
          <ToolbarButton icon={Table} onClick={addTable} tooltip="Tabelle einfügen" />
          <ToolbarButton
            icon={Code}
            onClick={() => editor.chain().focus().toggleCodeBlock().run()}
            active={editor.isActive('codeBlock')}
            tooltip="Code-Block"
          />
          <ToolbarButton
            icon={Quote}
            onClick={() => editor.chain().focus().toggleBlockquote().run()}
            active={editor.isActive('blockquote')}
            tooltip="Zitat"
          />
          <ToolbarButton
            icon={Minus}
            onClick={() => editor.chain().focus().setHorizontalRule().run()}
            tooltip="Trennlinie"
          />
        </>
      )}
    </div>
  )
}
