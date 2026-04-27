"use memo"
/**
 * Contacts list page with search, pagination, visibility, and import/export.
 *
 * Displays contacts in a table with columns for name, email, phone,
 * company, tags, visibility, and creation date. Supports search,
 * pagination, visibility filtering, import wizard, and export dialog.
 */
import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useVirtualizer } from '@tanstack/react-virtual'
import { useTranslation } from 'react-i18next'
import {
  Plus,
  Search,
  ChevronLeft,
  ChevronRight,
  Users,
  Upload,
  Download,
  Globe,
  Lock,
  Contact,
} from 'lucide-react'
import { toast } from 'sonner'
import { useContacts, useCreateContact } from '@/api/hooks/useContacts'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import ImportWizard from '@/modules/mails/ImportWizard'
import ExportDialog from '@/modules/mails/ExportDialog'
import { PageHeader } from '@/components/shared'
import { ContactFormDialog, type ContactFormData } from './ContactFormDialog'
import { formatDate } from '@/lib/format'

const PAGE_SIZE = 20

type VisibilityFilter = 'all' | 'shared' | 'personal'

export default function ContactsListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [visibilityFilter, setVisibilityFilter] = useState<VisibilityFilter>('all')
  const [importOpen, setImportOpen] = useState(false)
  const [exportOpen, setExportOpen] = useState(false)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const createContact = useCreateContact()

  // Debounce search input (300ms)
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading, error, refetch } = useContacts({
    page,
    page_size: PAGE_SIZE,
    search: debouncedSearch || undefined,
  })

  const contacts = data?.contacts ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // Client-side visibility filtering (until backend supports query param)
  const filteredContacts =
    visibilityFilter === 'all'
      ? contacts
      : contacts.filter(
          (c) =>
            (c as unknown as { visibility?: string }).visibility === visibilityFilter,
        )

  const tableContainerRef = useRef<HTMLDivElement>(null)
  const ROW_HEIGHT = 52
  const virtualizer = useVirtualizer({
    count: filteredContacts.length,
    getScrollElement: () => tableContainerRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 5,
  })

  async function handleCreate(form: ContactFormData) {
    try {
      await createContact.mutateAsync({
        first_name: form.firstName,
        last_name: form.lastName,
        email: form.email || undefined,
        phone: form.phone || undefined,
        title: form.title || undefined,
        notes: form.notes || undefined,
        custom_fields: {
          ...(form.salutation ? { _salutation: form.salutation } : {}),
          ...(form.mobile ? { _mobile: form.mobile } : {}),
          ...(form.company ? { _company: form.company } : {}),
          ...(form.jobTitle ? { _jobTitle: form.jobTitle } : {}),
          ...(form.department ? { _department: form.department } : {}),
          ...(form.category !== 'customer' ? { _category: form.category } : {}),
          ...(form.status !== 'active' ? { _status: form.status } : {}),
          ...((form.street || form.zip || form.city) ? {
            _address: { street: form.street, zip: form.zip, city: form.city, country: form.country },
          } : {}),
          ...(form.website ? { _website: form.website } : {}),
          ...((form.linkedin || form.xing) ? {
            _socialMedia: { linkedin: form.linkedin, xing: form.xing },
          } : {}),
          ...(form.tags.length > 0 ? { _tags: form.tags } : {}),
        },
      })
      toast.success(t('crm.contacts.created'))
    } catch {
      toast.error(t('crm.contacts.createError'))
    }
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            {t('crm.contacts.loadError')}
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
        title={t('crm.contacts.title')}
        description={t('crm.contacts.count', { count: total })}
        icon={Contact}
        moduleId="contacts"
        actions={
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={() => setImportOpen(true)} className="gap-2">
              <Upload className="h-4 w-4" />
              {t('crm.import')}
            </Button>
            <Button
              variant="outline"
              onClick={() => setExportOpen(true)}
              className="gap-2"
            >
              <Download className="h-4 w-4" />
              {t('crm.export')}
            </Button>
            <Button onClick={() => setShowCreateDialog(true)} className="gap-2">
              <Plus className="h-4 w-4" />
              {t('crm.contacts.new')}
            </Button>
          </div>
        }
      />

      {/* Search bar + visibility filter */}
      <div className="flex items-center gap-3">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            aria-label={t('crm.contacts.search')}
            placeholder={t('crm.contacts.search')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select
          value={visibilityFilter}
          onValueChange={(v) => {
            setVisibilityFilter(v as VisibilityFilter)
            setPage(1)
          }}
        >
          <SelectTrigger className="w-44">
            <SelectValue placeholder={t('crm.contacts.visibility')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('crm.contacts.visibilityAll')}</SelectItem>
            <SelectItem value="shared">{t('crm.contacts.visibilityShared')}</SelectItem>
            <SelectItem value="personal">{t('crm.contacts.visibilityPersonal')}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Table */}
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : filteredContacts.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16">
          <Users className="h-12 w-12 text-muted-foreground" />
          <p className="mt-4 text-lg font-medium text-foreground">
            {t('crm.contacts.noResults')}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {debouncedSearch
              ? t('crm.tryDifferentSearch')
              : t('crm.contacts.emptyHint')}
          </p>
          {!debouncedSearch && (
            <div className="mt-4 flex gap-2">
              <Button className="gap-2" onClick={() => setShowCreateDialog(true)}>
                <Plus className="h-4 w-4" />
                {t('crm.contacts.createFirst')}
              </Button>
              <Button
                variant="outline"
                className="gap-2"
                onClick={() => setImportOpen(true)}
              >
                <Upload className="h-4 w-4" />
                {t('crm.import')}
              </Button>
            </div>
          )}
        </div>
      ) : (
        <>
          {/* Scrollable container for virtualized table body */}
          <div
            ref={tableContainerRef}
            style={{ overflow: 'auto', maxHeight: 'calc(100vh - 320px)' }}
          >
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-8"><span className="sr-only">{t('crm.contacts.visibility')}</span></TableHead>
                  <TableHead>{t('crm.field.name')}</TableHead>
                  <TableHead>{t('crm.field.email')}</TableHead>
                  <TableHead>{t('crm.field.phone')}</TableHead>
                  <TableHead>{t('crm.field.company')}</TableHead>
                  <TableHead>{t('crm.field.tags')}</TableHead>
                  <TableHead>{t('crm.field.created')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody style={{ position: 'relative', height: `${virtualizer.getTotalSize()}px` }}>
                {virtualizer.getVirtualItems().map((virtualRow) => {
                  const contact = filteredContacts[virtualRow.index]
                  const visibility = (contact as unknown as { visibility?: string }).visibility
                  return (
                    <TableRow
                      key={virtualRow.key}
                      data-index={virtualRow.index}
                      tabIndex={0}
                      className="cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:ring-inset"
                      style={{
                        position: 'absolute',
                        top: 0,
                        left: 0,
                        width: '100%',
                        height: `${virtualRow.size}px`,
                        transform: `translateY(${virtualRow.start}px)`,
                      }}
                      onClick={() => navigate(`/crm/contacts/${contact.id}`)}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); navigate(`/crm/contacts/${contact.id}`) } }}
                    >
                      <TableCell className="w-8 pr-0">
                        {visibility === 'personal' ? (
                          <Lock
                            className="h-3.5 w-3.5 text-warning"
                            aria-label={t('crm.contacts.visibilityPersonal')}
                          />
                        ) : (
                          <Globe
                            className="h-3.5 w-3.5 text-muted-foreground"
                            aria-label={t('crm.contacts.visibilityShared')}
                          />
                        )}
                      </TableCell>
                      <TableCell className="font-medium">
                        {contact.firstName} {contact.lastName}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {contact.email || '-'}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {contact.phone || '-'}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {contact.companyName || '-'}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {contact.tags?.map((tag) => (
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
                      <TableCell className="text-muted-foreground">
                        {contact.createdAt ? formatDate(contact.createdAt) : '-'}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>

          {/* Pagination */}
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {t('crm.contacts.totalCount', { count: total })}
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

      {/* Import/Export dialogs */}
      <ImportWizard open={importOpen} onOpenChange={setImportOpen} />
      <ExportDialog
        open={exportOpen}
        onOpenChange={setExportOpen}
        contactIds={contacts.map((c) => c.id).filter(Boolean) as string[]}
      />

      <ContactFormDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSubmit={handleCreate}
      />
    </div>
  )
}
