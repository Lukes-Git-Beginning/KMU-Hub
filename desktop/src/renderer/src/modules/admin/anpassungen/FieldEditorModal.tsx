/**
 * FieldEditorModal — Create / Edit eines Custom Field.
 *
 * Folgt dem DetailModal-Muster (zentriertes Modal, Gradient-Stripe).
 *
 * Progressive Disclosure (KONZEPT §0.2 — Kern-Entscheid):
 *   Basis (immer sichtbar): Label, Typ, Pflichtfeld-Toggle
 *   Fortgeschritten (aufklappbar, „Weitere Optionen"):
 *     - Optionen-Liste bei select/multi_select
 *     - Validierung (min/max/pattern)
 *     - Standardwert
 *     - Sichtbarkeit (visible)
 *
 * Soft-Delete: Lösch-Button ruft onDeleteRequest auf, nicht direkt die API —
 * der Aufrufer zeigt den Konsequenz-Dialog (inUse-Check liegt im Parent).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ChevronDown,
  ChevronRight,
  Trash2,
  Type,
  Hash,
  Calendar,
  ToggleLeft,
  List,
  Link2,
  Mail,
  Phone,
  X,
} from 'lucide-react'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { cn } from '@/lib'
import type {
  CustomFieldDefinition,
  CustomFieldEntity,
  CustomFieldType,
  CreateCustomFieldInput,
  UpdateCustomFieldInput,
  CustomFieldValidation,
} from '@/mocks/data/custom-fields'

// ── Typ-Optionen ──────────────────────────────────────────────────────────────

interface TypeOption {
  value: CustomFieldType
  icon: typeof Type
  labelKey: string
}

const TYPE_OPTIONS: TypeOption[] = [
  { value: 'text', icon: Type, labelKey: 'customization.fields.type.text' },
  { value: 'number', icon: Hash, labelKey: 'customization.fields.type.number' },
  { value: 'date', icon: Calendar, labelKey: 'customization.fields.type.date' },
  { value: 'boolean', icon: ToggleLeft, labelKey: 'customization.fields.type.boolean' },
  { value: 'select', icon: ChevronDown, labelKey: 'customization.fields.type.select' },
  { value: 'multi_select', icon: List, labelKey: 'customization.fields.type.multi_select' },
  { value: 'url', icon: Link2, labelKey: 'customization.fields.type.url' },
  { value: 'email', icon: Mail, labelKey: 'customization.fields.type.email' },
  { value: 'phone', icon: Phone, labelKey: 'customization.fields.type.phone' },
]

const OPTION_TYPES: CustomFieldType[] = ['select', 'multi_select']

// ── Props ─────────────────────────────────────────────────────────────────────

/** Eine bindbare Werteliste inkl. aufgelöster Optionen (für die Vorschau im Dialog). */
export interface ValueSetChoice {
  id: string
  name: string
  options: { id: string; label: string; color?: string }[]
}

/** Wertelisten, an die ein Auswahl-Feld gebunden werden kann. Liefert der Aufrufer
 *  (der Modul-Editor kennt die Listen des Moduls); ohne sie bleibt die Quellen-Wahl
 *  ausgeblendet und das Feld verhält sich wie vorher (nur eigene Optionen). */
type ValueSetProp = { valueSetChoices?: ValueSetChoice[] }

type FieldEditorModalProps = ValueSetProp &
  (
    | {
        open: boolean
        mode: 'create'
        entity: CustomFieldEntity
        onClose: () => void
        onCreate: (input: CreateCustomFieldInput) => void
        field?: undefined
        onUpdate?: undefined
        onDeleteRequest?: undefined
      }
    | {
        open: boolean
        mode: 'edit'
        entity: CustomFieldEntity
        field: CustomFieldDefinition
        onClose: () => void
        onCreate?: undefined
        onUpdate: (input: UpdateCustomFieldInput) => void
        onDeleteRequest: (field: CustomFieldDefinition) => void
      }
  )

// ── Komponente ────────────────────────────────────────────────────────────────

export function FieldEditorModal(props: FieldEditorModalProps) {
  const { open, mode, entity, onClose } = props
  const { t } = useTranslation()

  // ── Formular-Zustand ──────────────────────────────────────────────────────

  const [label, setLabel] = useState(props.field?.label ?? '')
  const [type, setType] = useState<CustomFieldType>(props.field?.type ?? 'text')
  const [required, setRequired] = useState(props.field?.required ?? false)

  // Fortgeschritten — aufgeklappt/zugeklappt. Bei Auswahl-Feldern sofort offen:
  // ohne Optionen ist so ein Feld nutzlos, das ist kein „fortgeschritten".
  const [advancedOpen, setAdvancedOpen] = useState(
    OPTION_TYPES.includes(props.field?.type ?? 'text'),
  )

  // Optionen für select/multi_select (kommagetrennt im Input, intern Array)
  const [optionsRaw, setOptionsRaw] = useState(
    props.field?.options?.join(', ') ?? '',
  )

  // Optionen-Quelle: eigene Liste tippen oder eine geteilte Werteliste binden.
  const [optionSource, setOptionSource] = useState<'own' | 'set'>(
    props.field?.valueSetId ? 'set' : 'own',
  )
  const [valueSetId, setValueSetId] = useState(props.field?.valueSetId ?? '')
  const valueSetChoices = props.valueSetChoices ?? []
  const boundOptions = valueSetChoices.find((s) => s.id === valueSetId)?.options ?? []

  // Validierung
  const [valMin, setValMin] = useState<string>(
    props.field?.validation?.min !== undefined ? String(props.field.validation.min) : '',
  )
  const [valMax, setValMax] = useState<string>(
    props.field?.validation?.max !== undefined ? String(props.field.validation.max) : '',
  )
  const [valPattern, setValPattern] = useState(props.field?.validation?.pattern ?? '')

  // Standardwert + Sichtbarkeit
  const [defaultValue, setDefaultValue] = useState(props.field?.defaultValue ?? '')
  const [visible, setVisible] = useState(props.field?.visible ?? true)

  // ── Type-Wechsel: Optionen-Feld zurücksetzen wenn kein Select ────────────
  const handleTypeChange = (t: CustomFieldType) => {
    setType(t)
    if (!OPTION_TYPES.includes(t)) {
      setOptionsRaw('')
      setValueSetId('')
    } else {
      // Auswahl gewählt → Optionen direkt zeigen, statt sie zu verstecken.
      setAdvancedOpen(true)
    }
  }

  // ── Validation-Objekt bauen ───────────────────────────────────────────────
  function buildValidation(): CustomFieldValidation | undefined {
    const v: CustomFieldValidation = {}
    if (valMin !== '') v.min = Number(valMin)
    if (valMax !== '') v.max = Number(valMax)
    if (valPattern !== '') v.pattern = valPattern
    return Object.keys(v).length > 0 ? v : undefined
  }

  // ── Submit ────────────────────────────────────────────────────────────────
  const canSubmit = label.trim().length > 0

  const handleSubmit = () => {
    if (!canSubmit) return

    // Gebundene Werteliste schlägt eigene Optionen; ungebunden bleibt es beim Text.
    const boundSet = OPTION_TYPES.includes(type) && optionSource === 'set' ? valueSetId : ''
    const options = OPTION_TYPES.includes(type) && !boundSet
      ? optionsRaw.split(',').map((o) => o.trim()).filter(Boolean)
      : []

    if (mode === 'create' && props.onCreate) {
      props.onCreate({
        entity,
        label: label.trim(),
        type,
        required,
        options,
        valueSetId: boundSet || undefined,
        validation: buildValidation(),
        defaultValue: defaultValue.trim() || undefined,
        visible,
      })
    } else if (mode === 'edit' && props.onUpdate) {
      props.onUpdate({
        label: label.trim(),
        type,
        required,
        options,
        valueSetId: boundSet || undefined,
        validation: buildValidation(),
        defaultValue: defaultValue.trim() || undefined,
        visible,
      })
    }
  }

  const title = mode === 'create'
    ? t('customization.fields.newFieldTitle')
    : t('customization.fields.editFieldTitle', { label: props.field?.label ?? '' })

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent
        showCloseButton={false}
        className="flex max-h-[90vh] flex-col gap-0 overflow-hidden rounded-2xl p-0 max-w-lg"
      >
        <DialogTitle className="sr-only">{title}</DialogTitle>

        {/* Gradient-Stripe */}
        <div className="h-0.5 w-full shrink-0 bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)]" />

        {/* Header */}
        <div className="flex items-center gap-3 border-b border-border px-5 py-4">
          <span className="flex-1 text-sm font-semibold text-foreground">{title}</span>
          <button
            onClick={onClose}
            className="rounded-md p-1.5 text-muted-foreground transition-colors hover:text-foreground"
            aria-label={t('common.close')}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Body — scrollbar */}
        <div className="flex-1 overflow-y-auto px-5 py-5 space-y-5">

          {/* ── BASIS-SCHICHT ──────────────────────────────────────────── */}

          {/* Label */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-foreground">
              {t('customization.fields.labelField')}
              <span className="ml-1 text-muted-foreground/60">{t('customization.fields.required')}</span>
            </label>
            <input
              autoFocus
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter' && canSubmit) handleSubmit() }}
              placeholder={t('customization.fields.labelPlaceholder')}
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
            />
          </div>

          {/* Typ-Wahl */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-foreground">
              {t('customization.fields.typeField')}
            </label>
            <div className="grid grid-cols-3 gap-1.5">
              {TYPE_OPTIONS.map((opt) => {
                const Icon = opt.icon
                const active = type === opt.value
                return (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => handleTypeChange(opt.value)}
                    className={cn(
                      'flex items-center gap-1.5 rounded-md border px-2.5 py-2 text-xs font-medium transition-colors',
                      active
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'border-border text-muted-foreground hover:bg-secondary',
                    )}
                  >
                    <Icon className="h-3.5 w-3.5 shrink-0" />
                    {t(opt.labelKey)}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Pflichtfeld */}
          <label className="flex cursor-pointer items-center gap-2.5 text-sm text-foreground">
            <input
              type="checkbox"
              checked={required}
              onChange={(e) => setRequired(e.target.checked)}
              className="rounded accent-primary"
            />
            {t('customization.fields.requiredField')}
          </label>

          {/* ── FORTGESCHRITTENE SCHICHT (aufklappbar) ────────────────── */}

          <div className="rounded-lg border border-border overflow-hidden">
            {/* Aufklapp-Header */}
            <button
              type="button"
              onClick={() => setAdvancedOpen((v) => !v)}
              className="flex w-full items-center justify-between px-4 py-3 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors"
              aria-expanded={advancedOpen}
            >
              {t('customization.fields.advancedOptions')}
              {advancedOpen
                ? <ChevronDown className="h-3.5 w-3.5" />
                : <ChevronRight className="h-3.5 w-3.5" />}
            </button>

            {/* Aufgeklappter Inhalt */}
            {advancedOpen && (
              <div className="border-t border-border px-4 pb-4 pt-3 space-y-4">

                {/* Optionen (nur bei select / multi_select). Zwei Quellen: eigene
                    Optionen tippen ODER eine geteilte Werteliste binden — Letzteres
                    gibt der Werteliste erst ihren Platz im Modul (Darien 2026-08-04). */}
                {OPTION_TYPES.includes(type) && (
                  <div className="space-y-2">
                    <label className="text-xs font-medium text-foreground">
                      {t('customization.fields.options')}
                    </label>

                    {valueSetChoices.length > 0 && (
                      <div role="radiogroup" aria-label={t('customization.fields.optionSource')} className="flex gap-1">
                        {([
                          { v: 'own' as const, labelKey: 'customization.fields.sourceOwn' },
                          { v: 'set' as const, labelKey: 'customization.fields.sourceValueSet' },
                        ]).map(({ v, labelKey }) => (
                          <button
                            key={v}
                            type="button"
                            role="radio"
                            aria-checked={optionSource === v}
                            onClick={() => setOptionSource(v)}
                            className={cn(
                              'flex-1 rounded-md px-2.5 py-1.5 text-xs font-medium transition-colors',
                              optionSource === v
                                ? 'bg-primary text-primary-foreground'
                                : 'bg-secondary text-muted-foreground hover:text-foreground',
                            )}
                          >
                            {t(labelKey)}
                          </button>
                        ))}
                      </div>
                    )}

                    {optionSource === 'set' && valueSetChoices.length > 0 ? (
                      <>
                        <select
                          value={valueSetId}
                          onChange={(e) => setValueSetId(e.target.value)}
                          aria-label={t('customization.fields.sourceValueSet')}
                          className="h-9 w-full rounded-md border border-border bg-transparent px-2.5 text-sm outline-none focus:border-primary"
                        >
                          <option value="">{t('customization.fields.valueSetNone')}</option>
                          {valueSetChoices.map((s) => (
                            <option key={s.id} value={s.id}>{s.name}</option>
                          ))}
                        </select>
                        {/* Vorschau: was das Feld dann anbietet */}
                        {boundOptions.length > 0 && (
                          <div className="flex flex-wrap gap-1.5 pt-0.5">
                            {boundOptions.map((o) => (
                              <span key={o.id} className="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px]">
                                <span className="h-2 w-2 rounded-full" style={{ backgroundColor: o.color ?? 'transparent' }} aria-hidden="true" />
                                {o.label}
                              </span>
                            ))}
                          </div>
                        )}
                        <p className="text-[11px] leading-relaxed text-muted-foreground">
                          {t('customization.fields.valueSetHint')}
                        </p>
                      </>
                    ) : (
                      <>
                        <input
                          value={optionsRaw}
                          onChange={(e) => setOptionsRaw(e.target.value)}
                          placeholder={t('customization.fields.optionsPlaceholder')}
                          className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
                        />
                        <p className="text-[11px] text-muted-foreground">
                          {t('customization.fields.optionsHint')}
                        </p>
                      </>
                    )}
                  </div>
                )}

                {/* Validierung (min/max nur für text/number, pattern für text) */}
                {(type === 'number' || type === 'text') && (
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-foreground">
                      {t('customization.fields.validation')}
                    </label>
                    <div className="grid grid-cols-2 gap-2">
                      <div>
                        <label className="text-[11px] text-muted-foreground">
                          {t('customization.fields.validationMin')}
                        </label>
                        <input
                          type="number"
                          value={valMin}
                          onChange={(e) => setValMin(e.target.value)}
                          className="mt-0.5 h-8 w-full rounded-md border border-border bg-transparent px-2 text-sm outline-none focus:border-primary"
                        />
                      </div>
                      <div>
                        <label className="text-[11px] text-muted-foreground">
                          {t('customization.fields.validationMax')}
                        </label>
                        <input
                          type="number"
                          value={valMax}
                          onChange={(e) => setValMax(e.target.value)}
                          className="mt-0.5 h-8 w-full rounded-md border border-border bg-transparent px-2 text-sm outline-none focus:border-primary"
                        />
                      </div>
                    </div>
                    {type === 'text' && (
                      <div>
                        <label className="text-[11px] text-muted-foreground">
                          {t('customization.fields.validationPattern')}
                        </label>
                        <input
                          value={valPattern}
                          onChange={(e) => setValPattern(e.target.value)}
                          placeholder="^[A-Z]{2}[0-9]+$"
                          className="mt-0.5 h-8 w-full rounded-md border border-border bg-transparent px-2 font-mono text-xs outline-none focus:border-primary"
                        />
                      </div>
                    )}
                  </div>
                )}

                {/* Standardwert */}
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-foreground">
                    {t('customization.fields.defaultValue')}
                  </label>
                  <input
                    value={defaultValue}
                    onChange={(e) => setDefaultValue(e.target.value)}
                    placeholder={t('customization.fields.defaultValuePlaceholder')}
                    className="h-9 w-full rounded-md border border-border bg-transparent px-3 text-sm outline-none focus:border-primary"
                  />
                </div>

                {/* Sichtbarkeit */}
                <label className="flex cursor-pointer items-center gap-2.5 text-sm text-foreground">
                  <input
                    type="checkbox"
                    checked={visible}
                    onChange={(e) => setVisible(e.target.checked)}
                    className="rounded accent-primary"
                  />
                  {t('customization.fields.visibleField')}
                </label>
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-border px-5 py-4">
          {/* Löschen (nur im Edit-Modus) */}
          {mode === 'edit' && props.field && props.onDeleteRequest && (
            <button
              type="button"
              onClick={() => props.onDeleteRequest!(props.field!)}
              className="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-destructive"
            >
              <Trash2 className="h-3.5 w-3.5" />
              {t('common.delete')}
            </button>
          )}

          <div className={cn('flex items-center gap-2', mode === 'create' && 'ml-auto')}>
            <button
              type="button"
              onClick={onClose}
              className="rounded-md px-4 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              {t('common.cancel')}
            </button>
            <button
              type="button"
              onClick={handleSubmit}
              disabled={!canSubmit}
              className="rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-40"
            >
              {mode === 'create' ? t('customization.fields.create') : t('customization.fields.save')}
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
