/**
 * Editor identity bar (WP-3) — lets the writer set a cover + icon for an article
 * straight from the editor head, Notion-style. Both are controlled values stored
 * on the article and saved with the rest of the edit.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Smile, Image as ImageIcon, X } from 'lucide-react'
import { cn } from '@/lib'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import {
  WIKI_ICON_MAP,
  WIKI_ICON_NAMES,
  WIKI_COVERS,
  WikiArticleIcon,
  coverBackground,
} from './wikiIdentity'

interface WikiIdentityBarProps {
  title: string
  icon?: string
  coverUrl?: string
  onIconChange: (icon: string | undefined) => void
  onCoverChange: (cover: string | undefined) => void
  /** Width class applied to the icon row so it lines up with the title. */
  widthClass: string
}

// ---------------------------------------------------------------------------
// Icon picker
// ---------------------------------------------------------------------------

function IconPicker({
  value,
  onChange,
  children,
}: {
  value?: string
  onChange: (icon: string | undefined) => void
  children: React.ReactNode
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent className="w-64 p-2.5" align="start">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
            {t('wiki.identity.iconLabel')}
          </span>
          {value && (
            <button
              type="button"
              onClick={() => { onChange(undefined); setOpen(false) }}
              className="text-[11px] text-muted-foreground transition-colors hover:text-destructive"
            >
              {t('wiki.identity.remove')}
            </button>
          )}
        </div>
        <div className="grid grid-cols-6 gap-1">
          {WIKI_ICON_NAMES.map((name) => {
            const Icon = WIKI_ICON_MAP[name]
            const active = value === name
            return (
              <button
                key={name}
                type="button"
                onClick={() => { onChange(name); setOpen(false) }}
                className={cn(
                  'flex h-8 w-8 items-center justify-center rounded-md transition-colors',
                  active ? 'bg-primary/15 text-primary' : 'text-muted-foreground hover:bg-secondary',
                )}
                aria-label={name}
              >
                <Icon className="h-4 w-4" />
              </button>
            )
          })}
        </div>
      </PopoverContent>
    </Popover>
  )
}

// ---------------------------------------------------------------------------
// Cover picker
// ---------------------------------------------------------------------------

function CoverPicker({
  value,
  onChange,
  children,
}: {
  value?: string
  onChange: (cover: string | undefined) => void
  children: React.ReactNode
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [url, setUrl] = useState('')
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent className="w-72 p-2.5" align="end">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
            {t('wiki.identity.coverLabel')}
          </span>
          {value && (
            <button
              type="button"
              onClick={() => { onChange(undefined); setOpen(false) }}
              className="text-[11px] text-muted-foreground transition-colors hover:text-destructive"
            >
              {t('wiki.identity.remove')}
            </button>
          )}
        </div>
        <div className="grid grid-cols-4 gap-1.5">
          {WIKI_COVERS.map((cover) => {
            const active = value === `grad:${cover.id}`
            return (
              <button
                key={cover.id}
                type="button"
                onClick={() => { onChange(`grad:${cover.id}`); setOpen(false) }}
                style={{ background: cover.gradient }}
                className={cn(
                  'h-10 rounded-md ring-offset-1 ring-offset-popover transition-all',
                  active ? 'ring-2 ring-primary' : 'hover:ring-2 hover:ring-border',
                )}
                aria-label={cover.id}
              />
            )
          })}
        </div>
        <div className="mt-2.5 flex items-center gap-1.5 border-t border-border-muted pt-2.5">
          <input
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && url.trim()) { onChange(url.trim()); setOpen(false) }
            }}
            placeholder={t('wiki.identity.coverUrlPlaceholder')}
            className="h-7 min-w-0 flex-1 rounded-md border border-border bg-transparent px-2 text-xs outline-none focus:border-primary"
          />
          <button
            type="button"
            onClick={() => url.trim() && (onChange(url.trim()), setOpen(false))}
            disabled={!url.trim()}
            className="h-7 shrink-0 rounded-md bg-primary px-2.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
          >
            {t('wiki.identity.apply')}
          </button>
        </div>
      </PopoverContent>
    </Popover>
  )
}

// ---------------------------------------------------------------------------
// Bar
// ---------------------------------------------------------------------------

const chipCls =
  'flex items-center gap-1.5 rounded-md border border-border bg-card/80 px-2 py-1 text-[11px] text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground'

export function WikiIdentityBar({
  title,
  icon,
  coverUrl,
  onIconChange,
  onCoverChange,
  widthClass,
}: WikiIdentityBarProps) {
  const { t } = useTranslation()
  const coverBg = coverBackground(coverUrl)

  return (
    <>
      {/* Cover banner */}
      {coverBg && (
        <div
          className="group relative mb-1 h-36 w-full overflow-hidden rounded-xl"
          style={{ background: coverBg }}
        >
          <div className="absolute right-3 top-3 flex gap-1.5 opacity-0 transition-opacity group-hover:opacity-100">
            <CoverPicker value={coverUrl} onChange={onCoverChange}>
              <button type="button" className="rounded-md bg-card/90 px-2 py-1 text-[11px] font-medium text-foreground shadow-sm transition-colors hover:bg-card">
                {t('wiki.identity.changeCover')}
              </button>
            </CoverPicker>
            <button
              type="button"
              onClick={() => onCoverChange(undefined)}
              className="flex h-6 w-6 items-center justify-center rounded-md bg-card/90 text-muted-foreground shadow-sm transition-colors hover:text-destructive"
              aria-label={t('wiki.identity.removeCover')}
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}

      <div className={widthClass}>
        {/* Icon (overlaps the cover when present) */}
        {icon && (
          <div className={cn(coverBg ? '-mt-9 mb-2' : 'mb-2')}>
            <IconPicker value={icon} onChange={onIconChange}>
              <button type="button" className="rounded-xl transition-transform hover:scale-[1.03]" aria-label={t('wiki.identity.iconLabel')}>
                <WikiArticleIcon
                  icon={icon}
                  title={title}
                  size="lg"
                  className={coverBg ? 'ring-4 ring-background' : ''}
                />
              </button>
            </IconPicker>
          </div>
        )}

        {/* Add controls — only the ones not yet set */}
        {(!icon || !coverBg) && (
          <div className="mb-3 flex items-center gap-1.5">
            {!icon && (
              <IconPicker value={icon} onChange={onIconChange}>
                <button type="button" className={chipCls}>
                  <Smile className="h-3.5 w-3.5" />
                  {t('wiki.identity.addIcon')}
                </button>
              </IconPicker>
            )}
            {!coverBg && (
              <CoverPicker value={coverUrl} onChange={onCoverChange}>
                <button type="button" className={chipCls}>
                  <ImageIcon className="h-3.5 w-3.5" />
                  {t('wiki.identity.addCover')}
                </button>
              </CoverPicker>
            )}
          </div>
        )}
      </div>
    </>
  )
}
