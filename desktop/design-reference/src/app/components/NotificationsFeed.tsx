import { useState } from 'react';
import { Mail, Calendar, MessageSquare, FolderKanban, CheckSquare, Clock, ChevronDown, Minimize2, Maximize2 } from 'lucide-react';
import { Avatar, AvatarFallback } from './ui/avatar';

type NotificationType = 'mails' | 'calendar' | 'messages' | 'projects' | 'tasks';

interface Notification {
  id: string;
  title: string;
  description: string;
  time: string;
  avatar?: string;
  type: string;
}

const tabs = [
  { id: 'mails' as NotificationType, label: 'Mails', icon: Mail },
  { id: 'calendar' as NotificationType, label: 'Kalender', icon: Calendar },
  { id: 'messages' as NotificationType, label: 'Nachrichten', icon: MessageSquare },
  { id: 'projects' as NotificationType, label: 'Projekte', icon: FolderKanban },
  { id: 'tasks' as NotificationType, label: 'Aufgaben', icon: CheckSquare },
];

const notificationData: Record<NotificationType, Notification[]> = {
  mails: [
    {
      id: '1',
      title: 'Neue E-Mail von Achim Weber',
      description: 'Betreff: Website Relaunch - Deadline Update',
      time: 'Vor 10 Minuten',
      avatar: 'AW',
      type: 'mail',
    },
    {
      id: '2',
      title: 'Neue E-Mail von Lisa Meier',
      description: 'Betreff: Budget-Freigabe Q2 2025',
      time: 'Vor 45 Minuten',
      avatar: 'LM',
      type: 'mail',
    },
    {
      id: '3',
      title: 'Neue E-Mail von Thomas Klein',
      description: 'Betreff: Team Meeting nächste Woche',
      time: 'Vor 2 Stunden',
      avatar: 'TK',
      type: 'mail',
    },
  ],
  calendar: [
    {
      id: '1',
      title: 'Neuer Termin eingetragen',
      description: 'Sprint Planning - Projekt "Mobile App" am 23.12. um 14:00',
      time: 'Vor 20 Minuten',
      avatar: 'SK',
      type: 'calendar',
    },
    {
      id: '2',
      title: 'Termin verschoben',
      description: 'Client Meeting wurde auf 27.12. um 10:00 verschoben',
      time: 'Vor 1 Stunde',
      avatar: 'MB',
      type: 'calendar',
    },
    {
      id: '3',
      title: 'Erinnerung',
      description: 'Review Meeting morgen um 9:00',
      time: 'Vor 3 Stunden',
      avatar: 'AM',
      type: 'calendar',
    },
  ],
  messages: [
    {
      id: '1',
      title: 'Michael Berg in #design',
      description: 'Die neuen Mockups sind fertig! @Anna kannst du mal schauen?',
      time: 'Vor 5 Minuten',
      avatar: 'MB',
      type: 'message',
    },
    {
      id: '2',
      title: 'Sarah Klein in #entwicklung',
      description: 'API Integration ist jetzt live! 🎉',
      time: 'Vor 30 Minuten',
      avatar: 'SK',
      type: 'message',
    },
    {
      id: '3',
      title: 'Anna Müller in #allgemein',
      description: 'Reminder: Team Lunch morgen um 12:30',
      time: 'Vor 2 Stunden',
      avatar: 'AM',
      type: 'message',
    },
  ],
  projects: [
    {
      id: '1',
      title: 'Projekt "Website Relaunch" aktualisiert',
      description: 'Status geändert: In Progress → Review',
      time: 'Vor 15 Minuten',
      avatar: 'MB',
      type: 'project',
    },
    {
      id: '2',
      title: 'Neues Projekt erstellt',
      description: '"Mobile App Development" von Sarah Klein erstellt',
      time: 'Vor 1 Stunde',
      avatar: 'SK',
      type: 'project',
    },
    {
      id: '3',
      title: 'Projekt-Deadline Warnung',
      description: '"Brand Refresh" endet in 3 Tagen',
      time: 'Vor 4 Stunden',
      avatar: 'TK',
      type: 'project',
    },
  ],
  tasks: [
    {
      id: '1',
      title: 'Neue Aufgabe zugewiesen',
      description: 'Sarah hat dir "API Dokumentation schreiben" zugewiesen',
      time: 'Vor 8 Minuten',
      avatar: 'SK',
      type: 'task',
    },
    {
      id: '2',
      title: 'Aufgabe fällig bald',
      description: '"Design Review vorbereiten" fällig morgen',
      time: 'Vor 25 Minuten',
      avatar: 'MB',
      type: 'task',
    },
    {
      id: '3',
      title: 'Aufgabe abgeschlossen',
      description: 'Michael hat "Logo Redesign" als erledigt markiert',
      time: 'Vor 1 Stunde',
      avatar: 'MB',
      type: 'task',
    },
  ],
};

export function NotificationsFeed() {
  const [activeTab, setActiveTab] = useState<NotificationType>('mails');
  const [viewState, setViewState] = useState<'minimized' | 'half' | 'full'>('half');

  const currentNotifications = notificationData[activeTab];

  // Determine which notifications to show based on view state
  const displayNotifications = 
    viewState === 'minimized' 
      ? [] 
      : viewState === 'half' 
      ? currentNotifications.slice(0, 3)
      : currentNotifications;

  return (
    <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">
      {/* Header */}
      <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
        <h2 className="text-lg text-slate-900 dark:text-white">Benachrichtigungen</h2>
        
        {/* View State Toggle Buttons */}
        <div className="flex items-center gap-1">
          <button
            onClick={() => setViewState('minimized')}
            className={`p-1.5 rounded transition-colors ${
              viewState === 'minimized'
                ? 'bg-emerald-100 dark:bg-emerald-900 text-emerald-600 dark:text-emerald-400'
                : 'text-slate-400 dark:text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700'
            }`}
            title="Minimiert"
          >
            <Minimize2 className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={() => setViewState('half')}
            className={`p-1.5 rounded transition-colors ${
              viewState === 'half'
                ? 'bg-emerald-100 dark:bg-emerald-900 text-emerald-600 dark:text-emerald-400'
                : 'text-slate-400 dark:text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700'
            }`}
            title="Halb"
          >
            <ChevronDown className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={() => setViewState('full')}
            className={`p-1.5 rounded transition-colors ${
              viewState === 'full'
                ? 'bg-emerald-100 dark:bg-emerald-900 text-emerald-600 dark:text-emerald-400'
                : 'text-slate-400 dark:text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700'
            }`}
            title="Voll"
          >
            <Maximize2 className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Content - Only show if not minimized */}
      {viewState !== 'minimized' && (
        <>
          {/* Tabs - Horizontal Scrollable */}
          <div className="border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/50">
            <div className="flex overflow-x-auto scrollbar-hide">
              {tabs.map((tab) => {
                const Icon = tab.icon;
                const isActive = activeTab === tab.id;
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`flex items-center gap-2 px-4 py-3 border-b-2 transition-colors whitespace-nowrap ${
                      isActive
                        ? 'border-emerald-600 text-emerald-600 dark:text-emerald-400 bg-white dark:bg-slate-800'
                        : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800'
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                    <span className="text-sm font-medium">{tab.label}</span>
                  </button>
                );
              })}
            </div>
          </div>

          {/* Notifications List */}
          <div className="divide-y divide-slate-100 dark:divide-slate-700">
            {displayNotifications.length === 0 ? (
              <div className="p-8 text-center">
                <div className="w-12 h-12 bg-slate-100 dark:bg-slate-700 rounded-full flex items-center justify-center mx-auto mb-3">
                  {tabs.find((t) => t.id === activeTab) && (
                    (() => {
                      const Icon = tabs.find((t) => t.id === activeTab)!.icon;
                      return <Icon className="w-6 h-6 text-slate-400 dark:text-slate-500" />;
                    })()
                  )}
                </div>
                <p className="text-sm text-slate-500 dark:text-slate-400">Keine Benachrichtigungen</p>
              </div>
            ) : (
              displayNotifications.map((notification) => (
                <div
                  key={notification.id}
                  className="p-4 hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors cursor-pointer"
                >
                  <div className="flex items-start gap-3">
                    {/* Avatar */}
                    <Avatar className="w-10 h-10 flex-shrink-0">
                      <AvatarFallback className="bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300 text-xs">
                        {notification.avatar}
                      </AvatarFallback>
                    </Avatar>

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-slate-900 dark:text-white mb-1">
                        {notification.title}
                      </p>
                      <p className="text-sm text-slate-600 dark:text-slate-400 line-clamp-2 mb-1">
                        {notification.description}
                      </p>
                      <div className="flex items-center gap-1 text-xs text-slate-500 dark:text-slate-500">
                        <Clock className="w-3 h-3" />
                        <span>{notification.time}</span>
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
  );
}