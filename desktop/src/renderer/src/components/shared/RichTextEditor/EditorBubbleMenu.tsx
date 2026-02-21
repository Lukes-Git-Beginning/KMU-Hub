/**
 * Inline formatting toolbar that appears on text selection.
 *
 * Replaces the TipTap v2 <BubbleMenu> React component which
 * is no longer exported from @tiptap/react v3.
 */
import type { Editor } from '@tiptap/react'
import { Bold, Italic, Underline, Link, Unlink } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { ToolbarButton } from './ToolbarButton'

interface EditorBubbleMenuProps {
  editor: Editor
}

export function EditorBubbleMenu({ editor }: EditorBubbleMenuProps) {
  const ref = useRef<HTMLDivElement>(null)
  const [visible, setVisible] = useState(false)
  const [pos, setPos] = useState({ top: 0, left: 0 })

  useEffect(() => {
    const update = () => {
      const { from, to, empty } = editor.state.selection
      if (empty || from === to) {
        setVisible(false)
        return
      }
      const dom = editor.view.dom.closest('.rounded-lg') as HTMLElement | null
      if (!dom) { setVisible(false); return }
      const coords = editor.view.coordsAtPos(from)
      const parentRect = dom.getBoundingClientRect()
      setPos({
        top: coords.top - parentRect.top - 44,
        left: coords.left - parentRect.left,
      })
      setVisible(true)
    }
    editor.on('selectionUpdate', update)
    editor.on('blur', () => setVisible(false))
    return () => {
      editor.off('selectionUpdate', update)
    }
  }, [editor])

  const setLink = useCallback(() => {
    const previousUrl = editor.getAttributes('link').href as string | undefined
    const url = window.prompt('URL eingeben:', previousUrl)
    if (url === null) return
    if (url === '') {
      editor.chain().focus().unsetLink().run()
      return
    }
    editor.chain().focus().setLink({ href: url }).run()
  }, [editor])

  if (!visible) return null

  return (
    <div
      ref={ref}
      className="absolute z-50 flex items-center gap-0.5 rounded-lg border border-border bg-popover px-1.5 py-1 shadow-md"
      style={{ top: pos.top, left: pos.left }}
    >
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
      {editor.isActive('link') ? (
        <ToolbarButton
          icon={Unlink}
          onClick={() => editor.chain().focus().unsetLink().run()}
          tooltip="Link entfernen"
        />
      ) : (
        <ToolbarButton
          icon={Link}
          onClick={setLink}
          tooltip="Link einfuegen"
        />
      )}
    </div>
  )
}
