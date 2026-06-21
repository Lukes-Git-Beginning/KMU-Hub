/**
 * Table of contents (WP-4) — sticky outline of the article's H1–H3 with anchor
 * jump and scroll-spy. Observes the rendered headings inside the scroll
 * container and highlights the section currently in view.
 */
import { useEffect, useState, type RefObject } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib'
import type { TocHeading } from './wikiReading'

interface WikiTocProps {
  headings: TocHeading[]
  /** The article's scroll container — observer root + scroll target. */
  scrollRef: RefObject<HTMLElement | null>
}

export function WikiToc({ headings, scrollRef }: WikiTocProps) {
  const { t } = useTranslation()
  const [activeId, setActiveId] = useState('')

  useEffect(() => {
    const root = scrollRef.current
    if (!root || headings.length === 0) return

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
        if (visible.length > 0) setActiveId(visible[0].target.id)
      },
      { root, rootMargin: '0px 0px -70% 0px', threshold: 0 },
    )

    headings.forEach((h) => {
      const el = document.getElementById(h.id)
      if (el) observer.observe(el)
    })
    return () => observer.disconnect()
  }, [headings, scrollRef])

  const jump = (id: string) => {
    const el = document.getElementById(id)
    const root = scrollRef.current
    if (!el || !root) return
    const top = el.getBoundingClientRect().top - root.getBoundingClientRect().top + root.scrollTop - 12
    root.scrollTo({ top, behavior: 'smooth' })
    setActiveId(id)
  }

  if (headings.length < 2) return null

  return (
    <nav aria-label={t('wiki.toc.title')} className="text-sm">
      <p className="mb-2 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
        {t('wiki.toc.title')}
      </p>
      <ul className="space-y-0.5 border-l border-border">
        {headings.map((h) => {
          const active = h.id === activeId
          return (
            <li key={h.id}>
              <button
                type="button"
                onClick={() => jump(h.id)}
                style={{ paddingLeft: `${(h.level - 1) * 0.75 + 0.75}rem` }}
                className={cn(
                  '-ml-px block w-full border-l-2 py-1 pr-2 text-left text-[13px] leading-snug transition-colors',
                  active
                    ? 'border-primary font-medium text-primary'
                    : 'border-transparent text-muted-foreground hover:border-border hover:text-foreground',
                )}
              >
                <span className="line-clamp-2">{h.text}</span>
              </button>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
