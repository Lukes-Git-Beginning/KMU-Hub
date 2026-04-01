/**
 * FirmaDetailPanel — Company detail view in the Kontakte module.
 *
 * Shows company header, linked employees, deals, activity timeline,
 * and company-level custom fields. Slide-in panel from the right.
 */
import { useMemo } from 'react'
import {
  Building2,
  Globe,
  MapPin,
  Users,
  Phone,
  Mail,
  TrendingUp,
  ExternalLink,
  X,
  Briefcase,
  Calendar,
} from 'lucide-react'
import { useContactsStore } from '@/stores/contacts'

// ---------------------------------------------------------------------------
// Mock deal data for companies
// ---------------------------------------------------------------------------

interface CompanyDeal {
  id: string
  name: string
  value: number
  currency: string
  stage: string
  stageColor: string
  probability: number
}

const mockCompanyDeals: Record<string, CompanyDeal[]> = {
  'Cosmi AG': [
    { id: 'cd1', name: 'Plattform-Lizenz 2026', value: 120000, currency: 'CHF', stage: 'Gewonnen', stageColor: '#22c55e', probability: 100 },
    { id: 'cd2', name: 'Schulungspaket', value: 15000, currency: 'CHF', stage: 'Angebot', stageColor: '#a78bfa', probability: 50 },
  ],
  'ABC GmbH': [
    { id: 'cd3', name: 'CRM-Integration', value: 45000, currency: 'CHF', stage: 'Verhandlung', stageColor: '#f59e0b', probability: 75 },
    { id: 'cd4', name: 'Support-Vertrag', value: 8000, currency: 'CHF', stage: 'Gewonnen', stageColor: '#22c55e', probability: 100 },
  ],
  'Design Studio GmbH': [
    { id: 'cd5', name: 'Website Relaunch', value: 32000, currency: 'CHF', stage: 'Qualifiziert', stageColor: '#60a5fa', probability: 25 },
  ],
  'XYZ AG': [
    { id: 'cd6', name: 'CRM-Evaluierung', value: 60000, currency: 'CHF', stage: 'Lead', stageColor: '#94a3b8', probability: 10 },
  ],
  'Steiner Bauunternehmen AG': [
    { id: 'cd7', name: 'Projektmanagement-Modul', value: 85000, currency: 'CHF', stage: 'Angebot', stageColor: '#a78bfa', probability: 50 },
  ],
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface FirmaDetailPanelProps {
  companyName: string
  onClose: () => void
}

export function FirmaDetailPanel({ companyName, onClose }: FirmaDetailPanelProps) {
  const contacts = useContactsStore((s) => s.contacts)

  // Find employees at this company
  const employees = useMemo(
    () => contacts.filter((c) => c.company === companyName),
    [contacts, companyName],
  )

  const deals = mockCompanyDeals[companyName] ?? []
  const totalDealValue = deals.reduce((sum, d) => sum + d.value, 0)
  const weightedValue = deals.reduce((sum, d) => sum + (d.value * d.probability) / 100, 0)

  // Aggregate company info from contacts
  const companyInfo = useMemo(() => {
    const firstContact = employees[0]
    if (!firstContact) return null
    return {
      city: firstContact.address.city,
      country: firstContact.address.country,
      website: firstContact.website,
      industry: (firstContact.customFields as Record<string, unknown>)?.cf3 as string | undefined,
    }
  }, [employees])

  return (
    <div className="fixed inset-y-0 right-0 z-40 flex w-[420px] flex-col border-l border-border bg-card shadow-xl">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <Building2 className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h2 className="text-sm font-semibold text-foreground">{companyName}</h2>
            <p className="text-[11px] text-muted-foreground">
              {employees.length} Kontakt{employees.length !== 1 ? 'e' : ''} · {deals.length} Deal{deals.length !== 1 ? 's' : ''}
            </p>
          </div>
        </div>
        <button
          onClick={onClose}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {/* Company info */}
        {companyInfo && (
          <div className="border-b border-border px-4 py-3 space-y-2">
            <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Unternehmensdaten</h3>
            {companyInfo.city && (
              <div className="flex items-center gap-2 text-sm text-foreground">
                <MapPin className="h-3.5 w-3.5 text-muted-foreground" />
                {companyInfo.city}, {companyInfo.country}
              </div>
            )}
            {companyInfo.website && (
              <div className="flex items-center gap-2 text-sm">
                <Globe className="h-3.5 w-3.5 text-muted-foreground" />
                <a
                  href={companyInfo.website.startsWith('http') ? companyInfo.website : `https://${companyInfo.website}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary hover:underline flex items-center gap-1"
                >
                  {companyInfo.website}
                  <ExternalLink className="h-3 w-3" />
                </a>
              </div>
            )}
            {companyInfo.industry && (
              <div className="flex items-center gap-2 text-sm text-foreground">
                <Briefcase className="h-3.5 w-3.5 text-muted-foreground" />
                {companyInfo.industry}
              </div>
            )}
          </div>
        )}

        {/* Employees */}
        <div className="border-b border-border px-4 py-3">
          <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
            <Users className="inline h-3 w-3 mr-1" />
            Mitarbeiter ({employees.length})
          </h3>
          {employees.length === 0 ? (
            <p className="text-xs text-muted-foreground">Keine Kontakte zugeordnet.</p>
          ) : (
            <div className="space-y-1.5">
              {employees.map((emp) => (
                <div
                  key={emp.id}
                  className="flex items-center gap-2.5 rounded-md p-2 hover:bg-accent/50 transition-colors cursor-pointer"
                >
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-secondary text-xs font-medium text-foreground">
                    {emp.initials}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-foreground truncate">
                      {emp.title ? `${emp.title} ` : ''}{emp.salutation ? `${emp.salutation} ` : ''}{emp.firstName} {emp.lastName}
                    </p>
                    <p className="text-[11px] text-muted-foreground truncate">{emp.jobTitle}</p>
                  </div>
                  <div className="flex items-center gap-1">
                    {emp.email && (
                      <button className="rounded-md p-1 text-muted-foreground hover:text-primary transition-colors" title={emp.email} aria-label={`E-Mail an ${emp.email}`}>
                        <Mail className="h-3 w-3" />
                      </button>
                    )}
                    {emp.phone && (
                      <button className="rounded-md p-1 text-muted-foreground hover:text-primary transition-colors" title={emp.phone} aria-label={`Anrufen: ${emp.phone}`}>
                        <Phone className="h-3 w-3" />
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Deals */}
        <div className="border-b border-border px-4 py-3">
          <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
            <TrendingUp className="inline h-3 w-3 mr-1" />
            Deals ({deals.length})
          </h3>
          {deals.length === 0 ? (
            <p className="text-xs text-muted-foreground">Keine Deals verknüpft.</p>
          ) : (
            <>
              <div className="space-y-1.5">
                {deals.map((deal) => (
                  <div
                    key={deal.id}
                    className="flex items-center gap-2 rounded-md border border-border p-2"
                  >
                    <div
                      className="h-2 w-2 rounded-full shrink-0"
                      style={{ backgroundColor: deal.stageColor }}
                    />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-foreground truncate">{deal.name}</p>
                      <p className="text-[10px] text-muted-foreground">{deal.stage} · {deal.probability}%</p>
                    </div>
                    <span className="text-xs font-medium text-foreground">
                      {new Intl.NumberFormat('de-DE', { style: 'currency', currency: deal.currency, maximumFractionDigits: 0 }).format(deal.value)}
                    </span>
                  </div>
                ))}
              </div>

              {/* Deal summary */}
              <div className="mt-2 rounded-md bg-secondary/50 px-3 py-2">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-muted-foreground">Gesamtwert</span>
                  <span className="font-medium text-foreground">
                    {new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'CHF', maximumFractionDigits: 0 }).format(totalDealValue)}
                  </span>
                </div>
                <div className="flex items-center justify-between text-xs mt-0.5">
                  <span className="text-muted-foreground">Gewichtet</span>
                  <span className="font-medium text-primary">
                    {new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'CHF', maximumFractionDigits: 0 }).format(weightedValue)}
                  </span>
                </div>
              </div>
            </>
          )}
        </div>

        {/* Recent activity */}
        <div className="px-4 py-3">
          <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
            <Calendar className="inline h-3 w-3 mr-1" />
            Letzte Aktivitaeten
          </h3>
          <div className="space-y-2">
            {employees.slice(0, 3).flatMap((emp) =>
              emp.activities.slice(0, 2).map((act) => (
                <div key={act.id} className="flex items-start gap-2">
                  <div className="mt-1 h-1.5 w-1.5 rounded-full bg-muted-foreground/40 shrink-0" />
                  <div className="flex-1">
                    <p className="text-xs text-foreground">{act.description}</p>
                    <p className="text-[10px] text-muted-foreground">{emp.firstName} {emp.lastName} · {act.date}</p>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
