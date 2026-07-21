/**
 * AuditEventDetailModal — Entra-Muster: Detail-Ansicht eines Audit-Eintrags.
 *
 * Zeigt alle Felder des Events sowie — wenn details.old_value / new_value
 * vorhanden — ein dezentes Alt/Neu-Delta-Panel (zwei Spalten, generisches
 * Rendering für primitive Werte, Arrays, Objekte und Grant-Maps).
 *
 * Wiederverwendet von AuditLogPage (volle Liste) und RolesAuditTab
 * (Rollen-gefilterte Sicht im Rollen-Bereich).
 */
import { useTranslation } from 'react-i18next'
import { useFormatDate } from '@/hooks/useFormatters'
import { DetailModal } from '@/components/shared'
import type { AuditEntry } from '@/api/security-types'

// ---------------------------------------------------------------------------
// Action-Label-Map für RBAC Live-Actions
// ---------------------------------------------------------------------------

const LIVE_ACTION_KEY_MAP: Record<string, string> = {
  // Seed-era action name (000039 naming) — surfaces in the roles protocol tab
  // next to translated live events, so it gets a label too.
  user_role_changed: 'rbac.audit.action.user_role_changed',
  'role.assigned': 'rbac.audit.action.role_assigned',
  'role.revoked': 'rbac.audit.action.role_revoked',
  'role.definition_created': 'rbac.audit.action.role_definition_created',
  'role.definition_updated': 'rbac.audit.action.role_definition_updated',
  'role.definition_deleted': 'rbac.audit.action.role_definition_deleted',
  'user.invited': 'rbac.audit.action.user_invited',
  'user.deactivated': 'rbac.audit.action.user_deactivated',
  'user.reactivated': 'rbac.audit.action.user_reactivated',
  'user.offboarded': 'rbac.audit.action.user_offboarded',
  'user.view_as': 'rbac.audit.action.user_view_as',
  'vendor_access.requested': 'rbac.audit.action.vendor_access_requested',
  'vendor_access.approved': 'rbac.audit.action.vendor_access_approved',
  'vendor_access.declined': 'rbac.audit.action.vendor_access_declined',
  'vendor_access.counter_proposed': 'rbac.audit.action.vendor_access_counter_proposed',
  'vendor_access.granted': 'rbac.audit.action.vendor_access_granted',
  'vendor_access.revoked': 'rbac.audit.action.vendor_access_revoked',
  'vendor_access.expired': 'rbac.audit.action.vendor_access_expired',
  'vendor_access.completed': 'rbac.audit.action.vendor_access_completed',
  'permission.override_set': 'rbac.audit.action.permission_override_set',
  'permission.override_removed': 'rbac.audit.action.permission_override_removed',
  'setting.changed': 'rbac.audit.action.setting_changed',
}

export function getActionLabel(t: (key: string) => string, action: string): string {
  const key = LIVE_ACTION_KEY_MAP[action]
  if (key) return t(key)
  return action
}

// ---------------------------------------------------------------------------
// Delta-Rendering Utilities
// ---------------------------------------------------------------------------

/** Prüft ob ein Wert eine Grant-Map ist: { capKey: 'none' | 'own' | 'all' | … } */
function isGrantMap(val: unknown): val is Record<string, string> {
  if (typeof val !== 'object' || val === null || Array.isArray(val)) return false
  const entries = Object.entries(val as Record<string, unknown>)
  if (entries.length === 0) return false
  return entries.every(([, v]) => typeof v === 'string')
}

function DeltaValue({ value }: { value: unknown }) {
  if (value === null || value === undefined) {
    return <span className="text-muted-foreground/60 italic text-xs">—</span>
  }
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return <span className="text-foreground text-sm">{String(value)}</span>
  }
  if (Array.isArray(value)) {
    if (value.length === 0) return <span className="text-muted-foreground/60 italic text-xs">—</span>
    return (
      <div className="flex flex-wrap gap-1">
        {(value as unknown[]).map((item, i) => (
          <span
            key={i}
            className="rounded-full bg-secondary px-2 py-0.5 text-[11px] font-medium text-foreground"
          >
            {String(item)}
          </span>
        ))}
      </div>
    )
  }
  if (isGrantMap(value)) {
    const entries = Object.entries(value)
    return (
      <div className="space-y-0.5">
        {entries.map(([cap, scope]) => (
          <div key={cap} className="flex items-baseline gap-2">
            <code className="shrink-0 text-[10px] font-mono text-muted-foreground">{cap}</code>
            <span className="text-xs text-foreground">{scope}</span>
          </div>
        ))}
      </div>
    )
  }
  // Generic object — key-value rows
  const entries = Object.entries(value as Record<string, unknown>)
  return (
    <div className="space-y-0.5">
      {entries.map(([k, v]) => (
        <div key={k} className="flex items-baseline gap-2">
          <span className="shrink-0 text-[10px] font-medium text-muted-foreground">{k}</span>
          <span className="text-xs text-foreground">{String(v)}</span>
        </div>
      ))}
    </div>
  )
}

/** Berechnet ob sich ein Key zwischen old und new geändert hat, für dezente Hervorhebung. */
function isChangedKey(key: string, oldVal: unknown, newVal: unknown): boolean {
  if (!isGrantMap(oldVal) || !isGrantMap(newVal)) return false
  return (oldVal as Record<string, string>)[key] !== (newVal as Record<string, string>)[key]
}

function DeltaBlock({
  oldValue,
  newValue,
}: {
  oldValue: unknown
  newValue: unknown
}) {
  const { t } = useTranslation()

  // Wenn beide Grant-Maps → zeilenweise Diff-Hervorhebung
  const isGrant = isGrantMap(oldValue) && isGrantMap(newValue)
  const allKeys = isGrant
    ? Array.from(
        new Set([
          ...Object.keys(oldValue as Record<string, string>),
          ...Object.keys(newValue as Record<string, string>),
        ]),
      )
    : []

  return (
    <div className="mt-5">
      <p className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
        {t('rbac.audit.deltaTitle')}
      </p>
      <div className="grid grid-cols-2 gap-3 rounded-lg border border-border bg-secondary/30 p-3">
        {/* Vorher */}
        <div>
          <p className="mb-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {t('rbac.audit.deltaBefore')}
          </p>
          {isGrant ? (
            <div className="space-y-0.5">
              {allKeys.map((key) => {
                const changed = isChangedKey(key, oldValue, newValue)
                const val = (oldValue as Record<string, string>)[key] ?? '—'
                return (
                  <div
                    key={key}
                    className={`flex items-baseline gap-2 rounded px-1 py-0.5 ${changed ? 'bg-warning/10' : ''}`}
                  >
                    <code className="shrink-0 text-[10px] font-mono text-muted-foreground">{key}</code>
                    <span className="text-xs text-foreground">{val}</span>
                  </div>
                )
              })}
            </div>
          ) : (
            <DeltaValue value={oldValue} />
          )}
        </div>
        {/* Nachher */}
        <div>
          <p className="mb-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {t('rbac.audit.deltaAfter')}
          </p>
          {isGrant ? (
            <div className="space-y-0.5">
              {allKeys.map((key) => {
                const changed = isChangedKey(key, oldValue, newValue)
                const val = (newValue as Record<string, string>)[key] ?? '—'
                return (
                  <div
                    key={key}
                    className={`flex items-baseline gap-2 rounded px-1 py-0.5 ${changed ? 'bg-info/10' : ''}`}
                  >
                    <code className="shrink-0 text-[10px] font-mono text-muted-foreground">{key}</code>
                    <span className="text-xs text-foreground">{val}</span>
                  </div>
                )
              })}
            </div>
          ) : (
            <DeltaValue value={newValue} />
          )}
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Extra-Details (role, exit_type, successor, changed_grants, …)
// ---------------------------------------------------------------------------

const SKIP_DETAIL_KEYS = new Set(['old_value', 'new_value'])

function ExtraDetailRow({ label, value }: { label: string; value: unknown }) {
  return (
    <div className="flex items-start gap-3 py-1.5 border-b border-border-muted last:border-0">
      <span className="w-32 shrink-0 text-xs text-muted-foreground">{label}</span>
      <span className="flex-1 text-sm text-foreground">
        <DeltaValue value={value} />
      </span>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main Modal Component
// ---------------------------------------------------------------------------

interface AuditEventDetailModalProps {
  entry: AuditEntry | null
  onClose: () => void
}

export function AuditEventDetailModal({ entry, onClose }: AuditEventDetailModalProps) {
  const { t } = useTranslation()
  const formatDate = useFormatDate()

  if (!entry) return null

  const hasOld = 'old_value' in entry.details
  const hasNew = 'new_value' in entry.details
  const hasDelta = hasOld && hasNew

  // Extrafelder (alles außer old_value / new_value)
  const extraEntries = Object.entries(entry.details).filter(([k]) => !SKIP_DETAIL_KEYS.has(k))

  const actionLabel = getActionLabel(t, entry.action)

  return (
    <DetailModal
      open={entry !== null}
      onClose={onClose}
      title={actionLabel}
      subtitle={formatDate(entry.timestamp, {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      })}
      badge={
        <span
          className={`ml-2 inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${
            entry.result === 'success'
              ? 'bg-success-light text-success'
              : 'bg-error-light text-error'
          }`}
        >
          {t(entry.result === 'success' ? 'audit.result.success' : 'audit.result.failure')}
        </span>
      }
      maxWidth="max-w-2xl"
    >
      {/* Kern-Metadaten */}
      <div className="space-y-0">
        <DetailRow label={t('rbac.audit.detail.actor')}>
          <span className="text-sm font-medium text-foreground">{entry.user_name || entry.user_id}</span>
          {entry.user_name && entry.user_id && (
            <span className="ml-1.5 text-xs text-muted-foreground">({entry.user_id})</span>
          )}
        </DetailRow>
        <DetailRow label={t('rbac.audit.detail.target')}>
          <div className="flex items-center gap-1.5">
            {entry.target_type && (
              <span className="rounded-full bg-secondary px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                {entry.target_type}
              </span>
            )}
            <span className="text-sm text-foreground">{entry.target || '—'}</span>
          </div>
        </DetailRow>
        <DetailRow label={t('rbac.audit.detail.event')}>
          <code className="text-xs font-mono text-muted-foreground">{entry.action}</code>
        </DetailRow>
        <DetailRow label={t('rbac.audit.detail.ip')}>
          <code className="text-xs font-mono text-foreground">{entry.ip_address || '—'}</code>
        </DetailRow>
        {entry.user_agent && (
          <DetailRow label={t('rbac.audit.detail.userAgent')}>
            <span className="text-xs text-muted-foreground break-all">{entry.user_agent}</span>
          </DetailRow>
        )}
        <DetailRow label={t('rbac.audit.detail.sequenceNum')}>
          <span className="text-xs font-mono text-muted-foreground">{String(entry.sequence_num)}</span>
        </DetailRow>
      </div>

      {/* Zusatz-Details-Felder (role, exit_type, successor, changed_grants …) */}
      {extraEntries.length > 0 && (
        <div className="mt-4">
          <p className="mb-1.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            {t('rbac.audit.detail.additionalContext')}
          </p>
          <div className="rounded-lg border border-border bg-secondary/20 px-3">
            {extraEntries.map(([key, value]) => (
              <ExtraDetailRow key={key} label={key} value={value} />
            ))}
          </div>
        </div>
      )}

      {/* Alt/Neu-Delta */}
      {hasDelta && (
        <DeltaBlock
          oldValue={entry.details.old_value}
          newValue={entry.details.new_value}
        />
      )}
    </DetailModal>
  )
}

// ---------------------------------------------------------------------------
// Layout helper
// ---------------------------------------------------------------------------

function DetailRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-3 py-2 border-b border-border-muted last:border-0">
      <span className="w-32 shrink-0 text-xs text-muted-foreground pt-0.5">{label}</span>
      <div className="flex-1 min-w-0">{children}</div>
    </div>
  )
}
