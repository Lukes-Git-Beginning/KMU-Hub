import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { Skeleton } from '@/components/ui/skeleton'
import { Search, UserPlus, Filter, Users } from 'lucide-react'
import { useContacts } from '@/api/hooks/useContacts'
import { useSavedFilters } from '@/api/hooks/useSavedFilters'
import { cn } from '@/lib/cn'

type ImportMode = 'manual' | 'filter'

export interface AddContactsPayload {
  contact_ids?: string[]
  filter_id?: string
}

interface AddContactsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (payload: AddContactsPayload) => void
  isLoading?: boolean
}

export default function AddContactsDialog({
  open,
  onOpenChange,
  onSubmit,
  isLoading,
}: AddContactsDialogProps) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<ImportMode>('manual')
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [selectedFilterId, setSelectedFilterId] = useState<string | null>(null)

  const { data, isLoading: contactsLoading } = useContacts({
    search: search || undefined,
    page_size: 50,
  })

  const { data: filters, isLoading: filtersLoading } = useSavedFilters('contact')

  const contacts = (data as { contacts?: Array<{ id: string; first_name: string; last_name: string; email?: string; company_name?: string }> })?.contacts ?? []

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleSubmit = () => {
    if (mode === 'filter' && selectedFilterId) {
      onSubmit({ filter_id: selectedFilterId })
    } else {
      onSubmit({ contact_ids: Array.from(selected) })
    }
    setSelected(new Set())
    setSearch('')
    setSelectedFilterId(null)
  }

  const canSubmit = mode === 'filter' ? !!selectedFilterId : selected.size > 0

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserPlus className="h-5 w-5" />
            {t('dialer.campaign.contacts.add')}
          </DialogTitle>
        </DialogHeader>

        {/* Mode toggle */}
        <div className="flex gap-1 rounded-lg bg-muted p-1">
          <button
            onClick={() => setMode('manual')}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
              mode === 'manual'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            <Users className="h-3.5 w-3.5" />
            {t('dialer.campaign.contacts.modeManual')}
          </button>
          <button
            onClick={() => setMode('filter')}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
              mode === 'filter'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            <Filter className="h-3.5 w-3.5" />
            {t('dialer.campaign.contacts.modeFilter')}
          </button>
        </div>

        {mode === 'manual' ? (
          <>
            {/* Search */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('dialer.campaign.contacts.searchPlaceholder')}
                className="pl-9"
              />
            </div>

            {/* Contact list */}
            <div className="flex-1 overflow-y-auto -mx-6 px-6 space-y-1 min-h-[200px] max-h-[400px]">
              {contactsLoading ? (
                Array.from({ length: 6 }).map((_, i) => (
                  <Skeleton key={i} className="h-11 w-full" />
                ))
              ) : contacts.length === 0 ? (
                <p className="py-8 text-center text-sm text-muted-foreground">
                  {t('dialer.campaign.contacts.noContactsFound')}
                </p>
              ) : (
                contacts.map((contact) => (
                  <label
                    key={contact.id}
                    className="flex items-center gap-3 rounded-lg px-3 py-2.5 hover:bg-accent cursor-pointer transition-colors"
                  >
                    <Checkbox
                      checked={selected.has(contact.id)}
                      onCheckedChange={() => toggle(contact.id)}
                    />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">
                        {contact.first_name} {contact.last_name}
                      </p>
                      {contact.company_name && (
                        <p className="text-xs text-muted-foreground truncate">
                          {contact.company_name}
                        </p>
                      )}
                    </div>
                  </label>
                ))
              )}
            </div>
          </>
        ) : (
          /* Filter mode */
          <div className="flex-1 overflow-y-auto -mx-6 px-6 space-y-1 min-h-[200px] max-h-[400px]">
            {filtersLoading ? (
              Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-14 w-full" />
              ))
            ) : !filters || filters.length === 0 ? (
              <div className="py-8 text-center space-y-2">
                <Filter className="h-8 w-8 text-muted-foreground/30 mx-auto" />
                <p className="text-sm text-muted-foreground">
                  {t('dialer.campaign.contacts.noFiltersFound')}
                </p>
              </div>
            ) : (
              filters.map((filter) => (
                <button
                  key={filter.id}
                  onClick={() => setSelectedFilterId(
                    selectedFilterId === filter.id ? null : filter.id!,
                  )}
                  className={cn(
                    'flex w-full items-center gap-3 rounded-lg px-3 py-3 text-left transition-colors',
                    selectedFilterId === filter.id
                      ? 'bg-primary/10 border border-primary/30'
                      : 'hover:bg-accent border border-transparent',
                  )}
                >
                  <Filter className={cn(
                    'h-4 w-4 shrink-0',
                    selectedFilterId === filter.id ? 'text-primary' : 'text-muted-foreground',
                  )} />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{filter.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {t('dialer.campaign.contacts.filterDescription')}
                    </p>
                  </div>
                </button>
              ))
            )}
          </div>
        )}

        {/* Footer */}
        <div className="flex items-center justify-between border-t pt-3">
          <span className="text-xs text-muted-foreground">
            {mode === 'manual'
              ? t('dialer.campaign.contacts.selectedCount', { count: selected.size })
              : selectedFilterId
                ? t('dialer.campaign.contacts.filterSelected')
                : t('dialer.campaign.contacts.noFilterSelected')
            }
          </span>
          <div className="flex gap-2">
            <Button variant="ghost" onClick={() => onOpenChange(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={!canSubmit || isLoading}
              loading={isLoading}
            >
              {mode === 'manual'
                ? t('dialer.campaign.contacts.addCount', { count: selected.size })
                : t('dialer.campaign.contacts.importFromFilter')
              }
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
