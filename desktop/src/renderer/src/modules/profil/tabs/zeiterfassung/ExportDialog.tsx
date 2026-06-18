import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Download, FileSpreadsheet, FileText, Building2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/cn'
import { useWorkTimeEntries } from '@/api/hooks/hr-hooks'
import { formatMinutes } from './time-utils'
import { toast } from 'sonner'

type ExportFormat = 'datev' | 'csv' | 'excel'

interface ExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const FORMAT_OPTIONS: { key: ExportFormat; label: string; description: string; icon: typeof FileText }[] = [
  {
    key: 'datev',
    label: 'DATEV',
    description: 'CSV (Windows-1252, Semikolon) — kompatibel mit DATEV Lohn',
    icon: Building2,
  },
  {
    key: 'csv',
    label: 'CSV',
    description: 'Standard CSV (UTF-8, Komma)',
    icon: FileText,
  },
  {
    key: 'excel',
    label: 'Excel',
    description: 'XLSX-kompatibel (Tab-getrennt)',
    icon: FileSpreadsheet,
  },
]

export default function ExportDialog({ open, onOpenChange }: ExportDialogProps) {
  const { t } = useTranslation()

  // Default range: last 30 days
  const today = new Date()
  const thirtyDaysAgo = new Date()
  thirtyDaysAgo.setDate(today.getDate() - 30)

  const [format, setFormat] = useState<ExportFormat>('csv')
  const [dateFrom, setDateFrom] = useState(thirtyDaysAgo.toISOString().split('T')[0])
  const [dateTo, setDateTo] = useState(today.toISOString().split('T')[0])

  const { data: entriesData } = useWorkTimeEntries({
    start_date: dateFrom,
    end_date: dateTo,
  })
  const allEntries = entriesData?.entries ?? []

  const filteredEntries = useMemo(
    () =>
      allEntries.filter((e) => {
        const date = e.clockIn?.slice(0, 10) ?? ''
        return date >= dateFrom && date <= dateTo
      }),
    [allEntries, dateFrom, dateTo],
  )

  const totalMinutes = filteredEntries.reduce((sum, e) => sum + (e.netWorkMinutes ?? 0), 0)
  const totalHours = (totalMinutes / 60).toFixed(1)

  const handleExport = () => {
    if (filteredEntries.length === 0) {
      toast.error(t('profil.zeiterfassung.export.noEntries'))
      return
    }
    const formatLabel = FORMAT_OPTIONS.find((f) => f.key === format)?.label ?? format
    toast.success(t('profil.zeiterfassung.export.started', { format: formatLabel, count: filteredEntries.length }))
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5 text-primary" />
            {t('profil.zeiterfassung.export.title')}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-5 py-2">
          {/* Format Selection */}
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">{t('profil.zeiterfassung.export.format')}</label>
            <div className="grid grid-cols-3 gap-2">
              {FORMAT_OPTIONS.map((opt) => {
                const Icon = opt.icon
                return (
                  <button
                    key={opt.key}
                    onClick={() => setFormat(opt.key)}
                    className={cn(
                      'flex flex-col items-center gap-1.5 p-3 rounded-lg border-2 transition-all text-center',
                      format === opt.key
                        ? 'border-primary bg-primary/5'
                        : 'border-border bg-card hover:border-primary/30',
                    )}
                  >
                    <Icon className={cn(
                      'h-5 w-5',
                      format === opt.key ? 'text-primary' : 'text-muted-foreground',
                    )} />
                    <span className={cn(
                      'text-sm font-medium',
                      format === opt.key ? 'text-primary' : 'text-foreground',
                    )}>
                      {opt.label}
                    </span>
                  </button>
                )
              })}
            </div>
            <p className="text-xs text-muted-foreground">
              {FORMAT_OPTIONS.find((f) => f.key === format)?.description}
            </p>
          </div>

          {/* Date Range */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('profil.zeiterfassung.export.from')}</label>
              <Input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('profil.zeiterfassung.export.to')}</label>
              <Input
                type="date"
                value={dateTo}
                max={today.toISOString().split('T')[0]}
                onChange={(e) => setDateTo(e.target.value)}
              />
            </div>
          </div>

          {/* Preview */}
          <div className="rounded-lg border border-border bg-secondary/30 p-4">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
              {t('profil.zeiterfassung.export.preview')}
            </h4>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <p className="text-2xl font-bold text-foreground tabular-nums">
                  {filteredEntries.length}
                </p>
                <p className="text-xs text-muted-foreground">{t('profil.zeiterfassung.entries')}</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-foreground tabular-nums">
                  {totalHours}h
                </p>
                <p className="text-xs text-muted-foreground">
                  ({formatMinutes(totalMinutes)})
                </p>
              </div>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleExport} disabled={filteredEntries.length === 0} className="gap-2">
            <Download className="h-4 w-4" />
            {t('profil.zeiterfassung.export.export')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
