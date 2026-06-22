/**
 * Special block implementations (edit + read view) shared across document
 * surfaces — the structural elements beyond plain prose. Built once here,
 * registered à la carte via createSpecialBlockDefs({ only | omit }). Styling
 * mirrors the core blocks so a document reads as one coherent surface.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronRight, ListTree } from 'lucide-react'
import DOMPurify from 'dompurify'
import { RichTextEditor } from '@/components/shared/RichTextEditor'
import { docUid } from '../types'
import { defineBlock, type BlockEditProps, type BlockTypeDef, type BlockViewProps } from '../block-registry'
import type { ToggleBlock } from './special-types'

/* --------------------------------- toggle --------------------------------- */

function ToggleEdit({ block, onPatch }: BlockEditProps<ToggleBlock>) {
  const { t } = useTranslation()
  return (
    <div className="rounded-xl border border-border bg-card">
      <div className="flex items-center gap-2 border-b border-border-muted px-3 py-2">
        <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
        <input
          value={block.title}
          onChange={(e) => onPatch({ title: e.target.value })}
          placeholder={t('document.block.toggle.titlePlaceholder', { defaultValue: 'Aufklapp-Titel' })}
          className="min-w-0 flex-1 rounded-md border border-transparent bg-transparent px-1.5 py-1 text-sm font-medium text-foreground hover:border-border focus:border-border focus:outline-none"
        />
        <label className="flex shrink-0 cursor-pointer items-center gap-1.5 text-[11px] text-muted-foreground">
          <input
            type="checkbox"
            checked={block.open ?? false}
            onChange={(e) => onPatch({ open: e.target.checked })}
            className="h-3.5 w-3.5 accent-primary"
          />
          {t('document.block.toggle.defaultOpen', { defaultValue: 'Offen anzeigen' })}
        </label>
      </div>
      <div className="px-3 py-2">
        <RichTextEditor
          content={block.html}
          onChange={(html) => onPatch({ html })}
          compact
          placeholder={t('document.block.toggle.bodyPlaceholder', { defaultValue: 'Inhalt, der sich aufklappt…' })}
        />
      </div>
    </div>
  )
}

function ToggleView({ block }: BlockViewProps<ToggleBlock>) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(block.open ?? false)
  return (
    <div className="report-keep overflow-hidden rounded-xl border border-border-muted bg-card">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-full items-center gap-2 px-3 py-2.5 text-left text-sm font-medium text-foreground transition-colors hover:bg-secondary/50"
      >
        <ChevronRight
          className={`h-4 w-4 shrink-0 text-muted-foreground transition-transform ${open ? 'rotate-90' : ''}`}
          aria-hidden="true"
        />
        <span className="min-w-0 flex-1">
          {block.title || t('document.block.toggle.untitled', { defaultValue: 'Aufklappen' })}
        </span>
      </button>
      {open && (
        <div
          className="prose prose-sm max-w-[60ch] border-t border-border-muted px-3 py-3 pl-9 text-foreground dark:prose-invert"
          dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(block.html) }}
        />
      )}
    </div>
  )
}

/* ------------------------------ registry defs ----------------------------- */

/**
 * Every special block definition, keyed by type, so a surface can pick exactly
 * the ones it needs. Pass `only` to whitelist (the common case — "wiki wants
 * toggle + code + …") or `omit` to drop a few from the full set.
 */
const SPECIAL_DEFS: Record<string, BlockTypeDef> = {
  toggle: defineBlock<ToggleBlock>({
    type: 'toggle',
    icon: ListTree,
    labelKey: 'document.block.toggle.label',
    group: 'content',
    makeDefault: () => ({ id: docUid('b'), type: 'toggle', title: '', html: '', open: false }),
    Edit: ToggleEdit,
    View: ToggleView,
  }),
}

export interface SpecialBlockOptions {
  /** Whitelist: only these types, in this order. */
  only?: string[]
  /** Blacklist: every special block except these. */
  omit?: string[]
}

export function createSpecialBlockDefs(options?: SpecialBlockOptions): BlockTypeDef[] {
  if (options?.only) {
    return options.only.map((type) => SPECIAL_DEFS[type]).filter(Boolean)
  }
  const omit = new Set(options?.omit ?? [])
  return Object.values(SPECIAL_DEFS).filter((def) => !omit.has(def.type))
}
