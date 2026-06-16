/**
 * CustomerAccountSection — wiederverwendbarer „Kundenkonto"-Block für die
 * Beleg-Detail-Modals (Rechnung/Angebot/Gutschrift/Abo). Zeigt den Kunden,
 * Konto-Summen (Umsatz/offen), alle weiteren Belege desselben Kunden (klickbar
 * → öffnet das jeweilige Detail-Modal) und einen Sprung in die CRM-360°-Ansicht.
 *
 * Richtung Kontakte-360°, bleibt aber robust: ohne CRM-Treffer ist der Button
 * deaktiviert, ohne Navigations-Context sind die Belege nur informativ.
 */
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowUpRight, ChevronRight, FileText, Receipt, Users } from 'lucide-react'
import type { CustomerSnapshot } from '@/types/finance-types'
import { formatMoney } from '@/stores/finance'
import { formatDate } from '@/lib/format'
import { useCustomerAccount, useContactMatch } from './lib/customer-account'
import { useFinanceDetailNavOptional, type FinanceDocKind } from './FinanceDetailNav'

interface AccountDoc {
  kind: FinanceDocKind
  id: string
  number: string
  typeLabelKey: string
  amount: number
  date: string
  currency?: string
}

interface CustomerAccountSectionProps {
  customer: CustomerSnapshot
  /** ID des aktuell offenen Belegs — wird aus der „weitere Belege"-Liste gefiltert. */
  currentDocId?: string
}

export function CustomerAccountSection({ customer, currentDocId }: CustomerAccountSectionProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const nav = useFinanceDetailNavOptional()
  const account = useCustomerAccount(customer)
  const contactId = useContactMatch(customer)

  const gross = (d: { tax_breakdown?: { gross_total?: string }; total_gross?: number | string }) =>
    Number(d.tax_breakdown?.gross_total ?? d.total_gross ?? 0)

  const docs: AccountDoc[] = [
    ...account.invoices.map((i) => ({
      kind: 'invoice' as const,
      id: i.id,
      number: i.invoice_number,
      typeLabelKey: 'finanzen.customerAccount.typeInvoice',
      amount: gross(i),
      date: i.invoice_date,
      currency: i.currency,
    })),
    ...account.quotes.map((q) => ({
      kind: 'quote' as const,
      id: q.id,
      number: q.quote_number,
      typeLabelKey: 'finanzen.customerAccount.typeQuote',
      amount: gross(q),
      date: q.created_at,
      currency: q.currency,
    })),
    ...account.creditNotes.map((c) => ({
      kind: 'creditNote' as const,
      id: c.id,
      number: c.credit_note_number,
      typeLabelKey: 'finanzen.customerAccount.typeCreditNote',
      amount: gross(c),
      date: c.created_at,
      currency: c.currency,
    })),
  ]
    .filter((d) => d.id !== currentDocId)
    .sort((a, b) => (a.date < b.date ? 1 : -1))

  const visibleDocs = docs.slice(0, 6)
  const overflow = docs.length - visibleDocs.length

  return (
    <section>
      <h4 className="mb-2 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
        <Users className="h-3 w-3" />
        {t('finanzen.customerAccount.title')}
      </h4>

      <div className="overflow-hidden rounded-lg border border-border">
        {/* Kunden-Kopf + CRM-Sprung */}
        <div className="flex items-start justify-between gap-3 border-b border-border-muted bg-secondary/30 px-3 py-2.5">
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-foreground">{customer.name}</p>
            {customer.ust_id_nr && (
              <p className="truncate text-[10px] text-muted-foreground">
                {t('finanzen.customerAccount.ustId')}: {customer.ust_id_nr}
              </p>
            )}
          </div>
          <button
            type="button"
            disabled={!contactId}
            onClick={() => contactId && navigate(`/kontakte?contact=${contactId}`)}
            title={contactId ? undefined : t('finanzen.customerAccount.noCrmMatch')}
            className="inline-flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] font-medium text-foreground transition-colors hover:bg-secondary disabled:cursor-not-allowed disabled:opacity-50"
          >
            <ArrowUpRight className="h-3 w-3" />
            {t('finanzen.customerAccount.openInCrm')}
          </button>
        </div>

        {/* Konto-Summen */}
        <div className="grid grid-cols-3 divide-x divide-border-muted border-b border-border-muted text-center">
          <div className="px-2 py-2">
            <p className="text-[9px] uppercase tracking-wide text-muted-foreground">
              {t('finanzen.customerAccount.revenue')}
            </p>
            <p className="text-xs font-semibold tabular-nums text-foreground">
              {formatMoney(account.totalRevenue)}
            </p>
          </div>
          <div className="px-2 py-2">
            <p className="text-[9px] uppercase tracking-wide text-muted-foreground">
              {t('finanzen.customerAccount.open')}
            </p>
            <p
              className={`text-xs font-semibold tabular-nums ${account.totalOpen > 0 ? 'text-warning' : 'text-foreground'}`}
            >
              {formatMoney(account.totalOpen)}
            </p>
          </div>
          <div className="px-2 py-2">
            <p className="text-[9px] uppercase tracking-wide text-muted-foreground">
              {t('finanzen.customerAccount.documents')}
            </p>
            <p className="text-xs font-semibold tabular-nums text-foreground">{account.docCount}</p>
          </div>
        </div>

        {/* Weitere Belege des Kunden */}
        {visibleDocs.length > 0 ? (
          <div>
            {visibleDocs.map((doc, idx) => {
              const Icon = doc.kind === 'creditNote' ? Receipt : FileText
              const clickable = !!nav
              return (
                <button
                  key={`${doc.kind}-${doc.id}`}
                  type="button"
                  disabled={!clickable}
                  onClick={() => nav?.open(doc.kind, doc.id)}
                  className={`flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-xs transition-colors ${
                    idx > 0 ? 'border-t border-border-muted' : ''
                  } ${clickable ? 'hover:bg-secondary/50' : 'cursor-default'}`}
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <Icon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                    <span className="min-w-0">
                      <span className="block truncate font-mono text-primary">{doc.number}</span>
                      <span className="block text-[10px] text-muted-foreground">
                        {t(doc.typeLabelKey)} · {formatDate(doc.date)}
                      </span>
                    </span>
                  </span>
                  <span className="flex shrink-0 items-center gap-1">
                    <span className="font-medium tabular-nums text-foreground">
                      {formatMoney(doc.amount, doc.currency)}
                    </span>
                    {clickable && <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />}
                  </span>
                </button>
              )
            })}
            {overflow > 0 && (
              <p className="border-t border-border-muted px-3 py-1.5 text-[10px] text-muted-foreground">
                {t('finanzen.customerAccount.more', { count: overflow })}
              </p>
            )}
          </div>
        ) : (
          <p className="px-3 py-2.5 text-[11px] text-muted-foreground">
            {t('finanzen.customerAccount.onlyThis')}
          </p>
        )}
      </div>
    </section>
  )
}
