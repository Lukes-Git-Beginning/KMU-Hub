/**
 * Inline formatting toolbar that appears on text selection.
 *
 * Replaces the TipTap v2 <BubbleMenu> React component which
 * is no longer exported from @tiptap/react v3.
 */
import type { Editor } from '@tiptap/react'
import {
  Bold,
  Italic,
  Underline,
  Link,
  Unlink,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
} from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ToolbarButton } from './ToolbarButton'
import { Separator } from '@/components/ui/separator'

interface EditorBubbleMenuProps {
  editor: Editor
  /** Long-form (frameless) mode: also offer heading + list toggles on selection. */
  richBlocks?: boolean
}

export function EditorBubbleMenu({ editor, richBlocks = false }: EditorBubbleMenuProps) {
  const { t } = useTranslation()
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
      {richBlocks && (
        <>
          <ToolbarButton
            icon={Heading1}
            onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
            active={editor.isActive('heading', { level: 1 })}
            tooltip={t('shared.editor.heading1')}
          />
          <ToolbarButton
            icon={Heading2}
            onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
            active={editor.isActive('heading', { level: 2 })}
            tooltip={t('shared.editor.heading2')}
          />
          <ToolbarButton
            icon={Heading3}
            onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
            active={editor.isActive('heading', { level: 3 })}
            tooltip={t('shared.editor.heading3')}
          />
          <ToolbarButton
            icon={List}
            onClick={() => editor.chain().focus().toggleBulletList().run()}
            active={editor.isActive('bulletList')}
            tooltip={t('shared.editor.bulletList')}
          />
          <ToolbarButton
            icon={ListOrdered}
            onClick={() => editor.chain().focus().toggleOrderedList().run()}
            active={editor.isActive('orderedList')}
            tooltip={t('shared.editor.orderedList')}
          />
          <Separator orientation="vertical" className="mx-0.5 h-5" />
        </>
      )}
      <ToolbarButton
        icon={Bold}
        onClick={() => editor.chain().focus().toggleBold().run()}
        active={editor.isActive('bold')}
        tooltip={t('shared.editor.bold')}
      />
      <ToolbarButton
        icon={Italic}
        onClick={() => editor.chain().focus().toggleItalic().run()}
        active={editor.isActive('italic')}
        tooltip={t('shared.editor.italic')}
      />
      <ToolbarButton
        icon={Underline}
        onClick={() => editor.chain().focus().toggleUnderline().run()}
        active={editor.isActive('underline')}
        tooltip={t('shared.editor.underline')}
      />
      {editor.isActive('link') ? (
        <ToolbarButton
          icon={Unlink}
          onClick={() => editor.chain().focus().unsetLink().run()}
          tooltip={t('shared.editor.removeLink')}
        />
      ) : (
        <ToolbarButton
          icon={Link}
          onClick={setLink}
          tooltip={t('shared.editor.insertLink')}
        />
      )}
    </div>
  )
}
