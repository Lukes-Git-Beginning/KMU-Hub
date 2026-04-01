/**
 * CompanyFormDialog — Create/Edit Company.
 *
 * Used from CompanyDetailPage and CompaniesListPage.
 * Mock-first: works with local state, ready for API swap.
 */
import { useState, useEffect } from 'react'
import {
  Building2,
  Globe,
  MapPin,
  Factory,
  Users,
  Phone,
  Mail,
  Plus,
  X,
} from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Industry options (DACH-relevant)
// ---------------------------------------------------------------------------

const industries = [
  'IT & Software',
  'Beratung',
  'Handel',
  'Produktion',
  'Bau & Immobilien',
  'Gesundheitswesen',
  'Bildung',
  'Gastronomie & Hotellerie',
  'Handwerk',
  'Logistik & Transport',
  'Finanzen & Versicherung',
  'Marketing & Medien',
  'Recht',
  'Landwirtschaft',
  'Energie',
  'Sonstige',
]

const companySizes = [
  { value: '1-10', label: '1-10 Mitarbeiter' },
  { value: '11-50', label: '11-50 Mitarbeiter' },
  { value: '51-200', label: '51-200 Mitarbeiter' },
  { value: '201-500', label: '201-500 Mitarbeiter' },
  { value: '500+', label: '500+ Mitarbeiter' },
]

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface CompanyFormData {
  name: string
  industry: string
  website: string
  phone: string
  email: string
  street: string
  zip: string
  city: string
  country: string
  size: string
  notes: string
  tags: string[]
}

interface CompanyFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialData?: Partial<CompanyFormData>
  onSubmit: (data: CompanyFormData) => void
  isEdit?: boolean
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function CompanyFormDialog({
  open,
  onOpenChange,
  initialData,
  onSubmit,
  isEdit = false,
}: CompanyFormDialogProps) {
  const [name, setName] = useState('')
  const [industry, setIndustry] = useState('')
  const [website, setWebsite] = useState('')
  const [phone, setPhone] = useState('')
  const [email, setEmail] = useState('')
  const [street, setStreet] = useState('')
  const [zip, setZip] = useState('')
  const [city, setCity] = useState('')
  const [country, setCountry] = useState('Schweiz')
  const [size, setSize] = useState('')
  const [notes, setNotes] = useState('')
  const [tags, setTags] = useState<string[]>([])
  const [tagInput, setTagInput] = useState('')

  useEffect(() => {
    if (open && initialData) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setName(initialData.name ?? '')
      setIndustry(initialData.industry ?? '')
      setWebsite(initialData.website ?? '')
      setPhone(initialData.phone ?? '')
      setEmail(initialData.email ?? '')
      setStreet(initialData.street ?? '')
      setZip(initialData.zip ?? '')
      setCity(initialData.city ?? '')
      setCountry(initialData.country ?? 'Schweiz')
      setSize(initialData.size ?? '')
      setNotes(initialData.notes ?? '')
      setTags(initialData.tags ?? [])
    } else if (open) {
      setName('')
      setIndustry('')
      setWebsite('')
      setPhone('')
      setEmail('')
      setStreet('')
      setZip('')
      setCity('')
      setCountry('Schweiz')
      setSize('')
      setNotes('')
      setTags([])
    }
  }, [open, initialData])

  const handleSubmit = () => {
    if (!name.trim()) return
    onSubmit({
      name: name.trim(),
      industry,
      website,
      phone,
      email,
      street,
      zip,
      city,
      country,
      size,
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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Unternehmen bearbeiten' : 'Neues Unternehmen'}</DialogTitle>
          <DialogDescription>
            {isEdit ? 'Unternehmensdaten aktualisieren.' : 'Erstelle einen neuen Firmeneintrag.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Name */}
          <div className="space-y-1.5">
            <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
              <Building2 className="h-3.5 w-3.5 text-muted-foreground" />
              Firmenname *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Muster AG"
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              autoFocus
            />
          </div>

          {/* Industry + Size */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Factory className="h-3.5 w-3.5 text-muted-foreground" />
                Branche
              </label>
              <select
                value={industry}
                onChange={(e) => setIndustry(e.target.value)}
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
              >
                <option value="">Auswählen...</option>
                {industries.map((i) => (
                  <option key={i} value={i}>{i}</option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Users className="h-3.5 w-3.5 text-muted-foreground" />
                Größe
              </label>
              <select
                value={size}
                onChange={(e) => setSize(e.target.value)}
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
              >
                <option value="">Auswählen...</option>
                {companySizes.map((s) => (
                  <option key={s.value} value={s.value}>{s.label}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Contact info */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Phone className="h-3.5 w-3.5 text-muted-foreground" />
                Telefon
              </label>
              <input
                type="text"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="+41 44 123 45 67"
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              />
            </div>
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Mail className="h-3.5 w-3.5 text-muted-foreground" />
                E-Mail
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="info@firma.ch"
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              />
            </div>
          </div>

          {/* Website */}
          <div className="space-y-1.5">
            <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
              <Globe className="h-3.5 w-3.5 text-muted-foreground" />
              Website
            </label>
            <input
              type="text"
              value={website}
              onChange={(e) => setWebsite(e.target.value)}
              placeholder="www.firma.ch"
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
            />
          </div>

          {/* Address */}
          <div className="space-y-1.5">
            <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
              <MapPin className="h-3.5 w-3.5 text-muted-foreground" />
              Adresse
            </label>
            <input
              type="text"
              value={street}
              onChange={(e) => setStreet(e.target.value)}
              placeholder="Strasse und Hausnummer"
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
            />
            <div className="grid grid-cols-[100px_1fr_1fr] gap-3">
              <input
                type="text"
                value={zip}
                onChange={(e) => setZip(e.target.value)}
                placeholder="PLZ"
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              />
              <input
                type="text"
                value={city}
                onChange={(e) => setCity(e.target.value)}
                placeholder="Ort"
                className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
              />
              <input
                type="text"
                value={country}
                onChange={(e) => setCountry(e.target.value)}
                placeholder="Land"
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
                placeholder="Tag hinzufügen..."
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
              placeholder="Interne Notizen..."
              rows={3}
              className="w-full rounded-md border border-border bg-transparent px-3 py-2 text-sm outline-none placeholder:text-muted-foreground focus:border-primary resize-none"
            />
          </div>

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
