/**
 * Presentation helpers for the admin Benutzer area — role/status visuals,
 * initials, and locale-correct relative timestamps. Kept framework-light so the
 * tab, detail modal and invite dialog stay consistent.
 */
import { CheckCircle2, Clock, MinusCircle, type LucideIcon } from 'lucide-react'
import type { RoleId } from '@/config/roles'
import type { AdminUserStatus } from '@/api/admin-types'

/** Subtle role accent dot — mirrors the DEV_PROFILES palette in @/config/roles. */
export const ROLE_DOT: Record<RoleId, string> = {
  admin: 'bg-[hsl(0_72%_51%)]',
  manager: 'bg-[hsl(217_91%_60%)]',
  member: 'bg-[hsl(142_71%_45%)]',
  hr: 'bg-[hsl(270_76%_55%)]',
  it_support: 'bg-[hsl(25_95%_53%)]',
}

export const ROLE_ORDER: RoleId[] = ['admin', 'it_support', 'manager', 'hr', 'member']

/** i18n key for a role label — reuses the existing config.roles.* namespace. */
export function roleLabelKey(role: RoleId): string {
  return `config.roles.${role}.label`
}

export interface StatusMeta {
  labelKey: string
  /** Pill classes (filled, semantic). */
  badgeClass: string
  /** Small status icon so the state reads without relying on colour alone. */
  icon: LucideIcon
  /** Accent for the leading dot in dense contexts. */
  dotClass: string
}

export const STATUS_META: Record<AdminUserStatus, StatusMeta> = {
  active: {
    labelKey: 'admin.users.status.active',
    badgeClass: 'bg-success-light text-success',
    icon: CheckCircle2,
    dotClass: 'bg-success',
  },
  invited: {
    labelKey: 'admin.users.status.invited',
    badgeClass: 'bg-warning-light text-warning',
    icon: Clock,
    dotClass: 'bg-warning',
  },
  deactivated: {
    labelKey: 'admin.users.status.deactivated',
    badgeClass: 'bg-secondary text-muted-foreground',
    icon: MinusCircle,
    dotClass: 'bg-muted-foreground',
  },
}

export function initials(firstName: string, lastName: string): string {
  return `${firstName[0] ?? ''}${lastName[0] ?? ''}`.toUpperCase()
}

/** Locale-correct "vor X" using Intl — covers de/en/fr/it without i18n keys. */
export function formatRelative(iso: string | null, locale: string, neverLabel: string): string {
  if (!iso) return neverLabel
  const diffMs = Date.now() - new Date(iso).getTime()
  const rtf = new Intl.RelativeTimeFormat(locale || 'de', { numeric: 'auto' })
  const minutes = Math.round(diffMs / 60_000)
  if (Math.abs(minutes) < 60) return rtf.format(-minutes, 'minute')
  const hours = Math.round(minutes / 60)
  if (Math.abs(hours) < 24) return rtf.format(-hours, 'hour')
  const days = Math.round(hours / 24)
  if (Math.abs(days) < 30) return rtf.format(-days, 'day')
  const months = Math.round(days / 30)
  return rtf.format(-months, 'month')
}
