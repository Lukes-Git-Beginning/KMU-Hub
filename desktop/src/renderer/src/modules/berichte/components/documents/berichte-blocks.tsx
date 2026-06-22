/**
 * berichte block registry — the report-specific blocks (cover, KPI, chart, table,
 * page break) plus the shared core blocks, wired into the shared document engine.
 * Edit/view components are the original berichte implementations; only the plumbing
 * (registry instead of switch statements) changed.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BarChart3,
  BookOpen,
  Gauge,
  Heading1,
  Heading2,
  RefreshCw,
  Scissors,
  Table as TableIcon,
  TrendingDown,
  TrendingUp,
  TrendingUp as KpiIcon,
} from 'lucide-react'
import { Skeleton } from '@/components/shared'
import {
  buildRegistry,
  createCoreBlockDefs,
  createSpecialBlockDefs,
  defineBlock,
  docUid,
  emptyDocColumn,
  type BlockEditProps,
  type BlockRegistry,
  type BlockTypeDef,
  type BlockViewProps,
  type DocRow,
  type QuickInsert,
} from '@/components/shared/document'
import { formatDate } from '@/lib/format'
import { useReportResult } from '@/api/hooks/useBerichte'
import type {
  ChartBlock,
  CoverBlock,
  HeadingBlock,
  KpiBlock,
  PageBreakBlock,
  TableBlock,
} from '@/api/berichte-types'
import { ChartBlockPicker } from './ChartBlockPicker'
import { ChartRenderer } from '../charts/ChartRenderer'

/* --------------------------- heading (berichte) --------------------------- */
// berichte keeps its own heading edit (its i18n keys + H1/H2 toggle) but shares
// the read-view voice via the core registry's heading, so we override only Edit.

function HeadingEdit({ block, onPatch }: BlockEditProps<HeadingBlock>) {
  const { t } = useTranslation()
  const inputCls =
    'w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'
  return (
    <div className="flex items-center gap-2">
      <div className="flex overflow-hidden rounded-lg border border-border">
        {([1, 2] as const).map((lvl) => {
          const Icon = lvl === 1 ? Heading1 : Heading2
          return (
            <button
              key={lvl}
              type="button"
              onClick={() => onPatch({ level: lvl })}
              className={`p-2 ${block.level === lvl ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'}`}
              aria-label={`H${lvl}`}
            >
              <Icon className="h-4 w-4" />
            </button>
          )
        })}
      </div>
      <input
        value={block.text}
        onChange={(e) => onPatch({ text: e.target.value })}
        placeholder={t('berichte.docs.ph.heading')}
        className={`${inputCls} ${block.level === 1 ? 'text-lg font-semibold' : 'text-base font-medium'}`}
      />
    </div>
  )
}

function HeadingView({ block }: BlockViewProps<HeadingBlock>) {
  return block.level === 1 ? (
    <h2 className="report-serif mt-2 text-[2rem] font-semibold leading-tight tracking-tight text-foreground">
      {block.text}
    </h2>
  ) : (
    <h3 className="text-lg font-semibold tracking-tight text-foreground">{block.text}</h3>
  )
}

/* -------------------------------- cover ----------------------------------- */

function CoverEdit({ block, onPatch }: BlockEditProps<CoverBlock>) {
  const { t } = useTranslation()
  const inputCls =
    'w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'
  return (
    <div className="space-y-2 rounded-xl border border-border bg-card p-4">
      <input
        value={block.title}
        onChange={(e) => onPatch({ title: e.target.value })}
        placeholder={t('berichte.docs.ph.coverTitle')}
        className={`${inputCls} text-base font-semibold`}
      />
      <input
        value={block.subtitle ?? ''}
        onChange={(e) => onPatch({ subtitle: e.target.value })}
        placeholder={t('berichte.docs.ph.coverSubtitle')}
        className={inputCls}
      />
      <input
        value={block.author ?? ''}
        onChange={(e) => onPatch({ author: e.target.value })}
        placeholder={t('berichte.docs.ph.author')}
        className={inputCls}
      />
    </div>
  )
}

function CoverView({ block, accent }: BlockViewProps<CoverBlock>) {
  const { t } = useTranslation()
  const accentStyle = accent ? { backgroundColor: accent } : undefined
  return (
    <div className="flex min-h-[920px] flex-col py-2">
      {block.logoUrl && <img src={block.logoUrl} alt="" className="h-10 self-start object-contain" />}
      <div className="mt-28 max-w-[20ch]">
        <div
          className={`mb-7 h-1.5 w-16 rounded-full ${accent ? '' : 'bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)]'}`}
          style={accentStyle}
        />
        <h1 className="report-serif text-[3.25rem] font-semibold leading-[1.06] tracking-tight text-foreground">
          {block.title}
        </h1>
        {block.subtitle && (
          <p className="mt-5 max-w-[34ch] text-lg leading-snug text-muted-foreground">{block.subtitle}</p>
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
              <dd className="mt-0.5 text-sm text-foreground">{formatDate(new Date().toISOString())}</dd>
            </div>
          )}
        </dl>
      </div>
    </div>
  )
}

/* --------------------------------- kpi ------------------------------------ */

function KpiEdit({ block, onPatch }: BlockEditProps<KpiBlock>) {
  const { t } = useTranslation()
  const positive = (block.changePercent ?? 0) >= 0
  const hasTrend = block.changePercent != null
  const fieldCls =
    'rounded-md border border-transparent bg-transparent px-2 py-1 text-sm text-foreground hover:border-border focus:border-border focus:outline-none'
  return (
    <div className="space-y-2 rounded-xl border border-border bg-secondary/20 p-3">
      <input
        value={block.label}
        onChange={(e) => onPatch({ label: e.target.value })}
        placeholder={t('berichte.docs.ph.kpiLabel')}
        className={`${fieldCls} w-full text-xs text-muted-foreground`}
      />
      <div className="flex items-baseline gap-1.5">
        <input
          value={block.value}
          onChange={(e) => onPatch({ value: e.target.value })}
          placeholder={t('berichte.docs.ph.kpiValue')}
          className={`${fieldCls} min-w-0 flex-1 text-2xl font-semibold`}
        />
        <input
          value={block.unit ?? ''}
          onChange={(e) => onPatch({ unit: e.target.value })}
          placeholder={t('berichte.docs.ph.kpiUnit')}
          className={`${fieldCls} w-16 text-base text-muted-foreground`}
        />
      </div>
      <div className="flex items-center gap-2 border-t border-border-muted pt-2">
        <span className="text-muted-foreground">
          {hasTrend ? (
            positive ? (
              <TrendingUp className="h-3.5 w-3.5 text-success" />
            ) : (
              <TrendingDown className="h-3.5 w-3.5 text-destructive" />
            )
          ) : (
            <TrendingUp className="h-3.5 w-3.5 text-muted-foreground/40" />
          )}
        </span>
        <input
          type="number"
          value={block.changePercent ?? ''}
          onChange={(e) => onPatch({ changePercent: e.target.value === '' ? null : Number(e.target.value) })}
          placeholder={t('berichte.docs.ph.kpiTrend')}
          className={`${fieldCls} w-20 text-xs`}
        />
        <input
          value={block.source ?? ''}
          onChange={(e) => onPatch({ source: e.target.value })}
          placeholder={t('berichte.docs.ph.kpiSource')}
          className={`${fieldCls} min-w-0 flex-1 text-xs text-muted-foreground`}
        />
      </div>
    </div>
  )
}

function KpiView({ block }: BlockViewProps<KpiBlock>) {
  const positive = (block.changePercent ?? 0) >= 0
  return (
    <div className="report-keep flex h-full flex-col rounded-lg border border-border-muted bg-card p-4">
      <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{block.label}</p>
      <p className="mt-3 text-[1.75rem] font-semibold leading-none tracking-tight text-foreground tabular-nums">
        {block.value}
        {block.unit && <span className="ml-1 text-base font-normal text-muted-foreground">{block.unit}</span>}
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

/* ----------------------------- chart / table ------------------------------ */

function EmbeddedEdit({ block, onPatch }: BlockEditProps<ChartBlock | TableBlock>) {
  const { t } = useTranslation()
  const [pickerOpen, setPickerOpen] = useState(false)
  const { data, isLoading } = useReportResult(block.definitionId)
  const result = data?.result
  const isTable = block.type === 'table'
  const viz = block.type === 'table' ? 'table' : ((block as ChartBlock).viz ?? 'bar')
  const labelKey = isTable ? 'berichte.docs.blockType.table' : 'berichte.docs.blockType.chart'
  const configureKey = isTable ? 'berichte.docs.configureTable' : 'berichte.docs.configureChart'
  const changeKey = isTable ? 'berichte.docs.changeTable' : 'berichte.docs.changeChart'
  const Icon = isTable ? TableIcon : BarChart3
  const captionCls =
    'w-full rounded-md border border-transparent bg-transparent px-2 py-1 text-center text-xs text-muted-foreground hover:border-border focus:border-border focus:outline-none'

  const picker = (
    <ChartBlockPicker
      open={pickerOpen}
      onClose={() => setPickerOpen(false)}
      mode={isTable ? 'table' : 'chart'}
      onApply={(patch) => onPatch(isTable ? { definitionId: patch.definitionId } : patch)}
    />
  )

  if (!block.definitionId) {
    return (
      <>
        <button
          type="button"
          onClick={() => setPickerOpen(true)}
          className="flex w-full items-center gap-2.5 rounded-xl border border-dashed border-border bg-secondary/20 p-3 text-left transition-colors hover:border-primary/40 hover:bg-secondary/40"
        >
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-secondary text-muted-foreground">
            <Icon className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="text-xs font-medium text-foreground">{t(labelKey)}</p>
            <p className="truncate text-[11px] text-primary">{t(configureKey)}</p>
          </div>
        </button>
        {picker}
      </>
    )
  }

  return (
    <>
      <div className="space-y-2 rounded-xl border border-border bg-card p-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-muted-foreground">{t(labelKey)}</span>
          <button
            type="button"
            onClick={() => setPickerOpen(true)}
            className="flex items-center gap-1 rounded-md px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <RefreshCw className="h-3 w-3" />
            {t(changeKey)}
          </button>
        </div>
        {result ? (
          <ChartRenderer result={result} viz={viz} height={220} />
        ) : isLoading ? (
          <Skeleton className="h-[200px] w-full rounded-lg" />
        ) : (
          <div className="flex h-[160px] items-center justify-center text-sm text-muted-foreground">
            {t('berichte.docs.noChartSource')}
          </div>
        )}
        <input
          value={block.caption ?? ''}
          onChange={(e) => onPatch({ caption: e.target.value })}
          placeholder={t('berichte.docs.ph.caption')}
          className={captionCls}
        />
      </div>
      {picker}
    </>
  )
}

function EmbeddedView({ block, palette }: BlockViewProps<ChartBlock | TableBlock>) {
  const { t } = useTranslation()
  const { data, isLoading } = useReportResult(block.definitionId)
  const result = data?.result
  const viz = block.type === 'table' ? 'table' : ((block as ChartBlock).viz ?? 'bar')
  return (
    <figure className="report-keep space-y-2.5">
      <div className="rounded-lg border border-border-muted bg-card p-4">
        {result ? (
          <ChartRenderer result={result} viz={viz} height={280} options={palette ? { palette } : undefined} />
        ) : isLoading ? (
          <Skeleton className="h-[240px] w-full rounded-lg" />
        ) : (
          <div className="flex h-[200px] items-center justify-center text-sm text-muted-foreground">
            {t('berichte.docs.noChartSource')}
          </div>
        )}
      </div>
      {block.caption && (
        <figcaption className="text-center text-xs italic text-muted-foreground">{block.caption}</figcaption>
      )}
    </figure>
  )
}

/* ------------------------------- page break ------------------------------- */

function PageBreakEdit(_props: BlockEditProps<PageBreakBlock>) {
  const { t } = useTranslation()
  return (
    <div className="flex items-center gap-2 text-[10px] uppercase tracking-wider text-muted-foreground/60">
      <span className="h-px flex-1 bg-border" />
      {t('berichte.docs.blockType.pagebreak')}
      <span className="h-px flex-1 bg-border" />
    </div>
  )
}

function PageBreakView(_props: BlockViewProps<PageBreakBlock>) {
  // Invisible on screen; only matters for print pagination.
  return null
}

/* ------------------------------- registry --------------------------------- */

const coverDef = defineBlock<CoverBlock>({
  type: 'cover',
  icon: BookOpen,
  labelKey: 'berichte.docs.blockType.cover',
  group: 'layout',
  insertable: false,
  atomic: true,
  makeDefault: () => ({ id: docUid('b'), type: 'cover', title: '', showDate: true }),
  Edit: CoverEdit,
  View: CoverView,
})
const headingDef = defineBlock<HeadingBlock>({
  type: 'heading',
  icon: BookOpen,
  labelKey: 'berichte.docs.blockType.heading',
  group: 'content',
  makeDefault: () => ({ id: docUid('b'), type: 'heading', level: 1, text: '' }),
  Edit: HeadingEdit,
  View: HeadingView,
})
const chartDef = defineBlock<ChartBlock>({
  type: 'chart',
  icon: BarChart3,
  labelKey: 'berichte.docs.blockType.chart',
  group: 'data',
  atomic: true,
  makeDefault: () => ({ id: docUid('b'), type: 'chart' }),
  Edit: EmbeddedEdit as BlockTypeDef<ChartBlock>['Edit'],
  View: EmbeddedView as BlockTypeDef<ChartBlock>['View'],
})
const tableDef = defineBlock<TableBlock>({
  type: 'table',
  icon: TableIcon,
  labelKey: 'berichte.docs.blockType.table',
  group: 'data',
  atomic: true,
  makeDefault: () => ({ id: docUid('b'), type: 'table' }),
  Edit: EmbeddedEdit as BlockTypeDef<TableBlock>['Edit'],
  View: EmbeddedView as BlockTypeDef<TableBlock>['View'],
})
const kpiDef = defineBlock<KpiBlock>({
  type: 'kpi',
  icon: KpiIcon,
  labelKey: 'berichte.docs.blockType.kpi',
  group: 'data',
  atomic: true,
  makeDefault: () => ({ id: docUid('b'), type: 'kpi', label: '', value: '' }),
  Edit: KpiEdit,
  View: KpiView,
})
const pageBreakDef = defineBlock<PageBreakBlock>({
  type: 'pagebreak',
  icon: Scissors,
  labelKey: 'berichte.docs.blockType.pagebreak',
  group: 'layout',
  makeDefault: () => ({ id: docUid('b'), type: 'pagebreak' }),
  Edit: PageBreakEdit,
  View: PageBreakView,
})

// Core blocks shared with every surface; berichte overrides heading (its own keys).
const core = createCoreBlockDefs({ omit: ['heading'] })
const byType = Object.fromEntries(core.map((d) => [d.type, d]))

// Shared special blocks that make sense on paper: a code listing, a manual
// comparison table and an editorial pull-quote. Toggle/attachment/bookmark stay
// wiki-only (collapse and downloads don't translate to a printed report).
const special = Object.fromEntries(
  createSpecialBlockDefs({ only: ['code'] }).map((d) => [d.type, d]),
)

/** Insert-menu order mirrors the original berichte block editor. */
export const berichteBlockRegistry: BlockRegistry = buildRegistry([
  headingDef,
  byType.text,
  chartDef,
  tableDef,
  kpiDef,
  byType.callout,
  byType.bullet,
  special.code,
  byType.image,
  byType.divider,
  pageBreakDef,
  coverDef,
])

/** Block types that stand alone on their own print sheet. */
export const berichteFullPageTypes = ['cover']

/** Quick-insert: a KPI row (three columns, one KPI block each). */
export const berichteQuickInserts: QuickInsert[] = [
  {
    id: 'kpi-row',
    labelKey: 'berichte.docs.kpiRow',
    icon: Gauge,
    makeRow: (): DocRow => ({
      id: docUid('row'),
      columns: Array.from({ length: 3 }, () => ({
        ...emptyDocColumn(),
        blocks: [{ id: docUid('b'), type: 'kpi', label: '', value: '' } as KpiBlock],
      })),
    }),
  },
]
