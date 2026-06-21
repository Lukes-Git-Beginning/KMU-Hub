/**
 * Inline link/mention autocomplete for the wiki long-form text block (PB-4b).
 *
 * Zero-dependency: instead of @tiptap/suggestion it watches the editor's
 * selection, matches a trailing `[[query` (articles) or `@query` (people) before
 * the caret, and shows a positioned picker. Selecting an item replaces the
 * trigger text with the corresponding atomic WikiLink / WikiMention node — the
 * same nodes the reader turns into preview popovers (PB-4a).
 */
import { useCallback, useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { AtSign, FileText } from 'lucide-react'
import type { Editor } from '@tiptap/react'
import { useArticles } from '@/api/hooks/useWiki'
import { useEmployees } from '@/api/hooks/hr-hooks'

type TriggerKind = 'article' | 'user'
interface Trigger {
  kind: TriggerKind
  query: string
  from: number
  to: number
  left: number
  top: number
}
interface SuggestItem {
  id: string
  title: string
  sub?: string
  slug?: string
}

const MAX_ITEMS = 6

function caretCoords(editor: Editor, pos: number): { left: number; top: number } {
  const c = editor.view.coordsAtPos(pos)
  return { left: c.left, top: c.bottom }
}

export function WikiSuggest({ editor }: { editor: Editor | null }) {
  const { t } = useTranslation()
  const [trigger, setTrigger] = useState<Trigger | null>(null)
  const [index, setIndex] = useState(0)
  const { data: articlesData } = useArticles()
  const { data: employeesData } = useEmployees()

  const detect = useCallback(() => {
    if (!editor || !editor.isFocused) {
      setTrigger(null)
      return
    }
    const { from, empty } = editor.state.selection
    if (!empty) {
      setTrigger(null)
      return
    }
    const before = editor.state.doc.textBetween(Math.max(0, from - 60), from, '\n')
    const link = before.match(/\[\[([^\]\n[]*)$/)
    const mention = before.match(/(?:^|\s)@([^\s@]*)$/)
    if (link) {
      setTrigger({ kind: 'article', query: link[1], from: from - link[0].length, to: from, ...caretCoords(editor, from) })
    } else if (mention) {
      const q = mention[1]
      setTrigger({ kind: 'user', query: q, from: from - q.length - 1, to: from, ...caretCoords(editor, from) })
    } else {
      setTrigger(null)
    }
  }, [editor])

  useEffect(() => {
    if (!editor) return
    detect()
    editor.on('selectionUpdate', detect)
    editor.on('update', detect)
    return () => {
      editor.off('selectionUpdate', detect)
      editor.off('update', detect)
    }
  }, [editor, detect])

  // Reset the highlighted row whenever the query or trigger kind changes.
  useEffect(() => {
    setIndex(0)
  }, [trigger?.kind, trigger?.query])

  const items: SuggestItem[] = useMemo(() => {
    if (!trigger) return []
    const q = trigger.query.toLowerCase()
    if (trigger.kind === 'article') {
      return (articlesData?.articles ?? [])
        .filter((a) => !q || a.title.toLowerCase().includes(q))
        .slice(0, MAX_ITEMS)
        .map((a) => ({ id: a.id, title: a.title, slug: a.slug }))
    }
    return (employeesData?.employees ?? [])
      .filter((e) => e.userId && (!q || (e.userName ?? '').toLowerCase().includes(q)))
      .slice(0, MAX_ITEMS)
      .map((e) => ({ id: e.userId, title: e.userName ?? '', sub: e.positionTitle ?? '' }))
  }, [trigger, articlesData, employeesData])

  const select = useCallback(
    (item: SuggestItem) => {
      if (!editor || !trigger) return
      const chain = editor.chain().focus().deleteRange({ from: trigger.from, to: trigger.to })
      if (trigger.kind === 'article') {
        chain.insertContent({ type: 'wikiLink', attrs: { articleId: item.id, slug: item.slug ?? null, label: item.title } })
      } else {
        chain.insertContent({ type: 'wikiMention', attrs: { userId: item.id, label: (item.title.split(' ')[0] || item.title) } })
      }
      chain.insertContent(' ').run()
      setTrigger(null)
    },
    [editor, trigger],
  )

  // Keyboard navigation — capture phase so ProseMirror doesn't also act on the keys.
  useEffect(() => {
    if (!trigger || items.length === 0) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        e.stopPropagation()
        setIndex((i) => Math.min(i + 1, items.length - 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        e.stopPropagation()
        setIndex((i) => Math.max(i - 1, 0))
      } else if (e.key === 'Enter') {
        e.preventDefault()
        e.stopPropagation()
        select(items[index])
      } else if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        setTrigger(null)
      }
    }
    document.addEventListener('keydown', onKey, true)
    return () => document.removeEventListener('keydown', onKey, true)
  }, [trigger, items, index, select])

  if (!trigger || items.length === 0) return null

  const Icon = trigger.kind === 'article' ? FileText : AtSign
  const left = Math.max(8, Math.min(trigger.left, window.innerWidth - 296))
  const top = Math.min(trigger.top + 4, window.innerHeight - 240)

  return createPortal(
    <div
      style={{ position: 'fixed', left, top, zIndex: 60 }}
      className="w-72 overflow-hidden rounded-lg border border-border bg-popover py-1 shadow-lg"
    >
      <p className="px-3 py-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
        {trigger.kind === 'article'
          ? t('wiki.suggest.articles', { defaultValue: 'Artikel verlinken' })
          : t('wiki.suggest.people', { defaultValue: 'Person erwähnen' })}
      </p>
      <div className="max-h-52 overflow-y-auto">
        {items.map((it, i) => (
          <button
            key={it.id}
            type="button"
            // mousedown + preventDefault keeps the editor focused so the insert lands.
            onMouseDown={(e) => {
              e.preventDefault()
              select(it)
            }}
            className={`flex w-full items-center gap-2.5 px-3 py-1.5 text-left text-sm transition-colors ${
              i === index ? 'bg-secondary' : 'hover:bg-secondary/50'
            }`}
          >
            <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
            <span className="min-w-0 flex-1">
              <span className="block truncate text-sm font-medium text-foreground">{it.title}</span>
              {it.sub && <span className="block truncate text-[11px] text-muted-foreground">{it.sub}</span>}
            </span>
          </button>
        ))}
      </div>
    </div>,
    document.body,
  )
}
