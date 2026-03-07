import { useState, useRef, useEffect } from 'react';
import { Bell, Pin, Trash2, Mail, FileText, Users, Calendar, CheckCircle } from 'lucide-react';
import { Link } from 'react-router-dom';

interface Notification {
  id: string;
  type: 'mail' | 'task' | 'team' | 'calendar' | 'system';
  title: string;
  message: string;
  time: string;
  read: boolean;
  pinned: boolean;
  icon?: React.ReactNode;
}

const mockNotifications: Notification[] = [
  {
    id: '1',
    type: 'mail',
    title: 'Neue Nachricht von Achim',
    message: 'Bzgl. Website Relaunch Projekt - Deadline wurde verschoben',
    time: 'Vor 5 Minuten',
    read: false,
    pinned: false,
  },
  {
    id: '2',
    type: 'task',
    title: 'Aufgabe zugewiesen',
    message: 'Sarah hat dir "API Dokumentation" zugewiesen',
    time: 'Vor 15 Minuten',
    read: false,
    pinned: false,
  },
  {
    id: '3',
    type: 'team',
    title: 'Team-Update',
    message: 'Max Weber ist dem Projekt "Brand Refresh" beigetreten',
    time: 'Vor 1 Stunde',
    read: false,
    pinned: false,
  },
  {
    id: '4',
    type: 'calendar',
    title: 'Meeting Erinnerung',
    message: 'Sprint Planning in 30 Minuten',
    time: 'Vor 2 Stunden',
    read: true,
    pinned: true,
  },
  {
    id: '5',
    type: 'system',
    title: 'Projekt abgeschlossen',
    message: '"Mobile App Development" wurde erfolgreich abgeschlossen',
    time: 'Gestern',
    read: true,
    pinned: false,
  },
];

export function NotificationCenter() {
  const [isOpen, setIsOpen] = useState(false);
  const [notifications, setNotifications] = useState<Notification[]>(mockNotifications);
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; notificationId: string } | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const contextMenuRef = useRef<HTMLDivElement>(null);

  const unreadCount = notifications.filter(n => !n.read).length;

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
      if (contextMenuRef.current && !contextMenuRef.current.contains(event.target as Node)) {
        setContextMenu(null);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const getIcon = (type: string) => {
    switch (type) {
      case 'mail':
        return <Mail className="w-4 h-4" />;
      case 'task':
        return <FileText className="w-4 h-4" />;
      case 'team':
        return <Users className="w-4 h-4" />;
      case 'calendar':
        return <Calendar className="w-4 h-4" />;
      case 'system':
        return <CheckCircle className="w-4 h-4" />;
      default:
        return <Bell className="w-4 h-4" />;
    }
  };

  const getIconBgColor = (type: string) => {
    switch (type) {
      case 'mail':
        return 'bg-blue-100 dark:bg-blue-900 text-blue-600 dark:text-blue-400';
      case 'task':
        return 'bg-purple-100 dark:bg-purple-900 text-purple-600 dark:text-purple-400';
      case 'team':
        return 'bg-green-100 dark:bg-green-900 text-green-600 dark:text-green-400';
      case 'calendar':
        return 'bg-orange-100 dark:bg-orange-900 text-orange-600 dark:text-orange-400';
      case 'system':
        return 'bg-emerald-100 dark:bg-emerald-900 text-emerald-600 dark:text-emerald-400';
      default:
        return 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400';
    }
  };

  const handleNotificationClick = (id: string) => {
    setNotifications(notifications.map(n => 
      n.id === id ? { ...n, read: true } : n
    ));
  };

  const handleContextMenu = (e: React.MouseEvent, notificationId: string) => {
    e.preventDefault();
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      notificationId,
    });
  };

  const handlePin = (id: string) => {
    setNotifications(notifications.map(n => 
      n.id === id ? { ...n, pinned: !n.pinned } : n
    ));
    setContextMenu(null);
  };

  const handleDelete = (id: string) => {
    setNotifications(notifications.filter(n => n.id !== id));
    setContextMenu(null);
  };

  const handleMarkAllAsRead = () => {
    setNotifications(notifications.map(n => ({ ...n, read: true })));
  };

  // Sort: pinned first, then by read status
  const sortedNotifications = [...notifications].sort((a, b) => {
    if (a.pinned && !b.pinned) return -1;
    if (!a.pinned && b.pinned) return 1;
    if (!a.read && b.read) return -1;
    if (a.read && !b.read) return 1;
    return 0;
  });

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Bell Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="relative p-2 text-slate-600 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white hover:bg-slate-50 dark:hover:bg-slate-700 rounded-lg transition-colors"
      >
        <Bell className="w-5 h-5" />
        {unreadCount > 0 && (
          <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-emerald-600 dark:bg-emerald-500 rounded-full animate-pulse"></span>
        )}
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div className="absolute right-0 mt-2 w-96 bg-white dark:bg-slate-800 rounded-lg shadow-xl border border-slate-200 dark:border-slate-700 z-50 overflow-hidden">
          {/* Header */}
          <div className="px-4 py-3 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
            <div>
              <h3 className="font-semibold text-slate-900 dark:text-white">Benachrichtigungen</h3>
              {unreadCount > 0 && (
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  {unreadCount} ungelesen
                </p>
              )}
            </div>
            {unreadCount > 0 && (
              <button
                onClick={handleMarkAllAsRead}
                className="text-xs text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300 font-medium"
              >
                Alle gelesen
              </button>
            )}
          </div>

          {/* Notifications List */}
          <div className="max-h-96 overflow-y-auto">
            {sortedNotifications.length === 0 ? (
              <div className="p-8 text-center">
                <Bell className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
                <p className="text-sm text-slate-500 dark:text-slate-400">Keine Benachrichtigungen</p>
              </div>
            ) : (
              sortedNotifications.map((notification) => (
                <div
                  key={notification.id}
                  onClick={() => handleNotificationClick(notification.id)}
                  onContextMenu={(e) => handleContextMenu(e, notification.id)}
                  className={`px-4 py-3 border-b border-slate-100 dark:border-slate-700 cursor-pointer transition-colors relative ${
                    notification.read
                      ? 'bg-white dark:bg-slate-800 hover:bg-slate-50 dark:hover:bg-slate-750'
                      : 'bg-emerald-50/50 dark:bg-emerald-900/10 hover:bg-emerald-50 dark:hover:bg-emerald-900/20'
                  }`}
                >
                  <div className="flex gap-3">
                    {/* Icon */}
                    <div className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 ${getIconBgColor(notification.type)}`}>
                      {getIcon(notification.type)}
                    </div>

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-2 mb-1">
                        <p className={`text-sm font-medium ${
                          notification.read 
                            ? 'text-slate-700 dark:text-slate-300' 
                            : 'text-slate-900 dark:text-white'
                        }`}>
                          {notification.title}
                        </p>
                        {notification.pinned && (
                          <Pin className="w-3 h-3 text-emerald-600 dark:text-emerald-400 flex-shrink-0 fill-current" />
                        )}
                      </div>
                      <p className="text-xs text-slate-500 dark:text-slate-400 line-clamp-2 mb-1">
                        {notification.message}
                      </p>
                      <p className="text-xs text-slate-400 dark:text-slate-500">
                        {notification.time}
                      </p>
                    </div>

                    {/* Unread Indicator */}
                    {!notification.read && (
                      <div className="w-2 h-2 bg-emerald-600 dark:bg-emerald-500 rounded-full flex-shrink-0 mt-2"></div>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>

          {/* Footer */}
          {notifications.length > 0 && (
            <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900 border-t border-slate-200 dark:border-slate-700">
              <Link to="/notifications" className="w-full text-sm text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white transition-colors">
                Alle Benachrichtigungen anzeigen
              </Link>
            </div>
          )}
        </div>
      )}

      {/* Context Menu */}
      {contextMenu && (
        <div
          ref={contextMenuRef}
          className="fixed bg-white dark:bg-slate-800 rounded-lg shadow-xl border border-slate-200 dark:border-slate-700 py-1 z-[60] min-w-[160px]"
          style={{
            left: `${contextMenu.x}px`,
            top: `${contextMenu.y}px`,
          }}
        >
          <button
            onClick={() => handlePin(contextMenu.notificationId)}
            className="w-full px-4 py-2 text-left text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 flex items-center gap-2"
          >
            <Pin className="w-4 h-4" />
            {notifications.find(n => n.id === contextMenu.notificationId)?.pinned ? 'Lösen' : 'Anpinnen'}
          </button>
          <button
            onClick={() => handleDelete(contextMenu.notificationId)}
            className="w-full px-4 py-2 text-left text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 flex items-center gap-2"
          >
            <Trash2 className="w-4 h-4" />
            Löschen
          </button>
        </div>
      )}
    </div>
  );
}