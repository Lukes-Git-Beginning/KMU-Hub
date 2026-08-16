import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Upload, FileText, AlertCircle, CheckCircle2, X } from 'lucide-react'
import type { Contact } from '@/stores/contacts'

type ParsedContact = Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>

interface ImportContactsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImport: (contacts: ParsedContact[]) => number
}

/** Parse a single CSV line respecting quoted fields */
function parseCSVLine(line: string): string[] {
  const result: string[] = []
  let current = ''
  let inQuotes = false
  for (let i = 0; i < line.length; i++) {
    const ch = line[i]
    if (ch === '"') {
      inQuotes = !inQuotes
    } else if ((ch === ',' || ch === ';') && !inQuotes) {
      result.push(current.trim())
      current = ''
    } else {
      current += ch
    }
  }
  result.push(current.trim())
  return result
}

/** Map CSV headers to contact fields */
const FIELD_MAP: Record<string, keyof ParsedContact | string> = {
  vorname: 'firstName',
  nachname: 'lastName',
  firstname: 'firstName',
  lastname: 'lastName',
  first_name: 'firstName',
  last_name: 'lastName',
  email: 'email',
  'e-mail': 'email',
  telefon: 'phone',
  phone: 'phone',
  mobil: 'mobile',
  mobile: 'mobile',
  firma: 'company',
  company: 'company',
  unternehmen: 'company',
  position: 'jobTitle',
  jobtitle: 'jobTitle',
  job_title: 'jobTitle',
  titel: 'jobTitle',
  abteilung: 'department',
  department: 'department',
  strasse: 'address.street',
  street: 'address.street',
  plz: 'address.zip',
  zip: 'address.zip',
  ort: 'address.city',
  city: 'address.city',
  stadt: 'address.city',
  land: 'address.country',
  country: 'address.country',
  website: 'website',
  webseite: 'website',
  notizen: 'notes',
  notes: 'notes',
  anrede: 'salutation',
  salutation: 'salutation',
  kategorie: 'category',
  category: 'category',
}

function parseCSV(text: string): ParsedContact[] {
  const lines = text.split(/\r?\n/).filter((l) => l.trim())
  if (lines.length < 2) return []

  // Transliterate before stripping non-ASCII, or a German header loses its
  // umlaut entirely: "Straße" would become "strae" and match no key below.
  const headers = parseCSVLine(lines[0]).map((h) =>
    h
      .toLowerCase()
      .replace(/ä/g, 'ae')
      .replace(/ö/g, 'oe')
      .replace(/ü/g, 'ue')
      .replace(/ß/g, 'ss')
      .replace(/[^a-z_-]/g, ''),
  )
  const fieldIndices: { field: string; index: number }[] = []

  for (let i = 0; i < headers.length; i++) {
    const mapped = FIELD_MAP[headers[i]]
    if (mapped) fieldIndices.push({ field: mapped, index: i })
  }

  const contacts: ParsedContact[] = []
  for (let row = 1; row < lines.length; row++) {
    const values = parseCSVLine(lines[row])
    if (values.every((v) => !v)) continue

    const c: ParsedContact = {
      salutation: '' as const,
      firstName: '',
      lastName: '',
      email: '',
      phone: '',
      mobile: '',
      company: '',
      jobTitle: '',
      department: '',
      address: { street: '', zip: '', city: '', country: 'Deutschland' },
      website: '',
      category: 'customer',
      status: 'active',
      tags: [],
      notes: '',
      socialMedia: { linkedin: '', xing: '' },
      lastContact: new Date().toISOString().split('T')[0],
      projects: [],
      isFavorite: false,
    }

    for (const { field, index } of fieldIndices) {
      const val = values[index] || ''
      if (field.startsWith('address.')) {
        const sub = field.split('.')[1] as keyof typeof c.address
        c.address[sub] = val
      } else if (field === 'salutation') {
        c.salutation = val.includes('Frau') ? 'Frau' : val.includes('Herr') ? 'Herr' : ''
      } else if (field === 'category') {
        const lower = val.toLowerCase()
        c.category = lower.includes('partner') ? 'partner' : lower.includes('mitarbeiter') || lower.includes('employee') ? 'employee' : 'customer'
      } else {
        ;(c as Record<string, unknown>)[field] = val
      }
    }

    if (c.firstName || c.lastName || c.email) {
      contacts.push(c)
    }
  }
  return contacts
}

export function ImportContactsDialog({ open, onOpenChange, onImport }: ImportContactsDialogProps) {
  const { t } = useTranslation()
  const [file, setFile] = useState<File | null>(null)
  const [parsed, setParsed] = useState<ParsedContact[]>([])
  const [error, setError] = useState('')
  const [imported, setImported] = useState(false)
  const [importCount, setImportCount] = useState(0)

  const reset = () => {
    setFile(null)
    setParsed([])
    setError('')
    setImported(false)
    setImportCount(0)
  }

  const handleFile = useCallback(async (f: File) => {
    setFile(f)
    setError('')
    setParsed([])
    setImported(false)

    if (!f.name.endsWith('.csv')) {
      setError(t('kontakte.import.csvOnly'))
      return
    }

    try {
      const text = await f.text()
      const contacts = parseCSV(text)
      if (contacts.length === 0) {
        setError(t('kontakte.import.noContactsFound'))
      } else {
        setParsed(contacts)
      }
    } catch {
      setError(t('kontakte.import.readError'))
    }
  }, [])

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      const f = e.dataTransfer.files[0]
      if (f) handleFile(f)
    },
    [handleFile]
  )

  const handleImport = () => {
    const count = onImport(parsed)
    setImportCount(count)
    setImported(true)
  }

  const handleClose = () => {
    onOpenChange(false)
    setTimeout(reset, 200)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('kontakte.import.title')}</DialogTitle>
        </DialogHeader>

        {imported ? (
          <div className="py-8 text-center">
            <CheckCircle2 className="mx-auto h-12 w-12 text-green-500 mb-3" />
            <p className="text-base font-medium text-[var(--heading)]">
              {t('kontakte.import.importedCount', { count: importCount })}
            </p>
            <p className="text-sm text-[var(--muted)] mt-1">
              {t('kontakte.import.importedSuccess')}
            </p>
          </div>
        ) : (
          <div className="space-y-4 py-2">
            {/* Drop zone */}
            <div
              onDragOver={(e) => e.preventDefault()}
              onDrop={handleDrop}
              className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-border p-8 text-center hover:border-primary/50 hover:bg-secondary/30 transition-colors cursor-pointer"
              onClick={() => document.getElementById('csv-upload')?.click()}
            >
              <Upload className="h-8 w-8 text-[var(--muted)] mb-2" />
              <p className="text-sm font-medium text-[var(--body)]">
                {t('kontakte.import.dragCsv')}
              </p>
              <p className="text-xs text-[var(--muted)] mt-1">
                {t('kontakte.import.orClickToSelect')}
              </p>
              <input
                id="csv-upload"
                type="file"
                accept=".csv"
                className="hidden"
                onChange={(e) => {
                  const f = e.target.files?.[0]
                  if (f) handleFile(f)
                }}
              />
            </div>

            {/* File selected */}
            {file && !error && (
              <div className="flex items-center gap-2 rounded-md border border-border px-3 py-2">
                <FileText className="h-4 w-4 text-primary shrink-0" />
                <span className="flex-1 text-sm text-[var(--body)] truncate">{file.name}</span>
                <button onClick={reset} className="text-[var(--muted)] hover:text-[var(--body)]">
                  <X className="h-4 w-4" />
                </button>
              </div>
            )}

            {/* Error */}
            {error && (
              <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900/20 px-3 py-2">
                <AlertCircle className="h-4 w-4 text-red-500 shrink-0 mt-0.5" />
                <p className="text-sm text-red-700 dark:text-red-400">{error}</p>
              </div>
            )}

            {/* Preview */}
            {parsed.length > 0 && (
              <div>
                <h4 className="text-xs font-medium uppercase text-[var(--muted)] mb-2">
                  {t('kontakte.import.preview', { count: parsed.length })}
                </h4>
                <div className="max-h-48 overflow-y-auto rounded-md border border-border">
                  <table className="w-full text-xs">
                    <thead className="bg-secondary/50 sticky top-0">
                      <tr>
                        <th className="text-left px-2 py-1.5 font-medium text-[var(--muted)]">{t('kontakte.import.colName')}</th>
                        <th className="text-left px-2 py-1.5 font-medium text-[var(--muted)]">{t('kontakte.import.colEmail')}</th>
                        <th className="text-left px-2 py-1.5 font-medium text-[var(--muted)]">{t('kontakte.import.colCompany')}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {parsed.slice(0, 20).map((c, i) => (
                        <tr key={i} className="border-t border-border-muted">
                          <td className="px-2 py-1.5 text-[var(--body)]">
                            {c.firstName} {c.lastName}
                          </td>
                          <td className="px-2 py-1.5 text-[var(--muted)]">{c.email || '—'}</td>
                          <td className="px-2 py-1.5 text-[var(--muted)]">{c.company || '—'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {parsed.length > 20 && (
                    <p className="px-2 py-1.5 text-[10px] text-[var(--muted)] border-t border-border-muted">
                      {t('kontakte.import.andMore', { count: parsed.length - 20 })}
                    </p>
                  )}
                </div>
              </div>
            )}

            {/* Format hint */}
            <div className="rounded-md bg-secondary/50 px-3 py-2">
              <p className="text-xs text-[var(--muted)]">
                <strong>{t('kontakte.import.formatLabel')}:</strong> {t('kontakte.import.formatDescription')}
              </p>
            </div>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            {imported ? t('common.close') : t('common.cancel')}
          </Button>
          {!imported && parsed.length > 0 && (
            <Button onClick={handleImport}>
              <Upload className="mr-1.5 h-4 w-4" />
              {t('kontakte.import.importCount', { count: parsed.length })}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
