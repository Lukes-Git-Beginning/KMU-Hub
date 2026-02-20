import { BubbleMenu } from '@tiptap/react'
import type { Editor } from '@tiptap/react'
import { Bold, Italic, Underline, Link, Unlink } from 'lucide-react'
import { useCallback } from 'react'
import { ToolbarButton } from './ToolbarButton'

interface EditorBubbleMenuProps {
  editor: Editor
}

export function EditorBubbleMenu({ editor }: EditorBubbleMenuProps) {
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

  return (
    <BubbleMenu
      editor={editor}
      tippyOptions={{ duration: 150 }}
      className="flex items-center gap-0.5 rounded-lg border border-border bg-popover px-1.5 py-1 shadow-md"
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
          tooltip="Link einfügen"
        />
      )}
    </BubbleMenu>
  )
}
