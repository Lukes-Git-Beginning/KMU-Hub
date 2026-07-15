import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ClipboardCheck } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { useCreateInventurSession } from '@/api/hooks/useInventar'
import type { InventarItem, InventoryLocation } from '@/api/inventar-types'

/**
 * NewInventurDialog — legt eine Inventur-Session an (Zähl-Workflow-Start).
 *
 * Markt-Muster (weclapp/Zoho/myfactory): Name + Stichtag + Lagerort-Auswahl
 * (Teilinventur via Scope) + Eröffnungsmethode „nur Artikel mit Bestand".
 * Die Zählliste (counts) wird aus dem gewählten Scope aufgebaut.
 */
export function NewInventurDialog({
  open,
  onClose,
  items,
  locations,
}: {
  open: boolean
  onClose: () => void
  items: InventarItem[]
  locations: InventoryLocation[]
}) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [date, setDate] = useState(() => new Date().toISOString().slice(0, 10))
  const [locationId, setLocationId] = useState<string>('all')
  const [onlyWithStock, setOnlyWithStock] = useState(false)

  const createMutation = useCreateInventurSession()

  const scopedItems = useMemo(() => {
    let result = items
    if (locationId !== 'all') {
      const loc = locations.find((l) => l.id === locationId)
      result = result.filter((i) => i.location_id === locationId || (loc && i.location === loc.name))
    }
    if (onlyWithStock) {
      result = result.filter((i) => Number(i.quantity) > 0)
    }
    return result
  }, [items, locations, locationId, onlyWithStock])

  const resetAndClose = () => {
    setName('')
    setDate(new Date().toISOString().slice(0, 10))
    setLocationId('all')
    setOnlyWithStock(false)
    onClose()
  }

  const handleCreate = () => {
    if (!name.trim()) {
      toast.error(t('inventar.inventur.dialog.errorName'))
      return
    }
    if (scopedItems.length === 0) {
      toast.error(t('inventar.inventur.dialog.errorNoItems'))
      return
    }
    createMutation.mutate(
      {
        name: name.trim(),
        date,
        location_id: locationId === 'all' ? undefined : locationId,
        item_ids: scopedItems.map((i) => i.id),
      },
      {
        onSuccess: () => {
          toast.success(t('inventar.inventur.dialog.createSuccess', { name: name.trim() }))
          resetAndClose()
        },
        onError: () => toast.error(t('common.error')),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) resetAndClose() }}>
      <DialogContent className="gap-0 p-0 max-w-md">
        <div className="p-6">
          <DialogHeader className="mb-5">
            <div className="flex items-center gap-2">
              <ClipboardCheck className="h-5 w-5 text-primary" />
              <DialogTitle className="text-lg font-semibold text-foreground">
                {t('inventar.inventur.dialog.title')}
              </DialogTitle>
            </div>
            <DialogDescription className="text-xs text-muted-foreground mt-0.5">
              {t('inventar.inventur.dialog.subtitle')}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {/* Name */}
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('inventar.inventur.dialog.labelName')} <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t('inventar.inventur.dialog.placeholderName')}
                autoFocus
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>

            {/* Stichtag + Lagerort */}
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  {t('inventar.inventur.dialog.labelDate')}
                </label>
                <input
                  type="date"
                  value={date}
                  onChange={(e) => setDate(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  {t('inventar.inventur.dialog.labelLocation')}
                </label>
                <select
                  value={locationId}
                  onChange={(e) => setLocationId(e.target.value)}
                  className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <option value="all">{t('inventar.inventur.dialog.allLocations')}</option>
                  {locations.map((loc) => (
                    <option key={loc.id} value={loc.id}>{loc.name}</option>
                  ))}
                </select>
              </div>
            </div>

            {/* Nur Artikel mit Bestand */}
            <label className="flex cursor-pointer items-center gap-2">
              <input
                type="checkbox"
                checked={onlyWithStock}
                onChange={(e) => setOnlyWithStock(e.target.checked)}
                className="h-4 w-4 rounded border-border accent-[var(--primary)]"
              />
              <span className="text-sm text-foreground">{t('inventar.inventur.dialog.onlyWithStock')}</span>
            </label>

            {/* Zählliste-Vorschau */}
            <div className="rounded-lg border border-border bg-secondary/30 px-3 py-2">
              <p className="text-xs text-muted-foreground">
                {t('inventar.inventur.dialog.scopePreview', { count: scopedItems.length })}
              </p>
            </div>
          </div>

          <DialogFooter className="mt-6">
            <button
              onClick={resetAndClose}
              className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleCreate}
              disabled={createMutation.isPending}
              className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
            >
              {t('inventar.inventur.dialog.buttonCreate')}
            </button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}
