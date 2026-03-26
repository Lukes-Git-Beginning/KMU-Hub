import { useState } from 'react'
import { Camera, Save, X, Mail } from 'lucide-react'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { LazyRichTextEditor as RichTextEditor } from '@/components/shared/RichTextEditor'
import { toast } from 'sonner'

export default function ProfilTab() {
  const profile = useSettingsStore((s) => s.profile)
  const updateProfile = useSettingsStore((s) => s.updateProfile)
  const mailSignature = useSettingsStore((s) => s.mail.signature)
  const updateMail = useSettingsStore((s) => s.updateMail)
  const user = useAuthStore((s) => s.user)

  const [form, setForm] = useState({ ...profile })
  const [hasChanges, setHasChanges] = useState(false)
  const [signatureDraft, setSignatureDraft] = useState(mailSignature)
  const [signatureChanged, setSignatureChanged] = useState(false)

  const initials = `${form.firstName.charAt(0)}${form.lastName.charAt(0)}`.toUpperCase()
  const role = user?.roles?.includes('admin')
    ? 'Administrator'
    : user?.roles?.includes('manager')
      ? 'Projektleiter'
      : 'Mitarbeiter'

  const handleChange = (field: string, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
    setHasChanges(true)
  }

  const handleSave = () => {
    updateProfile(form)
    setHasChanges(false)
    toast.success('Profil gespeichert')
  }

  const handleCancel = () => {
    setForm({ ...profile })
    setHasChanges(false)
  }

  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      {/* Profile Header Card */}
      <div className="rounded-xl border border-border bg-gradient-to-br from-primary/10 via-card to-card p-6">
        <div className="flex items-start gap-5">
          <div className="relative">
            <Avatar className="h-20 w-20 ring-4 ring-primary/20">
              <AvatarFallback className="text-2xl font-bold bg-primary/10 text-primary">
                {initials}
              </AvatarFallback>
            </Avatar>
            <button className="absolute -bottom-1 -right-1 flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-md hover:bg-primary/90 transition-colors">
              <Camera className="h-4 w-4" />
            </button>
          </div>
          <div className="flex-1 space-y-1">
            <h2 className="text-xl font-bold text-foreground">
              {form.firstName} {form.lastName}
            </h2>
            <p className="text-sm text-muted-foreground">{form.position}</p>
            <div className="flex items-center gap-2 mt-2">
              <Badge variant="outline" className="border-primary/30 text-primary">
                {role}
              </Badge>
              <Badge variant="outline" className="border-emerald-500/30 text-emerald-600 dark:text-emerald-400">
                Online
              </Badge>
            </div>
          </div>
        </div>
      </div>

      {/* Form */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-5">
        <h3 className="font-semibold text-foreground">Persönliche Informationen</h3>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Vorname</label>
            <Input
              value={form.firstName}
              onChange={(e) => handleChange('firstName', e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Nachname</label>
            <Input
              value={form.lastName}
              onChange={(e) => handleChange('lastName', e.target.value)}
            />
          </div>
        </div>

        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">E-Mail</label>
          <Input
            type="email"
            value={form.email}
            onChange={(e) => handleChange('email', e.target.value)}
          />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Telefon</label>
            <Input
              value={form.phone}
              onChange={(e) => handleChange('phone', e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">Position</label>
            <Input
              value={form.position}
              onChange={(e) => handleChange('position', e.target.value)}
            />
          </div>
        </div>

        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">Bio</label>
          <textarea
            value={form.bio}
            onChange={(e) => handleChange('bio', e.target.value)}
            rows={3}
            className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
        </div>
      </div>

      {/* Read-Only Info */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-4">
        <h3 className="font-semibold text-foreground">Kontoinformationen</h3>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <p className="text-xs text-muted-foreground">Rolle</p>
            <p className="text-sm font-medium text-foreground">{role}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Abteilung</p>
            <p className="text-sm font-medium text-foreground">Management</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Mitglied seit</p>
            <p className="text-sm font-medium text-foreground">Januar 2024</p>
          </div>
        </div>
      </div>

      {/* E-Mail-Signatur */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-4">
        <div className="flex items-center gap-2">
          <Mail className="h-5 w-5 text-primary" />
          <h3 className="font-semibold text-foreground">E-Mail-Signatur</h3>
        </div>
        <p className="text-sm text-muted-foreground">
          Ihre Signatur wird automatisch an ausgehende E-Mails angehaengt.
        </p>
        <RichTextEditor
          content={signatureDraft}
          onChange={(html) => {
            setSignatureDraft(html)
            setSignatureChanged(html !== mailSignature)
          }}
          placeholder="Signatur eingeben..."
          compact
          showFooter={false}
          minHeight="80px"
          maxHeight="200px"
        />
        {signatureChanged && (
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setSignatureDraft(mailSignature)
                setSignatureChanged(false)
              }}
            >
              Abbrechen
            </Button>
            <Button
              size="sm"
              onClick={() => {
                updateMail({ signature: signatureDraft })
                setSignatureChanged(false)
                toast.success('Signatur gespeichert')
              }}
            >
              <Save className="h-3.5 w-3.5 mr-1.5" />
              Signatur speichern
            </Button>
          </div>
        )}
      </div>

      {/* Save/Cancel */}
      {hasChanges && (
        <div className="flex justify-end gap-3 sticky bottom-0 bg-background/80 backdrop-blur-sm py-4 border-t border-border -mx-6 px-6">
          <Button variant="outline" onClick={handleCancel}>
            <X className="h-4 w-4 mr-2" />
            Abbrechen
          </Button>
          <Button onClick={handleSave}>
            <Save className="h-4 w-4 mr-2" />
            Speichern
          </Button>
        </div>
      )}
    </div>
  )
}
