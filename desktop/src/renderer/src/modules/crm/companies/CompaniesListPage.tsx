/**
 * Companies list page with search, sort, bulk-select, pagination, and navigation.
 *
 * Sort is client-side (name asc/desc, createdAt asc/desc) because the
 * /api/v1/companies endpoint has no sort_by param. All data for the current
 * page is already fetched — sorting just reorders the in-memory slice.
 */
import { useState, useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Plus,
  Search,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  ChevronDown,
  Building2,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/shared'
import { useCompanies, useCreateCompany, useDeleteCompany } from '@/api/hooks/useCompanies'
import { backendCompanyToUI } from './adapters'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Skeleton } from '@/components/ui/skeleton'
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
import { CompanyFormDialog, type CompanyFormData } from './CompanyFormDialog'
import { formatDate } from '@/lib/format'

const PAGE_SIZE = 20

type SortField = 'name' | 'createdAt'

interface SortState {
  field: SortField
  desc: boolean
}

function SortIcon({ field, sort }: { field: SortField; sort: SortState }) {
  if (sort.field !== field) return null
  return sort.desc ? (
    <ChevronDown className="ml-1 inline h-3.5 w-3.5 shrink-0" />
  ) : (
    <ChevronUp className="ml-1 inline h-3.5 w-3.5 shrink-0" />
  )
}

export default function CompaniesListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [sort, setSort] = useState<SortState>({ field: 'name', desc: false })
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [showBulkDeleteDialog, setShowBulkDeleteDialog] = useState(false)

  const createCompany = useCreateCompany()
  const deleteCompany = useDeleteCompany()

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  // Clear selection when page/search changes
  useEffect(() => {
    setSelectedIds(new Set())
  }, [page, debouncedSearch])

  const { data, isLoading, error, refetch } = useCompanies({
    page,
    page_size: PAGE_SIZE,
    search: debouncedSearch || undefined,
  })

  // Client-side sort of the current page slice
  const companies = useMemo(() => {
    const raw = (data?.companies ?? []).map(backendCompanyToUI)
    return [...raw].sort((a, b) => {
      let cmp = 0
      if (sort.field === 'name') {
        cmp = (a.name ?? '').localeCompare(b.name ?? '', 'de')
      } else {
        const aDate = a.createdAt ? new Date(a.createdAt).getTime() : 0
        const bDate = b.createdAt ? new Date(b.createdAt).getTime() : 0
        cmp = aDate - bDate
      }
      return sort.desc ? -cmp : cmp
    })
  }, [data?.companies, sort])

  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  function toggleSort(field: SortField) {
    setSort((prev) =>
      prev.field === field ? { field, desc: !prev.desc } : { field, desc: false }
    )
  }

  // Selection helpers
  const allOnPageSelected =
    companies.length > 0 && companies.every((c) => c.id && selectedIds.has(c.id))
  const someOnPageSelected =
    companies.some((c) => c.id && selectedIds.has(c.id)) && !allOnPageSelected

  function toggleSelectAll() {
    if (allOnPageSelected) {
      const next = new Set(selectedIds)
      companies.forEach((c) => { if (c.id) next.delete(c.id) })
      setSelectedIds(next)
    } else {
      const next = new Set(selectedIds)
      companies.forEach((c) => { if (c.id) next.add(c.id) })
      setSelectedIds(next)
    }
  }

  function toggleSelectOne(id: string) {
    const next = new Set(selectedIds)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    setSelectedIds(next)
  }

  async function handleCreate(form: CompanyFormData) {
    const addressParts = [form.street, [form.zip, form.city].filter(Boolean).join(' '), form.country].filter(Boolean)
    try {
      await createCompany.mutateAsync({
        name: form.name,
        website: form.website || undefined,
        industry: form.industry || undefined,
        address: addressParts.join(', ') || undefined,
        notes: form.notes || undefined,
        custom_fields: {
          ...(form.phone ? { _phone: form.phone } : {}),
          ...(form.email ? { _email: form.email } : {}),
          ...(form.size ? { _size: form.size } : {}),
          ...(form.tags.length > 0 ? { _tags: form.tags } : {}),
        },
      })
      toast.success(t('crm.companies.created'))
    } catch {
      toast.error(t('crm.companies.createError'))
    }
  }

  async function handleBulkDelete() {
    const ids = Array.from(selectedIds)
    let errorCount = 0
    for (const id of ids) {
      try {
        await deleteCompany.mutateAsync(id)
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
            {t('crm.companies.loadError')}
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
      <PageHeader
        title={t('crm.companies.title')}
        description={t('crm.companies.count', { count: data?.total ?? 0 })}
        icon={Building2}
        moduleId="contacts"
        actions={
          <Button onClick={() => setShowCreateDialog(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            {t('crm.companies.new')}
          </Button>
        }
      />

      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder={t('crm.companies.search')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-9"
        />
      </div>

      {/* Bulk action bar */}
      {selectedIds.size > 0 && (
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
      ) : companies.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16">
          <Building2 className="h-12 w-12 text-muted-foreground" />
          <p className="mt-4 text-lg font-medium text-foreground">
            {t('crm.companies.noResults')}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {debouncedSearch
              ? t('crm.tryDifferentSearch')
              : t('crm.companies.emptyHint')}
          </p>
          {!debouncedSearch && (
            <Button className="mt-4 gap-2" onClick={() => setShowCreateDialog(true)}>
              <Plus className="h-4 w-4" />
              {t('crm.companies.createFirst')}
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
                <TableHead>{t('crm.field.website')}</TableHead>
                <TableHead>{t('crm.field.industry')}</TableHead>
                <TableHead>{t('crm.contacts.title')}</TableHead>
                <TableHead>{t('crm.field.tags')}</TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => toggleSort('createdAt')}
                >
                  {t('crm.field.created')}
                  <SortIcon field="createdAt" sort={sort} />
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {companies.map((company) => (
                <TableRow
                  key={company.id}
                  className="cursor-pointer"
                  onClick={() => company.id && navigate(`/kontakte/firmen/${company.id}`)}
                  data-state={company.id && selectedIds.has(company.id) ? 'selected' : undefined}
                >
                  <TableCell className="pr-0" onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={!!(company.id && selectedIds.has(company.id))}
                      onCheckedChange={() => company.id && toggleSelectOne(company.id)}
                      aria-label={company.name}
                    />
                  </TableCell>
                  <TableCell className="font-medium">{company.name}</TableCell>
                  <TableCell className="text-muted-foreground">
                    {company.website ? (
                      <span
                        className="text-primary hover:underline"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {company.website}
                      </span>
                    ) : (
                      '-'
                    )}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {company.industry || '-'}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {company.contactCount ?? 0}
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {company.tags?.map((tag) => (
                        <Badge
                          key={tag.id}
                          variant="secondary"
                          className="text-xs"
                          style={
                            tag.color
                              ? { backgroundColor: `${tag.color}20`, color: tag.color, borderColor: tag.color }
                              : undefined
                          }
                        >
                          {tag.name}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {company.createdAt ? formatDate(company.createdAt) : '-'}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {t('crm.companies.totalCount', { count: total })}
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

      <CompanyFormDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSubmit={handleCreate}
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
