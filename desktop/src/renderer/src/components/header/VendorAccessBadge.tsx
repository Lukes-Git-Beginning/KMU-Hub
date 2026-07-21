/**
 * VendorAccessBadge — dezente Header-Pill für aktiven/offenen Anbieter-Zugang.
 *
 * Sichtbar NUR für User mit `security:vendor_access:manage` UND wenn eine
 * active oder pending Anfrage existiert.
 *  - active:  „Zentria-Zugriff aktiv · noch {days} Tage"
 *  - pending: „Zugangs-Anfrage von Zentria"
 *
 * Klick → /admin/security mit Query-Parameter ?subtab=vendor-access,
 * sodass SecurityAdminHubTab direkt auf den richtigen Sub-Tab springt.
 */
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Shield } from 'lucide-react'
import { useHasCapability } from '@/hooks/useCapability'
import { useVendorAccessList } from '@/api/vendor-access'

function daysRemaining(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
}

export function VendorAccessBadge() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const canManage = useHasCapability('security:vendor_access:manage')

  // Nur laden wenn Berechtigung vorhanden — verhindert unnötige Requests
  const { data: requests = [] } = useVendorAccessList()

  if (!canManage) return null

  const activeReq = requests.find((r) => r.status === 'active')
  const pendingReq = requests.find(
    (r) => r.status === 'pending' || r.status === 'counter_proposed',
  )

  if (!activeReq && !pendingReq) return null

  const label = activeReq
    ? t('rbac.vendorAccess.badge.active', {
        count: daysRemaining(activeReq.expires_at),
      })
    : t('rbac.vendorAccess.badge.pending')

  function handleClick() {
    navigate('/admin/security?subtab=vendor-access')
  }

  return (
    <button
      onClick={handleClick}
      className={[
        'hidden lg:inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium',
        'transition-colors',
        activeReq
          ? 'border-primary/30 bg-primary/5 text-primary hover:bg-primary/10'
          : 'border-border bg-secondary text-muted-foreground hover:bg-accent hover:text-foreground',
      ].join(' ')}
      aria-label={label}
    >
      <Shield className="h-3 w-3 shrink-0" />
      <span className="max-w-[200px] truncate">{label}</span>
    </button>
  )
}
