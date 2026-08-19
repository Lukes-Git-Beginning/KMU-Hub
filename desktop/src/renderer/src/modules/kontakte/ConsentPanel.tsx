/**
 * ConsentPanel — DSGVO consent management per contact.
 *
 * Reads and writes the real consent records held by the CRM service. Every
 * purpose the backend knows is listed, including the ones nobody has decided
 * on yet, so an empty state reads as "no consent on file" rather than as a
 * missing feature. marketing_email and marketing_phone are the two the consent
 * asserter enforces before an email or a dialer call goes out.
 */
import { useMemo, useState } from 'react'
import {
  Shield,
  Mail,
  Phone,
  FileText,
  UserSearch,
  Database,
  Share2,
  Check,
  X,
  Clock,
  AlertTriangle,
  Info,
  ChevronDown,
  ChevronRight,
  Download,
  BadgeCheck,
  Loader2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  CONSENT_TYPES,
  useContactConsents,
  useConsentHistory,
  useGrantConsent,
  useRevokeConsent,
  type ConsentRecord,
  type ConsentType,
} from '@/api/hooks/useConsents'
import { useFormatDate } from '@/hooks/useFormatters'
import { downloadCsv } from '@/modules/finanzen/lib/finance-export'

const CONSENT_ICONS: Record<ConsentType, typeof Mail> = {
  marketing_email: Mail,
  marketing_phone: Phone,
  newsletter: FileText,
  profiling: UserSearch,
  data_processing: Database,
  data_sharing: Share2,
}

/**
 * Canonical source values. The backend stores the string verbatim, so keeping
 * these stable matters: the DOI badge and any later reporting key off them,
 * and a translated label would differ per user.
 */
const CONSENT_SOURCES = [
  'web_form',
  'contract',
  'email_confirmation',
  'phone',
  'in_person',
  'import',
] as const

/** A consent counts as active only while it is granted and not revoked. */
const isActive = (r: ConsentRecord | undefined): boolean => !!r?.granted && !r?.revoked_at

interface ConsentPanelProps {
  contactId: string
  contactName: string
}

export function ConsentPanel({ contactId, contactName }: ConsentPanelProps) {
  const { t } = useTranslation()
  const formatDate = useFormatDate()

  const { data: consents, isLoading, isError, refetch } = useContactConsents(contactId)
  const grant = useGrantConsent(contactId)
  const revoke = useRevokeConsent(contactId)

  const [grantingType, setGrantingType] = useState<string | null>(null)
  const [selectedSource, setSelectedSource] = useState<string>('web_form')
  const [expandedHistory, setExpandedHistory] = useState<Set<string>>(new Set())

  const busy = grant.isPending || revoke.isPending

  const { grantedCount, revokedCount } = useMemo(() => {
    let granted = 0
    let revoked = 0
    for (const type of CONSENT_TYPES) {
      const record = consents?.[type]
      if (isActive(record)) granted++
      else if (record?.revoked_at) revoked++
    }
    return { grantedCount: granted, revokedCount: revoked }
  }, [consents])

  const toggleHistory = (type: string) => {
    setExpandedHistory((prev) => {
      const next = new Set(prev)
      if (next.has(type)) next.delete(type)
      else next.add(type)
      return next
    })
  }

  const handleGrant = async (consentType: string) => {
    try {
      await grant.mutateAsync({ consentType, source: selectedSource })
      setGrantingType(null)
      toast.success(t('kontakte.consent.granted'))
    } catch {
      toast.error(t('kontakte.consent.actionFailed'))
    }
  }

  const handleRevoke = async (consentType: string) => {
    try {
      await revoke.mutateAsync({ consentType })
      toast.success(t('kontakte.consent.revoked'))
    } catch {
      toast.error(t('kontakte.consent.actionFailed'))
    }
  }

  /** Applies one action to every type that needs it and reports what stuck. */
  const applyToAll = async (mode: 'grant' | 'revoke') => {
    const targets = CONSENT_TYPES.filter((type) =>
      mode === 'grant' ? !isActive(consents?.[type]) : isActive(consents?.[type]),
    )
    if (targets.length === 0) return

    const results = await Promise.allSettled(
      targets.map((consentType) =>
        mode === 'grant'
          ? grant.mutateAsync({ consentType, source: selectedSource })
          : revoke.mutateAsync({ consentType }),
      ),
    )
    const failed = results.filter((r) => r.status === 'rejected').length
    if (failed > 0) {
      toast.error(t('kontakte.consent.partialFailure', { count: failed }))
      return
    }
    toast.success(
      t(mode === 'grant' ? 'kontakte.consent.allGranted' : 'kontakte.consent.allRevoked'),
    )
  }

  const handleExportCSV = () => {
    const rows = [
      ['Zweck', 'Status', 'Rechtsgrundlage', 'Quelle', 'Erteilt am', 'Widerrufen am'].join(';'),
      ...CONSENT_TYPES.map((type) => {
        const r = consents?.[type]
        const status = isActive(r)
          ? t('kontakte.consent.grantedLabel')
          : r?.revoked_at
            ? t('kontakte.consent.revokedLabel')
            : ''
        return [
          t(`kontakte.consent.type.${type}`),
          status,
          r?.legal_basis ?? '',
          r?.source ?? '',
          r?.granted_at ?? '',
          r?.revoked_at ?? '',
        ].join(';')
      }),
    ].join('\r\n')

    downloadCsv(rows + '\r\n', `einwilligungen-${contactId}.csv`)
    toast.success(t('kontakte.consent.exportedCSV'))
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 py-6 text-xs text-muted-foreground">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        {t('kontakte.consent.loading')}
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex items-center justify-between gap-2 rounded-md bg-error-light/40 px-3 py-2">
        <span className="flex items-center gap-2 text-[11px] text-error">
          <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
          {t('kontakte.consent.loadError')}
        </span>
        <button
          onClick={() => refetch()}
          className="rounded px-2 py-0.5 text-[10px] text-error hover:bg-error-light transition-colors"
        >
          {t('common.retry')}
        </button>
      </div>
    )
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
              onClick={() => applyToAll('grant')}
              disabled={busy || grantedCount === CONSENT_TYPES.length}
              className="rounded px-1.5 py-0.5 text-[10px] text-success hover:bg-success-light transition-colors disabled:opacity-40 disabled:hover:bg-transparent"
              title={t('kontakte.consent.grantAll')}
            >
              {t('kontakte.consent.grantAll')}
            </button>
            <button
              onClick={() => applyToAll('revoke')}
              disabled={busy || grantedCount === 0}
              className="rounded px-1.5 py-0.5 text-[10px] text-error hover:bg-error-light transition-colors disabled:opacity-40 disabled:hover:bg-transparent"
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
        {CONSENT_TYPES.map((type) => {
          const record = consents?.[type]
          const active = isActive(record)
          const Icon = CONSENT_ICONS[type]
          const isGranting = grantingType === type
          return (
            <div
              key={type}
              className={`rounded-lg border p-3 transition-colors ${
                active
                  ? 'border-success/30 bg-success-light/20'
                  : record?.revoked_at
                    ? 'border-warning/30 bg-warning-light/20'
                    : 'border-border'
              }`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Icon className={`h-4 w-4 ${active ? 'text-success' : 'text-muted-foreground'}`} />
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {t(`kontakte.consent.type.${type}`)}
                    </p>
                    <p className="text-[11px] text-muted-foreground">
                      {t(`kontakte.consent.typeDesc.${type}`)}
                    </p>
                  </div>
                </div>

                {/* Toggle */}
                {active ? (
                  <button
                    onClick={() => handleRevoke(type)}
                    disabled={busy}
                    className="flex items-center gap-1 rounded-md border border-error/30 px-2.5 py-1 text-[11px] font-medium text-error hover:bg-error-light transition-colors disabled:opacity-40"
                  >
                    <X className="h-3 w-3" />
                    {t('kontakte.consent.revoke')}
                  </button>
                ) : isGranting ? null : (
                  <button
                    onClick={() => setGrantingType(type)}
                    disabled={busy}
                    className="flex items-center gap-1 rounded-md border border-success/30 px-2.5 py-1 text-[11px] font-medium text-success hover:bg-success-light transition-colors disabled:opacity-40"
                  >
                    <Check className="h-3 w-3" />
                    {t('kontakte.consent.grant')}
                  </button>
                )}
              </div>

              {/* Metadata — flex-wrap so items stack instead of overflowing */}
              {(record?.granted_at || record?.revoked_at) && (
                <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 pl-6 text-[10px] text-muted-foreground">
                  {record?.granted_at && (
                    <span className="flex items-center gap-1 shrink-0">
                      <Clock className="h-2.5 w-2.5 shrink-0" />
                      <span className="truncate max-w-[140px]">
                        {t('kontakte.consent.grantedAt')}: {formatDate(record.granted_at)}
                      </span>
                    </span>
                  )}
                  {record?.source && (
                    <span className="truncate max-w-[140px]">
                      {t('kontakte.consent.source')}:{' '}
                      {t(`kontakte.consent.sourceLabel.${record.source}`, {
                        defaultValue: record.source,
                      })}
                    </span>
                  )}
                  {record?.source === 'email_confirmation' && (
                    <span className="flex items-center gap-1 shrink-0 rounded-full bg-primary-light px-1.5 py-0 text-[9px] font-medium text-primary">
                      <BadgeCheck className="h-2.5 w-2.5 shrink-0" />
                      DOI
                    </span>
                  )}
                  {record?.revoked_at && (
                    <span className="text-warning flex items-center gap-1 shrink-0">
                      <AlertTriangle className="h-2.5 w-2.5 shrink-0" />
                      <span className="truncate max-w-[140px]">
                        {t('kontakte.consent.revokedAt')}: {formatDate(record.revoked_at)}
                      </span>
                    </span>
                  )}
                  <button
                    onClick={() => toggleHistory(type)}
                    className="flex items-center gap-0.5 shrink-0 text-primary hover:underline"
                  >
                    {expandedHistory.has(type) ? (
                      <ChevronDown className="h-2.5 w-2.5" />
                    ) : (
                      <ChevronRight className="h-2.5 w-2.5" />
                    )}
                    {t('kontakte.consent.history')}
                  </button>
                </div>
              )}

              {/* History timeline — fetched only once expanded */}
              {expandedHistory.has(type) && (
                <ConsentHistory contactId={contactId} consentType={type} />
              )}

              {/* Grant form */}
              {isGranting && (
                <div className="mt-2 flex items-center gap-2 pl-6">
                  <select
                    value={selectedSource}
                    onChange={(e) => setSelectedSource(e.target.value)}
                    className="h-7 rounded-md border border-border bg-transparent px-2 text-xs outline-none focus:border-primary"
                  >
                    {CONSENT_SOURCES.map((s) => (
                      <option key={s} value={s}>
                        {t(`kontakte.consent.sourceLabel.${s}`)}
                      </option>
                    ))}
                  </select>
                  <button
                    onClick={() => handleGrant(type)}
                    disabled={busy}
                    className="h-7 rounded-md bg-success px-3 text-xs font-medium text-white hover:bg-success/90 transition-colors disabled:opacity-40"
                  >
                    {t('common.confirm')}
                  </button>
                  <button
                    onClick={() => setGrantingType(null)}
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

/** Audit trail for one consent type. Split out so the query runs only when opened. */
function ConsentHistory({ contactId, consentType }: { contactId: string; consentType: string }) {
  const { t } = useTranslation()
  const formatDate = useFormatDate()
  const { data: history, isLoading } = useConsentHistory(contactId, consentType, true)

  if (isLoading) {
    return (
      <div className="mt-2 ml-6 flex items-center gap-1.5 pl-3 text-[10px] text-muted-foreground">
        <Loader2 className="h-2.5 w-2.5 animate-spin" />
        {t('kontakte.consent.loading')}
      </div>
    )
  }

  if (!history || history.length === 0) {
    return (
      <div className="mt-2 ml-6 border-l-2 border-border pl-3 text-[10px] text-muted-foreground">
        {t('kontakte.consent.noHistory')}
      </div>
    )
  }

  return (
    <div className="mt-2 ml-6 border-l-2 border-border pl-3 space-y-1.5">
      {history.map((entry, i) => {
        const revoked = !!entry.revoked_at
        const stamp = entry.revoked_at ?? entry.granted_at ?? entry.created_at
        return (
          <div key={entry.id ?? i} className="flex items-center gap-2 text-[10px]">
            <span
              className={`h-1.5 w-1.5 rounded-full shrink-0 ${revoked ? 'bg-warning' : 'bg-success'}`}
            />
            <span className="text-muted-foreground">{stamp ? formatDate(stamp) : ''}</span>
            <span className={revoked ? 'text-warning' : 'text-success'}>
              {revoked ? t('kontakte.consent.revokedAt') : t('kontakte.consent.grantedAt')}
            </span>
            {entry.source && (
              <span className="text-muted-foreground truncate">
                {t(`kontakte.consent.sourceLabel.${entry.source}`, { defaultValue: entry.source })}
              </span>
            )}
          </div>
        )
      })}
    </div>
  )
}
