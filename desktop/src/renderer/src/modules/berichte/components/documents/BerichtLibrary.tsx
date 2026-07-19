import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BarChart3,
  CalendarRange,
  ClipboardList,
  Copy,
  FilePlus,
  FileText,
  LayoutGrid,
  List as ListIcon,
  Search,
  Trash2,
  TrendingUp,
  Zap,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { toast } from 'sonner'
import type { ReportDocStatus, ReportDocument } from '@/api/berichte-types'
import {
  useCreateReportDocument,
  useDeleteReportDocument,
  useReportDocuments,
  useReportTemplates,
} from '@/api/hooks/useBerichte'
import {
  ConfirmDialog,
  EmptyState,
  ItemActions,
  SkeletonCard,
  SortMenu,
  type SortDirection,
} from '@/components/shared'
import { useHasCapability, useScopedCapability } from '@/hooks/useCapability'
import { formatDate } from '@/lib/format'
import { ReportStatusBadge } from './ReportStatusBadge'
import { estimatePageCount } from './doc-utils'

interface BerichtLibraryProps {
  onOpen: (doc: ReportDocument) => void
  onNew: () => void
  onNewFromTemplate: (templateId: string) => void
}

const TEMPLATE_ICONS: Record<string, LucideIcon> = {
  CalendarRange,
  TrendingUp,
  BarChart3,
  ClipboardList,
  Zap,
}

const STATUS_FILTERS: { value: 'all' | ReportDocStatus; key: string }[] = [
  { value: 'all', key: 'berichte.docs.filter.all' },
  { value: 'draft', key: 'berichte.docs.status.draft' },
  { value: 'final', key: 'berichte.docs.status.final' },
  { value: 'released', key: 'berichte.docs.status.released' },
  { value: 'archived', key: 'berichte.docs.status.archived' },
]

export function BerichtLibrary({ onOpen, onNew, onNewFromTemplate }: BerichtLibraryProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useReportDocuments()
  const { data: templatesData } = useReportTemplates()
  const createMutation = useCreateReportDocument()
  const deleteMutation = useDeleteReportDocument()

  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | ReportDocStatus>('all')
  const [sortField, setSortField] = useState('updated_at')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')
  const [layout, setLayout] = useState<'grid' | 'list'>('grid')
  const [pendingDelete, setPendingDelete] = useState<ReportDocument | null>(null)

  // RBAC (R-3): create covers blank/template/duplicate; delete is scopeable
  // (own = author only) and checked per card.
  const canCreate = useHasCapability('berichte:reports:create')

  const docs = useMemo(() => data?.documents ?? [], [data])
  const templates = templatesData?.templates ?? []

  const filtered = useMemo(() => {
    let result = docs
    if (search) {
      result = result.filter((d) => d.title.toLowerCase().includes(search.toLowerCase()))
    }
    if (statusFilter !== 'all') result = result.filter((d) => d.status === statusFilter)
    return [...result].sort((a, b) => {
      const av = sortField === 'title' ? a.title : a.updated_at
      const bv = sortField === 'title' ? b.title : b.updated_at
      const cmp = String(av).localeCompare(String(bv))
      return sortDir === 'desc' ? -cmp : cmp
    })
  }, [docs, search, statusFilter, sortField, sortDir])

  function duplicate(doc: ReportDocument) {
    createMutation.mutate(
      {
        title: `${doc.title} (Kopie)`,
        description: doc.description,
        module: doc.module,
        rows: doc.rows,
        settings: doc.settings,
      },
      { onSuccess: () => toast.success(t('berichte.docs.duplicated')) },
    )
  }

  function confirmDelete() {
    if (!pendingDelete) return
    deleteMutation.mutate(pendingDelete.id, {
      onSuccess: () => toast.success(t('berichte.docs.deleted')),
    })
    setPendingDelete(null)
  }

  function Card({ doc }: { doc: ReportDocument }) {
    const pages = estimatePageCount(doc)
    // Scope-own: delete disappears silently on foreign documents.
    const canDeleteDoc = useScopedCapability('berichte:reports:delete', doc.created_by)
    const actions = [
      { label: t('berichte.docs.open'), icon: FileText, onClick: () => onOpen(doc) },
      ...(canCreate
        ? [{ label: t('berichte.docs.duplicate'), icon: Copy, onClick: () => duplicate(doc) }]
        : []),
      ...(canDeleteDoc
        ? [{
            label: t('berichte.docs.delete'),
            icon: Trash2,
            onClick: () => setPendingDelete(doc),
            variant: 'destructive' as const,
          }]
        : []),
    ]
    return (
      <div
        role="button"
        tabIndex={0}
        onClick={() => onOpen(doc)}
        onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && onOpen(doc)}
        className="group flex cursor-pointer items-start gap-3 rounded-xl border border-border bg-card p-4 text-left transition-colors hover:border-primary/40"
      >
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
          <FileText className="h-4.5 w-4.5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <h4 className="truncate text-sm font-medium text-foreground">{doc.title}</h4>
            <div onClick={(e) => e.stopPropagation()}>
              <ItemActions items={actions} />
            </div>
          </div>
          {doc.description && (
            <p className="mt-0.5 truncate text-[11px] text-muted-foreground">{doc.description}</p>
          )}
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            <ReportStatusBadge status={doc.status} />
            <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] capitalize text-muted-foreground">
              {doc.module}
            </span>
            <span className="text-[10px] text-muted-foreground/70">
              {t('berichte.docs.pages', { count: pages })}
            </span>
          </div>
          <p className="mt-1.5 text-[11px] text-muted-foreground/70">
            {t('berichte.docs.edited')} {formatDate(doc.updated_at)}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Template picker — start a new report from a prebuilt structure */}
      {canCreate && templates.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-muted-foreground">
            {t('berichte.docs.startFrom')}
          </p>
          <div className="flex gap-2 overflow-x-auto pb-1">
            <button
              type="button"
              onClick={onNew}
              className="flex w-44 shrink-0 items-center gap-2.5 rounded-xl border border-dashed border-border bg-card p-3 text-left transition-colors hover:border-primary/40"
            >
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-secondary text-muted-foreground">
                <FilePlus className="h-4 w-4" />
              </div>
              <p className="text-xs font-medium text-foreground">{t('berichte.docs.blank')}</p>
            </button>
            {templates.map((tpl) => {
              const Icon = TEMPLATE_ICONS[tpl.icon ?? ''] ?? FileText
              return (
                <button
                  key={tpl.id}
                  type="button"
                  onClick={() => onNewFromTemplate(tpl.id)}
                  className="flex w-52 shrink-0 items-start gap-2.5 rounded-xl border border-border bg-card p-3 text-left transition-colors hover:border-primary/40"
                >
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
                    <Icon className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="truncate text-xs font-medium text-foreground">{tpl.title}</p>
                    <p className="mt-0.5 line-clamp-2 text-[10px] leading-tight text-muted-foreground">
                      {tpl.description}
                    </p>
                  </div>
                </button>
              )
            })}
          </div>
        </div>
      )}

      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-[200px] flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('berichte.docs.search')}
            className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        </div>
        <SortMenu
          options={[
            { value: 'updated_at', label: t('berichte.docs.sortEdited') },
            { value: 'title', label: t('berichte.docs.sortName') },
          ]}
          field={sortField}
          direction={sortDir}
          onChange={(f, d) => {
            setSortField(f)
            setSortDir(d)
          }}
        />
        <div className="flex overflow-hidden rounded-lg border border-border">
          <button
            type="button"
            onClick={() => setLayout('grid')}
            className={`p-2 ${layout === 'grid' ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'}`}
            aria-label={t('berichte.docs.grid')}
          >
            <LayoutGrid className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={() => setLayout('list')}
            className={`p-2 ${layout === 'list' ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'}`}
            aria-label={t('berichte.docs.list')}
          >
            <ListIcon className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Status filter chips */}
      <div className="flex flex-wrap gap-1.5">
        {STATUS_FILTERS.map((f) => (
          <button
            key={f.value}
            type="button"
            onClick={() => setStatusFilter(f.value)}
            className={`rounded-full border px-2.5 py-1 text-xs transition-colors ${
              statusFilter === f.value
                ? 'border-primary bg-primary-light text-primary'
                : 'border-border text-muted-foreground hover:bg-secondary'
            }`}
          >
            {t(f.key)}
          </button>
        ))}
      </div>

      {/* Grid / list */}
      {isLoading ? (
        <div
          className={
            layout === 'grid'
              ? 'grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3'
              : 'flex flex-col gap-2'
          }
        >
          {Array.from({ length: 6 }).map((_, i) => (
            <SkeletonCard key={i} />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={FileText}
          title={t('berichte.docs.emptyTitle')}
          description={t('berichte.docs.emptyHint')}
          action={canCreate ? { label: t('berichte.actions.neuerBericht'), onClick: onNew } : undefined}
        />
      ) : (
        <div
          className={
            layout === 'grid'
              ? 'grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3'
              : 'flex flex-col gap-2'
          }
        >
          {filtered.map((doc) => (
            <Card key={doc.id} doc={doc} />
          ))}
        </div>
      )}

      <ConfirmDialog
        open={!!pendingDelete}
        onOpenChange={(o) => !o && setPendingDelete(null)}
        title={t('berichte.docs.deleteTitle')}
        description={t('berichte.docs.deleteDesc', { title: pendingDelete?.title ?? '' })}
        confirmLabel={t('berichte.docs.delete')}
        onConfirm={confirmDelete}
        variant="destructive"
      />
    </div>
  )
}
