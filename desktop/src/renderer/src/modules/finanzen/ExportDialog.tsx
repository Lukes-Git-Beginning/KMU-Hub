import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Download, FileSpreadsheet, Info } from 'lucide-react'
import { toast } from 'sonner'
import { useExportDATEV } from '@/api/hooks/useFinance'

interface ExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ExportDialog({ open, onOpenChange }: ExportDialogProps) {
  const now = new Date()
  const year = now.getFullYear()
  const month = now.getMonth()
  const firstOfMonth = new Date(year, month, 1).toISOString().split('T')[0]
  const lastOfMonth = new Date(year, month + 1, 0).toISOString().split('T')[0]

  const [startDate, setStartDate] = useState(firstOfMonth)
  const [endDate, setEndDate] = useState(lastOfMonth)

  const exportDATEV = useExportDATEV()

  const handleExport = () => {
    if (!startDate || !endDate) return
    exportDATEV.mutate(
      { date_from: startDate, date_to: endDate },
      {
        onSuccess: () => {
          toast.success(
            `DATEV Export heruntergeladen: EXTF_Buchungsstapel_${startDate}_${endDate}.csv`,
          )
          onOpenChange(false)
        },
        onError: (err) => toast.error(err.message),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            DATEV Export
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Format info */}
          <div className="flex items-start gap-3 rounded-lg border border-border p-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 shrink-0">
              <FileSpreadsheet className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">
                DATEV Buchungsstapel
              </p>
              <p className="text-xs text-muted-foreground">
                EXTF-Format (CSV) fuer den Import in DATEV oder Steuerberater-Software
              </p>
            </div>
          </div>

          {/* Date range */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Von</Label>
              <Input
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Bis</Label>
              <Input
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
              />
            </div>
          </div>

          {/* Info */}
          <div className="flex items-start gap-2 rounded-lg border border-info/30 bg-info/5 p-3 text-xs text-info">
            <Info className="h-4 w-4 mt-0.5 shrink-0" />
            <span>
              Der Export umfasst alle Rechnungen und Gutschriften im
              gewaehlten Zeitraum. Dateiname:{' '}
              <span className="font-mono">
                EXTF_Buchungsstapel_{startDate}_{endDate}.csv
              </span>
            </span>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            onClick={handleExport}
            disabled={!startDate || !endDate || exportDATEV.isPending}
          >
            <Download className="mr-1.5 h-4 w-4" />
            {exportDATEV.isPending ? 'Exportiert...' : 'Exportieren'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
