import { useState, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  Plus,
  FileSignature,
  Building,
  Truck,
  Wrench,
  UserCheck,
  KeyRound,
  Shield,
  Eye,
  Edit,
  Trash2,
  X,
  ChevronDown,
  Filter,
  AlertTriangle,
  CalendarClock,
  Clock,
  RefreshCw,
  Ban,
  Archive,
  Wallet,
  Coins,
  Pen,
  FileText,
  Bell,
  LayoutTemplate,
  FilePlus,
  FilePen,
  FileX,
  BellRing,
  History,
  Upload,
  Loader2,
  History as VersionIcon,
  FileSearch,
  User,
  Briefcase,
  Receipt,
  Sparkles,
  Link2,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useVertraegeStore, type Contract, type ContractType, type ContractStatus, type ContractTemplate, type ContractDocument } from '@/stores/vertraege'
import { useVertraegePrefsStore } from '@/stores/vertraegePrefs'
import { useVertraegeSettingsStore } from '@/stores/vertraegeSettings'
import { ItemActions, ConfirmDialog, EmptyState, DetailPanel, PageHeader } from '@/components/shared'
import { formatCurrency, formatDate } from '@/lib/format'
import ESignaturDialog from './ESignaturDialog'
import { FilePreviewModal } from '@/modules/dokumente/FilePreviewModal'
import { VersionHistoryPanel } from '@/modules/dokumente/VersionHistoryPanel'
import { useDocumentFiles } from '@/api/hooks/useDocuments'
import { useDocumentUpload } from '@/api/hooks/useDocumentUpload'
import type { DocumentFile } from '@/api/types/document-types'
import { useContacts } from '@/api/hooks/useContacts'
import { useDeals } from '@/api/hooks/useDeals'
import { useInvoices } from '@/api/hooks/useFinance'

// ─── Type Config ─────────────────────────────────────────────────

type TabKey = 'aktiv' | 'auslaufend' | 'archiv' | 'vorlagen'
type TypeFilter = 'all' | ContractType

const contractTypeConfig: Record<ContractType, { labelKey: string; icon: typeof Building; colorClass: string }> = {
  mietvertrag: { labelKey: 'vertraege.type.mietvertrag', icon: Building, colorClass: 'bg-info-light text-info' },
  liefervertrag: { labelKey: 'vertraege.type.liefervertrag', icon: Truck, colorClass: 'bg-warning-light text-warning' },
  servicevertrag: { labelKey: 'vertraege.type.servicevertrag', icon: Wrench, colorClass: 'bg-success-light text-success' },
  arbeitsvertrag: { labelKey: 'vertraege.type.arbeitsvertrag', icon: UserCheck, colorClass: 'bg-purple-100 text-purple-600 dark:bg-purple-500/20 dark:text-purple-400' },
  lizenz: { labelKey: 'vertraege.type.lizenz', icon: KeyRound, colorClass: 'bg-cyan-100 text-cyan-600 dark:bg-cyan-500/20 dark:text-cyan-400' },
  versicherung: { labelKey: 'vertraege.type.versicherung', icon: Shield, colorClass: 'bg-error-light text-error' },
}

const statusConfig: Record<ContractStatus, { labelKey: string; colorClass: string }> = {
  active: { labelKey: 'vertraege.status.active', colorClass: 'bg-success-light text-success' },
  expiring: { labelKey: 'vertraege.status.expiring', colorClass: 'bg-warning-light text-warning' },
  terminated: { labelKey: 'vertraege.status.terminated', colorClass: 'bg-secondary text-muted-foreground' },
  expired: { labelKey: 'vertraege.status.expired', colorClass: 'bg-error-light text-error' },
}

const renewalLabelKeys: Record<string, string> = {
  auto: 'vertraege.renewal.auto',
  manual: 'vertraege.renewal.manual',
}

// ─── Currency Options ───────────────────────────────────────────

const CURRENCIES = ['EUR', 'CHF', 'USD'] as const

// ─── Helpers ─────────────────────────────────────────────────────



function formatContractDate(dateStr: string): string {
  if (!dateStr) return 'Unbefristet'
  return formatDate(dateStr.includes('T') ? dateStr : dateStr + 'T00:00:00')
}

function daysUntil(dateStr: string): number {
  if (!dateStr) return Infinity
  const end = new Date(dateStr + 'T00:00:00')
  const now = new Date()
  now.setHours(0, 0, 0, 0)
  return Math.ceil((end.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
}

function daysFromStart(startStr: string, endStr: string): { elapsed: number; total: number; pct: number } {
  if (!endStr) return { elapsed: 0, total: 0, pct: 0 }
  const start = new Date(startStr + 'T00:00:00')
  const end = new Date(endStr + 'T00:00:00')
  const now = new Date()
  now.setHours(0, 0, 0, 0)
  const total = Math.max(1, Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)))
  const elapsed = Math.ceil((now.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))
  const pct = Math.min(100, Math.max(0, (elapsed / total) * 100))
  return { elapsed, total, pct }
}

// ─── Contract Form Dialog ────────────────────────────────────────

interface ContractFormData {
  title: string
  type: ContractType
  partner: string
  contractNumber: string
  startDate: string
  endDate: string
  noticePeriodDays: number
  renewal: 'auto' | 'manual'
  monthlyCost: number
  notes: string
  currency: string
  documents: ContractDocument[]
  reminderDays: number[]
  // Phase 10 — CRM/Finance links
  contactId?: string
  contactName?: string
  dealId?: string
  dealTitle?: string
  invoiceIds: string[]
  invoiceNames: string[]
}

function ContractDialog({
  open,
  onClose,
  initial,
}: {
  open: boolean
  onClose: () => void
  initial?: Contract | null
}) {
  const { t } = useTranslation()
  const addContract = useVertraegeStore((s) => s.addContract)
  const updateContract = useVertraegeStore((s) => s.updateContract)

  // Tenant defaults ("Für alle") + personal reminder override (settings panel).
  const enabledTypes = useVertraegeSettingsStore((s) => s.enabledTypes)
  const tenantNoticePeriodDays = useVertraegeSettingsStore((s) => s.defaultNoticePeriodDays)
  const tenantRenewal = useVertraegeSettingsStore((s) => s.defaultRenewal)
  const tenantReminderDays = useVertraegeSettingsStore((s) => s.tenantReminderDays)
  const personalReminderDays = useVertraegePrefsStore((s) => s.defaultReminderDays)

  const [form, setForm] = useState<ContractFormData>({
    title: initial?.title ?? '',
    type: initial?.type ?? (enabledTypes.includes('servicevertrag') ? 'servicevertrag' : enabledTypes[0]),
    partner: initial?.partner ?? '',
    contractNumber: initial?.contractNumber ?? '',
    startDate: initial?.startDate ?? '',
    endDate: initial?.endDate ?? '',
    noticePeriodDays: initial?.noticePeriodDays ?? tenantNoticePeriodDays,
    renewal: initial?.renewal ?? tenantRenewal,
    monthlyCost: initial?.monthlyCost ?? 0,
    notes: initial?.notes ?? '',
    currency: initial?.currency ?? 'EUR',
    documents: initial?.documents ?? [],
    reminderDays: initial ? (initial.reminderDays ?? []) : (personalReminderDays ?? tenantReminderDays),
    // Phase 10 — CRM/Finance links
    contactId: initial?.contactId,
    contactName: initial?.contactName,
    dealId: initial?.dealId,
    dealTitle: initial?.dealTitle,
    invoiceIds: initial?.invoiceIds ?? [],
    invoiceNames: initial?.invoiceNames ?? [],
  })

  // Document picker state
  const [pickerOpen, setPickerOpen] = useState(false)
  const [pickerSearch, setPickerSearch] = useState('')
  const [uploadProgress, setUploadProgress] = useState<number | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const upload = useDocumentUpload()
  const { data: filesData } = useDocumentFiles()
  const allFiles = filesData?.files ?? []

  // Phase 10 — CRM/Finance link pickers
  const [contactPickerOpen, setContactPickerOpen] = useState(false)
  const [contactSearch, setContactSearch] = useState('')
  const [dealPickerOpen, setDealPickerOpen] = useState(false)
  const [dealSearch, setDealSearch] = useState('')
  const [invoicePickerOpen, setInvoicePickerOpen] = useState(false)
  const [invoiceSearch, setInvoiceSearch] = useState('')
  const { data: contactsData } = useContacts({ page_size: 100 })
  const allContacts = contactsData?.contacts ?? []
  const { data: dealsData } = useDeals({ page_size: 100 })
  const allDeals = dealsData?.deals ?? []
  const { data: invoicesData } = useInvoices({ page_size: 100 })
  const allInvoices = invoicesData?.invoices ?? []

  const filteredContacts = allContacts.filter((c) => {
    if (!contactSearch) return true
    const name = `${(c as { firstName?: string }).firstName ?? ''} ${(c as { lastName?: string }).lastName ?? ''}`.toLowerCase()
    return name.includes(contactSearch.toLowerCase())
  })
  const filteredDeals = allDeals.filter((d) => {
    if (!dealSearch) return true
    return (d as { name?: string }).name?.toLowerCase().includes(dealSearch.toLowerCase())
  })
  const filteredInvoices = allInvoices.filter((inv) => {
    if (!invoiceSearch) return true
    const num = (inv as { invoice_number?: string; number?: string }).invoice_number ?? (inv as { number?: string }).number ?? ''
    return num.toLowerCase().includes(invoiceSearch.toLowerCase())
  })

  const handlePickerSelect = (file: DocumentFile) => {
    const alreadyAdded = form.documents.some((d) => d.fileId === file.id)
    if (alreadyAdded) {
      toast.info(t('vertraege.documents.alreadyAdded'))
      return
    }
    const doc: ContractDocument = {
      fileId: file.id,
      name: file.filename,
      mimeType: file.mime_type,
      size: file.file_size,
      addedAt: new Date().toISOString().split('T')[0],
    }
    setForm((f) => ({ ...f, documents: [...f.documents, doc] }))
    setPickerOpen(false)
    setPickerSearch('')
  }

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    e.target.value = ''
    try {
      setUploadProgress(0)
      const uploaded = await upload.mutateAsync({
        folderId: 'fld-verträge',
        file,
        onProgress: setUploadProgress,
      })
      const doc: ContractDocument = {
        fileId: uploaded.id,
        name: uploaded.filename,
        mimeType: uploaded.mime_type,
        size: uploaded.file_size,
        addedAt: new Date().toISOString().split('T')[0],
      }
      setForm((f) => ({ ...f, documents: [...f.documents, doc] }))
      toast.success(t('vertraege.documents.uploadSuccess', { name: uploaded.filename }))
    } catch (err) {
      toast.error(t('vertraege.documents.uploadError'))
    } finally {
      setUploadProgress(null)
    }
  }

  const handleRemoveDoc = (fileId: string) => {
    setForm((f) => ({ ...f, documents: f.documents.filter((d) => d.fileId !== fileId) }))
  }

  const filteredPickerFiles = allFiles.filter((f) => {
    if (!pickerSearch) return true
    return f.filename.toLowerCase().includes(pickerSearch.toLowerCase())
  })

  const isEdit = !!initial

  const toggleReminder = (days: number) => {
    setForm((f) => ({
      ...f,
      reminderDays: f.reminderDays.includes(days)
        ? f.reminderDays.filter((d) => d !== days)
        : [...f.reminderDays, days].sort((a, b) => a - b),
    }))
  }

  const handleSave = () => {
    if (!form.title.trim()) {
      toast.error(t('vertraege.contractDialog.errorTitle'))
      return
    }
    if (!form.partner.trim()) {
      toast.error(t('vertraege.contractDialog.errorPartner'))
      return
    }
    if (!form.contractNumber.trim()) {
      toast.error(t('vertraege.contractDialog.errorContractNumber'))
      return
    }
    if (!form.startDate) {
      toast.error(t('vertraege.contractDialog.errorStartDate'))
      return
    }

    if (isEdit && initial) {
      updateContract(initial.id, {
        title: form.title,
        type: form.type,
        partner: form.partner,
        contractNumber: form.contractNumber,
        startDate: form.startDate,
        endDate: form.endDate,
        noticePeriodDays: form.noticePeriodDays,
        renewal: form.renewal,
        monthlyCost: form.monthlyCost,
        notes: form.notes,
        currency: form.currency,
        documents: form.documents,
        reminderDays: form.reminderDays,
        contactId: form.contactId,
        contactName: form.contactName,
        dealId: form.dealId,
        dealTitle: form.dealTitle,
        invoiceIds: form.invoiceIds.length > 0 ? form.invoiceIds : undefined,
        invoiceNames: form.invoiceNames.length > 0 ? form.invoiceNames : undefined,
        totalValue: form.endDate
          ? form.monthlyCost * Math.max(1, Math.ceil(
              (new Date(form.endDate).getTime() - new Date(form.startDate).getTime()) / (1000 * 60 * 60 * 24 * 30)
            ))
          : 0,
        history: [
          ...initial.history,
          { date: new Date().toISOString().split('T')[0], action: 'contract_updated', user: 'Aktueller Benutzer' },
        ],
      })
      toast.success(t('vertraege.contractDialog.updateSuccess', { title: form.title }))
    } else {
      const months = form.endDate
        ? Math.max(1, Math.ceil(
            (new Date(form.endDate).getTime() - new Date(form.startDate).getTime()) / (1000 * 60 * 60 * 24 * 30)
          ))
        : 0
      const today = new Date().toISOString().split('T')[0]
      const creationHistory = [
        { date: today, action: 'contract_created' as const, user: 'Aktueller Benutzer' },
      ]
      if (form.contactId && form.contactName) {
        creationHistory.push({ date: today, action: 'contact_linked' as const, meta: form.contactName, user: 'Aktueller Benutzer' })
      }
      if (form.dealId && form.dealTitle) {
        creationHistory.push({ date: today, action: 'deal_linked' as const, meta: form.dealTitle, user: 'Aktueller Benutzer' })
      }
      for (const invName of form.invoiceNames) {
        creationHistory.push({ date: today, action: 'invoice_linked' as const, meta: invName, user: 'Aktueller Benutzer' })
      }
      addContract({
        id: `v-${Date.now()}`,
        title: form.title,
        type: form.type,
        partner: form.partner,
        contractNumber: form.contractNumber,
        startDate: form.startDate,
        endDate: form.endDate,
        noticePeriodDays: form.noticePeriodDays,
        renewal: form.renewal,
        monthlyCost: form.monthlyCost,
        notes: form.notes,
        currency: form.currency,
        documents: form.documents,
        reminderDays: form.reminderDays,
        contactId: form.contactId,
        contactName: form.contactName,
        dealId: form.dealId,
        dealTitle: form.dealTitle,
        invoiceIds: form.invoiceIds.length > 0 ? form.invoiceIds : undefined,
        invoiceNames: form.invoiceNames.length > 0 ? form.invoiceNames : undefined,
        status: 'active',
        totalValue: form.monthlyCost * months,
        history: creationHistory,
      })
      toast.success(t('vertraege.contractDialog.createSuccess', { title: form.title }))
    }
    onClose()
  }

  if (!open) return null

  // Tenant taxonomy: only enabled types are offered; a disabled type stays
  // visible while editing a contract that already uses it.
  const typeOptions: { key: ContractType; labelKey: string; icon: typeof Building }[] = [
    { key: 'mietvertrag', labelKey: 'vertraege.type.mietvertrag', icon: Building },
    { key: 'liefervertrag', labelKey: 'vertraege.type.liefervertrag', icon: Truck },
    { key: 'servicevertrag', labelKey: 'vertraege.type.servicevertrag', icon: Wrench },
    { key: 'arbeitsvertrag', labelKey: 'vertraege.type.arbeitsvertrag', icon: UserCheck },
    { key: 'lizenz', labelKey: 'vertraege.type.lizenz', icon: KeyRound },
    { key: 'versicherung', labelKey: 'vertraege.type.versicherung', icon: Shield },
  ].filter((o) => enabledTypes.includes(o.key) || form.type === o.key)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-lg rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated max-h-[calc(100vh-2rem)] overflow-y-auto">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-semibold text-foreground">
            {isEdit ? t('vertraege.contractDialog.titleEdit') : t('vertraege.contractDialog.titleNew')}
          </h2>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          {/* Titel */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('vertraege.contractDialog.labelTitle')} <span className="text-destructive">*</span>
            </label>
            <input
              type="text"
              value={form.title}
              onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
              placeholder={t('vertraege.contractDialog.placeholderTitle')}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
          </div>

          {/* Typ */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vertraege.contractDialog.labelTyp')}</label>
            <div className="grid grid-cols-3 gap-2">
              {typeOptions.map((opt) => {
                const Icon = opt.icon
                const isActive = form.type === opt.key
                return (
                  <button
                    key={opt.key}
                    onClick={() => setForm((f) => ({ ...f, type: opt.key }))}
                    className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-xs transition-colors ${
                      isActive
                        ? 'border-primary bg-primary-light text-primary font-medium'
                        : 'border-border text-muted-foreground hover:bg-secondary'
                    }`}
                  >
                    <Icon className="h-3.5 w-3.5" />
                    {t(opt.labelKey)}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Partner + Vertragsnummer */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('vertraege.contractDialog.labelPartner')} <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                value={form.partner}
                onChange={(e) => setForm((f) => ({ ...f, partner: e.target.value }))}
                placeholder={t('vertraege.contractDialog.placeholderPartner')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('vertraege.contractDialog.labelContractNumber')} <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                value={form.contractNumber}
                onChange={(e) => setForm((f) => ({ ...f, contractNumber: e.target.value }))}
                placeholder={t('vertraege.contractDialog.placeholderContractNumber')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring font-mono"
              />
            </div>
          </div>

          {/* Start / Ende */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('vertraege.contractDialog.labelStartDate')} <span className="text-destructive">*</span>
              </label>
              <input
                type="date"
                value={form.startDate}
                onChange={(e) => setForm((f) => ({ ...f, startDate: e.target.value }))}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('vertraege.contractDialog.labelEndDate')} <span className="text-xs text-muted-foreground font-normal">({t('vertraege.contractDialog.endDateHint')})</span>
              </label>
              <input
                type="date"
                value={form.endDate}
                onChange={(e) => setForm((f) => ({ ...f, endDate: e.target.value }))}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          </div>

          {/* Kündigungsfrist + Verlaengerung */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('vertraege.contractDialog.labelNoticePeriod')}</label>
              <input
                type="number"
                min={0}
                value={form.noticePeriodDays}
                onChange={(e) => setForm((f) => ({ ...f, noticePeriodDays: Number(e.target.value) }))}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">{t('vertraege.contractDialog.labelRenewal')}</label>
              <div className="flex gap-2">
                {(['auto', 'manual'] as const).map((r) => (
                  <button
                    key={r}
                    onClick={() => setForm((f) => ({ ...f, renewal: r }))}
                    className={`flex-1 flex items-center justify-center gap-1.5 rounded-lg border px-3 py-2 text-xs transition-colors ${
                      form.renewal === r
                        ? 'border-primary bg-primary-light text-primary font-medium'
                        : 'border-border text-muted-foreground hover:bg-secondary'
                    }`}
                  >
                    {r === 'auto' ? <RefreshCw className="h-3 w-3" /> : <CalendarClock className="h-3 w-3" />}
                    {t(renewalLabelKeys[r])}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Waehrung + Monatliche Kosten (10.10) */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('vertraege.contractDialog.labelMonthlyCost')}</label>
            <div className="flex gap-2">
              <div className="flex rounded-lg border border-border overflow-hidden shrink-0">
                {CURRENCIES.map((cur) => (
                  <button
                    key={cur}
                    onClick={() => setForm((f) => ({ ...f, currency: cur }))}
                    className={`px-2.5 py-2 text-xs font-medium transition-colors ${
                      form.currency === cur
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-card text-muted-foreground hover:bg-secondary'
                    }`}
                  >
                    {cur}
                  </button>
                ))}
              </div>
              <input
                type="number"
                min={0}
                step={0.01}
                value={form.monthlyCost}
                onChange={(e) => setForm((f) => ({ ...f, monthlyCost: Number(e.target.value) }))}
                placeholder="0.00"
                className="flex-1 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring tabular-nums"
              />
            </div>
          </div>

          {/* Dokument-Verknüpfung (Phase 7) */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('vertraege.documents.label')} <span className="text-xs text-muted-foreground font-normal">({t('common.optional')})</span>
            </label>

            {/* Verknüpfte Dokumente */}
            {form.documents.length > 0 && (
              <div className="space-y-1.5 mb-2">
                {form.documents.map((doc) => (
                  <div
                    key={doc.fileId}
                    className="flex items-center gap-2 rounded-lg border border-border bg-secondary/30 px-3 py-2"
                  >
                    <FileText className="h-3.5 w-3.5 text-primary shrink-0" />
                    <span className="flex-1 text-sm text-foreground truncate">{doc.name}</span>
                    <button
                      type="button"
                      onClick={() => handleRemoveDoc(doc.fileId)}
                      aria-label={t('vertraege.documents.remove')}
                      className="text-muted-foreground hover:text-destructive transition-colors shrink-0"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Aktionen: Aus Dokumenten wählen + Hochladen */}
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setPickerOpen(true)}
                aria-label={t('vertraege.documents.pickFromLibrary')}
                className="flex-1 flex items-center justify-center gap-2 rounded-lg border border-border px-3 py-2 text-xs text-muted-foreground hover:bg-secondary transition-colors"
              >
                <FileSearch className="h-3.5 w-3.5" />
                {t('vertraege.documents.pickFromLibrary')}
              </button>
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploadProgress !== null}
                aria-label={t('vertraege.documents.upload')}
                className="flex-1 flex items-center justify-center gap-2 rounded-lg border border-border px-3 py-2 text-xs text-muted-foreground hover:bg-secondary transition-colors disabled:opacity-50"
              >
                {uploadProgress !== null ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    {uploadProgress}%
                  </>
                ) : (
                  <>
                    <Upload className="h-3.5 w-3.5" />
                    {t('vertraege.documents.upload')}
                  </>
                )}
              </button>
              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                onChange={handleFileUpload}
                accept=".pdf,.docx,.doc,.xlsx,.xls,.png,.jpg,.jpeg"
              />
            </div>

            {/* Picker Dropdown */}
            {pickerOpen && (
              <div className="rounded-lg border border-border bg-card shadow-lg z-10 relative">
                <div className="p-2 border-b border-border">
                  <input
                    type="text"
                    value={pickerSearch}
                    onChange={(e) => setPickerSearch(e.target.value)}
                    placeholder={t('vertraege.documents.pickerSearch')}
                    className="w-full rounded-md border border-border bg-secondary/30 px-3 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
                    autoFocus
                  />
                </div>
                <div className="max-h-48 overflow-y-auto">
                  {filteredPickerFiles.length === 0 ? (
                    <p className="text-xs text-muted-foreground text-center py-4">{t('vertraege.documents.pickerEmpty')}</p>
                  ) : (
                    filteredPickerFiles.slice(0, 20).map((f) => (
                      <button
                        key={f.id}
                        type="button"
                        onClick={() => handlePickerSelect(f)}
                        className="flex items-center gap-2 w-full px-3 py-2 text-left text-xs hover:bg-secondary transition-colors"
                      >
                        <FileText className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                        <span className="flex-1 truncate text-foreground">{f.filename}</span>
                      </button>
                    ))
                  )}
                </div>
                <div className="p-2 border-t border-border">
                  <button
                    type="button"
                    onClick={() => { setPickerOpen(false); setPickerSearch('') }}
                    className="w-full text-xs text-muted-foreground hover:text-foreground"
                  >
                    {t('common.cancel')}
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Erinnerungen (10.8) */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('vertraege.contractDialog.labelReminders')} <span className="text-xs text-muted-foreground font-normal">({t('vertraege.contractDialog.remindersHint')})</span>
            </label>
            <div className="flex gap-3">
              {[30, 60, 90].map((days) => (
                <label key={days} className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={form.reminderDays.includes(days)}
                    onChange={() => toggleReminder(days)}
                    className="rounded border-border"
                  />
                  <span className="text-sm text-foreground">{t('vertraege.contractDialog.reminderDays', { days })}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Notizen */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('vertraege.contractDialog.labelNotes')} <span className="text-xs text-muted-foreground font-normal">({t('common.optional')})</span>
            </label>
            <textarea
              value={form.notes}
              onChange={(e) => setForm((f) => ({ ...f, notes: e.target.value }))}
              placeholder={t('vertraege.contractDialog.placeholderNotes')}
              rows={3}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
          </div>

          {/* Verknüpfungen — Phase 10 */}
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">
              {t('vertraege.links.label')} <span className="text-xs text-muted-foreground font-normal">({t('common.optional')})</span>
            </label>

            {/* Kontakt */}
            <div className="space-y-1">
              <p className="text-xs text-muted-foreground">{t('vertraege.links.contact')}</p>
              {form.contactId ? (
                <div className="flex items-center gap-2 rounded-lg border border-border bg-secondary/30 px-3 py-2">
                  <User className="h-3.5 w-3.5 text-info shrink-0" />
                  <span className="flex-1 text-sm text-foreground truncate">{form.contactName}</span>
                  <button type="button" onClick={() => setForm((f) => ({ ...f, contactId: undefined, contactName: undefined }))} aria-label={t('vertraege.links.removeContact')} className="text-muted-foreground hover:text-destructive transition-colors shrink-0">
                    <X className="h-3.5 w-3.5" />
                  </button>
                </div>
              ) : (
                <div className="relative">
                  <button type="button" onClick={() => { setContactPickerOpen(true); setDealPickerOpen(false); setInvoicePickerOpen(false) }} className="w-full flex items-center gap-2 rounded-lg border border-dashed border-border px-3 py-2 text-xs text-muted-foreground hover:bg-secondary transition-colors">
                    <User className="h-3.5 w-3.5" />
                    {t('vertraege.links.addContact')}
                  </button>
                  {contactPickerOpen && (
                    <div className="absolute top-full left-0 right-0 z-20 mt-1 rounded-lg border border-border bg-card shadow-lg">
                      <div className="p-2 border-b border-border">
                        <input type="text" value={contactSearch} onChange={(e) => setContactSearch(e.target.value)} placeholder={t('vertraege.links.searchContact')} className="w-full rounded-md border border-border bg-secondary/30 px-3 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring" autoFocus />
                      </div>
                      <div className="max-h-40 overflow-y-auto">
                        {filteredContacts.length === 0 ? (
                          <p className="text-xs text-muted-foreground text-center py-3">{t('vertraege.links.noResults')}</p>
                        ) : filteredContacts.slice(0, 15).map((c) => {
                          const name = `${(c as { firstName?: string }).firstName ?? ''} ${(c as { lastName?: string }).lastName ?? ''}`.trim()
                          return (
                            <button key={(c as { id: string }).id} type="button" onClick={() => { setForm((f) => ({ ...f, contactId: (c as { id: string }).id, contactName: name })); setContactPickerOpen(false); setContactSearch('') }} className="flex items-center gap-2 w-full px-3 py-2 text-left text-xs hover:bg-secondary transition-colors">
                              <User className="h-3 w-3 text-muted-foreground shrink-0" />
                              <span className="flex-1 truncate text-foreground">{name}</span>
                            </button>
                          )
                        })}
                      </div>
                      <div className="p-2 border-t border-border"><button type="button" onClick={() => { setContactPickerOpen(false); setContactSearch('') }} className="w-full text-xs text-muted-foreground hover:text-foreground">{t('common.cancel')}</button></div>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Deal */}
            <div className="space-y-1">
              <p className="text-xs text-muted-foreground">{t('vertraege.links.deal')}</p>
              {form.dealId ? (
                <div className="flex items-center gap-2 rounded-lg border border-border bg-secondary/30 px-3 py-2">
                  <Briefcase className="h-3.5 w-3.5 text-warning shrink-0" />
                  <span className="flex-1 text-sm text-foreground truncate">{form.dealTitle}</span>
                  <button type="button" onClick={() => setForm((f) => ({ ...f, dealId: undefined, dealTitle: undefined }))} aria-label={t('vertraege.links.removeDeal')} className="text-muted-foreground hover:text-destructive transition-colors shrink-0">
                    <X className="h-3.5 w-3.5" />
                  </button>
                </div>
              ) : (
                <div className="relative">
                  <button type="button" onClick={() => { setDealPickerOpen(true); setContactPickerOpen(false); setInvoicePickerOpen(false) }} className="w-full flex items-center gap-2 rounded-lg border border-dashed border-border px-3 py-2 text-xs text-muted-foreground hover:bg-secondary transition-colors">
                    <Briefcase className="h-3.5 w-3.5" />
                    {t('vertraege.links.addDeal')}
                  </button>
                  {dealPickerOpen && (
                    <div className="absolute top-full left-0 right-0 z-20 mt-1 rounded-lg border border-border bg-card shadow-lg">
                      <div className="p-2 border-b border-border">
                        <input type="text" value={dealSearch} onChange={(e) => setDealSearch(e.target.value)} placeholder={t('vertraege.links.searchDeal')} className="w-full rounded-md border border-border bg-secondary/30 px-3 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring" autoFocus />
                      </div>
                      <div className="max-h-40 overflow-y-auto">
                        {filteredDeals.length === 0 ? (
                          <p className="text-xs text-muted-foreground text-center py-3">{t('vertraege.links.noResults')}</p>
                        ) : filteredDeals.slice(0, 15).map((d) => (
                            <button key={(d as { id: string }).id} type="button" onClick={() => { setForm((f) => ({ ...f, dealId: (d as { id: string }).id, dealTitle: (d as { name?: string }).name ?? '' })); setDealPickerOpen(false); setDealSearch('') }} className="flex items-center gap-2 w-full px-3 py-2 text-left text-xs hover:bg-secondary transition-colors">
                              <Briefcase className="h-3 w-3 text-muted-foreground shrink-0" />
                              <span className="flex-1 truncate text-foreground">{(d as { name?: string }).name}</span>
                            </button>
                          ))}
                      </div>
                      <div className="p-2 border-t border-border"><button type="button" onClick={() => { setDealPickerOpen(false); setDealSearch('') }} className="w-full text-xs text-muted-foreground hover:text-foreground">{t('common.cancel')}</button></div>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Rechnungen */}
            <div className="space-y-1">
              <p className="text-xs text-muted-foreground">{t('vertraege.links.invoices')}</p>
              {form.invoiceIds.length > 0 && (
                <div className="space-y-1.5 mb-1">
                  {form.invoiceIds.map((invId, idx) => (
                    <div key={invId} className="flex items-center gap-2 rounded-lg border border-border bg-secondary/30 px-3 py-2">
                      <Receipt className="h-3.5 w-3.5 text-success shrink-0" />
                      <span className="flex-1 text-sm text-foreground truncate">{form.invoiceNames[idx] ?? invId}</span>
                      <button type="button" onClick={() => setForm((f) => ({ ...f, invoiceIds: f.invoiceIds.filter((_, i) => i !== idx), invoiceNames: f.invoiceNames.filter((_, i) => i !== idx) }))} aria-label={t('vertraege.links.removeInvoice')} className="text-muted-foreground hover:text-destructive transition-colors shrink-0">
                        <X className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  ))}
                </div>
              )}
              <div className="relative">
                <button type="button" onClick={() => { setInvoicePickerOpen(true); setContactPickerOpen(false); setDealPickerOpen(false) }} className="w-full flex items-center gap-2 rounded-lg border border-dashed border-border px-3 py-2 text-xs text-muted-foreground hover:bg-secondary transition-colors">
                  <Receipt className="h-3.5 w-3.5" />
                  {t('vertraege.links.addInvoice')}
                </button>
                {invoicePickerOpen && (
                  <div className="absolute top-full left-0 right-0 z-20 mt-1 rounded-lg border border-border bg-card shadow-lg">
                    <div className="p-2 border-b border-border">
                      <input type="text" value={invoiceSearch} onChange={(e) => setInvoiceSearch(e.target.value)} placeholder={t('vertraege.links.searchInvoice')} className="w-full rounded-md border border-border bg-secondary/30 px-3 py-1.5 text-xs text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring" autoFocus />
                    </div>
                    <div className="max-h-40 overflow-y-auto">
                      {filteredInvoices.length === 0 ? (
                        <p className="text-xs text-muted-foreground text-center py-3">{t('vertraege.links.noResults')}</p>
                      ) : filteredInvoices.slice(0, 15).map((inv) => {
                        const invNum = (inv as { invoice_number?: string; number?: string }).invoice_number ?? (inv as { number?: string }).number ?? inv.id
                        const already = form.invoiceIds.includes(inv.id)
                        return (
                          <button key={inv.id} type="button" disabled={already} onClick={() => { if (already) return; setForm((f) => ({ ...f, invoiceIds: [...f.invoiceIds, inv.id], invoiceNames: [...f.invoiceNames, invNum] })); setInvoicePickerOpen(false); setInvoiceSearch('') }} className={`flex items-center gap-2 w-full px-3 py-2 text-left text-xs transition-colors ${already ? 'opacity-40 cursor-not-allowed' : 'hover:bg-secondary'}`}>
                            <Receipt className="h-3 w-3 text-muted-foreground shrink-0" />
                            <span className="flex-1 truncate text-foreground">{invNum}</span>
                          </button>
                        )
                      })}
                    </div>
                    <div className="p-2 border-t border-border"><button type="button" onClick={() => { setInvoicePickerOpen(false); setInvoiceSearch('') }} className="w-full text-xs text-muted-foreground hover:text-foreground">{t('common.cancel')}</button></div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 mt-6">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            {isEdit ? t('common.save') : t('vertraege.contractDialog.buttonCreate')}
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Termination Dialog ──────────────────────────────────────────

function TerminationDialog({
  open,
  onClose,
  contract,
}: {
  open: boolean
  onClose: () => void
  contract: Contract | null
}) {
  const { t } = useTranslation()
  const terminateContract = useVertraegeStore((s) => s.terminateContract)
  const [terminationDate, setTerminationDate] = useState('')
  const [reason, setReason] = useState('')
  const [confirmed, setConfirmed] = useState(false)

  const handleTerminate = () => {
    if (!terminationDate) {
      toast.error(t('vertraege.terminationDialog.errorDate'))
      return
    }
    if (!reason.trim()) {
      toast.error(t('vertraege.terminationDialog.errorReason'))
      return
    }
    if (!confirmed) {
      toast.error(t('vertraege.terminationDialog.errorConfirm'))
      return
    }
    if (contract) {
      terminateContract(contract.id, reason, terminationDate)
      toast.success(t('vertraege.terminationDialog.success', { title: contract.title }))
    }
    setTerminationDate('')
    setReason('')
    setConfirmed(false)
    onClose()
  }

  if (!open || !contract) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl glass-elevated">
        <div className="flex items-center justify-between mb-5">
          <div>
            <h2 className="text-lg font-semibold text-foreground">{t('vertraege.terminationDialog.title')}</h2>
            <p className="text-xs text-muted-foreground mt-0.5">{contract.title}</p>
          </div>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="rounded-lg border border-warning/30 bg-warning-light/30 p-3 mb-4">
          <div className="flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 text-warning mt-0.5 shrink-0" />
            <div className="text-xs text-warning">
              <p className="font-medium">{t('vertraege.terminationDialog.warning')}</p>
              <p>{t('vertraege.terminationDialog.warningDesc')}</p>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          {/* Kündigungsdatum */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('vertraege.terminationDialog.labelDate')} <span className="text-destructive">*</span>
            </label>
            <input
              type="date"
              value={terminationDate}
              onChange={(e) => setTerminationDate(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
            {contract.noticePeriodDays > 0 && (
              <p className="text-xs text-muted-foreground">
                {t('vertraege.terminationDialog.noticePeriodHint', { days: contract.noticePeriodDays })}
              </p>
            )}
          </div>

          {/* Grund */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">
              {t('vertraege.terminationDialog.labelReason')} <span className="text-destructive">*</span>
            </label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder={t('vertraege.terminationDialog.placeholderReason')}
              rows={3}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
          </div>

          {/* Bestätigung */}
          <label className="flex items-start gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={confirmed}
              onChange={(e) => setConfirmed(e.target.checked)}
              className="mt-0.5 rounded border-border"
            />
            <span className="text-sm text-foreground">
              {t('vertraege.terminationDialog.confirmText', { title: contract.title })}
            </span>
          </label>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 mt-6">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleTerminate}
            disabled={!confirmed}
            className="rounded-lg bg-destructive px-4 py-2 text-sm text-white hover:bg-destructive/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {t('vertraege.terminationDialog.buttonSubmit')}
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Contract Duration Bar ───────────────────────────────────────

function DurationBar({ contract }: { contract: Contract }) {
  const { t } = useTranslation()
  if (!contract.endDate) {
    return <span className="text-xs text-muted-foreground">{t('vertraege.durationBar.unbefristet')}</span>
  }

  const { pct } = daysFromStart(contract.startDate, contract.endDate)
  const remaining = daysUntil(contract.endDate)
  const isExpiring = remaining <= 90 && remaining > 0
  const isExpired = remaining <= 0

  const barColor = isExpired
    ? 'bg-error'
    : isExpiring
      ? 'bg-warning'
      : 'bg-primary'

  return (
    <TooltipProvider delayDuration={200}>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="space-y-1 min-w-[100px]">
            <div className="relative h-1.5 w-full rounded-full bg-secondary overflow-hidden">
              <div
                className={`absolute inset-y-0 left-0 rounded-full ${barColor} transition-all`}
                style={{ width: `${pct}%` }}
              />
            </div>
            <div className="flex justify-between text-[10px] text-muted-foreground">
              <span>{formatDate(contract.startDate)}</span>
              <span>{formatDate(contract.endDate)}</span>
            </div>
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <p className="text-xs">
            {remaining > 0
              ? t('vertraege.durationBar.remaining', { days: remaining, pct: Math.round(pct) })
              : t('vertraege.durationBar.expired', { days: Math.abs(remaining) })}
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

// ─── Stat Card ───────────────────────────────────────────────────

function StatCard({
  icon: Icon,
  label,
  value,
  iconColor,
  iconBg,
}: {
  icon: typeof FileSignature
  label: string
  value: string | number
  iconColor: string
  iconBg: string
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center gap-3 mb-2">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${iconBg}`}>
          <Icon className={`h-5 w-5 ${iconColor}`} />
        </div>
      </div>
      <p className="text-2xl font-semibold text-foreground tabular-nums">{value}</p>
      <p className="text-xs text-muted-foreground mt-1">{label}</p>
    </div>
  )
}

// ─── Template Card (10.9) ────────────────────────────────────────

function TemplateCard({
  template,
  onUse,
}: {
  template: ContractTemplate
  onUse: () => void
}) {
  const { t } = useTranslation()
  const typeConf = contractTypeConfig[template.type]
  const TypeIcon = typeConf.icon

  return (
    <div className="rounded-lg border border-border bg-card p-4 hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between mb-3">
        <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${typeConf.colorClass}`}>
          <TypeIcon className="h-3 w-3" />
          {t(typeConf.labelKey)}
        </span>
      </div>
      <h3 className="text-sm font-semibold text-foreground mb-1">{template.name}</h3>
      <p className="text-xs text-muted-foreground mb-3 line-clamp-2">{template.description}</p>
      <div className="flex flex-wrap gap-2 mb-3">
        {template.defaultDuration && (
          <span className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
            <Clock className="h-3 w-3" />
            {t('vertraege.template.months', { count: template.defaultDuration })}
          </span>
        )}
        {!template.defaultDuration && (
          <span className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
            <Clock className="h-3 w-3" />
            {t('vertraege.template.unbefristet')}
          </span>
        )}
        <span className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
          <CalendarClock className="h-3 w-3" />
          {t('vertraege.template.noticePeriod', { days: template.defaultNoticePeriodDays })}
        </span>
        <span className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
          <RefreshCw className="h-3 w-3" />
          {t(renewalLabelKeys[template.defaultRenewal])}
        </span>
        {template.defaultMonthlyCost != null && (
          <span className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
            <Coins className="h-3 w-3" />
            {formatCurrency(template.defaultMonthlyCost, 'EUR')}{t('vertraege.template.perMonth')}
          </span>
        )}
      </div>
      <button
        onClick={onUse}
        className="w-full flex items-center justify-center gap-2 rounded-lg border border-primary bg-primary-light px-3 py-2 text-xs font-medium text-primary hover:bg-primary hover:text-primary-foreground transition-colors"
      >
        <Plus className="h-3.5 w-3.5" />
        {t('vertraege.template.buttonUse')}
      </button>
    </div>
  )
}

// ─── Main Page ───────────────────────────────────────────────────

export default function VertraegePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { contracts, contractTemplates, deleteContract, updateContract } = useVertraegeStore()

  // Personal prefs (settings panel): default tab on open + table density.
  const defaultTab = useVertraegePrefsStore((s) => s.defaultTab)
  const density = useVertraegePrefsStore((s) => s.density)

  const [tab, setTab] = useState<TabKey>(defaultTab)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('all')

  // Detail panel — store the ID, not the snapshot, so mutations are reflected live.
  const [selectedContractId, setSelectedContractId] = useState<string | null>(null)
  const selectedContract = contracts.find((c) => c.id === selectedContractId) ?? null

  // Dialogs
  const [contractDialogOpen, setContractDialogOpen] = useState(false)
  const [editContract, setEditContract] = useState<Contract | null>(null)
  const [terminationDialogOpen, setTerminationDialogOpen] = useState(false)
  const [terminationContract, setTerminationContract] = useState<Contract | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<Contract | null>(null)
  const [eSignaturContract, setESignaturContract] = useState<Contract | null>(null)

  // Phase 7: Dokument-Vorschau und Versions-Panel
  const [previewDoc, setPreviewDoc] = useState<{ file: DocumentFile; open: boolean } | null>(null)
  const [versionsDoc, setVersionsDoc] = useState<{ fileId: string; name: string; open: boolean } | null>(null)

  // ── Computed ──

  const activeContracts = useMemo(
    () => contracts.filter((c) => c.status === 'active'),
    [contracts],
  )

  const expiringContracts = useMemo(
    () => contracts.filter((c) => {
      if (c.status === 'terminated' || c.status === 'expired') return false
      if (!c.endDate) return false
      const days = daysUntil(c.endDate)
      return days <= 90 && days > 0
    }),
    [contracts],
  )

  const archivedContracts = useMemo(
    () => contracts.filter((c) => c.status === 'terminated' || c.status === 'expired'),
    [contracts],
  )

  const expiring30 = useMemo(
    () => contracts.filter((c) => {
      if (c.status === 'terminated' || c.status === 'expired') return false
      if (!c.endDate) return false
      const days = daysUntil(c.endDate)
      return days <= 30 && days > 0
    }),
    [contracts],
  )

  const totalMonthlyCost = useMemo(
    () => activeContracts.reduce((sum, c) => sum + c.monthlyCost, 0),
    [activeContracts],
  )

  const totalContractValue = useMemo(
    () => contracts.filter((c) => c.status !== 'terminated' && c.status !== 'expired').reduce((sum, c) => sum + c.totalValue, 0),
    [contracts],
  )

  // Get base list for current tab
  const baseContracts = useMemo(() => {
    if (tab === 'aktiv') return activeContracts
    if (tab === 'auslaufend') return expiringContracts
    if (tab === 'archiv') return archivedContracts
    return [] // vorlagen tab doesn't use baseContracts
  }, [tab, activeContracts, expiringContracts, archivedContracts])

  // Filter + search
  const filteredContracts = useMemo(() => {
    let result = [...baseContracts]

    if (typeFilter !== 'all') {
      result = result.filter((c) => c.type === typeFilter)
    }

    if (search) {
      const q = search.toLowerCase()
      result = result.filter(
        (c) =>
          c.title.toLowerCase().includes(q) ||
          c.partner.toLowerCase().includes(q) ||
          c.contractNumber.toLowerCase().includes(q),
      )
    }

    return result.sort((a, b) => {
      // Sort by end date ascending (soonest first)
      if (a.endDate && b.endDate) return a.endDate.localeCompare(b.endDate)
      if (!a.endDate) return 1
      if (!b.endDate) return -1
      return 0
    })
  }, [baseContracts, typeFilter, search])

  // ── Actions ──

  const openContractDialog = (contract?: Contract) => {
    setEditContract(contract ?? null)
    setContractDialogOpen(true)
  }

  const openTerminationDialog = (contract: Contract) => {
    setTerminationContract(contract)
    setTerminationDialogOpen(true)
  }

  const openFromTemplate = (template: ContractTemplate) => {
    // Create a "pseudo" initial contract pre-filled from template defaults
    const today = new Date().toISOString().split('T')[0]
    let endDate = ''
    if (template.defaultDuration) {
      const end = new Date()
      end.setMonth(end.getMonth() + Number(template.defaultDuration))
      endDate = end.toISOString().split('T')[0]
    }
    const prefilled: Contract = {
      id: '',
      contractNumber: '',
      title: '',
      partner: '',
      type: template.type,
      status: 'active',
      startDate: today,
      endDate,
      noticePeriodDays: template.defaultNoticePeriodDays,
      renewal: template.defaultRenewal,
      monthlyCost: template.defaultMonthlyCost ?? 0,
      totalValue: 0,
      notes: '',
      templateId: template.id,
      currency: 'EUR',
      documents: [],
      history: [],
    }
    setEditContract(prefilled)
    setContractDialogOpen(true)
  }

  const getContractActions = (contract: Contract) => [
    { label: t('vertraege.actions.showDetails'), icon: Eye, onClick: () => setSelectedContractId(contract.id) },
    { label: t('common.edit'), icon: Edit, onClick: () => openContractDialog(contract) },
    { label: t('vertraege.actions.signature'), icon: Pen, onClick: () => setESignaturContract(contract) },
    ...(contract.status === 'active' || contract.status === 'expiring'
      ? [{ label: t('vertraege.actions.terminate'), icon: Ban, onClick: () => openTerminationDialog(contract), separator: true }]
      : []),
    { separator: true, label: t('common.delete'), icon: Trash2, variant: 'destructive' as const, onClick: () => setConfirmDelete(contract) },
  ]

  const handleDelete = (contract: Contract) => {
    deleteContract(contract.id)
    setConfirmDelete(null)
    if (selectedContract?.id === contract.id) setSelectedContractId(null)
    toast.success(t('vertraege.delete.success', { title: contract.title }))
  }

  // Tab labels for empty states
  const tabEmptyConfig: Record<string, { icon: typeof FileSignature; titleKey: string; descKey: string }> = {
    aktiv: { icon: FileSignature, titleKey: 'vertraege.empty.aktiv.title', descKey: 'vertraege.empty.aktiv.desc' },
    auslaufend: { icon: Clock, titleKey: 'vertraege.empty.auslaufend.title', descKey: 'vertraege.empty.auslaufend.desc' },
    archiv: { icon: Archive, titleKey: 'vertraege.empty.archiv.title', descKey: 'vertraege.empty.archiv.desc' },
    vorlagen: { icon: LayoutTemplate, titleKey: 'vertraege.empty.vorlagen.title', descKey: 'vertraege.empty.vorlagen.desc' },
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <PageHeader
        title={t('vertraege.page.title')}
        description={`${t('vertraege.page.description', { count: activeContracts.length })}${expiringContracts.length > 0 ? t('vertraege.page.descriptionExpiring', { count: expiringContracts.length }) : ''}`}
        icon={FileSignature}
        moduleId="verträge"
        className="mb-6"
        actions={
          <button
            onClick={() => openContractDialog()}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            {t('vertraege.page.buttonNew')}
          </button>
        }
      />

      {/* Stats Row */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard
          icon={FileSignature}
          label={t('vertraege.stats.activeContracts')}
          value={activeContracts.length}
          iconColor="text-primary"
          iconBg="bg-primary-light"
        />
        <StatCard
          icon={Coins}
          label={t('vertraege.stats.monthlyCost')}
          value={formatCurrency(totalMonthlyCost, 'EUR')}
          iconColor="text-success"
          iconBg="bg-success-light"
        />
        <StatCard
          icon={AlertTriangle}
          label={t('vertraege.stats.expiring30')}
          value={expiring30.length}
          iconColor="text-warning"
          iconBg="bg-warning-light"
        />
        <StatCard
          icon={Wallet}
          label={t('vertraege.stats.totalValue')}
          value={formatCurrency(totalContractValue, 'EUR')}
          iconColor="text-info"
          iconBg="bg-info-light"
        />
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 border-b border-border mb-6">
        {([
          { key: 'aktiv' as TabKey, labelKey: 'vertraege.tab.aktiv', count: activeContracts.length },
          { key: 'auslaufend' as TabKey, labelKey: 'vertraege.tab.auslaufend', count: expiringContracts.length },
          { key: 'archiv' as TabKey, labelKey: 'vertraege.tab.archiv', count: archivedContracts.length },
          { key: 'vorlagen' as TabKey, labelKey: 'vertraege.tab.vorlagen', count: contractTemplates.length },
        ]).map((tabItem) => (
          <button
            key={tabItem.key}
            onClick={() => { setTab(tabItem.key); setSearch(''); setTypeFilter('all') }}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === tabItem.key ? 'border-primary text-primary font-medium tab-accent-active' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t(tabItem.labelKey)} ({tabItem.count})
          </button>
        ))}
      </div>

      {/* ─── VORLAGEN TAB (10.9) ─────────────────────────────── */}
      {tab === 'vorlagen' ? (
        contractTemplates.length === 0 ? (
          <EmptyState
            icon={LayoutTemplate}
            title={t('vertraege.empty.vorlagen.title')}
            description={t('vertraege.empty.vorlagen.desc')}
          />
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {contractTemplates.map((template) => (
              <TemplateCard
                key={template.id}
                template={template}
                onUse={() => openFromTemplate(template)}
              />
            ))}
          </div>
        )
      ) : (
        <>
          {/* Search + Filters */}
          <div className="flex flex-wrap items-center gap-3 mb-4">
            <div className="relative flex-1 min-w-[200px] max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder={t('vertraege.search.placeholder')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>

            <div className="relative">
              <select
                value={typeFilter}
                onChange={(e) => setTypeFilter(e.target.value as TypeFilter)}
                className="appearance-none rounded-lg border border-border bg-card pl-3 pr-8 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              >
                <option value="all">{t('vertraege.search.allTypes')}</option>
                <option value="mietvertrag">{t('vertraege.type.mietvertrag')}</option>
                <option value="liefervertrag">{t('vertraege.type.liefervertrag')}</option>
                <option value="servicevertrag">{t('vertraege.type.servicevertrag')}</option>
                <option value="arbeitsvertrag">{t('vertraege.type.arbeitsvertrag')}</option>
                <option value="lizenz">{t('vertraege.type.lizenz')}</option>
                <option value="versicherung">{t('vertraege.type.versicherung')}</option>
              </select>
              <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            </div>

            {(typeFilter !== 'all' || search) && (
              <button
                onClick={() => { setTypeFilter('all'); setSearch('') }}
                className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Filter className="h-3.5 w-3.5" />
                {t('common.resetFilters')}
              </button>
            )}
          </div>

          {/* ─── CONTRACT TABLE ──────────────────────────────────── */}
          {filteredContracts.length === 0 ? (
            <EmptyState
              icon={tabEmptyConfig[tab].icon}
              title={t(tabEmptyConfig[tab].titleKey)}
              description={
                search || typeFilter !== 'all'
                  ? t('vertraege.empty.filterDesc')
                  : t(tabEmptyConfig[tab].descKey)
              }
            />
          ) : (
            <TooltipProvider delayDuration={300}>
              <div className="overflow-x-auto rounded-lg border border-border">
                <table className={`w-full text-sm ${density === 'compact' ? '[&_td]:py-1.5 [&_th]:py-2' : ''}`}>
                  <thead>
                    <tr className="border-b border-border bg-card">
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vertraege.table.vertragsnr')}</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vertraege.table.title')}</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vertraege.table.partner')}</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vertraege.table.typ')}</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vertraege.table.laufzeit')}</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground">{t('vertraege.table.monthlyCost')}</th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">{t('vertraege.table.status')}</th>
                      <th className="px-4 py-3 text-right font-medium text-muted-foreground w-12"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredContracts.map((contract) => {
                      const typeConf = contractTypeConfig[contract.type]
                      const statusConf = statusConfig[contract.status]
                      const TypeIcon = typeConf.icon
                      const isSelected = selectedContract?.id === contract.id
                      const hasReminders = contract.reminderDays && contract.reminderDays.length > 0

                      return (
                        <tr
                          key={contract.id}
                          onClick={() => setSelectedContractId(contract.id)}
                          className={`border-b border-border-muted last:border-0 hover:bg-secondary/50 transition-colors cursor-pointer ${
                            isSelected ? 'bg-primary-light/30' : ''
                          }`}
                        >
                          <td className="px-4 py-3 text-muted-foreground font-mono text-xs">
                            {contract.contractNumber}
                          </td>
                          <td className="px-4 py-3 font-medium text-foreground max-w-[200px] truncate">
                            <span className="flex items-center gap-1.5">
                              {contract.title}
                              {hasReminders && (
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <Bell className="h-3 w-3 text-warning shrink-0" />
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    <p className="text-xs">{t('vertraege.table.reminders', { days: contract.reminderDays!.join('/') })}</p>
                                  </TooltipContent>
                                </Tooltip>
                              )}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-muted-foreground">
                            {contract.partner}
                          </td>
                          <td className="px-4 py-3">
                            <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${typeConf.colorClass}`}>
                              <TypeIcon className="h-3 w-3" />
                              {t(typeConf.labelKey)}
                            </span>
                          </td>
                          <td className="px-4 py-3">
                            <DurationBar contract={contract} />
                          </td>
                          <td className="px-4 py-3 text-right text-foreground tabular-nums">
                            {formatCurrency(contract.monthlyCost, contract.currency ?? 'EUR')}
                          </td>
                          <td className="px-4 py-3">
                            <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${statusConf.colorClass}`}>
                              {t(statusConf.labelKey)}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                            <ItemActions items={getContractActions(contract)} />
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            </TooltipProvider>
          )}
        </>
      )}

      {/* ─── DETAIL PANEL ────────────────────────────────────── */}
      <DetailPanel
        open={!!selectedContract}
        onClose={() => setSelectedContractId(null)}
        title={selectedContract?.title}
        subtitle={selectedContract ? `${selectedContract.contractNumber} · ${selectedContract.partner}` : undefined}
        badge={
          selectedContract ? (
            <span className={`ml-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
              statusConfig[selectedContract.status].colorClass
            }`}>
              {t(statusConfig[selectedContract.status].labelKey)}
            </span>
          ) : undefined
        }
        width="w-[400px]"
        footer={
          selectedContract ? (
            <div className="flex gap-2">
              <button
                onClick={() => openContractDialog(selectedContract)}
                className="flex-1 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                {t('vertraege.detail.buttonEdit')}
              </button>
              <button
                onClick={() => setESignaturContract(selectedContract)}
                className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-secondary transition-colors"
              >
                <Pen className="h-3.5 w-3.5" />
                {t('vertraege.detail.buttonSignature')}
              </button>
              {(selectedContract.status === 'active' || selectedContract.status === 'expiring') && (
                <button
                  onClick={() => openTerminationDialog(selectedContract)}
                  className="flex-1 rounded-lg bg-destructive px-3 py-2 text-sm text-white hover:bg-destructive/90 transition-colors"
                >
                  {t('vertraege.detail.buttonTerminate')}
                </button>
              )}
            </div>
          ) : undefined
        }
      >
        {selectedContract && (
          <DetailPanelContent
            contract={selectedContract}
            onPreviewDoc={(file) => setPreviewDoc({ file, open: true })}
            onVersionsDoc={(fileId, name) => setVersionsDoc({ fileId, name, open: true })}
            onNavigate={(path) => navigate(path)}
          />
        )}
      </DetailPanel>

      {/* ─── DIALOGS ─────────────────────────────────────────── */}
      <ContractDialog
        open={contractDialogOpen}
        onClose={() => {
          setContractDialogOpen(false)
          setEditContract(null)
        }}
        initial={editContract}
      />

      <TerminationDialog
        open={terminationDialogOpen}
        onClose={() => {
          setTerminationDialogOpen(false)
          setTerminationContract(null)
        }}
        contract={terminationContract}
      />

      {eSignaturContract && (
        <ESignaturDialog
          open={!!eSignaturContract}
          onClose={() => setESignaturContract(null)}
          contract={eSignaturContract}
          onUpdateSigners={(_signers) => {
            // Signers mutations are handled directly in ESignaturDialog via the store.
            // This callback is kept for external callers that still pass signers.
          }}
        />
      )}

      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={() => setConfirmDelete(null)}
        title={t('vertraege.confirm.deleteTitle')}
        description={t('vertraege.confirm.deleteDesc', { title: confirmDelete?.title ?? '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={() => confirmDelete && handleDelete(confirmDelete)}
      />

      {/* Phase 7: Dokument-Vorschau */}
      <FilePreviewModal
        file={previewDoc?.file ?? null}
        open={previewDoc?.open ?? false}
        onOpenChange={(open) => {
          if (!open) setPreviewDoc(null)
        }}
      />

      {/* Phase 7: Versions-Panel */}
      {versionsDoc && (
        <VersionHistoryPanel
          fileId={versionsDoc.fileId}
          fileName={versionsDoc.name}
          open={versionsDoc.open}
          onClose={() => setVersionsDoc(null)}
        />
      )}
    </div>
  )
}

// ─── Audit-Log helpers ───────────────────────────────────────────

/** Stable action codes that map to i18n keys. */
const HISTORY_ACTION_CODES = [
  'contract_created',
  'contract_updated',
  'contract_terminated',
  'contract_signed',
  'reminder_triggered',
  'document_added',
  'document_removed',
  'contact_linked',
  'contact_unlinked',
  'deal_linked',
  'deal_unlinked',
  'invoice_linked',
  'invoice_unlinked',
] as const satisfies readonly string[]

type HistoryActionCode = typeof HISTORY_ACTION_CODES[number]

function isActionCode(action: string): action is HistoryActionCode {
  return (HISTORY_ACTION_CODES as readonly string[]).includes(action)
}

function getHistoryIcon(action: string) {
  switch (action as HistoryActionCode) {
    case 'contract_created':
      return FilePlus
    case 'contract_updated':
      return FilePen
    case 'contract_terminated':
      return FileX
    case 'contract_signed':
      return Pen
    case 'reminder_triggered':
      return BellRing
    case 'document_added':
      return FileText
    case 'document_removed':
      return FileX
    default:
      return History
  }
}

function getHistoryIconColor(action: string): string {
  switch (action as HistoryActionCode) {
    case 'contract_created':
      return 'text-success bg-success-light'
    case 'contract_updated':
      return 'text-primary bg-primary-light'
    case 'contract_terminated':
      return 'text-error bg-error-light'
    case 'contract_signed':
      return 'text-info bg-info-light'
    case 'reminder_triggered':
      return 'text-warning bg-warning-light'
    case 'document_added':
      return 'text-info bg-info-light'
    case 'document_removed':
      return 'text-muted-foreground bg-secondary'
    default:
      return 'text-muted-foreground bg-secondary'
  }
}

// ─── Audit Log Feed ──────────────────────────────────────────────

function AuditLogFeed({ history }: { history: import('@/stores/vertraege').ContractHistoryEntry[] }) {
  const { t } = useTranslation()

  const sorted = [...history].reverse()

  return (
    <div className="space-y-2">
      <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
        {t('vertraege.detail.history', { count: history.length })}
      </h4>
      {sorted.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-5 text-center">
          <History className="h-6 w-6 text-muted-foreground/40 mb-2" />
          <p className="text-xs text-muted-foreground">{t('vertraege.detail.historyEmpty')}</p>
        </div>
      ) : (
        <div className="space-y-1">
          {sorted.map((entry, idx) => {
            const Icon = getHistoryIcon(entry.action)
            const colorClass = getHistoryIconColor(entry.action)
            const label = isActionCode(entry.action)
              ? t(`vertraege.history.action.${entry.action}` as const, { meta: entry.meta ?? '' })
              : entry.action // legacy free-form text fallback

            return (
              <div
                key={idx}
                className="flex items-start gap-3 rounded-md px-2 py-2 transition-colors hover:bg-secondary/40"
              >
                <div className={`mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full ${colorClass}`}>
                  <Icon className="h-3.5 w-3.5" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className={`text-sm ${idx === 0 ? 'font-medium text-foreground' : 'text-muted-foreground'}`}>
                    {label}
                  </p>
                  {entry.meta && (
                    <p className="text-xs text-muted-foreground/70 mt-0.5 leading-snug truncate" title={entry.meta}>
                      {entry.meta}
                    </p>
                  )}
                  <p className="text-[10px] text-muted-foreground mt-0.5">
                    {formatDate(entry.date)}&ensp;&middot;&ensp;{entry.user}
                  </p>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

// ─── Reminder Schedule ───────────────────────────────────────────

function ReminderSchedule({ contract }: { contract: import('@/stores/vertraege').Contract }) {
  const { t } = useTranslation()

  const { reminderDays, endDate } = contract

  if (!reminderDays || reminderDays.length === 0) {
    return (
      <div className="space-y-2">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          {t('vertraege.detail.erinnerungen')}
        </h4>
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-4 text-center">
          <Bell className="h-5 w-5 text-muted-foreground/40 mb-1.5" />
          <p className="text-xs text-muted-foreground">{t('vertraege.detail.reminderNone')}</p>
        </div>
      </div>
    )
  }

  const today = new Date()
  today.setHours(0, 0, 0, 0)

  const schedule = reminderDays
    .slice()
    .sort((a, b) => b - a) // largest offset first = earliest date
    .map((days) => {
      if (!endDate) {
        return { days, dateStr: null, isPast: false }
      }
      const target = new Date(endDate + 'T00:00:00')
      target.setDate(target.getDate() - days)
      const isPast = target.getTime() < today.getTime()
      return { days, dateStr: target.toISOString().split('T')[0], isPast }
    })

  return (
    <div className="space-y-2">
      <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
        {t('vertraege.detail.erinnerungen')}
      </h4>
      {!endDate ? (
        <p className="text-xs text-muted-foreground">{t('vertraege.detail.reminderUnbefristet')}</p>
      ) : (
        <div className="space-y-1.5">
          {schedule.map(({ days, dateStr, isPast }) => (
            <div
              key={days}
              className={`flex items-center justify-between rounded-lg border px-3 py-2 ${
                isPast
                  ? 'border-border bg-secondary/20 opacity-50'
                  : 'border-warning/30 bg-warning-light/30'
              }`}
            >
              <div className="flex items-center gap-2">
                <Bell className={`h-3.5 w-3.5 shrink-0 ${isPast ? 'text-muted-foreground' : 'text-warning'}`} />
                <span className={`text-xs font-medium ${isPast ? 'text-muted-foreground' : 'text-foreground'}`}>
                  {t('vertraege.detail.reminderDaysBefore', { days })}
                </span>
              </div>
              {dateStr && (
                <span className={`text-[10px] tabular-nums ${isPast ? 'text-muted-foreground line-through' : 'text-muted-foreground'}`}>
                  {formatDate(dateStr)}
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ─── Detail Panel Content ────────────────────────────────────────

// Adapter: ContractDocument → minimal DocumentFile shape for FilePreviewModal
function contractDocToDocumentFile(doc: ContractDocument): DocumentFile {
  return {
    id: doc.fileId,
    folder_id: 'fld-verträge',
    filename: doc.name,
    mime_type: doc.mimeType ?? 'application/octet-stream',
    file_size: doc.size ?? 0,
    storage_key: `contract-doc/${doc.fileId}`,
    thumbnail_key: null,
    current_version: 1,
    owner_id: '',
    is_favorite: false,
    is_deleted: false,
    tags: [],
    created_at: doc.addedAt,
    updated_at: doc.addedAt,
  }
}

function getMimeIcon(mimeType: string) {
  if (mimeType === 'application/pdf') return FileText
  if (mimeType.startsWith('image/')) return FileText
  if (mimeType.includes('spreadsheet') || mimeType.includes('excel')) return FileText
  return FileText
}

function formatBytes(bytes: number): string {
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(1)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(1)} MB`
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${bytes} B`
}

// ─── Signer Section ──────────────────────────────────────────────

function SignerSection({ contract }: { contract: Contract }) {
  const { t } = useTranslation()
  const signers = contract.signers ?? []

  return (
    <div className="space-y-2">
      <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
        {t('vertraege.detail.signaturen')}
        {signers.length > 0 && (
          <span className="ml-1.5 text-muted-foreground font-normal normal-case">({signers.length})</span>
        )}
      </h4>

      {signers.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-4 text-center">
          <Pen className="h-5 w-5 text-muted-foreground/40 mb-1.5" />
          <p className="text-xs text-muted-foreground">{t('vertraege.detail.signaturen.none')}</p>
        </div>
      ) : (
        <div className="space-y-2">
          {signers.map((signer, idx) => {
            const statusConf: { labelKey: string; colorClass: string } = {
              pending: { labelKey: 'vertraege.esignatur.status.pending', colorClass: 'bg-secondary text-muted-foreground' },
              sent: { labelKey: 'vertraege.esignatur.status.sent', colorClass: 'bg-info-light text-info' },
              viewed: { labelKey: 'vertraege.esignatur.status.viewed', colorClass: 'bg-warning-light text-warning' },
              signed: { labelKey: 'vertraege.esignatur.status.signed', colorClass: 'bg-success-light text-success' },
              declined: { labelKey: 'vertraege.esignatur.status.declined', colorClass: 'bg-error-light text-error' },
            }[signer.status]

            return (
              <div key={idx} className="rounded-lg border border-border bg-secondary/20 px-3 py-2.5 space-y-1.5">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium text-foreground truncate">{signer.name}</span>
                  <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium shrink-0 ${statusConf.colorClass}`}>
                    {t(statusConf.labelKey)}
                  </span>
                </div>
                <p className="text-[10px] text-muted-foreground">{signer.email}</p>
                {signer.status === 'signed' && signer.signedAt && (
                  <p className="text-[10px] text-muted-foreground">
                    {t('vertraege.detail.signaturen.signedAt', { date: formatDate(signer.signedAt) })}
                  </p>
                )}
                {/* Thumbnail — only for canvas signatures */}
                {signer.status === 'signed' && signer.signatureDataUrl && (
                  <img
                    src={signer.signatureDataUrl}
                    alt={t('vertraege.esignatur.signatureThumbnailAlt')}
                    className="h-9 w-20 rounded border border-border object-contain bg-white mt-1"
                  />
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

// ─── AI Deadline Check Panel (Phase 10) ─────────────────────────

type DeadlineSeverity = 'warning' | 'info' | 'error'

interface DeadlineFinding {
  key: string
  severity: DeadlineSeverity
}

function computeDeadlineFindings(contract: Contract): DeadlineFinding[] {
  const findings: DeadlineFinding[] = []
  const today = new Date()
  today.setHours(0, 0, 0, 0)

  // 1. Notice window approaching: endDate - noticePeriodDays <= 30 days from now
  if (contract.endDate && contract.noticePeriodDays > 0) {
    const end = new Date(contract.endDate + 'T00:00:00')
    const noticeDeadline = new Date(end)
    noticeDeadline.setDate(noticeDeadline.getDate() - contract.noticePeriodDays)
    const daysToNotice = Math.ceil((noticeDeadline.getTime() - today.getTime()) / (1000 * 60 * 60 * 24))
    if (daysToNotice > 0 && daysToNotice <= 30) {
      findings.push({ key: 'noticeWindowSoon', severity: 'warning' })
    }
  }

  // 2. No reminders set and contract has an endDate
  if (contract.endDate && (!contract.reminderDays || contract.reminderDays.length === 0)) {
    findings.push({ key: 'noReminders', severity: 'info' })
  }

  // 3. Contract expired but not terminated / renewed
  if (contract.endDate && (contract.status === 'active' || contract.status === 'expiring')) {
    const end = new Date(contract.endDate + 'T00:00:00')
    if (end < today) {
      findings.push({ key: 'expiredActive', severity: 'error' })
    }
  }

  // 4. Active contract has no documents and no signers
  if ((contract.status === 'active' || contract.status === 'expiring') &&
    (!contract.documents || contract.documents.length === 0) &&
    (!contract.signers || contract.signers.length === 0)) {
    findings.push({ key: 'noDocumentsOrSignature', severity: 'info' })
  }

  return findings
}

const severityStyles: Record<DeadlineSeverity, { bg: string; border: string; dot: string; text: string }> = {
  error:   { bg: 'bg-error-light',   border: 'border-error/20',   dot: 'bg-error',   text: 'text-error' },
  warning: { bg: 'bg-warning-light', border: 'border-warning/20', dot: 'bg-warning', text: 'text-warning' },
  info:    { bg: 'bg-info-light',    border: 'border-info/20',    dot: 'bg-info',    text: 'text-info' },
}

function DeadlineCheckPanel({ contract }: { contract: Contract }) {
  const { t } = useTranslation()
  const [ran, setRan] = useState(false)
  const [findings, setFindings] = useState<DeadlineFinding[]>([])

  const handleRun = () => {
    setFindings(computeDeadlineFindings(contract))
    setRan(true)
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider flex-1">
          {t('vertraege.deadlineCheck.title')}
        </h4>
        <span className="inline-flex items-center gap-1 rounded-full border border-primary/30 bg-primary-light px-2 py-0.5 text-[10px] font-medium text-primary">
          <Sparkles className="h-2.5 w-2.5" />
          {t('vertraege.deadlineCheck.badge')}
        </span>
      </div>

      {!ran ? (
        <div className="rounded-lg border border-dashed border-border bg-secondary/20 p-3">
          <p className="text-xs text-muted-foreground mb-2.5">{t('vertraege.deadlineCheck.description')}</p>
          <button
            onClick={handleRun}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
            aria-label={t('vertraege.deadlineCheck.runButton')}
          >
            <Sparkles className="h-3.5 w-3.5" />
            {t('vertraege.deadlineCheck.runButton')}
          </button>
        </div>
      ) : findings.length === 0 ? (
        <div className="rounded-lg border border-success/20 bg-success-light p-3">
          <p className="text-xs text-success font-medium">{t('vertraege.deadlineCheck.allClear')}</p>
          <p className="text-xs text-muted-foreground mt-0.5">{t('vertraege.deadlineCheck.disclaimer')}</p>
        </div>
      ) : (
        <div className="space-y-2">
          {findings.map((f) => {
            const styles = severityStyles[f.severity]
            return (
              <div key={f.key} className={`rounded-lg border ${styles.bg} ${styles.border} p-3 flex items-start gap-2`}>
                <span className={`mt-1 h-1.5 w-1.5 rounded-full shrink-0 ${styles.dot}`} />
                <p className={`text-xs ${styles.text}`}>{t(`vertraege.deadlineCheck.findings.${f.key}`)}</p>
              </div>
            )
          })}
          <p className="text-[10px] text-muted-foreground">{t('vertraege.deadlineCheck.disclaimer')}</p>
        </div>
      )}
    </div>
  )
}

interface DetailPanelContentProps {
  contract: Contract
  onPreviewDoc: (file: DocumentFile) => void
  onVersionsDoc: (fileId: string, name: string) => void
  onNavigate: (path: string) => void
}

function DetailPanelContent({ contract, onPreviewDoc, onVersionsDoc, onNavigate }: DetailPanelContentProps) {
  const { t } = useTranslation()
  const typeConf = contractTypeConfig[contract.type]
  const TypeIcon = typeConf.icon
  const remaining = contract.endDate ? daysUntil(contract.endDate) : Infinity
  const { pct } = contract.endDate
    ? daysFromStart(contract.startDate, contract.endDate)
    : { pct: 0 }

  // Calculate notice period deadline
  const noticeDateStr = contract.endDate
    ? (() => {
        const end = new Date(contract.endDate + 'T00:00:00')
        end.setDate(end.getDate() - contract.noticePeriodDays)
        return end.toISOString().split('T')[0]
      })()
    : null

  const daysUntilNoticeDeadline = noticeDateStr ? daysUntil(noticeDateStr) : Infinity

  const currency = contract.currency ?? 'EUR'

  return (
    <div className="space-y-5">
      {/* Type badge + Partner */}
      <div className="space-y-3">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vertraege.detail.vertragsdetails')}</h4>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <p className="text-xs text-muted-foreground">{t('vertraege.detail.fieldPartner')}</p>
            <p className="text-sm text-foreground font-medium">{contract.partner}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">{t('vertraege.detail.fieldContractNumber')}</p>
            <p className="text-sm text-foreground font-mono">{contract.contractNumber}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">{t('vertraege.detail.fieldTyp')}</p>
            <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${typeConf.colorClass}`}>
              <TypeIcon className="h-3 w-3" />
              {t(typeConf.labelKey)}
            </span>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">{t('vertraege.detail.fieldStatus')}</p>
            <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusConfig[contract.status].colorClass}`}>
              {t(statusConfig[contract.status].labelKey)}
            </span>
          </div>
        </div>
      </div>

      {/* Duration visualization */}
      <div className="space-y-2">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vertraege.detail.laufzeit')}</h4>
        <div className="rounded-lg border border-border bg-secondary/30 p-3">
          {contract.endDate ? (
            <>
              <div className="flex items-baseline justify-between mb-2">
                <span className="text-sm font-medium text-foreground">
                  {formatDate(contract.startDate)} &mdash; {formatDate(contract.endDate)}
                </span>
              </div>
              {/* Bar with notice period marker */}
              <div className="relative h-2.5 w-full rounded-full bg-secondary overflow-hidden mb-1">
                <div
                  className={`absolute inset-y-0 left-0 rounded-full transition-all ${
                    remaining <= 0
                      ? 'bg-error'
                      : remaining <= 90
                        ? 'bg-warning'
                        : 'bg-primary'
                  }`}
                  style={{ width: `${pct}%` }}
                />
                {/* Notice period marker */}
                {noticeDateStr && (
                  <div
                    className="absolute inset-y-0 w-0.5 bg-foreground/40"
                    style={{ left: `${Math.max(0, 100 - (contract.noticePeriodDays / Math.max(1, (new Date(contract.endDate + 'T00:00:00').getTime() - new Date(contract.startDate + 'T00:00:00').getTime()) / (1000 * 60 * 60 * 24))) * 100)}%` }}
                    title={t('vertraege.detail.noticePeriodFrom', { date: formatDate(noticeDateStr) })}
                  />
                )}
              </div>
              <div className="flex justify-between text-[10px] text-muted-foreground">
                <span>{t('vertraege.durationBar.start')}</span>
                <span>{t('vertraege.durationBar.today', { pct: Math.round(pct) })}</span>
                <span>{t('vertraege.durationBar.end')}</span>
              </div>
              {remaining > 0 && (
                <p className="text-xs text-muted-foreground mt-2">
                  {t('vertraege.detail.remainingDays', { days: remaining })}
                </p>
              )}
              {remaining <= 0 && (
                <p className="text-xs text-error mt-2 font-medium">
                  {t('vertraege.detail.expiredDays', { days: Math.abs(remaining) })}
                </p>
              )}
            </>
          ) : (
            <p className="text-sm text-foreground">
              {t('vertraege.detail.unbefristetSince', { date: formatDate(contract.startDate) })}
            </p>
          )}
        </div>
      </div>

      {/* Financial info (10.10 — currency-aware) */}
      <div className="space-y-2">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vertraege.detail.wert')}</h4>
        <div className="grid grid-cols-2 gap-3">
          <div className="rounded-lg border border-border p-3">
            <p className="text-xs text-muted-foreground">{t('vertraege.detail.monthly')}</p>
            <p className="text-lg font-semibold text-foreground tabular-nums">{formatCurrency(contract.monthlyCost, currency)}</p>
          </div>
          <div className="rounded-lg border border-border p-3">
            <p className="text-xs text-muted-foreground">{t('vertraege.detail.totalValue')}</p>
            <p className="text-lg font-semibold text-foreground tabular-nums">
              {contract.totalValue > 0 ? formatCurrency(contract.totalValue, currency) : '\u2014'}
            </p>
          </div>
        </div>
      </div>

      {/* Contract terms */}
      <div className="space-y-2">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vertraege.detail.konditionen')}</h4>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <p className="text-xs text-muted-foreground">{t('vertraege.detail.fieldRenewal')}</p>
            <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
              contract.renewal === 'auto'
                ? 'bg-success-light text-success'
                : 'bg-secondary text-muted-foreground'
            }`}>
              {contract.renewal === 'auto' ? <RefreshCw className="h-3 w-3" /> : <CalendarClock className="h-3 w-3" />}
              {t(renewalLabelKeys[contract.renewal])}
            </span>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">{t('vertraege.detail.fieldNoticePeriod')}</p>
            <p className="text-sm text-foreground font-medium">{t('vertraege.detail.noticePeriodDays', { days: contract.noticePeriodDays })}</p>
          </div>
        </div>
      </div>

      {/* Reminder schedule (phase 4) */}
      <ReminderSchedule contract={contract} />

      {/* Next action hint */}
      {contract.endDate && (contract.status === 'active' || contract.status === 'expiring') && (
        <div className="space-y-2">
          <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vertraege.detail.nextAction')}</h4>
          <div className={`rounded-lg border p-3 ${
            daysUntilNoticeDeadline <= 30
              ? 'border-warning/30 bg-warning-light/30'
              : 'border-border bg-secondary/30'
          }`}>
            {daysUntilNoticeDeadline > 0 ? (
              <div className="flex items-start gap-2">
                <Clock className={`h-4 w-4 mt-0.5 shrink-0 ${daysUntilNoticeDeadline <= 30 ? 'text-warning' : 'text-muted-foreground'}`} />
                <div>
                  <p className="text-sm text-foreground">
                    {t('vertraege.detail.terminationUntil', { date: formatDate(noticeDateStr!) })}
                  </p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {t('vertraege.detail.terminationDaysLeft', { days: daysUntilNoticeDeadline })}
                  </p>
                </div>
              </div>
            ) : contract.renewal === 'auto' ? (
              <div className="flex items-start gap-2">
                <RefreshCw className="h-4 w-4 text-success mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm text-foreground">{t('vertraege.detail.autoRenewal')}</p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {t('vertraege.detail.autoRenewalDesc')}
                  </p>
                </div>
              </div>
            ) : (
              <div className="flex items-start gap-2">
                <AlertTriangle className="h-4 w-4 text-warning mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm text-foreground">{t('vertraege.detail.manualRenewal')}</p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {t('vertraege.detail.manualRenewalDesc')}
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Notes */}
      {contract.notes && (
        <div className="space-y-2">
          <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('vertraege.detail.notizen')}</h4>
          <p className="text-sm text-foreground leading-relaxed">{contract.notes}</p>
        </div>
      )}

      {/* Phase 7: Dokumente-Sektion */}
      <div className="space-y-2">
        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          {t('vertraege.documents.sectionTitle')}
          {(contract.documents?.length ?? 0) > 0 && (
            <span className="ml-1.5 text-muted-foreground font-normal normal-case">
              ({contract.documents!.length})
            </span>
          )}
        </h4>
        {(contract.documents?.length ?? 0) === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-5 text-center">
            <FileText className="h-6 w-6 text-muted-foreground/40 mb-2" />
            <p className="text-xs text-muted-foreground">{t('vertraege.documents.emptyState')}</p>
          </div>
        ) : (
          <div className="space-y-1.5">
            {contract.documents!.map((doc) => {
              const DocIcon = getMimeIcon(doc.mimeType ?? '')
              return (
                <div
                  key={doc.fileId}
                  className="flex items-center gap-2 rounded-lg border border-border bg-secondary/20 px-3 py-2 group"
                >
                  <DocIcon className="h-4 w-4 text-primary shrink-0" />
                  <button
                    onClick={() => onPreviewDoc(contractDocToDocumentFile(doc))}
                    aria-label={t('vertraege.documents.preview', { name: doc.name })}
                    className="flex-1 text-left text-sm text-foreground hover:text-primary transition-colors truncate"
                  >
                    {doc.name}
                  </button>
                  {doc.size != null && (
                    <span className="text-[10px] text-muted-foreground shrink-0">
                      {formatBytes(doc.size)}
                    </span>
                  )}
                  <TooltipProvider delayDuration={200}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <button
                          onClick={() => onVersionsDoc(doc.fileId, doc.name)}
                          aria-label={t('vertraege.documents.versions', { name: doc.name })}
                          className="text-muted-foreground hover:text-foreground transition-colors shrink-0 opacity-0 group-hover:opacity-100"
                        >
                          <VersionIcon className="h-3.5 w-3.5" />
                        </button>
                      </TooltipTrigger>
                      <TooltipContent>
                        <p className="text-xs">{t('vertraege.documents.versionsTooltip')}</p>
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Signer section (phase 8) */}
      <SignerSection contract={contract} />

      {/* Phase 10: CRM/Finance Verknüpfungen */}
      {(contract.contactId || contract.dealId || (contract.invoiceIds && contract.invoiceIds.length > 0)) && (
        <div className="space-y-2">
          <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
            <Link2 className="h-3.5 w-3.5" />
            {t('vertraege.links.sectionTitle')}
          </h4>
          <div className="flex flex-wrap gap-2">
            {contract.contactId && (
              <button
                onClick={() => onNavigate(`/kontakte?contact=${contract.contactId}`)}
                aria-label={t('vertraege.links.goToContact', { name: contract.contactName })}
                className="inline-flex items-center gap-1.5 rounded-full border border-info/30 bg-info-light px-2.5 py-1 text-xs font-medium text-info hover:bg-info hover:text-white transition-colors"
              >
                <User className="h-3 w-3" />
                {contract.contactName}
              </button>
            )}
            {contract.dealId && (
              <button
                onClick={() => onNavigate(`/kontakte/pipeline/${contract.dealId}`)}
                aria-label={t('vertraege.links.goToDeal', { name: contract.dealTitle })}
                className="inline-flex items-center gap-1.5 rounded-full border border-warning/30 bg-warning-light px-2.5 py-1 text-xs font-medium text-warning hover:bg-warning hover:text-white transition-colors"
              >
                <Briefcase className="h-3 w-3" />
                {contract.dealTitle}
              </button>
            )}
            {(contract.invoiceIds ?? []).map((invId, idx) => (
              <button
                key={invId}
                onClick={() => onNavigate(`/finanzen?invoice=${invId}`)}
                aria-label={t('vertraege.links.goToInvoice', { name: contract.invoiceNames?.[idx] ?? invId })}
                className="inline-flex items-center gap-1.5 rounded-full border border-success/30 bg-success-light px-2.5 py-1 text-xs font-medium text-success hover:bg-success hover:text-white transition-colors"
              >
                <Receipt className="h-3 w-3" />
                {contract.invoiceNames?.[idx] ?? invId}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Phase 10: AI Fristencheck (Heuristik-Panel) */}
      <DeadlineCheckPanel contract={contract} />

      {/* Audit log feed (phase 4) */}
      <AuditLogFeed history={contract.history} />
    </div>
  )
}
