/**
 * Export dialog — downloads time entries as CSV (real, client-side).
 *
 * XLSX / DATEV-Lohn / PDF are marked as backend-pending (backend-gaps.md):
 * DATEV LODAS export needs server-side generation. CSV covers the immediate
 * "Auswertung & Export" market-parity need and works in the demo.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Download, FileSpreadsheet, FileText, Landmark } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/cn'
import { toast } from 'sonner'
import { useWorkTimeEntries } from '@/api/hooks/hr-hooks'
import type { WorkTimeEntry } from '@/api/hr-types'

interface ExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

type Format = 'csv' | 'xlsx' | 'datev' | 'pdf'

function fmtTime(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

function csvCell(v: unknown): string {
  const s = String(v ?? '')
  return /[";\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

function buildCsv(entries: WorkTimeEntry[], headers: string[]): string {
  const rows = entries
    .filter((e) => e.clockOut)
    .map((e) => [
      e.clockIn.slice(0, 10),
      fmtTime(e.clockIn),
      fmtTime(e.clockOut),
      e.breakMinutes,
      e.netWorkMinutes ?? '',
      e.projectName ?? '',
      e.customerName ?? '',
      e.activity ?? '',
      e.billable ? 'Ja' : 'Nein',
    ].map(csvCell).join(';'))
  return [headers.join(';'), ...rows].join('\r\n')
}

function downloadFile(content: string, filename: string, mime: string) {
  const blob = new Blob(['﻿' + content], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

export function ExportDialog({ open, onOpenChange }: ExportDialogProps) {
  const { t } = useTranslation()
  const { data } = useWorkTimeEntries()
  const [format, setFormat] = useState<Format>('csv')

  const formats: { id: Format; labelKey: string; icon: typeof Download; ready: boolean }[] = [
    { id: 'csv', labelKey: 'zeiterfassung.export.csv', icon: FileSpreadsheet, ready: true },
    { id: 'xlsx', labelKey: 'zeiterfassung.export.xlsx', icon: FileSpreadsheet, ready: false },
    { id: 'datev', labelKey: 'zeiterfassung.export.datev', icon: Landmark, ready: false },
    { id: 'pdf', labelKey: 'zeiterfassung.export.pdf', icon: FileText, ready: false },
  ]

  const entries = data?.entries ?? []
  const exportableCount = entries.filter((e) => e.clockOut).length

  const handleExport = () => {
    if (format !== 'csv') {
      toast.info(t('zeiterfassung.export.comingSoon'))
      return
    }
    const headers = [
      t('zeiterfassung.export.col.date'),
      t('zeiterfassung.export.col.from'),
      t('zeiterfassung.export.col.to'),
      t('zeiterfassung.export.col.break'),
      t('zeiterfassung.export.col.net'),
      t('zeiterfassung.export.col.project'),
      t('zeiterfassung.export.col.customer'),
      t('zeiterfassung.export.col.activity'),
      t('zeiterfassung.export.col.billable'),
    ]
    const csv = buildCsv(entries, headers)
    downloadFile(csv, `zeiterfassung-${new Date().toISOString().slice(0, 10)}.csv`, 'text/csv;charset=utf-8;')
    toast.success(t('zeiterfassung.export.done', { count: exportableCount }))
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('zeiterfassung.export.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-3 py-2">
          <p className="text-xs text-muted-foreground">
            {t('zeiterfassung.export.subtitle', { count: exportableCount })}
          </p>
          <div className="grid grid-cols-2 gap-2">
            {formats.map((f) => {
              const Icon = f.icon
              const active = format === f.id
              return (
                <button
                  key={f.id}
                  onClick={() => setFormat(f.id)}
                  className={cn(
                    'flex items-center gap-2 rounded-lg border px-3 py-2.5 text-sm transition-colors',
                    active ? 'border-primary bg-primary/5 text-primary' : 'border-border text-foreground hover:bg-secondary',
                  )}
                >
                  <Icon className="h-4 w-4 shrink-0" />
                  <span className="flex-1 text-left">{t(f.labelKey)}</span>
                  {!f.ready && (
                    <span className="rounded-full bg-secondary px-1.5 py-0.5 text-[9px] font-medium text-muted-foreground">
                      {t('zeiterfassung.export.soon')}
                    </span>
                  )}
                </button>
              )
            })}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t('common.cancel')}</Button>
          <Button onClick={handleExport} disabled={exportableCount === 0} className="gap-1.5">
            <Download className="h-4 w-4" />
            {t('zeiterfassung.export.action')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
