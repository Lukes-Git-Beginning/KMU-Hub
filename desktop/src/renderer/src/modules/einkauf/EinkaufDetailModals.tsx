/**
 * Detail modals for the Einkauf module (shared/DetailModal, Cosmi-Fenster-Stil):
 *
 * - OrderDetailModal — Bestellung mit Freigabe-Banner, Summen, Terminen,
 *   Status-Timeline, Positionen (via usePOLines — die Listen-Query trägt keine
 *   lines!), Lieferant-Link (Back-Kette) und Belegkette. Statusabhängige
 *   Footer-Aktionen: Entwurf senden (useSubmitPO), Wareneingang, Bearbeiten,
 *   PDF-Export, Stornieren.
 * - SupplierDetailModal — Lieferant mit Kontakt/Konditionen, Bewertungen
 *   inkl. Inline-Formular (useCreateSupplierRating) und klickbarer
 *   Bestellhistorie (Back-Kette).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  X,
  Check,
  Circle,
  Star,
  Send,
  PackageCheck,
  CalendarDays,
  Clock,
  ChevronRight,
  Truck,
  Mail,
  Phone,
  CreditCard,
  UserCircle,
  ShieldCheck,
  FileText,
  Link2,
  Coins,
  BarChart3,
  FileDown,
  Pencil,
  Ban,
  MapPin,
  StickyNote,
} from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import { formatCurrency, formatDate } from '@/lib/format'
import type { PurchaseOrder, Supplier, RatingCategory } from '@/api/einkauf-types'
import {
  usePOLines,
  useSupplierRatings,
  useCreateSupplierRating,
  useSubmitPO,
} from '@/api/hooks/useEinkauf'
import { useEinkaufTenantStore } from '@/stores/einkaufTenant'
import {
  orderStatusLabels,
  orderStatusColors,
  ratingCategoryLabels,
  STATUS_TIMELINE,
  timelineStage,
  canReceiveGoods,
  isEditableDraft,
} from './einkauf-shared'
import { buildPOPdf, downloadBlob } from './einkauf-export'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

export function StarRating({ rating, max = 5 }: { rating: number; max?: number }) {
  return (
    <div className="flex items-center gap-0.5">
      {Array.from({ length: max }, (_, i) => (
        <Star
          key={i}
          className={`h-3.5 w-3.5 ${i < rating ? 'fill-warning text-warning' : 'text-border'}`}
        />
      ))}
    </div>
  )
}

function formatDateLocal(d: string) {
  return formatDate(d.includes('T') ? d : d + 'T00:00:00')
}

function OrderStatusBadge({ status }: { status: string }) {
  const { t } = useTranslation()
  return (
    <span
      className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
        orderStatusColors[status] ?? 'bg-secondary text-muted-foreground'
      }`}
    >
      {orderStatusLabels[status] ? t(orderStatusLabels[status]) : status}
    </span>
  )
}

function StatusTimeline({ status }: { status: PurchaseOrder['status'] }) {
  const { t } = useTranslation()
  if (status === 'cancelled') {
    return (
      <div className="flex items-center gap-2 text-xs text-error">
        <X className="h-3.5 w-3.5" />
        <span>{t('einkauf.detail.cancelled')}</span>
      </div>
    )
  }
  const currentIdx = STATUS_TIMELINE.indexOf(timelineStage(status))
  return (
    <div className="flex items-center gap-1">
      {STATUS_TIMELINE.map((s, idx) => {
        const reached = idx <= currentIdx
        return (
          <div key={s} className="flex items-center gap-1">
            <div className="flex flex-col items-center">
              <div
                className={`flex h-5 w-5 items-center justify-center rounded-full ${
                  reached ? 'bg-primary text-primary-foreground' : 'bg-secondary text-muted-foreground'
                }`}
              >
                {reached ? <Check className="h-3 w-3" /> : <Circle className="h-3 w-3" />}
              </div>
              <span
                className={`mt-1 text-[9px] leading-tight ${
                  reached ? 'font-medium text-foreground' : 'text-muted-foreground'
                }`}
              >
                {orderStatusLabels[s] ? t(orderStatusLabels[s]) : s}
              </span>
            </div>
            {idx < STATUS_TIMELINE.length - 1 && (
              <div className={`mx-0.5 h-px w-4 ${idx < currentIdx ? 'bg-primary' : 'bg-border'}`} />
            )}
          </div>
        )
      })}
    </div>
  )
}

// ---------------------------------------------------------------------------
// OrderDetailModal
// ---------------------------------------------------------------------------

interface OrderDetailModalProps {
  order: PurchaseOrder | null
  suppliers: Supplier[]
  onClose: () => void
  /** Set when this modal was opened from the supplier modal (back chain). */
  onBack?: () => void
  onOpenSupplier: (supplier: Supplier) => void
  onBookReceipt: (order: PurchaseOrder) => void
  onEdit: (order: PurchaseOrder) => void
  onCancel: (order: PurchaseOrder) => void
}

export function OrderDetailModal({
  order,
  suppliers,
  onClose,
  onBack,
  onOpenSupplier,
  onBookReceipt,
  onEdit,
  onCancel,
}: OrderDetailModalProps) {
  const { t } = useTranslation()
  const approvalThreshold = useEinkaufTenantStore((s) => s.approvalThreshold)
  const submitPOMutation = useSubmitPO()

  // The list query carries no lines — fetch them for the open order.
  const { data: linesData } = usePOLines(order?.id ?? '')
  const lines = linesData?.lines ?? []

  const supplier = order ? suppliers.find((s) => s.id === order.supplier_id) : undefined
  const currency = order?.currency || 'EUR'

  const handleSubmit = (approve: boolean) => {
    if (!order) return
    submitPOMutation.mutate(order.id, {
      onSuccess: () => {
        toast.success(
          t(approve ? 'einkauf.toast.orderApproved' : 'einkauf.toast.orderSubmitted', {
            orderNumber: order.po_number,
          }),
        )
        onClose()
      },
      onError: () => toast.error(t('einkauf.toast.orderSubmitFailed')),
    })
  }

  const handleExportPdf = () => {
    if (!order) return
    const blob = buildPOPdf(order, lines, supplier?.name ?? order.supplier_id, supplier?.payment_terms ?? '', {
      title: t('einkauf.pdf.title'),
      supplier: t('einkauf.detail.supplier'),
      orderDate: t('einkauf.pdf.orderDate'),
      deliveryDate: t('einkauf.pdf.deliveryDate'),
      paymentTerms: t('einkauf.pdf.paymentTerms'),
      positions: t('einkauf.newOrder.positions'),
      qty: t('einkauf.newOrder.qty'),
      net: t('einkauf.pdf.net'),
      vat: t('einkauf.pdf.vat'),
      gross: t('einkauf.pdf.gross'),
      notes: t('einkauf.newOrder.notes'),
      noPositions: t('einkauf.detail.noPositions'),
      morePositions: (count) => t('einkauf.pdf.morePositions', { count }),
    })
    downloadBlob(blob, `${order.po_number}.pdf`)
    toast.success(t('einkauf.toast.pdfExported', { orderNumber: order.po_number }))
  }

  const receivable = order ? canReceiveGoods(order.status) : false
  const draft = order ? isEditableDraft(order.status) : false
  const cancellable = order ? order.status !== 'received' && order.status !== 'closed' && order.status !== 'cancelled' : false

  return (
    <DetailModal
      open={!!order}
      onClose={onClose}
      onBack={onBack}
      title={order?.po_number ?? ''}
      subtitle={supplier?.name ?? order?.supplier_id}
      badge={order ? <OrderStatusBadge status={order.status} /> : undefined}
      footer={
        order ? (
          <div className="flex flex-wrap items-center gap-2">
            {draft && (
              <button
                onClick={() => handleSubmit(false)}
                disabled={submitPOMutation.isPending}
                className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
              >
                <Send className="h-4 w-4" />
                {t('einkauf.action.submit')}
              </button>
            )}
            {receivable && (
              <button
                onClick={() => onBookReceipt(order)}
                className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                <PackageCheck className="h-4 w-4" />
                {t('einkauf.action.bookReceipt')}
              </button>
            )}
            <button
              onClick={() => onEdit(order)}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              <Pencil className="h-3.5 w-3.5" />
              {t('einkauf.action.edit')}
            </button>
            <button
              onClick={handleExportPdf}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              <FileDown className="h-3.5 w-3.5" />
              {t('einkauf.action.exportPdf')}
            </button>
            {cancellable && (
              <button
                onClick={() => onCancel(order)}
                className="ml-auto flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-error hover:bg-error-light transition-colors"
              >
                <Ban className="h-3.5 w-3.5" />
                {t('einkauf.action.cancel')}
              </button>
            )}
          </div>
        ) : undefined
      }
    >
      {order && (
        <div className="space-y-5">
          {/* Approval banner — draft orders above the tenant threshold */}
          {draft && approvalThreshold > 0 && parseFloat(order.total_amount) > approvalThreshold && (
            <div className="rounded-lg p-3 flex items-center gap-3 bg-warning-light border border-warning/20">
              <ShieldCheck className="h-5 w-5 shrink-0 text-warning" />
              <div className="flex-1 min-w-0">
                <p className="text-sm text-warning font-medium">
                  {t('einkauf.detail.approvalRequired', {
                    threshold: formatCurrency(approvalThreshold, currency),
                  })}
                </p>
              </div>
              <button
                onClick={() => handleSubmit(true)}
                disabled={submitPOMutation.isPending}
                className="shrink-0 rounded-lg bg-warning px-3 py-1.5 text-xs font-medium text-warning-foreground hover:opacity-90 transition-opacity disabled:opacity-50"
              >
                {t('einkauf.action.approve')}
              </button>
            </div>
          )}

          {/* Summary */}
          <div className="grid grid-cols-2 gap-3">
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                {t('einkauf.detail.amount')}
              </p>
              <p className="text-lg font-semibold text-foreground tabular-nums">
                {formatCurrency(parseFloat(order.total_amount), currency)}
              </p>
            </div>
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                {t('einkauf.detail.positions')}
              </p>
              <p className="text-lg font-semibold text-foreground">{lines.length}</p>
            </div>
          </div>

          {/* Dates + terms */}
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <CalendarDays className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">{t('einkauf.detail.deliveryDateLabel')}</span>
              <span className="text-foreground">
                {order.expected_delivery_date ? formatDateLocal(order.expected_delivery_date) : '—'}
              </span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">{t('einkauf.detail.createdLabel')}</span>
              <span className="text-foreground">{formatDate(order.created_at)}</span>
            </div>
            {supplier?.payment_terms && (
              <div className="flex items-center gap-2 text-sm">
                <CreditCard className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">{t('einkauf.detail.paymentTermsLabel')}</span>
                <span className="text-foreground">{supplier.payment_terms}</span>
              </div>
            )}
          </div>

          {/* Status timeline */}
          <div>
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">
              {t('einkauf.detail.statusHistory')}
            </h4>
            <StatusTimeline status={order.status} />
          </div>

          {/* Order items */}
          <div>
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
              {t('einkauf.detail.orderedPositions')}
            </h4>
            <div className="rounded-lg border border-border divide-y divide-border-muted">
              {lines.length === 0 ? (
                <p className="px-3 py-4 text-xs text-muted-foreground text-center">
                  {t('einkauf.detail.noPositions')}
                </p>
              ) : (
                lines.map((item) => {
                  const qty = parseFloat(item.quantity)
                  const unitPrice = parseFloat(item.unit_price)
                  const receivedQty = parseFloat(item.received_quantity)
                  return (
                    <div key={item.id} className="flex items-center justify-between px-3 py-2.5">
                      <div>
                        <p className="text-sm text-foreground">{item.product_name}</p>
                        <p className="text-xs text-muted-foreground">
                          {t('einkauf.detail.qty', {
                            qty,
                            price: formatCurrency(unitPrice, currency),
                          })}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm font-medium text-foreground tabular-nums">
                          {formatCurrency(qty * unitPrice, currency)}
                        </p>
                        {receivedQty > 0 && (
                          <p className="text-[10px] text-success">
                            {t('einkauf.detail.receivedQty', { received: receivedQty, total: qty })}
                          </p>
                        )}
                      </div>
                    </div>
                  )
                })
              )}
            </div>
          </div>

          {/* Notes */}
          {order.notes && (
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                {t('einkauf.detail.notes')}
              </h4>
              <div className="flex items-start gap-2 rounded-lg border border-border p-3">
                <StickyNote className="h-4 w-4 shrink-0 text-muted-foreground mt-0.5" />
                <p className="text-sm text-foreground whitespace-pre-wrap">{order.notes}</p>
              </div>
            </div>
          )}

          {/* Supplier link */}
          <div>
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
              {t('einkauf.detail.supplierSection')}
            </h4>
            <button
              onClick={() => supplier && onOpenSupplier(supplier)}
              disabled={!supplier}
              className="flex w-full items-center justify-between rounded-lg border border-border p-3 hover:bg-secondary/50 transition-colors disabled:opacity-60"
            >
              <div className="flex items-center gap-2">
                <Truck className="h-4 w-4 text-primary" />
                <span className="text-sm text-foreground">{supplier?.name ?? order.supplier_id}</span>
              </div>
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            </button>
          </div>

          {/* Belegkette (document chain) — only for non-draft, non-cancelled */}
          {order.status !== 'draft' && order.status !== 'cancelled' && (
            <div>
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                {t('einkauf.detail.documentChain')}
              </h4>
              <div className="rounded-lg border border-border p-3">
                <div className="flex items-center gap-2">
                  <div className="flex items-center gap-1.5 rounded-lg bg-primary-light px-2.5 py-1.5">
                    <FileText className="h-3.5 w-3.5 text-primary" />
                    <span className="text-xs font-medium text-primary">{t('einkauf.detail.order')}</span>
                  </div>
                  <div className="h-px w-4 bg-border" />
                  <div
                    className={`flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 ${
                      order.status === 'partially_received' || order.status === 'received' || order.status === 'closed'
                        ? 'bg-success-light'
                        : 'bg-secondary'
                    }`}
                  >
                    <Link2
                      className={`h-3.5 w-3.5 ${
                        order.status === 'partially_received' || order.status === 'received' || order.status === 'closed'
                          ? 'text-success'
                          : 'text-muted-foreground'
                      }`}
                    />
                    <span
                      className={`text-xs font-medium ${
                        order.status === 'partially_received' || order.status === 'received' || order.status === 'closed'
                          ? 'text-success'
                          : 'text-muted-foreground'
                      }`}
                    >
                      {t('einkauf.detail.deliveryNote')}
                    </span>
                  </div>
                  <div className="h-px w-4 bg-border" />
                  <div className="flex items-center gap-1.5 rounded-lg bg-secondary px-2.5 py-1.5">
                    <Coins className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="text-xs font-medium text-muted-foreground">
                      {t('einkauf.detail.incomingInvoice')}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </DetailModal>
  )
}

// ---------------------------------------------------------------------------
// SupplierDetailModal
// ---------------------------------------------------------------------------

interface SupplierDetailModalProps {
  supplier: Supplier | null
  orders: PurchaseOrder[]
  onClose: () => void
  /** Set when this modal was opened from the order modal (back chain). */
  onBack?: () => void
  onOpenOrder: (order: PurchaseOrder) => void
  onEdit: (supplier: Supplier) => void
  onDeactivate: (supplier: Supplier) => void
}

const RATING_CATEGORIES: RatingCategory[] = ['quality', 'delivery', 'price']

export function SupplierDetailModal({
  supplier,
  orders,
  onClose,
  onBack,
  onOpenOrder,
  onEdit,
  onDeactivate,
}: SupplierDetailModalProps) {
  const { t } = useTranslation()
  const { data: ratingsData } = useSupplierRatings(supplier?.id ?? '')
  const ratings = ratingsData?.ratings ?? []
  const createRatingMutation = useCreateSupplierRating()

  const [showRatingForm, setShowRatingForm] = useState(false)
  const [ratingCategory, setRatingCategory] = useState<RatingCategory>('quality')
  const [ratingValue, setRatingValue] = useState(4)
  const [ratingComment, setRatingComment] = useState('')

  const avg = ratings.length === 0 ? 0 : ratings.reduce((sum, r) => sum + r.rating, 0) / ratings.length

  const resetRatingForm = () => {
    setShowRatingForm(false)
    setRatingCategory('quality')
    setRatingValue(4)
    setRatingComment('')
  }

  const handleSaveRating = () => {
    if (!supplier) return
    createRatingMutation.mutate(
      {
        supplierId: supplier.id,
        category: ratingCategory,
        rating: ratingValue,
        comment: ratingComment.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast.success(t('einkauf.toast.ratingSaved', { name: supplier.name }))
          resetRatingForm()
        },
        onError: () => toast.error(t('einkauf.toast.ratingSaveFailed')),
      },
    )
  }

  return (
    <DetailModal
      open={!!supplier}
      onClose={onClose}
      onBack={onBack}
      title={supplier?.name ?? ''}
      subtitle={supplier?.email}
      footer={
        supplier ? (
          <div className="flex items-center gap-2">
            <button
              onClick={() => onEdit(supplier)}
              className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              <Pencil className="h-3.5 w-3.5" />
              {t('einkauf.action.edit')}
            </button>
            <button
              onClick={() => onDeactivate(supplier)}
              className="ml-auto flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-error hover:bg-error-light transition-colors"
            >
              <Ban className="h-3.5 w-3.5" />
              {t('einkauf.action.deactivate')}
            </button>
          </div>
        ) : undefined
      }
    >
      {supplier && (
        <div className="space-y-5">
          {/* Contact info */}
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
                <UserCircle className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-sm font-medium text-foreground">{supplier.name}</p>
                <p className="text-xs text-muted-foreground">{t('einkauf.supplier.contactPerson')}</p>
              </div>
            </div>

            <div className="rounded-lg border border-border divide-y divide-border-muted">
              {supplier.email && (
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <Mail className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm text-foreground">{supplier.email}</span>
                </div>
              )}
              {supplier.phone && (
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <Phone className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm text-foreground">{supplier.phone}</span>
                </div>
              )}
              {supplier.address && (
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <MapPin className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm text-foreground">{supplier.address}</span>
                </div>
              )}
              {supplier.payment_terms && (
                <div className="flex items-center gap-3 px-3 py-2.5">
                  <CreditCard className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm text-foreground">{supplier.payment_terms}</span>
                </div>
              )}
            </div>
          </div>

          {/* Ratings + inline form */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                {t('einkauf.detail.ratings')}
              </h4>
              {!showRatingForm && (
                <button
                  onClick={() => setShowRatingForm(true)}
                  className="text-xs text-primary hover:underline"
                >
                  {t('einkauf.action.addRating')}
                </button>
              )}
            </div>

            {ratings.length > 0 && (
              <div className="rounded-lg border border-border p-3 space-y-3 mb-3">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-warning-light">
                    <BarChart3 className="h-5 w-5 text-warning" />
                  </div>
                  <div>
                    <p className="text-lg font-semibold text-foreground tabular-nums">{avg.toFixed(1)}</p>
                    <p className="text-[10px] text-muted-foreground">{t('einkauf.detail.average')}</p>
                  </div>
                  <div className="ml-auto">
                    <StarRating rating={Math.round(avg)} />
                  </div>
                </div>

                <div className="space-y-2 border-t border-border-muted pt-3">
                  {RATING_CATEGORIES.map((cat) => {
                    const catRatings = ratings.filter((r) => r.category === cat)
                    if (catRatings.length === 0) return null
                    const catAvg = catRatings.reduce((s, r) => s + r.rating, 0) / catRatings.length
                    return (
                      <div key={cat} className="flex items-center justify-between">
                        <span className="text-xs text-muted-foreground">
                          {ratingCategoryLabels[cat] ? t(ratingCategoryLabels[cat]) : cat}
                        </span>
                        <div className="flex items-center gap-2">
                          <StarRating rating={Math.round(catAvg)} />
                          <span className="text-xs text-foreground font-medium tabular-nums w-6 text-right">
                            {catAvg.toFixed(1)}
                          </span>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            )}

            {showRatingForm && (
              <div className="rounded-lg border border-border p-3 space-y-3">
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <label className="text-xs text-muted-foreground">{t('einkauf.rating.category')}</label>
                    <select
                      value={ratingCategory}
                      onChange={(e) => setRatingCategory(e.target.value as RatingCategory)}
                      className="w-full rounded-lg border border-border bg-card px-2 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    >
                      {RATING_CATEGORIES.map((cat) => (
                        <option key={cat} value={cat}>
                          {t(ratingCategoryLabels[cat])}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="space-y-1">
                    <label className="text-xs text-muted-foreground">{t('einkauf.rating.stars')}</label>
                    <div className="flex items-center gap-1 py-1.5" role="radiogroup" aria-label={t('einkauf.rating.stars')}>
                      {[1, 2, 3, 4, 5].map((v) => (
                        <button
                          key={v}
                          role="radio"
                          aria-checked={ratingValue === v}
                          aria-label={String(v)}
                          onClick={() => setRatingValue(v)}
                          className="p-0.5"
                        >
                          <Star
                            className={`h-5 w-5 transition-colors ${
                              v <= ratingValue ? 'fill-warning text-warning' : 'text-border hover:text-warning'
                            }`}
                          />
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
                <input
                  type="text"
                  value={ratingComment}
                  onChange={(e) => setRatingComment(e.target.value)}
                  placeholder={t('einkauf.rating.commentPlaceholder')}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
                <div className="flex items-center justify-end gap-2">
                  <button
                    onClick={resetRatingForm}
                    className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                  >
                    {t('common.cancel')}
                  </button>
                  <button
                    onClick={handleSaveRating}
                    disabled={createRatingMutation.isPending}
                    className="rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
                  >
                    {t('einkauf.rating.save')}
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Recent orders */}
          <div>
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
              {t('einkauf.detail.orders', { count: orders.length })}
            </h4>
            <div className="rounded-lg border border-border divide-y divide-border-muted">
              {orders.length === 0 ? (
                <p className="px-3 py-4 text-xs text-muted-foreground text-center">
                  {t('einkauf.detail.noOrders')}
                </p>
              ) : (
                orders.map((order) => (
                  <button
                    key={order.id}
                    onClick={() => onOpenOrder(order)}
                    className="flex w-full items-center justify-between px-3 py-2.5 hover:bg-secondary/50 transition-colors"
                  >
                    <div className="text-left">
                      <p className="text-sm font-mono text-foreground">{order.po_number}</p>
                      <p className="text-xs text-muted-foreground">
                        {formatDate(order.created_at)} ·{' '}
                        {formatCurrency(parseFloat(order.total_amount), order.currency || 'EUR')}
                      </p>
                    </div>
                    <OrderStatusBadge status={order.status} />
                  </button>
                ))
              )}
            </div>
          </div>
        </div>
      )}
    </DetailModal>
  )
}
