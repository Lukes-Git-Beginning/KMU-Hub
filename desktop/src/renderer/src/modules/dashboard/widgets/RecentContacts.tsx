/**
 * RecentContacts widget -- shows the 5 most recently updated contacts.
 *
 * Each row displays initials avatar, name, company, and last update time.
 * Click navigates to the contact detail page.
 */
import { memo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Users } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { useContacts } from '@/api/hooks/useContacts'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function RecentContacts(_props: WidgetProps) {
  const navigate = useNavigate()
  const { data, isLoading, error, refetch } = useContacts({
    page_size: 5,
  })

  const contacts = data?.contacts ?? []

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <p className="text-sm text-muted-foreground">Fehler beim Laden</p>
          <Button variant="ghost" size="sm" className="mt-2" onClick={() => refetch()}>
            Erneut versuchen
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-3 p-4">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3">
            <Skeleton className="h-8 w-8 rounded-full" />
            <div className="flex-1 space-y-1">
              <Skeleton className="h-3.5 w-2/3" />
              <Skeleton className="h-3 w-1/3" />
            </div>
          </div>
        ))}
      </div>
    )
  }

  if (contacts.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <Users className="mx-auto h-8 w-8 text-muted-foreground/50" />
          <p className="mt-2 text-sm text-muted-foreground">Noch keine Kontakte</p>
        </div>
      </div>
    )
  }

  return (
    <div className="divide-y">
      {contacts.map((contact) => {
        const initials = `${(contact.firstName ?? '').charAt(0)}${(contact.lastName ?? '').charAt(0)}`.toUpperCase()
        const timeAgo = contact.updatedAt
          ? formatDistanceToNow(new Date(contact.updatedAt), { addSuffix: true, locale: de })
          : ''

        return (
          <div
            key={contact.id}
            className="flex cursor-pointer items-center gap-3 px-4 py-2.5 transition-colors hover:bg-accent/50"
            onClick={() => navigate(`/crm/contacts/${contact.id}`)}
          >
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
              {initials || '??'}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium">
                {contact.firstName} {contact.lastName}
              </p>
              {contact.companyName && (
                <p className="truncate text-xs text-muted-foreground">
                  {contact.companyName}
                </p>
              )}
            </div>
            <span className="shrink-0 text-xs text-muted-foreground">
              {timeAgo}
            </span>
          </div>
        )
      })}
    </div>
  )
}

export default memo(RecentContacts)
