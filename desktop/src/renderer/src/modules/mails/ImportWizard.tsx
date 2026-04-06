/**
 * 5-step contact import wizard.
 *
 * Steps:
 * 1. Upload - file picker with drag-and-drop (CSV or vCard)
 * 2. Preview - shows first 5 rows of CSV data (vCard skips to step 4)
 * 3. Field Mapping - map CSV columns to CRM fields
 * 4. Options - visibility and merge settings
 * 5. Confirm - summary, execute import, show results
 */
import { useState, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Upload, FileSpreadsheet, FileText, Check, AlertCircle, Loader2 } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  usePreviewImportCSV,
  useImportContactsCSV,
  useImportContactsVCard,
} from '@/api/hooks/contacts-import'
import type { ImportPreview, ImportResult } from '@/api/crm-types'
import { IMPORT_CRM_FIELDS } from '@/api/crm-types'

interface ImportWizardProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

type FileType = 'csv' | 'vcard'

export default function ImportWizard({ open, onOpenChange }: ImportWizardProps) {
  const { t } = useTranslation()

  const STEP_TITLES = [
    t('mails.import.steps.upload'),
    t('mails.import.steps.preview'),
    t('mails.import.steps.mapping'),
    t('mails.import.steps.options'),
    t('mails.import.steps.import'),
  ]

  const [step, setStep] = useState(0)
  const [file, setFile] = useState<File | null>(null)
  const [fileType, setFileType] = useState<FileType>('csv')
  const [preview, setPreview] = useState<ImportPreview | null>(null)
  const [fieldMapping, setFieldMapping] = useState<Record<string, string>>({})
  const [visibility, setVisibility] = useState<string>('shared')
  const [mergeByEmail, setMergeByEmail] = useState(true)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [dragActive, setDragActive] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const previewMutation = usePreviewImportCSV()
  const importCSVMutation = useImportContactsCSV()
  const importVCardMutation = useImportContactsVCard()

  const isImporting = importCSVMutation.isPending || importVCardMutation.isPending

  const reset = useCallback(() => {
    setStep(0)
    setFile(null)
    setFileType('csv')
    setPreview(null)
    setFieldMapping({})
    setVisibility('shared')
    setMergeByEmail(true)
    setResult(null)
    previewMutation.reset()
    importCSVMutation.reset()
    importVCardMutation.reset()
  }, [previewMutation, importCSVMutation, importVCardMutation])

  function handleClose() {
    reset()
    onOpenChange(false)
  }

  function detectFileType(f: File): FileType {
    const ext = f.name.toLowerCase().split('.').pop() ?? ''
    if (ext === 'vcf' || ext === 'vcard') return 'vcard'
    return 'csv'
  }

  function handleFileSelect(f: File) {
    const type = detectFileType(f)
    setFile(f)
    setFileType(type)
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragActive(false)
    const f = e.dataTransfer.files[0]
    if (f) handleFileSelect(f)
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault()
    setDragActive(true)
  }

  function handleDragLeave(e: React.DragEvent) {
    e.preventDefault()
    setDragActive(false)
  }

  async function handleNext() {
    if (step === 0 && file) {
      if (fileType === 'vcard') {
        // vCard skips preview and mapping, go straight to options
        setStep(3)
      } else {
        // CSV: preview first
        try {
          const previewData = await previewMutation.mutateAsync(file)
          setPreview(previewData)
          // Initialize field mapping from detected mapping
          setFieldMapping({ ...previewData.detected_mapping })
          setStep(1)
        } catch {
          // Error shown via mutation state
        }
      }
    } else if (step === 1) {
      setStep(2)
    } else if (step === 2) {
      setStep(3)
    } else if (step === 3) {
      setStep(4)
    }
  }

  async function handleImport() {
    if (!file) return

    try {
      let importResult: ImportResult
      if (fileType === 'csv') {
        // Filter out ignored fields
        const cleanMapping: Record<string, string> = {}
        for (const [csvCol, crmField] of Object.entries(fieldMapping)) {
          if (crmField !== '__ignore') {
            cleanMapping[csvCol] = crmField
          }
        }
        importResult = await importCSVMutation.mutateAsync({
          file,
          fieldMapping: cleanMapping,
          visibility,
          mergeByEmail,
        })
      } else {
        importResult = await importVCardMutation.mutateAsync({
          file,
          visibility,
          mergeByEmail,
        })
      }
      setResult(importResult)
    } catch {
      // Error shown via mutation state
    }
  }

  function handleMappingChange(csvColumn: string, crmField: string) {
    setFieldMapping((prev) => ({ ...prev, [csvColumn]: crmField }))
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('mails.import.title')}</DialogTitle>
          <DialogDescription>
            {STEP_TITLES[step]} ({t('mails.import.stepOf', { current: step + 1, total: 5 })})
          </DialogDescription>
        </DialogHeader>

        {/* Step indicators */}
        <div className="flex items-center gap-1 mb-4">
          {STEP_TITLES.map((_, i) => (
            <div
              key={i}
              className={`h-1 flex-1 rounded-full transition-colors ${
                i <= step ? 'bg-primary' : 'bg-muted'
              }`}
            />
          ))}
        </div>

        {/* Step 0: Upload */}
        {step === 0 && (
          <div className="space-y-4">
            <div
              className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors cursor-pointer ${
                dragActive
                  ? 'border-primary bg-primary/5'
                  : 'border-muted-foreground/25 hover:border-primary/50'
              }`}
              onDrop={handleDrop}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onClick={() => fileInputRef.current?.click()}
            >
              <Upload className="mx-auto h-10 w-10 text-muted-foreground mb-3" />
              <p className="text-sm font-medium text-foreground">
                {t('mails.import.dropzone')}
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                {t('mails.import.fileTypes')}
              </p>
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv,.vcf,.vcard"
                className="hidden"
                onChange={(e) => {
                  const f = e.target.files?.[0]
                  if (f) handleFileSelect(f)
                }}
              />
            </div>

            {file && (
              <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/50">
                {fileType === 'csv' ? (
                  <FileSpreadsheet className="h-5 w-5 text-green-600" />
                ) : (
                  <FileText className="h-5 w-5 text-blue-600" />
                )}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{file.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {fileType === 'csv' ? t('mails.import.csvFile') : t('mails.import.vcardFile')} ({(file.size / 1024).toFixed(1)} KB)
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    setFile(null)
                  }}
                >
                  {t('mails.import.remove')}
                </Button>
              </div>
            )}

            {previewMutation.isError && (
              <div className="flex items-center gap-2 text-sm text-destructive">
                <AlertCircle className="h-4 w-4" />
                {t('mails.import.readError')}
              </div>
            )}
          </div>
        )}

        {/* Step 1: Preview */}
        {step === 1 && preview && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t('mails.import.previewDescription')}
            </p>
            <div className="border rounded-md overflow-auto max-h-64">
              <Table>
                <TableHeader>
                  <TableRow>
                    {preview.columns.map((col, i) => (
                      <TableHead key={i} className="whitespace-nowrap text-xs">
                        {col}
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {preview.sample_rows.map((row, ri) => (
                    <TableRow key={ri}>
                      {row.map((cell, ci) => (
                        <TableCell key={ci} className="text-xs whitespace-nowrap">
                          {cell || <span className="text-muted-foreground">-</span>}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            <p className="text-xs text-muted-foreground">
              {t('mails.import.columnsDetected', { columns: preview.columns.length, mapped: Object.keys(preview.detected_mapping).length })}
            </p>
          </div>
        )}

        {/* Step 2: Field Mapping */}
        {step === 2 && preview && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t('mails.import.mappingDescription')}
            </p>
            <div className="space-y-3 max-h-64 overflow-y-auto">
              {preview.columns.map((col) => (
                <div key={col} className="flex items-center gap-3">
                  <span className="text-sm font-medium w-40 truncate" title={col}>
                    {col}
                  </span>
                  <Select
                    value={fieldMapping[col] ?? '__ignore'}
                    onValueChange={(val) => handleMappingChange(col, val)}
                  >
                    <SelectTrigger className="w-48">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {IMPORT_CRM_FIELDS.map((field) => (
                        <SelectItem key={field.key} value={field.key}>
                          {field.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Step 3: Options */}
        {step === 3 && (
          <div className="space-y-6">
            <div className="space-y-3">
              <Label className="text-sm font-medium">{t('mails.import.visibility')}</Label>
              <Select value={visibility} onValueChange={setVisibility}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="shared">{t('mails.import.shared')}</SelectItem>
                  <SelectItem value="personal">{t('mails.import.personal')}</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {visibility === 'shared'
                  ? t('mails.import.sharedDescription')
                  : t('mails.import.personalDescription')}
              </p>
            </div>

            <div className="flex items-start gap-3">
              <Checkbox
                id="merge"
                checked={mergeByEmail}
                onCheckedChange={(checked) => setMergeByEmail(checked === true)}
              />
              <div>
                <Label htmlFor="merge" className="text-sm font-medium cursor-pointer">
                  {t('mails.import.mergeDuplicates')}
                </Label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {t('mails.import.mergeDescription')}
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Step 4: Confirm + Import */}
        {step === 4 && (
          <div className="space-y-4">
            {!result && !isImporting && (
              <>
                <div className="p-4 rounded-lg bg-muted/50 space-y-2">
                  <p className="text-sm">
                    <span className="font-medium">{t('mails.import.confirmFile')}:</span> {file?.name}
                  </p>
                  <p className="text-sm">
                    <span className="font-medium">{t('mails.export.format')}:</span>{' '}
                    {fileType === 'csv' ? 'CSV' : 'vCard'}
                  </p>
                  <p className="text-sm">
                    <span className="font-medium">{t('mails.import.visibility')}:</span>{' '}
                    {visibility === 'shared' ? t('mails.import.sharedLabel') : t('mails.import.personalLabel')}
                  </p>
                  <p className="text-sm">
                    <span className="font-medium">{t('mails.import.mergeDuplicates')}:</span>{' '}
                    {mergeByEmail ? t('common.yes') : t('common.no')}
                  </p>
                  {fileType === 'csv' && (
                    <p className="text-sm">
                      <span className="font-medium">{t('mails.import.mappedFields')}:</span>{' '}
                      {Object.values(fieldMapping).filter((v) => v !== '__ignore').length}
                    </p>
                  )}
                </div>
                <Button onClick={handleImport} className="w-full">
                  {t('mails.import.importBtn')}
                </Button>
              </>
            )}

            {isImporting && (
              <div className="flex flex-col items-center gap-3 py-8">
                <Loader2 className="h-8 w-8 animate-spin text-primary" />
                <p className="text-sm text-muted-foreground">{t('mails.import.importing')}</p>
              </div>
            )}

            {result && (
              <div className="space-y-4">
                <div className="flex items-center gap-2 text-green-600">
                  <Check className="h-5 w-5" />
                  <span className="font-medium">{t('mails.import.completed')}</span>
                </div>

                <div className="grid grid-cols-3 gap-3">
                  <div className="p-3 rounded-lg bg-muted/50 text-center">
                    <p className="text-2xl font-bold">{result.imported_count}</p>
                    <p className="text-xs text-muted-foreground">{t('mails.import.imported')}</p>
                  </div>
                  <div className="p-3 rounded-lg bg-muted/50 text-center">
                    <p className="text-2xl font-bold">{result.merged_count}</p>
                    <p className="text-xs text-muted-foreground">{t('mails.import.merged')}</p>
                  </div>
                  <div className="p-3 rounded-lg bg-muted/50 text-center">
                    <p className="text-2xl font-bold">{result.skipped_count}</p>
                    <p className="text-xs text-muted-foreground">{t('mails.import.skipped')}</p>
                  </div>
                </div>

                {result.errors && result.errors.length > 0 && (
                  <div className="space-y-2">
                    <p className="text-sm font-medium text-destructive">
                      {t('mails.import.errorCount', { count: result.errors.length })}:
                    </p>
                    <div className="max-h-32 overflow-y-auto space-y-1">
                      {result.errors.map((err, i) => (
                        <p key={i} className="text-xs text-muted-foreground">
                          {t('mails.import.errorRow', { row: err.row })}: {err.reason}
                        </p>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {(importCSVMutation.isError || importVCardMutation.isError) && (
              <div className="flex items-center gap-2 text-sm text-destructive">
                <AlertCircle className="h-4 w-4" />
                {t('mails.import.failed')}
              </div>
            )}
          </div>
        )}

        <DialogFooter>
          {step < 4 && (
            <>
              <Button variant="outline" onClick={handleClose}>
                {t('common.cancel')}
              </Button>
              {step > 0 && (
                <Button
                  variant="outline"
                  onClick={() => setStep((s) => Math.max(0, s - 1))}
                >
                  {t('common.back')}
                </Button>
              )}
              <Button
                onClick={handleNext}
                disabled={step === 0 && !file}
              >
                {previewMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    {t('common.loading')}
                  </>
                ) : (
                  t('common.next')
                )}
              </Button>
            </>
          )}
          {step === 4 && result && (
            <Button onClick={handleClose}>{t('mails.import.done')}</Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
