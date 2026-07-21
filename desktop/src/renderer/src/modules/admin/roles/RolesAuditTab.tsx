/**
 * RolesAuditTab — gefilterte Audit-Sicht im Rollen-Bereich.
 *
 * Zeigt nur Events mit action-Präfix `role.` oder `permission.` sowie den
 * Legacy-Seed-Action `user_role_changed`. Client-seitiger Filter über den
 * bestehenden useAuditLog-Hook (der MSW-Handler filtert nur exakte Matches).
 *
 * Wiederverwendet AuditEventDetailModal aus modules/security — kein doppelter Code.
 * Gate: security:audit:read (useHasCapability).
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useFormatDate } from '@/hooks/useFormatters'
import { Shield } from 'lucide-react'
import { useAuditLog } from '@/api/hooks/useSecurity'
import { EmptyState } from '@/components/shared'
import { AuditEventDetailModal, getActionLabel } from '@/modules/security/AuditEventDetailModal'
import type { AuditEntry } from '@/api/security-types'

/** Actions die im Protokoll-Tab erscheinen sollen. */
function isRoleRelatedAction(action: string): boolean {
  return (
    action.startsWith('role.') ||
    action.startsWith('permission.') ||
    action === 'user_role_changed'
  )
}

export function RolesAuditTab() {
  const { t } = useTranslation()
  const formatDate = useFormatDate()
  const [selectedEntry, setSelectedEntry] = useState<AuditEntry | null>(null)

  // Lade alle Events ohne Paging — der Rollen-Protokoll-Tab zeigt maximal die
  // letzten 200 Einträge (MSW-Limit), client-seitig auf Rollen-Events gefiltert.
  const { data, isLoading } = useAuditLog({ offset: 0, limit: 200 })

  const entries = useMemo(
    () => (data?.entries ?? []).filter((e) => isRoleRelatedAction(e.action)),
    [data?.entries],
  )

  if (isLoading) {
    return (
      <div className="px-6 py-6">
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-10 animate-pulse rounded-lg bg-secondary/40" />
          ))}
        </div>
      </div>
    )
  }

  if (entries.length === 0) {
    return (
      <div className="px-6 py-4">
        <EmptyState
          icon={Shield}
          title={t('rbac.audit.protocolEmpty')}
        />
      </div>
    )
  }

  return (
    <div className="px-6 py-4">
      <div className="rounded-lg border border-border bg-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  {t('audit.timestamp')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  {t('audit.user')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  {t('audit.action')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                  {t('audit.target')}
                </th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <tr
                  key={entry.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => setSelectedEntry(entry)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      setSelectedEntry(entry)
                    }
                  }}
                  className="border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-focus-ring"
                >
                  <td className="px-4 py-3 whitespace-nowrap text-xs text-muted-foreground">
                    {formatDate(entry.timestamp, {
                      year: 'numeric',
                      month: '2-digit',
                      day: '2-digit',
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </td>
                  <td className="px-4 py-3 text-sm text-foreground">
                    {entry.user_name || entry.user_id}
                  </td>
                  <td className="px-4 py-3 text-sm font-medium text-foreground">
                    {getActionLabel(t, entry.action)}
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">
                    {entry.target_type && (
                      <span className="rounded-full bg-secondary px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground mr-1.5">
                        {entry.target_type}
                      </span>
                    )}
                    {entry.target}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Detail-Modal — gleiche Komponente wie in AuditLogPage */}
      <AuditEventDetailModal
        entry={selectedEntry}
        onClose={() => setSelectedEntry(null)}
      />
    </div>
  )
}
