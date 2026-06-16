import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2, X } from 'lucide-react'
import { toast } from 'sonner'
import { useCalendars } from '@/api/hooks/useCalendars'
import { useCreateBookingPage, useUpdateBookingPage } from '@/api/hooks/useBookingPages'
import type {
  BookingPage,
  BookingPageService,
  AvailabilityRules,
  AvailabilitySlot,
  CreateBookingPageRequest,
  UpdateBookingPageRequest,
} from '@/api/booking-client'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface BookingPageEditorProps {
  page?: BookingPage
  onSave: (page: BookingPage) => void
  onCancel: () => void
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const SLUG_PATTERN = /^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$/

const WEEKDAY_LABELS: { value: number; label: string }[] = [
  { value: 1, label: 'Mo' },
  { value: 2, label: 'Di' },
  { value: 3, label: 'Mi' },
  { value: 4, label: 'Do' },
  { value: 5, label: 'Fr' },
  { value: 6, label: 'Sa' },
  { value: 0, label: 'So' },
]

function emptyService(): BookingPageService {
  return {
    id: crypto.randomUUID(),
    name: '',
    duration_min: 30,
    price: '0.00',
    description: null,
  }
}

function emptySlot(): AvailabilitySlot {
  return { start: '09:00', end: '17:00' }
}

function emptyBreak(): AvailabilitySlot {
  return { start: '12:00', end: '13:00' }
}

function defaultRules(): AvailabilityRules {
  return {
    weekdays: [1, 2, 3, 4, 5],
    slots_per_weekday: [{ start: '09:00', end: '17:00' }],
    slot_duration_min: 30,
    buffer_min: 0,
    lead_time_hours: 24,
    breaks: [],
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function BookingPageEditor({ page, onSave, onCancel }: BookingPageEditorProps) {
  const { t } = useTranslation()
  const isEdit = !!page

  const { data: calendarsData } = useCalendars()
  const calendars = calendarsData?.calendars ?? []

  const createMutation = useCreateBookingPage()
  const updateMutation = useUpdateBookingPage()

  const isBusy = createMutation.isPending || updateMutation.isPending

  // Form state
  const [calendarId, setCalendarId] = useState(page?.calendar_id ?? calendars[0]?.id ?? '')
  const [slug, setSlug] = useState(page?.slug ?? '')
  const [slugError, setSlugError] = useState<string | null>(null)
  const [companyName, setCompanyName] = useState(page?.company_name ?? '')
  const [logoUrl, setLogoUrl] = useState(page?.logo_url ?? '')
  const [active, setActive] = useState(page?.active ?? true)

  // Services
  const [services, setServices] = useState<BookingPageService[]>(
    page?.services?.length ? page.services : [emptyService()],
  )

  // Availability rules
  const initRules = page?.availabilityRules ?? defaultRules()
  const [weekdays, setWeekdays] = useState<number[]>(initRules.weekdays)
  const [slots, setSlots] = useState<AvailabilitySlot[]>(initRules.slots_per_weekday)
  const [slotDuration, setSlotDuration] = useState(initRules.slot_duration_min)
  const [bufferMin, setBufferMin] = useState(initRules.buffer_min)
  const [leadTimeHours, setLeadTimeHours] = useState(initRules.lead_time_hours)
  const [breaks, setBreaks] = useState<AvailabilitySlot[]>(initRules.breaks ?? [])

  // ---------------------------------------------------------------------------
  // Handlers — Services
  // ---------------------------------------------------------------------------

  function addService() {
    setServices((prev) => [...prev, emptyService()])
  }

  function removeService(index: number) {
    setServices((prev) => prev.filter((_, i) => i !== index))
  }

  function updateService<K extends keyof BookingPageService>(
    index: number,
    key: K,
    value: BookingPageService[K],
  ) {
    setServices((prev) =>
      prev.map((s, i) => (i === index ? { ...s, [key]: value } : s)),
    )
  }

  // ---------------------------------------------------------------------------
  // Handlers — Weekdays
  // ---------------------------------------------------------------------------

  function toggleWeekday(day: number) {
    setWeekdays((prev) =>
      prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day],
    )
  }

  // ---------------------------------------------------------------------------
  // Handlers — Slots
  // ---------------------------------------------------------------------------

  function addSlot() {
    setSlots((prev) => [...prev, emptySlot()])
  }

  function removeSlot(index: number) {
    setSlots((prev) => prev.filter((_, i) => i !== index))
  }

  function updateSlot(index: number, key: 'start' | 'end', value: string) {
    setSlots((prev) =>
      prev.map((s, i) => (i === index ? { ...s, [key]: value } : s)),
    )
  }

  // ---------------------------------------------------------------------------
  // Handlers — Breaks
  // ---------------------------------------------------------------------------

  function addBreak() {
    setBreaks((prev) => [...prev, emptyBreak()])
  }

  function removeBreak(index: number) {
    setBreaks((prev) => prev.filter((_, i) => i !== index))
  }

  function updateBreak(index: number, key: 'start' | 'end', value: string) {
    setBreaks((prev) =>
      prev.map((s, i) => (i === index ? { ...s, [key]: value } : s)),
    )
  }

  // ---------------------------------------------------------------------------
  // Validation & Submit
  // ---------------------------------------------------------------------------

  function validateSlug(value: string): boolean {
    if (!SLUG_PATTERN.test(value)) {
      setSlugError(t('kalender.booking.pages.editor.slugInvalid'))
      return false
    }
    setSlugError(null)
    return true
  }

  async function handleSubmit() {
    if (!validateSlug(slug)) return
    if (services.length === 0) {
      toast.error(t('kalender.booking.pages.editor.serviceMinRequired'))
      return
    }

    const rules: AvailabilityRules = {
      weekdays,
      slots_per_weekday: slots,
      slot_duration_min: slotDuration,
      buffer_min: bufferMin,
      lead_time_hours: leadTimeHours,
      breaks,
    }

    try {
      let saved: BookingPage
      if (isEdit && page) {
        const req: UpdateBookingPageRequest = {
          company_name: companyName,
          logo_url: logoUrl || null,
          services,
          availability_rules: JSON.stringify(rules),
          active,
        }
        const result = await updateMutation.mutateAsync({ id: page.id, req })
        saved = result.page
      } else {
        const req: CreateBookingPageRequest = {
          calendar_id: calendarId || calendars[0]?.id || '',
          slug,
          company_name: companyName,
          logo_url: logoUrl || null,
          services,
          availability_rules: JSON.stringify(rules),
        }
        const result = await createMutation.mutateAsync(req)
        saved = result.page
      }
      toast.success(t('kalender.booking.pages.editor.savedSuccess'))
      onSave(saved)
    } catch (err: unknown) {
      const status = (err as { status?: number })?.status
      if (status === 409) {
        setSlugError(t('kalender.booking.pages.editor.slugTaken'))
      } else {
        toast.error(t('kalender.booking.pages.editor.saveError'))
      }
    }
  }

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="space-y-6 rounded-lg border border-border bg-card p-6">
      {/* Calendar picker */}
      <div>
        <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
          {t('kalender.booking.pages.editor.calendarLabel')}
        </label>
        <select
          value={calendarId}
          onChange={(e) => setCalendarId(e.target.value)}
          disabled={isEdit}
          className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary disabled:opacity-60"
        >
          {calendars.map((cal) => (
            <option key={cal.id} value={cal.id}>
              {cal.name}
            </option>
          ))}
        </select>
      </div>

      {/* Slug */}
      {!isEdit && (
        <div>
          <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
            {t('kalender.booking.pages.editor.slugLabel')}
          </label>
          <input
            type="text"
            value={slug}
            onChange={(e) => {
              setSlug(e.target.value)
              setSlugError(null)
            }}
            onBlur={() => slug && validateSlug(slug)}
            placeholder="mein-unternehmen"
            className={`w-full rounded-lg border bg-input-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary ${slugError ? 'border-error' : 'border-input-border'}`}
          />
          {slugError ? (
            <p className="mt-1 text-xs text-error">{slugError}</p>
          ) : (
            <p className="mt-1 text-xs text-muted-foreground">
              {t('kalender.booking.pages.editor.slugHint')}
            </p>
          )}
        </div>
      )}

      {/* Company name */}
      <div>
        <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
          {t('kalender.booking.pages.editor.companyName')}
        </label>
        <input
          type="text"
          value={companyName}
          onChange={(e) => setCompanyName(e.target.value)}
          className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary"
        />
      </div>

      {/* Logo URL */}
      <div>
        <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
          {t('kalender.booking.pages.editor.logoUrl')}
        </label>
        <input
          type="url"
          value={logoUrl}
          onChange={(e) => setLogoUrl(e.target.value)}
          placeholder="https://beispiel.de/logo.png"
          className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary"
        />
      </div>

      {/* Services */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <h4 className="text-sm font-medium text-foreground">
            {t('kalender.booking.pages.editor.servicesTitle')}
          </h4>
          <button
            type="button"
            onClick={addService}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <Plus className="h-3 w-3" />
            {t('kalender.booking.pages.editor.addService')}
          </button>
        </div>
        <div className="space-y-3">
          {services.map((svc, idx) => (
            <div key={svc.id} className="rounded-lg border border-border bg-secondary/20 p-4">
              <div className="mb-3 flex items-center justify-between">
                <span className="text-xs font-medium text-muted-foreground">#{idx + 1}</span>
                {services.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeService(idx)}
                    className="rounded p-0.5 text-muted-foreground hover:text-error transition-colors"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
              <div className="grid grid-cols-2 gap-3">
                {/* Name */}
                <div className="col-span-2">
                  <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
                    {t('kalender.booking.pages.editor.serviceName')}
                  </label>
                  <input
                    type="text"
                    value={svc.name}
                    onChange={(e) => updateService(idx, 'name', e.target.value)}
                    className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
                  />
                </div>
                {/* Duration */}
                <div>
                  <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
                    {t('kalender.booking.pages.editor.serviceDuration')}
                  </label>
                  <input
                    type="number"
                    min={1}
                    value={svc.duration_min}
                    onChange={(e) => updateService(idx, 'duration_min', parseInt(e.target.value) || 1)}
                    className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
                  />
                </div>
                {/* Price */}
                <div>
                  <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
                    {t('kalender.booking.pages.editor.servicePrice')}
                  </label>
                  <input
                    type="text"
                    value={svc.price}
                    onChange={(e) => updateService(idx, 'price', e.target.value)}
                    placeholder="0.00"
                    className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
                  />
                </div>
                {/* Description */}
                <div className="col-span-2">
                  <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
                    {t('kalender.booking.pages.editor.serviceDescription')}
                  </label>
                  <input
                    type="text"
                    value={svc.description ?? ''}
                    onChange={(e) => updateService(idx, 'description', e.target.value || null)}
                    className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
                  />
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Weekdays */}
      <div>
        <h4 className="mb-3 text-sm font-medium text-foreground">
          {t('kalender.booking.pages.editor.weekdaysTitle')}
        </h4>
        <div className="flex flex-wrap gap-2">
          {WEEKDAY_LABELS.map(({ value, label }) => (
            <button
              key={value}
              type="button"
              onClick={() => toggleWeekday(value)}
              className={`rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors ${
                weekdays.includes(value)
                  ? 'border-primary bg-primary text-primary-foreground'
                  : 'border-border text-muted-foreground hover:border-primary/50'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Available time slots */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <h4 className="text-sm font-medium text-foreground">
            {t('kalender.booking.pages.editor.slotsTitle')}
          </h4>
          <button
            type="button"
            onClick={addSlot}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <Plus className="h-3 w-3" />
            {t('kalender.booking.pages.editor.addSlot')}
          </button>
        </div>
        <div className="space-y-2">
          {slots.map((slot, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <input
                type="time"
                value={slot.start}
                onChange={(e) => updateSlot(idx, 'start', e.target.value)}
                className="flex-1 rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
              />
              <span className="text-xs text-muted-foreground">–</span>
              <input
                type="time"
                value={slot.end}
                onChange={(e) => updateSlot(idx, 'end', e.target.value)}
                className="flex-1 rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
              />
              {slots.length > 1 && (
                <button
                  type="button"
                  onClick={() => removeSlot(idx)}
                  className="rounded p-0.5 text-muted-foreground hover:text-error transition-colors"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Slot settings */}
      <div className="grid grid-cols-3 gap-4">
        <div>
          <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
            {t('kalender.booking.pages.editor.slotDuration')}
          </label>
          <input
            type="number"
            min={5}
            value={slotDuration}
            onChange={(e) => setSlotDuration(parseInt(e.target.value) || 30)}
            className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
          />
        </div>
        <div>
          <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
            {t('kalender.booking.pages.editor.bufferMin')}
          </label>
          <input
            type="number"
            min={0}
            value={bufferMin}
            onChange={(e) => setBufferMin(parseInt(e.target.value) || 0)}
            className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
          />
        </div>
        <div>
          <label className="mb-1 block text-[10px] uppercase tracking-wider text-muted-foreground">
            {t('kalender.booking.pages.editor.leadTimeHours')}
          </label>
          <input
            type="number"
            min={0}
            value={leadTimeHours}
            onChange={(e) => setLeadTimeHours(parseInt(e.target.value) || 0)}
            className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
          />
        </div>
      </div>

      {/* Breaks */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <h4 className="text-sm font-medium text-foreground">
            {t('kalender.booking.pages.editor.breaksTitle')}
          </h4>
          <button
            type="button"
            onClick={addBreak}
            className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <Plus className="h-3 w-3" />
            {t('kalender.booking.pages.editor.addBreak')}
          </button>
        </div>
        <div className="space-y-2">
          {breaks.map((br, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <input
                type="time"
                value={br.start}
                onChange={(e) => updateBreak(idx, 'start', e.target.value)}
                className="flex-1 rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
              />
              <span className="text-xs text-muted-foreground">–</span>
              <input
                type="time"
                value={br.end}
                onChange={(e) => updateBreak(idx, 'end', e.target.value)}
                className="flex-1 rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary"
              />
              <button
                type="button"
                onClick={() => removeBreak(idx)}
                className="rounded p-0.5 text-muted-foreground hover:text-error transition-colors"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Active toggle (edit mode only) */}
      {isEdit && (
        <div className="flex items-center gap-3">
          <button
            type="button"
            role="switch"
            aria-checked={active}
            onClick={() => setActive((v) => !v)}
            className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${active ? 'bg-primary' : 'bg-secondary'}`}
          >
            <span
              className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white shadow transition-transform ${active ? 'translate-x-4' : 'translate-x-0.5'}`}
            />
          </button>
          <span className="text-sm text-foreground">
            {t('kalender.booking.pages.editor.activeToggle')}
          </span>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-end gap-3 border-t border-border pt-4">
        <button
          type="button"
          onClick={onCancel}
          disabled={isBusy}
          className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors disabled:opacity-60"
        >
          {/* reuse existing key */}
          Abbrechen
        </button>
        <button
          type="button"
          onClick={handleSubmit}
          disabled={isBusy}
          className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-60"
        >
          {isBusy
            ? '...'
            : isEdit
              ? t('kalender.booking.pages.editor.saveUpdate')
              : t('kalender.booking.pages.editor.saveCreate')}
        </button>
      </div>
    </div>
  )
}
