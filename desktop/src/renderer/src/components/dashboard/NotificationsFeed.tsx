import { useState } from 'react'
import {
  Mail,
  Calendar,
  MessageSquare,
  FolderKanban,
  CheckSquare,
  Clock,
  ChevronDown,
  Minimize2,
  Maximize2,
} from 'lucide-react'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { cn } from '@/lib/cn'

type TabId = 'mails' | 'calendar' | 'messages' | 'projects' | 'tasks'
type ViewState = 'minimized' | 'half' | 'full'

interface FeedNotification {
  id: string
  title: string
  description: string
  time: string
  avatar: string
}

const tabs = [
  { id: 'mails' as TabId, label: 'Mails', icon: Mail },
  { id: 'calendar' as TabId, label: 'Kalender', icon: Calendar },
  { id: 'messages' as TabId, label: 'Nachrichten', icon: MessageSquare },
  { id: 'projects' as TabId, label: 'Projekte', icon: FolderKanban },
  { id: 'tasks' as TabId, label: 'Aufgaben', icon: CheckSquare },
]

const feedData: Record<TabId, FeedNotification[]> = {
  mails: [
    { id: '1', title: 'Neue E-Mail von Achim Weber', description: 'Betreff: Website Relaunch - Deadline Update', time: 'Vor 10 Minuten', avatar: 'AW' },
    { id: '2', title: 'Neue E-Mail von Lisa Meier', description: 'Betreff: Budget-Freigabe Q2 2025', time: 'Vor 45 Minuten', avatar: 'LM' },
    { id: '3', title: 'Neue E-Mail von Thomas Klein', description: 'Betreff: Team Meeting nächste Woche', time: 'Vor 2 Stunden', avatar: 'TK' },
  ],
  calendar: [
    { id: '1', title: 'Neuer Termin eingetragen', description: 'Sprint Planning - Projekt "Mobile App" am 23.12. um 14:00', time: 'Vor 20 Minuten', avatar: 'SK' },
    { id: '2', title: 'Termin verschoben', description: 'Client Meeting wurde auf 27.12. um 10:00 verschoben', time: 'Vor 1 Stunde', avatar: 'MB' },
    { id: '3', title: 'Erinnerung', description: 'Review Meeting morgen um 9:00', time: 'Vor 3 Stunden', avatar: 'AM' },
  ],
  messages: [
    { id: '1', title: 'Michael Berg in #design', description: 'Die neuen Mockups sind fertig! @Anna kannst du mal schauen?', time: 'Vor 5 Minuten', avatar: 'MB' },
    { id: '2', title: 'Sarah Klein in #entwicklung', description: 'API Integration ist jetzt live!', time: 'Vor 30 Minuten', avatar: 'SK' },
    { id: '3', title: 'Anna Mueller in #allgemein', description: 'Reminder: Team Lunch morgen um 12:30', time: 'Vor 2 Stunden', avatar: 'AM' },
  ],
  projects: [
    { id: '1', title: 'Projekt "Website Relaunch" aktualisiert', description: 'Status geändert: In Progress → Review', time: 'Vor 15 Minuten', avatar: 'MB' },
    { id: '2', title: 'Neues Projekt erstellt', description: '"Mobile App Development" von Sarah Klein erstellt', time: 'Vor 1 Stunde', avatar: 'SK' },
    { id: '3', title: 'Projekt-Deadline Warnung', description: '"Brand Refresh" endet in 3 Tagen', time: 'Vor 4 Stunden', avatar: 'TK' },
  ],
  tasks: [
    { id: '1', title: 'Neue Aufgabe zugewiesen', description: 'Sarah hat dir "API Dokumentation schreiben" zugewiesen', time: 'Vor 8 Minuten', avatar: 'SK' },
    { id: '2', title: 'Aufgabe fällig bald', description: '"Design Review vorbereiten" fällig morgen', time: 'Vor 25 Minuten', avatar: 'MB' },
    { id: '3', title: 'Aufgabe abgeschlossen', description: 'Michael hat "Logo Redesign" als erledigt markiert', time: 'Vor 1 Stunde', avatar: 'MB' },
  ],
}

function ViewToggle({
  state,
  onChange,
}: {
  state: ViewState
  onChange: (s: ViewState) => void
}) {
  const items: { value: ViewState; icon: typeof Minimize2; title: string }[] = [
    { value: 'minimized', icon: Minimize2, title: 'Minimiert' },
    { value: 'half', icon: ChevronDown, title: 'Halb' },
    { value: 'full', icon: Maximize2, title: 'Voll' },
  ]

  return (
    <div className="flex items-center gap-1">
      {items.map((item) => {
        const Icon = item.icon
        return (
          <button
            key={item.value}
            onClick={() => onChange(item.value)}
            title={item.title}
            className={cn(
              'rounded p-1.5 transition-colors',
              state === item.value
                ? 'bg-primary/10 text-primary'
                : 'text-muted-foreground hover:bg-accent',
            )}
          >
            <Icon className="h-3.5 w-3.5" />
          </button>
        )
      })}
    </div>
  )
}

export function NotificationsFeed() {
  const [activeTab, setActiveTab] = useState<TabId>('mails')
  const [viewState, setViewState] = useState<ViewState>('half')

  const current = feedData[activeTab]
  const display =
    viewState === 'minimized'
      ? []
      : viewState === 'half'
        ? current.slice(0, 3)
        : current

  return (
    <div className="mb-8 overflow-hidden rounded-lg border border-border bg-card">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-6 py-4">
        <h2 className="text-lg font-semibold text-foreground">Benachrichtigungen</h2>
        <ViewToggle state={viewState} onChange={setViewState} />
      </div>

      {viewState !== 'minimized' && (
        <>
          {/* Tabs */}
          <div className="border-b border-border bg-accent/30">
            <div className="flex overflow-x-auto">
              {tabs.map((tab) => {
                const Icon = tab.icon
                const isActive = activeTab === tab.id
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={cn(
                      'flex items-center gap-2 whitespace-nowrap border-b-2 px-4 py-3 transition-colors',
                      isActive
                        ? 'border-primary bg-card text-primary'
                        : 'border-transparent text-muted-foreground hover:bg-accent hover:text-foreground',
                    )}
                  >
                    <Icon className="h-4 w-4" />
                    <span className="text-sm font-medium">{tab.label}</span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* List */}
          <div className="divide-y divide-border">
            {display.length === 0 ? (
              <div className="p-8 text-center">
                <p className="text-sm text-muted-foreground">
                  Keine Benachrichtigungen
                </p>
              </div>
            ) : (
              display.map((n) => (
                <div
                  key={n.id}
                  className="cursor-pointer p-4 transition-colors hover:bg-accent"
                >
                  <div className="flex items-start gap-3">
                    <Avatar className="h-10 w-10 shrink-0">
                      <AvatarFallback className="bg-primary/10 text-xs text-primary">
                        {n.avatar}
                      </AvatarFallback>
                    </Avatar>
                    <div className="min-w-0 flex-1">
                      <p className="mb-1 text-sm font-medium text-foreground">
                        {n.title}
                      </p>
                      <p className="mb-1 text-sm text-muted-foreground line-clamp-2">
                        {n.description}
                      </p>
                      <div className="flex items-center gap-1 text-xs text-muted-foreground">
                        <Clock className="h-3 w-3" />
                        <span>{n.time}</span>
                      </div>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </>
      )}
    </div>
  )
}
