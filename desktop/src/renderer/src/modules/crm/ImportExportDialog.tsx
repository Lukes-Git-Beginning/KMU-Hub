/**
 * ImportExportDialog — CSV/vCard import & export for CRM module.
 *
 * Multi-step flow: select entity type → upload file → field mapping → preview → import.
 * Export tab: entity picker, format (CSV/vCard/Excel), field selection.
 * Mock data for design — backend swap: replace with API hooks.
 */
import { useState, useCallback, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Upload,
  Download,
  FileText,
  FileSpreadsheet,
  Users,
  Building2,
  Briefcase,
  AlertCircle,
  CheckCircle2,
  X,
  ArrowRight,
  ArrowLeft,
  RefreshCw,
  AlertTriangle,
  Check,
  ChevronDown,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type EntityType = 'contacts' | 'companies' | 'deals'
type ExportFormat = 'csv' | 'vcard' | 'xlsx'
type ImportStep = 'upload' | 'mapping' | 'preview'

interface FieldMapping {
  sourceColumn: string
  targetField: string
  preview: string
}

interface ImportPreviewRow {
  fields: Record<string, string>
  hasDuplicate: boolean
  duplicateReason?: string
}

// ---------------------------------------------------------------------------
// Entity config
// ---------------------------------------------------------------------------

const entityConfig: Record<EntityType, {
  labelKey: string
  labelPluralKey: string
  icon: typeof Users
  fields: { key: string; labelKey: string }[]
  exportFormats: ExportFormat[]
}> = {
  contacts: {
    labelKey: 'crm.importExport.entity.contact',
    labelPluralKey: 'crm.importExport.entity.contacts',
    icon: Users,
    fields: [
      { key: 'salutation', labelKey: 'crm.importExport.field.salutation' },
      { key: 'title', labelKey: 'crm.importExport.field.title' },
      { key: 'firstName', labelKey: 'crm.field.firstName' },
      { key: 'lastName', labelKey: 'crm.field.lastName' },
      { key: 'email', labelKey: 'crm.field.email' },
      { key: 'phone', labelKey: 'crm.field.phone' },
      { key: 'mobile', labelKey: 'crm.field.mobile' },
      { key: 'company', labelKey: 'crm.field.company' },
      { key: 'jobTitle', labelKey: 'crm.field.jobTitle' },
      { key: 'department', labelKey: 'crm.field.department' },
      { key: 'street', labelKey: 'crm.importExport.field.street' },
      { key: 'zip', labelKey: 'crm.importExport.field.zip' },
      { key: 'city', labelKey: 'crm.importExport.field.city' },
      { key: 'country', labelKey: 'crm.importExport.field.country' },
      { key: 'website', labelKey: 'crm.field.website' },
      { key: 'notes', labelKey: 'crm.field.notes' },
      { key: 'tags', labelKey: 'crm.field.tags' },
    ],
    exportFormats: ['csv', 'vcard', 'xlsx'],
  },
  companies: {
    labelKey: 'crm.importExport.entity.company',
    labelPluralKey: 'crm.importExport.entity.companies',
    icon: Building2,
    fields: [
      { key: 'name', labelKey: 'crm.companies.companyName' },
      { key: 'industry', labelKey: 'crm.field.industry' },
      { key: 'size', labelKey: 'crm.field.size' },
      { key: 'phone', labelKey: 'crm.field.phone' },
      { key: 'email', labelKey: 'crm.field.email' },
      { key: 'website', labelKey: 'crm.field.website' },
      { key: 'street', labelKey: 'crm.importExport.field.street' },
      { key: 'zip', labelKey: 'crm.importExport.field.zip' },
      { key: 'city', labelKey: 'crm.importExport.field.city' },
      { key: 'country', labelKey: 'crm.importExport.field.country' },
      { key: 'notes', labelKey: 'crm.field.notes' },
    ],
    exportFormats: ['csv', 'xlsx'],
  },
  deals: {
    labelKey: 'crm.importExport.entity.deal',
    labelPluralKey: 'crm.importExport.entity.deals',
    icon: Briefcase,
    fields: [
      { key: 'name', labelKey: 'crm.deals.dealName' },
      { key: 'value', labelKey: 'crm.deals.value' },
      { key: 'currency', labelKey: 'crm.deals.currency' },
      { key: 'stage', labelKey: 'crm.deals.stage' },
      { key: 'probability', labelKey: 'crm.deals.probability' },
      { key: 'contact', labelKey: 'crm.field.contact' },
      { key: 'company', labelKey: 'crm.field.company' },
      { key: 'expectedClose', labelKey: 'crm.deals.expectedClose' },
      { key: 'notes', labelKey: 'crm.field.notes' },
      { key: 'tags', labelKey: 'crm.field.tags' },
    ],
    exportFormats: ['csv', 'xlsx'],
  },
}

const formatLabels: Record<ExportFormat, string> = {
  csv: 'CSV',
  vcard: 'vCard',
  xlsx: 'Excel (XLSX)',
}

const formatIcons: Record<ExportFormat, typeof FileText> = {
  csv: FileText,
  vcard: FileText,
  xlsx: FileSpreadsheet,
}

// ---------------------------------------------------------------------------
// Mock preview data
// ---------------------------------------------------------------------------

function generateMockPreview(entity: EntityType, mappings: FieldMapping[]): ImportPreviewRow[] {
  const rows: ImportPreviewRow[] = []
  const sampleData: Record<string, string[]> = {
    firstName: ['Anna', 'Max', 'Sara', 'Thomas', 'Laura'],
    lastName: ['Müller', 'Schmidt', 'Weber', 'Fischer', 'Wagner'],
    email: ['anna@example.de', 'max@test.de', 'sara@firma.de', 'thomas@corp.de', 'laura@web.de'],
    phone: ['+49 30 123 45 67', '+49 89 987 65 43', '+49 40 555 44 33', '+49 30 222 33 44', '+49 69 111 22 33'],
    company: ['Muster AG', 'Test GmbH', 'Beispiel AG', 'Demo GmbH', 'Berlin Corp'],
    name: ['Muster AG', 'Test GmbH', 'Beispiel AG', 'Demo GmbH', 'Berlin Corp'],
    value: ['25000', '48000', '12500', '95000', '33000'],
    stage: ['Qualifiziert', 'Angebot', 'Verhandlung', 'Lead', 'Gewonnen'],
  }

  for (let i = 0; i < 5; i++) {
    const fields: Record<string, string> = {}
    for (const m of mappings) {
      if (m.targetField !== '—') {
        fields[m.targetField] = sampleData[m.targetField]?.[i] ?? m.preview
      }
    }
    const isDuplicate = i === 1 || i === 3
    rows.push({
      fields,
      hasDuplicate: isDuplicate,
      duplicateReason: isDuplicate
        ? i === 1
          ? 'Gleiche E-Mail-Adresse'
          : 'Ähnlicher Name'
        : undefined,
    })
  }
  return rows
}

// ---------------------------------------------------------------------------
// Mock export counts
// ---------------------------------------------------------------------------

const mockExportCounts: Record<EntityType, number> = {
  contacts: 847,
  companies: 156,
  deals: 89,
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ImportExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  defaultTab?: 'import' | 'export'
  defaultEntity?: EntityType
}

export function ImportExportDialog({
  open,
  onOpenChange,
  defaultTab = 'import',
  defaultEntity = 'contacts',
}: ImportExportDialogProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<'import' | 'export'>(defaultTab)
  const [entity, setEntity] = useState<EntityType>(defaultEntity)
  const [importStep, setImportStep] = useState<ImportStep>('upload')
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState('')
  const [importing, setImporting] = useState(false)
  const [imported, setImported] = useState(false)
  const [importCount, setImportCount] = useState(0)
  const [duplicateAction, setDuplicateAction] = useState<'skip' | 'update' | 'create'>('skip')

  // Export state
  const [exportFormat, setExportFormat] = useState<ExportFormat>('csv')
  const [exportFields, setExportFields] = useState<Set<string>>(new Set())
  const [exporting, setExporting] = useState(false)

  // Mock mappings
  const [mappings, setMappings] = useState<FieldMapping[]>([])

  const config = entityConfig[entity]

  // Init export fields when entity changes
   
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- reset export fields on entity change
    setExportFields(new Set(config.fields.map((f) => f.key)))
  }, [entity, config])

  const reset = () => {
    setImportStep('upload')
    setFile(null)
    setError('')
    setImporting(false)
    setImported(false)
    setImportCount(0)
    setMappings([])
  }

  const handleClose = () => {
    onOpenChange(false)
    setTimeout(reset, 200)
  }

  // --- Import ---

  const handleFile = useCallback(
    (f: File) => {
      setFile(f)
      setError('')
      setImported(false)

      const isCSV = f.name.endsWith('.csv')
      const isVCard = f.name.endsWith('.vcf') || f.name.endsWith('.vcard')

      if (!isCSV && !isVCard) {
        setError(t('crm.importExport.errorFileType'))
        return
      }

      if (isVCard && entity !== 'contacts') {
        setError(t('crm.importExport.errorVcardContacts'))
        return
      }

      // Mock column detection
      const mockColumns =
        entity === 'contacts'
          ? ['Vorname', 'Nachname', 'E-Mail', 'Telefon', 'Firma', 'Position', 'Ort']
          : entity === 'companies'
            ? ['Firmenname', 'Branche', 'Telefon', 'E-Mail', 'Website', 'Ort']
            : ['Deal-Name', 'Wert', 'Phase', 'Kontakt', 'Firma', 'Abschlussdatum']

      const autoMap: Record<string, string> = {
        Vorname: 'firstName',
        Nachname: 'lastName',
        'E-Mail': 'email',
        Telefon: 'phone',
        Firma: 'company',
        Firmenname: 'name',
        Position: 'jobTitle',
        Branche: 'industry',
        Website: 'website',
        Ort: 'city',
        'Deal-Name': 'name',
        Wert: 'value',
        Phase: 'stage',
        Kontakt: 'contact',
        Abschlussdatum: 'expectedClose',
      }

      const sampleValues: Record<string, string> = {
        Vorname: 'Anna',
        Nachname: 'Müller',
        'E-Mail': 'anna@example.de',
        Telefon: '+49 30 123 45 67',
        Firma: 'Muster AG',
        Firmenname: 'Muster AG',
        Position: 'Geschaeftsfuehrerin',
        Branche: 'Technologie',
        Website: 'www.muster.de',
        Ort: 'Berlin',
        'Deal-Name': 'Enterprise-Lizenz',
        Wert: '25000',
        Phase: 'Qualifiziert',
        Kontakt: 'Anna Müller',
        Abschlussdatum: '2026-03-15',
      }

      setMappings(
        mockColumns.map((col) => ({
          sourceColumn: col,
          targetField: autoMap[col] ?? '—',
          preview: sampleValues[col] ?? '',
        })),
      )

      setImportStep('mapping')
    },
    [entity],
  )

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      const f = e.dataTransfer.files[0]
      if (f) handleFile(f)
    },
    [handleFile],
  )

  const handleImport = () => {
    setImporting(true)
    setTimeout(() => {
      setImporting(false)
      setImported(true)
      const count = Math.floor(Math.random() * 30) + 40
      setImportCount(count)
      toast.success(t('crm.importExport.importSuccess', { count, entity: t(config.labelPluralKey) }))
    }, 2000)
  }

  // --- Export ---

  const handleExport = () => {
    setExporting(true)
    setTimeout(() => {
      setExporting(false)
      toast.success(
        t('crm.importExport.exportSuccess', { count: mockExportCounts[entity], entity: t(config.labelPluralKey), format: formatLabels[exportFormat] }),
      )
    }, 1500)
  }

  const toggleExportField = (key: string) => {
    setExportFields((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  const previewRows = useMemo(
    () => (importStep === 'preview' ? generateMockPreview(entity, mappings) : []),
    [importStep, entity, mappings],
  )
  const duplicateCount = previewRows.filter((r) => r.hasDuplicate).length

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {activeTab === 'import' ? (
              <>
                <Upload className="h-5 w-5 text-primary" />
                CRM Import / Export
              </>
            ) : (
              <>
                <Download className="h-5 w-5 text-primary" />
                CRM Import / Export
              </>
            )}
          </DialogTitle>
        </DialogHeader>

        {/* Tab switcher */}
        <div className="flex items-center gap-1 border-b border-border">
          {([['import', t('crm.importExport.import')], ['export', t('crm.importExport.export')]] as const).map(([key, label]) => (
            <button
              key={key}
              onClick={() => {
                setActiveTab(key)
                reset()
              }}
              className={`border-b-2 px-4 pb-2 text-sm transition-colors ${
                activeTab === key
                  ? 'border-primary text-primary font-medium'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              {label}
            </button>
          ))}
        </div>

        {/* Entity selector */}
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">{t('crm.importExport.entityType')}:</span>
          {(Object.entries(entityConfig) as [EntityType, typeof entityConfig.contacts][]).map(
            ([key, cfg]) => {
              const Icon = cfg.icon
              return (
                <button
                  key={key}
                  onClick={() => {
                    setEntity(key)
                    reset()
                  }}
                  className={`flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs transition-colors ${
                    entity === key
                      ? 'border-primary bg-primary/5 text-primary font-medium'
                      : 'border-border text-muted-foreground hover:bg-accent'
                  }`}
                >
                  <Icon className="h-3.5 w-3.5" />
                  {t(cfg.labelPluralKey)}
                </button>
              )
            },
          )}
        </div>

        {/* ---- IMPORT TAB ---- */}
        {activeTab === 'import' && (
          <div className="space-y-4">
            {imported ? (
              <div className="py-8 text-center">
                <CheckCircle2 className="mx-auto h-12 w-12 text-success mb-3" />
                <p className="text-base font-medium text-foreground">
                  {t('crm.importExport.importedCount', { count: importCount, entity: t(config.labelPluralKey) })}
                </p>
                <p className="text-sm text-muted-foreground mt-1">
                  {t('crm.importExport.importedSuccess')}
                </p>
                <button
                  onClick={handleClose}
                  className="mt-4 h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
                >
                  {t('common.close')}
                </button>
              </div>
            ) : (
              <>
                {/* Step indicator */}
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  {(['upload', 'mapping', 'preview'] as const).map((step, i) => (
                    <div key={step} className="flex items-center gap-2">
                      {i > 0 && <ArrowRight className="h-3 w-3" />}
                      <span
                        className={`flex items-center gap-1 ${
                          importStep === step ? 'text-primary font-medium' : ''
                        }`}
                      >
                        <span
                          className={`flex h-5 w-5 items-center justify-center rounded-full text-[10px] ${
                            importStep === step
                              ? 'bg-primary text-primary-foreground'
                              : 'bg-secondary text-muted-foreground'
                          }`}
                        >
                          {i + 1}
                        </span>
                        {step === 'upload'
                          ? t('crm.importExport.stepFile')
                          : step === 'mapping'
                            ? t('crm.importExport.stepMapping')
                            : t('crm.importExport.stepPreview')}
                      </span>
                    </div>
                  ))}
                </div>

                {/* Step: Upload */}
                {importStep === 'upload' && (
                  <div className="space-y-3">
                    <div
                      onDragOver={(e) => e.preventDefault()}
                      onDrop={handleDrop}
                      className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-border p-8 text-center hover:border-primary/50 hover:bg-accent/30 transition-colors cursor-pointer"
                      onClick={() => document.getElementById('crm-import-upload')?.click()}
                    >
                      <Upload className="h-8 w-8 text-muted-foreground mb-2" />
                      <p className="text-sm font-medium text-foreground">
                        {t('crm.importExport.dropFile')}
                      </p>
                      <p className="text-xs text-muted-foreground mt-1">
                        {entity === 'contacts' ? t('crm.importExport.dropHintContacts') : t('crm.importExport.dropHintOther')}
                      </p>
                      <input
                        id="crm-import-upload"
                        type="file"
                        accept={entity === 'contacts' ? '.csv,.vcf,.vcard' : '.csv'}
                        className="hidden"
                        onChange={(e) => {
                          const f = e.target.files?.[0]
                          if (f) handleFile(f)
                        }}
                      />
                    </div>

                    {error && (
                      <div className="flex items-start gap-2 rounded-md border border-error/30 bg-error-light/20 px-3 py-2">
                        <AlertCircle className="h-4 w-4 text-error shrink-0 mt-0.5" />
                        <p className="text-sm text-error">{error}</p>
                      </div>
                    )}

                    <div className="rounded-md bg-secondary/50 px-3 py-2">
                      <p className="text-xs text-muted-foreground">
                        <strong>{t('crm.importExport.supportedFields', { entity: t(config.labelPluralKey) })}:</strong>{' '}
                        {config.fields.map((f) => t(f.labelKey)).join(', ')}
                      </p>
                    </div>
                  </div>
                )}

                {/* Step: Mapping */}
                {importStep === 'mapping' && (
                  <div className="space-y-3">
                    {/* File info */}
                    {file && (
                      <div className="flex items-center gap-2 rounded-md border border-border px-3 py-2">
                        <FileText className="h-4 w-4 text-primary shrink-0" />
                        <span className="flex-1 text-sm text-foreground truncate">{file.name}</span>
                        <button
                          onClick={() => {
                            reset()
                          }}
                          className="text-muted-foreground hover:text-foreground"
                        >
                          <X className="h-4 w-4" />
                        </button>
                      </div>
                    )}

                    <p className="text-xs text-muted-foreground">
                      {t('crm.importExport.mappingHint', { entity: t(config.labelKey) })}
                    </p>

                    {/* Mapping table */}
                    <div className="rounded-md border border-border overflow-hidden">
                      <div className="grid grid-cols-[1fr_24px_1fr_120px] gap-2 items-center bg-secondary/50 px-3 py-2 text-[11px] font-medium text-muted-foreground">
                        <span>{t('crm.importExport.sourceColumn')}</span>
                        <span />
                        <span>{t('crm.importExport.targetField')}</span>
                        <span>{t('crm.importExport.preview')}</span>
                      </div>
                      {mappings.map((m, i) => (
                        <div
                          key={m.sourceColumn}
                          className="grid grid-cols-[1fr_24px_1fr_120px] gap-2 items-center border-t border-border px-3 py-2"
                        >
                          <span className="text-sm text-foreground">{m.sourceColumn}</span>
                          <ArrowRight className="h-3 w-3 text-muted-foreground" />
                          <div className="relative">
                            <select
                              value={m.targetField}
                              onChange={(e) => {
                                setMappings((prev) =>
                                  prev.map((mp, j) =>
                                    j === i ? { ...mp, targetField: e.target.value } : mp,
                                  ),
                                )
                              }}
                              className="h-8 w-full rounded-md border border-border bg-transparent px-2 pr-7 text-xs outline-none focus:border-primary appearance-none"
                            >
                              <option value="—">— {t('crm.importExport.ignore')} —</option>
                              {config.fields.map((f) => (
                                <option key={f.key} value={f.key}>
                                  {t(f.labelKey)}
                                </option>
                              ))}
                            </select>
                            <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground pointer-events-none" />
                          </div>
                          <span className="text-xs text-muted-foreground truncate">{m.preview}</span>
                        </div>
                      ))}
                    </div>

                    {/* Duplicate handling */}
                    <div className="rounded-md border border-border p-3 space-y-2">
                      <p className="text-xs font-medium text-foreground">{t('crm.importExport.duplicateHandling')}</p>
                      <div className="flex items-center gap-2">
                        {([
                          ['skip', t('crm.importExport.duplicateSkip')],
                          ['update', t('crm.importExport.duplicateUpdate')],
                          ['create', t('crm.importExport.duplicateCreate')],
                        ] as const).map(([key, label]) => (
                          <button
                            key={key}
                            onClick={() => setDuplicateAction(key)}
                            className={`rounded-md border px-3 py-1.5 text-xs transition-colors ${
                              duplicateAction === key
                                ? 'border-primary bg-primary/5 text-primary font-medium'
                                : 'border-border text-muted-foreground hover:bg-accent'
                            }`}
                          >
                            {label}
                          </button>
                        ))}
                      </div>
                      <p className="text-[10px] text-muted-foreground">
                        {duplicateAction === 'skip'
                          ? t('crm.importExport.duplicateSkipDesc')
                          : duplicateAction === 'update'
                            ? t('crm.importExport.duplicateUpdateDesc')
                            : t('crm.importExport.duplicateCreateDesc')}
                      </p>
                    </div>

                    {/* Actions */}
                    <div className="flex justify-between pt-1">
                      <button
                        onClick={reset}
                        className="flex items-center gap-1.5 h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
                      >
                        <ArrowLeft className="h-3.5 w-3.5" />
                        {t('common.back')}
                      </button>
                      <button
                        onClick={() => setImportStep('preview')}
                        className="flex items-center gap-1.5 h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
                      >
                        {t('crm.importExport.stepPreview')}
                        <ArrowRight className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>
                )}

                {/* Step: Preview */}
                {importStep === 'preview' && (
                  <div className="space-y-3">
                    {/* Summary */}
                    <div className="flex items-center gap-4">
                      <div className="flex items-center gap-1.5 rounded-md bg-secondary px-3 py-1.5 text-xs">
                        <Check className="h-3.5 w-3.5 text-success" />
                        <span className="text-foreground font-medium">{previewRows.length}</span>
                        <span className="text-muted-foreground">{t(config.labelPluralKey)}</span>
                      </div>
                      {duplicateCount > 0 && (
                        <div className="flex items-center gap-1.5 rounded-md bg-warning-light/30 px-3 py-1.5 text-xs">
                          <AlertTriangle className="h-3.5 w-3.5 text-warning" />
                          <span className="text-warning font-medium">{duplicateCount}</span>
                          <span className="text-muted-foreground">
                            {t('crm.importExport.possibleDuplicates')} ({duplicateAction === 'skip' ? t('crm.importExport.willSkip') : duplicateAction === 'update' ? t('crm.importExport.willUpdate') : t('crm.importExport.willCreate')})
                          </span>
                        </div>
                      )}
                    </div>

                    {/* Preview table */}
                    <div className="max-h-56 overflow-auto rounded-md border border-border">
                      <table className="w-full text-xs">
                        <thead className="bg-secondary/50 sticky top-0">
                          <tr>
                            <th className="text-left px-2 py-1.5 font-medium text-muted-foreground w-6" />
                            {mappings
                              .filter((m) => m.targetField !== '—')
                              .slice(0, 5)
                              .map((m) => (
                                <th
                                  key={m.sourceColumn}
                                  className="text-left px-2 py-1.5 font-medium text-muted-foreground"
                                >
                                  {config.fields.find((f) => f.key === m.targetField)
                                    ? t(config.fields.find((f) => f.key === m.targetField)!.labelKey)
                                    : m.targetField}
                                </th>
                              ))}
                          </tr>
                        </thead>
                        <tbody>
                          {previewRows.map((row, i) => (
                            <tr
                              key={i}
                              className={`border-t border-border ${
                                row.hasDuplicate ? 'bg-warning-light/10' : ''
                              }`}
                            >
                              <td className="px-2 py-1.5">
                                {row.hasDuplicate && (
                                  <AlertTriangle className="h-3 w-3 text-warning" />
                                )}
                              </td>
                              {mappings
                                .filter((m) => m.targetField !== '—')
                                .slice(0, 5)
                                .map((m) => (
                                  <td
                                    key={m.sourceColumn}
                                    className="px-2 py-1.5 text-foreground"
                                  >
                                    {row.fields[m.targetField] || '—'}
                                  </td>
                                ))}
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>

                    {/* Duplicate warning */}
                    {duplicateCount > 0 && (
                      <div className="flex items-start gap-2 rounded-md border border-warning/30 bg-warning-light/20 px-3 py-2">
                        <AlertTriangle className="h-4 w-4 text-warning shrink-0 mt-0.5" />
                        <div>
                          <p className="text-xs font-medium text-foreground">
                            {t('crm.importExport.duplicatesDetected', { count: duplicateCount })}
                          </p>
                          <p className="text-[11px] text-muted-foreground mt-0.5">
                            {t('crm.importExport.duplicatesBasis')} {t('crm.importExport.duplicateAction')}: {duplicateAction === 'skip' ? t('crm.importExport.duplicateSkip') : duplicateAction === 'update' ? t('crm.importExport.duplicateUpdate') : t('crm.importExport.duplicateCreate')}.
                          </p>
                        </div>
                      </div>
                    )}

                    {/* Actions */}
                    <div className="flex justify-between pt-1">
                      <button
                        onClick={() => setImportStep('mapping')}
                        className="flex items-center gap-1.5 h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
                      >
                        <ArrowLeft className="h-3.5 w-3.5" />
                        {t('crm.importExport.stepMapping')}
                      </button>
                      <button
                        onClick={handleImport}
                        disabled={importing}
                        className="flex items-center gap-1.5 h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                      >
                        {importing ? (
                          <>
                            <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                            {t('crm.importExport.importing')}
                          </>
                        ) : (
                          <>
                            <Upload className="h-3.5 w-3.5" />
                            {t('crm.importExport.importCount', { count: previewRows.length - (duplicateAction === 'skip' ? duplicateCount : 0), entity: t(config.labelPluralKey) })}
                          </>
                        )}
                      </button>
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
        )}

        {/* ---- EXPORT TAB ---- */}
        {activeTab === 'export' && (
          <div className="space-y-4">
            {/* Format selector */}
            <div>
              <p className="text-xs font-medium text-foreground mb-2">{t('crm.importExport.format')}</p>
              <div className="flex items-center gap-2">
                {config.exportFormats.map((fmt) => {
                  const FmtIcon = formatIcons[fmt]
                  return (
                    <button
                      key={fmt}
                      onClick={() => setExportFormat(fmt)}
                      className={`flex items-center gap-1.5 rounded-md border px-3 py-2 text-xs transition-colors ${
                        exportFormat === fmt
                          ? 'border-primary bg-primary/5 text-primary font-medium'
                          : 'border-border text-muted-foreground hover:bg-accent'
                      }`}
                    >
                      <FmtIcon className="h-3.5 w-3.5" />
                      {formatLabels[fmt]}
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Field selection */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <p className="text-xs font-medium text-foreground">
                  {t('crm.importExport.fields')} ({exportFields.size} / {config.fields.length})
                </p>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setExportFields(new Set(config.fields.map((f) => f.key)))}
                    className="text-[10px] text-primary hover:underline"
                  >
                    {t('crm.importExport.selectAll')}
                  </button>
                  <button
                    onClick={() => setExportFields(new Set())}
                    className="text-[10px] text-muted-foreground hover:underline"
                  >
                    {t('crm.importExport.selectNone')}
                  </button>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-1.5">
                {config.fields.map((field) => (
                  <button
                    key={field.key}
                    onClick={() => toggleExportField(field.key)}
                    className={`flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs text-left transition-colors ${
                      exportFields.has(field.key)
                        ? 'border-primary/30 bg-primary/5 text-foreground'
                        : 'border-border text-muted-foreground hover:bg-accent'
                    }`}
                  >
                    <div
                      className={`flex h-3.5 w-3.5 items-center justify-center rounded-sm border ${
                        exportFields.has(field.key)
                          ? 'border-primary bg-primary'
                          : 'border-border'
                      }`}
                    >
                      {exportFields.has(field.key) && <Check className="h-2.5 w-2.5 text-white" />}
                    </div>
                    {t(field.labelKey)}
                  </button>
                ))}
              </div>
            </div>

            {/* Export summary */}
            <div className="rounded-md bg-secondary/50 px-3 py-2.5 flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-foreground">
                  {mockExportCounts[entity]} {t(config.labelPluralKey)}
                </p>
                <p className="text-[11px] text-muted-foreground">
                  {t('crm.importExport.exportSummary', { fields: exportFields.size, format: formatLabels[exportFormat] })}
                </p>
              </div>
              <button
                onClick={handleExport}
                disabled={exporting || exportFields.size === 0}
                className="flex items-center gap-1.5 h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {exporting ? (
                  <>
                    <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                    {t('crm.importExport.exporting')}
                  </>
                ) : (
                  <>
                    <Download className="h-3.5 w-3.5" />
                    {t('crm.importExport.export')}
                  </>
                )}
              </button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
