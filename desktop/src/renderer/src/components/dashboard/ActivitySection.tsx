import { Clock } from 'lucide-react'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'

interface Activity {
  id: number
  user: string
  avatar: string
  action: string
  time: string
}

const recentActivity: Activity[] = [
  {
    id: 1,
    user: 'Michael Berg',
    avatar: 'MB',
    action: 'hat Projekt "Website Relaunch" aktualisiert',
    time: 'vor 5 Minuten',
  },
  {
    id: 2,
    user: 'Sarah Klein',
    avatar: 'SK',
    action: 'hat 3 neue Aufgaben erstellt',
    time: 'vor 1 Stunde',
  },
  {
    id: 3,
    user: 'Anna Müller',
    avatar: 'AM',
    action: 'hat Dokument "Q1 Budget.xlsx" hochgeladen',
    time: 'vor 2 Stunden',
  },
  {
    id: 4,
    user: 'Thomas Klein',
    avatar: 'TK',
    action: 'hat Aufgabe "Logo Redesign" abgeschlossen',
    time: 'vor 3 Stunden',
  },
]

export function ActivitySection() {
  return (
    <div className="rounded-lg border border-border bg-card p-6">
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">
          Letzte Aktivitäten
        </h2>
        <Clock className="h-5 w-5 text-muted-foreground" />
      </div>

      <div className="space-y-4">
        {recentActivity.map((activity) => (
          <div
            key={activity.id}
            className="flex items-start gap-3 border-b border-border pb-4 last:border-0 last:pb-0"
          >
            <Avatar className="h-10 w-10 shrink-0">
              <AvatarFallback className="bg-primary/10 text-sm text-primary">
                {activity.avatar}
              </AvatarFallback>
            </Avatar>
            <div className="min-w-0 flex-1">
              <p className="text-sm text-foreground">
                <span className="font-medium">{activity.user}</span>{' '}
                {activity.action}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {activity.time}
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
