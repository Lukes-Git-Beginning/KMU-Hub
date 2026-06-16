/**
 * Kundenkonto-Aggregation für die Beleg-Detail-Modals (finanzen — Modal-Tiefe).
 *
 * Finanz-Belege hängen an einem denormalisierten `CustomerSnapshot` (Name/E-Mail),
 * es gibt keinen harten `contact_id`-Link. Diese Helfer matchen Belege desselben
 * Kunden über Name+E-Mail und finden — heuristisch — den passenden CRM-Kontakt,
 * damit die Modals Richtung Kontakte-360° verlinken können.
 */
import { useMemo } from 'react'
import { useInvoices, useQuotes, useCreditNotes } from '@/api/hooks/useFinance'
import { useContacts } from '@/api/hooks/useContacts'
import type { CustomerSnapshot, Invoice, Quote, CreditNote } from '@/types/finance-types'

function norm(s?: string): string {
  return (s ?? '').trim().toLowerCase()
}

/** Bruttobetrag eines Belegs (tax_breakdown bevorzugt, sonst denormalisiert). */
function grossOf(doc: { tax_breakdown?: { gross_total?: string }; total_gross?: number | string }): number {
  return Number(doc.tax_breakdown?.gross_total ?? doc.total_gross ?? 0)
}

/** True, wenn Snapshot und ein anderer Beleg-Kunde dieselbe Person/Firma sind. */
function sameCustomer(a: CustomerSnapshot | undefined, b: CustomerSnapshot | undefined): boolean {
  if (!a || !b) return false
  const ea = norm(a.email)
  const eb = norm(b.email)
  if (ea && eb) return ea === eb
  const na = norm(a.name)
  return na !== '' && na === norm(b.name)
}

export interface CustomerAccount {
  invoices: Invoice[]
  quotes: Quote[]
  creditNotes: CreditNote[]
  /** Gesamtzahl verknüpfter Belege (Rechnungen + Angebote + Gutschriften). */
  docCount: number
  /** Summe Brutto aller nicht stornierten Rechnungen. */
  totalRevenue: number
  /** Summe Brutto offener Rechnungen (versendet/überfällig) — Näherung. */
  totalOpen: number
  isLoading: boolean
}

/**
 * Alle Belege eines Kunden (über alle Listen hinweg) + Konto-Summen.
 * React-Query dedupliziert die Listen-Queries, daher ist der Mehraufwand gering.
 */
export function useCustomerAccount(customer?: CustomerSnapshot): CustomerAccount {
  const { data: invData, isLoading: invLoading } = useInvoices()
  const { data: quoteData, isLoading: quoteLoading } = useQuotes()
  const { data: cnData, isLoading: cnLoading } = useCreditNotes()

  return useMemo<CustomerAccount>(() => {
    const invoices = (invData?.invoices ?? []).filter((i) => sameCustomer(customer, i.customer))
    const quotes = (quoteData?.quotes ?? []).filter((q) => sameCustomer(customer, q.customer))
    const creditNotes = (cnData?.credit_notes ?? []).filter((c) => sameCustomer(customer, c.customer))

    const totalRevenue = invoices
      .filter((i) => i.status !== 'cancelled')
      .reduce((sum, i) => sum + grossOf(i), 0)
    const totalOpen = invoices
      .filter((i) => i.status === 'sent' || i.status === 'overdue')
      .reduce((sum, i) => sum + grossOf(i), 0)

    return {
      invoices,
      quotes,
      creditNotes,
      docCount: invoices.length + quotes.length + creditNotes.length,
      totalRevenue,
      totalOpen,
      isLoading: invLoading || quoteLoading || cnLoading,
    }
  }, [customer, invData, quoteData, cnData, invLoading, quoteLoading, cnLoading])
}

/**
 * Heuristischer CRM-Treffer für einen Beleg-Kunden. Gibt die Kontakt-ID zurück,
 * mit der `/kontakte?contact=<id>` die 360°-Ansicht öffnet — oder null, wenn
 * kein eindeutiger Treffer existiert. E-Mail-Gleichheit schlägt Namens-Match.
 */
export function useContactMatch(customer?: CustomerSnapshot): string | null {
  const { data } = useContacts(customer?.name ? { search: customer.name, page_size: 10 } : undefined)

  return useMemo(() => {
    const list = ((data as { contacts?: Record<string, unknown>[] } | undefined)?.contacts ?? [])
    if (!list.length) return null
    const email = norm(customer?.email)
    const name = norm(customer?.name)

    if (email) {
      const byEmail = list.find((c) => norm(c.email as string) === email)
      if (byEmail) return String(byEmail.id)
    }
    if (name) {
      const byName = list.find((c) => {
        const full = norm(`${(c.firstName as string) ?? ''} ${(c.lastName as string) ?? ''}`)
        const company = norm(
          (c.company as string) ?? (c.companyName as string) ?? (c.company_name as string),
        )
        return (
          (!!full && full === name) ||
          (!!company && (company === name || name.includes(company) || company.includes(name)))
        )
      })
      if (byName) return String(byName.id)
    }
    return null
  }, [data, customer?.email, customer?.name])
}
