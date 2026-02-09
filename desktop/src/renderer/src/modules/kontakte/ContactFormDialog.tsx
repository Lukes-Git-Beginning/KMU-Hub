import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ChevronDown, ChevronUp, Plus, X, Globe, Linkedin } from 'lucide-react'
import type { Contact } from '@/stores/contacts'

interface ContactFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  contact?: Contact | null
  onSubmit: (data: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>) => void
}

const categories = [
  { value: 'employee', label: 'Mitarbeiter' },
  { value: 'customer', label: 'Kunde' },
  { value: 'partner', label: 'Partner' },
]

const statuses = [
  { value: 'active', label: 'Aktiv' },
  { value: 'prospect', label: 'Interessent' },
  { value: 'inactive', label: 'Inaktiv' },
]

const salutations = [
  { value: '', label: 'Keine Anrede' },
  { value: 'Herr', label: 'Herr' },
  { value: 'Frau', label: 'Frau' },
]

const availableTags = [
  'VIP', 'Entscheider', 'Technik', 'Design', 'Marketing', 'Sales',
  'Finanzen', 'Recht', 'HR', 'Frontend', 'Backend', 'Security',
  'Extern', 'DSGVO', 'Hosting', 'SLA', 'Beratung', 'Bau',
  'Content', 'Projektleitung', 'React', 'CISSP',
]

export function ContactFormDialog({ open, onOpenChange, contact, onSubmit }: ContactFormDialogProps) {
  const isEdit = !!contact

  const [salutation, setSalutation] = useState<'Herr' | 'Frau' | ''>('')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [mobile, setMobile] = useState('')
  const [company, setCompany] = useState('')
  const [jobTitle, setJobTitle] = useState('')
  const [department, setDepartment] = useState('')
  const [street, setStreet] = useState('')
  const [zip, setZip] = useState('')
  const [city, setCity] = useState('')
  const [country, setCountry] = useState('Schweiz')
  const [website, setWebsite] = useState('')
  const [category, setCategory] = useState<'employee' | 'customer' | 'partner'>('customer')
  const [status, setStatus] = useState<'active' | 'prospect' | 'inactive'>('active')
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [notes, setNotes] = useState('')
  const [linkedin, setLinkedin] = useState('')
  const [xing, setXing] = useState('')
  const [showExtras, setShowExtras] = useState(false)
  const [tagSearch, setTagSearch] = useState('')

  useEffect(() => {
    if (contact) {
      setSalutation(contact.salutation)
      setFirstName(contact.firstName)
      setLastName(contact.lastName)
      setEmail(contact.email)
      setPhone(contact.phone)
      setMobile(contact.mobile)
      setCompany(contact.company)
      setJobTitle(contact.jobTitle)
      setDepartment(contact.department)
      setStreet(contact.address.street)
      setZip(contact.address.zip)
      setCity(contact.address.city)
      setCountry(contact.address.country)
      setWebsite(contact.website)
      setCategory(contact.category)
      setStatus(contact.status)
      setSelectedTags(contact.tags)
      setNotes(contact.notes)
      setLinkedin(contact.socialMedia.linkedin)
      setXing(contact.socialMedia.xing)
    } else {
      setSalutation('')
      setFirstName('')
      setLastName('')
      setEmail('')
      setPhone('')
      setMobile('')
      setCompany('')
      setJobTitle('')
      setDepartment('')
      setStreet('')
      setZip('')
      setCity('')
      setCountry('Schweiz')
      setWebsite('')
      setCategory('customer')
      setStatus('active')
      setSelectedTags([])
      setNotes('')
      setLinkedin('')
      setXing('')
      setShowExtras(false)
    }
  }, [contact, open])

  const filteredTags = availableTags.filter(
    (t) =>
      !selectedTags.includes(t) &&
      t.toLowerCase().includes(tagSearch.toLowerCase())
  )

  const handleSubmit = () => {
    if (!firstName.trim() || !lastName.trim()) return
    onSubmit({
      salutation,
      firstName: firstName.trim(),
      lastName: lastName.trim(),
      email,
      phone,
      mobile,
      company,
      jobTitle,
      department,
      address: { street, zip, city, country },
      website,
      category,
      status,
      tags: selectedTags,
      notes,
      socialMedia: { linkedin, xing },
      lastContact: contact?.lastContact || new Date().toISOString().split('T')[0],
      projects: contact?.projects || [],
      isFavorite: contact?.isFavorite || false,
    })
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Kontakt bearbeiten' : 'Neuer Kontakt'}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Salutation + Name */}
          <div className="grid grid-cols-[100px_1fr_1fr] gap-3">
            <div className="space-y-1.5">
              <Label>Anrede</Label>
              <Select value={salutation} onValueChange={(v) => setSalutation(v as typeof salutation)}>
                <SelectTrigger><SelectValue placeholder="—" /></SelectTrigger>
                <SelectContent>
                  {salutations.map((s) => (
                    <SelectItem key={s.value} value={s.value || '_none'}>{s.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Vorname *</Label>
              <Input
                placeholder="Max"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                autoFocus
              />
            </div>
            <div className="space-y-1.5">
              <Label>Nachname *</Label>
              <Input
                placeholder="Mustermann"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
              />
            </div>
          </div>

          {/* Email + Phone */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>E-Mail</Label>
              <Input
                type="email"
                placeholder="max@firma.ch"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Telefon</Label>
              <Input
                placeholder="+41 44 123 45 67"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
              />
            </div>
          </div>

          {/* Mobile + Company */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Mobil</Label>
              <Input
                placeholder="+41 79 123 45 67"
                value={mobile}
                onChange={(e) => setMobile(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Firma</Label>
              <Input
                placeholder="Firma AG"
                value={company}
                onChange={(e) => setCompany(e.target.value)}
              />
            </div>
          </div>

          {/* Job Title + Department */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Position</Label>
              <Input
                placeholder="Geschaeftsfuehrer"
                value={jobTitle}
                onChange={(e) => setJobTitle(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Abteilung</Label>
              <Input
                placeholder="Entwicklung"
                value={department}
                onChange={(e) => setDepartment(e.target.value)}
              />
            </div>
          </div>

          {/* Category + Status */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Kategorie</Label>
              <Select value={category} onValueChange={(v) => setCategory(v as typeof category)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {categories.map((c) => (
                    <SelectItem key={c.value} value={c.value}>{c.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Status</Label>
              <Select value={status} onValueChange={(v) => setStatus(v as typeof status)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {statuses.map((s) => (
                    <SelectItem key={s.value} value={s.value}>{s.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Tags */}
          <div className="space-y-1.5">
            <Label>Tags</Label>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {selectedTags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary"
                >
                  {tag}
                  <button
                    onClick={() => setSelectedTags((t) => t.filter((x) => x !== tag))}
                    className="ml-0.5 rounded-full hover:bg-primary/20 p-0.5"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            <Input
              placeholder="Tag suchen oder hinzufuegen..."
              value={tagSearch}
              onChange={(e) => setTagSearch(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && tagSearch.trim() && !selectedTags.includes(tagSearch.trim())) {
                  setSelectedTags((t) => [...t, tagSearch.trim()])
                  setTagSearch('')
                }
              }}
              className="text-sm"
            />
            {tagSearch && filteredTags.length > 0 && (
              <div className="mt-1 max-h-28 overflow-y-auto rounded-md border bg-card p-1">
                {filteredTags.slice(0, 8).map((tag) => (
                  <button
                    key={tag}
                    onClick={() => {
                      setSelectedTags((t) => [...t, tag])
                      setTagSearch('')
                    }}
                    className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-secondary"
                  >
                    {tag}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Expandable extras */}
          <button
            onClick={() => setShowExtras(!showExtras)}
            className="flex items-center gap-1 text-sm text-primary hover:underline"
          >
            {showExtras ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
            {showExtras ? 'Weniger Optionen' : 'Adresse, Social Media & mehr'}
          </button>

          {showExtras && (
            <div className="space-y-4 rounded-lg border border-border p-3">
              {/* Address */}
              <div className="space-y-3">
                <Label className="text-xs font-medium uppercase text-muted-foreground">Adresse</Label>
                <Input
                  placeholder="Strasse und Hausnummer"
                  value={street}
                  onChange={(e) => setStreet(e.target.value)}
                />
                <div className="grid grid-cols-[100px_1fr_1fr] gap-3">
                  <Input
                    placeholder="PLZ"
                    value={zip}
                    onChange={(e) => setZip(e.target.value)}
                  />
                  <Input
                    placeholder="Ort"
                    value={city}
                    onChange={(e) => setCity(e.target.value)}
                  />
                  <Input
                    placeholder="Land"
                    value={country}
                    onChange={(e) => setCountry(e.target.value)}
                  />
                </div>
              </div>

              {/* Website */}
              <div className="space-y-1.5">
                <Label className="flex items-center gap-1">
                  <Globe className="h-3.5 w-3.5" />
                  Website
                </Label>
                <Input
                  placeholder="www.firma.ch"
                  value={website}
                  onChange={(e) => setWebsite(e.target.value)}
                />
              </div>

              {/* Social Media */}
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="flex items-center gap-1">
                    <Linkedin className="h-3.5 w-3.5" />
                    LinkedIn
                  </Label>
                  <Input
                    placeholder="profil-name"
                    value={linkedin}
                    onChange={(e) => setLinkedin(e.target.value)}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>Xing</Label>
                  <Input
                    placeholder="Profil_Name"
                    value={xing}
                    onChange={(e) => setXing(e.target.value)}
                  />
                </div>
              </div>

              {/* Notes */}
              <div className="space-y-1.5">
                <Label>Notizen</Label>
                <Textarea
                  placeholder="Interne Notizen zum Kontakt..."
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  rows={3}
                />
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSubmit} disabled={!firstName.trim() || !lastName.trim()}>
            <Plus className="mr-1.5 h-4 w-4" />
            {isEdit ? 'Speichern' : 'Kontakt erstellen'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
