/**
 * Convert a qualified lead into real CRM records.
 *
 * Always creates a contact; the user opts in to also creating a company and/or
 * an initial deal. Uses the existing CRM create hooks so the flow is wired for
 * the real backend, then marks the lead qualified (removing it from the inbox).
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { User, Building2, TrendingUp } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { useCreateContact } from '@/api/hooks/useContacts'
import { useCreateCompany } from '@/api/hooks/useCompanies'
import { useCreateDeal } from '@/api/hooks/useDeals'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import { useConvertLead, type Lead } from '@/api/hooks/useLeads'

interface ConvertLeadDialogProps {
  lead: Lead | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ConvertLeadDialog({ lead, open, onOpenChange }: ConvertLeadDialogProps) {
  const { t } = useTranslation()
  const createContact = useCreateContact()
  const createCompany = useCreateCompany()
  const createDeal = useCreateDeal()
  const convertLead = useConvertLead()
  const { data: stagesData } = usePipelineStages()
  const firstStageId = stagesData?.stages?.[0]?.id

  const [withCompany, setWithCompany] = useState(false)
  const [withDeal, setWithDeal] = useState(false)
  const [dealName, setDealName] = useState('')
  const [dealValue, setDealValue] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (open && lead) {
      setWithCompany(Boolean(lead.company))
      setWithDeal(false)
      setDealName(`${lead.company || `${lead.firstName} ${lead.lastName}`} — Erstprojekt`)
      setDealValue('')
      setSubmitting(false)
    }
  }, [open, lead])

  if (!lead) return null

  const handleConfirm = async () => {
    setSubmitting(true)
    try {
      let companyId: string | undefined
      if (withCompany && lead.company) {
        const company = await createCompany.mutateAsync({ name: lead.company }).catch(() => null)
        companyId = (company as { id?: string } | null)?.id
      }
      await createContact.mutateAsync({
        first_name: lead.firstName,
        last_name: lead.lastName,
        email: lead.email || undefined,
        phone: lead.phone || undefined,
        company_id: companyId,
        notes: lead.notes || undefined,
      })
      if (withDeal && firstStageId) {
        await createDeal.mutateAsync({
          name: dealName.trim() || `${lead.firstName} ${lead.lastName}`,
          value: Number(dealValue) || 0,
          currency: 'EUR',
          stage_id: firstStageId,
          custom_fields: { _contactName: `${lead.firstName} ${lead.lastName}`, _companyName: lead.company },
        }).catch(() => null)
      }
      await convertLead.mutateAsync({ id: lead.id, createCompany: withCompany, createDeal: withDeal })
      toast.success(t('leads.convert.success', { name: `${lead.firstName} ${lead.lastName}` }))
      onOpenChange(false)
    } catch {
      toast.error(t('leads.convert.error'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('leads.convert.title')}</DialogTitle>
          <DialogDescription>
            {t('leads.convert.description', { name: `${lead.firstName} ${lead.lastName}` })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 pt-1">
          {/* Always: contact */}
          <div className="flex items-center gap-2.5 rounded-lg border border-primary/30 bg-primary/5 px-3 py-2.5">
            <User className="h-4 w-4 text-primary" />
            <div className="min-w-0">
              <p className="text-sm font-medium text-foreground">{t('leads.convert.createContact')}</p>
              <p className="truncate text-xs text-muted-foreground">
                {lead.firstName} {lead.lastName}{lead.email ? ` · ${lead.email}` : ''}
              </p>
            </div>
          </div>

          {/* Optional: company */}
          <label className={`flex items-center gap-2.5 rounded-lg border px-3 py-2.5 transition-colors ${lead.company ? 'cursor-pointer border-border hover:bg-accent' : 'cursor-not-allowed border-border opacity-50'}`}>
            <input
              type="checkbox"
              checked={withCompany}
              disabled={!lead.company}
              onChange={(e) => setWithCompany(e.target.checked)}
              className="h-4 w-4 accent-primary"
            />
            <Building2 className="h-4 w-4 text-muted-foreground" />
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-foreground">{t('leads.convert.createCompany')}</p>
              <p className="truncate text-xs text-muted-foreground">{lead.company || t('leads.convert.noCompany')}</p>
            </div>
          </label>

          {/* Optional: deal */}
          <label className="flex cursor-pointer items-center gap-2.5 rounded-lg border border-border px-3 py-2.5 transition-colors hover:bg-accent">
            <input
              type="checkbox"
              checked={withDeal}
              onChange={(e) => setWithDeal(e.target.checked)}
              className="h-4 w-4 accent-primary"
            />
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
            <p className="flex-1 text-sm font-medium text-foreground">{t('leads.convert.createDeal')}</p>
          </label>

          {withDeal && (
            <div className="grid grid-cols-[1fr_120px] gap-2 pl-2">
              <input
                type="text"
                value={dealName}
                onChange={(e) => setDealName(e.target.value)}
                placeholder={t('leads.convert.dealName')}
                className="h-9 rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
              />
              <input
                type="number"
                value={dealValue}
                onChange={(e) => setDealValue(e.target.value)}
                placeholder="€"
                min="0"
                className="h-9 rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
              />
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleConfirm} disabled={submitting}>
            {t('leads.convert.confirm')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
