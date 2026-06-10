/**
 * Tenant-scope sections of the dokumente settings panel ("Für alle") —
 * editable only by a Modul-Leiter/admin (ModuleSettingsShell gates them).
 * Mock-first: backed by stores/dokumenteSettings until the document-settings
 * backend is wired (see .planning/backend-gaps.md).
 */
import { useTranslation } from 'react-i18next'
import { HardDrive, Lock as LockIcon, Users } from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import {
  useDokumenteSettingsStore,
  FILE_TYPE_GROUP_EXTENSIONS,
  STORAGE_QUOTA_BY_TIER_GB,
  type DokumenteFileTypeGroup,
  type DokumenteShareScope,
} from '@/stores/dokumenteSettings'

const segBtn = (active: boolean) =>
  `flex flex-1 items-center justify-center gap-2 rounded-lg border py-2 text-sm transition-colors ${
    active
      ? 'border-primary bg-primary/5 font-medium text-primary'
      : 'border-border text-foreground hover:bg-secondary'
  }`

/** Speicher & Aufbewahrung — quota display per tier + trash retention. */
export function DokumenteStorageSettings() {
  const { t } = useTranslation()
  const trashRetentionDays = useDokumenteSettingsStore((s) => s.trashRetentionDays)
  const setTrashRetentionDays = useDokumenteSettingsStore((s) => s.setTrashRetentionDays)

  return (
    <div className="space-y-5">
      {/* Quota per tier — display only, defined by the plan */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('dokumente.settings.storage.quota')}
        </label>
        <div className="overflow-hidden rounded-lg border border-border">
          {STORAGE_QUOTA_BY_TIER_GB.map((row, i) => (
            <div
              key={row.tier}
              className={`flex items-center justify-between px-3 py-2 text-sm ${
                i > 0 ? 'border-t border-border' : ''
              }`}
            >
              <span className="flex items-center gap-2 text-foreground">
                <HardDrive className="h-4 w-4 text-muted-foreground" />
                {t(`dokumente.settings.storage.tier.${row.tier}`)}
              </span>
              <span className="text-muted-foreground">
                {t('dokumente.settings.storage.quotaValue', { gb: row.quotaGb })}
              </span>
            </div>
          ))}
        </div>
        <p className="text-xs text-muted-foreground">{t('dokumente.settings.storage.quotaHint')}</p>
      </div>

      {/* Trash retention */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground" htmlFor="dokumente-trash-days">
          {t('dokumente.settings.storage.trashDays')}
        </label>
        <input
          id="dokumente-trash-days"
          type="number"
          min={1}
          max={365}
          value={trashRetentionDays}
          onChange={(e) => {
            const v = Number(e.target.value)
            if (Number.isFinite(v)) setTrashRetentionDays(Math.min(365, Math.max(1, Math.round(v))))
          }}
          className="h-9 w-32 rounded-lg border border-border bg-transparent px-2 text-sm text-foreground outline-none focus:border-primary"
        />
        <p className="text-xs text-muted-foreground">{t('dokumente.settings.storage.trashDaysHint')}</p>
      </div>
    </div>
  )
}

/** Erlaubte Dateitypen — toggle extension groups for upload. */
export function DokumenteFileTypeSettings() {
  const { t } = useTranslation()
  const allowed = useDokumenteSettingsStore((s) => s.allowedFileTypeGroups)
  const toggle = useDokumenteSettingsStore((s) => s.toggleFileTypeGroup)

  const groups = Object.keys(FILE_TYPE_GROUP_EXTENSIONS) as DokumenteFileTypeGroup[]

  return (
    <div className="space-y-2">
      {groups.map((group) => {
        const extensions = FILE_TYPE_GROUP_EXTENSIONS[group]
        return (
          <label
            key={group}
            className="flex cursor-pointer items-center justify-between gap-3 rounded-lg border border-border px-3 py-2.5 hover:bg-secondary/40 transition-colors"
          >
            <span className="min-w-0">
              <span className="block text-sm text-foreground">
                {t(`dokumente.settings.fileTypes.group.${group}`)}
              </span>
              {extensions.length > 0 && (
                <span className="block truncate text-xs text-muted-foreground">
                  {extensions.map((e) => `.${e}`).join(', ')}
                </span>
              )}
            </span>
            <Switch checked={allowed.includes(group)} onCheckedChange={() => toggle(group)} />
          </label>
        )
      })}
      <p className="text-xs text-muted-foreground">{t('dokumente.settings.fileTypes.hint')}</p>
    </div>
  )
}

/** Freigabe & Bearbeitung — default share scope + OnlyOffice toggle. */
export function DokumenteSharingSettings() {
  const { t } = useTranslation()
  const defaultShareScope = useDokumenteSettingsStore((s) => s.defaultShareScope)
  const onlyOfficeEnabled = useDokumenteSettingsStore((s) => s.onlyOfficeEnabled)
  const setDefaultShareScope = useDokumenteSettingsStore((s) => s.setDefaultShareScope)
  const setOnlyOfficeEnabled = useDokumenteSettingsStore((s) => s.setOnlyOfficeEnabled)

  // `as const` keeps the labelKeys as literal types for the typed t().
  const scopeOptions = [
    { id: 'private', labelKey: 'dokumente.settings.sharing.scopePrivate', icon: LockIcon },
    { id: 'team', labelKey: 'dokumente.settings.sharing.scopeTeam', icon: Users },
  ] as const satisfies readonly { id: DokumenteShareScope; labelKey: string; icon: typeof Users }[]

  return (
    <div className="space-y-5">
      {/* Default share scope for new uploads */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('dokumente.settings.sharing.defaultScope')}
        </label>
        <div className="flex gap-2">
          {scopeOptions.map((opt) => {
            const Icon = opt.icon
            return (
              <button
                key={opt.id}
                onClick={() => setDefaultShareScope(opt.id)}
                className={segBtn(defaultShareScope === opt.id)}
              >
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
        <p className="text-xs text-muted-foreground">{t('dokumente.settings.sharing.defaultScopeHint')}</p>
      </div>

      {/* OnlyOffice editor */}
      <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2.5">
        <span>
          <span className="block text-sm text-foreground">{t('dokumente.settings.sharing.onlyOffice')}</span>
          <span className="block text-xs text-muted-foreground">
            {t('dokumente.settings.sharing.onlyOfficeHint')}
          </span>
        </span>
        <Switch checked={onlyOfficeEnabled} onCheckedChange={setOnlyOfficeEnabled} />
      </div>
    </div>
  )
}
