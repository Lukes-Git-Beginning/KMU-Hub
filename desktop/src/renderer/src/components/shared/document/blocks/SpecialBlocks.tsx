/**
 * Special block implementations (edit + read view) shared across document
 * surfaces — the structural elements beyond plain prose. Built once here,
 * registered à la carte via createSpecialBlockDefs({ only | omit }). Styling
 * mirrors the core blocks so a document reads as one coherent surface.
 */
import { useRef, useState, type ChangeEvent } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Check,
  ChevronRight,
  Code2,
  Copy,
  Download,
  File as FileIcon,
  FileArchive,
  FileImage,
  FileSpreadsheet,
  FileText,
  ListTree,
  Minus,
  Paperclip,
  Plus,
  Table2,
  Trash2,
  Upload,
  type LucideIcon,
} from 'lucide-react'
import DOMPurify from 'dompurify'
import { RichTextEditor } from '@/components/shared/RichTextEditor'
import { docUid } from '../types'
import { defineBlock, type BlockEditProps, type BlockTypeDef, type BlockViewProps } from '../block-registry'
import { CODE_LANGUAGES, highlightToReact } from './code-highlight'
import type { AttachmentBlock, CodeBlock, SimpleTableBlock, ToggleBlock } from './special-types'

/* ------------------------------ file helpers ------------------------------ */

function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return ''
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let i = 0
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i += 1
  }
  return `${value.toFixed(i === 0 ? 0 : value >= 10 ? 0 : 1)} ${units[i]}`
}

function fileIconFor(mime?: string, name?: string): LucideIcon {
  const ext = name?.split('.').pop()?.toLowerCase() ?? ''
  if (mime?.startsWith('image/')) return FileImage
  if (mime === 'application/pdf' || ext === 'pdf') return FileText
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)) return FileArchive
  if (['xls', 'xlsx', 'csv'].includes(ext) || mime?.includes('spreadsheet')) return FileSpreadsheet
  if (['doc', 'docx', 'txt', 'md', 'rtf'].includes(ext)) return FileText
  return FileIcon
}

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

/* ------------------------------ simple table ------------------------------ */

function SimpleTableEdit({ block, onPatch }: BlockEditProps<SimpleTableBlock>) {
  const { t } = useTranslation()
  const cells = block.cells
  const cols = cells[0]?.length ?? 0
  const hasHeader = block.hasHeader ?? true

  const setCell = (r: number, c: number, value: string) => {
    const next = cells.map((row) => row.slice())
    next[r][c] = value
    onPatch({ cells: next })
  }
  const addRow = () => onPatch({ cells: [...cells.map((r) => r.slice()), Array(cols).fill('')] })
  const removeRow = (r: number) => onPatch({ cells: cells.filter((_, i) => i !== r) })
  const addCol = () => onPatch({ cells: cells.map((r) => [...r, '']) })
  const removeCol = (c: number) => onPatch({ cells: cells.map((r) => r.filter((_, j) => j !== c)) })

  const cellCls =
    'min-w-0 flex-1 rounded-md border border-border bg-card px-2 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'

  return (
    <div className="space-y-2 rounded-xl border border-border bg-card p-3">
      <label className="flex w-fit cursor-pointer items-center gap-1.5 text-[11px] text-muted-foreground">
        <input
          type="checkbox"
          checked={hasHeader}
          onChange={(e) => onPatch({ hasHeader: e.target.checked })}
          className="h-3.5 w-3.5 accent-primary"
        />
        {t('document.block.simpletable.header', { defaultValue: 'Kopfzeile' })}
      </label>

      {/* Per-column remove strip (aligned with the cells + trailing row-remove slot). */}
      {cols > 1 && (
        <div className="flex items-center gap-1">
          {Array.from({ length: cols }).map((_, c) => (
            <button
              key={c}
              type="button"
              onClick={() => removeCol(c)}
              className="flex flex-1 items-center justify-center rounded-md py-0.5 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
              aria-label={t('document.block.simpletable.removeColumn', { defaultValue: 'Spalte entfernen' })}
            >
              <Minus className="h-3 w-3" />
            </button>
          ))}
          <span className="w-7 shrink-0" />
        </div>
      )}

      <div className="space-y-1">
        {cells.map((row, r) => (
          <div key={r} className="flex items-center gap-1">
            {row.map((cell, c) => (
              <input
                key={c}
                value={cell}
                onChange={(e) => setCell(r, c, e.target.value)}
                placeholder={
                  r === 0 && hasHeader
                    ? t('document.block.simpletable.headerCell', { defaultValue: 'Spaltentitel' })
                    : t('document.block.simpletable.cell', { defaultValue: 'Zelle' })
                }
                className={`${cellCls} ${r === 0 && hasHeader ? 'bg-secondary/40 font-semibold' : ''}`}
              />
            ))}
            <button
              type="button"
              onClick={() => removeRow(r)}
              disabled={cells.length <= 1}
              className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-muted-foreground"
              aria-label={t('document.block.simpletable.removeRow', { defaultValue: 'Zeile entfernen' })}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        ))}
      </div>

      <div className="flex items-center gap-2 pt-0.5">
        <button
          type="button"
          onClick={addRow}
          className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
        >
          <Plus className="h-3 w-3" />
          {t('document.block.simpletable.addRow', { defaultValue: 'Zeile' })}
        </button>
        <button
          type="button"
          onClick={addCol}
          className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
        >
          <Plus className="h-3 w-3" />
          {t('document.block.simpletable.addColumn', { defaultValue: 'Spalte' })}
        </button>
      </div>
    </div>
  )
}

function SimpleTableView({ block }: BlockViewProps<SimpleTableBlock>) {
  const cells = block.cells
  if (!cells.length || !cells[0]?.length) return null
  const hasHeader = block.hasHeader ?? true
  const header = hasHeader ? cells[0] : null
  const bodyRows = hasHeader ? cells.slice(1) : cells
  return (
    <div className="report-keep overflow-x-auto rounded-xl border border-border-muted">
      <table className="w-full border-collapse text-sm">
        {header && (
          <thead>
            <tr>
              {header.map((cell, i) => (
                <th
                  key={i}
                  className="border-b border-border bg-secondary/50 px-3 py-2 text-left font-semibold text-foreground"
                >
                  {cell}
                </th>
              ))}
            </tr>
          </thead>
        )}
        <tbody>
          {bodyRows.map((row, r) => (
            <tr key={r} className="border-b border-border-muted last:border-0">
              {row.map((cell, i) => (
                <td key={i} className="px-3 py-2 align-top text-foreground">
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/* ------------------------------- attachment ------------------------------- */

function AttachmentEdit({ block, onPatch }: BlockEditProps<AttachmentBlock>) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)
  const pick = () => fileRef.current?.click()
  const onFile = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () =>
      onPatch({ url: String(reader.result), name: block.name || file.name, size: file.size, mime: file.type })
    reader.readAsDataURL(file)
    e.target.value = ''
  }
  const Icon = fileIconFor(block.mime, block.name)
  return (
    <div className="rounded-xl border border-border bg-card p-3">
      {block.url ? (
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
            <Icon className="h-5 w-5" />
          </div>
          <div className="min-w-0 flex-1">
            <input
              value={block.name}
              onChange={(e) => onPatch({ name: e.target.value })}
              placeholder={t('document.block.attachment.namePlaceholder', { defaultValue: 'Dateiname' })}
              className="w-full rounded-md border border-transparent bg-transparent px-1.5 py-0.5 text-sm font-medium text-foreground hover:border-border focus:border-border focus:outline-none"
            />
            <p className="px-1.5 text-xs text-muted-foreground">{formatBytes(block.size)}</p>
          </div>
          <button
            type="button"
            onClick={pick}
            className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <Upload className="h-3 w-3" />
            {t('document.block.attachment.replace', { defaultValue: 'Ersetzen' })}
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={pick}
          className="flex h-24 w-full flex-col items-center justify-center gap-1.5 rounded-lg border border-dashed border-border bg-secondary/30 text-muted-foreground transition-colors hover:border-primary/40 hover:bg-secondary/50"
        >
          <Paperclip className="h-6 w-6 text-muted-foreground/60" />
          <span className="text-xs font-medium">
            {t('document.block.attachment.pick', { defaultValue: 'Datei auswählen' })}
          </span>
          <span className="text-[11px] text-muted-foreground/70">
            {t('document.block.attachment.pickHint', { defaultValue: 'Aus deinen Dateien' })}
          </span>
        </button>
      )}
      <input ref={fileRef} type="file" className="hidden" onChange={onFile} />
    </div>
  )
}

function AttachmentView({ block }: BlockViewProps<AttachmentBlock>) {
  const { t } = useTranslation()
  if (!block.url) return null
  const Icon = fileIconFor(block.mime, block.name)
  return (
    <a
      href={block.url}
      download={block.name || 'download'}
      className="report-keep flex items-center gap-3 rounded-xl border border-border-muted bg-card p-3 no-underline transition-colors hover:border-primary/40 hover:bg-secondary/40"
    >
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
        <Icon className="h-5 w-5" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-foreground">
          {block.name || t('document.block.attachment.untitled', { defaultValue: 'Anhang' })}
        </p>
        {block.size ? <p className="text-xs text-muted-foreground">{formatBytes(block.size)}</p> : null}
      </div>
      <Download className="h-4 w-4 shrink-0 text-muted-foreground" aria-label={t('document.block.attachment.download', { defaultValue: 'Herunterladen' })} />
    </a>
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
  simpletable: defineBlock<SimpleTableBlock>({
    type: 'simpletable',
    icon: Table2,
    labelKey: 'document.block.simpletable.label',
    group: 'content',
    atomic: true,
    makeDefault: () => ({
      id: docUid('b'),
      type: 'simpletable',
      hasHeader: true,
      cells: [
        ['Spalte 1', 'Spalte 2'],
        ['', ''],
      ],
    }),
    Edit: SimpleTableEdit,
    View: SimpleTableView,
  }),
  attachment: defineBlock<AttachmentBlock>({
    type: 'attachment',
    icon: Paperclip,
    labelKey: 'document.block.attachment.label',
    group: 'content',
    atomic: true,
    makeDefault: () => ({ id: docUid('b'), type: 'attachment', name: '' }),
    Edit: AttachmentEdit,
    View: AttachmentView,
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
