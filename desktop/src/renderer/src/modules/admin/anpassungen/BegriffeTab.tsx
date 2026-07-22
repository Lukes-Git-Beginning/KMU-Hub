/**
 * BegriffeTab — Tab „Begriffe" im Anpassungen-Hub (v1.2).
 *
 * Zeigt alle LABEL_WHITELIST-Keys sinnvoll gruppiert (Modulnamen /
 * Objekt-Bezeichnungen / Navigation). Pro Zeile:
 *   - Cosmi-Standard (aus Code-Default, grau)
 *   - Effektiver Wert (vendor/tenant-Override, fett)
 *   - Provenance-Badge: Standard (grau) / Von Zentria (blau) / Angepasst (grün)
 *   - Inline-Bearbeiten-Eingabefeld
 *   - Reset pro Key (zu Schicht darunter)
 *
 * Sprach-Auswahl oben (welche Locale wird bearbeitet; Default = aktive App-Locale).
 * Reset-All-Dialog für alle Tenant-Overrides.
 *
 * Nach Speichern: applyLabelOverlay() → sofort in Sidebar/Nav sichtbar
 * (Live-Preview ohne Reload).
 *
 * Progressive Disclosure: schlicht — kein Modal, kein Aufklapp-Bereich.
 * Inline-Edit direkt in der Zeile (Klick auf Bearbeiten-Button).
 *
 * RBAC: gated via admin:customization:manage (Hub-Level), kein eigenes Gate hier.
 */
import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Info, RotateCcw, Pencil, Check, X as XIcon } from 'lucide-react'
import { getLabelDefault } from '@/i18n/useLabelOverlay'
import { toast } from 'sonner'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useLocale } from '@/hooks/useLocale'
import type { ConfigProvenance, ResolvedLabel } from '@/api/customization-types'
import {
  useLabelOverrides,
  useSetLabelOverrides,
  useResetLabelOverride,
  useResetAllLabelOverrides,
} from '@/api/hooks/useLabelOverrides'

// ── Whitelist-Gruppen (Reihenfolge spiegelt LABEL_WHITELIST) ─────────────────

interface LabelGroup {
  key: 'modules' | 'objects' | 'navigation'
  labelKey: string
  keys: string[]
}

const LABEL_GROUPS: LabelGroup[] = [
  {
    key: 'modules',
    labelKey: 'customization.labels.group.modules',
    keys: [
      'rbac.module.crm',
      'rbac.module.work',
      'rbac.module.helpdesk',
      'rbac.module.finance',
      'rbac.module.wiki',
      'rbac.module.team',
      'rbac.module.berichte',
      'rbac.module.formulare',
      'rbac.module.dialer',
      'rbac.module.schichten',
      'rbac.module.zeiterfassung',
      'rbac.module.vertraege',
      'rbac.module.inventar',
      'rbac.module.einkauf',
    ],
  },
  {
    key: 'objects',
    labelKey: 'customization.labels.group.objects',
    keys: [
      'crm.contacts.title',
      'crm.companies.title',
      'crm.deals.title',
      'work.tasks.title',
      'work.projects.title',
    ],
  },
  {
    key: 'navigation',
    labelKey: 'customization.labels.group.navigation',
    keys: [
      'layout.navItems.contacts',
      'layout.navItems.projects',
      'layout.navItems.tasks',
      'layout.navItems.team',
      'layout.navItems.finance',
      'layout.navItems.helpdesk',
      'layout.navItems.admin',
    ],
  },
]

// ── Unterstützte Sprachen ─────────────────────────────────────────────────────

const SUPPORTED_LOCALES = [
  { code: 'de', label: 'Deutsch' },
  { code: 'en', label: 'English' },
  { code: 'fr', label: 'Français' },
  { code: 'it', label: 'Italiano' },
] as const

type SupportedLocaleCode = (typeof SUPPORTED_LOCALES)[number]['code']

// ── Provenance-Badge ──────────────────────────────────────────────────────────

function ProvenanceBadge({ provenance }: { provenance: ConfigProvenance }) {
  const { t } = useTranslation()

  const cfg = {
    default: {
      labelKey: 'customization.labels.provenance.default',
      className: 'bg-secondary text-muted-foreground',
    },
    vendor: {
      labelKey: 'customization.labels.provenance.vendor',
      className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
    },
    tenant: {
      labelKey: 'customization.labels.provenance.tenant',
      className: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
    },
    draft: {
      labelKey: 'customization.labels.provenance.draft',
      className: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
    },
  }[provenance]

  return (
    <span className={`inline-flex items-center rounded px-1.5 py-0.5 text-[11px] font-medium ${cfg.className}`}>
      {t(cfg.labelKey)}
    </span>
  )
}

// ── Einzelne Label-Zeile ──────────────────────────────────────────────────────

interface LabelRowProps {
  i18nKey: string
  resolved: ResolvedLabel | undefined
  defaultValue: string
  editingKey: string | null
  onEdit: (key: string) => void
  onSave: (key: string, value: string) => void
  onCancelEdit: () => void
  onReset: (key: string) => void
}

function LabelRow({
  i18nKey,
  resolved,
  defaultValue,
  editingKey,
  onEdit,
  onSave,
  onCancelEdit,
  onReset,
}: LabelRowProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState('')

  const provenance = resolved?.provenance ?? 'default'
  const effectiveValue = provenance !== 'default' && resolved?.value ? resolved.value : defaultValue
  const isEditing = editingKey === i18nKey
  const isTenant = provenance === 'tenant'

  const startEdit = () => {
    setDraft(isTenant && resolved?.value ? resolved.value : '')
    onEdit(i18nKey)
  }

  const handleSave = () => {
    const trimmed = draft.trim()
    if (trimmed.length > 0) onSave(i18nKey, trimmed)
    else onCancelEdit()
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSave()
    if (e.key === 'Escape') onCancelEdit()
  }

  return (
    <div
      className={[
        'flex items-start gap-3 rounded-lg border px-4 py-3 transition-colors',
        isEditing
          ? 'border-primary/40 bg-primary/5'
          : 'border-border bg-card hover:border-border/60 hover:bg-accent/30',
      ].join(' ')}
    >
      {/* Key-Anzeige + Default-Wert */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          {/* Effektiver Wert */}
          <span className={[
            'text-sm font-medium',
            provenance === 'default' ? 'text-muted-foreground' : 'text-foreground',
          ].join(' ')}>
            {effectiveValue || <span className="italic text-muted-foreground/50">{t('customization.labels.defaultValue')}</span>}
          </span>

          <ProvenanceBadge provenance={provenance} />

          {/* Vendor-Tooltip-Hinweis */}
          {provenance === 'vendor' && (
            <span title={t('customization.labels.vendorTooltip')} className="cursor-help">
              <Info className="h-3 w-3 text-blue-500/70" />
            </span>
          )}
        </div>

        {/* Cosmi-Standard (nur wenn Override aktiv) */}
        {provenance !== 'default' && defaultValue && (
          <p className="mt-0.5 text-[11px] text-muted-foreground/60">
            {t('customization.labels.defaultValue')}: {defaultValue}
          </p>
        )}

        {/* i18n-Key (dezent, für Tech-User — aria-hidden um QA-innerText nicht zu stören) */}
        <p className="mt-0.5 font-mono text-[10px] text-muted-foreground/40" aria-hidden="true">{i18nKey}</p>

        {/* Inline-Edit-Eingabe */}
        {isEditing && (
          <div className="mt-2.5 flex items-center gap-2">
            <input
              autoFocus
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={t('customization.labels.editPlaceholder')}
              className="h-8 flex-1 rounded-md border border-border bg-background px-3 text-sm outline-none focus:border-primary"
            />
            <button
              type="button"
              onClick={handleSave}
              disabled={draft.trim().length === 0}
              className="flex h-8 items-center gap-1.5 rounded-md bg-primary px-3 text-xs font-medium text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-40"
            >
              <Check className="h-3 w-3" />
              {t('customization.labels.saveKey')}
            </button>
            <button
              type="button"
              onClick={onCancelEdit}
              className="flex h-8 items-center rounded-md px-2 text-xs text-muted-foreground transition-colors hover:text-foreground"
            >
              <XIcon className="h-3.5 w-3.5" />
            </button>
          </div>
        )}
      </div>

      {/* Aktions-Buttons (nur wenn nicht gerade editiert) */}
      {!isEditing && (
        <div className="shrink-0 flex items-center gap-1 mt-0.5">
          <button
            type="button"
            onClick={startEdit}
            data-qa="label-edit-btn"
            aria-label={`${i18nKey} bearbeiten`}
            className="rounded-md p-1.5 text-muted-foreground transition-colors hover:text-foreground"
          >
            <Pencil className="h-3.5 w-3.5" />
          </button>

          {isTenant && (
            <button
              type="button"
              onClick={() => onReset(i18nKey)}
              data-qa="label-reset-btn"
              aria-label={t('customization.labels.resetKeyTitle')}
              title={t('customization.labels.resetKeyHint')}
              className="rounded-md p-1.5 text-muted-foreground transition-colors hover:text-foreground"
            >
              <RotateCcw className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      )}
    </div>
  )
}

// ── Hauptkomponente ───────────────────────────────────────────────────────────

export default function BegriffeTab() {
  const { t } = useTranslation()
  const { locale: appLocale } = useLocale()

  const [selectedLocale, setSelectedLocale] = useState<SupportedLocaleCode>(
    (appLocale as SupportedLocaleCode) ?? 'de',
  )
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [resetAllOpen, setResetAllOpen] = useState(false)

  const { data: resolvedData, isLoading } = useLabelOverrides(selectedLocale)

  const setOverrides = useSetLabelOverrides()
  const resetSingle = useResetLabelOverride()
  const resetAll = useResetAllLabelOverrides()

  const resolvedLabels = resolvedData?.labels ?? {}

  /**
   * Cosmi-Standard = echter Code-Default, aus dem Snapshot der VOR dem
   * addResourceBundle-Merge eingefroren wurde (getLabelDefault). getResourceBundle
   * wäre hier kontaminiert, da der Bootstrap-Merge das Bundle in-place überschreibt.
   */
  const getStaticDefault = useCallback(
    (key: string): string => getLabelDefault(selectedLocale, key),
    [selectedLocale],
  )

  // Zählt nur Tenant-Overrides (eigene Änderungen, nicht vendor)
  const tenantCount = Object.values(resolvedLabels).filter(
    (l) => l.provenance === 'tenant',
  ).length

  const handleEdit = useCallback((key: string) => {
    setEditingKey(key)
  }, [])

  const handleCancelEdit = useCallback(() => {
    setEditingKey(null)
  }, [])

  const handleSave = useCallback(
    async (key: string, value: string) => {
      try {
        await setOverrides.mutateAsync({
          locale: selectedLocale,
          layer: 'tenant',
          overrides: { [key]: value },
        })
        toast.success(t('customization.labels.saved'))
        setEditingKey(null)
      } catch {
        toast.error(t('customization.labels.saveError'))
      }
    },
    [setOverrides, selectedLocale, t],
  )

  const handleResetKey = useCallback(
    async (key: string) => {
      try {
        await resetSingle.mutateAsync({ locale: selectedLocale, key })
        toast.success(t('customization.labels.resetSingle'))
      } catch {
        toast.error(t('customization.labels.saveError'))
      }
    },
    [resetSingle, selectedLocale, t],
  )

  const handleResetAll = useCallback(async () => {
    try {
      await resetAll.mutateAsync(selectedLocale)
      toast.success(t('customization.labels.resetAllDone'))
      setResetAllOpen(false)
    } catch {
      toast.error(t('customization.labels.saveError'))
    }
  }, [resetAll, selectedLocale, t])

  return (
    <div className="px-6 py-6 space-y-6">

      {/* Header-Bereich: Titel + Sprach-Auswahl + Reset-All */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1">
          <p className="text-sm font-semibold text-foreground">
            {t('customization.labels.title')}
          </p>
          <p className="text-xs text-muted-foreground">
            {t('customization.labels.subtitle')}
          </p>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          {/* Sprach-Auswahl */}
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground whitespace-nowrap">
              {t('customization.labels.localeLabel')}
            </span>
            <div
              role="radiogroup"
              aria-label={t('customization.labels.localeLabel')}
              className="flex gap-0.5 rounded-lg border border-border bg-secondary p-0.5"
            >
              {SUPPORTED_LOCALES.map((loc) => (
                <button
                  key={loc.code}
                  role="radio"
                  aria-checked={selectedLocale === loc.code}
                  onClick={() => { setSelectedLocale(loc.code); setEditingKey(null) }}
                  className={[
                    'rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
                    selectedLocale === loc.code
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground',
                  ].join(' ')}
                >
                  {loc.label}
                </button>
              ))}
            </div>
          </div>

          {/* Reset-All-Button (nur wenn Tenant-Overrides existieren) */}
          {tenantCount > 0 && (
            <button
              type="button"
              onClick={() => setResetAllOpen(true)}
              className="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-destructive"
            >
              <RotateCcw className="h-3.5 w-3.5" />
              {t('customization.labels.resetAll')}
            </button>
          )}
        </div>
      </div>

      {/* Live-Preview-Banner */}
      <div className="flex items-start gap-2 rounded-lg border border-primary/20 bg-primary/5 px-4 py-2.5">
        <Info className="mt-0.5 h-3.5 w-3.5 shrink-0 text-primary/70" />
        <p className="text-xs text-muted-foreground">
          {t('customization.labels.livePreviewBanner')}
        </p>
      </div>

      {/* Zähler eigener Begriffe */}
      {tenantCount > 0 && (
        <p className="text-xs text-muted-foreground">
          {t('customization.labels.overrideCount', { count: tenantCount })}
        </p>
      )}

      {/* Gruppen-Liste */}
      {isLoading ? (
        <div className="space-y-6">
          {[1, 2, 3].map((i) => (
            <div key={i} className="space-y-2">
              <div className="h-4 w-32 animate-pulse rounded bg-secondary" />
              {[1, 2, 3].map((j) => (
                <div key={j} className="h-16 animate-pulse rounded-lg bg-secondary" />
              ))}
            </div>
          ))}
        </div>
      ) : (
        <div className="space-y-6">
          {LABEL_GROUPS.map((group) => (
            <div key={group.key} className="space-y-2">
              {/* Gruppen-Überschrift */}
              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                {t(group.labelKey)}
              </p>

              {/* Zeilen */}
              <div className="space-y-1.5">
                {group.keys.map((i18nKey) => (
                  <LabelRow
                    key={i18nKey}
                    i18nKey={i18nKey}
                    resolved={resolvedLabels[i18nKey]}
                    defaultValue={getStaticDefault(i18nKey)}
                    editingKey={editingKey}
                    onEdit={handleEdit}
                    onSave={handleSave}
                    onCancelEdit={handleCancelEdit}
                    onReset={handleResetKey}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Reset-All-Bestätigungs-Dialog */}
      <AlertDialog open={resetAllOpen} onOpenChange={setResetAllOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t('customization.labels.resetAllTitle')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t('customization.labels.resetAllBody')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setResetAllOpen(false)}>
              {t('common.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => void handleResetAll()}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t('customization.labels.resetAllConfirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
