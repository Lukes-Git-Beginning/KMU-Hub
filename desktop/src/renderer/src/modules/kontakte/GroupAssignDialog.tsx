/**
 * Assign an existing contact to one or more groups (toggle membership).
 *
 * Contacts usually exist before a group is created, so this lets the user pick
 * which groups a contact belongs to from the full group list.
 */
import { useTranslation } from 'react-i18next'
import { FolderOpen, Settings2 } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { useContactsStore, type Contact } from '@/stores/contacts'

interface GroupAssignDialogProps {
  contact: Contact | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onManageGroups?: () => void
}

export function GroupAssignDialog({ contact, open, onOpenChange, onManageGroups }: GroupAssignDialogProps) {
  const { t } = useTranslation()
  const { groups, addContactToGroup, removeContactFromGroup } = useContactsStore()

  if (!contact) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('kontakte.groupAssign.title', { name: `${contact.firstName} ${contact.lastName}` })}</DialogTitle>
          <DialogDescription>{t('kontakte.groupAssign.description')}</DialogDescription>
        </DialogHeader>

        {groups.length === 0 ? (
          <div className="flex flex-col items-center gap-3 py-6 text-center">
            <FolderOpen className="h-8 w-8 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">{t('kontakte.groupAssign.empty')}</p>
            {onManageGroups && (
              <Button variant="outline" size="sm" onClick={() => { onOpenChange(false); onManageGroups() }}>
                <Settings2 className="mr-1.5 h-4 w-4" />
                {t('kontakte.sidebar.manageGroups')}
              </Button>
            )}
          </div>
        ) : (
          <div className="space-y-1 py-1">
            {groups.map((g) => {
              const member = g.contactIds.includes(contact.id)
              return (
                <label
                  key={g.id}
                  className="flex cursor-pointer items-center gap-3 rounded-lg border border-border px-3 py-2.5 transition-colors hover:bg-accent"
                >
                  <input
                    type="checkbox"
                    checked={member}
                    onChange={(e) => {
                      if (e.target.checked) addContactToGroup(g.id, contact.id)
                      else removeContactFromGroup(g.id, contact.id)
                    }}
                    className="h-4 w-4 accent-primary"
                  />
                  <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: g.color }} />
                  <span className="flex-1 truncate text-sm text-foreground">{g.name}</span>
                  <span className="text-xs text-muted-foreground">{g.contactIds.length}</span>
                </label>
              )
            })}
          </div>
        )}

        <div className="flex justify-end pt-1">
          <Button onClick={() => onOpenChange(false)}>{t('common.done')}</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
