/**
 * Lightweight CSV-style lead import: paste one lead per line.
 * Columns: Vorname, Nachname, Firma, E-Mail, Telefon (comma or semicolon).
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { useCreateLeadsBatch, type NewLeadInput } from '@/api/hooks/useLeads'

interface LeadImportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

function parseRows(text: string): NewLeadInput[] {
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const cols = line.split(/[;,\t]/).map((c) => c.trim())
      const [firstName = '', lastName = '', company = '', email = '', phone = ''] = cols
      return { firstName, lastName, company, email, phone, source: 'csv' as const }
    })
    .filter((r) => r.firstName || r.lastName)
}

export function LeadImportDialog({ open, onOpenChange }: LeadImportDialogProps) {
  const { t } = useTranslation()
  const createBatch = useCreateLeadsBatch()
  const [text, setText] = useState('')

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- reset on open
    if (open) setText('')
  }, [open])

  const rows = parseRows(text)

  const handleImport = async () => {
    if (rows.length === 0) return
    const count = await createBatch.mutateAsync(rows)
    toast.success(t('leads.imported', { count }))
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('leads.import.title')}</DialogTitle>
          <DialogDescription>{t('leads.import.help')}</DialogDescription>
        </DialogHeader>

        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          rows={8}
          placeholder={t('leads.import.placeholder')}
          className="w-full resize-none rounded-md border border-border bg-transparent px-3 py-2 font-mono text-xs outline-none placeholder:text-muted-foreground focus:border-primary"
          autoFocus
        />
        <p className="text-xs text-muted-foreground">{t('leads.import.detected', { count: rows.length })}</p>

        <div className="flex justify-end gap-2 pt-1">
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t('common.cancel')}</Button>
          <Button onClick={handleImport} disabled={rows.length === 0}>{t('leads.import.confirm')}</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
