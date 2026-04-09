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
import { Search, UserPlus } from 'lucide-react'
import { useContacts } from '@/api/hooks/useContacts'

interface AddContactsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (contactIds: string[]) => void
  isLoading?: boolean
}

export default function AddContactsDialog({
  open,
  onOpenChange,
  onSubmit,
  isLoading,
}: AddContactsDialogProps) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const { data, isLoading: contactsLoading } = useContacts({
    search: search || undefined,
    page_size: 50,
  })

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
    onSubmit(Array.from(selected))
    setSelected(new Set())
    setSearch('')
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserPlus className="h-5 w-5" />
            {t('dialer.campaign.contacts.add')}
          </DialogTitle>
        </DialogHeader>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Kontakt suchen..."
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
              Keine Kontakte gefunden
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

        {/* Footer */}
        <div className="flex items-center justify-between border-t pt-3">
          <span className="text-xs text-muted-foreground">
            {selected.size} ausgewählt
          </span>
          <div className="flex gap-2">
            <Button variant="ghost" onClick={() => onOpenChange(false)}>
              Abbrechen
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={selected.size === 0 || isLoading}
              loading={isLoading}
            >
              {selected.size} Kontakte hinzufügen
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
