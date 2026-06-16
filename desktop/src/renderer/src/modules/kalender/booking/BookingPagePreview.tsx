import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { ExternalLink, Copy, Calendar, Clock } from 'lucide-react'
import { toast } from 'sonner'
import type { BookingPage, BookingPageService, AvailabilitySlot } from '@/api/booking-client'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface BookingPagePreviewProps {
  page: BookingPage
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const DAYS_SHORT_DE = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']
const MONTHS_DE = [
  'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
]

function timeToMinutes(time: string): number {
  const [h, m] = time.split(':').map(Number)
  return h * 60 + m
}

function minutesToTime(minutes: number): string {
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}

function isInBreak(minuteOfDay: number, breaks: AvailabilitySlot[]): boolean {
  for (const br of breaks) {
    const bStart = timeToMinutes(br.start)
    const bEnd = timeToMinutes(br.end)
    if (minuteOfDay >= bStart && minuteOfDay < bEnd) return true
  }
  return false
}

function expandSlots(
  windowSlots: AvailabilitySlot[],
  slotDuration: number,
  breaks: AvailabilitySlot[],
): string[] {
  const result: string[] = []
  for (const window of windowSlots) {
    const windowStart = timeToMinutes(window.start)
    const windowEnd = timeToMinutes(window.end)
    let current = windowStart
    while (current + slotDuration <= windowEnd) {
      if (!isInBreak(current, breaks)) {
        result.push(minutesToTime(current))
      }
      current += slotDuration
    }
  }
  return result
}

function computeAvailableDates(rules: BookingPage['availabilityRules']): {
  key: string
  label: string
  dayShort: string
  dayNum: number
  available: boolean
}[] {
  const dates: { key: string; label: string; dayShort: string; dayNum: number; available: boolean }[] = []
  const now = new Date()
  const leadMs = rules.lead_time_hours * 60 * 60 * 1000
  const startFrom = new Date(now.getTime() + leadMs)
  for (let i = 0; i < 14; i++) {
    const d = new Date(startFrom)
    d.setDate(startFrom.getDate() + i)
    const dayOfWeek = d.getDay()
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    dates.push({
      key,
      label: `${d.getDate()}. ${MONTHS_DE[d.getMonth()].slice(0, 3)}`,
      dayShort: DAYS_SHORT_DE[dayOfWeek],
      dayNum: d.getDate(),
      available: rules.weekdays.includes(dayOfWeek),
    })
  }
  return dates
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function BookingPagePreview({ page }: BookingPagePreviewProps) {
  const { t } = useTranslation()
  const [selectedService, setSelectedService] = useState<BookingPageService | null>(null)
  const [selectedDate, setSelectedDate] = useState<string | null>(null)
  const [selectedTime, setSelectedTime] = useState<string | null>(null)
  const [step, setStep] = useState(1)

  const bookingUrl = `https://booking.zentria.tech/${page.slug}`
  const rules = page.availabilityRules

  const availableDates = useMemo(() => computeAvailableDates(rules), [rules])

  const availableSlots = useMemo(() => {
    if (!selectedDate) return []
    return expandSlots(rules.slots_per_weekday, rules.slot_duration_min, rules.breaks ?? [])
  }, [selectedDate, rules])

  function handleCopyLink() {
    navigator.clipboard?.writeText(bookingUrl).then(() => {
      toast.success(t('kalender.booking.pages.preview.linkCopied'))
    })
  }

  function handleBook() {
    toast.info(t('kalender.booking.pages.preview.previewHint'))
  }

  return (
    <div className="space-y-4">
      {/* Info banner */}
      <div className="flex items-center justify-between rounded-lg border border-primary/30 bg-primary-subtle px-4 py-3">
        <div className="flex items-center gap-2">
          <ExternalLink className="h-4 w-4 text-primary" />
          <p className="text-xs font-medium text-primary">
            {t('kalender.booking.pages.preview.previewBanner')}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <a
            href={bookingUrl}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <ExternalLink className="h-3 w-3" />
            {page.slug}
          </a>
          <button
            onClick={handleCopyLink}
            className="flex items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            <Copy className="h-3 w-3" />
            {t('kalender.booking.pages.preview.copyLink')}
          </button>
        </div>
      </div>

      {/* Preview panel */}
      <div className="mx-auto max-w-xl rounded-xl border border-border bg-card shadow-[var(--shadow-large)] overflow-hidden">
        {/* Company header */}
        <div className="border-b border-border bg-primary px-6 py-5 text-center">
          {page.logo_url ? (
            <img src={page.logo_url} alt={page.company_name} className="mx-auto mb-2 h-12 w-auto object-contain" />
          ) : (
            <div className="mx-auto mb-2 inline-flex h-12 w-12 items-center justify-center rounded-full bg-white/20">
              <Calendar className="h-6 w-6 text-white" />
            </div>
          )}
          <h3 className="text-base font-semibold text-white">{page.company_name}</h3>
        </div>

        <div className="p-6 space-y-5">
          {/* Step indicator */}
          <div className="flex items-center gap-2 justify-center">
            {[1, 2, 3, 4].map((s) => (
              <div key={s} className="flex items-center gap-2">
                <div
                  className={`flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-medium transition-colors ${
                    step >= s ? 'bg-primary text-primary-foreground' : 'bg-secondary text-muted-foreground'
                  }`}
                >
                  {s}
                </div>
                {s < 4 && (
                  <div className={`h-0.5 w-6 rounded-full ${step > s ? 'bg-primary' : 'bg-secondary'}`} />
                )}
              </div>
            ))}
          </div>

          {/* Step 1: Service selection */}
          {step === 1 && (
            <div className="space-y-3">
              <h4 className="text-center text-sm font-medium text-foreground">
                {t('kalender.external.selectService')}
              </h4>
              <div className="space-y-2">
                {page.services.map((svc) => (
                  <button
                    key={svc.id}
                    onClick={() => {
                      setSelectedService(svc)
                      setStep(2)
                    }}
                    className={`w-full flex items-center justify-between rounded-lg border px-4 py-3 transition-colors ${
                      selectedService?.id === svc.id
                        ? 'border-primary bg-primary-subtle'
                        : 'border-border hover:border-primary/50 hover:bg-secondary/30'
                    }`}
                  >
                    <div className="text-left">
                      <p className="text-sm font-medium text-foreground">{svc.name}</p>
                      <div className="mt-0.5 flex items-center gap-2">
                        <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
                          <Clock className="h-3 w-3" />
                          {svc.duration_min} Min.
                        </span>
                        {parseFloat(svc.price) > 0 ? (
                          <span className="text-[10px] text-muted-foreground">EUR {svc.price}</span>
                        ) : (
                          <span className="text-[10px] font-medium text-success">
                            {t('kalender.external.free')}
                          </span>
                        )}
                      </div>
                      {svc.description && (
                        <p className="mt-0.5 text-[10px] text-muted-foreground">{svc.description}</p>
                      )}
                    </div>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Step 2: Date selection */}
          {step === 2 && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <button
                  onClick={() => setStep(1)}
                  className="text-xs text-primary hover:underline"
                >
                  {t('common.back')}
                </button>
                <h4 className="text-sm font-medium text-foreground">
                  {t('kalender.external.selectDate')}
                </h4>
                <div className="w-10" />
              </div>
              {selectedService && (
                <p className="text-center text-xs text-muted-foreground">
                  {selectedService.name} ({selectedService.duration_min} Min.)
                </p>
              )}
              <div className="grid grid-cols-7 gap-1.5">
                {availableDates.map((d) => (
                  <button
                    key={d.key}
                    disabled={!d.available}
                    onClick={() => {
                      if (d.available) {
                        setSelectedDate(d.key)
                        setStep(3)
                      }
                    }}
                    className={`flex flex-col items-center rounded-lg py-2 text-xs transition-colors ${
                      !d.available
                        ? 'cursor-not-allowed opacity-40 text-text-disabled'
                        : selectedDate === d.key
                          ? 'bg-primary text-primary-foreground'
                          : 'border border-border text-foreground hover:bg-secondary'
                    }`}
                  >
                    <span className="text-[9px] uppercase">{d.dayShort}</span>
                    <span className="mt-0.5 text-sm font-medium">{d.dayNum}</span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Step 3: Time slot selection */}
          {step === 3 && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <button
                  onClick={() => setStep(2)}
                  className="text-xs text-primary hover:underline"
                >
                  {t('common.back')}
                </button>
                <h4 className="text-sm font-medium text-foreground">
                  {t('kalender.external.selectTime')}
                </h4>
                <div className="w-10" />
              </div>
              {selectedDate && (
                <p className="text-center text-xs text-muted-foreground">
                  {(() => {
                    const d = new Date(selectedDate)
                    return `${DAYS_SHORT_DE[d.getDay()]}, ${d.getDate()}. ${MONTHS_DE[d.getMonth()]}`
                  })()}
                </p>
              )}
              {availableSlots.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">
                  {t('kalender.booking.pages.preview.noSlotsAvailable')}
                </p>
              ) : (
                <div className="grid grid-cols-3 gap-2">
                  {availableSlots.map((slot) => (
                    <button
                      key={slot}
                      onClick={() => {
                        setSelectedTime(slot)
                        setStep(4)
                      }}
                      className={`rounded-lg border py-2.5 text-sm font-medium transition-colors ${
                        selectedTime === slot
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-border text-foreground hover:border-primary hover:bg-primary-subtle'
                      }`}
                    >
                      {slot}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Step 4: Contact form (preview-only) */}
          {step === 4 && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <button
                  onClick={() => setStep(3)}
                  className="text-xs text-primary hover:underline"
                >
                  {t('common.back')}
                </button>
                <h4 className="text-sm font-medium text-foreground">
                  {t('kalender.external.yourData')}
                </h4>
                <div className="w-10" />
              </div>

              {/* Summary */}
              <div className="rounded-lg bg-secondary/50 p-3 space-y-1 text-xs">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Calendar className="h-3 w-3" />
                  <span>{selectedService?.name}</span>
                </div>
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Clock className="h-3 w-3" />
                  <span>
                    {selectedDate &&
                      (() => {
                        const d = new Date(selectedDate)
                        return `${d.getDate()}. ${MONTHS_DE[d.getMonth()]} ${d.getFullYear()}`
                      })()}{' '}
                    {selectedTime}
                  </span>
                </div>
              </div>

              {/* Placeholder fields */}
              <div className="space-y-2 opacity-50">
                <div className="h-9 rounded-lg border border-input-border bg-input-background" />
                <div className="h-9 rounded-lg border border-input-border bg-input-background" />
                <div className="h-9 rounded-lg border border-input-border bg-input-background" />
              </div>

              <button
                onClick={handleBook}
                className="w-full rounded-lg bg-primary py-2.5 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                {t('kalender.booking.pages.preview.bookButton')}
              </button>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-border px-6 py-3 text-center">
          <p className="text-[10px] text-muted-foreground">
            Powered by <span className="font-medium text-foreground">Cosmi</span>
          </p>
        </div>
      </div>
    </div>
  )
}
