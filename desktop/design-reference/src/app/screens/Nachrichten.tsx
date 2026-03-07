import { useState } from 'react';
import { Hash, Send, Smile, Paperclip, MoreVertical, Users, Search, Plus, ChevronDown } from 'lucide-react';
import { Avatar, AvatarFallback } from '../components/ui/avatar';
import { Badge } from '../components/ui/badge';

const channels = [
  { id: 1, name: 'Allgemein', unread: 3, type: 'channel', icon: '#' },
  { id: 2, name: 'Entwicklung', unread: 0, type: 'channel', icon: '#' },
  { id: 3, name: 'Design', unread: 12, type: 'channel', icon: '#' },
  { id: 4, name: 'Marketing', unread: 0, type: 'channel', icon: '#' },
];

const directMessages = [
  { id: 5, name: 'Anna Müller', avatar: 'AM', unread: 2, status: 'online' },
  { id: 6, name: 'Michael Berg', avatar: 'MB', unread: 0, status: 'online' },
  { id: 7, name: 'Sarah Klein', avatar: 'SK', unread: 5, status: 'away' },
  { id: 8, name: 'Tom Weber', avatar: 'TW', unread: 0, status: 'offline' },
];

const messages = [
  {
    id: 1,
    user: 'Anna Müller',
    avatar: 'AM',
    time: '10:23',
    message: 'Hey Team! Wie läuft das neue Projekt?',
    isOwn: false,
  },
  {
    id: 2,
    user: 'Michael Berg',
    avatar: 'MB',
    time: '10:25',
    message: 'Sehr gut! Wir sind fast fertig mit dem Backend.',
    isOwn: false,
  },
  {
    id: 3,
    user: 'Sie',
    avatar: 'AM',
    time: '10:27',
    message: 'Die Designs sind auch bereit für das Review 🎨',
    isOwn: true,
  },
  {
    id: 4,
    user: 'Anna Müller',
    avatar: 'AM',
    time: '10:30',
    message: 'Perfekt! Können wir morgen ein Meeting machen?',
    isOwn: false,
  },
  {
    id: 5,
    user: 'Sie',
    avatar: 'AM',
    time: '10:32',
    message: 'Ja klar, 14 Uhr passt bei mir.',
    isOwn: true,
  },
];

export function Nachrichten() {
  const [activeChat, setActiveChat] = useState<any>(channels[0]);
  const [messageText, setMessageText] = useState('');

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online':
        return 'bg-green-500';
      case 'away':
        return 'bg-yellow-500';
      default:
        return 'bg-slate-300 dark:bg-slate-600';
    }
  };

  return (
    <div className="flex h-full bg-white dark:bg-slate-900">
      {/* Sidebar - Conversations */}
      <div className="w-80 border-r border-slate-200 dark:border-slate-700 flex flex-col bg-slate-50 dark:bg-[#1e293b]">
        {/* Sidebar Header */}
        <div className="p-4 border-b border-slate-200 dark:border-slate-700">
          <h2 className="text-slate-900 dark:text-white mb-3">Nachrichten</h2>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Konversationen suchen..."
              className="w-full pl-10 pr-4 py-2 bg-white dark:bg-slate-700 border border-slate-300 dark:border-slate-600 rounded-lg text-sm text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-emerald-500"
            />
          </div>
        </div>

        {/* Channels & Direct Messages Section */}
        <div className="flex-1 overflow-y-auto">
          <div className="p-2">
            {/* Channels */}
            <div className="mb-4">
              <button className="w-full flex items-center justify-between px-3 py-2 text-xs text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg">
                <div className="flex items-center gap-2">
                  <ChevronDown className="w-3 h-3" />
                  <span className="font-medium">Channels</span>
                </div>
                <Plus className="w-3 h-3" />
              </button>
              <div className="space-y-1 mt-1">
                {channels.map((channel) => (
                  <button
                    key={channel.id}
                    onClick={() => setActiveChat(channel)}
                    className={`w-full flex items-center justify-between px-3 py-2 rounded-lg transition-colors ${
                      activeChat.id === channel.id && activeChat.type === 'channel'
                        ? 'bg-emerald-100 dark:bg-[#4ECCA3]/15 text-emerald-700 dark:text-emerald-300'
                        : 'text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700'
                    }`}
                  >
                    <div className="flex items-center gap-3">
                      <span className="text-slate-500 dark:text-slate-400">#</span>
                      <span className="text-sm">{channel.name}</span>
                    </div>
                    {channel.unread > 0 && (
                      <Badge className="bg-emerald-600 text-white text-xs px-2">
                        {channel.unread}
                      </Badge>
                    )}
                  </button>
                ))}
              </div>
            </div>

            {/* Direct Messages */}
            <div>
              <button className="w-full flex items-center justify-between px-3 py-2 text-xs text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg">
                <div className="flex items-center gap-2">
                  <ChevronDown className="w-3 h-3" />
                  <span className="font-medium">Direktnachrichten</span>
                </div>
                <Plus className="w-3 h-3" />
              </button>
              <div className="space-y-1 mt-1">
                {directMessages.map((dm) => (
                  <button
                    key={dm.id}
                    onClick={() => setActiveChat(dm)}
                    className={`w-full flex items-center justify-between px-3 py-2 rounded-lg transition-colors ${
                      activeChat.id === dm.id && !activeChat.type
                        ? 'bg-emerald-100 dark:bg-[#4ECCA3]/15 text-emerald-700 dark:text-emerald-300'
                        : 'text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700'
                    }`}
                  >
                    <div className="flex items-center gap-3">
                      <div className="relative">
                        <Avatar className="w-6 h-6">
                          <AvatarFallback className="bg-slate-200 dark:bg-slate-600 text-slate-700 dark:text-slate-200 text-xs">
                            {dm.avatar}
                          </AvatarFallback>
                        </Avatar>
                        <div
                          className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-slate-50 dark:border-[#1e293b] ${getStatusColor(dm.status)}`}
                        />
                      </div>
                      <span className="text-sm">{dm.name}</span>
                    </div>
                    {dm.unread > 0 && (
                      <Badge className="bg-emerald-600 text-white text-xs px-2">
                        {dm.unread}
                      </Badge>
                    )}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* User Profile at Bottom */}
        <div className="p-4 border-t border-slate-200 dark:border-slate-700">
          <div className="flex items-center gap-3">
            <div className="relative">
              <Avatar className="w-10 h-10">
                <AvatarFallback className="bg-emerald-600 text-white">
                  AM
                </AvatarFallback>
              </Avatar>
              <div className="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-green-500 rounded-full border-2 border-slate-50 dark:border-[#1e293b]" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm text-slate-900 dark:text-white truncate">Anna Müller</p>
              <p className="text-xs text-slate-500 dark:text-slate-400">Online</p>
            </div>
            <button className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg">
              <MoreVertical className="w-4 h-4 text-slate-500 dark:text-slate-400" />
            </button>
          </div>
        </div>
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col bg-slate-100 dark:bg-slate-800">
        {/* Chat Header */}
        <div className="h-16 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between px-6 bg-white dark:bg-slate-800">
          <div className="flex items-center gap-3">
            {activeChat.type === 'channel' ? (
              <div className="w-10 h-10 bg-emerald-100 dark:bg-emerald-600/20 rounded-lg flex items-center justify-center">
                <Hash className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
              </div>
            ) : (
              <Avatar className="w-10 h-10">
                <AvatarFallback className="bg-slate-200 dark:bg-slate-600 text-slate-700 dark:text-slate-200">
                  {activeChat.avatar}
                </AvatarFallback>
              </Avatar>
            )}
            <div>
              <h3 className="text-slate-900 dark:text-white">{activeChat.name}</h3>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {activeChat.type === 'channel' ? `${channels.length} Mitglieder` : activeChat.status}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg">
              <Users className="w-5 h-5 text-slate-500 dark:text-slate-400" />
            </button>
            <button className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg">
              <MoreVertical className="w-5 h-5 text-slate-500 dark:text-slate-400" />
            </button>
          </div>
        </div>

        {/* Messages Area */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6 bg-white dark:bg-slate-800">
          {/* Date Divider */}
          <div className="flex items-center gap-4">
            <div className="flex-1 h-px bg-slate-200 dark:bg-slate-700" />
            <span className="text-xs text-slate-500 dark:text-slate-400">Heute</span>
            <div className="flex-1 h-px bg-slate-200 dark:bg-slate-700" />
          </div>

          {/* Messages */}
          {messages.map((message) =>
            message.isOwn ? (
              <div key={message.id} className="flex justify-end">
                <div className="max-w-lg">
                  <div className="bg-emerald-600 text-white rounded-2xl rounded-tr-sm px-4 py-3">
                    <p className="text-sm">{message.message}</p>
                  </div>
                  <p className="text-xs text-slate-400 dark:text-slate-500 mt-1 text-right">{message.time}</p>
                </div>
              </div>
            ) : (
              <div key={message.id} className="flex gap-3">
                <Avatar className="w-10 h-10 flex-shrink-0">
                  <AvatarFallback className="bg-slate-200 dark:bg-slate-600 text-slate-700 dark:text-slate-200">
                    {message.avatar}
                  </AvatarFallback>
                </Avatar>
                <div className="flex-1 max-w-lg">
                  <div className="flex items-baseline gap-2 mb-1">
                    <span className="text-sm text-slate-900 dark:text-white">{message.user}</span>
                    <span className="text-xs text-slate-400 dark:text-slate-500">{message.time}</span>
                  </div>
                  <div className="bg-slate-100 dark:bg-slate-700 rounded-2xl rounded-tl-sm px-4 py-3">
                    <p className="text-sm text-slate-900 dark:text-slate-200">{message.message}</p>
                  </div>
                </div>
              </div>
            )
          )}
        </div>

        {/* Message Input */}
        <div className="p-4 border-t border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
          <div className="flex items-end gap-3">
            <div className="flex-1 bg-slate-100 dark:bg-slate-700 rounded-lg px-4 py-3">
              <textarea
                value={messageText}
                onChange={(e) => setMessageText(e.target.value)}
                placeholder="Nachricht schreiben..."
                className="w-full bg-transparent text-slate-900 dark:text-slate-200 placeholder:text-slate-500 resize-none focus:outline-none"
                rows={1}
              />
              <div className="flex items-center gap-2 mt-2">
                <button className="p-1.5 hover:bg-slate-200 dark:hover:bg-slate-600 rounded">
                  <Paperclip className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                </button>
                <button className="p-1.5 hover:bg-slate-200 dark:hover:bg-slate-600 rounded">
                  <Smile className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                </button>
              </div>
            </div>
            <button className="px-4 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition-colors">
              <Send className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}