/**
 * Special block implementations (edit + read view) shared across document
 * surfaces — the structural elements beyond plain prose. Built once here,
 * registered à la carte via createSpecialBlockDefs({ only | omit }). Styling
 * mirrors the core blocks so a document reads as one coherent surface.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, ChevronRight, Code2, Copy, ListTree } from 'lucide-react'
import DOMPurify from 'dompurify'
import { RichTextEditor } from '@/components/shared/RichTextEditor'
import { docUid } from '../types'
import { defineBlock, type BlockEditProps, type BlockTypeDef, type BlockViewProps } from '../block-registry'
import { CODE_LANGUAGES, highlightToReact } from './code-highlight'
import type { CodeBlock, ToggleBlock } from './special-types'

/** Copy-to-clipboard button with a brief "copied" confirmation. */
function CopyButton({ value, className }: { value: string; className?: string }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1600)
    } catch {
      // Clipboard unavailable (e.g. headless) — silently no-op.
    }
  }
  return (
    <button
      type="button"
      onClick={copy}
      className={
        className ??
        'flex items-center gap-1 rounded-md px-2 py-1 text-[11px] text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground'
      }
      aria-label={t('document.block.code.copy', { defaultValue: 'Kopieren' })}
    >
      {copied ? <Check className="h-3 w-3 text-success" /> : <Copy className="h-3 w-3" />}
      {copied
        ? t('document.block.code.copied', { defaultValue: 'Kopiert' })
        : t('document.block.code.copy', { defaultValue: 'Kopieren' })}
    </button>
  )
}

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

/* ---------------------------------- code ---------------------------------- */

function CodeEdit({ block, onPatch }: BlockEditProps<CodeBlock>) {
  const { t } = useTranslation()
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-[var(--secondary)]/40">
      <div className="flex items-center justify-between gap-2 border-b border-border-muted px-3 py-1.5">
        <select
          value={block.language}
          onChange={(e) => onPatch({ language: e.target.value })}
          aria-label={t('document.block.code.language', { defaultValue: 'Sprache' })}
          className="rounded-md border border-border bg-card px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
        >
          {CODE_LANGUAGES.map((lang) => (
            <option key={lang.value} value={lang.value}>
              {lang.label}
            </option>
          ))}
        </select>
        {block.code.trim().length > 0 && <CopyButton value={block.code} />}
      </div>
      <textarea
        value={block.code}
        onChange={(e) => onPatch({ code: e.target.value })}
        spellCheck={false}
        rows={Math.min(Math.max(block.code.split('\n').length, 3), 20)}
        placeholder={t('document.block.code.placeholder', { defaultValue: 'Code einfügen…' })}
        className="block w-full resize-y bg-transparent px-3 py-2.5 font-mono text-[13px] leading-relaxed text-foreground placeholder:text-muted-foreground/50 focus:outline-none"
        style={{ fontFamily: "'JetBrains Mono', 'Geist Mono', monospace" }}
      />
    </div>
  )
}

function CodeView({ block }: BlockViewProps<CodeBlock>) {
  const lang = CODE_LANGUAGES.find((l) => l.value === block.language)
  if (!block.code.trim()) return null
  return (
    <div className="report-keep overflow-hidden rounded-xl border border-border-muted bg-[var(--secondary)]/40">
      <div className="flex items-center justify-between gap-2 border-b border-border-muted px-3 py-1.5">
        <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          {lang?.label ?? block.language}
        </span>
        <CopyButton value={block.code} />
      </div>
      <pre className="overflow-x-auto px-3 py-2.5 text-[13px] leading-relaxed" style={{ fontFamily: "'JetBrains Mono', 'Geist Mono', monospace" }}>
        <code className="hljs bg-transparent p-0 text-foreground">{highlightToReact(block.code, block.language)}</code>
      </pre>
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
  code: defineBlock<CodeBlock>({
    type: 'code',
    icon: Code2,
    labelKey: 'document.block.code.label',
    group: 'content',
    atomic: true,
    makeDefault: () => ({ id: docUid('b'), type: 'code', code: '', language: 'plaintext' }),
    Edit: CodeEdit,
    View: CodeView,
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
