import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { AlertTriangle, Hash, Link2, Paperclip, ShoppingCart, Warehouse, X, Download } from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import {
  useInventarMovements,
  useItemAttachments,
  useUploadItemAttachment,
  useDeleteItemAttachment,
} from '@/api/hooks/useInventar'
import { useAvatarSrc } from '@/api/hooks/useAvatarSrc'
import type { InventarItem, InventoryLocation, ItemAttachment } from '@/api/inventar-types'
import { formatCurrency, formatDate, formatDateTime } from '@/lib/format'
import { useInventarPrefsStore } from '@/stores/inventarPrefs'
import {
  getStockStatus,
  getStockStatusDisplay,
  stockStatusBadgeClass,
  movementTypeKeys,
  movementTypeColors,
  movementTypeIcons,
  locationTypeIcons,
} from './inventar-shared'
import { buildMovementsCsv, downloadCsv, csvDateStamp } from './inventar-export'
import { useHasCapability, useModuleVisible } from '@/hooks/useCapability'

// ─── Stock Bar Visual ─────────────────────────────────────────────
function StockBar({ current, min }: { current: number; min: number }) {
  const { t } = useTranslation()
  const max = Math.max(min * 3, current * 1.2, 1)
  const pct = Math.min((current / max) * 100, 100)
  const minPct = Math.min((min / max) * 100, 100)
  const status = current <= min ? 'bg-error' : current < min * 2 ? 'bg-warning' : 'bg-success'

  return (
    <div className="space-y-1">
      <div className="relative h-2 w-full rounded-full bg-secondary overflow-hidden">
        <div
          className={`absolute inset-y-0 left-0 rounded-full ${status} transition-all`}
          style={{ width: `${pct}%` }}
        />
        {/* Min threshold marker */}
        <div
          className="absolute inset-y-0 w-0.5 bg-foreground/30"
          style={{ left: `${minPct}%` }}
          title={t('inventar.stockBar.minThreshold', { min })}
        />
      </div>
      <div className="flex justify-between text-[10px] text-muted-foreground">
        <span>0</span>
        <span>{t('inventar.detail.stockBarMin', { min, unit: '' }).trim()}</span>
        <span>{Math.round(max)}</span>
      </div>
    </div>
  )
}

// ─── Item Attachments ─────────────────────────────────────────────
// Browser-direct upload (scope=inventar) + register against the item.

function ItemAttachmentsSection({ itemId }: { itemId: string }) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)
  const attachmentsQuery = useItemAttachments(itemId)
  const uploadMut = useUploadItemAttachment()
  const deleteMut = useDeleteItemAttachment()
  const attachments = attachmentsQuery.data?.attachments ?? []
  const canManageAttachments = useHasCapability('inventar:attachment:manage')

  const handleFile = (file: File | undefined) => {
    if (!file) return
    uploadMut.mutate(
      { itemId, file },
      { onError: () => toast.error(t('inventar.detail.attachmentUploadError')) },
    )
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          {t('inventar.detail.attachments')}
        </h4>
        <input
          ref={fileRef}
          type="file"
          className="hidden"
          onChange={(e) => {
            handleFile(e.target.files?.[0])
            e.target.value = ''
          }}
        />
        {canManageAttachments && (
          <button
            type="button"
            onClick={() => fileRef.current?.click()}
            disabled={uploadMut.isPending}
            className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50"
          >
            <Paperclip className="h-3 w-3" />
            {t('inventar.detail.attachmentAdd')}
          </button>
        )}
      </div>
      {attachments.length === 0 ? (
        <p className="text-xs text-muted-foreground py-2">{t('inventar.detail.attachmentsEmpty')}</p>
      ) : (
        <div className="space-y-1">
          {attachments.map((att) => (
            <ItemAttachmentRow
              key={att.id}
              attachment={att}
              onRemove={canManageAttachments ? () => deleteMut.mutate({ id: att.id, itemId }) : undefined}
              removeLabel={t('inventar.detail.attachmentRemove')}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function ItemAttachmentRow({
  attachment,
  onRemove,
  removeLabel,
}: {
  attachment: ItemAttachment
  /** Absent when the role may not manage attachments (delete hidden). */
  onRemove?: () => void
  removeLabel: string
}) {
  const src = useAvatarSrc(attachment.object_key)
  return (
    <div className="flex items-center gap-2 rounded-md border border-border-muted p-2">
      <Paperclip className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
      {src ? (
        <a
          href={src}
          target="_blank"
          rel="noopener noreferrer"
          className="flex-1 min-w-0 truncate text-xs text-primary hover:underline"
        >
          {attachment.name}
        </a>
      ) : (
        <span className="flex-1 min-w-0 truncate text-xs text-foreground">{attachment.name}</span>
      )}
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          aria-label={removeLabel}
          title={removeLabel}
          className="shrink-0 rounded p-0.5 text-muted-foreground hover:text-error transition-colors"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  )
}

// ─── Item Detail Modal ────────────────────────────────────────────

/**
 * ItemDetailModal — zentriertes Cosmi-Detailfenster für einen Inventar-Artikel
 * (ersetzt das frühere Slide-over-DetailPanel). Meta-Grid, Bestandsbalken,
 * Chargen/Seriennummern, Bewegungs-Historie (mit CSV-Export), Anhänge und
 * Footer-Aktionen (Bearbeiten / Bestandsbewegung).
 */
export function ItemDetailModal({
  item,
  locations,
  onClose,
  onEdit,
  onMovement,
  onBack,
}: {
  item: InventarItem | null
  locations: InventoryLocation[]
  onClose: () => void
  onEdit: (item: InventarItem) => void
  onMovement: (item: InventarItem) => void
  /** Set when opened from the location modal (back-navigation chain). */
  onBack?: () => void
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const showMinStockWarnings = useInventarPrefsStore((s) => s.showMinStockWarnings)
  const movementsQuery = useInventarMovements(item?.id ?? '', { page: 1, page_size: 5 })

  // RBAC: footer actions + export follow the page-level gating; the einkauf
  // shortcuts only render when the role can see that module at all.
  const canEditItem = useHasCapability('inventar:item:edit')
  const canRecordMovement = useHasCapability('inventar:movement:create')
  const canExport = useHasCapability('inventar:export:run')
  const einkaufVisible = useModuleVisible('einkauf')

  const handleExportMovements = () => {
    if (!item) return
    const movements = movementsQuery.data?.movements ?? []
    downloadCsv(
      buildMovementsCsv(movements, item.name),
      `inventar-bewegungen-${item.sku}-${csvDateStamp()}.csv`,
    )
    toast.success(t('inventar.export.movementsSuccess', { name: item.name }))
  }

  return (
    <DetailModal
      open={!!item}
      onClose={onClose}
      onBack={onBack}
      title={item?.name}
      subtitle={item ? `${item.sku}${item.category ? ` · ${item.category}` : ''}` : undefined}
      badge={
        item ? (
          <span
            className={`ml-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${stockStatusBadgeClass(item)}`}
          >
            {t(getStockStatusDisplay(item).labelKey)}
          </span>
        ) : undefined
      }
      footer={
        item && (canEditItem || canRecordMovement) ? (
          <div className="flex gap-2">
            {canEditItem && (
              <button
                onClick={() => onEdit(item)}
                className="flex-1 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('common.edit')}
              </button>
            )}
            {canRecordMovement && (
              <button
                onClick={() => onMovement(item)}
                className="flex-1 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                {t('inventar.detail.buttonMovement')}
              </button>
            )}
          </div>
        ) : undefined
      }
    >
      {item && (
        <div className="space-y-5">
          {/* Nachbestell-Banner (critical stock) */}
          {showMinStockWarnings && getStockStatus(item) === 'critical' && (
            <div className="rounded-lg border border-error/30 bg-error-light p-3">
              <div className="flex items-start gap-2">
                <AlertTriangle className="h-4 w-4 text-error mt-0.5 shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-error">
                    {t('inventar.detail.criticalBanner.title')}
                  </p>
                  <p className="text-xs text-error/80 mt-0.5">
                    {t('inventar.detail.criticalBanner.desc', {
                      current: item.quantity,
                      min: item.min_quantity,
                    })}
                  </p>
                </div>
              </div>
              {einkaufVisible && (
                <button
                  onClick={() => {
                    onClose()
                    navigate('/einkauf')
                  }}
                  className="mt-2 w-full flex items-center justify-center gap-2 rounded-lg bg-error px-3 py-2 text-sm text-white hover:bg-error/90 transition-colors"
                >
                  <ShoppingCart className="h-4 w-4" />
                  {t('inventar.detail.criticalBanner.button')}
                </button>
              )}
            </div>
          )}

          {/* Basic info */}
          <div className="space-y-3">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t('inventar.detail.details')}
            </h4>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
              <div>
                <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldName')}</p>
                <p className="text-sm text-foreground font-medium">{item.name}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">SKU</p>
                <p className="text-sm text-foreground font-mono">{item.sku}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldCategory')}</p>
                <p className="text-sm text-foreground">{item.category ?? '—'}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldUnit')}</p>
                <p className="text-sm text-foreground">{item.unit}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldPrice')}</p>
                <p className="text-sm text-foreground">
                  {formatCurrency(item.price ?? 0, item.currency ?? 'EUR')}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldLastMovement')}</p>
                <p className="text-sm text-foreground">{formatDate(item.updated_at)}</p>
              </div>
            </div>
            {item.barcode && (
              <div>
                <p className="text-xs text-muted-foreground">{t('inventar.detail.fieldBarcode')}</p>
                <p className="text-sm text-foreground font-mono">{item.barcode}</p>
              </div>
            )}
          </div>

          {/* Stock visual bar */}
          <div className="space-y-2">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t('inventar.detail.stockBar.title')}
            </h4>
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <div className="flex items-baseline justify-between mb-2">
                <span className="text-2xl font-semibold text-foreground tabular-nums">
                  {item.quantity}
                </span>
                <span className="text-sm text-muted-foreground">
                  Min: {item.min_quantity} {item.unit}
                </span>
              </div>
              <StockBar current={Number(item.quantity)} min={Number(item.min_quantity)} />
            </div>
          </div>

          {/* Chargen & Seriennummern */}
          {(item.batch_number || (item.serial_numbers && item.serial_numbers.length > 0)) && (
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                {t('inventar.detail.chargenTitle')}
              </h4>
              <div className="rounded-lg border border-border p-3 space-y-2">
                {item.batch_number && (
                  <div className="flex items-center gap-2">
                    <Hash className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="text-xs text-muted-foreground">{t('inventar.detail.chargeLabel')}</span>
                    <span className="text-sm text-foreground font-mono">{item.batch_number}</span>
                  </div>
                )}
                {item.serial_numbers && item.serial_numbers.length > 0 && (
                  <div>
                    <div className="flex items-center gap-2 mb-1.5">
                      <Hash className="h-3.5 w-3.5 text-muted-foreground" />
                      <span className="text-xs text-muted-foreground">{t('inventar.detail.serialLabel')}</span>
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {item.serial_numbers.map((sn) => (
                        <span
                          key={sn}
                          className="inline-flex items-center rounded-md bg-secondary px-2 py-0.5 text-xs font-mono text-foreground"
                        >
                          {sn}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Belegkette / Einkauf-Anbindung */}
          {item.linked_purchase_order && (
            <div className="space-y-2">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                {t('inventar.detail.einkaufTitle')}
              </h4>
              <div className="rounded-lg border border-border p-3">
                <div className="flex items-center gap-2 mb-2">
                  <Link2 className="h-4 w-4 text-primary" />
                  <span className="text-sm text-foreground">
                    {t('inventar.detail.linkedOrder')}{' '}
                    <span className="font-mono font-medium">{item.linked_purchase_order}</span>
                  </span>
                </div>
                {einkaufVisible && (
                  <button
                    onClick={() => {
                      onClose()
                      navigate('/einkauf')
                    }}
                    className="w-full flex items-center justify-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                  >
                    <Link2 className="h-3.5 w-3.5" />
                    {t('inventar.detail.toOrder')}
                  </button>
                )}
              </div>
            </div>
          )}

          {/* Lagerort */}
          <div className="space-y-2">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t('inventar.detail.lagerortTitle')}
            </h4>
            <div className="rounded-lg border border-border p-3">
              <div className="flex items-center gap-2">
                {(() => {
                  const loc = locations.find((l) => l.id === item.location_id)
                  const LIcon = loc ? locationTypeIcons[loc.type] || Warehouse : Warehouse
                  return (
                    <>
                      <LIcon className="h-4 w-4 text-primary" />
                      <div className="flex-1">
                        <p className="text-sm font-medium text-foreground">{item.location ?? '—'}</p>
                        {loc && <p className="text-xs text-muted-foreground">{loc.address}</p>}
                      </div>
                      <span className="text-sm tabular-nums text-foreground font-medium">
                        {item.quantity} {item.unit}
                      </span>
                    </>
                  )
                })()}
              </div>
            </div>
          </div>

          {/* Letzte Bewegungen */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                {t('inventar.detail.lastMovements')}
              </h4>
              {canExport && (movementsQuery.data?.movements?.length ?? 0) > 0 && (
                <button
                  type="button"
                  onClick={handleExportMovements}
                  className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                >
                  <Download className="h-3 w-3" />
                  {t('inventar.export.movementsButton')}
                </button>
              )}
            </div>
            {(() => {
              const itemMovements = movementsQuery.data?.movements?.slice(0, 5) ?? []
              if (itemMovements.length === 0) {
                return (
                  <p className="text-xs text-muted-foreground py-2">{t('inventar.detail.noMovements')}</p>
                )
              }
              return (
                <div className="space-y-1">
                  {itemMovements.map((mov) => {
                    const MIcon = movementTypeIcons[mov.movement_type]
                    return (
                      <div key={mov.id} className="flex items-center gap-2 rounded-md border border-border-muted p-2">
                        <span className={`flex h-6 w-6 items-center justify-center rounded-md ${movementTypeColors[mov.movement_type]}`}>
                          {MIcon && <MIcon className="h-3 w-3" />}
                        </span>
                        <div className="flex-1 min-w-0">
                          <p className="text-xs font-medium text-foreground">
                            {movementTypeKeys[mov.movement_type] ? t(movementTypeKeys[mov.movement_type]) : mov.movement_type}
                            {Number(mov.quantity) > 0 ? ` +${mov.quantity}` : ` ${mov.quantity}`}
                          </p>
                          <p className="text-[10px] text-muted-foreground truncate">
                            {mov.reference ?? '—'} · {mov.performed_by ?? '—'}
                          </p>
                        </div>
                        <span className="text-[10px] text-muted-foreground whitespace-nowrap">
                          {formatDateTime(mov.created_at)}
                        </span>
                      </div>
                    )
                  })}
                </div>
              )
            })()}
          </div>

          {/* Anhänge */}
          <ItemAttachmentsSection itemId={item.id} />
        </div>
      )}
    </DetailModal>
  )
}
