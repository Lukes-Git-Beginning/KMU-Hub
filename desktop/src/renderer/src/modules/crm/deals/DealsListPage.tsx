/**
 * Deals list page with toggle between table view and pipeline (Kanban) view.
 *
 * Table view: sortable columns (server-side via sort_by/sort_desc params),
 * checkbox multi-select, and bulk-delete. Pipeline view is read-only (unchanged).
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Plus,
  Search,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  ChevronDown,
  LayoutList,
  Columns3,
  BarChart3,
  TrendingUp,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useDeals, useCreateDeal, useDeleteDeal } from '@/api/hooks/useDeals'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Skeleton } from '@/components/ui/skeleton'
import { useHasCapability, useScopedCapability } from '@/hooks/useCapability'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import DealPipelineView from './DealPipelineView'
import DealForecastView from './DealForecastView'
import { PageHeader } from '@/components/shared'
import { DealFormDialog, type DealFormData } from './DealFormDialog'
import { formatDate } from '@/lib/format'

const PAGE_SIZE = 20

type DealSortField = 'name' | 'value' | 'created_at' | 'updated_at' | 'expected_close_date'

interface SortState {
  field: DealSortField
  desc: boolean
}

function formatCurrency(value?: number, currency?: string): string {
  if (value == null) return '-'
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: currency || 'EUR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

function SortIcon({ field, sort }: { field: DealSortField; sort: SortState }) {
  if (sort.field !== field) return null
  return sort.desc ? (
    <ChevronDown className="ml-1 inline h-3.5 w-3.5 shrink-0" />
  ) : (
    <ChevronUp className="ml-1 inline h-3.5 w-3.5 shrink-0" />
  )
}

export default function DealsListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [view, setView] = useState<'list' | 'pipeline' | 'forecast'>('list')
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [sort, setSort] = useState<SortState>({ field: 'created_at', desc: true })
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [showBulkDeleteDialog, setShowBulkDeleteDialog] = useState(false)
  const [showCreateDialog, setShowCreateDialog] = useState(false)

  const createDeal = useCreateDeal()
  const deleteDeal = useDeleteDeal()
  const { data: stagesData } = usePipelineStages()
  const stages = stagesData?.stages ?? []

  // RBAC
  const canCreate = useHasCapability('crm:deal:create')
  // CRM-Objekte kein owner-Feld → ownerIds leer → scope='own' ergibt deny
  const canDelete = useScopedCapability('crm:deal:delete')

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  // Clear selection when page/search/sort changes
  useEffect(() => {
    setSelectedIds(new Set())
  }, [page, debouncedSearch, sort.field, sort.desc])

  const { data, isLoading, error, refetch } = useDeals({
    page,
    page_size: PAGE_SIZE,
    search: debouncedSearch || undefined,
    sort_by: sort.field,
    sort_desc: sort.desc,
  })

  const deals = data?.deals ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  function toggleSort(field: DealSortField) {
    setSort((prev) =>
      prev.field === field ? { field, desc: !prev.desc } : { field, desc: false }
    )
    setPage(1)
  }

  // Selection helpers
  const allOnPageSelected =
    deals.length > 0 && deals.every((d) => d.id && selectedIds.has(d.id))
  const someOnPageSelected =
    deals.some((d) => d.id && selectedIds.has(d.id)) && !allOnPageSelected

  function toggleSelectAll() {
    if (allOnPageSelected) {
      const next = new Set(selectedIds)
      deals.forEach((d) => { if (d.id) next.delete(d.id) })
      setSelectedIds(next)
    } else {
      const next = new Set(selectedIds)
      deals.forEach((d) => { if (d.id) next.add(d.id) })
      setSelectedIds(next)
    }
  }

  function toggleSelectOne(id: string) {
    const next = new Set(selectedIds)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    setSelectedIds(next)
  }

  async function handleCreate(form: DealFormData) {
    try {
      await createDeal.mutateAsync({
        name: form.name,
        value: form.value,
        currency: form.currency,
        stage_id: form.stage,
        expected_close_date: form.expectedCloseDate || undefined,
        notes: form.notes || undefined,
        custom_fields: {
          ...(form.contactName ? { _contactName: form.contactName } : {}),
          ...(form.companyName ? { _companyName: form.companyName } : {}),
          ...(form.priority !== 'normal' ? { _priority: form.priority } : {}),
          ...(form.probability !== 50 ? { _probability: form.probability } : {}),
          ...(form.tags.length > 0 ? { _tags: form.tags } : {}),
        },
      })
      toast.success(t('crm.deals.created'))
    } catch {
      toast.error(t('crm.deals.createError'))
    }
  }

  async function handleBulkDelete() {
    const ids = Array.from(selectedIds)
    let errorCount = 0
    for (const id of ids) {
      try {
        await deleteDeal.mutateAsync(id)
      } catch {
        errorCount++
      }
    }
    setSelectedIds(new Set())
    setShowBulkDeleteDialog(false)
    if (errorCount > 0) {
      toast.error(t('crm.bulk.deleteError'))
    } else {
      toast.success(t('crm.bulk.deleted', { count: ids.length }))
    }
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            {t('crm.deals.loadError')}
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : t('crm.error.unexpected')}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            {t('common.retry')}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-4">
      {/* Header */}
      <PageHeader
        title={t('crm.deals.title')}
        description={t('crm.deals.count', { count: data?.total ?? 0 })}
        icon={TrendingUp}
        moduleId="contacts"
        actions={
          <div className="flex items-center gap-2">
            <div className="flex items-center rounded-md border border-border">
              <Button
                variant={view === 'list' ? 'secondary' : 'ghost'}
                size="sm"
                className="rounded-r-none"
                onClick={() => setView('list')}
              >
                <LayoutList className="h-4 w-4" />
              </Button>
              <Button
                variant={view === 'pipeline' ? 'secondary' : 'ghost'}
                size="sm"
                className="rounded-none border-x border-border"
                onClick={() => setView('pipeline')}
                title={t('crm.deals.stage')}
              >
                <Columns3 className="h-4 w-4" />
              </Button>
              <Button
                variant={view === 'forecast' ? 'secondary' : 'ghost'}
                size="sm"
                className="rounded-l-none"
                onClick={() => setView('forecast')}
                title={t('crm.deals.forecast.view')}
              >
                <BarChart3 className="h-4 w-4" />
              </Button>
            </div>
            {canCreate && (
              <Button onClick={() => setShowCreateDialog(true)} className="gap-2">
                <Plus className="h-4 w-4" />
                {t('crm.deals.new')}
              </Button>
            )}
          </div>
        }
      />

      {view === 'pipeline' ? (
        <DealPipelineView />
      ) : view === 'forecast' ? (
        <DealForecastView />
      ) : (
        <>
          {/* Search bar */}
          <div className="relative max-w-sm">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder={t('crm.deals.search')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>

          {/* Bulk action bar */}
          {selectedIds.size > 0 && canDelete && (
            <div className="flex items-center gap-3 rounded-lg border border-border bg-muted/50 px-4 py-2">
              <span className="text-sm font-medium text-foreground">
                {t('crm.bulk.selected', { count: selectedIds.size })}
              </span>
              <Button
                variant="destructive"
                size="sm"
                className="gap-1.5"
                onClick={() => setShowBulkDeleteDialog(true)}
              >
                <Trash2 className="h-3.5 w-3.5" />
                {t('common.delete')}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setSelectedIds(new Set())}
              >
                {t('crm.bulk.clearSelection')}
              </Button>
            </div>
          )}

          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 8 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : deals.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16">
              <TrendingUp className="h-12 w-12 text-muted-foreground" />
              <p className="mt-4 text-lg font-medium text-foreground">
                {t('crm.deals.noResults')}
              </p>
              <p className="mt-1 text-sm text-muted-foreground">
                {debouncedSearch
                  ? t('crm.tryDifferentSearch')
                  : t('crm.deals.emptyHint')}
              </p>
              {!debouncedSearch && canCreate && (
                <Button className="mt-4 gap-2" onClick={() => setShowCreateDialog(true)}>
                  <Plus className="h-4 w-4" />
                  {t('crm.deals.createFirst')}
                </Button>
              )}
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    {/* Checkbox column */}
                    <TableHead className="w-10 pr-0">
                      <Checkbox
                        checked={allOnPageSelected ? true : someOnPageSelected ? 'indeterminate' : false}
                        onCheckedChange={toggleSelectAll}
                        aria-label={t('common.actions')}
                      />
                    </TableHead>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => toggleSort('name')}
                    >
                      {t('crm.field.name')}
                      <SortIcon field="name" sort={sort} />
                    </TableHead>
                    <TableHead>{t('crm.deals.stage')}</TableHead>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => toggleSort('value')}
                    >
                      {t('crm.deals.value')}
                      <SortIcon field="value" sort={sort} />
                    </TableHead>
                    <TableHead>{t('crm.field.contact')}</TableHead>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => toggleSort('expected_close_date')}
                    >
                      {t('crm.deals.expectedClose')}
                      <SortIcon field="expected_close_date" sort={sort} />
                    </TableHead>
                    <TableHead>{t('crm.field.tags')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {deals.map((deal) => (
                    <TableRow
                      key={deal.id}
                      className="cursor-pointer"
                      onClick={() => deal.id && navigate(`/kontakte/pipeline/${deal.id}`)}
                      data-state={deal.id && selectedIds.has(deal.id) ? 'selected' : undefined}
                    >
                      <TableCell className="pr-0" onClick={(e) => e.stopPropagation()}>
                        <Checkbox
                          checked={!!(deal.id && selectedIds.has(deal.id))}
                          onCheckedChange={() => deal.id && toggleSelectOne(deal.id)}
                          aria-label={deal.name}
                        />
                      </TableCell>
                      <TableCell className="font-medium">{deal.name}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{deal.stageName || '-'}</Badge>
                      </TableCell>
                      <TableCell className="font-medium">
                        {formatCurrency(deal.value, deal.currency)}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {deal.contactName || '-'}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {deal.expectedCloseDate ? formatDate(deal.expectedCloseDate) : '-'}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {deal.tags?.map((tag) => (
                            <Badge
                              key={tag.id}
                              variant="secondary"
                              className="text-xs"
                              style={
                                tag.color
                                  ? {
                                      backgroundColor: `${tag.color}20`,
                                      color: tag.color,
                                      borderColor: tag.color,
                                    }
                                  : undefined
                              }
                            >
                              {tag.name}
                            </Badge>
                          ))}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              <div className="flex items-center justify-between">
                <p className="text-sm text-muted-foreground">
                  {t('crm.deals.totalCount', { count: total })}
                </p>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page <= 1}
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </Button>
                  <span className="text-sm text-muted-foreground">
                    {t('crm.pagination', { page, totalPages })}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page >= totalPages}
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  >
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </>
          )}
        </>
      )}

      <DealFormDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSubmit={handleCreate}
        stages={stages}
      />

      <AlertDialog open={showBulkDeleteDialog} onOpenChange={setShowBulkDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('crm.bulk.deleteTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('crm.bulk.deleteConfirm', { count: selectedIds.size })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={handleBulkDelete}
            >
              {t('common.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
