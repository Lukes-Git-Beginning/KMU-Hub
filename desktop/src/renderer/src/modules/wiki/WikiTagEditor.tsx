import { useState, useRef, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Tag, X, Plus } from 'lucide-react'
import { useUpdateArticle } from '@/api/hooks/useWiki'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiTagEditorProps {
  articleId: string
  tags: string[]
  /** All tags across articles — drives the add-suggestions. */
  suggestions: string[]
}

/**
 * Inline tag chip editor for the article header. Add via a small input with
 * prefix suggestions from existing tags, remove via the chip's ✕. Each change
 * persists the full tag array through useUpdateArticle (real MSW field).
 */
export function WikiTagEditor({ articleId, tags, suggestions }: WikiTagEditorProps) {
  const { t } = useTranslation()
  const updateArticle = useUpdateArticle()
  const [adding, setAdding] = useState(false)
  const [input, setInput] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const matches = useMemo(() => {
    const q = input.trim().toLowerCase()
    return suggestions
      .filter((s) => !tags.some((x) => x.toLowerCase() === s.toLowerCase()))
      .filter((s) => (q ? s.toLowerCase().includes(q) : true))
      .slice(0, 6)
  }, [input, suggestions, tags])

  const addTag = async (raw: string) => {
    const tag = raw.trim()
    if (!tag) return
    if (tags.some((x) => x.toLowerCase() === tag.toLowerCase())) {
      setInput('')
      return
    }
    try {
      await updateArticle.mutateAsync({ id: articleId, tags: [...tags, tag] })
      toast.success(t('wiki.tags.added'))
      setInput('')
      inputRef.current?.focus()
    } catch {
      toast.error(t('wiki.tags.updateError'))
    }
  }

  const removeTag = async (tag: string) => {
    try {
      await updateArticle.mutateAsync({ id: articleId, tags: tags.filter((x) => x !== tag) })
      toast.success(t('wiki.tags.removed'))
    } catch {
      toast.error(t('wiki.tags.updateError'))
    }
  }

  return (
    <div className="mt-2 flex flex-wrap items-center gap-1.5">
      {tags.map((tag) => (
        <span
          key={tag}
          className="group/tag inline-flex items-center gap-1 rounded-full bg-secondary px-2.5 py-0.5 text-xs text-muted-foreground"
        >
          <Tag className="h-3 w-3" />
          {tag}
          <button
            onClick={() => removeTag(tag)}
            disabled={updateArticle.isPending}
            className="ml-0.5 rounded-full p-0.5 text-muted-foreground/60 hover:bg-background hover:text-destructive transition-colors disabled:opacity-50"
            title={t('wiki.tags.remove')}
            aria-label={t('wiki.tags.remove')}
          >
            <X className="h-2.5 w-2.5" />
          </button>
        </span>
      ))}

      {adding ? (
        <div className="relative">
          <input
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') addTag(input)
              if (e.key === 'Escape') {
                setAdding(false)
                setInput('')
              }
            }}
            onBlur={() => {
              // Allow a suggestion click to register before closing.
              setTimeout(() => {
                setAdding(false)
                setInput('')
              }, 150)
            }}
            autoFocus
            placeholder={t('wiki.tags.placeholder')}
            aria-label={t('wiki.tags.add')}
            className="h-6 w-32 rounded-full border border-primary/50 bg-background px-2.5 text-xs outline-none"
          />
          {matches.length > 0 && (
            <div className="absolute left-0 top-7 z-20 min-w-32 overflow-hidden rounded-md border border-border bg-popover py-1 shadow-md">
              {matches.map((s) => (
                <button
                  key={s}
                  // onMouseDown fires before the input's onBlur, so the add still runs.
                  onMouseDown={(e) => {
                    e.preventDefault()
                    addTag(s)
                  }}
                  className="flex w-full items-center gap-1.5 px-2.5 py-1 text-left text-xs text-foreground hover:bg-accent"
                >
                  <Tag className="h-3 w-3 text-muted-foreground" />
                  {s}
                </button>
              ))}
            </div>
          )}
        </div>
      ) : (
        <button
          onClick={() => setAdding(true)}
          className="inline-flex items-center gap-1 rounded-full border border-dashed border-border px-2.5 py-0.5 text-xs text-muted-foreground hover:border-primary/50 hover:text-foreground transition-colors"
        >
          <Plus className="h-3 w-3" />
          {t('wiki.tags.add')}
        </button>
      )}
    </div>
  )
}
