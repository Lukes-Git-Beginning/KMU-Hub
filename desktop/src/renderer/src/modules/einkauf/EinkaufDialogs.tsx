/**
 * Action dialogs for the Einkauf module — sie verdrahten die bislang toten
 * toast.info-Buttons an die (großteils schon vorhandenen) Mutations-Hooks:
 *
 * - EditOrderDialog     — Bestellung bearbeiten (useUpdatePO); bei Entwürfen
 *   zusätzlich Positions-Editing (useAddPOLine/useUpdatePOLine/useDeletePOLine)
 * - EditSupplierDialog  — Lieferant bearbeiten (useUpdateSupplier)
 * - CartDialog          — Einkaufs-Warenkorb: Katalog-Artikel gesammelt, pro
 *   Lieferant gebündelt zu Bestellungen (weclapp/Xentral-Bestellvorschlag)
 * - NewCallDialog       — Abruf aus Rahmenvertrag (useCreateContractCall)
 * - ContractCallsList   — bisherige Abrufe eines Rahmenvertrags
 *
 * Alle Dialoge erwarten einen Remount-key an der Aufrufstelle
 * (key={entity?.id ?? 'closed'}) gegen stale Form-State.
 */
import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Plus, Trash2, Pencil, Truck, ShoppingCart, ScrollText, PackageCheck } from 'lucide-react'
import { toast } from 'sonner'
import { formatCurrency, formatDate } from '@/lib/format'
import type { PurchaseOrder, Supplier, CatalogItem, FrameworkContract } from '@/api/einkauf-types'
import {
  usePOLines,
  useUpdatePO,
  useAddPOLine,
  useUpdatePOLine,
  useDeletePOLine,
  useUpdateSupplier,
  useCreatePO,
  useContractCalls,
  useCreateContractCall,
  useReceiveGoods,
  usePartialReceive,
} from '@/api/hooks/useEinkauf'
import { useEinkaufTenantStore } from '@/stores/einkaufTenant'
import { PAYMENT_TERMS_OPTIONS, isEditableDraft } from './einkauf-shared'

const inputCls =
  'w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring'

// ---------------------------------------------------------------------------
// EditOrderDialog
// ---------------------------------------------------------------------------

interface EditRow {
  /** Existing line id, or null for a freshly added row. */
  lineId: string | null
  name: string
  quantity: number
  unitPrice: number
}

interface EditOrderDialogProps {
  order: PurchaseOrder | null
  onClose: () => void
}

export function EditOrderDialog({ order, onClose }: EditOrderDialogProps) {
  const { t } = useTranslation()
  const draft = order ? isEditableDraft(order.status) : false

  const [deliveryDate, setDeliveryDate] = useState(order?.expected_delivery_date?.slice(0, 10) ?? '')
  const [notes, setNotes] = useState(order?.notes ?? '')
  // null = positions not seeded yet (lines arrive async)
  const [rows, setRows] = useState<EditRow[] | null>(null)
  const [removedLineIds, setRemovedLineIds] = useState<string[]>([])

  const { data: linesData } = usePOLines(order?.id ?? '')
  useEffect(() => {
    if (rows === null && linesData) {
      setRows(
        linesData.lines
          .slice()
          .sort((a, b) => a.line_position - b.line_position)
          .map((l) => ({
            lineId: l.id,
            name: l.product_name,
            quantity: parseFloat(l.quantity),
            unitPrice: parseFloat(l.unit_price),
          })),
      )
    }
  }, [linesData, rows])

  const updatePOMutation = useUpdatePO()
  const addLineMutation = useAddPOLine()
  const updateLineMutation = useUpdatePOLine()
  const deleteLineMutation = useDeletePOLine()

  const saving =
    updatePOMutation.isPending ||
    addLineMutation.isPending ||
    updateLineMutation.isPending ||
    deleteLineMutation.isPending

  const total = useMemo(
    () => (rows ?? []).reduce((s, r) => s + r.quantity * r.unitPrice, 0),
    [rows],
  )

  const updateRow = (idx: number, patch: Partial<EditRow>) => {
    setRows((prev) => (prev ?? []).map((r, i) => (i === idx ? { ...r, ...patch } : r)))
  }

  const removeRow = (idx: number) => {
    setRows((prev) => {
      const list = prev ?? []
      const row = list[idx]
      if (row?.lineId) setRemovedLineIds((ids) => [...ids, row.lineId as string])
      return list.filter((_, i) => i !== idx)
    })
  }

  const handleSave = async () => {
    if (!order) return
    if (draft && (rows ?? []).some((r) => !r.name.trim())) {
      toast.error(t('einkauf.validation.itemsNeedName'))
      return
    }
    try {
      await updatePOMutation.mutateAsync({
        id: order.id,
        expected_delivery_date: deliveryDate || null,
        notes,
      })
      if (draft && rows) {
        for (const lineId of removedLineIds) {
          await deleteLineMutation.mutateAsync({ poId: order.id, lineId })
        }
        for (const [idx, row] of rows.entries()) {
          if (row.lineId) {
            await updateLineMutation.mutateAsync({
              poId: order.id,
              lineId: row.lineId,
              product_name: row.name,
              quantity: String(row.quantity),
              unit_price: String(row.unitPrice),
              line_position: idx + 1,
            })
          } else {
            await addLineMutation.mutateAsync({
              poId: order.id,
              product_name: row.name,
              quantity: String(row.quantity),
              unit_price: String(row.unitPrice),
              line_position: idx + 1,
            })
          }
        }
      }
      toast.success(t('einkauf.toast.orderUpdated', { orderNumber: order.po_number }))
      onClose()
    } catch {
      toast.error(t('einkauf.toast.orderUpdateFailed'))
    }
  }

  return (
    <Dialog open={!!order} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-lg max-h-[85vh] flex flex-col">
        <DialogHeader className="border-b border-border px-5 py-4">
          <div className="flex items-center gap-2">
            <Pencil className="h-5 w-5 text-primary" />
            <DialogTitle className="text-base font-semibold text-foreground">
              {t('einkauf.editOrder.title', { orderNumber: order?.po_number ?? '' })}
            </DialogTitle>
          </div>
          <DialogDescription className="sr-only">
            {t('einkauf.editOrder.title', { orderNumber: order?.po_number ?? '' })}
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          {!draft && (
            <p className="rounded-lg bg-secondary/50 px-3 py-2 text-xs text-muted-foreground">
              {t('einkauf.editOrder.lockedHint')}
            </p>
          )}

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('einkauf.newOrder.deliveryDate')}</label>
            <input
              type="date"
              value={deliveryDate}
              onChange={(e) => setDeliveryDate(e.target.value)}
              className={inputCls}
            />
          </div>

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('einkauf.newOrder.notes')}</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={2}
              className={`${inputCls} resize-none`}
            />
          </div>

          {draft && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium text-foreground">{t('einkauf.newOrder.positions')}</label>
                <button
                  onClick={() => setRows((prev) => [...(prev ?? []), { lineId: null, name: '', quantity: 1, unitPrice: 0 }])}
                  className="flex items-center gap-1 rounded-lg border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                >
                  <Plus className="h-3 w-3" />
                  {t('einkauf.newOrder.addPosition')}
                </button>
              </div>
              <div className="space-y-2">
                {(rows ?? []).map((row, idx) => (
                  <div key={row.lineId ?? `new-${idx}`} className="flex items-start gap-2 rounded-lg border border-border-muted p-2.5">
                    <div className="flex-1 space-y-2">
                      <input
                        type="text"
                        placeholder={t('einkauf.newOrder.itemNamePlaceholder')}
                        value={row.name}
                        onChange={(e) => updateRow(idx, { name: e.target.value })}
                        className={inputCls}
                      />
                      <div className="flex gap-2">
                        <div className="flex-1">
                          <label className="text-[10px] text-muted-foreground">{t('einkauf.newOrder.qty')}</label>
                          <input
                            type="number"
                            min={1}
                            value={row.quantity}
                            onChange={(e) => updateRow(idx, { quantity: Math.max(1, parseInt(e.target.value) || 1) })}
                            className={inputCls}
                          />
                        </div>
                        <div className="flex-1">
                          <label className="text-[10px] text-muted-foreground">
                            {t('einkauf.newOrder.unitPrice', { currency: order?.currency || 'EUR' })}
                          </label>
                          <input
                            type="number"
                            min={0}
                            step={0.01}
                            value={row.unitPrice}
                            onChange={(e) => updateRow(idx, { unitPrice: Math.max(0, parseFloat(e.target.value) || 0) })}
                            className={inputCls}
                          />
                        </div>
                      </div>
                    </div>
                    <button
                      onClick={() => removeRow(idx)}
                      className="mt-1 rounded p-1 text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
                      aria-label={t('common.delete')}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
              </div>
              <div className="flex items-center justify-between rounded-lg bg-secondary/50 px-3 py-2">
                <span className="text-sm font-medium text-muted-foreground">{t('einkauf.newOrder.total')}</span>
                <span className="text-base font-semibold text-foreground tabular-nums">
                  {formatCurrency(total, order?.currency || 'EUR')}
                </span>
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="border-t border-border px-5 py-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
            {t('common.save')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// EditSupplierDialog
// ---------------------------------------------------------------------------

interface EditSupplierDialogProps {
  supplier: Supplier | null
  onClose: () => void
}

export function EditSupplierDialog({ supplier, onClose }: EditSupplierDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState(supplier?.name ?? '')
  const [email, setEmail] = useState(supplier?.email ?? '')
  const [phone, setPhone] = useState(supplier?.phone ?? '')
  const [address, setAddress] = useState(supplier?.address ?? '')
  const [paymentTerms, setPaymentTerms] = useState(supplier?.payment_terms ?? PAYMENT_TERMS_OPTIONS[0])
  const [notes, setNotes] = useState(supplier?.notes ?? '')

  const updateSupplierMutation = useUpdateSupplier()

  const handleSave = () => {
    if (!supplier) return
    if (!name.trim()) {
      toast.error(t('einkauf.validation.supplierNameRequired'))
      return
    }
    updateSupplierMutation.mutate(
      {
        id: supplier.id,
        name: name.trim(),
        email: email.trim(),
        phone: phone.trim(),
        address: address.trim(),
        payment_terms: paymentTerms,
        notes: notes.trim(),
      },
      {
        onSuccess: () => {
          toast.success(t('einkauf.toast.supplierUpdated', { name: name.trim() }))
          onClose()
        },
        onError: () => toast.error(t('einkauf.toast.supplierUpdateFailed')),
      },
    )
  }

  return (
    <Dialog open={!!supplier} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-md max-h-[85vh] flex flex-col">
        <DialogHeader className="border-b border-border px-5 py-4">
          <div className="flex items-center gap-2">
            <Truck className="h-5 w-5 text-primary" />
            <DialogTitle className="text-base font-semibold text-foreground">
              {t('einkauf.editSupplier.title')}
            </DialogTitle>
          </div>
          <DialogDescription className="sr-only">{t('einkauf.editSupplier.title')}</DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('einkauf.newSupplier.name')} <span className="text-destructive">*</span>
            </label>
            <input type="text" value={name} onChange={(e) => setName(e.target.value)} className={inputCls} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('einkauf.newSupplier.email')}</label>
              <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} className={inputCls} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('einkauf.newSupplier.phone')}</label>
              <input type="tel" value={phone} onChange={(e) => setPhone(e.target.value)} className={inputCls} />
            </div>
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('einkauf.editSupplier.address')}</label>
            <input type="text" value={address} onChange={(e) => setAddress(e.target.value)} className={inputCls} />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('einkauf.newSupplier.paymentTerms')}</label>
            <select value={paymentTerms} onChange={(e) => setPaymentTerms(e.target.value)} className={inputCls}>
              {(PAYMENT_TERMS_OPTIONS.includes(paymentTerms) ? PAYMENT_TERMS_OPTIONS : [paymentTerms, ...PAYMENT_TERMS_OPTIONS]).map((pt) => (
                <option key={pt} value={pt}>{pt}</option>
              ))}
            </select>
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('einkauf.editSupplier.notes')}</label>
            <textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} className={`${inputCls} resize-none`} />
          </div>
        </div>

        <DialogFooter className="border-t border-border px-5 py-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={updateSupplierMutation.isPending}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
            {t('common.save')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// CartDialog — Einkaufs-Warenkorb
// ---------------------------------------------------------------------------

export interface CartEntry {
  item: CatalogItem
  qty: number
}

interface CartDialogProps {
  open: boolean
  entries: CartEntry[]
  suppliers: Supplier[]
  /** Count of existing POs — used for sequential order numbers. */
  poCount: number
  onClose: () => void
  onUpdateQty: (itemId: string, qty: number) => void
  onRemove: (itemId: string) => void
  /** Called after the orders were created (clear cart, switch tab). */
  onCreated: (orderCount: number) => void
}

export function CartDialog({
  open,
  entries,
  suppliers,
  poCount,
  onClose,
  onUpdateQty,
  onRemove,
  onCreated,
}: CartDialogProps) {
  const { t } = useTranslation()
  const poNumberPrefix = useEinkaufTenantStore((s) => s.poNumberPrefix)
  const createPOMutation = useCreatePO()
  const addLineMutation = useAddPOLine()
  const [creating, setCreating] = useState(false)

  // Sammelbestellung: group cart entries per supplier (weclapp/Xentral pattern)
  const groups = useMemo(() => {
    const bySupplier = new Map<string, CartEntry[]>()
    for (const e of entries) {
      const list = bySupplier.get(e.item.supplier_id) ?? []
      list.push(e)
      bySupplier.set(e.item.supplier_id, list)
    }
    return Array.from(bySupplier.entries()).map(([supplierId, items]) => ({
      supplierId,
      supplierName: suppliers.find((s) => s.id === supplierId)?.name ?? supplierId,
      items,
      total: items.reduce((s, e) => s + e.qty * parseFloat(e.item.price), 0),
    }))
  }, [entries, suppliers])

  const grandTotal = groups.reduce((s, g) => s + g.total, 0)

  const handleCreateOrders = async () => {
    if (entries.length === 0) return
    setCreating(true)
    try {
      let seq = poCount
      for (const group of groups) {
        seq += 1
        const nr = `${poNumberPrefix}-${new Date().getFullYear()}-${String(seq).padStart(3, '0')}`
        const po = await createPOMutation.mutateAsync({
          supplier_id: group.supplierId,
          po_number: nr,
          order_date: new Date().toISOString(),
          currency: group.items[0]?.item.currency || 'EUR',
        })
        for (const [idx, entry] of group.items.entries()) {
          await addLineMutation.mutateAsync({
            poId: po.id,
            product_name: entry.item.name,
            sku: entry.item.sku || undefined,
            quantity: String(entry.qty),
            unit_price: entry.item.price,
            line_position: idx + 1,
          })
        }
      }
      toast.success(t('einkauf.toast.cartOrdersCreated', { count: groups.length }))
      onCreated(groups.length)
    } catch {
      toast.error(t('einkauf.toast.orderCreateFailed'))
    } finally {
      setCreating(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-lg max-h-[85vh] flex flex-col">
        <DialogHeader className="border-b border-border px-5 py-4">
          <div className="flex items-center gap-2">
            <ShoppingCart className="h-5 w-5 text-primary" />
            <DialogTitle className="text-base font-semibold text-foreground">
              {t('einkauf.cart.title')}
            </DialogTitle>
          </div>
          <DialogDescription className="text-xs text-muted-foreground">
            {t('einkauf.cart.subtitle')}
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          {groups.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">{t('einkauf.cart.empty')}</p>
          ) : (
            groups.map((group) => (
              <div key={group.supplierId} className="rounded-lg border border-border overflow-hidden">
                <div className="flex items-center justify-between bg-secondary/30 px-3 py-2">
                  <div className="flex items-center gap-2">
                    <Truck className="h-3.5 w-3.5 text-primary" />
                    <span className="text-xs font-medium text-foreground">{group.supplierName}</span>
                  </div>
                  <span className="text-xs text-muted-foreground tabular-nums">
                    {formatCurrency(group.total, group.items[0]?.item.currency || 'EUR')}
                  </span>
                </div>
                <div className="divide-y divide-border-muted">
                  {group.items.map((entry) => {
                    const minQty = Math.max(1, parseFloat(entry.item.min_order_qty) || 1)
                    return (
                      <div key={entry.item.id} className="flex items-center gap-2 px-3 py-2.5">
                        <div className="flex-1 min-w-0">
                          <p className="text-sm text-foreground truncate">{entry.item.name}</p>
                          <p className="text-[10px] text-muted-foreground">
                            {formatCurrency(parseFloat(entry.item.price), entry.item.currency)}{' '}
                            {t('einkauf.catalog.perUnit', { unit: entry.item.unit })}
                            {minQty > 1 && ` · ${t('einkauf.cart.minQty', { qty: minQty })}`}
                          </p>
                        </div>
                        <input
                          type="number"
                          min={minQty}
                          value={entry.qty}
                          onChange={(e) =>
                            onUpdateQty(entry.item.id, Math.max(minQty, parseInt(e.target.value) || minQty))
                          }
                          className="w-20 rounded-lg border border-border bg-card px-2 py-1 text-sm text-foreground text-right tabular-nums focus:outline-none focus:ring-2 focus:ring-focus-ring"
                          aria-label={t('einkauf.newOrder.qty')}
                        />
                        <button
                          onClick={() => onRemove(entry.item.id)}
                          className="rounded p-1 text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
                          aria-label={t('common.delete')}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    )
                  })}
                </div>
              </div>
            ))
          )}

          {groups.length > 0 && (
            <div className="flex items-center justify-between rounded-lg bg-secondary/50 px-3 py-2">
              <span className="text-sm font-medium text-muted-foreground">
                {t('einkauf.cart.totalWithOrders', { count: groups.length })}
              </span>
              <span className="text-base font-semibold text-foreground tabular-nums">
                {formatCurrency(grandTotal, groups[0]?.items[0]?.item.currency || 'EUR')}
              </span>
            </div>
          )}
        </div>

        <DialogFooter className="border-t border-border px-5 py-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleCreateOrders}
            disabled={creating || groups.length === 0}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
            {t('einkauf.cart.createOrders', { count: groups.length })}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// NewCallDialog — Abruf aus Rahmenvertrag
// ---------------------------------------------------------------------------

interface NewCallDialogProps {
  contract: FrameworkContract | null
  onClose: () => void
}

export function NewCallDialog({ contract, onClose }: NewCallDialogProps) {
  const { t } = useTranslation()
  const [amount, setAmount] = useState(0)
  const [notes, setNotes] = useState('')
  const createCallMutation = useCreateContractCall()

  const remaining = contract
    ? Math.max(0, parseFloat(contract.total_value) - parseFloat(contract.used_value))
    : 0

  const handleSave = () => {
    if (!contract) return
    if (amount <= 0) {
      toast.error(t('einkauf.call.amountRequired'))
      return
    }
    if (amount > remaining) {
      toast.error(t('einkauf.call.amountExceedsRemaining', {
        remaining: formatCurrency(remaining, contract.currency),
      }))
      return
    }
    createCallMutation.mutate(
      {
        contractId: contract.id,
        amount: amount.toFixed(2),
        currency: contract.currency,
        notes: notes.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast.success(t('einkauf.toast.callCreated', { title: contract.title }))
          onClose()
        },
        onError: () => toast.error(t('einkauf.toast.callCreateFailed')),
      },
    )
  }

  return (
    <Dialog open={!!contract} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-md">
        <DialogHeader className="border-b border-border px-5 py-4">
          <div className="flex items-center gap-2">
            <ScrollText className="h-5 w-5 text-primary" />
            <div>
              <DialogTitle className="text-base font-semibold text-foreground">
                {t('einkauf.call.title')}
              </DialogTitle>
              <DialogDescription className="text-xs text-muted-foreground">
                {contract?.title} · {contract?.contract_nr}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="p-5 space-y-4">
          <div className="rounded-lg bg-secondary/50 px-3 py-2 text-xs text-muted-foreground">
            {t('einkauf.call.remainingHint', {
              remaining: formatCurrency(remaining, contract?.currency || 'EUR'),
            })}
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('einkauf.call.amount', { currency: contract?.currency || 'EUR' })}{' '}
              <span className="text-destructive">*</span>
            </label>
            <input
              type="number"
              min={0}
              step={0.01}
              value={amount || ''}
              onChange={(e) => setAmount(Math.max(0, parseFloat(e.target.value) || 0))}
              placeholder="0,00"
              className={inputCls}
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('einkauf.newOrder.notes')}</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={2}
              placeholder={t('einkauf.call.notesPlaceholder')}
              className={`${inputCls} resize-none`}
            />
          </div>
        </div>

        <DialogFooter className="border-t border-border px-5 py-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={createCallMutation.isPending}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
            {t('einkauf.call.save')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// WareneingangDialog — Wareneingang buchen (voll oder teilweise)
// ---------------------------------------------------------------------------

interface WareneingangDialogProps {
  order: PurchaseOrder | null
  supplierName: string
  onClose: () => void
}

export function WareneingangDialog({ order, supplierName, onClose }: WareneingangDialogProps) {
  const { t } = useTranslation()
  const { data: linesData } = usePOLines(order?.id ?? '')
  const lines = linesData?.lines ?? []

  const [receivedQtys, setReceivedQtys] = useState<Record<string, number>>({})
  const [partialDelivery, setPartialDelivery] = useState(false)
  const [bookToInventory, setBookToInventory] = useState(false)

  const receiveGoodsMutation = useReceiveGoods()
  const partialReceiveMutation = usePartialReceive()
  const saving = receiveGoodsMutation.isPending || partialReceiveMutation.isPending

  const handleSave = () => {
    if (!order) return
    const hasPartialEntries = lines.some((item) => (receivedQtys[item.id] ?? 0) > 0)

    if (!partialDelivery && !hasPartialEntries) {
      // Full receive via ReceiveGoods (simpler path — no per-line qtys specified)
      receiveGoodsMutation.mutate(order.id, {
        onSuccess: () => {
          const key = bookToInventory
            ? 'einkauf.toast.wareneingangWithInventory'
            : 'einkauf.toast.wareneingang'
          toast.success(t(key, { orderNumber: order.po_number }))
          onClose()
        },
        onError: () => toast.error(t('einkauf.toast.wareneingangFailed')),
      })
    } else {
      const items = lines
        .filter((item) => (receivedQtys[item.id] ?? 0) > 0)
        .map((item) => ({
          line_id: item.id,
          received_quantity: String((receivedQtys[item.id] ?? 0) + parseFloat(item.received_quantity)),
        }))
      if (items.length === 0) {
        toast.error(t('einkauf.validation.noQuantityEntered'))
        return
      }
      partialReceiveMutation.mutate(
        { poId: order.id, items },
        {
          onSuccess: () => {
            const key = bookToInventory
              ? 'einkauf.toast.wareneingangWithInventory'
              : 'einkauf.toast.wareneingang'
            toast.success(t(key, { orderNumber: order.po_number }))
            onClose()
          },
          onError: () => toast.error(t('einkauf.toast.wareneingangFailed')),
        },
      )
    }
  }

  return (
    <Dialog open={!!order} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-lg max-h-[80vh] flex flex-col">
        <DialogHeader className="border-b border-border px-5 py-4">
          <div className="flex items-center gap-2">
            <PackageCheck className="h-5 w-5 text-primary" />
            <div>
              <DialogTitle className="text-base font-semibold text-foreground">
                {t('einkauf.wareneingang.title')}
              </DialogTitle>
              <DialogDescription className="text-xs text-muted-foreground">
                {order?.po_number} — {supplierName}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          <div className="rounded-lg border border-border divide-y divide-border-muted">
            <div className="grid grid-cols-[1fr_80px_80px] gap-2 px-3 py-2 bg-secondary/30">
              <span className="text-[10px] font-medium text-muted-foreground uppercase">
                {t('einkauf.wareneingang.article')}
              </span>
              <span className="text-[10px] font-medium text-muted-foreground uppercase text-right">
                {t('einkauf.wareneingang.ordered')}
              </span>
              <span className="text-[10px] font-medium text-muted-foreground uppercase text-right">
                {t('einkauf.wareneingang.received')}
              </span>
            </div>
            {lines.length === 0 ? (
              <p className="px-3 py-4 text-xs text-muted-foreground text-center">
                {t('einkauf.detail.noPositions')}
              </p>
            ) : (
              lines.map((item) => {
                const itemQty = parseFloat(item.quantity)
                const itemReceived = parseFloat(item.received_quantity)
                return (
                  <div key={item.id} className="grid grid-cols-[1fr_80px_80px] gap-2 items-center px-3 py-2.5">
                    <div>
                      <p className="text-sm text-foreground">{item.product_name}</p>
                      <p className="text-[10px] text-muted-foreground">
                        {t('einkauf.wareneingang.alreadyReceived', { qty: itemReceived })}
                      </p>
                    </div>
                    <p className="text-sm text-muted-foreground text-right tabular-nums">{itemQty}</p>
                    <input
                      type="number"
                      min={0}
                      max={itemQty - itemReceived}
                      value={receivedQtys[item.id] ?? 0}
                      onChange={(e) =>
                        setReceivedQtys((prev) => ({
                          ...prev,
                          [item.id]: Math.max(0, parseInt(e.target.value) || 0),
                        }))
                      }
                      className="w-full rounded-lg border border-border bg-card px-2 py-1 text-sm text-foreground text-right tabular-nums focus:outline-none focus:ring-2 focus:ring-focus-ring"
                      aria-label={t('einkauf.wareneingang.received')}
                    />
                  </div>
                )
              })
            )}
          </div>

          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={partialDelivery}
              onChange={(e) => setPartialDelivery(e.target.checked)}
              className="h-4 w-4 rounded border-border text-primary focus:ring-focus-ring"
            />
            <span className="text-sm text-foreground">{t('einkauf.wareneingang.partialDelivery')}</span>
          </label>

          <label className="flex items-start gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={bookToInventory}
              onChange={(e) => setBookToInventory(e.target.checked)}
              className="h-4 w-4 mt-0.5 rounded border-border text-primary focus:ring-focus-ring"
            />
            <div>
              <span className="text-sm text-foreground">{t('einkauf.wareneingang.bookToInventory')}</span>
              {bookToInventory && (
                <p className="text-xs text-muted-foreground mt-0.5">
                  {t('einkauf.wareneingang.bookToInventoryHint')}
                </p>
              )}
            </div>
          </label>
        </div>

        <DialogFooter className="border-t border-border px-5 py-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
            {t('einkauf.wareneingang.book')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// ContractCallsList — bisherige Abrufe im expandierten Rahmenvertrag
// ---------------------------------------------------------------------------

export function ContractCallsList({ contractId, currency }: { contractId: string; currency: string }) {
  const { t } = useTranslation()
  const { data } = useContractCalls(contractId)
  const calls = data?.calls ?? []
  if (calls.length === 0) return null

  return (
    <div className="border-t border-border-muted px-4 py-3">
      <h5 className="text-[10px] font-medium text-muted-foreground uppercase mb-2">
        {t('einkauf.contract.callsTitle', { count: calls.length })}
      </h5>
      <div className="space-y-1">
        {calls.map((call) => (
          <div key={call.id} className="flex items-center justify-between text-xs">
            <span className="text-muted-foreground">
              {formatDate(call.called_at)}
              {call.notes ? ` · ${call.notes}` : ''}
            </span>
            <span className="text-foreground font-medium tabular-nums">
              {formatCurrency(parseFloat(call.amount), call.currency || currency)}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
