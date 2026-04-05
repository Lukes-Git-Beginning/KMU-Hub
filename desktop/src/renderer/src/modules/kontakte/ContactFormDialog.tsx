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
import { useTranslation } from 'react-i18next'
import type { Contact } from '@/stores/contacts'

interface ContactFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  contact?: Contact | null
  onSubmit: (data: Omit<Contact, 'id' | 'initials' | 'createdAt' | 'activities'>) => void
}

const categoryKeys = [
  { value: 'employee', labelKey: 'kontakte.category.employee' },
  { value: 'customer', labelKey: 'kontakte.category.customer' },
  { value: 'partner', labelKey: 'kontakte.category.partner' },
]

const statusKeys = [
  { value: 'active', labelKey: 'kontakte.status.active' },
  { value: 'prospect', labelKey: 'kontakte.status.prospect' },
  { value: 'inactive', labelKey: 'kontakte.status.inactive' },
]

const salutationKeys = [
  { value: '', labelKey: 'kontakte.form.noSalutation' },
  { value: 'Herr', labelKey: 'kontakte.form.salutationMr' },
  { value: 'Frau', labelKey: 'kontakte.form.salutationMs' },
]

const academicTitles = [
  { value: '', labelKey: 'kontakte.form.noTitle' },
  { value: 'Dr.', label: 'Dr.' },
  { value: 'Prof.', label: 'Prof.' },
  { value: 'Prof. Dr.', label: 'Prof. Dr.' },
  { value: 'Dipl.-Ing.', label: 'Dipl.-Ing.' },
  { value: 'Mag.', label: 'Mag.' },
  { value: 'lic.', label: 'lic.' },
]

const availableTags = [
  'VIP', 'Entscheider', 'Technik', 'Design', 'Marketing', 'Sales',
  'Finanzen', 'Recht', 'HR', 'Frontend', 'Backend', 'Security',
  'Extern', 'DSGVO', 'Hosting', 'SLA', 'Beratung', 'Bau',
  'Content', 'Projektleitung', 'React', 'CISSP',
]

export function ContactFormDialog({ open, onOpenChange, contact, onSubmit }: ContactFormDialogProps) {
  const { t } = useTranslation()
  const isEdit = !!contact

  const [salutation, setSalutation] = useState<'Herr' | 'Frau' | ''>('')
  const [title, setTitle] = useState('')
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
  const [country, setCountry] = useState('Deutschland')
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
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setSalutation(contact.salutation)
      setTitle(contact.title ?? '')
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
      setTitle('')
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
      setCountry('Deutschland')
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
      title,
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
          <DialogTitle>{isEdit ? t('kontakte.form.editTitle') : t('kontakte.form.createTitle')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Salutation + Title + Name */}
          <div className="grid grid-cols-[100px_110px_1fr_1fr] gap-3">
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.salutation')}</Label>
              <Select value={salutation} onValueChange={(v) => setSalutation(v as typeof salutation)}>
                <SelectTrigger><SelectValue placeholder="—" /></SelectTrigger>
                <SelectContent>
                  {salutationKeys.map((s) => (
                    <SelectItem key={s.value} value={s.value || '_none'}>{t(s.labelKey)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.title')}</Label>
              <Select value={title} onValueChange={setTitle}>
                <SelectTrigger><SelectValue placeholder="—" /></SelectTrigger>
                <SelectContent>
                  {academicTitles.map((at) => (
                    <SelectItem key={at.value} value={at.value || '_none'}>{at.labelKey ? t(at.labelKey) : at.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.firstName')} *</Label>
              <Input
                placeholder={t('kontakte.form.firstNamePlaceholder')}
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                autoFocus
              />
            </div>
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.lastName')} *</Label>
              <Input
                placeholder={t('kontakte.form.lastNamePlaceholder')}
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
              />
            </div>
          </div>

          {/* Email + Phone */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.email')}</Label>
              <Input
                type="email"
                placeholder={t('kontakte.form.emailPlaceholder')}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.phone')}</Label>
              <Input
                placeholder="+49 30 123 45 67"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
              />
            </div>
          </div>

          {/* Mobile + Company */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.mobile')}</Label>
              <Input
                placeholder="+49 170 123 4567"
                value={mobile}
                onChange={(e) => setMobile(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.company')}</Label>
              <Input
                placeholder={t('kontakte.form.companyPlaceholder')}
                value={company}
                onChange={(e) => setCompany(e.target.value)}
              />
            </div>
          </div>

          {/* Job Title + Department */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.jobTitle')}</Label>
              <Input
                placeholder={t('kontakte.form.jobTitlePlaceholder')}
                value={jobTitle}
                onChange={(e) => setJobTitle(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.department')}</Label>
              <Input
                placeholder={t('kontakte.form.departmentPlaceholder')}
                value={department}
                onChange={(e) => setDepartment(e.target.value)}
              />
            </div>
          </div>

          {/* Category + Status */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('kontakte.form.category')}</Label>
              <Select value={category} onValueChange={(v) => setCategory(v as typeof category)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {categoryKeys.map((c) => (
                    <SelectItem key={c.value} value={c.value}>{t(c.labelKey)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('common.status')}</Label>
              <Select value={status} onValueChange={(v) => setStatus(v as typeof status)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {statusKeys.map((s) => (
                    <SelectItem key={s.value} value={s.value}>{t(s.labelKey)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Tags */}
          <div className="space-y-1.5">
            <Label>{t('kontakte.form.tags')}</Label>
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
              placeholder={t('kontakte.form.tagSearchPlaceholder')}
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
            {showExtras ? t('kontakte.form.lessOptions') : t('kontakte.form.moreOptions')}
          </button>

          {showExtras && (
            <div className="space-y-4 rounded-lg border border-border p-3">
              {/* Address */}
              <div className="space-y-3">
                <Label className="text-xs font-medium uppercase text-muted-foreground">{t('kontakte.form.address')}</Label>
                <Input
                  placeholder={t('kontakte.form.streetPlaceholder')}
                  value={street}
                  onChange={(e) => setStreet(e.target.value)}
                />
                <div className="grid grid-cols-[100px_1fr_1fr] gap-3">
                  <Input
                    placeholder={t('kontakte.form.zipPlaceholder')}
                    value={zip}
                    onChange={(e) => setZip(e.target.value)}
                  />
                  <Input
                    placeholder={t('kontakte.form.cityPlaceholder')}
                    value={city}
                    onChange={(e) => setCity(e.target.value)}
                  />
                  <Input
                    placeholder={t('kontakte.form.countryPlaceholder')}
                    value={country}
                    onChange={(e) => setCountry(e.target.value)}
                  />
                </div>
              </div>

              {/* Website */}
              <div className="space-y-1.5">
                <Label className="flex items-center gap-1">
                  <Globe className="h-3.5 w-3.5" />
                  {t('kontakte.form.website')}
                </Label>
                <Input
                  placeholder={t('kontakte.form.websitePlaceholder')}
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
                    placeholder={t('kontakte.form.linkedinPlaceholder')}
                    value={linkedin}
                    onChange={(e) => setLinkedin(e.target.value)}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>{t('kontakte.form.xing')}</Label>
                  <Input
                    placeholder={t('kontakte.form.xingPlaceholder')}
                    value={xing}
                    onChange={(e) => setXing(e.target.value)}
                  />
                </div>
              </div>

              {/* Notes */}
              <div className="space-y-1.5">
                <Label>{t('kontakte.form.notes')}</Label>
                <Textarea
                  placeholder={t('kontakte.form.notesPlaceholder')}
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
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSubmit} disabled={!firstName.trim() || !lastName.trim()}>
            <Plus className="mr-1.5 h-4 w-4" />
            {isEdit ? t('common.save') : t('kontakte.form.createContact')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
