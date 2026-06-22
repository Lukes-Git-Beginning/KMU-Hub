import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronsUpDown, Check, Layers, AtSign } from 'lucide-react'
import type { EmailAccountInfo } from '@/api/email-types'

interface AccountSwitcherProps {
  accounts: EmailAccountInfo[]
  activeAccountId: string
  unifiedView: boolean
  onSelectAccount: (id: string) => void
  onSelectUnified: () => void
}

/**
 * Compact mailbox switcher shown at the top of the folder sidebar. Lets the
 * user jump between connected accounts or open the merged "all inboxes" view.
 */
export function AccountSwitcher({
  accounts,
  activeAccountId,
  unifiedView,
  onSelectAccount,
  onSelectUnified,
}: AccountSwitcherProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const active = accounts.find((a) => a.id === activeAccountId)

  // A single account → no switcher needed.
  if (accounts.length <= 1) return null

  const label = unifiedView
    ? t('mails.accounts.allInboxes', { defaultValue: 'Alle Eingänge' })
    : active?.email_address ?? ''

  return (
    <div className="relative mb-3">
      <button
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-2 rounded-xl border border-border bg-background px-3 py-2 text-left transition-colors hover:bg-secondary"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={t('mails.accounts.switch', { defaultValue: 'Konto wechseln' })}
      >
        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
          {unifiedView ? <Layers className="h-4 w-4" /> : <AtSign className="h-4 w-4" />}
        </div>
        <span className="min-w-0 flex-1 truncate text-sm font-medium text-foreground">{label}</span>
        <ChevronsUpDown className="h-4 w-4 shrink-0 text-muted-foreground" />
      </button>

      {open && (
        <>
          <button
            className="fixed inset-0 z-10 cursor-default"
            aria-hidden="true"
            tabIndex={-1}
            onClick={() => setOpen(false)}
          />
          <div
            className="absolute left-0 right-0 top-full z-20 mt-1 overflow-hidden rounded-xl border border-border bg-card shadow-lg"
            role="listbox"
          >
            <button
              onClick={() => {
                onSelectUnified()
                setOpen(false)
              }}
              className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors hover:bg-secondary"
              role="option"
              aria-selected={unifiedView}
            >
              <Layers className="h-4 w-4 shrink-0 text-muted-foreground" />
              <span className="flex-1 truncate text-foreground">
                {t('mails.accounts.allInboxes', { defaultValue: 'Alle Eingänge' })}
              </span>
              {unifiedView && <Check className="h-4 w-4 shrink-0 text-primary" />}
            </button>
            <div className="h-px bg-border" />
            {accounts.map((acc) => {
              const selected = !unifiedView && acc.id === activeAccountId
              return (
                <button
                  key={acc.id}
                  onClick={() => {
                    onSelectAccount(acc.id)
                    setOpen(false)
                  }}
                  className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors hover:bg-secondary"
                  role="option"
                  aria-selected={selected}
                >
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-primary-light text-[10px] font-medium text-primary">
                    {acc.display_name.slice(0, 2).toUpperCase()}
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-foreground">{acc.email_address}</p>
                    <p className="truncate text-[10px] text-muted-foreground">{acc.display_name}</p>
                  </div>
                  {selected && <Check className="h-4 w-4 shrink-0 text-primary" />}
                </button>
              )
            })}
          </div>
        </>
      )}
    </div>
  )
}
