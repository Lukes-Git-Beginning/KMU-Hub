/**
 * "Empfohlen von" — picks the contact that referred this one (self-referential).
 * Searchable popover over existing contacts; stored in the referrals overlay.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { UserPlus, X, Search } from 'lucide-react'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Input } from '@/components/ui/input'
import { useContacts } from '@/api/hooks/useContacts'
import { useReferralsStore } from '@/stores/referrals'

interface ReferredByFieldProps {
  contactId: string
}

export function ReferredByField({ contactId }: ReferredByFieldProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')

  const { data: contactsData } = useContacts({ page_size: 200 })
  const referrerId = useReferralsStore((s) => s.referredBy[contactId])
  const setReferrer = useReferralsStore((s) => s.setReferrer)
  const clearReferrer = useReferralsStore((s) => s.clearReferrer)

  const candidates = useMemo(() => {
    const list = (contactsData?.contacts ?? [])
      .filter((c) => c.id && c.id !== contactId)
      .map((c) => ({
        id: c.id as string,
        label: `${c.firstName ?? ''} ${c.lastName ?? ''}`.trim(),
        sublabel: c.companyName ?? '',
      }))
      .filter((c) => c.label)
    if (!query.trim()) return list.slice(0, 50)
    const q = query.toLowerCase()
    return list.filter((c) => c.label.toLowerCase().includes(q) || c.sublabel.toLowerCase().includes(q)).slice(0, 50)
  }, [contactsData, contactId, query])

  const referrer = useMemo(
    () => candidates.find((c) => c.id === referrerId) ?? null,
    [candidates, referrerId],
  )
  // Resolve name even when filtered out by the current query.
  const referrerLabel = useMemo(() => {
    if (!referrerId) return ''
    const c = (contactsData?.contacts ?? []).find((x) => x.id === referrerId)
    return c ? `${c.firstName ?? ''} ${c.lastName ?? ''}`.trim() : t('kontakte.referral.unknown')
  }, [contactsData, referrerId, t])

  if (referrerId) {
    return (
      <div className="flex items-center gap-2">
        <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-secondary px-2.5 py-1 text-sm text-foreground">
          {referrerLabel || referrer?.label}
          <button
            onClick={() => clearReferrer(contactId)}
            className="ml-0.5 rounded-full p-0.5 text-muted-foreground transition-colors hover:text-destructive"
            aria-label={t('kontakte.referral.remove')}
          >
            <X className="h-3 w-3" />
          </button>
        </span>
      </div>
    )
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button className="inline-flex items-center gap-1.5 rounded-full border border-dashed border-border px-2.5 py-1 text-sm text-muted-foreground transition-colors hover:border-primary hover:text-primary">
          <UserPlus className="h-3.5 w-3.5" />
          {t('kontakte.referral.add')}
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-64 p-2">
        <div className="relative mb-2">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('kontakte.referral.search')}
            className="h-9 pl-8 text-sm"
            autoFocus
          />
        </div>
        <div className="max-h-56 space-y-0.5 overflow-y-auto">
          {candidates.length === 0 ? (
            <p className="py-3 text-center text-xs text-muted-foreground">{t('kontakte.referral.noResults')}</p>
          ) : (
            candidates.map((c) => (
              <button
                key={c.id}
                onClick={() => {
                  setReferrer(contactId, c.id)
                  setOpen(false)
                  setQuery('')
                }}
                className="flex w-full flex-col items-start rounded-md px-2 py-1.5 text-left transition-colors hover:bg-accent"
              >
                <span className="text-sm text-foreground">{c.label}</span>
                {c.sublabel && <span className="text-xs text-muted-foreground">{c.sublabel}</span>}
              </button>
            ))
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
