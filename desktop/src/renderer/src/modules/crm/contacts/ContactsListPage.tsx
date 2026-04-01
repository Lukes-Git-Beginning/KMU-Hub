/**
 * Contacts list page with search, pagination, visibility, and import/export.
 *
 * Displays contacts in a table with columns for name, email, phone,
 * company, tags, visibility, and creation date. Supports search,
 * pagination, visibility filtering, import wizard, and export dialog.
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
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

const PAGE_SIZE = 20

type VisibilityFilter = 'all' | 'shared' | 'personal'

export default function ContactsListPage() {
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
      toast.success('Kontakt erstellt')
    } catch {
      toast.error('Fehler beim Erstellen des Kontakts')
    }
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Fehler beim Laden der Kontakte
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : 'Ein unerwarteter Fehler ist aufgetreten.'}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            Erneut versuchen
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-4">
      {/* Header */}
      <PageHeader
        title="Kontakte"
        description={`${total} Kontakte`}
        icon={Contact}
        moduleId="contacts"
        actions={
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={() => setImportOpen(true)} className="gap-2">
              <Upload className="h-4 w-4" />
              Importieren
            </Button>
            <Button
              variant="outline"
              onClick={() => setExportOpen(true)}
              className="gap-2"
            >
              <Download className="h-4 w-4" />
              Exportieren
            </Button>
            <Button onClick={() => setShowCreateDialog(true)} className="gap-2">
              <Plus className="h-4 w-4" />
              Neuer Kontakt
            </Button>
          </div>
        }
      />

      {/* Search bar + visibility filter */}
      <div className="flex items-center gap-3">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Kontakte suchen..."
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
            <SelectValue placeholder="Sichtbarkeit" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Alle Kontakte</SelectItem>
            <SelectItem value="shared">Geteilte</SelectItem>
            <SelectItem value="personal">Persönliche</SelectItem>
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
            Keine Kontakte gefunden
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {debouncedSearch
              ? 'Versuche einen anderen Suchbegriff.'
              : 'Erstelle deinen ersten Kontakt oder importiere Kontakte.'}
          </p>
          {!debouncedSearch && (
            <div className="mt-4 flex gap-2">
              <Button className="gap-2" onClick={() => setShowCreateDialog(true)}>
                <Plus className="h-4 w-4" />
                Ersten Kontakt erstellen
              </Button>
              <Button
                variant="outline"
                className="gap-2"
                onClick={() => setImportOpen(true)}
              >
                <Upload className="h-4 w-4" />
                Importieren
              </Button>
            </div>
          )}
        </div>
      ) : (
        <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-8"></TableHead>
                <TableHead>Name</TableHead>
                <TableHead>E-Mail</TableHead>
                <TableHead>Telefon</TableHead>
                <TableHead>Unternehmen</TableHead>
                <TableHead>Tags</TableHead>
                <TableHead>Erstellt</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredContacts.map((contact) => {
                const visibility = (contact as unknown as { visibility?: string })
                  .visibility
                return (
                  <TableRow
                    key={contact.id}
                    className="cursor-pointer"
                    onClick={() => navigate(`/crm/contacts/${contact.id}`)}
                  >
                    <TableCell className="w-8 pr-0">
                      {visibility === 'personal' ? (
                        <Lock
                          className="h-3.5 w-3.5 text-amber-500"
                          aria-label="Persönlich"
                        />
                      ) : (
                        <Globe
                          className="h-3.5 w-3.5 text-muted-foreground"
                          aria-label="Geteilt"
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
                      {contact.createdAt
                        ? new Date(contact.createdAt).toLocaleDateString('de-DE')
                        : '-'}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>

          {/* Pagination */}
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {total} Kontakt{total !== 1 ? 'e' : ''} gesamt
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
                Seite {page} von {totalPages}
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
