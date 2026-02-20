/**
 * Left sidebar for the Unified Inbox.
 *
 * Three sections:
 * 1. Smart Views (Alle, Ungelesen, Markiert, Mir zugewiesen)
 * 2. Kanaele (E-Mail, Chat, Benachrichtigungen) with colored icons
 * 3. Team-Postfaecher (dynamic from API) with create/settings buttons
 */
import { useState } from 'react'
import {
  Inbox,
  Mail,
  MessageSquare,
  Bell,
  Star,
  UserCheck,
  ChevronLeft,
  ChevronRight,
  Plus,
  Settings,
  Users,
} from 'lucide-react'
import { useKommunikationStore } from '@/stores/kommunikation'
import { useUnreadCount, useTeamInboxes } from '@/api/hooks/useInbox'
import type { UnreadCountResponse, TeamInbox } from '@/api/inbox-types'

interface InboxSidebarProps {
  onCreateTeam: () => void
  onTeamSettings: (team: TeamInbox) => void
}

function getChannelCount(
  counts: UnreadCountResponse | undefined,
  channel: string,
): number {
  if (!counts) return 0
  const entry = counts.counts.find((c) => c.channel === channel)
  return entry?.count ?? 0
}

export function InboxSidebar({
  onCreateTeam,
  onTeamSettings,
}: InboxSidebarProps) {
  const { activeView, setActiveView, sidebarCollapsed, toggleSidebar } =
    useKommunikationStore()
  const { data: unreadData } = useUnreadCount()
  const { data: teams } = useTeamInboxes()

  const totalUnread = unreadData?.total ?? 0
  const emailUnread = getChannelCount(unreadData, 'email')
  const chatUnread = getChannelCount(unreadData, 'chat')
  const notifUnread = getChannelCount(unreadData, 'notification')

  const collapsed = sidebarCollapsed

  const viewItem = (
    id: string,
    icon: React.ReactNode,
    label: string,
    count?: number,
  ) => {
    const isActive = activeView === id
    return (
      <button
        key={id}
        onClick={() => setActiveView(id)}
        className={`flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors ${
          isActive
            ? 'bg-accent text-accent-foreground font-medium'
            : 'text-foreground hover:bg-secondary'
        }`}
        title={collapsed ? label : undefined}
      >
        {icon}
        {!collapsed && (
          <>
            <span className="flex-1 text-left truncate">{label}</span>
            {count !== undefined && count > 0 && (
              <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-primary text-[10px] text-primary-foreground px-1">
                {count}
              </span>
            )}
          </>
        )}
      </button>
    )
  }

  return (
    <aside
      className={`shrink-0 border-r border-border bg-card flex flex-col overflow-y-auto transition-all ${
        collapsed ? 'w-14' : 'w-64'
      }`}
    >
      {/* Collapse toggle */}
      <div className="flex items-center justify-end p-2">
        <button
          onClick={toggleSidebar}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title={collapsed ? 'Sidebar einblenden' : 'Sidebar ausblenden'}
        >
          {collapsed ? (
            <ChevronRight className="h-4 w-4" />
          ) : (
            <ChevronLeft className="h-4 w-4" />
          )}
        </button>
      </div>

      <div className="flex-1 px-2 pb-3 space-y-4">
        {/* Smart Views */}
        <div>
          {!collapsed && (
            <p className="px-3 pb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              Ansichten
            </p>
          )}
          <nav className="space-y-0.5">
            {viewItem(
              'all',
              <Inbox className="h-4 w-4 shrink-0" />,
              'Alle',
              totalUnread,
            )}
            {viewItem(
              'unread',
              <Mail className="h-4 w-4 shrink-0" />,
              'Ungelesen',
              totalUnread,
            )}
            {viewItem(
              'starred',
              <Star className="h-4 w-4 shrink-0" />,
              'Markiert',
            )}
            {viewItem(
              'assigned',
              <UserCheck className="h-4 w-4 shrink-0" />,
              'Mir zugewiesen',
            )}
          </nav>
        </div>

        {/* Channels */}
        <div>
          {!collapsed && (
            <p className="px-3 pb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              Kanaele
            </p>
          )}
          <nav className="space-y-0.5">
            {viewItem(
              'email',
              <Mail className="h-4 w-4 shrink-0 text-blue-500" />,
              'E-Mail',
              emailUnread,
            )}
            {viewItem(
              'chat',
              <MessageSquare className="h-4 w-4 shrink-0 text-green-500" />,
              'Chat',
              chatUnread,
            )}
            {viewItem(
              'notification',
              <Bell className="h-4 w-4 shrink-0 text-orange-500" />,
              'Benachrichtigungen',
              notifUnread,
            )}
          </nav>
        </div>

        {/* Team Inboxes */}
        <div>
          {!collapsed && (
            <div className="flex items-center justify-between px-3 pb-1">
              <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                Team-Postfaecher
              </p>
              <button
                onClick={onCreateTeam}
                className="rounded p-0.5 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
                title="Team-Postfach erstellen"
              >
                <Plus className="h-3.5 w-3.5" />
              </button>
            </div>
          )}
          <nav className="space-y-0.5">
            {teams?.map((team) => (
              <div key={team.id} className="group flex items-center">
                <button
                  onClick={() => setActiveView(team.id)}
                  className={`flex flex-1 items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors ${
                    activeView === team.id
                      ? 'bg-accent text-accent-foreground font-medium'
                      : 'text-foreground hover:bg-secondary'
                  }`}
                  title={collapsed ? team.name : undefined}
                >
                  <Users className="h-4 w-4 shrink-0 text-muted-foreground" />
                  {!collapsed && (
                    <span className="flex-1 text-left truncate">
                      {team.name}
                    </span>
                  )}
                </button>
                {!collapsed && (
                  <button
                    onClick={() => onTeamSettings(team)}
                    className="mr-1 rounded p-1 text-muted-foreground opacity-0 group-hover:opacity-100 hover:text-foreground hover:bg-secondary transition-all"
                    title="Einstellungen"
                  >
                    <Settings className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
            ))}
            {collapsed && (
              <button
                onClick={onCreateTeam}
                className="flex w-full items-center justify-center rounded-md p-2 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
                title="Team-Postfach erstellen"
              >
                <Plus className="h-4 w-4" />
              </button>
            )}
          </nav>
        </div>
      </div>
    </aside>
  )
}
