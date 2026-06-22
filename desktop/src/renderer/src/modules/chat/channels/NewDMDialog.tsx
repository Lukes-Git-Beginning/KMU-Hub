/**
 * Dialog for starting a direct message conversation.
 *
 * One selected person → 1:1 DM, multiple → group DM (KO-2). The internal team
 * chat owns its own conversations, so group chats live here (external channels
 * arrive via integrations). People are picked from the employee directory.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Search, Users, X } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useStartConversation } from '@/api/hooks/useChannels'
import { EMPLOYEES } from '@/mocks/mock-db'
import { CURRENT_USER } from '@/mocks/data/shared-ids'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface NewDMDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (channelId: string) => void
}

const directory = EMPLOYEES.filter((e) => `usr-${e.id}` !== CURRENT_USER.id).map((e) => ({
  userId: `usr-${e.id}`,
  name: `${e.firstName} ${e.lastName}`,
  initials: e.initials,
  role: e.role,
}))

export function NewDMDialog({ open, onOpenChange, onCreated }: NewDMDialogProps) {
  const { t } = useTranslation()
  const [filter, setFilter] = useState('')
  const [selected, setSelected] = useState<string[]>([])
  const startConversation = useStartConversation()

  const filtered = useMemo(() => {
    if (!filter.trim()) return directory
    const q = filter.toLowerCase()
    return directory.filter((p) => p.name.toLowerCase().includes(q))
  }, [filter])

  const selectedPeople = useMemo(
    () => directory.filter((p) => selected.includes(p.userId)),
    [selected],
  )
  const isGroup = selected.length > 1

  const toggle = (userId: string) => {
    setSelected((prev) =>
      prev.includes(userId) ? prev.filter((id) => id !== userId) : [...prev, userId],
    )
  }

  const reset = () => {
    setFilter('')
    setSelected([])
  }

  const handleSubmit = async () => {
    if (selected.length === 0) return
    try {
      const result = await startConversation.mutateAsync(selected)
      const channelId = result?.channel?.id
      if (channelId) onCreated(channelId)
      reset()
    } catch {
      // surfaced via mutation state
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(val) => {
        if (!val) reset()
        onOpenChange(val)
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('chat.newDm.title')}</DialogTitle>
          <DialogDescription>
            {isGroup ? t('chat.newDm.groupHint') : t('chat.newDm.description')}
          </DialogDescription>
        </DialogHeader>

        {selectedPeople.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {selectedPeople.map((p) => (
              <span
                key={p.userId}
                className="inline-flex items-center gap-1 rounded-full bg-secondary py-0.5 pl-2 pr-1 text-xs text-secondary-foreground"
              >
                {p.name}
                <button
                  type="button"
                  onClick={() => toggle(p.userId)}
                  className="rounded-full p-0.5 hover:bg-accent"
                  aria-label={t('chat.newDm.removePerson')}
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            ))}
          </div>
        )}

        <div className="relative">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={t('chat.newDm.searchPlaceholder')}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="pl-9 h-9"
            autoFocus
          />
        </div>

        <ScrollArea className="h-64 -mx-1 px-1">
          <div className="space-y-0.5">
            {filtered.map((p) => {
              const isSelected = selected.includes(p.userId)
              return (
                <button
                  key={p.userId}
                  type="button"
                  onClick={() => toggle(p.userId)}
                  className={cn(
                    'flex w-full items-center gap-3 rounded-md px-2 py-1.5 text-left text-sm transition-colors',
                    isSelected ? 'bg-secondary' : 'hover:bg-accent',
                  )}
                >
                  <Avatar className="h-7 w-7">
                    <AvatarFallback className="text-[11px]">{p.initials}</AvatarFallback>
                  </Avatar>
                  <span className="flex-1 truncate text-foreground">{p.name}</span>
                  <span
                    className={cn(
                      'flex h-4 w-4 items-center justify-center rounded-full border',
                      isSelected ? 'border-primary bg-primary text-primary-foreground' : 'border-border',
                    )}
                  >
                    {isSelected && <Check className="h-3 w-3" />}
                  </span>
                </button>
              )
            })}
            {filtered.length === 0 && (
              <p className="py-6 text-center text-xs text-muted-foreground">
                {t('chat.newDm.noResults')}
              </p>
            )}
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button type="button" disabled={selected.length === 0 || startConversation.isPending} onClick={handleSubmit}>
            {isGroup && <Users className="mr-1.5 h-4 w-4" />}
            {startConversation.isPending
              ? t('chat.newDm.creating')
              : isGroup
                ? t('chat.newDm.startGroup', { count: selected.length })
                : t('chat.newDm.startDm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
