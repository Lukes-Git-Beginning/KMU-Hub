/**
 * Leads inbox — the raw, pre-qualified pipeline stage.
 *
 * Lists leads with source, auto-score temperature (ampel) and lifecycle status.
 * Supports status transitions, manual temperature override, manual create,
 * CSV-style import, and conversion into contact/company/deal.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Upload, Inbox, Phone, Mail, Building2, Flame, Trophy, PhoneCall, XCircle, RotateCcw } from 'lucide-react'
import { toast } from 'sonner'
import {
  useLeads,
  useUpdateLeadStatus,
  useSetLeadTemperature,
  effectiveTemperature,
  type Lead,
  type LeadStatus,
  type LeadSource,
} from '@/api/hooks/useLeads'
import { ItemActions, EmptyState, type ActionItem } from '@/components/shared'
import { Skeleton } from '@/components/ui/skeleton'
import { formatDate } from '@/lib/format'
import { TEMP_COLORS } from './leadVisuals'
import { ConvertLeadDialog } from './ConvertLeadDialog'
import { LeadFormDialog } from './LeadFormDialog'
import { LeadImportDialog } from './LeadImportDialog'

type StatusFilter = 'all' | LeadStatus
const STATUS_ORDER: LeadStatus[] = ['new', 'contacted', 'qualified', 'disqualified']

const statusBadge: Record<LeadStatus, string> = {
  new: 'bg-primary-light text-primary',
  contacted: 'bg-warning-light text-warning',
  qualified: 'bg-success-light text-success',
  disqualified: 'bg-secondary text-text-disabled',
}

const sourceIcon: Record<LeadSource, typeof Phone> = {
  manual: Building2,
  csv: Upload,
  dialer: PhoneCall,
}

export default function LeadsInboxPage() {
  const { t } = useTranslation()
  const { data: leads, isLoading } = useLeads()
  const updateStatus = useUpdateLeadStatus()
  const setTemperature = useSetLeadTemperature()

  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [convertLead, setConvertLead] = useState<Lead | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [importOpen, setImportOpen] = useState(false)

  const all = useMemo(() => leads ?? [], [leads])
  const counts = useMemo(() => {
    const c: Record<StatusFilter, number> = { all: all.length, new: 0, contacted: 0, qualified: 0, disqualified: 0 }
    for (const l of all) c[l.status]++
    return c
  }, [all])

  const filtered = statusFilter === 'all' ? all : all.filter((l) => l.status === statusFilter)

  const changeStatus = (id: string, status: LeadStatus) => {
    updateStatus.mutate({ id, status })
    toast.success(t('leads.toast.statusChanged'))
  }
  const setTemp = (id: string, temperature: 'hot' | 'warm' | 'cold') => {
    setTemperature.mutate({ id, temperature })
    toast.success(t('leads.toast.tempChanged'))
  }

  const getActions = (l: Lead): ActionItem[] => {
    const items: ActionItem[] = [
      { label: t('leads.action.convert'), icon: Trophy, onClick: () => setConvertLead(l) },
    ]
    if (l.status === 'new') items.push({ label: t('leads.action.markContacted'), icon: PhoneCall, onClick: () => changeStatus(l.id, 'contacted') })
    if (l.status === 'disqualified' || l.status === 'qualified') items.push({ label: t('leads.action.reopen'), icon: RotateCcw, onClick: () => changeStatus(l.id, 'new') })
    items.push({ label: t('leads.action.setHot'), icon: Flame, onClick: () => setTemp(l.id, 'hot'), separator: true })
    items.push({ label: t('leads.action.setWarm'), icon: Flame, onClick: () => setTemp(l.id, 'warm') })
    items.push({ label: t('leads.action.setCold'), icon: Flame, onClick: () => setTemp(l.id, 'cold') })
    if (l.status !== 'disqualified') items.push({ label: t('leads.action.disqualify'), icon: XCircle, variant: 'destructive', onClick: () => changeStatus(l.id, 'disqualified'), separator: true })
    return items
  }

  const filterChips: { key: StatusFilter; label: string }[] = [
    { key: 'all', label: t('leads.status.all') },
    ...STATUS_ORDER.map((s) => ({ key: s as StatusFilter, label: t(`leads.status.${s}`) })),
  ]

  return (
    <div className="flex h-full flex-col animate-fade-in">
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2 border-b border-border px-4 py-3">
        <div className="flex flex-wrap items-center gap-1">
          {filterChips.map((chip) => (
            <button
              key={chip.key}
              onClick={() => setStatusFilter(chip.key)}
              className={`flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors ${
                statusFilter === chip.key ? 'bg-primary text-primary-foreground' : 'bg-secondary text-muted-foreground hover:text-foreground'
              }`}
            >
              {chip.label}
              <span className={`rounded-full px-1.5 ${statusFilter === chip.key ? 'bg-primary-foreground/20' : 'bg-background/60'}`}>
                {counts[chip.key]}
              </span>
            </button>
          ))}
        </div>
        <div className="ml-auto flex items-center gap-2">
          <button
            onClick={() => setImportOpen(true)}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <Upload className="h-4 w-4" />
            <span className="hidden sm:inline">{t('leads.import.button')}</span>
          </button>
          <button
            onClick={() => setFormOpen(true)}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
          >
            <Plus className="h-4 w-4" />
            {t('leads.new')}
          </button>
        </div>
      </div>

      {/* List */}
      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="space-y-2 p-4">
            {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-16 w-full" />)}
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={Inbox}
            title={t('leads.empty.title')}
            description={statusFilter === 'all' ? t('leads.empty.description') : t('leads.empty.filtered')}
            action={statusFilter === 'all' ? { label: t('leads.new'), onClick: () => setFormOpen(true) } : undefined}
          />
        ) : (
          filtered.map((lead) => {
            const temp = effectiveTemperature(lead)
            const SourceIcon = sourceIcon[lead.source]
            const initials = `${lead.firstName[0] ?? ''}${lead.lastName[0] ?? ''}`.toUpperCase()
            return (
              <div
                key={lead.id}
                className="group flex items-center gap-3 border-b border-border-muted px-4 py-3 transition-colors hover:bg-secondary/40"
              >
                {/* Temperature ampel */}
                <span
                  className="h-2.5 w-2.5 shrink-0 rounded-full"
                  style={{ backgroundColor: TEMP_COLORS[temp] }}
                  title={t(`leads.temp.${temp}`)}
                />
                {/* Avatar */}
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-light text-xs font-medium text-primary">
                  {initials || '–'}
                </div>
                {/* Name + company */}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="truncate text-sm font-medium text-foreground">
                      {lead.firstName} {lead.lastName}
                    </span>
                    <span className={`inline-flex shrink-0 rounded-full px-1.5 py-0.5 text-[10px] ${statusBadge[lead.status]}`}>
                      {t(`leads.status.${lead.status}`)}
                    </span>
                  </div>
                  <p className="truncate text-xs text-muted-foreground">
                    {lead.company || t('leads.noCompany')}
                  </p>
                </div>
                {/* Source */}
                <span className="hidden shrink-0 items-center gap-1 text-xs text-muted-foreground sm:flex" title={t(`leads.source.${lead.source}`)}>
                  <SourceIcon className="h-3.5 w-3.5" />
                  <span className="hidden md:inline">{t(`leads.source.${lead.source}`)}</span>
                </span>
                {/* Score */}
                <div className="hidden w-24 shrink-0 sm:block" title={`${t('leads.score')}: ${lead.score}`}>
                  <div className="mb-0.5 flex items-center justify-between text-[10px] text-muted-foreground">
                    <span>{t('leads.score')}</span>
                    <span className="font-medium text-foreground">{lead.score}</span>
                  </div>
                  <div className="h-1.5 w-full overflow-hidden rounded-full bg-secondary">
                    <div className="h-full rounded-full" style={{ width: `${lead.score}%`, backgroundColor: TEMP_COLORS[temp] }} />
                  </div>
                </div>
                {/* Date */}
                <span className="hidden shrink-0 text-xs text-muted-foreground lg:block">{formatDate(lead.createdAt)}</span>
                {/* Quick contact + actions */}
                <div className="flex shrink-0 items-center gap-1">
                  {lead.email && (
                    <a href={`mailto:${lead.email}`} className="hidden rounded-md border border-border p-1 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground group-hover:inline-flex" title={lead.email}>
                      <Mail className="h-3.5 w-3.5" />
                    </a>
                  )}
                  {lead.phone && (
                    <span className="hidden rounded-md border border-border p-1 text-muted-foreground group-hover:inline-flex" title={lead.phone}>
                      <Phone className="h-3.5 w-3.5" />
                    </span>
                  )}
                  <ItemActions items={getActions(lead)} />
                </div>
              </div>
            )
          })
        )}
      </div>

      <ConvertLeadDialog lead={convertLead} open={!!convertLead} onOpenChange={(o) => { if (!o) setConvertLead(null) }} />
      <LeadFormDialog open={formOpen} onOpenChange={setFormOpen} />
      <LeadImportDialog open={importOpen} onOpenChange={setImportOpen} />
    </div>
  )
}
