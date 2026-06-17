/**
 * Fristen-Reminder → Notifications.
 *
 * Beim Mount (und wenn sich die Verträge ändern) prüft `useContractExpiryNotifications`,
 * welche Verträge innerhalb ihrer `reminderDays` vor Ablauf liegen, und legt pro
 * fälligem Reminder eine `contract_expiry`-Notification an. Die Notification-ID ist
 * stabil je Vertrag + Schwelle (`contract-expiry-<id>-<threshold>`), sodass
 * Re-Renders/Reloads keine Duplikate erzeugen (idempotent via `ensureNotification`).
 *
 * Die Reminder erscheinen im Notification-Center; die dashboard-AlertsSection liest
 * die auslaufenden Verträge unabhängig direkt aus `useVertraegeStore`.
 */
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useVertraegeStore, type Contract } from '@/stores/vertraege'
import { useNotificationsStore } from '@/stores/notifications'

const MS_PER_DAY = 1000 * 60 * 60 * 24

function startOfDay(d: Date): Date {
  const x = new Date(d)
  x.setHours(0, 0, 0, 0)
  return x
}

export interface DueContractReminder {
  /** Stabile, idempotente Notification-ID: contract-expiry-<contractId>-<threshold> */
  id: string
  contractId: string
  contractTitle: string
  partner: string
  /** Erreichte (engste) Reminder-Schwelle in Tagen, z. B. 30/60/90 */
  threshold: number
  /** Tage bis Vertragsende (> 0) */
  daysUntilEnd: number
  endDate: string
}

/**
 * Ermittelt die fälligen Reminder. Pro Vertrag genau **ein** Reminder: die engste
 * erreichte Schwelle (z. B. 22 Tage Restlaufzeit → 30-Tage-Band). Verträge ohne
 * `reminderDays`, ohne Enddatum, bereits abgelaufen oder gekündigt/expired werden
 * übersprungen.
 */
export function computeDueContractReminders(
  contracts: Contract[],
  now: Date = new Date(),
): DueContractReminder[] {
  const today = startOfDay(now)
  const result: DueContractReminder[] = []
  for (const c of contracts) {
    if (!c.endDate) continue
    if (c.status === 'terminated' || c.status === 'expired') continue
    const reminderDays = c.reminderDays ?? []
    if (reminderDays.length === 0) continue

    const end = startOfDay(new Date(c.endDate + 'T00:00:00'))
    const daysUntilEnd = Math.round((end.getTime() - today.getTime()) / MS_PER_DAY)
    if (daysUntilEnd <= 0) continue

    // Engste erreichte Schwelle = dringendstes aktives Reminder-Band.
    const reached = reminderDays.filter((d) => daysUntilEnd <= d).sort((a, b) => a - b)
    if (reached.length === 0) continue
    const threshold = reached[0]

    result.push({
      id: `contract-expiry-${c.id}-${threshold}`,
      contractId: c.id,
      contractTitle: c.title,
      partner: c.partner,
      threshold,
      daysUntilEnd,
      endDate: c.endDate,
    })
  }
  return result
}

/** Deterministischer Zeitstempel: der Tag, an dem das Reminder-Fenster öffnete. */
function reminderCreatedAt(endDate: string, threshold: number): string {
  const d = new Date(endDate + 'T09:00:00')
  d.setDate(d.getDate() - threshold)
  return d.toISOString()
}

/**
 * Mount-Hook: synchronisiert fällige Fristen-Reminder in den Notification-Store.
 * Idempotent — mehrfaches Mounten/Reload erzeugt keine Duplikate.
 */
export function useContractExpiryNotifications(): void {
  const { t } = useTranslation()
  const contracts = useVertraegeStore((s) => s.contracts)
  const ensureNotification = useNotificationsStore((s) => s.ensureNotification)

  useEffect(() => {
    const due = computeDueContractReminders(contracts)
    for (const r of due) {
      ensureNotification({
        id: r.id,
        type: 'contract_expiry',
        title: t('vertraege.notifications.expiryTitle'),
        body: t('vertraege.notifications.expiryBody', {
          title: r.contractTitle,
          partner: r.partner,
          days: r.daysUntilEnd,
        }),
        icon: 'FileWarning',
        priority: r.daysUntilEnd <= 30 ? 'high' : 'normal',
        actionUrl: '/vertraege',
        createdAt: reminderCreatedAt(r.endDate, r.threshold),
      })
    }
  }, [contracts, ensureNotification, t])
}
