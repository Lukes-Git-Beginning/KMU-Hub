/**
 * Wiki-specific rich text editor.
 *
 * Wraps TipTap directly (rather than the shared RichTextEditor) so it can add
 * the wiki-only WikiLink + WikiMention nodes and an autocomplete overlay, while
 * reusing the shared editor's toolbar / bubble menu for all standard formatting.
 *
 * Triggers:
 *   @query   → mention a team member (suggestions from useEmployees)
 *   [[query  → link an article (suggestions from useArticles)
 */
import { useMemo, useRef, useState } from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Underline from '@tiptap/extension-underline'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import { Table, TableRow, TableCell, TableHeader } from '@tiptap/extension-table'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import TextAlign from '@tiptap/extension-text-align'
import { useTranslation } from 'react-i18next'
import { AtSign, FileText } from 'lucide-react'
import { cn } from '@/lib'
import { EditorToolbar } from '@/components/shared/RichTextEditor/EditorToolbar'
import { EditorBubbleMenu } from '@/components/shared/RichTextEditor/EditorBubbleMenu'
import { useArticles } from '@/api/hooks/useWiki'
import { useEmployees } from '@/api/hooks/hr-hooks'
import { WikiLink } from './extensions/WikiLinkExtension'
import { WikiMention } from './extensions/WikiMentionExtension'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface SuggestState {
  type: 'mention' | 'wikilink'
  query: string
  from: number
  left: number
  top: number
}

interface SuggestItem {
  key: string
  label: string
  sub: string
  insert: { articleId?: string; slug?: string; userId?: string; label: string }
}

interface WikiRichEditorProps {
  content: string
  onChange: (html: string) => void
  placeholder?: string
  autofocus?: boolean
  /** Ctrl/Cmd+S inside the editor. */
  onSave?: () => void
  /** Escape inside the editor (when no suggestion is open). */
  onCancel?: () => void
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function WikiRichEditor({ content, onChange, placeholder, autofocus, onSave, onCancel }: WikiRichEditorProps) {
  const { t } = useTranslation()
  const { data: articlesData } = useArticles()
  const { data: employeesData } = useEmployees()

  const [suggest, setSuggest] = useState<SuggestState | null>(null)
  const [index, setIndex] = useState(0)

  // Keyboard handler reads live state through a ref (editorProps is set once).
  const kbRef = useRef({
    open: false,
    count: 0,
    index: 0,
    move: (_d: number) => {},
    choose: (_i: number) => {},
    close: () => {},
    save: () => {},
    cancel: () => {},
  })

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ heading: { levels: [1, 2, 3] }, codeBlock: false }),
      Placeholder.configure({ placeholder: placeholder ?? '' }),
      Underline,
      Link.configure({ openOnClick: false, HTMLAttributes: { class: 'text-primary underline cursor-pointer' } }),
      Image.configure({ inline: false, HTMLAttributes: { class: 'rounded-md max-w-full' } }),
      Table.configure({ resizable: true }),
      TableRow,
      TableCell,
      TableHeader,
      TaskList,
      TaskItem.configure({ nested: true }),
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
      WikiLink,
      WikiMention,
    ],
    content,
    autofocus,
    onUpdate: ({ editor: e }) => {
      onChange(e.getHTML())
      detect()
    },
    onSelectionUpdate: () => detect(),
    editorProps: {
      attributes: {
        class: cn('tiptap-content prose prose-sm dark:prose-invert max-w-none', 'focus:outline-none px-4 py-3'),
        style: 'min-height: 320px;',
      },
      handleKeyDown: (_view, event) => {
        const k = kbRef.current
        // Ctrl/Cmd+S saves regardless of suggestion state.
        if (event.key === 's' && (event.ctrlKey || event.metaKey)) {
          event.preventDefault()
          k.save()
          return true
        }
        if (k.open && k.count > 0) {
          if (event.key === 'ArrowDown') { k.move(1); return true }
          if (event.key === 'ArrowUp') { k.move(-1); return true }
          if (event.key === 'Enter') { k.choose(k.index); return true }
          if (event.key === 'Escape') { k.close(); return true }
        } else if (event.key === 'Escape') {
          k.cancel()
          return true
        }
        return false
      },
    },
  })

  // ---- trigger detection ----
  function detect() {
    if (!editor) return
    const sel = editor.state.selection
    if (!sel.empty) { setSuggest(null); return }
    const from = sel.from
    const slice = editor.state.doc.textBetween(Math.max(0, from - 80), from, '\n', '\n')

    const wl = /\[\[([^[\]\n]*)$/.exec(slice)
    if (wl) {
      const coords = editor.view.coordsAtPos(from)
      setIndex(0)
      setSuggest({ type: 'wikilink', query: wl[1], from: from - wl[0].length, left: coords.left, top: coords.bottom })
      return
    }
    const mn = /(?:^|[\s(])@([^\s@]{0,30})$/.exec(slice)
    if (mn) {
      const coords = editor.view.coordsAtPos(from)
      setIndex(0)
      setSuggest({ type: 'mention', query: mn[1], from: from - (mn[1].length + 1), left: coords.left, top: coords.bottom })
      return
    }
    setSuggest(null)
  }

  // ---- suggestion items ----
  const items: SuggestItem[] = useMemo(() => {
    if (!suggest) return []
    const q = suggest.query.toLowerCase()
    if (suggest.type === 'wikilink') {
      return (articlesData?.articles ?? [])
        .filter((a) => a.title.toLowerCase().includes(q))
        .slice(0, 6)
        .map((a) => ({
          key: a.id,
          label: a.title,
          sub: a.slug,
          insert: { articleId: a.id, slug: a.slug, label: a.title },
        }))
    }
    return (employeesData?.employees ?? [])
      .filter((e) => (e.userName ?? '').toLowerCase().includes(q))
      .slice(0, 6)
      .map((e) => {
        const first = (e.userName ?? '').split(' ')[0] || e.userName || ''
        return {
          key: e.id,
          label: e.userName ?? first,
          sub: e.positionTitle ?? '',
          insert: { userId: e.userId, label: first },
        }
      })
  }, [suggest, articlesData, employeesData])

  const choose = (i: number) => {
    if (!editor || !suggest) return
    const item = items[i]
    if (!item) return
    const to = editor.state.selection.from
    const node =
      suggest.type === 'wikilink'
        ? { type: 'wikiLink', attrs: { articleId: item.insert.articleId, slug: item.insert.slug, label: item.insert.label } }
        : { type: 'wikiMention', attrs: { userId: item.insert.userId, label: item.insert.label } }
    editor
      .chain()
      .focus()
      .deleteRange({ from: suggest.from, to })
      .insertContent([node, { type: 'text', text: ' ' }])
      .run()
    setSuggest(null)
  }

  // keep the keyboard ref in sync each render
  kbRef.current = {
    open: !!suggest,
    count: items.length,
    index,
    move: (d) => setIndex((i) => Math.max(0, Math.min(items.length - 1, i + d))),
    choose,
    close: () => setSuggest(null),
    save: () => onSave?.(),
    cancel: () => onCancel?.(),
  }

  if (!editor) return null

  return (
    <div className="relative rounded-lg border border-border bg-card text-card-foreground focus-within:ring-1 focus-within:ring-primary/30">
      <EditorToolbar editor={editor} />
      <EditorBubbleMenu editor={editor} />
      <EditorContent editor={editor} />

      {/* Autocomplete overlay */}
      {suggest && items.length > 0 && (
        <div
          className="fixed z-[60] w-72 overflow-hidden rounded-lg border border-border bg-popover py-1 shadow-lg"
          style={{ left: suggest.left, top: suggest.top + 4 }}
        >
          <p className="flex items-center gap-1.5 px-3 py-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
            {suggest.type === 'mention' ? <AtSign className="h-3 w-3" /> : <FileText className="h-3 w-3" />}
            {suggest.type === 'mention' ? t('wiki.mention.title') : t('wiki.wikilink.title')}
          </p>
          <div className="max-h-56 overflow-y-auto">
            {items.map((item, i) => (
              <button
                key={item.key}
                onMouseDown={(e) => { e.preventDefault(); choose(i) }}
                onMouseEnter={() => setIndex(i)}
                className={cn(
                  'flex w-full items-center gap-2 px-3 py-1.5 text-left transition-colors',
                  i === index ? 'bg-secondary' : 'hover:bg-secondary/50',
                )}
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-foreground">{item.label}</p>
                  {item.sub && <p className="truncate text-[11px] text-muted-foreground">{item.sub}</p>}
                </div>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
