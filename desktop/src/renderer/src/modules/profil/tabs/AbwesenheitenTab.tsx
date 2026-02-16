import { useState } from 'react'
import {
  Plus, Palmtree, Thermometer, Home, GraduationCap, HelpCircle,
  X, Calendar,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore, type AbsenceRequest } from '@/stores/timetracking'
import { toast } from 'sonner'

type FilterKey = 'all' | AbsenceRequest['type']

const ABSENCE_TYPES: { key: AbsenceRequest['type']; label: string; icon: typeof Palmtree; color: string }[] = [
  { key: 'vacation', label: 'Urlaub', icon: Palmtree, color: 'text-emerald-600 dark:text-emerald-400' },
  { key: 'sick', label: 'Krankheit', icon: Thermometer, color: 'text-red-600 dark:text-red-400' },
  { key: 'homeoffice', label: 'Homeoffice', icon: Home, color: 'text-blue-600 dark:text-blue-400' },
  { key: 'education', label: 'Weiterbildung', icon: GraduationCap, color: 'text-purple-600 dark:text-purple-400' },
  { key: 'other', label: 'Sonstiges', icon: HelpCircle, color: 'text-gray-600 dark:text-gray-400' },
]

const STATUS_BADGES: Record<AbsenceRequest['status'], { label: string; className: string }> = {
  pending: { label: 'Ausstehend', className: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400' },
  approved: { label: 'Genehmigt', className: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400' },
  rejected: { label: 'Abgelehnt', className: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400' },
}

// Mock vacation entitlement
const VACATION_TOTAL = 25
const VACATION_USED = 10

export default function AbwesenheitenTab() {
  const [filter, setFilter] = useState<FilterKey>('all')
  const [showDialog, setShowDialog] = useState(false)
  const [formType, setFormType] = useState<AbsenceRequest['type']>('vacation')
  const [formStart, setFormStart] = useState('')
  const [formEnd, setFormEnd] = useState('')
  const [formReason, setFormReason] = useState('')

  const absences = useTimeTrackingStore((s) => s.absences)
  const addAbsence = useTimeTrackingStore((s) => s.addAbsence)
  const deleteAbsence = useTimeTrackingStore((s) => s.deleteAbsence)

  const filtered = filter === 'all' ? absences : absences.filter((a) => a.type === filter)
  const sorted = [...filtered].sort((a, b) => b.createdAt.localeCompare(a.createdAt))

  // Stats
  const sickDays = absences.filter((a) => a.type === 'sick' && a.status === 'approved').reduce((s, a) => s + a.days, 0)
  const homeofficeDays = absences.filter((a) => a.type === 'homeoffice' && a.status === 'approved').reduce((s, a) => s + a.days, 0)
  const educationDays = absences.filter((a) => a.type === 'education' && a.status === 'approved').reduce((s, a) => s + a.days, 0)
  const vacationRemaining = VACATION_TOTAL - VACATION_USED
  const vacationPercent = Math.round((VACATION_USED / VACATION_TOTAL) * 100)

  const handleSubmit = () => {
    if (!formStart || !formEnd || !formReason.trim()) {
      toast.error('Bitte alle Felder ausfuellen')
      return
    }
    const start = new Date(formStart)
    const end = new Date(formEnd)
    if (end < start) {
      toast.error('Enddatum muss nach Startdatum liegen')
      return
    }
    const days = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)) + 1
    addAbsence({
      type: formType,
      startDate: formStart,
      endDate: formEnd,
      days,
      status: 'pending',
      reason: formReason,
    })
    setShowDialog(false)
    setFormStart('')
    setFormEnd('')
    setFormReason('')
    toast.success('Abwesenheitsantrag eingereicht')
  }

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr)
    return `${d.getDate().toString().padStart(2, '0')}.${(d.getMonth() + 1).toString().padStart(2, '0')}.${d.getFullYear()}`
  }

  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-bold text-foreground">Abwesenheiten</h2>
        <Button size="sm" onClick={() => setShowDialog(true)} className="gap-2">
          <Plus className="h-4 w-4" />
          Neue Anfrage
        </Button>
      </div>

      {/* Filter */}
      <div className="flex flex-wrap gap-2">
        <button
          onClick={() => setFilter('all')}
          className={cn(
            'px-3 py-1.5 rounded-full text-xs font-medium border transition-colors',
            filter === 'all'
              ? 'bg-primary text-primary-foreground border-primary'
              : 'bg-card text-muted-foreground border-border hover:border-primary/50',
          )}
        >
          Alle ({absences.length})
        </button>
        {ABSENCE_TYPES.map((type) => {
          const count = absences.filter((a) => a.type === type.key).length
          return (
            <button
              key={type.key}
              onClick={() => setFilter(type.key)}
              className={cn(
                'px-3 py-1.5 rounded-full text-xs font-medium border transition-colors',
                filter === type.key
                  ? 'bg-primary text-primary-foreground border-primary'
                  : 'bg-card text-muted-foreground border-border hover:border-primary/50',
              )}
            >
              {type.label} ({count})
            </button>
          )
        })}
      </div>

      {/* Requests List */}
      <div className="space-y-3">
        {sorted.length === 0 && (
          <div className="text-center py-12 text-muted-foreground">
            <Calendar className="h-10 w-10 mx-auto mb-3 opacity-50" />
            <p className="font-medium">Keine Abwesenheiten</p>
            <p className="text-sm">Erstelle eine neue Anfrage</p>
          </div>
        )}
        {sorted.map((absence) => {
          const typeConfig = ABSENCE_TYPES.find((t) => t.key === absence.type)!
          const Icon = typeConfig.icon
          const statusBadge = STATUS_BADGES[absence.status]
          return (
            <div
              key={absence.id}
              className="flex items-start gap-4 p-4 rounded-xl border border-border bg-card hover:border-primary/30 transition-colors"
            >
              <div className={cn('mt-0.5', typeConfig.color)}>
                <Icon className="h-5 w-5" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-medium text-sm text-foreground">{typeConfig.label}</span>
                  <span className="text-sm text-muted-foreground">
                    {formatDate(absence.startDate)} - {formatDate(absence.endDate)}
                  </span>
                  <span className="text-xs text-muted-foreground">({absence.days} {absence.days === 1 ? 'Tag' : 'Tage'})</span>
                </div>
                <p className="text-sm text-muted-foreground">{absence.reason}</p>
                {absence.comment && (
                  <p className="text-xs text-muted-foreground mt-1 italic">Kommentar: {absence.comment}</p>
                )}
              </div>
              <div className="flex items-center gap-2">
                <span className={cn('px-2.5 py-0.5 rounded-full text-xs font-medium', statusBadge.className)}>
                  {statusBadge.label}
                </span>
                {absence.status === 'pending' && (
                  <button
                    onClick={() => deleteAbsence(absence.id)}
                    className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {/* Vacation Balance */}
        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-3">Resturlaub</h3>
          <div className="flex items-end gap-2 mb-3">
            <span className="text-3xl font-bold text-primary">{vacationRemaining}</span>
            <span className="text-sm text-muted-foreground mb-1">/ {VACATION_TOTAL} Tage</span>
          </div>
          <div className="w-full h-3 rounded-full bg-secondary overflow-hidden">
            <div
              className="h-full rounded-full bg-primary transition-all"
              style={{ width: `${vacationPercent}%` }}
            />
          </div>
          <p className="text-xs text-muted-foreground mt-2">{VACATION_USED} Tage genommen</p>
        </div>

        {/* Yearly Overview */}
        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-3">Jahresübersicht 2026</h3>
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Krankheitstage</span>
              <span className="font-medium text-foreground">{sickDays} Tage</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Homeoffice</span>
              <span className="font-medium text-foreground">{homeofficeDays} Tage</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Weiterbildung</span>
              <span className="font-medium text-foreground">{educationDays} Tage</span>
            </div>
          </div>
        </div>
      </div>

      {/* New Request Dialog */}
      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Neue Abwesenheit beantragen</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Art der Abwesenheit</label>
              <Select value={formType} onValueChange={(v) => setFormType(v as AbsenceRequest['type'])}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ABSENCE_TYPES.map((t) => (
                    <SelectItem key={t.key} value={t.key}>{t.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">Von</label>
                <Input type="date" value={formStart} onChange={(e) => setFormStart(e.target.value)} />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">Bis</label>
                <Input type="date" value={formEnd} onChange={(e) => setFormEnd(e.target.value)} />
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">Grund</label>
              <textarea
                value={formReason}
                onChange={(e) => setFormReason(e.target.value)}
                rows={3}
                placeholder="Beschreibe den Grund für deine Abwesenheit..."
                className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDialog(false)}>Abbrechen</Button>
            <Button onClick={handleSubmit}>Antrag einreichen</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
