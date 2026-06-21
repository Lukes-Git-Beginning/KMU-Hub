/**
 * DocumentReader — shared read-mode renderer. Two modes:
 *  - 'print': A4 sheet stack (berichte premium report look), paginated on
 *    page-break blocks; full-page blocks (cover) get their own sheet.
 *  - 'web':   a single flowing column (wiki article look), no pagination.
 *
 * Block bodies are rendered via the registry's View component, so every module
 * decides how its blocks look while sharing the page/flow chrome.
 */
import { useTranslation } from 'react-i18next'
import { FileText } from 'lucide-react'
import { isDocEmpty, type DocRow } from './types'
import type { BlockRegistry } from './block-registry'

export interface DocReaderSettings {
  showHeader?: boolean
  headerText?: string
  showFooter?: boolean
  showPageNumbers?: boolean
  logoUrl?: string
  palette?: string
  accentColor?: string
}

export interface DocumentReaderProps {
  rows: DocRow[]
  registry: BlockRegistry
  mode?: 'print' | 'web'
  settings?: DocReaderSettings
  /** Block type that forces a page split in print mode. Default 'pagebreak'. */
  pageBreakType?: string
  /** Block types that stand alone on their own sheet in print mode (e.g. 'cover'). */
  fullPageTypes?: string[]
}

export function DocumentReader({
  rows,
  registry,
  mode = 'web',
  settings,
  pageBreakType = 'pagebreak',
  fullPageTypes = [],
}: DocumentReaderProps) {
  const { t } = useTranslation()

  if (isDocEmpty(rows)) {
    return (
      <div className="mx-auto flex max-w-3xl flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border bg-card px-10 py-20 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-secondary text-muted-foreground">
          <FileText className="h-6 w-6" />
        </div>
        <p className="text-sm font-medium text-foreground">
          {t('document.reader.emptyTitle', { defaultValue: 'Noch kein Inhalt' })}
        </p>
        <p className="max-w-xs text-xs text-muted-foreground">
          {t('document.reader.emptyHint', { defaultValue: 'Zum Bearbeiten wechseln, um Inhalt hinzuzufügen.' })}
        </p>
      </div>
    )
  }

  if (mode === 'web') {
    return (
      <div className="mx-auto max-w-[var(--doc-measure,72ch)] space-y-6">
        {rows.map((row) => (
          <RowBlocks key={row.id} row={row} registry={registry} settings={settings} gap="gap-6" />
        ))}
      </div>
    )
  }

  const fullPage = new Set(fullPageTypes)
  const pages = paginate(rows, pageBreakType, fullPage)

  return (
    <div className="report-desk -mx-6 -my-6 min-h-full bg-secondary/50 px-6 py-10">
      <div className="mx-auto flex flex-col items-center gap-8">
        {pages.map((pageRows, i) => (
          <ReportPage
            key={i}
            rows={pageRows}
            registry={registry}
            settings={settings}
            fullPage={fullPage}
            pageIndex={i}
            pageCount={pages.length}
          />
        ))}
      </div>
    </div>
  )
}

/** Renders one row's columns + blocks via the registry's View components. */
function RowBlocks({
  row,
  registry,
  settings,
  gap,
}: {
  row: DocRow
  registry: BlockRegistry
  settings?: DocReaderSettings
  gap: string
}) {
  return (
    <div className={row.columns.length > 1 ? `flex flex-wrap ${gap}` : undefined}>
      {row.columns.map((col) => (
        <div key={col.id} className="min-w-0 space-y-6" style={{ flex: col.width ?? 1 }}>
          {col.blocks.map((block) => {
            const def = registry[block.type]
            if (!def) return null
            const View = def.View
            return (
              <View
                key={block.id}
                block={block}
                accent={settings?.accentColor}
                palette={settings?.palette}
              />
            )
          })}
        </div>
      ))}
    </div>
  )
}

/* -------------------------------- print mode ------------------------------ */

function isPageBreakRow(row: DocRow, pageBreakType: string): boolean {
  return (
    row.columns.length === 1 &&
    row.columns[0].blocks.length === 1 &&
    row.columns[0].blocks[0].type === pageBreakType
  )
}

function rowHasFullPage(row: DocRow, fullPage: Set<string>): boolean {
  return row.columns.some((col) => col.blocks.some((b) => fullPage.has(b.type)))
}

function paginate(rows: DocRow[], pageBreakType: string, fullPage: Set<string>): DocRow[][] {
  const pages: DocRow[][] = [[]]
  for (const row of rows) {
    if (isPageBreakRow(row, pageBreakType)) {
      pages.push([])
      continue
    }
    pages[pages.length - 1].push(row)
    if (rowHasFullPage(row, fullPage)) pages.push([])
  }
  return pages.filter((page) => page.length > 0)
}

function ReportPage({
  rows,
  registry,
  settings,
  fullPage,
  pageIndex,
  pageCount,
}: {
  rows: DocRow[]
  registry: BlockRegistry
  settings?: DocReaderSettings
  fullPage: Set<string>
  pageIndex: number
  pageCount: number
}) {
  const { t } = useTranslation()
  const accent = settings?.accentColor
  const hasFullPage = Boolean(rows[0]?.columns[0]?.blocks[0] && fullPage.has(rows[0].columns[0].blocks[0].type))
  const showHeader = !hasFullPage && settings?.showHeader && Boolean(settings.headerText || settings.logoUrl)
  const showFooter = !hasFullPage && (settings?.showFooter || settings?.showPageNumbers)

  return (
    <article
      className="report-page relative flex w-[794px] max-w-full flex-col rounded-[2px] bg-card shadow-[0_2px_4px_-1px_rgba(15,23,42,0.06),0_22px_48px_-20px_rgba(15,23,42,0.24)]"
      style={{ minHeight: 1123 }}
    >
      {accent ? (
        <div className="h-1 w-full rounded-t-[2px]" style={{ backgroundColor: accent }} />
      ) : (
        <div className="h-1 w-full rounded-t-[2px] bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)]" />
      )}

      {showHeader && (
        <header className="flex items-center justify-between gap-4 border-b border-border-muted px-[72px] pb-4 pt-7 text-xs text-muted-foreground">
          <span className="truncate">{settings?.headerText}</span>
          {settings?.logoUrl && <img src={settings.logoUrl} alt="" className="h-6 shrink-0 object-contain" />}
        </header>
      )}

      <div className="flex-1 px-[72px] py-[56px]">
        <div className="space-y-7">
          {rows.map((row) => (
            <RowBlocks key={row.id} row={row} registry={registry} settings={settings} gap="gap-6" />
          ))}
        </div>
      </div>

      {showFooter && (
        <footer className="mt-auto flex items-center justify-between gap-4 border-t border-border-muted px-[72px] pb-7 pt-4 text-[11px] text-muted-foreground">
          <span className="truncate">{settings?.showFooter ? settings?.headerText : ''}</span>
          {settings?.showPageNumbers && (
            <span className="shrink-0 tabular-nums">
              {t('document.reader.pageXofY', { defaultValue: 'Seite {x} von {y}', x: pageIndex + 1, y: pageCount })}
            </span>
          )}
        </footer>
      )}
    </article>
  )
}
