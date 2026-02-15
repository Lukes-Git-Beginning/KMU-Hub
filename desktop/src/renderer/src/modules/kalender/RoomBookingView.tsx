import { useState, useMemo } from 'react'
import { ChevronLeft, ChevronRight, Users, Monitor, X, Clock } from 'lucide-react'
import { toast } from 'sonner'

// ============================================================
// Types
// ============================================================

interface Room {
  id: string
  name: string
  capacity: number
  tags: string[]
}

interface Booking {
  id: string
  title: string
  room: string
  date: string
  startTime: string
  endTime: string
  organizer: string
  participants: number
}

// ============================================================
// Constants
// ============================================================

const ROOMS: Room[] = [
  { id: 'r1', name: 'Raum A — Besprechung', capacity: 8, tags: ['Beamer', 'Whiteboard'] },
  { id: 'r2', name: 'Raum B — Klein', capacity: 4, tags: ['Display'] },
  { id: 'r3', name: 'Telefonkabine 1', capacity: 1, tags: [] },
  { id: 'r4', name: 'Telefonkabine 2', capacity: 1, tags: [] },
]

const HOURS = Array.from({ length: 11 }, (_, i) => i + 8) // 08:00 - 18:00

const MOCK_BOOKINGS: Booking[] = [
  { id: 'b1', title: 'Sprint Planning', room: 'Raum A — Besprechung', date: '2026-02-09', startTime: '10:00', endTime: '11:30', organizer: 'Anna Mueller', participants: 6 },
  { id: 'b2', title: 'Lunch & Learn', room: 'Raum A — Besprechung', date: '2026-02-09', startTime: '12:30', endTime: '13:30', organizer: 'Max Berg', participants: 5 },
  { id: 'b3', title: 'Product Roadmap', room: 'Raum B — Klein', date: '2026-02-09', startTime: '16:00', endTime: '17:00', organizer: 'Anna Mueller', participants: 3 },
  { id: 'b4', title: '1:1 Sarah', room: 'Telefonkabine 1', date: '2026-02-09', startTime: '14:00', endTime: '15:00', organizer: 'Darien', participants: 2 },
  { id: 'b5', title: 'Kundentermin Meier AG', room: 'Raum A — Besprechung', date: '2026-02-10', startTime: '10:00', endTime: '11:30', organizer: 'Peter Keller', participants: 4 },
  { id: 'b6', title: 'Design Review', room: 'Raum B — Klein', date: '2026-02-10', startTime: '14:00', endTime: '15:30', organizer: 'Jonas Diaz', participants: 3 },
  { id: 'b7', title: 'HR Gespreach', room: 'Telefonkabine 2', date: '2026-02-10', startTime: '10:00', endTime: '11:00', organizer: 'Lisa Weber', participants: 2 },
  { id: 'b8', title: 'Code Review', room: 'Raum B — Klein', date: '2026-02-11', startTime: '09:00', endTime: '10:00', organizer: 'Max Berg', participants: 3 },
  { id: 'b9', title: 'Sprint Demo', room: 'Raum A — Besprechung', date: '2026-02-13', startTime: '10:00', endTime: '11:00', organizer: 'Anna Mueller', participants: 8 },
  { id: 'b10', title: 'Retro', room: 'Raum B — Klein', date: '2026-02-13', startTime: '15:00', endTime: '16:00', organizer: 'Anna Mueller', participants: 4 },
]

const DAYS_SHORT = ['Mo', 'Di', 'Mi', 'Do', 'Fr']
const MONTHS_DE = [
  'Januar', 'Februar', 'Maerz', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
]

const ROOM_COLORS = ['#1e7e74', '#3d5c7d', '#c4873a', '#7c5a8a']

// ============================================================
// Helpers
// ============================================================

function timeToHourFraction(time: string): number {
  const [h, m] = time.split(':').map(Number)
  return h + m / 60
}

function getWeekDays(date: Date): Date[] {
  const d = new Date(date)
  const day = d.getDay()
  const diff = d.getDate() - day + (day === 0 ? -6 : 1)
  const monday = new Date(d.setDate(diff))
  return Array.from({ length: 5 }, (_, i) => {
    const dd = new Date(monday)
    dd.setDate(monday.getDate() + i)
    return dd
  })
}

function formatDateKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function isToday(d: Date): boolean {
  return formatDateKey(d) === formatDateKey(new Date())
}

// ============================================================
// Quick Book Dialog
// ============================================================

function QuickBookDialog({
  room,
  date,
  startHour,
  onClose,
  onBook,
}: {
  room: Room
  date: string
  startHour: number
  onClose: () => void
  onBook: (booking: Omit<Booking, 'id'>) => void
}) {
  const [title, setTitle] = useState('')
  const [endHour, setEndHour] = useState(startHour + 1)

  const handleBook = () => {
    if (!title.trim()) return
    onBook({
      title: title.trim(),
      room: room.name,
      date,
      startTime: `${String(startHour).padStart(2, '0')}:00`,
      endTime: `${String(endHour).padStart(2, '0')}:00`,
      organizer: 'Du',
      participants: 1,
    })
    onClose()
  }

  return (
    <>
      <div className="fixed inset-0 z-40" onClick={onClose} />
      <div className="fixed inset-0 z-50 flex items-center justify-center">
        <div className="absolute inset-0 bg-black/30" onClick={onClose} />
        <div className="relative w-full max-w-sm rounded-xl border border-border bg-card shadow-[var(--shadow-large)] overflow-hidden">
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <h3 className="text-sm font-medium text-foreground">Raum buchen</h3>
            <button onClick={onClose} className="rounded-md p-1 text-muted-foreground hover:bg-secondary">
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="p-4 space-y-3">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Monitor className="h-3.5 w-3.5" />
              <span className="font-medium text-foreground">{room.name}</span>
              <span>({room.capacity} Pl.)</span>
            </div>

            <input
              autoFocus
              type="text"
              placeholder="Titel des Meetings..."
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') handleBook() }}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder outline-none focus:border-primary"
            />

            <div className="flex items-center gap-2">
              <Clock className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-xs text-foreground">{String(startHour).padStart(2, '0')}:00</span>
              <span className="text-xs text-muted-foreground">bis</span>
              <select
                value={endHour}
                onChange={(e) => setEndHour(Number(e.target.value))}
                className="rounded-lg border border-input-border bg-input-background px-2 py-1 text-xs text-foreground outline-none focus:border-primary"
              >
                {HOURS.filter((h) => h > startHour).map((h) => (
                  <option key={h} value={h}>{String(h).padStart(2, '0')}:00</option>
                ))}
                <option value={19}>19:00</option>
              </select>
            </div>

            <div className="flex gap-2 pt-1">
              <button
                onClick={onClose}
                className="flex-1 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleBook}
                disabled={!title.trim()}
                className="flex-1 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
              >
                Buchen
              </button>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

// ============================================================
// Main Component
// ============================================================

interface RoomBookingViewProps {
  onClose: () => void
}

export function RoomBookingView({ onClose }: RoomBookingViewProps) {
  const [currentDate, setCurrentDate] = useState(new Date(2026, 1, 9))
  const [bookings, setBookings] = useState(MOCK_BOOKINGS)
  const [quickBook, setQuickBook] = useState<{ room: Room; date: string; hour: number } | null>(null)
  const [selectedBooking, setSelectedBooking] = useState<Booking | null>(null)

  const weekDays = useMemo(() => getWeekDays(currentDate), [currentDate])
  const [selectedDay, setSelectedDay] = useState(0) // index into weekDays

  const currentDayKey = formatDateKey(weekDays[selectedDay])
  const dayBookings = useMemo(
    () => bookings.filter((b) => b.date === currentDayKey),
    [bookings, currentDayKey],
  )

  const navigate = (dir: -1 | 1) => {
    const d = new Date(currentDate)
    d.setDate(d.getDate() + dir * 7)
    setCurrentDate(d)
    setSelectedDay(0)
  }

  const handleSlotClick = (room: Room, hour: number) => {
    // Check if slot is free
    const occupied = dayBookings.some(
      (b) => b.room === room.name && timeToHourFraction(b.startTime) <= hour && timeToHourFraction(b.endTime) > hour,
    )
    if (!occupied) {
      setQuickBook({ room, date: currentDayKey, hour })
    }
  }

  const handleBook = (booking: Omit<Booking, 'id'>) => {
    setBookings((prev) => [...prev, { ...booking, id: `b${Date.now()}` }])
    toast.success(`${booking.room} gebucht: ${booking.title}`)
  }

  const handleDeleteBooking = (id: string) => {
    setBookings((prev) => prev.filter((b) => b.id !== id))
    setSelectedBooking(null)
    toast.success('Buchung gelöscht')
  }

  const slotWidth = 80
  const roomLabelWidth = 200

  return (
    <div className="fixed inset-0 z-40 flex flex-col bg-background">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border bg-card px-4 py-2.5">
        <div className="flex items-center gap-3">
          <button onClick={onClose} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors">
            <X className="h-4 w-4" />
          </button>
          <h2 className="text-sm font-medium text-foreground">Raumplanung</h2>
        </div>

        <div className="flex items-center gap-2">
          <button onClick={() => navigate(-1)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors">
            <ChevronLeft className="h-4 w-4" />
          </button>
          <span className="text-sm font-medium text-foreground min-w-[140px] text-center">
            KW {getISOWeek(weekDays[0])} · {MONTHS_DE[weekDays[0].getMonth()]} {weekDays[0].getFullYear()}
          </span>
          <button onClick={() => navigate(1)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors">
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>

        <div className="w-20" />
      </div>

      {/* Day tabs */}
      <div className="flex border-b border-border bg-card px-4">
        {weekDays.map((d, i) => {
          const today = isToday(d)
          const active = i === selectedDay
          const dayBookingCount = bookings.filter((b) => b.date === formatDateKey(d)).length
          return (
            <button
              key={i}
              onClick={() => setSelectedDay(i)}
              className={`relative flex items-center gap-2 px-4 py-2.5 text-xs transition-colors ${
                active
                  ? 'text-primary font-medium'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <span className={today ? 'flex h-5 w-5 items-center justify-center rounded-full bg-primary text-[10px] text-primary-foreground' : ''}>
                {d.getDate()}
              </span>
              <span>{DAYS_SHORT[i]}</span>
              {dayBookingCount > 0 && (
                <span className="flex h-4 min-w-4 items-center justify-center rounded-full bg-secondary text-[9px] font-medium text-muted-foreground px-1">
                  {dayBookingCount}
                </span>
              )}
              {active && <span className="absolute bottom-0 left-2 right-2 h-0.5 rounded-full bg-primary" />}
            </button>
          )
        })}
      </div>

      {/* Timeline grid */}
      <div className="flex-1 overflow-auto">
        <div className="min-w-[1100px]">
          {/* Hour headers */}
          <div className="flex sticky top-0 bg-card border-b border-border z-10">
            <div className="shrink-0 border-r border-border px-3 py-2" style={{ width: roomLabelWidth }}>
              <span className="text-[10px] uppercase tracking-wider text-muted-foreground">Raum</span>
            </div>
            {HOURS.map((hour) => (
              <div
                key={hour}
                className="shrink-0 border-r border-border-muted px-1 py-2 text-center"
                style={{ width: slotWidth }}
              >
                <span className="text-[10px] text-muted-foreground">{String(hour).padStart(2, '0')}:00</span>
              </div>
            ))}
          </div>

          {/* Room rows */}
          {ROOMS.map((room, roomIdx) => {
            const roomBookings = dayBookings.filter((b) => b.room === room.name)
            const color = ROOM_COLORS[roomIdx % ROOM_COLORS.length]

            return (
              <div key={room.id} className="flex border-b border-border-muted hover:bg-secondary/20 transition-colors">
                {/* Room label */}
                <div className="shrink-0 border-r border-border px-3 py-3 flex flex-col justify-center" style={{ width: roomLabelWidth }}>
                  <p className="text-xs font-medium text-foreground truncate">{room.name}</p>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                      <Users className="h-2.5 w-2.5" /> {room.capacity}
                    </span>
                    {room.tags.map((tag) => (
                      <span key={tag} className="rounded bg-secondary px-1.5 py-0.5 text-[9px] text-muted-foreground">
                        {tag}
                      </span>
                    ))}
                  </div>
                </div>

                {/* Time slots */}
                <div className="flex relative" style={{ height: 56 }}>
                  {HOURS.map((hour) => (
                    <div
                      key={hour}
                      className="shrink-0 border-r border-border-muted cursor-pointer hover:bg-primary/5 transition-colors"
                      style={{ width: slotWidth }}
                      onClick={() => handleSlotClick(room, hour)}
                    />
                  ))}

                  {/* Booking blocks */}
                  {roomBookings.map((booking) => {
                    const startFrac = timeToHourFraction(booking.startTime)
                    const endFrac = timeToHourFraction(booking.endTime)
                    const left = (startFrac - 8) * slotWidth
                    const width = (endFrac - startFrac) * slotWidth - 2

                    return (
                      <button
                        key={booking.id}
                        onClick={(e) => {
                          e.stopPropagation()
                          setSelectedBooking(booking)
                        }}
                        className="absolute top-1 bottom-1 rounded-md px-2 flex items-center gap-1.5 overflow-hidden transition-all hover:brightness-95 hover:shadow-sm"
                        style={{
                          left,
                          width: Math.max(width, 40),
                          backgroundColor: `${color}20`,
                          borderLeft: `3px solid ${color}`,
                        }}
                      >
                        <p className="text-[10px] font-medium truncate" style={{ color }}>
                          {booking.title}
                        </p>
                        {width > 120 && (
                          <span className="text-[9px] shrink-0" style={{ color, opacity: 0.7 }}>
                            {booking.startTime}–{booking.endTime}
                          </span>
                        )}
                      </button>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 border-t border-border bg-card px-4 py-2">
        <span className="text-[10px] text-muted-foreground">Klicke auf einen freien Slot um zu buchen</span>
        <span className="text-[10px] text-muted-foreground ml-auto">
          {dayBookings.length} Buchung{dayBookings.length !== 1 ? 'en' : ''} heute
        </span>
      </div>

      {/* Quick book dialog */}
      {quickBook && (
        <QuickBookDialog
          room={quickBook.room}
          date={quickBook.date}
          startHour={quickBook.hour}
          onClose={() => setQuickBook(null)}
          onBook={handleBook}
        />
      )}

      {/* Booking detail popover */}
      {selectedBooking && (
        <>
          <div className="fixed inset-0 z-50" onClick={() => setSelectedBooking(null)} />
          <div className="fixed z-50 top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-72 rounded-xl border border-border bg-card shadow-[var(--shadow-large)] overflow-hidden">
            <div className="p-4 space-y-2.5">
              <div className="flex items-start justify-between">
                <h4 className="text-sm font-medium text-foreground">{selectedBooking.title}</h4>
                <button onClick={() => setSelectedBooking(null)} className="rounded p-0.5 text-muted-foreground hover:bg-secondary">
                  <X className="h-3.5 w-3.5" />
                </button>
              </div>
              <div className="space-y-1.5 text-xs text-muted-foreground">
                <div className="flex items-center gap-2">
                  <Monitor className="h-3.5 w-3.5" />
                  <span>{selectedBooking.room}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Clock className="h-3.5 w-3.5" />
                  <span>{selectedBooking.startTime} – {selectedBooking.endTime}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Users className="h-3.5 w-3.5" />
                  <span>{selectedBooking.organizer} · {selectedBooking.participants} TN</span>
                </div>
              </div>
              <div className="flex gap-2 pt-1">
                <button
                  onClick={() => handleDeleteBooking(selectedBooking.id)}
                  className="flex-1 rounded-lg border border-red-200 px-3 py-1.5 text-xs text-red-600 hover:bg-red-50 transition-colors dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20"
                >
                  Stornieren
                </button>
                <button
                  onClick={() => setSelectedBooking(null)}
                  className="flex-1 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  Schliessen
                </button>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

// Helper
function getISOWeek(date: Date): number {
  const d = new Date(date)
  d.setHours(0, 0, 0, 0)
  d.setDate(d.getDate() + 3 - ((d.getDay() + 6) % 7))
  const week1 = new Date(d.getFullYear(), 0, 4)
  return 1 + Math.round(((d.getTime() - week1.getTime()) / 86400000 - 3 + ((week1.getDay() + 6) % 7)) / 7)
}
