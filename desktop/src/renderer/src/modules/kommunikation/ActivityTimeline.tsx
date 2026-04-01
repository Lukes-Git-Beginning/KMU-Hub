import {
  Mail,
  Phone,
  Video,
  FileText,
  MessageCircle,
  Calendar,
  History,
} from 'lucide-react'

// ---------------------------------------------------------------------------
// Mock activity data
// ---------------------------------------------------------------------------

interface Activity {
  id: string
  type: 'email' | 'call' | 'meeting' | 'note' | 'message' | 'document'
  description: string
  timestamp: string
  user: string
}

const activityIcons: Record<string, typeof Mail> = {
  email: Mail,
  call: Phone,
  meeting: Video,
  note: FileText,
  message: MessageCircle,
  document: Calendar,
}

const activityColors: Record<string, string> = {
  email: 'text-blue-500 bg-blue-50 dark:bg-blue-950/30',
  call: 'text-green-500 bg-green-50 dark:bg-green-950/30',
  meeting: 'text-violet-500 bg-violet-50 dark:bg-violet-950/30',
  note: 'text-orange-500 bg-orange-50 dark:bg-orange-950/30',
  message: 'text-teal-500 bg-teal-50 dark:bg-teal-950/30',
  document: 'text-muted-foreground bg-secondary',
}

function formatRelative(dateStr: string): string {
  try {
    const d = new Date(dateStr)
    const now = new Date()
    const diffH = Math.floor((now.getTime() - d.getTime()) / 3600000)
    if (diffH < 1) return 'Gerade eben'
    if (diffH < 24) return `Vor ${diffH}h`
    const diffD = Math.floor(diffH / 24)
    if (diffD < 7) return `Vor ${diffD}d`
    return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })
  } catch {
    return dateStr
  }
}

// ---------------------------------------------------------------------------
// Generate mock activities based on contact name
// ---------------------------------------------------------------------------

function getMockActivities(contactName: string): Activity[] {
  return [
    { id: 'act1', type: 'email', description: `E-Mail an ${contactName} gesendet`, timestamp: '2026-02-20T09:15:00', user: 'Anna Müller' },
    { id: 'act2', type: 'call', description: `Telefonat mit ${contactName}`, timestamp: '2026-02-19T14:30:00', user: 'Peter Schmidt' },
    { id: 'act3', type: 'note', description: 'Interne Notiz zum Angebot', timestamp: '2026-02-18T16:00:00', user: 'Anna Müller' },
    { id: 'act4', type: 'meeting', description: `Online-Meeting mit ${contactName}`, timestamp: '2026-02-15T10:00:00', user: 'Thomas Weber' },
    { id: 'act5', type: 'document', description: 'Angebot erstellt (PDF)', timestamp: '2026-02-14T11:30:00', user: 'Peter Schmidt' },
  ]
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ActivityTimelineProps {
  contactName: string
}

export function ActivityTimeline({ contactName }: ActivityTimelineProps) {
  const activities = getMockActivities(contactName)

  return (
    <div className="p-3 border-b border-border">
      <div className="flex items-center gap-1.5 mb-2">
        <History className="h-3 w-3 text-muted-foreground" />
        <h4 className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">
          Aktivitaeten
        </h4>
      </div>
      <div className="space-y-2">
        {activities.map((act, idx) => {
          const Icon = activityIcons[act.type] ?? Mail
          const colors = activityColors[act.type] ?? ''
          return (
            <div key={act.id} className="flex gap-2">
              {/* Icon + connecting line */}
              <div className="flex flex-col items-center">
                <div className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-full ${colors}`}>
                  <Icon className="h-2.5 w-2.5" />
                </div>
                {idx < activities.length - 1 && (
                  <div className="w-px flex-1 bg-border mt-0.5" />
                )}
              </div>

              {/* Content */}
              <div className="min-w-0 pb-2">
                <p className="text-[11px] text-foreground/90 leading-tight">{act.description}</p>
                <div className="flex items-center gap-1.5 mt-0.5 text-[10px] text-muted-foreground">
                  <span>{act.user}</span>
                  <span>·</span>
                  <span>{formatRelative(act.timestamp)}</span>
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
