/**
 * Contact export dialog with field selection and format picker.
 *
 * Allows users to choose which fields to include and the export format
 * (CSV or vCard). Triggers a browser file download on export.
 */
import { useState } from 'react'
import { Download, Loader2, AlertCircle } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import {
  useExportContactsCSV,
  useExportContactsVCard,
} from '@/api/hooks/contacts-import'
import type { ExportFormat } from '@/api/crm-types'
import { EXPORT_FIELDS } from '@/api/crm-types'

interface ExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** IDs of contacts to export. If empty, exports all visible contacts. */
  contactIds: string[]
}

export default function ExportDialog({
  open,
  onOpenChange,
  contactIds,
}: ExportDialogProps) {
  const [format, setFormat] = useState<ExportFormat>('csv')
  const [selectedFields, setSelectedFields] = useState<Set<string>>(
    () => new Set(EXPORT_FIELDS.map((f) => f.key)),
  )

  const exportCSV = useExportContactsCSV()
  const exportVCard = useExportContactsVCard()

  const isExporting = exportCSV.isPending || exportVCard.isPending
  const isError = exportCSV.isError || exportVCard.isError

  function toggleField(key: string) {
    setSelectedFields((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  function toggleAll() {
    if (selectedFields.size === EXPORT_FIELDS.length) {
      setSelectedFields(new Set())
    } else {
      setSelectedFields(new Set(EXPORT_FIELDS.map((f) => f.key)))
    }
  }

  async function handleExport() {
    if (format === 'csv') {
      await exportCSV.mutateAsync({
        contactIds,
        fields: Array.from(selectedFields),
      })
    } else {
      await exportVCard.mutateAsync({ contactIds })
    }
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Kontakte exportieren</DialogTitle>
          <DialogDescription>
            {contactIds.length > 0
              ? `${contactIds.length} Kontakt${contactIds.length !== 1 ? 'e' : ''} exportieren`
              : 'Alle sichtbaren Kontakte exportieren'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Format selection */}
          <div className="space-y-3">
            <Label className="text-sm font-medium">Format</Label>
            <div className="flex gap-3">
              <button
                type="button"
                className={`flex-1 p-3 rounded-lg border-2 text-center transition-colors ${
                  format === 'csv'
                    ? 'border-primary bg-primary/5'
                    : 'border-muted hover:border-primary/30'
                }`}
                onClick={() => setFormat('csv')}
              >
                <p className="text-sm font-medium">CSV</p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Für Excel, Google Sheets
                </p>
              </button>
              <button
                type="button"
                className={`flex-1 p-3 rounded-lg border-2 text-center transition-colors ${
                  format === 'vcard'
                    ? 'border-primary bg-primary/5'
                    : 'border-muted hover:border-primary/30'
                }`}
                onClick={() => setFormat('vcard')}
              >
                <p className="text-sm font-medium">vCard</p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Für Outlook, Apple Kontakte
                </p>
              </button>
            </div>
          </div>

          {/* Field selection (CSV only) */}
          {format === 'csv' && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <Label className="text-sm font-medium">Felder</Label>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-xs h-auto py-1"
                  onClick={toggleAll}
                >
                  {selectedFields.size === EXPORT_FIELDS.length
                    ? 'Keine auswählen'
                    : 'Alle auswählen'}
                </Button>
              </div>
              <div className="space-y-2">
                {EXPORT_FIELDS.map((field) => (
                  <div key={field.key} className="flex items-center gap-3">
                    <Checkbox
                      id={`export-${field.key}`}
                      checked={selectedFields.has(field.key)}
                      onCheckedChange={() => toggleField(field.key)}
                    />
                    <Label
                      htmlFor={`export-${field.key}`}
                      className="text-sm cursor-pointer"
                    >
                      {field.label}
                    </Label>
                  </div>
                ))}
              </div>
            </div>
          )}

          {isError && (
            <div className="flex items-center gap-2 text-sm text-destructive">
              <AlertCircle className="h-4 w-4" />
              Export fehlgeschlagen. Bitte versuchen Sie es erneut.
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={handleExport}
            disabled={isExporting || (format === 'csv' && selectedFields.size === 0)}
          >
            {isExporting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Exportieren...
              </>
            ) : (
              <>
                <Download className="mr-2 h-4 w-4" />
                Exportieren
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
