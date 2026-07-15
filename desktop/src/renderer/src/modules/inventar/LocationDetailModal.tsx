import { useTranslation } from 'react-i18next'
import { Download, MapPin, Package, Warehouse } from 'lucide-react'
import { toast } from 'sonner'
import { DetailModal } from '@/components/shared'
import type { InventarItem, InventoryLocation } from '@/api/inventar-types'
import {
  getStockStatus,
  getStockStatusDisplay,
  locationTypeIcons,
  locationTypeKeys,
} from './inventar-shared'
import { buildItemsCsv, downloadCsv, csvDateStamp } from './inventar-export'

/**
 * LocationDetailModal — zentriertes Cosmi-Detailfenster für einen Lagerort.
 * Meta (Typ, Adresse), Kennzahlen und die klickbare Artikel-Liste dieses
 * Lagerorts (Klick → Artikel-Modal via onBack-Kette) + Bestandsliste-Export.
 */
export function LocationDetailModal({
  location,
  items,
  onClose,
  onItemClick,
}: {
  location: InventoryLocation | null
  /** Items stored at this location. */
  items: InventarItem[]
  onClose: () => void
  /** Opens the item detail modal (caller wires the back-chain). */
  onItemClick: (item: InventarItem) => void
}) {
  const { t } = useTranslation()
  const criticalCount = items.filter((i) => getStockStatus(i) === 'critical').length
  const LIcon = location ? locationTypeIcons[location.type] || Warehouse : Warehouse

  const handleExport = () => {
    if (!location) return
    downloadCsv(
      buildItemsCsv(items),
      `inventar-bestand-${location.name.replace(/\s+/g, '-').toLowerCase()}-${csvDateStamp()}.csv`,
    )
    toast.success(t('inventar.export.itemsSuccess', { count: items.length }))
  }

  return (
    <DetailModal
      open={!!location}
      onClose={onClose}
      title={location?.name}
      subtitle={location?.address}
      badge={
        location ? (
          <span className="ml-2 inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-xs font-medium text-muted-foreground">
            {t(locationTypeKeys[location.type])}
          </span>
        ) : undefined
      }
      footer={
        location && items.length > 0 ? (
          <button
            onClick={handleExport}
            className="flex w-full items-center justify-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          >
            <Download className="h-4 w-4" />
            {t('inventar.lagerortDetail.exportButton')}
          </button>
        ) : undefined
      }
    >
      {location && (
        <div className="space-y-5">
          {/* Kennzahlen */}
          <div className="grid grid-cols-3 gap-3">
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <p className="text-xs text-muted-foreground">{t('inventar.lagerortDetail.statItems')}</p>
              <p className="text-xl font-semibold text-foreground tabular-nums">{items.length}</p>
            </div>
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <p className="text-xs text-muted-foreground">{t('inventar.lagerortDetail.statCritical')}</p>
              <p className={`text-xl font-semibold tabular-nums ${criticalCount > 0 ? 'text-error' : 'text-foreground'}`}>
                {criticalCount}
              </p>
            </div>
            <div className="rounded-lg border border-border bg-secondary/30 p-3">
              <p className="text-xs text-muted-foreground">{t('inventar.lagerortDetail.statType')}</p>
              <p className="flex items-center gap-1.5 text-sm font-medium text-foreground pt-1">
                <LIcon className="h-4 w-4 text-primary" />
                {t(locationTypeKeys[location.type])}
              </p>
            </div>
          </div>

          {/* Adresse */}
          <div className="flex items-center gap-2 rounded-lg border border-border p-3">
            <MapPin className="h-4 w-4 text-primary shrink-0" />
            <p className="text-sm text-foreground">{location.address}</p>
          </div>

          {/* Artikel-Liste */}
          <div className="space-y-2">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t('inventar.lagerortDetail.itemsTitle')}
            </h4>
            {items.length === 0 ? (
              <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed border-border py-6">
                <Package className="h-5 w-5 text-muted-foreground" />
                <p className="text-xs text-muted-foreground">{t('inventar.lagerortDetail.emptyItems')}</p>
              </div>
            ) : (
              <div className="space-y-1">
                {items.map((item) => {
                  const st = getStockStatusDisplay(item)
                  return (
                    <div
                      key={item.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => onItemClick(item)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          onItemClick(item)
                        }
                      }}
                      className="flex cursor-pointer items-center gap-3 rounded-md border border-border-muted p-2.5 hover:bg-secondary/50 transition-colors focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    >
                      <span className={`h-2 w-2 shrink-0 rounded-full ${st.dotColor}`} />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium text-foreground">{item.name}</p>
                        <p className="truncate text-xs text-muted-foreground font-mono">{item.sku}</p>
                      </div>
                      <span className="shrink-0 text-sm tabular-nums text-foreground">
                        {item.quantity} {item.unit}
                      </span>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      )}
    </DetailModal>
  )
}
