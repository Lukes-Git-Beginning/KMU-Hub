/**
 * Wiki-specific rich text editor.
 *
 * Wraps TipTap directly (rather than the shared RichTextEditor) so it can add
 * the wiki-only WikiLink + WikiMention nodes, the rich blocks (Callout, Code,
 * Toggle, Figure — WP-2) and three trigger overlays, while reusing the shared
 * editor's toolbar / bubble menu for all standard formatting.
 *
 * Triggers:
 *   /query   → insert a block (slash menu, keyboard-navigable — WP-2)
 *   @query   → mention a team member (suggestions from useEmployees)
 *   [[query  → link an article (suggestions from useArticles)
 */
import { useMemo, useRef, useState } from 'react'
import { useEditor, EditorContent, type Editor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Underline from '@tiptap/extension-underline'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import { Table, TableRow, TableCell, TableHeader } from '@tiptap/extension-table'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import TextAlign from '@tiptap/extension-text-align'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import { useTranslation } from 'react-i18next'
import {
  AtSign,
  FileText,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  ListChecks,
  Info,
  Code2,
  ChevronRight,
  Quote,
  Minus,
  ImagePlus,
  Table as TableIcon,
  type LucideIcon,
} from 'lucide-react'
import { cn } from '@/lib'
import { EditorToolbar } from '@/components/shared/RichTextEditor/EditorToolbar'
import { EditorBubbleMenu } from '@/components/shared/RichTextEditor/EditorBubbleMenu'
import { useArticles } from '@/api/hooks/useWiki'
import { useEmployees } from '@/api/hooks/hr-hooks'
import { WikiLink } from './extensions/WikiLinkExtension'
import { WikiMention } from './extensions/WikiMentionExtension'
import { Callout } from './extensions/CalloutExtension'
import { DetailsBlock } from './extensions/DetailsExtension'
import { FigureImage } from './extensions/FigureExtension'
import { lowlight } from './extensions/lowlight'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface SuggestState {
  type: 'mention' | 'wikilink' | 'slash'
  query: string
  from: number
  left: number
  top: number
}

interface SuggestItem {
  key: string
  label: string
  sub: string
  icon?: LucideIcon
  /** mention / wikilink → insert a node; slash → run an editor command. */
  insert?: { articleId?: string; slug?: string; userId?: string; label: string }
  run?: (editor: Editor) => void
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
// Slash commands — the block insert menu (Notion-style)
// ---------------------------------------------------------------------------

interface SlashCommand {
  key: string
  icon: LucideIcon
  labelKey: string
  hintKey: string
  /** Extra search terms so e.g. "h1" or "bild" still find the right block. */
  terms: string
  run: (editor: Editor) => void
}

const SLASH_COMMANDS: SlashCommand[] = [
  {
    key: 'h1',
    icon: Heading1,
    labelKey: 'wiki.block.slash.h1',
    hintKey: 'wiki.block.slash.h1Hint',
    terms: 'h1 heading überschrift titel title',
    run: (e) => e.chain().focus().setNode('heading', { level: 1 }).run(),
  },
  {
    key: 'h2',
    icon: Heading2,
    labelKey: 'wiki.block.slash.h2',
    hintKey: 'wiki.block.slash.h2Hint',
    terms: 'h2 heading überschrift',
    run: (e) => e.chain().focus().setNode('heading', { level: 2 }).run(),
  },
  {
    key: 'h3',
    icon: Heading3,
    labelKey: 'wiki.block.slash.h3',
    hintKey: 'wiki.block.slash.h3Hint',
    terms: 'h3 heading überschrift',
    run: (e) => e.chain().focus().setNode('heading', { level: 3 }).run(),
  },
  {
    key: 'bullet',
    icon: List,
    labelKey: 'wiki.block.slash.bullet',
    hintKey: 'wiki.block.slash.bulletHint',
    terms: 'bullet list liste aufzählung ul',
    run: (e) => e.chain().focus().toggleBulletList().run(),
  },
  {
    key: 'ordered',
    icon: ListOrdered,
    labelKey: 'wiki.block.slash.ordered',
    hintKey: 'wiki.block.slash.orderedHint',
    terms: 'ordered list nummeriert liste ol',
    run: (e) => e.chain().focus().toggleOrderedList().run(),
  },
  {
    key: 'task',
    icon: ListChecks,
    labelKey: 'wiki.block.slash.task',
    hintKey: 'wiki.block.slash.taskHint',
    terms: 'task todo aufgabe checkliste check',
    run: (e) => e.chain().focus().toggleTaskList().run(),
  },
  {
    key: 'callout',
    icon: Info,
    labelKey: 'wiki.block.slash.callout',
    hintKey: 'wiki.block.slash.calloutHint',
    terms: 'callout hinweis info box note',
    run: (e) =>
      e
        .chain()
        .focus()
        .insertContent({
          type: 'callout',
          attrs: { variant: 'info' },
          content: [{ type: 'paragraph' }],
        })
        .run(),
  },
  {
    key: 'code',
    icon: Code2,
    labelKey: 'wiki.block.slash.code',
    hintKey: 'wiki.block.slash.codeHint',
    terms: 'code snippet syntax pre',
    run: (e) => e.chain().focus().setNode('codeBlock').run(),
  },
  {
    key: 'toggle',
    icon: ChevronRight,
    labelKey: 'wiki.block.slash.toggle',
    hintKey: 'wiki.block.slash.toggleHint',
    terms: 'toggle details aufklappen accordion collapse',
    run: (e) =>
      e
        .chain()
        .focus()
        .insertContent({
          type: 'detailsBlock',
          attrs: { summary: '', open: true },
          content: [{ type: 'paragraph' }],
        })
        .run(),
  },
  {
    key: 'quote',
    icon: Quote,
    labelKey: 'wiki.block.slash.quote',
    hintKey: 'wiki.block.slash.quoteHint',
    terms: 'quote blockquote zitat',
    run: (e) => e.chain().focus().toggleBlockquote().run(),
  },
  {
    key: 'divider',
    icon: Minus,
    labelKey: 'wiki.block.slash.divider',
    hintKey: 'wiki.block.slash.dividerHint',
    terms: 'divider hr trenner rule line',
    run: (e) => e.chain().focus().setHorizontalRule().run(),
  },
  {
    key: 'image',
    icon: ImagePlus,
    labelKey: 'wiki.block.slash.image',
    hintKey: 'wiki.block.slash.imageHint',
    terms: 'image bild figure foto picture',
    run: (e) => e.chain().focus().insertContent({ type: 'figureImage' }).run(),
  },
  {
    key: 'table',
    icon: TableIcon,
    labelKey: 'wiki.block.slash.table',
    hintKey: 'wiki.block.slash.tableHint',
    terms: 'table tabelle grid',
    run: (e) => e.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run(),
  },
]

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
      CodeBlockLowlight.configure({ lowlight, HTMLAttributes: { class: 'wiki-code' } }),
      Callout,
      DetailsBlock,
      FigureImage,
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
        class: cn('tiptap-content wiki-canvas prose dark:prose-invert max-w-none', 'focus:outline-none'),
        style: 'min-height: 52vh;',
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
          if (event.key === 'Tab') { k.choose(k.index); return true }
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

    // Slash menu — only outside code blocks, at a block start or after a space.
    if (!editor.isActive('codeBlock')) {
      const sl = /(?:^|\s)\/([\w]*)$/.exec(slice)
      if (sl) {
        const coords = editor.view.coordsAtPos(from)
        setIndex(0)
        setSuggest({ type: 'slash', query: sl[1], from: from - sl[1].length - 1, left: coords.left, top: coords.bottom })
        return
      }
    }

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
    if (suggest.type === 'slash') {
      return SLASH_COMMANDS.filter(
        (c) => !q || t(c.labelKey).toLowerCase().includes(q) || c.terms.includes(q),
      ).map((c) => ({
        key: c.key,
        label: t(c.labelKey),
        sub: t(c.hintKey),
        icon: c.icon,
        run: c.run,
      }))
    }
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
  }, [suggest, articlesData, employeesData, t])

  const choose = (i: number) => {
    if (!editor || !suggest) return
    const item = items[i]
    if (!item) return
    const to = editor.state.selection.from

    // Slash command — delete the "/query" text, then run the block command.
    if (suggest.type === 'slash' && item.run) {
      editor.chain().focus().deleteRange({ from: suggest.from, to }).run()
      item.run(editor)
      setSuggest(null)
      return
    }

    const node =
      suggest.type === 'wikilink'
        ? { type: 'wikiLink', attrs: { articleId: item.insert?.articleId, slug: item.insert?.slug, label: item.insert?.label } }
        : { type: 'wikiMention', attrs: { userId: item.insert?.userId, label: item.insert?.label } }
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

  const isSlash = suggest?.type === 'slash'

  return (
    // `rounded-lg` is kept purely as the positioning anchor the shared
    // EditorBubbleMenu looks up via closest('.rounded-lg') — visually the canvas
    // is frameless (no border/bg), so the page reads as one quiet surface.
    <div className="relative rounded-lg">
      <div className="wiki-toolbar-sticky -mx-1 mb-3 rounded-md border-b border-border/60">
        <EditorToolbar editor={editor} />
      </div>
      <EditorBubbleMenu editor={editor} />
      <EditorContent editor={editor} />

      {/* Autocomplete / slash overlay */}
      {suggest && items.length > 0 && (
        <div
          className={cn(
            'fixed z-[60] overflow-hidden rounded-lg border border-border bg-popover py-1 shadow-lg',
            isSlash ? 'w-72' : 'w-72',
          )}
          style={{ left: suggest.left, top: suggest.top + 4 }}
        >
          <p className="flex items-center gap-1.5 px-3 py-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
            {suggest.type === 'mention' ? (
              <AtSign className="h-3 w-3" />
            ) : suggest.type === 'wikilink' ? (
              <FileText className="h-3 w-3" />
            ) : null}
            {suggest.type === 'mention'
              ? t('wiki.mention.title')
              : suggest.type === 'wikilink'
                ? t('wiki.wikilink.title')
                : t('wiki.block.slash.title')}
          </p>
          <div className="max-h-72 overflow-y-auto">
            {items.map((item, i) => {
              const ItemIcon = item.icon
              return (
                <button
                  key={item.key}
                  onMouseDown={(e) => { e.preventDefault(); choose(i) }}
                  onMouseEnter={() => setIndex(i)}
                  className={cn(
                    'flex w-full items-center gap-2.5 px-3 py-1.5 text-left transition-colors',
                    i === index ? 'bg-secondary' : 'hover:bg-secondary/50',
                  )}
                >
                  {ItemIcon && (
                    <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-border bg-card text-muted-foreground">
                      <ItemIcon className="h-3.5 w-3.5" />
                    </span>
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-foreground">{item.label}</p>
                    {item.sub && <p className="truncate text-[11px] text-muted-foreground">{item.sub}</p>}
                  </div>
                </button>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
