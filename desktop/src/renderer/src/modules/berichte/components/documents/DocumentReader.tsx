import { useTranslation } from 'react-i18next'
import { FileText, TrendingDown, TrendingUp } from 'lucide-react'
import DOMPurify from 'dompurify'
import type {
  CalloutVariant,
  ChartBlock,
  KpiBlock,
  ReportBlock,
  ReportDocSettings,
  ReportRow,
  TableBlock,
} from '@/api/berichte-types'
import { useReportResult } from '@/api/hooks/useBerichte'
import { Skeleton } from '@/components/shared'
import { formatDate } from '@/lib/format'
import { ChartRenderer } from '../charts/ChartRenderer'

const CALLOUT_STYLE: Record<CalloutVariant, string> = {
  info: 'border-info/30 bg-info-light',
  success: 'border-success/30 bg-success-light',
  warning: 'border-warning/30 bg-warning-light',
  recommendation: 'border-primary/30 bg-primary-light',
}

/** A row that is purely a page break — used as a sheet separator in read mode. */
function isPageBreakRow(row: ReportRow): boolean {
  return (
    row.columns.length === 1 &&
    row.columns[0].blocks.length === 1 &&
    row.columns[0].blocks[0].type === 'pagebreak'
  )
}

/** Split the document into sheets on dedicated page-break rows. */
function paginate(rows: ReportRow[]): ReportRow[][] {
  const pages: ReportRow[][] = [[]]
  for (const row of rows) {
    if (isPageBreakRow(row)) pages.push([])
    else pages[pages.length - 1].push(row)
  }
  return pages.filter((page) => page.length > 0)
}

/**
 * Renders a report document as a stack of A4 sheets on a tinted desk — the
 * premium "printed report" read mode (R-2p). Page breaks separate the sheets;
 * each sheet carries an accent rule and a running header/footer per settings.
 */
export function DocumentReader({
  rows,
  settings,
}: {
  rows: ReportRow[]
  settings?: ReportDocSettings
}) {
  const { t } = useTranslation()
  const isEmpty = rows.every((row) => row.columns.every((col) => col.blocks.length === 0))

  if (isEmpty) {
    return (
      <div className="mx-auto flex max-w-3xl flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border bg-card px-10 py-20 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-secondary text-muted-foreground">
          <FileText className="h-6 w-6" />
        </div>
        <p className="text-sm font-medium text-foreground">{t('berichte.docs.readerEmptyTitle')}</p>
        <p className="max-w-xs text-xs text-muted-foreground">
          {t('berichte.docs.readerEmptyHint')}
        </p>
      </div>
    )
  }

  const pages = paginate(rows)

  return (
    <div className="report-desk -mx-6 -my-6 min-h-full bg-secondary/50 px-6 py-10">
      <div className="mx-auto flex flex-col items-center gap-8">
        {pages.map((pageRows, i) => (
          <ReportPage
            key={i}
            rows={pageRows}
            settings={settings}
            pageIndex={i}
            pageCount={pages.length}
          />
        ))}
      </div>
    </div>
  )
}

/** A single A4 sheet with accent rule, running header/footer and body blocks. */
function ReportPage({
  rows,
  settings,
  pageIndex,
  pageCount,
}: {
  rows: ReportRow[]
  settings?: ReportDocSettings
  pageIndex: number
  pageCount: number
}) {
  const { t } = useTranslation()
  const accent = settings?.accentColor
  const hasCover = rows[0]?.columns[0]?.blocks[0]?.type === 'cover'
  // Real reports leave the cover page without running header/footer/page number.
  const showHeader = !hasCover && settings?.showHeader && Boolean(settings.headerText || settings.logoUrl)
  const showFooter = !hasCover && (settings?.showFooter || settings?.showPageNumbers)

  return (
    <article
      className="report-page relative flex w-[794px] max-w-full flex-col rounded-[2px] bg-card shadow-[0_2px_4px_-1px_rgba(15,23,42,0.06),0_22px_48px_-20px_rgba(15,23,42,0.24)]"
      style={{ minHeight: 1123 }}
    >
      {/* Accent rule — custom colour or the Cosmi brand gradient */}
      {accent ? (
        <div className="h-1 w-full rounded-t-[2px]" style={{ backgroundColor: accent }} />
      ) : (
        <div className="h-1 w-full rounded-t-[2px] bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)]" />
      )}

      {showHeader && (
        <header className="flex items-center justify-between gap-4 border-b border-border-muted px-[72px] pb-4 pt-7 text-xs text-muted-foreground">
          <span className="truncate">{settings?.headerText}</span>
          {settings?.logoUrl && (
            <img src={settings.logoUrl} alt="" className="h-6 shrink-0 object-contain" />
          )}
        </header>
      )}

      <div className="flex-1 px-[72px] py-[56px]">
        <div className="space-y-7">
          {rows.map((row) => (
            <div
              key={row.id}
              className={row.columns.length > 1 ? 'flex flex-wrap gap-6' : undefined}
            >
              {row.columns.map((col) => (
                <div key={col.id} className="min-w-0 space-y-6" style={{ flex: col.width ?? 1 }}>
                  {col.blocks.map((block) => (
                    <BlockView key={block.id} block={block} />
                  ))}
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>

      {showFooter && (
        <footer className="mt-auto flex items-center justify-between gap-4 border-t border-border-muted px-[72px] pb-7 pt-4 text-[11px] text-muted-foreground">
          <span className="truncate">{settings?.showFooter ? settings?.headerText : ''}</span>
          {settings?.showPageNumbers && (
            <span className="shrink-0 tabular-nums">
              {t('berichte.docs.pageXofY', { x: pageIndex + 1, y: pageCount })}
            </span>
          )}
        </footer>
      )}
    </article>
  )
}

function BlockView({ block }: { block: ReportBlock }) {
  switch (block.type) {
    case 'cover':
      return (
        <div className="flex flex-col items-center gap-2 py-12 text-center">
          {block.logoUrl && (
            <img src={block.logoUrl} alt="" className="mb-2 h-12 object-contain" />
          )}
          <h1 className="text-3xl font-bold tracking-tight text-foreground">{block.title}</h1>
          {block.subtitle && <p className="text-lg text-muted-foreground">{block.subtitle}</p>}
          {block.author && <p className="mt-1 text-sm text-muted-foreground">{block.author}</p>}
          {block.showDate && (
            <p className="text-sm text-muted-foreground">{formatDate(new Date().toISOString())}</p>
          )}
        </div>
      )

    case 'heading':
      return block.level === 1 ? (
        <h2 className="border-b border-border pb-1.5 text-2xl font-bold text-foreground">
          {block.text}
        </h2>
      ) : (
        <h3 className="text-xl font-semibold text-foreground">{block.text}</h3>
      )

    case 'text':
      return (
        <div
          className="tiptap-content prose prose-sm max-w-none text-foreground dark:prose-invert"
          dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(block.html) }}
        />
      )

    case 'kpi':
      return <KpiView block={block} />

    case 'chart':
    case 'table':
      return <ChartBlockView block={block} />

    case 'callout':
      return (
        <div className={`rounded-xl border p-4 ${CALLOUT_STYLE[block.variant]}`}>
          {block.title && (
            <p className="mb-1 text-sm font-semibold text-foreground">{block.title}</p>
          )}
          <div
            className="prose prose-sm max-w-none text-foreground dark:prose-invert"
            dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(block.html) }}
          />
        </div>
      )

    case 'bullet':
      return (
        <ul className="list-disc space-y-1 pl-5 text-sm text-foreground">
          {block.items.map((item, i) => (
            <li key={i}>{item}</li>
          ))}
        </ul>
      )

    case 'divider':
      return <hr className="border-border" />

    case 'image':
      return (
        <figure className="space-y-2">
          <img src={block.url} alt={block.alt ?? ''} className="w-full rounded-lg" />
          {block.caption && (
            <figcaption className="text-center text-xs text-muted-foreground">
              {block.caption}
            </figcaption>
          )}
        </figure>
      )

    case 'pagebreak':
    default:
      // Page breaks are invisible on screen; they only matter in the PDF (R-3).
      return null
  }
}

function KpiView({ block }: { block: KpiBlock }) {
  const positive = (block.changePercent ?? 0) >= 0
  return (
    <div className="rounded-xl border border-border bg-secondary/20 p-4">
      <p className="text-xs text-muted-foreground">{block.label}</p>
      <p className="mt-1 text-2xl font-semibold text-foreground">
        {block.value}
        {block.unit && (
          <span className="ml-1 text-base font-normal text-muted-foreground">{block.unit}</span>
        )}
      </p>
      <div className="mt-1 flex items-center justify-between gap-2">
        {block.changePercent != null ? (
          <span
            className={`inline-flex items-center gap-1 text-xs ${positive ? 'text-success' : 'text-destructive'}`}
          >
            {positive ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}
            {Math.abs(block.changePercent)} %
          </span>
        ) : (
          <span />
        )}
        {block.source && (
          <span className="truncate text-[10px] uppercase tracking-wide text-muted-foreground/70">
            {block.source}
          </span>
        )}
      </div>
    </div>
  )
}

function ChartBlockView({ block }: { block: ChartBlock | TableBlock }) {
  const { t } = useTranslation()
  const { data, isLoading } = useReportResult(block.definitionId)
  const result = data?.result
  const viz = block.type === 'table' ? 'table' : (block.viz ?? 'bar')

  return (
    <figure className="space-y-2">
      <div className="rounded-xl border border-border bg-card p-4">
        {result ? (
          <ChartRenderer result={result} viz={viz} height={280} />
        ) : isLoading ? (
          <Skeleton className="h-[240px] w-full rounded-lg" />
        ) : (
          <div className="flex h-[200px] items-center justify-center text-sm text-muted-foreground">
            {t('berichte.docs.noChartSource')}
          </div>
        )}
      </div>
      {block.caption && (
        <figcaption className="text-center text-xs text-muted-foreground">
          {block.caption}
        </figcaption>
      )}
    </figure>
  )
}
