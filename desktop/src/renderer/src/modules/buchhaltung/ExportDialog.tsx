import { useState } from 'react'
import { useTranslation } from 'react-i18next'
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
import { Download, FileText, Table2, FileSpreadsheet } from 'lucide-react'
import { toast } from 'sonner'

interface ExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const FORMATS = [
  { id: 'csv', label: 'CSV', icon: Table2, descKey: 'buchhaltung.exportFormat.csvDesc' },
  { id: 'pdf', label: 'PDF', icon: FileText, descKey: 'buchhaltung.exportFormat.pdfDesc' },
  { id: 'datev', label: 'DATEV', icon: FileSpreadsheet, descKey: 'buchhaltung.exportFormat.datevDesc' },
]

export function ExportDialog({ open, onOpenChange }: ExportDialogProps) {
  const { t } = useTranslation()
  const [format, setFormat] = useState('csv')
  const [startDate, setStartDate] = useState('2026-01-01')
  const [endDate, setEndDate] = useState('2026-02-28')

  const handleExport = () => {
    const formatLabel = FORMATS.find((f) => f.id === format)?.label ?? format
    toast.success(t('buchhaltung.toast.exportCreating', { format: formatLabel }))
    setTimeout(() => {
      toast.success(t('buchhaltung.toast.exportDownloaded', { name: `${formatLabel}_${startDate}_${endDate}` }))
    }, 1500)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            {t('buchhaltung.exportDialog.title')}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Date range */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('buchhaltung.exportDialog.from')}</Label>
              <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('buchhaltung.exportDialog.to')}</Label>
              <Input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
            </div>
          </div>

          {/* Format selection */}
          <div className="space-y-1.5">
            <Label>{t('buchhaltung.exportDialog.format')}</Label>
            <div className="space-y-2">
              {FORMATS.map((f) => {
                const Icon = f.icon
                const active = format === f.id
                return (
                  <button
                    key={f.id}
                    onClick={() => setFormat(f.id)}
                    className={`flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors ${
                      active ? 'border-primary bg-primary/5' : 'border-border hover:bg-secondary/50'
                    }`}
                  >
                    <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${active ? 'bg-primary/10' : 'bg-secondary'}`}>
                      <Icon className={`h-4 w-4 ${active ? 'text-primary' : 'text-muted-foreground'}`} />
                    </div>
                    <div>
                      <p className={`text-sm font-medium ${active ? 'text-primary' : 'text-foreground'}`}>{f.label}</p>
                      <p className="text-[10px] text-muted-foreground">{t(f.descKey)}</p>
                    </div>
                  </button>
                )
              })}
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t('common.cancel')}</Button>
          <Button onClick={handleExport}>
            <Download className="mr-1.5 h-4 w-4" />
            {t('buchhaltung.exportDialog.exportBtn')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
