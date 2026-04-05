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
  ChevronDown,
  ChevronRight,
  Download,
  BadgeCheck,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Consent types
// ---------------------------------------------------------------------------

interface ConsentHistoryEntry {
  action: 'granted' | 'revoked'
  date: string
  source: string
}

interface ConsentEntry {
  id: string
  purpose: string
  icon: typeof Mail
  description: string
  granted: boolean
  grantedAt: string | null
  source: string | null
  revokedAt: string | null
  history: ConsentHistoryEntry[]
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
      source: isVip ? 'E-Mail-Bestätigung' : null,
      revokedAt: null,
      history: isVip
        ? [
            { action: 'granted', date: '2025-06-15', source: 'E-Mail-Bestätigung' },
            { action: 'revoked', date: '2025-04-10', source: 'Webformular' },
            { action: 'granted', date: '2025-03-01', source: 'Webformular' },
          ]
        : [],
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
      history: isVip
        ? [{ action: 'granted', date: '2025-06-15', source: 'Vertrag' }]
        : [],
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
      history: [],
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
      history: contactId === 'c9'
        ? [
            { action: 'revoked', date: '2025-11-15', source: 'E-Mail-Bestätigung' },
            { action: 'granted', date: '2025-08-01', source: 'Webformular' },
          ]
        : [],
    },
  ]
}

// ---------------------------------------------------------------------------
// Sources for granting consent
// ---------------------------------------------------------------------------

const consentSources = [
  'Webformular',
  'Vertrag',
  'E-Mail-Bestätigung',
  'Muendlich (Telefon)',
  'Muendlich (persönlich)',
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
  const { t } = useTranslation()
  const [consents, setConsents] = useState<ConsentEntry[]>(() => createMockConsents(contactId))
  const [grantingId, setGrantingId] = useState<string | null>(null)
  const [selectedSource, setSelectedSource] = useState('Webformular')
  const [expandedHistory, setExpandedHistory] = useState<Set<string>>(new Set())

  const grantedCount = consents.filter((c) => c.granted).length
  const revokedCount = consents.filter((c) => c.revokedAt && !c.granted).length

  const toggleHistory = (id: string) => {
    setExpandedHistory((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleGrant = (id: string) => {
    const today = new Date().toISOString().split('T')[0]
    setConsents((prev) =>
      prev.map((c) =>
        c.id === id
          ? {
              ...c,
              granted: true,
              grantedAt: today,
              source: selectedSource,
              revokedAt: null,
              history: [{ action: 'granted' as const, date: today, source: selectedSource }, ...c.history],
            }
          : c,
      ),
    )
    setGrantingId(null)
    toast.success(t('kontakte.consent.granted'))
  }

  const handleRevoke = (id: string) => {
    const today = new Date().toISOString().split('T')[0]
    setConsents((prev) =>
      prev.map((c) =>
        c.id === id
          ? {
              ...c,
              granted: false,
              revokedAt: today,
              history: [{ action: 'revoked' as const, date: today, source: 'Manuell' }, ...c.history],
            }
          : c,
      ),
    )
    toast.success(t('kontakte.consent.revoked'))
  }

  const handleGrantAll = () => {
    const today = new Date().toISOString().split('T')[0]
    setConsents((prev) =>
      prev.map((c) =>
        c.granted
          ? c
          : {
              ...c,
              granted: true,
              grantedAt: today,
              source: 'Webformular',
              revokedAt: null,
              history: [{ action: 'granted' as const, date: today, source: 'Webformular (Batch)' }, ...c.history],
            },
      ),
    )
    toast.success(t('kontakte.consent.allGranted'))
  }

  const handleRevokeAll = () => {
    const today = new Date().toISOString().split('T')[0]
    setConsents((prev) =>
      prev.map((c) =>
        !c.granted
          ? c
          : {
              ...c,
              granted: false,
              revokedAt: today,
              history: [{ action: 'revoked' as const, date: today, source: 'Manuell (Batch)' }, ...c.history],
            },
      ),
    )
    toast.success(t('kontakte.consent.allRevoked'))
  }

  const handleExportCSV = () => {
    toast.success(t('kontakte.consent.exportedCSV'))
  }

  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Shield className="h-4 w-4 text-primary" />
          <span className="text-sm font-medium text-foreground">{t('kontakte.consent.title')}</span>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 text-[10px] text-muted-foreground">
            <span className="flex items-center gap-1">
              <Check className="h-3 w-3 text-success" />
              {grantedCount} {t('kontakte.consent.grantedLabel')}
            </span>
            {revokedCount > 0 && (
              <span className="flex items-center gap-1">
                <AlertTriangle className="h-3 w-3 text-warning" />
                {revokedCount} {t('kontakte.consent.revokedLabel')}
              </span>
            )}
          </div>
          <div className="flex items-center gap-1 border-l border-border pl-3">
            <button
              onClick={handleGrantAll}
              className="rounded px-1.5 py-0.5 text-[10px] text-success hover:bg-success-light transition-colors"
              title={t('kontakte.consent.grantAll')}
            >
              {t('kontakte.consent.grantAll')}
            </button>
            <button
              onClick={handleRevokeAll}
              className="rounded px-1.5 py-0.5 text-[10px] text-error hover:bg-error-light transition-colors"
              title={t('kontakte.consent.revokeAll')}
            >
              {t('kontakte.consent.revokeAll')}
            </button>
            <button
              onClick={handleExportCSV}
              className="rounded p-0.5 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
              title={t('kontakte.consent.exportCSVTitle')}
            >
              <Download className="h-3 w-3" />
            </button>
          </div>
        </div>
      </div>

      {/* Info banner */}
      <div className="flex items-start gap-2 rounded-md bg-info-light/50 px-3 py-2">
        <Info className="h-3.5 w-3.5 text-info mt-0.5 shrink-0" />
        <p className="text-[11px] text-info">
          {t('kontakte.consent.infoBanner', { name: contactName })}
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
                    {t('kontakte.consent.revoke')}
                  </button>
                ) : (
                  <button
                    onClick={() => setGrantingId(isGranting ? null : consent.id)}
                    className="flex items-center gap-1 rounded-md border border-success/30 px-2.5 py-1 text-[11px] font-medium text-success hover:bg-success-light transition-colors"
                  >
                    <Check className="h-3 w-3" />
                    {t('kontakte.consent.grant')}
                  </button>
                )}
              </div>

              {/* Metadata */}
              {(consent.grantedAt || consent.revokedAt) && (
                <div className="mt-2 flex items-center gap-3 pl-6 text-[10px] text-muted-foreground">
                  {consent.grantedAt && (
                    <span className="flex items-center gap-1">
                      <Clock className="h-2.5 w-2.5" />
                      {t('kontakte.consent.grantedAt')}: {consent.grantedAt}
                    </span>
                  )}
                  {consent.source && (
                    <span>{t('kontakte.consent.source')}: {consent.source}</span>
                  )}
                  {consent.source === 'E-Mail-Bestätigung' && (
                    <span className="flex items-center gap-1 rounded-full bg-primary-light px-1.5 py-0 text-[9px] font-medium text-primary">
                      <BadgeCheck className="h-2.5 w-2.5" />
                      DOI
                    </span>
                  )}
                  {consent.revokedAt && (
                    <span className="text-warning flex items-center gap-1">
                      <AlertTriangle className="h-2.5 w-2.5" />
                      {t('kontakte.consent.revokedAt')}: {consent.revokedAt}
                    </span>
                  )}
                  {consent.history.length > 0 && (
                    <button
                      onClick={() => toggleHistory(consent.id)}
                      className="flex items-center gap-0.5 text-primary hover:underline"
                    >
                      {expandedHistory.has(consent.id) ? (
                        <ChevronDown className="h-2.5 w-2.5" />
                      ) : (
                        <ChevronRight className="h-2.5 w-2.5" />
                      )}
                      {t('kontakte.consent.history')} ({consent.history.length})
                    </button>
                  )}
                </div>
              )}

              {/* History timeline */}
              {expandedHistory.has(consent.id) && consent.history.length > 0 && (
                <div className="mt-2 ml-6 border-l-2 border-border pl-3 space-y-1.5">
                  {consent.history.map((h, i) => (
                    <div key={i} className="flex items-center gap-2 text-[10px]">
                      <span className={`h-1.5 w-1.5 rounded-full shrink-0 ${h.action === 'granted' ? 'bg-success' : 'bg-warning'}`} />
                      <span className="text-muted-foreground">{h.date}</span>
                      <span className={h.action === 'granted' ? 'text-success' : 'text-warning'}>
                        {h.action === 'granted' ? t('kontakte.consent.grantedAt') : t('kontakte.consent.revokedAt')}
                      </span>
                      <span className="text-muted-foreground">via {h.source}</span>
                    </div>
                  ))}
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
                    {t('common.confirm')}
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
