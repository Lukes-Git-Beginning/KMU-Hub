/**
 * Shared status maps / constants for the Einkauf module — extracted from
 * EinkaufPage so the detail modals, dialogs and exports use one source.
 * Labels are i18n keys, resolved with t() at the call site.
 */
import type { POStatus } from '@/api/einkauf-types'

export const orderStatusLabels: Record<string, string> = {
  draft: 'einkauf.orderStatus.draft',
  submitted: 'einkauf.orderStatus.submitted',
  sent: 'einkauf.orderStatus.sent',
  confirmed: 'einkauf.orderStatus.confirmed',
  partial: 'einkauf.orderStatus.partial',
  partially_received: 'einkauf.orderStatus.partial',
  received: 'einkauf.orderStatus.received',
  closed: 'einkauf.orderStatus.received',
  cancelled: 'einkauf.orderStatus.cancelled',
}

export const orderStatusColors: Record<string, string> = {
  draft: 'bg-secondary text-muted-foreground',
  submitted: 'bg-info-light text-info',
  sent: 'bg-info-light text-info',
  confirmed: 'bg-primary-light text-primary',
  partial: 'bg-warning-light text-warning',
  partially_received: 'bg-warning-light text-warning',
  received: 'bg-success-light text-success',
  closed: 'bg-success-light text-success',
  cancelled: 'bg-error-light text-error',
}

export const contractStatusColors: Record<string, string> = {
  active: 'bg-success-light text-success',
  expired: 'bg-secondary text-muted-foreground',
  draft: 'bg-info-light text-info',
}

export const contractStatusLabels: Record<string, string> = {
  active: 'einkauf.contractStatus.active',
  expired: 'einkauf.contractStatus.expired',
  draft: 'einkauf.contractStatus.draft',
}

export const ratingCategoryLabels: Record<string, string> = {
  quality: 'einkauf.ratingCategory.quality',
  delivery: 'einkauf.ratingCategory.delivery',
  price: 'einkauf.ratingCategory.price',
}

/** Simplified progress rail for the detail modal (submitted/sent share a stage). */
export type TimelineStatus = 'draft' | 'sent' | 'partially_received' | 'received'
export const STATUS_TIMELINE: TimelineStatus[] = [
  'draft',
  'sent',
  'partially_received',
  'received',
]

/** Maps every PO status onto its timeline stage (cancelled handled separately). */
export function timelineStage(status: POStatus): TimelineStatus {
  switch (status) {
    case 'submitted':
      return 'sent'
    case 'closed':
      return 'received'
    case 'draft':
    case 'sent':
    case 'partially_received':
    case 'received':
      return status
    default:
      return 'draft'
  }
}

export const PAYMENT_TERMS_OPTIONS = [
  '30 Tage netto',
  '60 Tage netto',
  '14 Tage 2% Skonto',
  '45 Tage netto',
  'Vorkasse',
  'Rechnung',
]

/** Statuses that still allow booking a goods receipt. */
export function canReceiveGoods(status: POStatus): boolean {
  return status === 'submitted' || status === 'sent' || status === 'partially_received'
}

/** Only drafts can be freely edited (positions); others allow date/notes only. */
export function isEditableDraft(status: POStatus): boolean {
  return status === 'draft'
}

/** Open = neither fully received/closed nor cancelled. */
export function isOpenOrder(status: POStatus): boolean {
  return status !== 'received' && status !== 'closed' && status !== 'cancelled'
}
