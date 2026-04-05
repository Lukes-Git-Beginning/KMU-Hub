/**
 * Companies list page with search, pagination, and navigation to detail.
 *
 * Displays companies in a table with columns for name, website,
 * contacts count, tags, and creation date.
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Plus, Search, ChevronLeft, ChevronRight, Building2 } from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/shared'
import { useCompanies, useCreateCompany } from '@/api/hooks/useCompanies'
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
import { CompanyFormDialog, type CompanyFormData } from './CompanyFormDialog'

const PAGE_SIZE = 20

export default function CompaniesListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const createCompany = useCreateCompany()

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading, error, refetch } = useCompanies({
    page,
    page_size: PAGE_SIZE,
    search: debouncedSearch || undefined,
  })

  const companies = data?.companies ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

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
                <TableHead>{t('crm.field.name')}</TableHead>
                <TableHead>{t('crm.field.website')}</TableHead>
                <TableHead>{t('crm.field.industry')}</TableHead>
                <TableHead>{t('crm.contacts.title')}</TableHead>
                <TableHead>{t('crm.field.tags')}</TableHead>
                <TableHead>{t('crm.field.created')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {companies.map((company) => (
                <TableRow
                  key={company.id}
                  className="cursor-pointer"
                  onClick={() => navigate(`/crm/companies/${company.id}`)}
                >
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
                    {company.createdAt
                      ? new Date(company.createdAt).toLocaleDateString('de-DE')
                      : '-'}
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
    </div>
  )
}
