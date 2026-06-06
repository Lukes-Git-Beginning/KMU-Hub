/**
 * Manually create a single lead, with a live auto-score preview.
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
import {
  useCreateLead,
  computeLeadScore,
  scoreToTemperature,
  type LeadSource,
} from '@/api/hooks/useLeads'
import { TEMP_COLORS } from './leadVisuals'

interface LeadFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const SOURCES: LeadSource[] = ['manual', 'csv', 'dialer']

export function LeadFormDialog({ open, onOpenChange }: LeadFormDialogProps) {
  const { t } = useTranslation()
  const createLead = useCreateLead()
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [company, setCompany] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [source, setSource] = useState<LeadSource>('manual')
  const [notes, setNotes] = useState('')

  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- reset on open
      setFirstName(''); setLastName(''); setCompany(''); setEmail(''); setPhone(''); setSource('manual'); setNotes('')
    }
  }, [open])

  const score = computeLeadScore({ source, email, phone, company, notes })
  const temp = scoreToTemperature(score)

  const handleSubmit = async () => {
    if (!firstName.trim() && !lastName.trim()) return
    await createLead.mutateAsync({ firstName: firstName.trim(), lastName: lastName.trim(), company: company.trim(), email: email.trim(), phone: phone.trim(), source, notes: notes.trim() })
    toast.success(t('leads.created'))
    onOpenChange(false)
  }

  const inputCls = 'h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('leads.form.title')}</DialogTitle>
          <DialogDescription>{t('leads.form.description')}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3 pt-1">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('leads.form.firstName')}</label>
              <input value={firstName} onChange={(e) => setFirstName(e.target.value)} className={inputCls} autoFocus />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('leads.form.lastName')}</label>
              <input value={lastName} onChange={(e) => setLastName(e.target.value)} className={inputCls} />
            </div>
          </div>

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('leads.form.company')}</label>
            <input value={company} onChange={(e) => setCompany(e.target.value)} className={inputCls} />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('leads.form.email')}</label>
              <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} className={inputCls} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('leads.form.phone')}</label>
              <input value={phone} onChange={(e) => setPhone(e.target.value)} className={inputCls} />
            </div>
          </div>

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('leads.form.source')}</label>
            <select value={source} onChange={(e) => setSource(e.target.value as LeadSource)} className={inputCls}>
              {SOURCES.map((s) => (
                <option key={s} value={s}>{t(`leads.source.${s}`)}</option>
              ))}
            </select>
          </div>

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('leads.form.notes')}</label>
            <textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} className="w-full resize-none rounded-md border border-border bg-transparent px-3 py-2 text-sm outline-none placeholder:text-muted-foreground focus:border-primary" />
          </div>

          {/* Live auto-score preview */}
          <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-3 py-2.5">
            <span className="text-sm text-muted-foreground">{t('leads.form.scorePreview')}</span>
            <span className="flex items-center gap-2">
              <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: TEMP_COLORS[temp] }} />
              <span className="text-sm font-semibold text-foreground">{score}</span>
              <span className="text-xs text-muted-foreground">{t(`leads.temp.${temp}`)}</span>
            </span>
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t('common.cancel')}</Button>
          <Button onClick={handleSubmit} disabled={!firstName.trim() && !lastName.trim()}>{t('leads.form.create')}</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
