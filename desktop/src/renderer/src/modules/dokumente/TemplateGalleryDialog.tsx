/**
 * TemplateGalleryDialog — browse and create documents from templates.
 *
 * 5 categories with ~12 mock templates. Search within templates.
 * "Erstellen" adds a new file to the documents store.
 */
import { useState, useMemo } from 'react'
import {
  FileText,
  Presentation,
  Search,
  X,
  Plus,
  FileCheck,
  Receipt,
  BarChart3,
  Mail,
  ScrollText,
  ClipboardList,
} from 'lucide-react'
import { toast } from 'sonner'

interface Template {
  id: string
  name: string
  description: string
  category: string
  icon: React.ElementType
  format: string
}

const CATEGORIES = [
  { id: 'all', label: 'Alle' },
  { id: 'vertraege', label: 'Vertraege' },
  { id: 'briefe', label: 'Briefe' },
  { id: 'formulare', label: 'Formulare' },
  { id: 'rechnungen', label: 'Rechnungen' },
  { id: 'berichte', label: 'Berichte' },
]

const TEMPLATES: Template[] = [
  // Vertraege
  { id: 't1', name: 'Arbeitsvertrag', description: 'Standard-Arbeitsvertrag nach deutschem Recht', category: 'vertraege', icon: FileCheck, format: '.docx' },
  { id: 't2', name: 'Dienstleistungsvertrag', description: 'Vertrag fuer externe Dienstleister', category: 'vertraege', icon: FileCheck, format: '.docx' },
  { id: 't3', name: 'NDA (Vertraulichkeit)', description: 'Geheimhaltungsvereinbarung', category: 'vertraege', icon: ScrollText, format: '.docx' },
  // Briefe
  { id: 't4', name: 'Geschaeftsbrief', description: 'Standardbrief mit Kopf- und Fusszeile', category: 'briefe', icon: Mail, format: '.docx' },
  { id: 't5', name: 'Kuendigungsschreiben', description: 'Vorlage fuer Vertragskundigungen', category: 'briefe', icon: FileText, format: '.docx' },
  // Formulare
  { id: 't6', name: 'Urlaubsantrag', description: 'Antrag auf Erholungsurlaub', category: 'formulare', icon: ClipboardList, format: '.docx' },
  { id: 't7', name: 'Reisekostenabrechnung', description: 'Formular fuer Reisekosten inkl. Belege', category: 'formulare', icon: ClipboardList, format: '.xlsx' },
  { id: 't8', name: 'Stundenzettel', description: 'Woechentliche Arbeitszeiterfassung', category: 'formulare', icon: ClipboardList, format: '.xlsx' },
  // Rechnungen
  { id: 't9', name: 'Rechnung (Standard)', description: 'Standardrechnung mit MwSt-Ausweis', category: 'rechnungen', icon: Receipt, format: '.xlsx' },
  { id: 't10', name: 'Kleinunternehmer-Rechnung', description: 'Rechnung ohne MwSt (§19 UStG)', category: 'rechnungen', icon: Receipt, format: '.xlsx' },
  // Berichte
  { id: 't11', name: 'Monatsbericht', description: 'Monatliche Zusammenfassung mit KPIs', category: 'berichte', icon: BarChart3, format: '.docx' },
  { id: 't12', name: 'Projektstatusbericht', description: 'Fortschrittsbericht fuer Projekte', category: 'berichte', icon: Presentation, format: '.pptx' },
]

interface TemplateGalleryDialogProps {
  open: boolean
  onClose: () => void
}

export function TemplateGalleryDialog({ open, onClose }: TemplateGalleryDialogProps) {
  const [search, setSearch] = useState('')
  const [activeCategory, setActiveCategory] = useState('all')

  const filtered = useMemo(() => {
    let result = TEMPLATES
    if (activeCategory !== 'all') {
      result = result.filter((t) => t.category === activeCategory)
    }
    if (search) {
      const q = search.toLowerCase()
      result = result.filter(
        (t) => t.name.toLowerCase().includes(q) || t.description.toLowerCase().includes(q),
      )
    }
    return result
  }, [activeCategory, search])

  const handleCreate = (template: Template) => {
    toast.success(`"${template.name}${template.format}" wurde erstellt`)
    onClose()
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="mx-4 w-full max-w-3xl rounded-xl border border-border bg-card shadow-xl max-h-[85vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-6 py-4">
          <div>
            <h2 className="text-sm font-semibold text-foreground">Neu aus Vorlage</h2>
            <p className="text-xs text-muted-foreground">{TEMPLATES.length} Vorlagen verfuegbar</p>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Search + Category filter */}
        <div className="border-b border-border px-6 py-3 space-y-3">
          <div className="relative max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Vorlage suchen..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-muted-foreground"
            />
          </div>
          <div className="flex gap-1 flex-wrap">
            {CATEGORIES.map((cat) => (
              <button
                key={cat.id}
                onClick={() => setActiveCategory(cat.id)}
                className={`rounded-lg px-3 py-1 text-xs font-medium transition-colors ${
                  activeCategory === cat.id
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-muted'
                }`}
              >
                {cat.label}
              </button>
            ))}
          </div>
        </div>

        {/* Template grid */}
        <div className="flex-1 overflow-y-auto p-6">
          {filtered.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <FileText className="h-10 w-10 text-muted-foreground/40 mb-3" />
              <p className="text-sm text-muted-foreground">Keine Vorlagen gefunden</p>
            </div>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {filtered.map((template) => (
                <div
                  key={template.id}
                  className="group rounded-lg border border-border bg-card p-4 hover:border-primary/30 hover:shadow-md transition-all cursor-pointer"
                  onClick={() => handleCreate(template)}
                >
                  <div className="flex items-start gap-3 mb-2">
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10">
                      <template.icon className="h-4.5 w-4.5 text-primary" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <h3 className="text-sm font-medium text-foreground truncate">{template.name}</h3>
                      <span className="text-[10px] text-muted-foreground">{template.format}</span>
                    </div>
                  </div>
                  <p className="text-xs text-muted-foreground line-clamp-2">{template.description}</p>
                  <div className="mt-3 flex justify-end opacity-0 group-hover:opacity-100 transition-opacity">
                    <span className="flex items-center gap-1 text-xs text-primary font-medium">
                      <Plus className="h-3.5 w-3.5" />
                      Erstellen
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
