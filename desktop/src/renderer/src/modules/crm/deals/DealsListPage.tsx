/**
 * Deals list page with toggle between table view and pipeline (Kanban) view.
 *
 * Table view shows deals with columns for name, stage, value, contact,
 * expected close date, and tags. Pipeline view renders the DealPipelineView
 * component with horizontal stage columns.
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Plus,
  Search,
  ChevronLeft,
  ChevronRight,
  LayoutList,
  Columns3,
  TrendingUp,
} from 'lucide-react'
import { toast } from 'sonner'
import { useDeals, useCreateDeal } from '@/api/hooks/useDeals'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import DealPipelineView from './DealPipelineView'
import { PageHeader } from '@/components/shared'
import { DealFormDialog, type DealFormData } from './DealFormDialog'

const PAGE_SIZE = 20

function formatCurrency(value?: number, currency?: string): string {
  if (value == null) return '-'
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: currency || 'EUR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

export default function DealsListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [view, setView] = useState<'list' | 'pipeline'>('list')
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const createDeal = useCreateDeal()
  const { data: stagesData } = usePipelineStages()
  const stages = stagesData?.stages ?? []

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading, error, refetch } = useDeals({
    page,
    page_size: PAGE_SIZE,
    search: debouncedSearch || undefined,
  })

  const deals = data?.deals ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

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
                className="rounded-l-none"
                onClick={() => setView('pipeline')}
              >
                <Columns3 className="h-4 w-4" />
              </Button>
            </div>
            <Button onClick={() => setShowCreateDialog(true)} className="gap-2">
              <Plus className="h-4 w-4" />
              {t('crm.deals.new')}
            </Button>
          </div>
        }
      />

      {view === 'pipeline' ? (
        <DealPipelineView />
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
              {!debouncedSearch && (
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
                    <TableHead>{t('crm.field.name')}</TableHead>
                    <TableHead>{t('crm.deals.stage')}</TableHead>
                    <TableHead>{t('crm.deals.value')}</TableHead>
                    <TableHead>{t('crm.field.contact')}</TableHead>
                    <TableHead>{t('crm.deals.expectedClose')}</TableHead>
                    <TableHead>{t('crm.field.tags')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {deals.map((deal) => (
                    <TableRow
                      key={deal.id}
                      className="cursor-pointer"
                      onClick={() => navigate(`/crm/deals/${deal.id}`)}
                    >
                      <TableCell className="font-medium">
                        {deal.name}
                      </TableCell>
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
                        {deal.expectedCloseDate
                          ? new Date(deal.expectedCloseDate).toLocaleDateString(
                              'de-DE'
                            )
                          : '-'}
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
    </div>
  )
}
