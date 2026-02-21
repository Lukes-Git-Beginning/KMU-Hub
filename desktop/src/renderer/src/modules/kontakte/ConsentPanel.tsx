/**
 * ConsentPanel — DSGVO consent management per contact.
 *
 * Shows consent flags per purpose with timestamp, source, and revocation.
 * Embedded in ContactDetailPanel or as a standalone section.
 */
import { useState } from 'react'
import {
  Shield,
  Mail,
  Phone,
  FileText,
  UserSearch,
  Check,
  X,
  Clock,
  AlertTriangle,
  Info,
} from 'lucide-react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Consent types
// ---------------------------------------------------------------------------

interface ConsentEntry {
  id: string
  purpose: string
  icon: typeof Mail
  description: string
  granted: boolean
  grantedAt: string | null
  source: string | null
  revokedAt: string | null
}

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

const createMockConsents = (contactId: string): ConsentEntry[] => {
  const isVip = contactId === 'c4' || contactId === 'c12'
  return [
    {
      id: 'consent-email',
      purpose: 'E-Mail-Marketing',
      icon: Mail,
      description: 'Newsletter und Marketing-E-Mails senden',
      granted: isVip,
      grantedAt: isVip ? '2025-06-15' : null,
      source: isVip ? 'Webformular' : null,
      revokedAt: null,
    },
    {
      id: 'consent-phone',
      purpose: 'Telefonische Kontaktaufnahme',
      icon: Phone,
      description: 'Anrufe zu Verkaufs- und Informationszwecken',
      granted: isVip,
      grantedAt: isVip ? '2025-06-15' : null,
      source: isVip ? 'Vertrag' : null,
      revokedAt: null,
    },
    {
      id: 'consent-post',
      purpose: 'Postalische Werbung',
      icon: FileText,
      description: 'Briefe und Prospekte an die Postadresse',
      granted: false,
      grantedAt: null,
      source: null,
      revokedAt: null,
    },
    {
      id: 'consent-profiling',
      purpose: 'Profiling & Analyse',
      icon: UserSearch,
      description: 'Verhaltensanalyse zur Personalisierung',
      granted: false,
      grantedAt: null,
      source: null,
      revokedAt: contactId === 'c9' ? '2025-11-15' : null,
    },
  ]
}

// ---------------------------------------------------------------------------
// Sources for granting consent
// ---------------------------------------------------------------------------

const consentSources = [
  'Webformular',
  'Vertrag',
  'E-Mail-Bestaetigung',
  'Muendlich (Telefon)',
  'Muendlich (persoenlich)',
  'Import',
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ConsentPanelProps {
  contactId: string
  contactName: string
}

export function ConsentPanel({ contactId, contactName }: ConsentPanelProps) {
  const [consents, setConsents] = useState<ConsentEntry[]>(() => createMockConsents(contactId))
  const [grantingId, setGrantingId] = useState<string | null>(null)
  const [selectedSource, setSelectedSource] = useState('Webformular')

  const grantedCount = consents.filter((c) => c.granted).length
  const revokedCount = consents.filter((c) => c.revokedAt && !c.granted).length

  const handleGrant = (id: string) => {
    setConsents((prev) =>
      prev.map((c) =>
        c.id === id
          ? {
              ...c,
              granted: true,
              grantedAt: new Date().toISOString().split('T')[0],
              source: selectedSource,
              revokedAt: null,
            }
          : c,
      ),
    )
    setGrantingId(null)
    toast.success('Einwilligung erteilt')
  }

  const handleRevoke = (id: string) => {
    setConsents((prev) =>
      prev.map((c) =>
        c.id === id
          ? {
              ...c,
              granted: false,
              revokedAt: new Date().toISOString().split('T')[0],
            }
          : c,
      ),
    )
    toast.success('Einwilligung widerrufen')
  }

  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Shield className="h-4 w-4 text-primary" />
          <span className="text-sm font-medium text-foreground">DSGVO-Einwilligungen</span>
        </div>
        <div className="flex items-center gap-2 text-[10px] text-muted-foreground">
          <span className="flex items-center gap-1">
            <Check className="h-3 w-3 text-success" />
            {grantedCount} erteilt
          </span>
          {revokedCount > 0 && (
            <span className="flex items-center gap-1">
              <AlertTriangle className="h-3 w-3 text-warning" />
              {revokedCount} widerrufen
            </span>
          )}
        </div>
      </div>

      {/* Info banner */}
      <div className="flex items-start gap-2 rounded-md bg-info-light/50 px-3 py-2">
        <Info className="h-3.5 w-3.5 text-info mt-0.5 shrink-0" />
        <p className="text-[11px] text-info">
          Einwilligungen fuer {contactName}. Jede Aenderung wird mit Zeitstempel und Quelle protokolliert.
        </p>
      </div>

      {/* Consent entries */}
      <div className="space-y-2">
        {consents.map((consent) => {
          const Icon = consent.icon
          const isGranting = grantingId === consent.id
          return (
            <div
              key={consent.id}
              className={`rounded-lg border p-3 transition-colors ${
                consent.granted
                  ? 'border-success/30 bg-success-light/20'
                  : consent.revokedAt
                    ? 'border-warning/30 bg-warning-light/20'
                    : 'border-border'
              }`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Icon className={`h-4 w-4 ${consent.granted ? 'text-success' : 'text-muted-foreground'}`} />
                  <div>
                    <p className="text-sm font-medium text-foreground">{consent.purpose}</p>
                    <p className="text-[11px] text-muted-foreground">{consent.description}</p>
                  </div>
                </div>

                {/* Toggle */}
                {consent.granted ? (
                  <button
                    onClick={() => handleRevoke(consent.id)}
                    className="flex items-center gap-1 rounded-md border border-error/30 px-2.5 py-1 text-[11px] font-medium text-error hover:bg-error-light transition-colors"
                  >
                    <X className="h-3 w-3" />
                    Widerrufen
                  </button>
                ) : (
                  <button
                    onClick={() => setGrantingId(isGranting ? null : consent.id)}
                    className="flex items-center gap-1 rounded-md border border-success/30 px-2.5 py-1 text-[11px] font-medium text-success hover:bg-success-light transition-colors"
                  >
                    <Check className="h-3 w-3" />
                    Erteilen
                  </button>
                )}
              </div>

              {/* Metadata */}
              {(consent.grantedAt || consent.revokedAt) && (
                <div className="mt-2 flex items-center gap-3 pl-6 text-[10px] text-muted-foreground">
                  {consent.grantedAt && (
                    <span className="flex items-center gap-1">
                      <Clock className="h-2.5 w-2.5" />
                      Erteilt: {consent.grantedAt}
                    </span>
                  )}
                  {consent.source && (
                    <span>Quelle: {consent.source}</span>
                  )}
                  {consent.revokedAt && (
                    <span className="text-warning flex items-center gap-1">
                      <AlertTriangle className="h-2.5 w-2.5" />
                      Widerrufen: {consent.revokedAt}
                    </span>
                  )}
                </div>
              )}

              {/* Grant form */}
              {isGranting && (
                <div className="mt-2 flex items-center gap-2 pl-6">
                  <select
                    value={selectedSource}
                    onChange={(e) => setSelectedSource(e.target.value)}
                    className="h-7 rounded-md border border-border bg-transparent px-2 text-xs outline-none focus:border-primary"
                  >
                    {consentSources.map((s) => (
                      <option key={s} value={s}>{s}</option>
                    ))}
                  </select>
                  <button
                    onClick={() => handleGrant(consent.id)}
                    className="h-7 rounded-md bg-success px-3 text-xs font-medium text-white hover:bg-success/90 transition-colors"
                  >
                    Bestaetigen
                  </button>
                  <button
                    onClick={() => setGrantingId(null)}
                    className="h-7 rounded-md border border-border px-2 text-xs text-muted-foreground hover:bg-accent transition-colors"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
