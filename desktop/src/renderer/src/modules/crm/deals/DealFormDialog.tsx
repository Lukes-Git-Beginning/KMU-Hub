/**
 * DealFormDialog -- Create/Edit Deal.
 *
 * Used from DealDetailPage and DealsListPage.
 * Mock-first: works with local state, ready for API swap.
 */
import { useState, useEffect } from 'react'
import {
  TrendingUp,
  Calendar,
  User,
  Building2,
  Banknote,
  Plus,
  X,
  Target,
  Percent,
} from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Pipeline stages
// ---------------------------------------------------------------------------

const pipelineStages = [
  { value: 'lead', label: 'Lead', color: '#94a3b8' },
  { value: 'qualified', label: 'Qualifiziert', color: '#60a5fa' },
  { value: 'proposal', label: 'Angebot', color: '#a78bfa' },
  { value: 'negotiation', label: 'Verhandlung', color: '#f59e0b' },
  { value: 'won', label: 'Gewonnen', color: '#22c55e' },
  { value: 'lost', label: 'Verloren', color: '#ef4444' },
]

const currencies = [
  { value: 'CHF', label: 'CHF (Schweizer Franken)' },
  { value: 'EUR', label: 'EUR (Euro)' },
  { value: 'USD', label: 'USD (US-Dollar)' },
]

const priorities = [
  { value: 'low', label: 'Niedrig' },
  { value: 'normal', label: 'Normal' },
  { value: 'high', label: 'Hoch' },
  { value: 'urgent', label: 'Dringend' },
]

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface DealFormData {
  name: string
  value: number
  currency: string
  stage: string
  priority: string
  probability: number
  contactName: string
  companyName: string
  expectedCloseDate: string
  notes: string
  tags: string[]
}

export interface PipelineStageOption {
  id: string
  name: string
  color: string
  probability?: number
}

interface DealFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialData?: Partial<DealFormData>
  onSubmit: (data: DealFormData) => void
  isEdit?: boolean
  stages?: PipelineStageOption[]
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function DealFormDialog({
  open,
  onOpenChange,
  initialData,
  onSubmit,
  isEdit = false,
  stages: stagesProp,
}: DealFormDialogProps) {
  const [name, setName] = useState('')
  const [value, setValue] = useState('')
  const [currency, setCurrency] = useState('CHF')
  const [stage, setStage] = useState('lead')
  const [priority, setPriority] = useState('normal')
  const [probability, setProbability] = useState('50')
  const [contactName, setContactName] = useState('')
  const [companyName, setCompanyName] = useState('')
  const [expectedCloseDate, setExpectedCloseDate] = useState('')
  const [notes, setNotes] = useState('')
  const [tags, setTags] = useState<string[]>([])
  const [tagInput, setTagInput] = useState('')

  useEffect(() => {
    if (open && initialData) {
      setName(initialData.name ?? '')
      setValue(initialData.value != null ? String(initialData.value) : '')
      setCurrency(initialData.currency ?? 'CHF')
      setStage(initialData.stage ?? 'lead')
      setPriority(initialData.priority ?? 'normal')
      setProbability(initialData.probability != null ? String(initialData.probability) : '50')
      setContactName(initialData.contactName ?? '')
      setCompanyName(initialData.companyName ?? '')
      setExpectedCloseDate(initialData.expectedCloseDate ?? '')
      setNotes(initialData.notes ?? '')
      setTags(initialData.tags ?? [])
    } else if (open) {
      setName('')
      setValue('')
      setCurrency('CHF')
      setStage(stagesProp?.[0]?.id ?? 'lead')
      setPriority('normal')
      setProbability(stagesProp?.[0]?.probability != null ? String(stagesProp[0].probability) : '50')
      setContactName('')
      setCompanyName('')
      setExpectedCloseDate('')
      setNotes('')
      setTags([])
    }
  }, [open, initialData])

  const handleSubmit = () => {
    if (!name.trim()) return
    onSubmit({
      name: name.trim(),
      value: Number(value) || 0,
      currency,
      stage,
      priority,
      probability: Number(probability) || 50,
      contactName: contactName.trim(),
      companyName: companyName.trim(),
      expectedCloseDate,
      notes,
      tags,
    })
    onOpenChange(false)
  }

  const addTag = () => {
    const t = tagInput.trim()
    if (t && !tags.includes(t)) {
      setTags((prev) => [...prev, t])
      setTagInput('')
    }
  }

  // Resolve which stages to render: API stages (prop) or hardcoded fallback
  const resolvedStages = stagesProp
    ? stagesProp.map((s) => ({ value: s.id, label: s.name, color: s.color, probability: s.probability }))
    : pipelineStages.map((s) => ({ value: s.value, label: s.label, color: s.color, probability: undefined }))

  // Auto-update probability based on stage
  const handleStageChange = (newStage: string) => {
    setStage(newStage)
    if (stagesProp) {
      const found = stagesProp.find((s) => s.id === newStage)
      setProbability(found?.probability != null ? String(found.probability) : '50')
    } else {
      const stageProbabilities: Record<string, string> = {
        lead: '10',
        qualified: '25',
        proposal: '50',
        negotiation: '75',
        won: '100',
        lost: '0',
      }
      setProbability(stageProbabilities[newStage] ?? '50')
    }
  }

  const selectedStageColor = resolvedStages.find((s) => s.value === stage)?.color ?? '#94a3b8'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Deal bearbeiten' : 'Neuer Deal'}</DialogTitle>
          <DialogDescription>
            {isEdit ? 'Deal-Daten aktualisieren.' : 'Erstelle eine neue Verkaufschance.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Deal name */}
          <div className="space-y-1.5">
            <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
              <Target className="h-3.5 w-3.5 text-muted-foreground" />
              Deal-Name *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="z.B. CRM-Lizenz ABC GmbH"
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              autoFocus
            />
          </div>

          {/* Value + Currency */}
          <div className="grid grid-cols-[1fr_140px] gap-3">
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Banknote className="h-3.5 w-3.5 text-muted-foreground" />
                Wert
              </label>
              <input
                type="number"
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder="25000"
                min="0"
                step="100"
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Waehrung</label>
              <select
                value={currency}
                onChange={(e) => setCurrency(e.target.value)}
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
              >
                {currencies.map((c) => (
                  <option key={c.value} value={c.value}>{c.value}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Stage + Priority */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <TrendingUp className="h-3.5 w-3.5 text-muted-foreground" />
                Phase
              </label>
              <div className="grid grid-cols-3 gap-1">
                {resolvedStages.map((s) => (
                  <button
                    key={s.value}
                    type="button"
                    onClick={() => handleStageChange(s.value)}
                    className={`rounded-md border px-2 py-1.5 text-xs font-medium transition-colors ${
                      stage === s.value
                        ? 'border-transparent text-white'
                        : 'border-border text-muted-foreground hover:bg-accent'
                    }`}
                    style={stage === s.value ? { backgroundColor: s.color } : undefined}
                  >
                    {s.label}
                  </button>
                ))}
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Prioritaet</label>
              <select
                value={priority}
                onChange={(e) => setPriority(e.target.value)}
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
              >
                {priorities.map((p) => (
                  <option key={p.value} value={p.value}>{p.label}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Probability + Expected Close */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Percent className="h-3.5 w-3.5 text-muted-foreground" />
                Wahrscheinlichkeit (%)
              </label>
              <div className="flex items-center gap-2">
                <input
                  type="range"
                  min="0"
                  max="100"
                  step="5"
                  value={probability}
                  onChange={(e) => setProbability(e.target.value)}
                  className="flex-1 accent-primary"
                />
                <span
                  className="w-10 rounded-md border border-border px-2 py-1 text-center text-xs font-medium"
                  style={{ color: selectedStageColor }}
                >
                  {probability}%
                </span>
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
                Erw. Abschluss
              </label>
              <input
                type="date"
                value={expectedCloseDate}
                onChange={(e) => setExpectedCloseDate(e.target.value)}
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
              />
            </div>
          </div>

          {/* Contact + Company */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <User className="h-3.5 w-3.5 text-muted-foreground" />
                Kontakt
              </label>
              <input
                type="text"
                value={contactName}
                onChange={(e) => setContactName(e.target.value)}
                placeholder="Thomas Weber"
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              />
            </div>
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Building2 className="h-3.5 w-3.5 text-muted-foreground" />
                Unternehmen
              </label>
              <input
                type="text"
                value={companyName}
                onChange={(e) => setCompanyName(e.target.value)}
                placeholder="ABC GmbH"
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              />
            </div>
          </div>

          {/* Tags */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Tags</label>
            {tags.length > 0 && (
              <div className="flex flex-wrap gap-1.5 mb-1">
                {tags.map((tag) => (
                  <span key={tag} className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary">
                    {tag}
                    <button onClick={() => setTags((t) => t.filter((x) => x !== tag))} className="rounded-full hover:bg-primary/20 p-0.5">
                      <X className="h-3 w-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}
            <div className="flex gap-2">
              <input
                type="text"
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addTag() } }}
                placeholder="Tag hinzufuegen..."
                className="h-9 flex-1 rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              />
              <button
                onClick={addTag}
                disabled={!tagInput.trim()}
                className="h-9 rounded-md border border-border px-3 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors disabled:opacity-50"
              >
                <Plus className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Notizen</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Interne Notizen zum Deal..."
              rows={3}
              className="w-full rounded-md border border-border bg-transparent px-3 py-2 text-sm outline-none placeholder:text-muted-foreground focus:border-primary resize-none"
            />
          </div>

          {/* Weighted value display */}
          {Number(value) > 0 && (
            <div className="rounded-lg border border-border bg-secondary/30 px-4 py-3">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Gewichteter Wert</span>
                <span className="font-medium text-foreground">
                  {new Intl.NumberFormat('de-CH', { style: 'currency', currency }).format(
                    (Number(value) * Number(probability)) / 100
                  )}
                </span>
              </div>
              <div className="mt-1 flex items-center gap-2">
                <div className="flex-1 h-1.5 rounded-full bg-border overflow-hidden">
                  <div
                    className="h-full rounded-full transition-all"
                    style={{
                      width: `${probability}%`,
                      backgroundColor: selectedStageColor,
                    }}
                  />
                </div>
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex justify-end gap-2 pt-1">
            <button
              onClick={() => onOpenChange(false)}
              className="h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
            >
              Abbrechen
            </button>
            <button
              onClick={handleSubmit}
              disabled={!name.trim()}
              className="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {isEdit ? 'Speichern' : 'Erstellen'}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
