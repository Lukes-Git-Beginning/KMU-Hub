import { useTranslation } from 'react-i18next'
import { TrendingDown, TrendingUp } from 'lucide-react'
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

/** Renders a report document as a clean, readable page (the read mode). */
export function DocumentReader({
  rows,
  settings,
}: {
  rows: ReportRow[]
  settings?: ReportDocSettings
}) {
  return (
    <div className="mx-auto max-w-3xl rounded-xl border border-border bg-card px-10 py-10 shadow-sm">
      {settings?.showHeader && settings.headerText && (
        <div className="mb-8 border-b border-border pb-3 text-xs text-muted-foreground">
          {settings.headerText}
        </div>
      )}

      <div className="space-y-6">
        {rows.map((row) => (
          <div
            key={row.id}
            className={row.columns.length > 1 ? 'flex flex-wrap gap-5' : undefined}
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
      {block.changePercent != null && (
        <p
          className={`mt-1 inline-flex items-center gap-1 text-xs ${positive ? 'text-success' : 'text-destructive'}`}
        >
          {positive ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}
          {Math.abs(block.changePercent)} %
        </p>
      )}
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
