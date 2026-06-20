import { useTranslation } from 'react-i18next'
import {
  AlertTriangle,
  CheckCircle2,
  FileText,
  Info,
  Lightbulb,
  TrendingDown,
  TrendingUp,
  type LucideIcon,
} from 'lucide-react'
import DOMPurify from 'dompurify'
import type {
  CalloutVariant,
  ChartBlock,
  CoverBlock,
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
  info: 'border-info/25 bg-info-light',
  success: 'border-success/25 bg-success-light',
  warning: 'border-warning/25 bg-warning-light',
  recommendation: 'border-primary/25 bg-primary-light',
}

const CALLOUT_ICON: Record<CalloutVariant, { icon: LucideIcon; tone: string }> = {
  info: { icon: Info, tone: 'text-info' },
  success: { icon: CheckCircle2, tone: 'text-success' },
  warning: { icon: AlertTriangle, tone: 'text-warning' },
  recommendation: { icon: Lightbulb, tone: 'text-primary' },
}

/** A row that is purely a page break — used as a sheet separator in read mode. */
function isPageBreakRow(row: ReportRow): boolean {
  return (
    row.columns.length === 1 &&
    row.columns[0].blocks.length === 1 &&
    row.columns[0].blocks[0].type === 'pagebreak'
  )
}

/** Whether a row carries a cover block (which always gets its own sheet). */
function rowHasCover(row: ReportRow): boolean {
  return row.columns.some((col) => col.blocks.some((block) => block.type === 'cover'))
}

/** Split the document into sheets on page-break rows; a cover always stands alone. */
function paginate(rows: ReportRow[]): ReportRow[][] {
  const pages: ReportRow[][] = [[]]
  for (const row of rows) {
    if (isPageBreakRow(row)) {
      pages.push([])
      continue
    }
    pages[pages.length - 1].push(row)
    if (rowHasCover(row)) pages.push([])
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
                    <BlockView key={block.id} block={block} accent={accent} />
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

function BlockView({ block, accent }: { block: ReportBlock; accent?: string }) {
  switch (block.type) {
    case 'cover':
      return <CoverView block={block} accent={accent} />

    case 'heading':
      // H1 is the editorial serif voice; H2 stays in the sans body face.
      return block.level === 1 ? (
        <h2 className="report-serif mt-2 text-[2rem] font-semibold leading-tight tracking-tight text-foreground">
          {block.text}
        </h2>
      ) : (
        <h3 className="text-lg font-semibold tracking-tight text-foreground">{block.text}</h3>
      )

    case 'text':
      return (
        <div
          className="tiptap-content prose prose-sm max-w-[65ch] leading-relaxed text-foreground dark:prose-invert"
          dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(block.html) }}
        />
      )

    case 'kpi':
      return <KpiView block={block} />

    case 'chart':
    case 'table':
      return <ChartBlockView block={block} />

    case 'callout': {
      const { icon: CalloutIcon, tone } = CALLOUT_ICON[block.variant]
      return (
        <div className={`flex gap-3 rounded-xl border p-4 ${CALLOUT_STYLE[block.variant]}`}>
          <CalloutIcon className={`mt-0.5 h-4 w-4 shrink-0 ${tone}`} aria-hidden="true" />
          <div className="min-w-0">
            {block.title && (
              <p className="mb-1 text-sm font-semibold text-foreground">{block.title}</p>
            )}
            <div
              className="prose prose-sm max-w-[60ch] text-foreground dark:prose-invert"
              dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(block.html) }}
            />
          </div>
        </div>
      )
    }

    case 'bullet': {
      const items = block.items.filter((item) => item.trim().length > 0)
      const cls = 'max-w-[65ch] space-y-1.5 pl-5 text-sm leading-relaxed text-foreground'
      return block.ordered ? (
        <ol className={`list-decimal marker:text-muted-foreground ${cls}`}>
          {items.map((item, i) => (
            <li key={i}>{item}</li>
          ))}
        </ol>
      ) : (
        <ul className={`list-disc marker:text-muted-foreground/60 ${cls}`}>
          {items.map((item, i) => (
            <li key={i}>{item}</li>
          ))}
        </ul>
      )
    }

    case 'divider':
      return <hr className="my-2 border-border-muted" />

    case 'image':
      return (
        <figure className="space-y-2">
          <img
            src={block.url}
            alt={block.alt ?? ''}
            className="w-full rounded-lg border border-border-muted"
          />
          {block.caption && (
            <figcaption className="text-center text-xs italic text-muted-foreground">
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

/** Editorial cover page: kicker accent, large serif title, footed meta block. */
function CoverView({ block, accent }: { block: CoverBlock; accent?: string }) {
  const { t } = useTranslation()
  const accentStyle = accent ? { backgroundColor: accent } : undefined
  return (
    <div className="flex min-h-[920px] flex-col py-2">
      {block.logoUrl && (
        <img src={block.logoUrl} alt="" className="h-10 self-start object-contain" />
      )}

      <div className="mt-28 max-w-[20ch]">
        <div
          className={`mb-7 h-1.5 w-16 rounded-full ${
            accent ? '' : 'bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)]'
          }`}
          style={accentStyle}
        />
        <h1 className="report-serif text-[3.25rem] font-semibold leading-[1.06] tracking-tight text-foreground">
          {block.title}
        </h1>
        {block.subtitle && (
          <p className="mt-5 max-w-[34ch] text-lg leading-snug text-muted-foreground">
            {block.subtitle}
          </p>
        )}
      </div>

      <div className="flex-1" />

      <div className="border-t border-border-muted pt-5">
        <dl className="flex flex-wrap gap-x-14 gap-y-3">
          {block.author && (
            <div>
              <dt className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground/60">
                {t('berichte.docs.cover.author')}
              </dt>
              <dd className="mt-0.5 text-sm text-foreground">{block.author}</dd>
            </div>
          )}
          {block.showDate && (
            <div>
              <dt className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground/60">
                {t('berichte.docs.cover.date')}
              </dt>
              <dd className="mt-0.5 text-sm text-foreground">
                {formatDate(new Date().toISOString())}
              </dd>
            </div>
          )}
        </dl>
      </div>
    </div>
  )
}

function KpiView({ block }: { block: KpiBlock }) {
  const positive = (block.changePercent ?? 0) >= 0
  return (
    <div className="flex h-full flex-col rounded-lg border border-border-muted bg-card p-4">
      <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        {block.label}
      </p>
      <p className="mt-3 text-[1.75rem] font-semibold leading-none tracking-tight text-foreground tabular-nums">
        {block.value}
        {block.unit && (
          <span className="ml-1 text-base font-normal text-muted-foreground">{block.unit}</span>
        )}
      </p>
      <div className="mt-auto flex items-center justify-between gap-2 pt-3">
        {block.changePercent != null ? (
          <span
            className={`inline-flex items-center gap-1 text-xs font-medium ${positive ? 'text-success' : 'text-destructive'}`}
          >
            {positive ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}
            {Math.abs(block.changePercent)} %
          </span>
        ) : (
          <span />
        )}
        {block.source && (
          <span className="truncate text-[10px] uppercase tracking-wide text-muted-foreground/60">
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
    <figure className="space-y-2.5">
      <div className="rounded-lg border border-border-muted bg-card p-4">
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
        <figcaption className="text-center text-xs italic text-muted-foreground">
          {block.caption}
        </figcaption>
      )}
    </figure>
  )
}
