import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { formatDate } from '@/lib/format'
import {
  Send,
  UserPlus,
  Check,
  Eye,
  Clock,
  AlertTriangle,
  FileSignature,
  Trash2,
  ToggleLeft,
  ToggleRight,
  Info,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import type { Contract, ContractSigner } from '@/stores/vertraege'

// ─── Status Config ─────────────────────────────────────────────

const signerStatusConfig: Record<ContractSigner['status'], { labelKey: string; colorClass: string; icon: typeof Check }> = {
  pending: { labelKey: 'vertraege.esignatur.status.pending', colorClass: 'bg-secondary text-muted-foreground', icon: Clock },
  sent: { labelKey: 'vertraege.esignatur.status.sent', colorClass: 'bg-info-light text-info', icon: Send },
  viewed: { labelKey: 'vertraege.esignatur.status.viewed', colorClass: 'bg-warning-light text-warning', icon: Eye },
  signed: { labelKey: 'vertraege.esignatur.status.signed', colorClass: 'bg-success-light text-success', icon: Check },
  declined: { labelKey: 'vertraege.esignatur.status.declined', colorClass: 'bg-error-light text-error', icon: AlertTriangle },
}

// ─── Timeline Builder ──────────────────────────────────────────

interface TimelineEvent {
  date: string
  label: string
  icon: typeof Check
  iconColor: string
}

function buildTimeline(contract: Contract, signers: ContractSigner[], t: (key: string, opts?: Record<string, unknown>) => string): TimelineEvent[] {
  const events: TimelineEvent[] = []

  // Contract created
  const createdEntry = contract.history.find((h) => h.action.includes('angelegt') || h.action.includes('unterzeichnet') || h.action.includes('erstellt'))
  events.push({
    date: createdEntry?.date ?? contract.startDate,
    label: t('vertraege.esignatur.timeline.created'),
    icon: FileSignature,
    iconColor: 'text-primary',
  })

  // Check if any signer has been sent
  const hasSent = signers.some((s) => s.status !== 'pending')
  if (hasSent) {
    events.push({
      date: new Date().toISOString().split('T')[0],
      label: t('vertraege.esignatur.timeline.sent'),
      icon: Send,
      iconColor: 'text-info',
    })
  }

  // Add viewed/signed/declined events per signer
  for (const signer of signers) {
    if (signer.status === 'viewed') {
      events.push({
        date: new Date().toISOString().split('T')[0],
        label: t('vertraege.esignatur.timeline.viewed', { name: signer.name }),
        icon: Eye,
        iconColor: 'text-warning',
      })
    }
    if (signer.status === 'signed') {
      events.push({
        date: signer.signedAt?.split('T')[0] ?? new Date().toISOString().split('T')[0],
        label: t('vertraege.esignatur.timeline.signed', { name: signer.name }),
        icon: Check,
        iconColor: 'text-success',
      })
    }
    if (signer.status === 'declined') {
      events.push({
        date: new Date().toISOString().split('T')[0],
        label: t('vertraege.esignatur.timeline.declined', { name: signer.name }),
        icon: AlertTriangle,
        iconColor: 'text-error',
      })
    }
  }

  return events
}

// ─── E-Signatur Dialog ─────────────────────────────────────────

export default function ESignaturDialog({
  open,
  onClose,
  contract,
  onUpdateSigners,
}: {
  open: boolean
  onClose: () => void
  contract: Contract
  onUpdateSigners: (signers: ContractSigner[]) => void
}) {
  const { t } = useTranslation()
  const [signers, setSigners] = useState<ContractSigner[]>(contract.signers ?? [])
  const [sequential, setSequential] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [newName, setNewName] = useState('')
  const [newEmail, setNewEmail] = useState('')
  const [newOrder, setNewOrder] = useState(signers.length + 1)

  const timeline = useMemo(() => buildTimeline(contract, signers, t), [contract, signers, t])

  const handleAddSigner = () => {
    if (!newName.trim()) {
      toast.error(t('vertraege.esignatur.errorName'))
      return
    }
    if (!newEmail.trim() || !newEmail.includes('@')) {
      toast.error(t('vertraege.esignatur.errorEmail'))
      return
    }
    const signer: ContractSigner = {
      name: newName.trim(),
      email: newEmail.trim(),
      order: sequential ? newOrder : signers.length + 1,
      status: 'pending',
    }
    setSigners((prev) => [...prev, signer])
    setNewName('')
    setNewEmail('')
    setNewOrder(signers.length + 2)
    setShowAddForm(false)
  }

  const handleRemoveSigner = (index: number) => {
    setSigners((prev) => prev.filter((_, i) => i !== index))
  }

  const handleSend = () => {
    if (signers.length === 0) {
      toast.error(t('vertraege.esignatur.errorNoSigners'))
      return
    }
    onUpdateSigners(signers.map((s) => ({ ...s, status: s.status === 'pending' ? 'sent' : s.status })))
    toast.success(t('vertraege.esignatur.sendSuccess', { count: signers.length }))
    onClose()
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-2xl max-h-[90vh] overflow-y-auto">
        <div className="p-6">
        {/* Header */}
        <DialogHeader className="mb-5">
          <DialogTitle className="text-lg font-semibold text-foreground flex items-center gap-2">
            <FileSignature className="h-5 w-5 text-primary" />
            {t('vertraege.esignatur.title')}
          </DialogTitle>
          <DialogDescription className="text-xs text-muted-foreground mt-0.5">{contract.title}</DialogDescription>
        </DialogHeader>

        {/* Info Banner */}
        <div className="rounded-lg border border-info/30 bg-info-light/30 p-3 mb-5">
          <div className="flex items-start gap-2">
            <Info className="h-4 w-4 text-info mt-0.5 shrink-0" />
            <p className="text-xs text-info">
              {t('vertraege.esignatur.infoBanner')}
            </p>
          </div>
        </div>

        {/* Signing Order Toggle */}
        <div className="space-y-3 mb-5">
          <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vertraege.esignatur.labelReihenfolge')}</h4>
          <div className="flex items-center gap-3">
            <button
              onClick={() => setSequential(true)}
              className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors ${
                sequential
                  ? 'border-primary bg-primary-light text-primary font-medium'
                  : 'border-border text-muted-foreground hover:bg-secondary'
              }`}
            >
              <ToggleRight className="h-4 w-4" />
              {t('vertraege.esignatur.sequentiell')}
            </button>
            <button
              onClick={() => setSequential(false)}
              className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors ${
                !sequential
                  ? 'border-primary bg-primary-light text-primary font-medium'
                  : 'border-border text-muted-foreground hover:bg-secondary'
              }`}
            >
              <ToggleLeft className="h-4 w-4" />
              {t('vertraege.esignatur.parallel')}
            </button>
            <span className="text-xs text-muted-foreground">
              {sequential ? t('vertraege.esignatur.sequentiellDesc') : t('vertraege.esignatur.parallelDesc')}
            </span>
          </div>
        </div>

        {/* Signer List */}
        <div className="space-y-3 mb-5">
          <div className="flex items-center justify-between">
            <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t('vertraege.esignatur.signers', { count: signers.length })}
            </h4>
            <button
              onClick={() => { setShowAddForm(true); setNewOrder(signers.length + 1) }}
              className="flex items-center gap-1 rounded-lg border border-border px-2.5 py-1 text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
            >
              <UserPlus className="h-3 w-3" />
              {t('vertraege.esignatur.buttonAddSigner')}
            </button>
          </div>

          {signers.length === 0 && !showAddForm && (
            <p className="text-sm text-muted-foreground py-4 text-center border border-dashed border-border rounded-lg">
              {t('vertraege.esignatur.noSigners')}
            </p>
          )}

          {signers.length > 0 && (
            <div className="rounded-lg border border-border overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-secondary/30">
                    {sequential && <th className="px-3 py-2 text-left font-medium text-muted-foreground text-xs w-12">#</th>}
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground text-xs">{t('vertraege.esignatur.table.name')}</th>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground text-xs">{t('vertraege.esignatur.table.email')}</th>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground text-xs">{t('vertraege.esignatur.table.status')}</th>
                    <th className="px-3 py-2 text-right font-medium text-muted-foreground text-xs w-10"></th>
                  </tr>
                </thead>
                <tbody>
                  {signers.map((signer, idx) => {
                    const statusConf = signerStatusConfig[signer.status]
                    const StatusIcon = statusConf.icon
                    return (
                      <tr key={idx} className="border-b border-border-muted last:border-0">
                        {sequential && (
                          <td className="px-3 py-2 text-muted-foreground font-mono text-xs">{signer.order}</td>
                        )}
                        <td className="px-3 py-2 text-foreground font-medium">{signer.name}</td>
                        <td className="px-3 py-2 text-muted-foreground font-mono text-xs">{signer.email}</td>
                        <td className="px-3 py-2">
                          <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${statusConf.colorClass}`}>
                            <StatusIcon className="h-3 w-3" />
                            {t(statusConf.labelKey)}
                          </span>
                        </td>
                        <td className="px-3 py-2 text-right">
                          <button
                            onClick={() => handleRemoveSigner(idx)}
                            className="rounded p-1 text-muted-foreground hover:text-error hover:bg-error-light transition-colors"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}

          {/* Inline Add Form */}
          {showAddForm && (
            <div className="rounded-lg border border-border bg-secondary/20 p-3 space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-xs font-medium text-foreground">{t('vertraege.esignatur.labelName')}</label>
                  <input
                    type="text"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder={t('vertraege.esignatur.placeholderName')}
                    className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                    autoFocus
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-xs font-medium text-foreground">{t('vertraege.esignatur.labelEmail')}</label>
                  <input
                    type="email"
                    value={newEmail}
                    onChange={(e) => setNewEmail(e.target.value)}
                    placeholder="max@firma.de"
                    className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>
              </div>
              {sequential && (
                <div className="space-y-1">
                  <label className="text-xs font-medium text-foreground">{t('vertraege.esignatur.labelOrder')}</label>
                  <input
                    type="number"
                    min={1}
                    value={newOrder}
                    onChange={(e) => setNewOrder(Number(e.target.value))}
                    className="w-20 rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
                  />
                </div>
              )}
              <div className="flex items-center gap-2">
                <button
                  onClick={handleAddSigner}
                  className="rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
                >
                  {t('vertraege.esignatur.buttonAdd')}
                </button>
                <button
                  onClick={() => { setShowAddForm(false); setNewName(''); setNewEmail('') }}
                  className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-secondary transition-colors"
                >
                  {t('common.cancel')}
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Status Timeline */}
        <div className="space-y-3 mb-6">
          <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
            {t('vertraege.esignatur.statusVerlauf')}
          </h4>
          {timeline.length === 0 ? (
            <p className="text-xs text-muted-foreground py-2">{t('vertraege.esignatur.noEntries')}</p>
          ) : (
            <div className="space-y-0 pl-1">
              {timeline.map((event, idx) => {
                const EventIcon = event.icon
                return (
                  <div key={idx} className="flex gap-3 pb-3 last:pb-0">
                    <div className="flex flex-col items-center">
                      <div className={`flex h-5 w-5 items-center justify-center rounded-full shrink-0 ${
                        idx === 0 ? 'bg-primary-light' : 'bg-secondary'
                      }`}>
                        <EventIcon className={`h-3 w-3 ${event.iconColor}`} />
                      </div>
                      {idx < timeline.length - 1 && (
                        <div className="w-px flex-1 bg-border mt-1" />
                      )}
                    </div>
                    <div className="min-w-0 pb-1">
                      <p className="text-sm text-foreground">{event.label}</p>
                      <span className="text-[10px] text-muted-foreground">
                        {formatDate(event.date + 'T00:00:00')}
                      </span>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* Footer */}
        <DialogFooter className="border-t border-border pt-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSend}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Send className="h-4 w-4" />
            {t('vertraege.esignatur.buttonSend')}
          </button>
        </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}
