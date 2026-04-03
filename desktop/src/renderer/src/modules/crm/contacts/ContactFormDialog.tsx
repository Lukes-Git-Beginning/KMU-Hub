/**
 * ContactFormDialog — Create/Edit Contact.
 *
 * Used from ContactDetailPage and ContactsListPage.
 * Stores extra fields (salutation, mobile, address, social media, etc.)
 * in custom_fields with `_`-prefixed keys.
 */
import { useState, useEffect } from 'react'
import {
  User,
  Mail,
  Phone,
  Smartphone,
  Building2,
  Briefcase,
  MapPin,
  Globe,
  Plus,
  X,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ContactFormData {
  salutation: string
  title: string
  firstName: string
  lastName: string
  email: string
  phone: string
  mobile: string
  company: string
  jobTitle: string
  department: string
  category: string
  status: string
  street: string
  zip: string
  city: string
  country: string
  website: string
  linkedin: string
  xing: string
  notes: string
  tags: string[]
}

interface ContactFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialData?: Partial<ContactFormData>
  onSubmit: (data: ContactFormData) => void
  isEdit?: boolean
}

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

const salutations = ['', 'Herr', 'Frau']
const categories = [
  { value: 'customer', label: 'Kunde' },
  { value: 'employee', label: 'Mitarbeiter' },
  { value: 'partner', label: 'Partner' },
]
const statuses = [
  { value: 'active', label: 'Aktiv' },
  { value: 'prospect', label: 'Interessent' },
  { value: 'inactive', label: 'Inaktiv' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ContactFormDialog({
  open,
  onOpenChange,
  initialData,
  onSubmit,
  isEdit = false,
}: ContactFormDialogProps) {
  const [salutation, setSalutation] = useState('')
  const [title, setTitle] = useState('')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [mobile, setMobile] = useState('')
  const [company, setCompany] = useState('')
  const [jobTitle, setJobTitle] = useState('')
  const [department, setDepartment] = useState('')
  const [category, setCategory] = useState('customer')
  const [status, setStatus] = useState('active')
  const [street, setStreet] = useState('')
  const [zip, setZip] = useState('')
  const [city, setCity] = useState('')
  const [country, setCountry] = useState('Deutschland')
  const [website, setWebsite] = useState('')
  const [linkedin, setLinkedin] = useState('')
  const [xing, setXing] = useState('')
  const [notes, setNotes] = useState('')
  const [tags, setTags] = useState<string[]>([])
  const [tagInput, setTagInput] = useState('')
  const [showMore, setShowMore] = useState(false)

  useEffect(() => {
    if (open && initialData) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setSalutation(initialData.salutation ?? '')
      setTitle(initialData.title ?? '')
      setFirstName(initialData.firstName ?? '')
      setLastName(initialData.lastName ?? '')
      setEmail(initialData.email ?? '')
      setPhone(initialData.phone ?? '')
      setMobile(initialData.mobile ?? '')
      setCompany(initialData.company ?? '')
      setJobTitle(initialData.jobTitle ?? '')
      setDepartment(initialData.department ?? '')
      setCategory(initialData.category ?? 'customer')
      setStatus(initialData.status ?? 'active')
      setStreet(initialData.street ?? '')
      setZip(initialData.zip ?? '')
      setCity(initialData.city ?? '')
      setCountry(initialData.country ?? 'Deutschland')
      setWebsite(initialData.website ?? '')
      setLinkedin(initialData.linkedin ?? '')
      setXing(initialData.xing ?? '')
      setNotes(initialData.notes ?? '')
      setTags(initialData.tags ?? [])
      // Expand "more" section if any extra fields are filled
      const hasExtra = !!(
        initialData.street ||
        initialData.zip ||
        initialData.city ||
        initialData.website ||
        initialData.linkedin ||
        initialData.xing
      )
      setShowMore(hasExtra)
    } else if (open) {
      setSalutation('')
      setTitle('')
      setFirstName('')
      setLastName('')
      setEmail('')
      setPhone('')
      setMobile('')
      setCompany('')
      setJobTitle('')
      setDepartment('')
      setCategory('customer')
      setStatus('active')
      setStreet('')
      setZip('')
      setCity('')
      setCountry('Deutschland')
      setWebsite('')
      setLinkedin('')
      setXing('')
      setNotes('')
      setTags([])
      setShowMore(false)
    }
  }, [open, initialData])

  const handleSubmit = () => {
    if (!firstName.trim() || !lastName.trim()) return
    onSubmit({
      salutation,
      title: title.trim(),
      firstName: firstName.trim(),
      lastName: lastName.trim(),
      email,
      phone,
      mobile,
      company: company.trim(),
      jobTitle: jobTitle.trim(),
      department: department.trim(),
      category,
      status,
      street,
      zip,
      city,
      country,
      website,
      linkedin,
      xing,
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

  const inputClass =
    'h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground focus:border-primary'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? 'Kontakt bearbeiten' : 'Neuer Kontakt'}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Kontaktdaten aktualisieren.'
              : 'Erstelle einen neuen Kontakt.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* Salutation + Title */}
          <div className="grid grid-cols-[120px_1fr] gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                Anrede
              </label>
              <select
                value={salutation}
                onChange={(e) => setSalutation(e.target.value)}
                className={inputClass}
              >
                {salutations.map((s) => (
                  <option key={s} value={s}>
                    {s || 'Keine'}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                Titel
              </label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="z.B. Dr., Prof."
                className={inputClass}
              />
            </div>
          </div>

          {/* First + Last name */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <User className="h-3.5 w-3.5 text-muted-foreground" />
                Vorname *
              </label>
              <input
                type="text"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                placeholder="Max"
                className={inputClass}
                autoFocus
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                Nachname *
              </label>
              <input
                type="text"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                placeholder="Muster"
                className={inputClass}
              />
            </div>
          </div>

          {/* Email + Phone */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Mail className="h-3.5 w-3.5 text-muted-foreground" />
                E-Mail
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="max@firma.de"
                className={inputClass}
              />
            </div>
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Phone className="h-3.5 w-3.5 text-muted-foreground" />
                Telefon
              </label>
              <input
                type="text"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="+49 30 123 45 67"
                className={inputClass}
              />
            </div>
          </div>

          {/* Mobile + Company */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Smartphone className="h-3.5 w-3.5 text-muted-foreground" />
                Mobil
              </label>
              <input
                type="text"
                value={mobile}
                onChange={(e) => setMobile(e.target.value)}
                placeholder="+49 170 123 4567"
                className={inputClass}
              />
            </div>
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Building2 className="h-3.5 w-3.5 text-muted-foreground" />
                Unternehmen
              </label>
              <input
                type="text"
                value={company}
                onChange={(e) => setCompany(e.target.value)}
                placeholder="Muster AG"
                className={inputClass}
              />
            </div>
          </div>

          {/* Job title + Department */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="flex items-center gap-1.5 text-sm font-medium text-foreground">
                <Briefcase className="h-3.5 w-3.5 text-muted-foreground" />
                Position
              </label>
              <input
                type="text"
                value={jobTitle}
                onChange={(e) => setJobTitle(e.target.value)}
                placeholder="Geschaeftsfuehrer"
                className={inputClass}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                Abteilung
              </label>
              <input
                type="text"
                value={department}
                onChange={(e) => setDepartment(e.target.value)}
                placeholder="Vertrieb"
                className={inputClass}
              />
            </div>
          </div>

          {/* Category + Status */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                Kategorie
              </label>
              <select
                value={category}
                onChange={(e) => setCategory(e.target.value)}
                className={inputClass}
              >
                {categories.map((c) => (
                  <option key={c.value} value={c.value}>
                    {c.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                Status
              </label>
              <select
                value={status}
                onChange={(e) => setStatus(e.target.value)}
                className={inputClass}
              >
                {statuses.map((s) => (
                  <option key={s.value} value={s.value}>
                    {s.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Expandable "Mehr" section */}
          <button
            type="button"
            onClick={() => setShowMore(!showMore)}
            className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            {showMore ? (
              <ChevronUp className="h-3.5 w-3.5" />
            ) : (
              <ChevronDown className="h-3.5 w-3.5" />
            )}
            {showMore ? 'Weniger anzeigen' : 'Adresse, Website & Social Media'}
          </button>

          {showMore && (
            <div className="space-y-4 border-l-2 border-border pl-4">
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
                  className={inputClass}
                />
                <div className="grid grid-cols-[100px_1fr_1fr] gap-3">
                  <input
                    type="text"
                    value={zip}
                    onChange={(e) => setZip(e.target.value)}
                    placeholder="PLZ"
                    className={inputClass}
                  />
                  <input
                    type="text"
                    value={city}
                    onChange={(e) => setCity(e.target.value)}
                    placeholder="Ort"
                    className={inputClass}
                  />
                  <input
                    type="text"
                    value={country}
                    onChange={(e) => setCountry(e.target.value)}
                    placeholder="Land"
                    className={inputClass}
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
                  placeholder="www.firma.de"
                  className={inputClass}
                />
              </div>

              {/* Social Media */}
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">
                    LinkedIn
                  </label>
                  <input
                    type="text"
                    value={linkedin}
                    onChange={(e) => setLinkedin(e.target.value)}
                    placeholder="linkedin.com/in/..."
                    className={inputClass}
                  />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium text-foreground">
                    XING
                  </label>
                  <input
                    type="text"
                    value={xing}
                    onChange={(e) => setXing(e.target.value)}
                    placeholder="xing.com/profile/..."
                    className={inputClass}
                  />
                </div>
              </div>
            </div>
          )}

          {/* Tags */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Tags</label>
            {tags.length > 0 && (
              <div className="flex flex-wrap gap-1.5 mb-1">
                {tags.map((tag) => (
                  <span
                    key={tag}
                    className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary"
                  >
                    {tag}
                    <button
                      onClick={() =>
                        setTags((t) => t.filter((x) => x !== tag))
                      }
                      className="rounded-full hover:bg-primary/20 p-0.5"
                    >
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
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    addTag()
                  }
                }}
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
            <label className="text-sm font-medium text-foreground">
              Notizen
            </label>
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
              disabled={!firstName.trim() || !lastName.trim()}
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
