import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Eye, EyeOff, Globe, Users, User } from 'lucide-react'

interface CalendarSource {
  id: string
  name: string
  group: 'mine' | 'shared' | 'other'
  color: string
  visible: boolean
}

interface CalendarBrowseDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  calendars: CalendarSource[]
  onToggleCalendar: (id: string) => void
}

const GROUP_META: Record<string, { label: string; icon: typeof User; description: string }> = {
  mine: { label: 'Meine Kalender', icon: User, description: 'Persönliche und Arbeitskalender' },
  shared: { label: 'Geteilte Kalender', icon: Users, description: 'Team- und Projektkalender' },
  other: { label: 'Andere', icon: Globe, description: 'Feiertage, Deadlines und externe Kalender' },
}

export function CalendarBrowseDialog({
  open,
  onOpenChange,
  calendars,
  onToggleCalendar,
}: CalendarBrowseDialogProps) {
  const groups = [
    { key: 'mine', items: calendars.filter((c) => c.group === 'mine') },
    { key: 'shared', items: calendars.filter((c) => c.group === 'shared') },
    { key: 'other', items: calendars.filter((c) => c.group === 'other') },
  ]

  const visibleCount = calendars.filter((c) => c.visible).length

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Kalender verwalten</DialogTitle>
        </DialogHeader>

        <p className="text-xs text-muted-foreground">
          {visibleCount} von {calendars.length} Kalendern sichtbar
        </p>

        <div className="space-y-4 py-2">
          {groups.map(({ key, items }) => {
            const meta = GROUP_META[key]
            const Icon = meta.icon
            return (
              <div key={key}>
                <div className="flex items-center gap-2 mb-2">
                  <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                  <div>
                    <h4 className="text-xs font-medium text-foreground">{meta.label}</h4>
                    <p className="text-[10px] text-muted-foreground">{meta.description}</p>
                  </div>
                </div>
                <div className="space-y-0.5">
                  {items.map((cal) => (
                    <button
                      key={cal.id}
                      onClick={() => onToggleCalendar(cal.id)}
                      className="flex w-full items-center gap-3 rounded-lg px-3 py-2 hover:bg-secondary/50 transition-colors"
                    >
                      {/* Color dot / checkbox */}
                      <span
                        className="flex h-5 w-5 shrink-0 items-center justify-center rounded border-2 transition-colors"
                        style={{
                          borderColor: cal.color,
                          backgroundColor: cal.visible ? cal.color : 'transparent',
                        }}
                      >
                        {cal.visible && (
                          <svg className="h-3 w-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                          </svg>
                        )}
                      </span>

                      <span className={`text-sm flex-1 text-left ${cal.visible ? 'text-foreground' : 'text-muted-foreground'}`}>
                        {cal.name}
                      </span>

                      {cal.visible ? (
                        <Eye className="h-3.5 w-3.5 text-muted-foreground" />
                      ) : (
                        <EyeOff className="h-3.5 w-3.5 text-text-disabled" />
                      )}
                    </button>
                  ))}
                </div>
              </div>
            )
          })}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Schließen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
