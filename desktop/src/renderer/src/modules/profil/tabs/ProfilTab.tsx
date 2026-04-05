import { useState } from 'react'
import { useTranslation } from 'react-i18next'
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
  const { t } = useTranslation()
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
    ? t('profil.role.admin')
    : user?.roles?.includes('manager')
      ? t('profil.role.manager')
      : t('profil.role.employee')

  const handleChange = (field: string, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
    setHasChanges(true)
  }

  const handleSave = () => {
    updateProfile(form)
    setHasChanges(false)
    toast.success(t('profil.info.profileSaved'))
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
              <Badge variant="outline" className="border-success/30 text-success">
                {t('profil.status.online')}
              </Badge>
            </div>
          </div>
        </div>
      </div>

      {/* Form */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-5">
        <h3 className="font-semibold text-foreground">{t('profil.info.personalInfo')}</h3>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('profil.field.firstName')}</label>
            <Input
              value={form.firstName}
              onChange={(e) => handleChange('firstName', e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('profil.field.lastName')}</label>
            <Input
              value={form.lastName}
              onChange={(e) => handleChange('lastName', e.target.value)}
            />
          </div>
        </div>

        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">{t('profil.field.email')}</label>
          <Input
            type="email"
            value={form.email}
            onChange={(e) => handleChange('email', e.target.value)}
          />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('profil.field.phone')}</label>
            <Input
              value={form.phone}
              onChange={(e) => handleChange('phone', e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t('profil.field.position')}</label>
            <Input
              value={form.position}
              onChange={(e) => handleChange('position', e.target.value)}
            />
          </div>
        </div>

        <div className="space-y-1.5">
          <label className="text-sm font-medium text-foreground">{t('profil.field.bio')}</label>
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
        <h3 className="font-semibold text-foreground">{t('profil.info.accountInfo')}</h3>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <p className="text-xs text-muted-foreground">{t('profil.info.role')}</p>
            <p className="text-sm font-medium text-foreground">{role}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">{t('profil.info.department')}</p>
            <p className="text-sm font-medium text-foreground">Management</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">{t('profil.info.memberSince')}</p>
            <p className="text-sm font-medium text-foreground">Januar 2024</p>
          </div>
        </div>
      </div>

      {/* E-Mail-Signatur */}
      <div className="rounded-xl border border-border bg-card p-6 space-y-4">
        <div className="flex items-center gap-2">
          <Mail className="h-5 w-5 text-primary" />
          <h3 className="font-semibold text-foreground">{t('profil.signature.title')}</h3>
        </div>
        <p className="text-sm text-muted-foreground">
          {t('profil.signature.description')}
        </p>
        <RichTextEditor
          content={signatureDraft}
          onChange={(html) => {
            setSignatureDraft(html)
            setSignatureChanged(html !== mailSignature)
          }}
          placeholder={t('profil.signature.placeholder')}
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
              {t('common.cancel')}
            </Button>
            <Button
              size="sm"
              onClick={() => {
                updateMail({ signature: signatureDraft })
                setSignatureChanged(false)
                toast.success(t('profil.signature.saved'))
              }}
            >
              <Save className="h-3.5 w-3.5 mr-1.5" />
              {t('profil.signature.save')}
            </Button>
          </div>
        )}
      </div>

      {/* Save/Cancel */}
      {hasChanges && (
        <div className="flex justify-end gap-3 sticky bottom-0 bg-background/80 backdrop-blur-sm py-4 border-t border-border -mx-6 px-6">
          <Button variant="outline" onClick={handleCancel}>
            <X className="h-4 w-4 mr-2" />
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSave}>
            <Save className="h-4 w-4 mr-2" />
            {t('common.save')}
          </Button>
        </div>
      )}
    </div>
  )
}
